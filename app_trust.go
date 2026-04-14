package main

import (
	"os"
	"path/filepath"
	"strings"
)

// ImportTrustedPublisherKey writes a raw Ed25519 public key (32 bytes or 64-char hex) to Temp/trusted_keys/<name>.pub
func (a *App) ImportTrustedPublisherKey(name, keyMaterial string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return os.ErrInvalid
	}
	if strings.ContainsAny(name, `/\:*?"<>|`) {
		return os.ErrInvalid
	}
	keyMaterial = strings.TrimSpace(keyMaterial)
	if keyMaterial == "" {
		return os.ErrInvalid
	}
	dir := a.trustedKeysDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, name+".pub")
	return os.WriteFile(path, []byte(keyMaterial+"\n"), 0o644)
}

// ListTrustedPublisherKeyNames returns base names of .pub files under Temp/trusted_keys.
func (a *App) ListTrustedPublisherKeyNames() []string {
	dir := a.trustedKeysDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(strings.ToLower(n), ".pub") {
			out = append(out, strings.TrimSuffix(n, ".pub"))
		}
	}
	return out
}

// ParanoidPackageTrust is true when TALOS_PARANOID_TRUST=1; Launchpad may refuse to open tampered packages.
func (a *App) ParanoidPackageTrust() bool {
	return strings.TrimSpace(os.Getenv("TALOS_PARANOID_TRUST")) == "1"
}
