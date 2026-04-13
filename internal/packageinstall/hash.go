package packageinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// HashManifest records SHA-256 of every file under a package directory (for tamper detection).
type HashManifest struct {
	AppID   string            `json:"app_id"`
	Root    string            `json:"root"`
	Files   map[string]string `json:"files"` // relative path -> hex sha256
}

// WriteHashManifest walks packageRoot and writes JSON to destPath.
func WriteHashManifest(appID, packageRoot, destPath string) error {
	files := make(map[string]string)
	err := filepath.WalkDir(packageRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(packageRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "..") {
			return fmt.Errorf("unexpected path outside root: %s", rel)
		}
		h, err := hashFile(path)
		if err != nil {
			return err
		}
		files[rel] = h
		return nil
	})
	if err != nil {
		return err
	}
	m := HashManifest{AppID: appID, Root: packageRoot, Files: files}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destPath, raw, 0o644)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ReadHashManifest loads a JSON hash manifest from disk.
func ReadHashManifest(path string) (*HashManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m HashManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// VerifyHashManifest checks current files against a saved manifest. Returns nil if OK.
func VerifyHashManifest(packageRoot string, manifest *HashManifest) error {
	if manifest == nil {
		return errors.New("packageinstall: nil manifest")
	}
	for rel, want := range manifest.Files {
		path := filepath.Join(packageRoot, filepath.FromSlash(rel))
		got, err := hashFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		if got != want {
			return fmt.Errorf("tampered or missing file %q: hash mismatch", rel)
		}
	}
	return nil
}
