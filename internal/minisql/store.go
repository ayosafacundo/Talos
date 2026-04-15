// Package minisql provides a unified host-managed SQLite data plane: one database file per app
// with filesystem isolation. Logical "limited user" semantics are enforced by the host (each app
// only receives its own DSN). A future embedded multi-user server can plug in behind the same env contract.
package minisql

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "modernc.org/sqlite"
)

// Store manages per-app SQLite files under <rootDir>/Temp/minisql/apps/.
type Store struct {
	rootDir string
}

// NewStore creates a minisql store rooted at Talos data directory.
func NewStore(rootDir string) *Store {
	return &Store{rootDir: rootDir}
}

func (s *Store) appsDir() string {
	return filepath.Join(s.rootDir, "Temp", "minisql", "apps")
}

var safeAppID = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeAppID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "unknown"
	}
	return safeAppID.ReplaceAllString(id, "_")
}

// Provision creates the per-app database file if needed and returns a DSN suitable for database/sql
// (driver name "sqlite") and the absolute path for diagnostics.
func (s *Store) Provision(appID string) (dsn string, absPath string, err error) {
	dir := s.appsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	name := sanitizeAppID(appID) + ".sqlite3"
	absPath = filepath.Join(dir, name)
	u := filepath.ToSlash(absPath)
	// modernc sqlite DSN
	open := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", u)
	db, err := sql.Open("sqlite", open)
	if err != nil {
		return "", "", err
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return "", "", err
	}
	dsn = fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", u)
	return dsn, absPath, nil
}

// Revoke removes the per-app database file if present (e.g. package uninstalled).
func (s *Store) Revoke(appID string) error {
	p := filepath.Join(s.appsDir(), sanitizeAppID(appID)+".sqlite3")
	err := os.Remove(p)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
