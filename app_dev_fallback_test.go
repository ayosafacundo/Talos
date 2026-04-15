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

func TestHttpDevEndpointReachable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ts.Close)
	if !httpDevEndpointReachable(ts.URL) {
		t.Fatal("expected test server to be reachable")
	}
	if httpDevEndpointReachable("http://127.0.0.1:59199") {
		t.Fatal("expected closed port to be unreachable")
	}
}

func TestManifestDevIFrameURL_FallbackWhenDevUnreachable(t *testing.T) {
	t.Setenv("TALOS_DEV_MODE", "1")

	dir := t.TempDir()
	distDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<!doctype html><html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	pkg := &packages.PackageInfo{
		DirName: "FallbackPkg",
		DirPath: dir,
		Manifest: &manifest.Definition{
			ID:       "app.dev.fallback.test",
			Name:     "Fallback",
			WebEntry: "dist/index.html",
			Development: &manifest.Development{
				URL:            "http://127.0.0.1:59199",
				AllowedOrigins: []string{"http://127.0.0.1:59199"},
			},
		},
	}

	u, origins, dev := app.manifestDevIFrameURL(pkg)
	if dev {
		t.Fatalf("expected packaged fallback (dev=false), got dev=true url=%q", u)
	}
	if len(origins) > 0 {
		t.Fatalf("expected empty allowlist for packaged, got %v", origins)
	}
	if !strings.Contains(u, "/talos-pkg/") || !strings.Contains(u, "index.html") {
		t.Fatalf("expected talos-pkg web entry URL, got %q", u)
	}
}

func TestManifestDevIFrameURL_CommandOnlyUsesResolvedOverride(t *testing.T) {
	t.Setenv("TALOS_DEV_MODE", "1")

	dir := t.TempDir()
	distDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<!doctype html><html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ts.Close)

	app := NewApp()
	pkg := &packages.PackageInfo{
		DirName: "CmdOnly",
		DirPath: dir,
		Manifest: &manifest.Definition{
			ID:       "app.dev.cmdonly.test",
			Name:     "CmdOnly",
			WebEntry: "dist/index.html",
			Development: &manifest.Development{
				Command:        []string{"npm", "run", "dev"},
				AllowedOrigins: []string{"http://127.0.0.1:59998"},
			},
		},
	}

	u0, _, _ := app.manifestDevIFrameURL(pkg)
	if u0 != "about:blank" {
		t.Fatalf("before resolve, expected about:blank, got %q", u0)
	}

	app.setDevResolvedURL(pkg.Manifest.ID, ts.URL+"/")

	u1, origins, dev := app.manifestDevIFrameURL(pkg)
	if !dev {
		t.Fatal("expected dev mode true")
	}
	if u1 != ts.URL+"/" {
		t.Fatalf("expected resolved dev URL %q, got %q", ts.URL+"/", u1)
	}
	if len(origins) == 0 {
		t.Fatalf("expected origins merged from override, got empty")
	}
}

func TestManifestDevIFrameURL_UsesDevWhenReachable(t *testing.T) {
	t.Setenv("TALOS_DEV_MODE", "1")

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
	devURL := ts.URL + "/"
	pkg := &packages.PackageInfo{
		DirName: "LivePkg",
		DirPath: dir,
		Manifest: &manifest.Definition{
			ID:       "app.dev.live.test",
			Name:     "Live",
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
		t.Fatal("expected non-empty allowed_origins in dev")
	}
}
