//go:build !production

package buildmode

import (
	"os"
	"strings"
)

// DevelopmentAllowed is true when this binary was built without the "production" tag and
// TALOS_DEV_MODE=1. Release installs (wails build -tags=production) never honor this env var.
func DevelopmentAllowed() bool {
	return strings.TrimSpace(os.Getenv("TALOS_DEV_MODE")) == "1"
}
