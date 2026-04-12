package security

import (
	"path/filepath"
	"testing"
)

func TestResolvePathInsideDataScope(t *testing.T) {
	t.Parallel()

	p := NewPermissions(nil)
	m := NewScopeManager("/tmp/Packages", p)

	resolved, err := m.ResolvePath("MyApp", "app.my", "cache/state.json")
	if err != nil {
		t.Fatalf("ResolvePath() error: %v", err)
	}

	expected := filepath.Clean("/tmp/Packages/MyApp/data/cache/state.json")
	if resolved != expected {
		t.Fatalf("expected %q, got %q", expected, resolved)
	}
}

func TestResolvePathEscapeDeniedWithoutPermission(t *testing.T) {
	t.Parallel()

	p := NewPermissions(nil)
	m := NewScopeManager("/tmp/Packages", p)

	if _, err := m.ResolvePath("MyApp", "app.my", "../secrets.txt"); err == nil {
		t.Fatalf("expected scope escape error")
	}
}
