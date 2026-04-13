package packageinstall

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"Talos/internal/manifest"
)

// Limits for zip extraction (MVP hardening).
const (
	MaxZipBytes     = 80 << 20 // 80 MiB
	MaxZipFiles     = 20000
	MaxSingleUncomp = 40 << 20
)

var allowedUnpackExt = map[string]struct{}{
	".html": {}, ".htm": {}, ".js": {}, ".mjs": {}, ".cjs": {},
	".css": {}, ".json": {}, ".yaml": {}, ".yml": {}, ".md": {},
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}, ".svg": {},
	".ico": {}, ".txt": {}, ".woff": {}, ".woff2": {}, ".ttf": {},
	".map": {}, ".ts": {}, ".tsx": {}, ".vue": {}, ".svelte": {},
	".wasm": {},
}

func extAllowed(name string) bool {
	slash := filepath.ToSlash(name)
	if strings.Contains(slash, "/bin/") || strings.HasPrefix(slash, "bin/") {
		return true
	}
	base := filepath.Base(name)
	if base == "manifest.yaml" || base == "manifest.yml" {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return false
	}
	_, ok := allowedUnpackExt[ext]
	return ok
}

// ExtractZipSecure extracts a zip into destDir with ZipSlip protection and extension allowlisting.
func ExtractZipSecure(r *zip.Reader, destDir string) error {
	if len(r.File) > MaxZipFiles {
		return fmt.Errorf("zip: too many files (%d)", len(r.File))
	}
	destRoot, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}
	for _, f := range r.File {
		if f.UncompressedSize64 > MaxSingleUncomp {
			return fmt.Errorf("zip: entry too large: %s", f.Name)
		}
		clean := filepath.Clean(f.Name)
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			return fmt.Errorf("zip: illegal path %q", f.Name)
		}
		target := filepath.Join(destRoot, clean)
		targAbs, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(targAbs+string(filepath.Separator), destRoot+string(filepath.Separator)) && targAbs != destRoot {
			return fmt.Errorf("zip: zipslip blocked: %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targAbs, 0o755); err != nil {
				return err
			}
			continue
		}
		if !extAllowed(f.Name) {
			return fmt.Errorf("zip: disallowed file type: %q", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(targAbs), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(targAbs, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			_ = rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		_ = rc.Close()
		_ = out.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// InstallFromZipReader extracts a zip into packagesRoot under a new folder derived from manifest name.
func InstallFromZipReader(zr *zip.Reader, packagesRoot, hashDir string) (appID string, outDir string, err error) {
	var total uint64
	for _, f := range zr.File {
		total += f.UncompressedSize64
	}
	if total > MaxZipBytes {
		return "", "", fmt.Errorf("zip: total uncompressed size exceeds limit")
	}
	staging, err := os.MkdirTemp("", "talos-pkg-*")
	if err != nil {
		return "", "", err
	}
	defer func() { _ = os.RemoveAll(staging) }()

	if err := ExtractZipSecure(zr, staging); err != nil {
		return "", "", err
	}
	manifestPath, err := findManifest(staging)
	if err != nil {
		return "", "", err
	}
	def, err := manifest.ParseFile(manifestPath)
	if err != nil {
		return "", "", err
	}
	appID = def.ID
	if appID == "" {
		return "", "", errors.New("packageinstall: manifest id required")
	}
	packageRoot := filepath.Dir(manifestPath)
	dest := filepath.Join(packagesRoot, sanitizeDirName(def.Name))
	if err := os.RemoveAll(dest); err != nil && !os.IsNotExist(err) {
		return "", "", err
	}
	if err := copyDir(packageRoot, dest); err != nil {
		return "", "", err
	}
	hashPath := filepath.Join(hashDir, appID+".json")
	if err := WriteHashManifest(appID, dest, hashPath); err != nil {
		return "", "", err
	}
	return appID, dest, nil
}

func findManifest(root string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := d.Name()
		if base == "manifest.yaml" || base == "manifest.yml" {
			if found != "" {
				return fmt.Errorf("multiple manifests under archive")
			}
			found = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", errors.New("packageinstall: no manifest.yaml in archive")
	}
	return found, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// InstallFromZipFile opens a local zip and installs it into packagesRoot.
func InstallFromZipFile(path, packagesRoot, hashDir string) (appID string, outDir string, err error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", "", err
	}
	defer r.Close()
	return InstallFromZipReader(&r.Reader, packagesRoot, hashDir)
}

func sanitizeDirName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return "package"
	}
	s = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		default:
			return r
		}
	}, s)
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}
