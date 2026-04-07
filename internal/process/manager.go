package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"Talos/internal/packages"
)

// Manager manages tiny app binary lifecycles.
type Manager struct {
	mu      sync.RWMutex
	running map[string]*exec.Cmd
}

func NewManager() *Manager {
	return &Manager{
		running: make(map[string]*exec.Cmd),
	}
}

// Start launches a package binary if not already running.
func (m *Manager) Start(ctx context.Context, pkg *packages.PackageInfo) error {
	if pkg == nil || pkg.Manifest == nil {
		return errors.New("process: package manifest is required")
	}

	appID := pkg.Manifest.ID
	m.mu.RLock()
	if _, ok := m.running[appID]; ok {
		m.mu.RUnlock()
		return fmt.Errorf("process: app %q already running", appID)
	}
	m.mu.RUnlock()

	binPath := filepath.Join(pkg.DirPath, filepath.Clean(pkg.Manifest.Binary))
	st, err := os.Stat(binPath)
	if err != nil {
		return fmt.Errorf("process: stat %q failed: %w", binPath, err)
	}
	if st.IsDir() {
		return fmt.Errorf("process: binary path %q is a directory", binPath)
	}

	cmd := exec.CommandContext(ctx, binPath)
	cmd.Dir = pkg.DirPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	configureCmd(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("process: start %q failed: %w", appID, err)
	}

	m.mu.Lock()
	m.running[appID] = cmd
	m.mu.Unlock()

	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		delete(m.running, appID)
		m.mu.Unlock()
	}()

	return nil
}

// Stop stops a running app by id.
func (m *Manager) Stop(appID string) error {
	m.mu.RLock()
	cmd := m.running[appID]
	m.mu.RUnlock()
	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("process: app %q is not running", appID)
	}

	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("process: kill %q failed: %w", appID, err)
	}
	return nil
}

// StopAll terminates all running app processes.
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
