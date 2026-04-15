package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func TestTalosPackageMiddleware_ServesFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	pkgDir := filepath.Join(root, "PkgA")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "hello.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	mw := talosPackageMiddleware(root)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/talos-pkg/PkgA/hello.txt", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestTalosPackageMiddleware_PackagesRootDirForbidden(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mw := talosPackageMiddleware(root)
	h := mw(http.NotFoundHandler())

	// Resolves to the packages root directory after path.Clean — must not be listable.
	req := httptest.NewRequest(http.MethodGet, "/talos-pkg/..", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestTalosPackageMiddleware_ChainsWithAssetServerType(t *testing.T) {
	t.Parallel()
	// Compile-time guard: middleware matches assetserver.Middleware signature.
	var _ assetserver.Middleware = talosPackageMiddleware(t.TempDir())
}
