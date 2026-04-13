//go:build production

package buildmode_test

import (
	"os"
	"testing"

	"Talos/internal/buildmode"
)

func TestDevelopmentAllowed_ProductionTagAlwaysFalse(t *testing.T) {
	_ = os.Setenv("TALOS_DEV_MODE", "1")
	t.Cleanup(func() { _ = os.Unsetenv("TALOS_DEV_MODE") })

	if buildmode.DevelopmentAllowed() {
		t.Fatal("expected DevelopmentAllowed false when built with -tags=production")
	}
}
