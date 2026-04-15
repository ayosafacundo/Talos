package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Talos/internal/manifest"
	"Talos/internal/packages"
)

func TestDevelopmentEnabledForPackage_PerDirectoryWithoutEnv(t *testing.T) {
	t.Setenv("TALOS_DEV_MODE", "")
	app := NewApp()
	pkg := &packages.PackageInfo{
		DirName: "MyPkg",
		DirPath: "/tmp/MyPkg",
		Manifest: &manifest.Definition{
			ID:   "app.perdir",
			Name: "P",
			Development: &manifest.Development{
				URL: "http://127.0.0.1:59999/",
			},
		},
	}
	if app.developmentEnabledForPackage(pkg) {
		t.Fatal("expected development off when no toggle and no env")
	}
	app.devMu.Lock()
	app.prefsDevModeByDir["MyPkg"] = true
	app.devMu.Unlock()
	if !app.developmentEnabledForPackage(pkg) {
		t.Fatal("expected development on for toggled directory")
	}
}

func TestManifestDevIFrameURL_UsesPackagedWhenPerDirOffAndNoEnv(t *testing.T) {
	t.Setenv("TALOS_DEV_MODE", "")

	dir := t.TempDir()
	distDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<!doctype html><html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.rootDir = dir
	app.packagesDir = filepath.Join(dir, "Packages")
	pkg := &packages.PackageInfo{
		DirName: "PerDirOff",
		DirPath: dir,
		Manifest: &manifest.Definition{
			ID:       "app.perdir.off",
			Name:     "Off",
			WebEntry: "dist/index.html",
			Development: &manifest.Development{
				URL:            "http://127.0.0.1:59199",
				AllowedOrigins: []string{"http://127.0.0.1:59199"},
			},
		},
	}

	u, origins, dev := app.manifestDevIFrameURL(pkg)
	if dev {
		t.Fatalf("expected production iframe, dev=%v url=%q", dev, u)
	}
	if len(origins) > 0 {
		t.Fatalf("expected no origins, got %v", origins)
	}
	if !strings.Contains(u, "/talos-pkg/") {
		t.Fatalf("expected packaged URL, got %q", u)
	}
}

func TestManifestDevIFrameURL_UsesDevWhenPerDirOnWithoutEnv(t *testing.T) {
	t.Setenv("TALOS_DEV_MODE", "")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	distDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<!doctype html><html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.devMu.Lock()
	app.prefsDevModeByDir["PerDirOn"] = true
	app.devMu.Unlock()

	devURL := ts.URL + "/"
	pkg := &packages.PackageInfo{
		DirName: "PerDirOn",
		DirPath: dir,
		Manifest: &manifest.Definition{
			ID:       "app.perdir.on",
			Name:     "On",
			WebEntry: "dist/index.html",
			Development: &manifest.Development{
				URL:            devURL,
				AllowedOrigins: []string{ts.URL},
			},
		},
	}

	u, origins, dev := app.manifestDevIFrameURL(pkg)
	if !dev {
		t.Fatalf("expected dev mode, got dev=false url=%q", u)
	}
	if u != devURL {
		t.Fatalf("expected dev URL %q, got %q", devURL, u)
	}
	if len(origins) == 0 {
		t.Fatal("expected origins")
	}
}
