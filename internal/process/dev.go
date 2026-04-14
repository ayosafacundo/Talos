package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"Talos/internal/packages"
)

// DevStartOptions configures optional dev-server URL discovery (log parse + HTTP probe).
type DevStartOptions struct {
	ManifestDevURL string
	OnResolvedURL  func(baseURL string)
}

// ringBuffer keeps recent dev-server output for URL parsing.
type ringBuffer struct {
	mu  sync.Mutex
	buf []byte
	max int
}

func newRingBuffer(max int) *ringBuffer {
	return &ringBuffer{max: max}
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf = append(r.buf, p...)
	if len(r.buf) > r.max {
		r.buf = r.buf[len(r.buf)-r.max:]
	}
	return len(p), nil
}

func (r *ringBuffer) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return string(r.buf)
}

// StartDev spawns development.command for a package (e.g. npm run dev). Logs to packageLogDir/<id>-dev.log.
// If opts is non-nil and OnResolvedURL is set, stdout/stderr are scanned for a loopback URL and a nearby-port HTTP probe runs as fallback.
func (m *Manager) StartDev(ctx context.Context, pkg *packages.PackageInfo, argv []string, opts *DevStartOptions) error {
	if pkg == nil || pkg.Manifest == nil || len(argv) == 0 {
		return errors.New("process: dev start requires package and argv")
	}
	appID := pkg.Manifest.ID
	m.devMu.Lock()
	if _, ok := m.devRunning[appID]; ok {
		m.devMu.Unlock()
		return nil
	}
	m.devMu.Unlock()

	manifestDevURL := ""
	if pkg.Manifest.Development != nil {
		manifestDevURL = strings.TrimSpace(pkg.Manifest.Development.URL)
	}
	if opts != nil && strings.TrimSpace(opts.ManifestDevURL) != "" {
		manifestDevURL = strings.TrimSpace(opts.ManifestDevURL)
	}
	prog := argv[0]
	args := argv[1:]
	execPath := prog
	if strings.ContainsAny(prog, `/\`) {
		execPath = filepath.Join(pkg.DirPath, filepath.Clean(prog))
	}
	pathResolved, err := exec.LookPath(execPath)
	if err != nil {
		pathResolved, err = exec.LookPath(prog)
		if err != nil {
			return fmt.Errorf("process: dev executable %q: %w", prog, err)
		}
	}

	cmd := exec.CommandContext(ctx, pathResolved, args...)
	cmd.Dir = pkg.DirPath
	m.mu.RLock()
	logDir := m.logDir
	m.mu.RUnlock()
	stdoutWriter := io.Writer(os.Stdout)
	stderrWriter := io.Writer(os.Stderr)
	var logFile *os.File
	if strings.TrimSpace(logDir) != "" {
		if err := os.MkdirAll(logDir, 0o755); err == nil {
			lp := filepath.Join(logDir, appID+"-dev.log")
			f, openErr := os.OpenFile(lp, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if openErr == nil {
				logFile = f
				stdoutWriter = io.MultiWriter(os.Stdout, f)
				stderrWriter = io.MultiWriter(os.Stderr, f)
			}
		}
	}

	var sniff *ringBuffer
	if opts != nil && opts.OnResolvedURL != nil {
		sniff = newRingBuffer(64 * 1024)
		stdoutWriter = io.MultiWriter(stdoutWriter, sniff)
		stderrWriter = io.MultiWriter(stderrWriter, sniff)
	}

	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, "NODE_ENV=development")
	if p := devServerPortFromURL(manifestDevURL); p != "" {
		cmd.Env = append(cmd.Env, "TALOS_DEV_SERVER_PORT="+p)
	}
	configureCmd(cmd)

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return fmt.Errorf("process: dev start %q: %w", appID, err)
	}
	m.devMu.Lock()
	m.devRunning[appID] = cmd
	m.devMu.Unlock()

	if sniff != nil && opts != nil && opts.OnResolvedURL != nil {
		go m.watchResolvedDevURL(ctx, manifestDevURL, sniff, opts.OnResolvedURL)
	}

	go func() {
		_ = cmd.Wait()
		if logFile != nil {
			_ = logFile.Close()
		}
		m.devMu.Lock()
		delete(m.devRunning, appID)
		m.devMu.Unlock()
	}()
	return nil
}

func devServerPortFromURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return u.Port()
}

func (m *Manager) watchResolvedDevURL(ctx context.Context, manifestDevURL string, sniff *ringBuffer, onResolved func(string)) {
	if onResolved == nil {
		return
	}
	var once sync.Once
	fire := func(s string) {
		once.Do(func() { onResolved(s) })
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(45 * time.Second)

	// Immediate attempt before first tick
	if s := ParseDevServerBaseURL(sniff.String()); s != "" {
		fire(s)
		return
	}
	if s := ProbeHTTPAroundManifest(ctx, manifestDevURL, 8); s != "" {
		fire(s)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline:
			return
		case <-ticker.C:
			if s := ParseDevServerBaseURL(sniff.String()); s != "" {
				fire(s)
				return
			}
			if s := ProbeHTTPAroundManifest(ctx, manifestDevURL, 8); s != "" {
				fire(s)
				return
			}
		}
	}
}

// StopDev terminates a dev server process for appID, if running.
func (m *Manager) StopDev(appID string) error {
	m.devMu.Lock()
	cmd := m.devRunning[appID]
	if cmd == nil {
		m.devMu.Unlock()
		return nil
	}
	delete(m.devRunning, appID)
	m.devMu.Unlock()
	if cmd.Process != nil {
		killCmdTree(cmd)
	}
	return nil
}

// StopAllDev kills all dev server processes.
func (m *Manager) StopAllDev() {
	m.devMu.Lock()
	ids := make([]string, 0, len(m.devRunning))
	for id := range m.devRunning {
		ids = append(ids, id)
	}
	m.devMu.Unlock()
	for _, id := range ids {
		_ = m.StopDev(id)
	}
}
