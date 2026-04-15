package buildmode

import (
	"os"
	"strings"
)

// DevelopmentAllowed is true when the process environment requests machine-level
// developer overrides (TALOS_DEV_MODE=1). User-controlled Developer mode in Settings
// is handled in the App layer (effective development) and OR-ed with this.
func DevelopmentAllowed() bool {
	return strings.TrimSpace(os.Getenv("TALOS_DEV_MODE")) == "1"
}
