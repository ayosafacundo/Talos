package security

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadGrants(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "permissions.json")
	in := map[string]map[string]bool{
		"app.test": {
			"fs:external": true,
		},
	}

	if err := SaveGrants(path, in); err != nil {
		t.Fatalf("SaveGrants() error: %v", err)
	}

	out, err := LoadGrants(path)
	if err != nil {
		t.Fatalf("LoadGrants() error: %v", err)
	}
	if !out["app.test"]["fs:external"] {
		t.Fatalf("expected persisted grant")
	}
}
