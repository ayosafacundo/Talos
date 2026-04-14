package packageinstall

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TrustStatus summarizes integrity + optional signature for a package.
type TrustStatus string

const (
	TrustUnknown       TrustStatus = "unknown"        // no hash manifest on disk
	TrustOK            TrustStatus = "ok"             // hash manifest matches files
	TrustTampered      TrustStatus = "tampered"       // hash mismatch
	TrustUnsigned      TrustStatus = "unsigned"       // content OK, no .talos-signature
	TrustSignedOK      TrustStatus = "signed_ok"      // hash OK + valid Ed25519 signature
	TrustSignedInvalid TrustStatus = "signed_invalid" // signature present but invalid / no matching key
)

const signatureFileName = ".talos-signature"

// EvaluateTrust loads the hash manifest for appID under hashDir, verifies file hashes, then
// optionally verifies a detached Ed25519 signature in packageRoot/.talos-signature over the
// canonical JSON bytes of the hash manifest (same file on disk).
func EvaluateTrust(packageRoot, hashManifestPath, trustedKeysDir string) (TrustStatus, error) {
	raw, err := os.ReadFile(hashManifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return TrustUnknown, nil
		}
		return TrustUnknown, err
	}
	var hm HashManifest
	if err := json.Unmarshal(raw, &hm); err != nil {
		return TrustUnknown, err
	}
	if err := VerifyHashManifest(packageRoot, &hm); err != nil {
		return TrustTampered, nil
	}

	sigPath := filepath.Join(packageRoot, signatureFileName)
	sigRaw, err := os.ReadFile(sigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return TrustUnsigned, nil
		}
		return TrustUnsigned, err
	}
	sig, err := parseSignatureFile(sigRaw)
	if err != nil {
		return TrustSignedInvalid, nil
	}
	keys, err := loadEd25519PublicKeys(trustedKeysDir)
	if err != nil || len(keys) == 0 {
		return TrustSignedInvalid, nil
	}
	msg := canonicalHashManifestBytes(raw)
	for _, pub := range keys {
		if ed25519.Verify(pub, msg, sig) {
			return TrustSignedOK, nil
		}
	}
	return TrustSignedInvalid, nil
}

func canonicalHashManifestBytes(storedJSON []byte) []byte {
	var m HashManifest
	if err := json.Unmarshal(storedJSON, &m); err != nil {
		return storedJSON
	}
	b, err := json.Marshal(m)
	if err != nil {
		return storedJSON
	}
	return b
}

func parseSignatureFile(raw []byte) ([]byte, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return nil, errors.New("empty signature")
	}
	// Hex-encoded 64-byte signature (128 hex chars).
	if len(s) == 128 {
		return hex.DecodeString(s)
	}
	// Raw file of 64 bytes.
	if len(raw) == ed25519.SignatureSize {
		return raw, nil
	}
	return hex.DecodeString(s)
}

func loadEd25519PublicKeys(dir string) ([]ed25519.PublicKey, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []ed25519.PublicKey
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".pub") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		raw = []byte(strings.TrimSpace(string(raw)))
		pub, err := decodePublicKey(raw)
		if err != nil {
			continue
		}
		out = append(out, pub)
	}
	return out, nil
}

func decodePublicKey(raw []byte) (ed25519.PublicKey, error) {
	if len(raw) == ed25519.PublicKeySize {
		return ed25519.PublicKey(raw), nil
	}
	// Hex
	if len(raw) == ed25519.PublicKeySize*2 {
		b, err := hex.DecodeString(string(raw))
		if err != nil {
			return nil, err
		}
		if len(b) == ed25519.PublicKeySize {
			return ed25519.PublicKey(b), nil
		}
	}
	return nil, fmt.Errorf("invalid ed25519 public key length %d", len(raw))
}

// WriteSignatureFile writes .talos-signature for testing or release tooling (hex-encoded signature).
func WriteSignatureFile(packageRoot string, sig []byte) error {
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("signature must be %d bytes", ed25519.SignatureSize)
	}
	path := filepath.Join(packageRoot, signatureFileName)
	return os.WriteFile(path, []byte(hex.EncodeToString(sig)+"\n"), 0o644)
}
