package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"Talos/internal/packages"
)

// Manager manages tiny app binary lifecycles.
type Manager struct {
	mu         sync.RWMutex
	running    map[string]*exec.Cmd
	logDir     string
	devMu      sync.Mutex
	devRunning map[string]*exec.Cmd
}

func NewManager() *Manager {
	return &Manager{
		running:    make(map[string]*exec.Cmd),
		devRunning: make(map[string]*exec.Cmd),
	}
}

func (m *Manager) SetLogDir(logDir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logDir = logDir
}

// Start launches a package binary if not already running.
func (m *Manager) Start(ctx context.Context, pkg *packages.PackageInfo, extraEnv map[string]string) error {
	if pkg == nil || pkg.Manifest == nil {
		return errors.New("process: package manifest is required")
	}
	if pkg.Manifest.Binary == "" {
		// Web-only package: no process lifecycle needed.
		return nil
	}

	appID := pkg.Manifest.ID
	m.mu.RLock()
	logDir := m.logDir
	m.mu.RUnlock()
	m.mu.RLock()
	if _, ok := m.running[appID]; ok {
		m.mu.RUnlock()
		return fmt.Errorf("process: app %q already running", appID)
	}
	m.mu.RUnlock()

	binPath := filepath.Join(pkg.DirPath, filepath.Clean(pkg.Manifest.Binary))
	absBinPath, err := filepath.Abs(binPath)
	if err != nil {
		return fmt.Errorf("process: resolve binary path %q failed: %w", binPath, err)
	}
	st, err := os.Stat(absBinPath)
	if err != nil {
		return fmt.Errorf("process: stat %q failed: %w", absBinPath, err)
	}
	if st.IsDir() {
		return fmt.Errorf("process: binary path %q is a directory", binPath)
	}

	cmd := exec.CommandContext(ctx, absBinPath)
	cmd.Dir = pkg.DirPath
	stdoutWriter := io.Writer(os.Stdout)
	stderrWriter := io.Writer(os.Stderr)
	var logFile *os.File
	if strings.TrimSpace(logDir) != "" {
		if err := os.MkdirAll(logDir, 0o755); err == nil {
			path := filepath.Join(logDir, appID+".log")
			f, openErr := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if openErr == nil {
				logFile = f
				stdoutWriter = io.MultiWriter(os.Stdout, f)
				stderrWriter = io.MultiWriter(os.Stderr, f)
			}
		}
	}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter
	if len(extraEnv) > 0 {
		env := os.Environ()
		for key, value := range extraEnv {
			env = append(env, key+"="+value)
		}
		cmd.Env = env
	}
	configureCmd(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("process: start %q failed: %w", appID, err)
	}

	m.mu.Lock()
	m.running[appID] = cmd
	m.mu.Unlock()

	go func() {
		_ = cmd.Wait()
		if logFile != nil {
			_ = logFile.Close()
		}
		m.mu.Lock()
		delete(m.running, appID)
		m.mu.Unlock()
	}()

	return nil
}

// Stop stops a running app by id. No-op if not running.
func (m *Manager) Stop(appID string) error {
	m.mu.RLock()
	cmd := m.running[appID]
	m.mu.RUnlock()
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	killCmdTree(cmd)
	return nil
}

// StopAll terminates all running app processes and dev servers.
func (m *Manager) StopAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.running))
	for appID := range m.running {
		ids = append(ids, appID)
	}
	m.mu.RUnlock()

	for _, appID := range ids {
		_ = m.Stop(appID)
	}
	m.StopAllDev()
}

// RunningIDs returns ids of all currently running apps.
func (m *Manager) RunningIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.running))
	for appID := range m.running {
		ids = append(ids, appID)
	}
	return ids
}
