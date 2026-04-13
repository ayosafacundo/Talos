//go:build production

package buildmode

// DevelopmentAllowed is always false in release binaries (built with -tags=production).
func DevelopmentAllowed() bool {
	return false
}
