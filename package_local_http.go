package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

// Sentinel errors returned by PackageLocalHTTP for stable iframe handling.
var (
	ErrPackageSidecarNotReady   = errors.New("package sidecar not ready")
	ErrPackageSidecarNotRunning = errors.New("package sidecar not running")
)

// Single bridge request may wait for the sidecar to create data/api-port.txt after StartPackage (fork/exec).
const (
	packageLocalHTTPMaxWaitAttempts = 80
	packageLocalHTTPBackoff         = 100 * time.Millisecond
	// After this many consecutive "port file missing / empty" resolves, return a synthetic 503 so the
	// iframe can poll quickly instead of blocking for the full max-wait window per bridge call.
	packageLocalHTTPNotReadyGiveUpAttempts = 35
)

// packageLocalHTTPSyntheticTransportError returns a 503 JSON body that does not use top-level "error"
// (reserved for fatal DB errors from the sidecar). The iframe treats this as retryable.
func packageLocalHTTPSyntheticTransportError(code, detail string) *PackageLocalHTTPResponse {
	b, err := json.Marshal(map[string]any{
		"talos_transport": map[string]string{"code": code, "detail": detail},
	})
	if err != nil {
		b = []byte(`{"talos_transport":{"code":"internal","detail":"encoding failed"}}`)
	}
	return &PackageLocalHTTPResponse{
		Status:      http.StatusServiceUnavailable,
		ContentType: "application/json",
		Body:        string(b),
	}
}

func readAPIPortFromPath(resolved string) (string, error) {
	portBytes, err := os.ReadFile(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrPackageSidecarNotReady
		}
		return "", err
	}
	port := strings.TrimSpace(string(portBytes))
	if port == "" {
		return "", ErrPackageSidecarNotReady
	}
	for _, c := range port {
		if c < '0' || c > '9' {
			return "", errors.New("invalid api-port file")
		}
	}
	return port, nil
}

func isDialFailure(err error) bool {
	if err == nil {
		return false
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	return false
}

func (a *App) sidecarProcessRunning(appID string) bool {
	if a.testSidecarRunningIDs != nil {
		for _, id := range a.testSidecarRunningIDs {
			if id == appID {
				return true
			}
		}
		return false
	}
	for _, id := range a.processManager.RunningIDs() {
		if id == appID {
			return true
		}
	}
	return false
}

func (a *App) getCachedSidecarPort(appID string) string {
	a.sidecarLoopbackMu.RLock()
	defer a.sidecarLoopbackMu.RUnlock()
	return a.sidecarLoopbackPort[appID]
}

func (a *App) setCachedSidecarPort(appID, port string) {
	a.sidecarLoopbackMu.Lock()
	defer a.sidecarLoopbackMu.Unlock()
	a.sidecarLoopbackPort[appID] = port
}

func (a *App) clearCachedSidecarPort(appID string) {
	a.sidecarLoopbackMu.Lock()
	defer a.sidecarLoopbackMu.Unlock()
	delete(a.sidecarLoopbackPort, appID)
}

func (a *App) removeAPIPortFileForApp(appID string) error {
	resolved, err := a.ResolveScopedPath(appID, "api-port.txt")
	if err != nil {
		return err
	}
	if err := os.Remove(resolved); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (a *App) resolveSidecarPort(appID, resolvedPortPath string) (port string, source string, err error) {
	if a.sidecarProcessRunning(appID) {
		if p := a.getCachedSidecarPort(appID); p != "" {
			return p, "cache", nil
		}
	}
	p, err := readAPIPortFromPath(resolvedPortPath)
	if err != nil {
		return "", "", err
	}
	return p, "file", nil
}

func doPackageLocalHTTPOnce(client *http.Client, method, port, requestPath, body string) (*PackageLocalHTTPResponse, error) {
	fullURL := "http://127.0.0.1:" + port + requestPath
	if _, err := url.Parse(fullURL); err != nil {
		return nil, err
	}
	var bodyReader io.Reader
	if method == http.MethodPost && body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}
	if method == http.MethodPost && body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	if isTextualPackageLocalContentType(ct) && utf8.Valid(bodyBytes) {
		return &PackageLocalHTTPResponse{
			Status:      resp.StatusCode,
			ContentType: ct,
			Body:        string(bodyBytes),
		}, nil
	}
	return &PackageLocalHTTPResponse{
		Status:      resp.StatusCode,
		ContentType: ct,
		BodyBase64:  base64.StdEncoding.EncodeToString(bodyBytes),
	}, nil
}

func isTextualPackageLocalContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "" {
		return false
	}
	if strings.HasPrefix(ct, "text/") {
		return true
	}
	if strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "application/problem+json") {
		return true
	}
	if strings.Contains(ct, "xml") && (strings.HasPrefix(ct, "application/") || strings.HasPrefix(ct, "text/")) {
		return true
	}
	return false
}

