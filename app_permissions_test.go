package main

import (
	"context"
	"sync"
	"testing"

	"Talos/internal/manifest"
	"Talos/internal/packages"
	"Talos/internal/security"
)

func TestPromptDeclaredNetworkPermissionsPromptsOnlyNetScopes(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.ctx = context.Background()
	var mu sync.Mutex
	var requested []string
	app.permissions = security.NewPermissions(func(_ string, scope, _ string) (bool, string) {
		mu.Lock()
		requested = append(requested, scope)
		mu.Unlock()
		return false, "denied"
	})

	pkg := &packages.PackageInfo{
		Manifest: &manifest.Definition{
			ID:          "app.dev.net",
			Permissions: []string{"fs:external", "net:internet", "net:api"},
		},
	}
	app.promptDeclaredNetworkPermissions(pkg)

	mu.Lock()
	defer mu.Unlock()
	if len(requested) != 2 {
		t.Fatalf("expected 2 network prompts, got %d (%v)", len(requested), requested)
	}
}

func TestPromptDeclaredNetworkPermissionsSkipsExistingDecision(t *testing.T) {
	t.Parallel()

	app := NewApp()
	app.ctx = context.Background()
	app.permissions = security.NewPermissions(func(_ string, _ string, _ string) (bool, string) {
		t.Fatal("did not expect prompt when decision already exists")
		return false, "unexpected"
	})
	app.permissions.Set("app.dev.net", "net:internet", false)

	pkg := &packages.PackageInfo{
		Manifest: &manifest.Definition{
			ID:          "app.dev.net",
			Permissions: []string{"net:internet"},
		},
	}
	app.promptDeclaredNetworkPermissions(pkg)
}
