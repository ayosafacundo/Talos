package minisql_test

import (
	"path/filepath"
	"testing"

	"Talos/internal/minisql"
)

func TestProvisionRevoke(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st := minisql.NewStore(dir)
	dsn, abs, err := st.Provision("app.example.test")
	if err != nil {
		t.Fatal(err)
	}
	if dsn == "" || abs == "" {
		t.Fatal("expected dsn and path")
	}
	if _, err := filepath.EvalSymlinks(abs); err != nil {
		t.Fatal(err)
	}
	if err := st.Revoke("app.example.test"); err != nil {
		t.Fatal(err)
	}
}