// PackageLocalHTTP forwards a request to a package binary's loopback HTTP server (port from cache or data/api-port.txt).
func (a *App) PackageLocalHTTP(appID, method, requestPath, body string) (*PackageLocalHTTPResponse, error) {
	method = strings.TrimSpace(strings.ToUpper(method))
	if method != http.MethodGet && method != http.MethodPost {
		return nil, fmt.Errorf("unsupported method %q", method)
	}
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" || !strings.HasPrefix(requestPath, "/api/") {
		return nil, errors.New("path must start with /api/")
	}
	if strings.Contains(requestPath, "..") {
		return nil, errors.New("invalid path")
	}
	if len(requestPath) > 16384 {
		return nil, errors.New("path too long")
	}

	a.mu.RLock()
	pkg := a.packages[appID]
	a.mu.RUnlock()
	if pkg == nil || pkg.Manifest == nil {
		return nil, errors.New("package not found")
	}
	resolvedPortPath, err := a.scopeManager.ResolvePath(pkg.DirName, appID, "api-port.txt")
	if err != nil {
		return nil, fmt.Errorf("resolve api-port: %w", err)
	}

	client := &http.Client{
		Timeout:   120 * time.Second,
		Transport: a.packageLocalHTTPTransport,
	}

	var lastErr error
	notReadyPasses := 0
	for attempt := 0; attempt < packageLocalHTTPMaxWaitAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(packageLocalHTTPBackoff)
		}

		port, portSource, err := a.resolveSidecarPort(appID, resolvedPortPath)
		if err != nil {
			lastErr = err
			if errors.Is(err, ErrPackageSidecarNotReady) {
				notReadyPasses++
				if notReadyPasses >= packageLocalHTTPNotReadyGiveUpAttempts {
					return packageLocalHTTPSyntheticTransportError("not_ready", ErrPackageSidecarNotReady.Error()), nil
				}
				if attempt == 0 || attempt%20 == 19 {
					a.logInfo("package-local-http", fmt.Sprintf("app_id=%q attempt=%d err_class=not_ready port_source=%s", appID, attempt+1, portSource))
				}
				continue
			}
			return nil, err
		}
		notReadyPasses = 0

		resp, err := doPackageLocalHTTPOnce(client, method, port, requestPath, body)
		if err == nil {
			a.setCachedSidecarPort(appID, port)
			return resp, nil
		}

		lastErr = err
		a.clearCachedSidecarPort(appID)

		if !isDialFailure(err) {
			a.logInfo("package-local-http", fmt.Sprintf("app_id=%q attempt=%d port=%s port_source=%s err_class=non_dial err=%q", appID, attempt+1, port, portSource, err.Error()))
			return nil, fmt.Errorf("sidecar request failed: %w", err)
		}

		a.logInfo("package-local-http", fmt.Sprintf("app_id=%q attempt=%d port=%s port_source=%s err_class=dial err=%q", appID, attempt+1, port, portSource, err.Error()))

		if !a.sidecarProcessRunning(appID) {
			if remErr := a.removeAPIPortFileForApp(appID); remErr != nil {
				a.logError("package-local-http", fmt.Sprintf("app_id=%q remove api-port: %v", appID, remErr))
			}
			return packageLocalHTTPSyntheticTransportError("not_running", ErrPackageSidecarNotRunning.Error()), nil
		}
	}

	if lastErr != nil {
		if errors.Is(lastErr, ErrPackageSidecarNotReady) {
			return packageLocalHTTPSyntheticTransportError("not_ready", ErrPackageSidecarNotReady.Error()), nil
		}
		return nil, fmt.Errorf("sidecar request failed: %w", lastErr)
	}
	return packageLocalHTTPSyntheticTransportError("not_ready", ErrPackageSidecarNotReady.Error()), nil
}
