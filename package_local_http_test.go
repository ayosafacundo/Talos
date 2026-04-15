package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"

	"Talos/internal/manifest"
	"Talos/internal/packages"
	"Talos/internal/security"
)

func TestReadAPIPortFromPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	good := filepath.Join(dir, "api-port.txt")
	if err := os.WriteFile(good, []byte("8080\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	port, err := readAPIPortFromPath(good)
	if err != nil || port != "8080" {
		t.Fatalf("got %q %v", port, err)
	}

	missing := filepath.Join(dir, "nope.txt")
	_, err = readAPIPortFromPath(missing)
	if !errors.Is(err, ErrPackageSidecarNotReady) {
		t.Fatalf("expected ErrPackageSidecarNotReady, got %v", err)
	}

	bad := filepath.Join(dir, "bad.txt")
	if err := os.WriteFile(bad, []byte("12a34"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = readAPIPortFromPath(bad)
	if err == nil || !strings.Contains(err.Error(), "invalid api-port") {
		t.Fatalf("expected invalid api-port error, got %v", err)
	}
}

func TestIsDialFailure(t *testing.T) {
	t.Parallel()
	if !isDialFailure(&net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}) {
		t.Fatal("expected dial OpError to be dial failure")
	}
	if isDialFailure(&net.OpError{Op: "read", Err: errors.New("eof")}) {
		t.Fatal("read OpError should not be classified as dial failure")
	}
	if isDialFailure(nil) {
		t.Fatal("nil should not be dial failure")
	}
}

func testPackageLayout(t *testing.T) (root string, appID string, pkg *packages.PackageInfo) {
	t.Helper()
	root = t.TempDir()
	packagesDir := filepath.Join(root, "Packages")
	appDir := filepath.Join(packagesDir, "TestPkg")
	if err := os.MkdirAll(filepath.Join(appDir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	appID = "app.test.sidecar"
	pkg = &packages.PackageInfo{
		DirPath: appDir,
		DirName: "TestPkg",
		Manifest: &manifest.Definition{
			ID: appID,
		},
	}
	return root, appID, pkg
}

func TestPackageLocalHTTP_BinaryResponseUsesBase64(t *testing.T) {
	t.Parallel()
	root, appID, pkgInfo := testPackageLayout(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0, 255, 0, 10})
	}))
	t.Cleanup(srv.Close)

	addr := srv.Listener.Addr().(*net.TCPAddr)
	portFile := filepath.Join(root, "Packages", "TestPkg", "data", "api-port.txt")
	if err := os.WriteFile(portFile, []byte(fmt.Sprintf("%d", addr.Port)), 0o600); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.rootDir = root
	app.packagesDir = filepath.Join(root, "Packages")
	app.permissions = security.NewPermissions(func(string, string, string) (bool, string) { return true, "" })
	app.scopeManager = security.NewScopeManager(app.packagesDir, app.permissions)
	app.mu.Lock()
	app.packages[appID] = pkgInfo
	app.mu.Unlock()

	resp, err := app.PackageLocalHTTP(appID, "GET", "/api/bin", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != 200 {
		t.Fatalf("status=%d", resp.Status)
	}
	if resp.BodyBase64 == "" {
		t.Fatal("expected body_base64 for binary payload")
	}
	if resp.Body != "" {
		t.Fatalf("expected empty body when using base64, got %q", resp.Body)
	}
}

func TestPackageLocalHTTP_Success(t *testing.T) {
	t.Parallel()
	root, appID, pkgInfo := testPackageLayout(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(srv.Close)

	addr := srv.Listener.Addr().(*net.TCPAddr)
	portFile := filepath.Join(root, "Packages", "TestPkg", "data", "api-port.txt")
	if err := os.WriteFile(portFile, []byte(fmt.Sprintf("%d", addr.Port)), 0o600); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.rootDir = root
	app.packagesDir = filepath.Join(root, "Packages")
	app.permissions = security.NewPermissions(func(string, string, string) (bool, string) { return true, "" })
	app.scopeManager = security.NewScopeManager(app.packagesDir, app.permissions)
	app.mu.Lock()
	app.packages[appID] = pkgInfo
	app.mu.Unlock()

	resp, err := app.PackageLocalHTTP(appID, "GET", "/api/health", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != 200 || !strings.Contains(resp.Body, "ok") {
		t.Fatalf("bad response: %+v", resp)
	}
	if got := app.getCachedSidecarPort(appID); got == "" {
		t.Fatal("expected cached port after success")
	}
}

func TestPackageLocalHTTP_NotRunningRemovesStalePortFile(t *testing.T) {
	t.Parallel()
	root, appID, pkgInfo := testPackageLayout(t)

	portFile := filepath.Join(root, "Packages", "TestPkg", "data", "api-port.txt")
	// Port 1 is almost always refused when nothing is listening.
	if err := os.WriteFile(portFile, []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.rootDir = root
	app.packagesDir = filepath.Join(root, "Packages")
	app.permissions = security.NewPermissions(func(string, string, string) (bool, string) { return true, "" })
	app.scopeManager = security.NewScopeManager(app.packagesDir, app.permissions)
	app.mu.Lock()
	app.packages[appID] = pkgInfo
	app.mu.Unlock()

	resp, err := app.PackageLocalHTTP(appID, "GET", "/api/health", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != http.StatusServiceUnavailable || !strings.Contains(resp.Body, "talos_transport") {
		t.Fatalf("expected synthetic 503 transport body, got status=%d body=%q", resp.Status, resp.Body)
	}
	if !strings.Contains(resp.Body, "not_running") {
		t.Fatalf("expected not_running code in body: %s", resp.Body)
	}
	if _, statErr := os.Stat(portFile); !os.IsNotExist(statErr) {
		t.Fatalf("expected api-port.txt removed, stat err=%v", statErr)
	}
}

func TestPackageLocalHTTP_RetriesDialWhenRunning(t *testing.T) {
	t.Parallel()
	root, appID, pkgInfo := testPackageLayout(t)

	portFile := filepath.Join(root, "Packages", "TestPkg", "data", "api-port.txt")
	var mu sync.Mutex
	attempts := 0
	app := NewApp()
	app.rootDir = root
	app.packagesDir = filepath.Join(root, "Packages")
	app.permissions = security.NewPermissions(func(string, string, string) (bool, string) { return true, "" })
	app.scopeManager = security.NewScopeManager(app.packagesDir, app.permissions)
	app.mu.Lock()
	app.packages[appID] = pkgInfo
	app.mu.Unlock()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`ok`))
	}))
	t.Cleanup(srv.Close)
	goodPort := strings.TrimPrefix(srv.Listener.Addr().String(), "127.0.0.1:")

	app.packageLocalHTTPTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		n := attempts
		attempts++
		mu.Unlock()
		if n < 2 {
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
		}
		u := *req.URL
		u.Host = net.JoinHostPort("127.0.0.1", goodPort)
		req2, _ := http.NewRequest(req.Method, u.String(), req.Body)
		req2.Header = req.Header.Clone()
		return http.DefaultTransport.RoundTrip(req2)
	})

	if err := os.WriteFile(portFile, []byte(goodPort), 0o600); err != nil {
		t.Fatal(err)
	}

	// Pretend sidecar process is still managed so we retry instead of ErrPackageSidecarNotRunning.
	app.testSidecarRunningIDs = []string{appID}

	resp, err := app.PackageLocalHTTP(appID, "GET", "/api/health", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != 200 {
		t.Fatalf("status=%d", resp.Status)
	}
	mu.Lock()
	n := attempts
	mu.Unlock()
	if n < 3 {
		t.Fatalf("expected at least 3 transport attempts, got %d", n)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
