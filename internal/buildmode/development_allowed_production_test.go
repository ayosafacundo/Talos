//go:build production

package buildmode_test

import (
	"testing"

	"Talos/internal/buildmode"
)

func TestDevelopmentAllowed_IgnoresEnvInProductionBuild(t *testing.T) {
	t.Setenv("TALOS_DEV_MODE", "1")
	if buildmode.DevelopmentAllowed() {
		t.Fatal("expected false in production-tagged build even when TALOS_DEV_MODE=1")
	}
}
