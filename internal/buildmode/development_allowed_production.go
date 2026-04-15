//go:build production

package buildmode

// DevelopmentAllowed is always false for release binaries (built with -tags=production).
// Per-directory development mode is controlled only via persisted user settings.
func DevelopmentAllowed() bool {
	return false
}
