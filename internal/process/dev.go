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

	"Talos/internal/packages"
)

// StartDev spawns development.command for a package (e.g. npm run dev). Logs to packageLogDir/<id>-dev.log.
func (m *Manager) StartDev(ctx context.Context, pkg *packages.PackageInfo, argv []string) error {
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
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, "NODE_ENV=development")
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
		_ = cmd.Process.Kill()
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
