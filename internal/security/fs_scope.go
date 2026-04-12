package security

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ScopeManager resolves and validates package-scoped filesystem access.
type ScopeManager struct {
	packagesDir string
	permissions *Permissions
}

func NewScopeManager(packagesDir string, permissions *Permissions) *ScopeManager {
	return &ScopeManager{
		packagesDir: packagesDir,
		permissions: permissions,
	}
}

// ResolvePath enforces /Packages/[AppName]/data default scoping.
// For paths outside data/, caller must hold matching fs permission scope.
func (m *ScopeManager) ResolvePath(appDirName, appID, relativePath string) (string, error) {
	if strings.TrimSpace(relativePath) == "" {
		return "", errors.New("empty relative path")
	}

	baseDataDir := filepath.Clean(filepath.Join(m.packagesDir, appDirName, "data"))
	candidate := filepath.Clean(filepath.Join(baseDataDir, relativePath))
	if hasPathPrefix(candidate, baseDataDir) {
		return candidate, nil
	}

	// Outside app data requires explicit external fs permission.
	if !m.permissions.IsGranted(appID, "fs:external") {
		return "", fmt.Errorf("path escapes app data scope and fs:external is not granted")
	}

	return candidate, nil
}

func hasPathPrefix(path, prefix string) bool {
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+string(filepath.Separator))
}
