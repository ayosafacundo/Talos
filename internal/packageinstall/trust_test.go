package packageinstall

import (
	"crypto/ed25519"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestEvaluateTrust_UnsignedOK(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	hashPath := filepath.Join(t.TempDir(), "h.json")
	if err := WriteHashManifest("app.x", dir, hashPath); err != nil {
		t.Fatal(err)
	}
	st, err := EvaluateTrust(dir, hashPath, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if st != TrustUnsigned {
		t.Fatalf("got %q want unsigned", st)
	}
}

func TestEvaluateTrust_SignedOK(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	hashPath := filepath.Join(t.TempDir(), "h.json")
	if err := WriteHashManifest("app.x", dir, hashPath); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(hashPath)
	if err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(priv, canonicalHashManifestBytes(raw))
	if err := WriteSignatureFile(dir, sig); err != nil {
		t.Fatal(err)
	}
	keyDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(keyDir, "test.pub"), []byte(hex.EncodeToString(pub)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := EvaluateTrust(dir, hashPath, keyDir)
	if err != nil {
		t.Fatal(err)
	}
	if st != TrustSignedOK {
		t.Fatalf("got %q want signed_ok", st)
	}
}

func TestEvaluateTrust_Tampered(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	hashPath := filepath.Join(t.TempDir(), "h.json")
	if err := WriteHashManifest("app.x", dir, hashPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("mutated"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := EvaluateTrust(dir, hashPath, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if st != TrustTampered {
		t.Fatalf("got %q want tampered", st)
	}
}

func TestEvaluateTrust_SignedInvalid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	hashPath := filepath.Join(t.TempDir(), "h.json")
	if err := WriteHashManifest("app.x", dir, hashPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".talos-signature"), []byte("not-a-valid-signature"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := EvaluateTrust(dir, hashPath, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if st != TrustSignedInvalid {
		t.Fatalf("got %q want signed_invalid", st)
	}
}

