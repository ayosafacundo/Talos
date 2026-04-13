//go:build !production

package buildmode

import (
	"os"
	"strings"
)

// DevelopmentAllowed is true only when Talos is run in developer mode (e.g. make dev / TALOS_DEV_MODE=1).
// Release builds use the production tag; see buildmode_prod.go.
func DevelopmentAllowed() bool {
	return strings.TrimSpace(os.Getenv("TALOS_DEV_MODE")) == "1"
}
