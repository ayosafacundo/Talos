package main

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"Talos/internal/manifest"
	"Talos/internal/packages"
)

func TestFileURLToLocalPath(t *testing.T) {
	t.Parallel()
	if goruntime.GOOS == "windows" {
		got := fileURLToLocalPath("file:///C:/Users/test/icon.png")
		want := filepath.Clean(`C:\Users\test\icon.png`)
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
		return
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "icon.png")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileURL := "file://" + filepath.ToSlash(p)
	got := fileURLToLocalPath(fileURL)
	if got != p {
		t.Fatalf("got %q want %q", got, p)
	}
}

func TestManifestRelativeIconPath(t *testing.T) {
	t.Parallel()
	pkgDir := filepath.Join(t.TempDir(), "OSRS-GE-App")
	if err := os.MkdirAll(filepath.Join(pkgDir, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(pkgDir, "dist", "icon.png")
	if err := os.WriteFile(want, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []string{"dist/icon.png", "/dist/icon.png", "./dist/icon.png"}
	if goruntime.GOOS == "windows" {
		cases = append(cases, `dist\icon.png`)
	}
	for _, raw := range cases {
		got, ok := manifestRelativeIconPath(pkgDir, raw)
		if !ok || got != want {
			t.Fatalf("%q: got %q ok=%v want %q", raw, got, ok, want)
		}
	}
	if _, ok := manifestRelativeIconPath(pkgDir, "../secret.png"); ok {
		t.Fatal("expected escape to be rejected")
	}
}

func TestResolveManifestIcon_RelativeDistPNG(t *testing.T) {
	t.Parallel()
	pkgDir := filepath.Join(t.TempDir(), "OSRS-GE-App")
	if err := os.MkdirAll(filepath.Join(pkgDir, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(pkgDir, "dist", "icon.png")
	if err := os.WriteFile(p, []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01"), 0o600); err != nil {
		t.Fatal(err)
	}
	pkg := &packages.PackageInfo{
		DirPath: pkgDir,
		Manifest: &manifest.Definition{
			Icon: "dist/icon.png",
		},
	}
	out := resolveManifestIcon(pkg)
	if !strings.HasPrefix(out, "data:image/png;base64,") {
		pl := 50
		if len(out) < pl {
			pl = len(out)
		}
		t.Fatalf("expected embedded png, got %q", out[:pl])
	}
}

func TestResolveManifestIcon_FileURLPNG(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "app.png")
	// Minimal PNG header so DetectContentType / sniff is image/png
	if err := os.WriteFile(pngPath, []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileURL := "file://" + filepath.ToSlash(pngPath)
	pkg := &packages.PackageInfo{
		DirPath: dir,
		Manifest: &manifest.Definition{
			Icon: fileURL,
		},
	}
	out := resolveManifestIcon(pkg)
	if out == "" || out == fileURL {
		t.Fatalf("expected data URL, got %q", out)
	}
	if !strings.HasPrefix(out, "data:image/png;base64,") {
		prefixLen := 40
		if len(out) < prefixLen {
			prefixLen = len(out)
		}
		t.Fatalf("expected data:image/png data URL, got prefix %q", out[:prefixLen])
	}
}
