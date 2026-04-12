package security

import "testing"

func TestPermissions_DefaultDataScopeAllowed(t *testing.T) {
	t.Parallel()

	p := NewPermissions(nil)
	if !p.IsGranted("app.test", ScopeFSData) {
		t.Fatalf("fs:data should be allowed by default")
	}
}

func TestPermissions_RequestWithPrompt(t *testing.T) {
	t.Parallel()

	p := NewPermissions(func(appID, scope, reason string) (bool, string) {
		return true, "approved"
	})
	granted, _ := p.Request("app.test", "net:internet", "sync")
	if !granted {
		t.Fatalf("expected permission granted")
	}
	if !p.IsGranted("app.test", "net:internet") {
		t.Fatalf("expected permission persisted after grant")
	}
}
