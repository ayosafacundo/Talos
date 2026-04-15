package buildmode_test

import (
	"testing"

	"Talos/internal/buildmode"
)

func TestDevelopmentAllowed_RespectsEnv(t *testing.T) {
	t.Setenv("TALOS_DEV_MODE", "")
	if buildmode.DevelopmentAllowed() {
		t.Fatal("expected false when TALOS_DEV_MODE unset")
	}

	t.Setenv("TALOS_DEV_MODE", "1")
	if !buildmode.DevelopmentAllowed() {
		t.Fatal("expected true when TALOS_DEV_MODE=1")
	}
}
