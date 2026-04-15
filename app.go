package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"time"

	hubpb "Talos/api/proto/talos/hub/v1"
	"Talos/internal/buildmode"
	"Talos/internal/hub"
	"Talos/internal/minisql"
	"Talos/internal/packageinstall"
	"Talos/internal/packages"
	"Talos/internal/process"
	"Talos/internal/security"
	"Talos/internal/state"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx    context.Context
	cancel context.CancelFunc

	rootDir     string
	packagesDir string

	discovery       *packages.Discovery
	processManager  *process.Manager
	hub             *hub.Server
	stateStore      *state.Store
	permissions     *security.Permissions
	scopeManager    *security.ScopeManager
	permissionsPath string
	logsDir         string
	hostLogPath     string
	devLogPath      string
	packageLogDir   string

	mu       sync.RWMutex
	packages map[string]*packages.PackageInfo

	devResolvedMu  sync.RWMutex
	devResolvedURL map[string]string // effective dev iframe base URL when discovered (overrides manifest until cleared)

	// devReachCache: short-lived probe results so GetInstalledApps does not hammer loopback; short TTL on
	// "packaged" so a late-started Vite is detected within a few seconds.
	devReachMu    sync.Mutex
	devReachCache map[string]devReachOutcome

	logMu   sync.Mutex
	auditMu sync.Mutex

	permissionAuditPath string

	devMu              sync.RWMutex
	prefsDeveloperMode bool
	sqlStore           *minisql.Store

	// sidecarLoopbackPort caches last successful loopback port per app (invalidated on dial failure / stop).
	sidecarLoopbackMu   sync.RWMutex
	sidecarLoopbackPort map[string]string
	// packageLocalHTTPTransport is only set by tests to stub RoundTrip behavior; production leaves it nil.
	packageLocalHTTPTransport http.RoundTripper
	// testSidecarRunningIDs, when non-nil, overrides sidecarProcessRunning (tests only).
	testSidecarRunningIDs []string
}

type ThemeInfo struct {
	Name string `json:"name"`
	File string `json:"file"`
}

type UserPrefs struct {
	Theme         string            `json:"theme"`
	TabColors     map[string]string `json:"tab_colors"`
	DeveloperMode bool              `json:"developer_mode,omitempty"`
}

// PackageLocalHTTPResponse is the result of proxying a package sidecar HTTP request through the host.
// Iframes must not call loopback URLs directly (WebView and timing issues); they use the bridge instead.
type PackageLocalHTTPResponse struct {
	Status      int    `json:"status"`
	ContentType string `json:"content_type"`
	Body        string `json:"body"`
}

// packageAssetURL returns a same-origin path for the package web entry (served by talosPackageMiddleware).
func packageAssetURL(pkg *packages.PackageInfo, webEntry string) string {
	entry := strings.TrimSpace(webEntry)
	if entry == "" || pkg == nil {
		return ""
	}
	webPath := filepath.Join(pkg.DirPath, entry)
	webPath, err := filepath.Abs(filepath.Clean(webPath))
	if err != nil {
		return ""
	}
	pkgRoot, err := filepath.Abs(filepath.Clean(pkg.DirPath))
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(pkgRoot, webPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	dirName := strings.TrimSpace(pkg.DirName)
	if dirName == "" {
		dirName = filepath.Base(pkg.DirPath)
	}
	relSlash := filepath.ToSlash(rel)
	var segs []string
	for _, s := range strings.Split(relSlash, "/") {
		if s == "" || s == "." {
			continue
		}
		if s == ".." {
			return ""
		}
		segs = append(segs, url.PathEscape(s))
	}
	if len(segs) == 0 {
		return ""
	}
	return talosPackageURLPrefix + url.PathEscape(dirName) + "/" + strings.Join(segs, "/")
}

// PermissionEntry is one persisted scope grant/deny row for host UI.
type PermissionEntry struct {
	AppID   string `json:"app_id"`
	Scope   string `json:"scope"`
	Granted bool   `json:"granted"`
}

// PermissionAuditEntry is one JSONL row in permission_audit.jsonl (export / diagnostics).
type PermissionAuditEntry struct {
	TS      string `json:"ts"`
	Action  string `json:"action"`
	AppID   string `json:"app_id"`
	Scope   string `json:"scope"`
	Granted bool   `json:"granted"`
	Message string `json:"message"`
}

type AppManifestView struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Icon           string   `json:"icon"`
	URL            string   `json:"url"`
	Description    string   `json:"description"`
	Category       string   `json:"category"`
	StoreURL       string   `json:"store_url,omitempty"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
	Development    bool     `json:"development,omitempty"`
	// TrustStatus: unknown | ok | tampered | unsigned | signed_ok | signed_invalid (see packageinstall.TrustStatus).
	TrustStatus string `json:"trust_status,omitempty"`
}

const launchpadPackageID = "app.launchpad"
const devServerLogName = "launchpad-dev"

func resolveRootDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	abs, err := filepath.Abs(wd)
	if err != nil {
		return wd
	}
	return abs
}

// NewApp creates a new App application struct
func NewApp() *App {
	rootDir := resolveRootDir()
	packagesDir := filepath.Join(rootDir, "Packages")
	hubServer := hub.NewServer(hub.DefaultSocketURL())

	return &App{
		rootDir:             rootDir,
		packagesDir:         packagesDir,
		processManager:      process.NewManager(),
		hub:                 hubServer,
		stateStore:          state.NewStore(),
		permissionsPath:     filepath.Join(rootDir, "Temp", "permissions.json"),
		logsDir:             filepath.Join(rootDir, "Temp", "logs"),
		hostLogPath:         filepath.Join(rootDir, "Temp", "logs", "talos.log"),
		devLogPath:          filepath.Join(rootDir, "Temp", "logs", "launchpad-dev.log"),
		packageLogDir:       filepath.Join(rootDir, "Temp", "logs", "packages"),
		permissionAuditPath: filepath.Join(rootDir, "Temp", "logs", "permission_audit.jsonl"),
		packages:            make(map[string]*packages.PackageInfo),
		devResolvedURL:      make(map[string]string),
		devReachCache:       make(map[string]devReachOutcome),
		sqlStore:            minisql.NewStore(rootDir),
		sidecarLoopbackPort: make(map[string]string),
	}
}

type devReachOutcome struct {
	usePackaged bool // true: last probe saw dev URL down; serve /talos-pkg/ instead
	until       time.Time
}

func devReachCacheKey(appID, endpoint string) string {
	return appID + "\x00" + strings.TrimSpace(endpoint)
}

func (a *App) clearDevReachCacheForApp(appID string) {
	a.devReachMu.Lock()
	defer a.devReachMu.Unlock()
	prefix := appID + "\x00"
	for k := range a.devReachCache {
		if strings.HasPrefix(k, prefix) {
			delete(a.devReachCache, k)
		}
	}
}

func (a *App) cachedDevReachOutcome(appID, endpoint string) (usePackaged bool, ok bool) {
	a.devReachMu.Lock()
	defer a.devReachMu.Unlock()
	key := devReachCacheKey(appID, endpoint)
	e, hit := a.devReachCache[key]
	if !hit || time.Now().After(e.until) {
		return false, false
	}
	return e.usePackaged, true
}

func (a *App) storeDevReachOutcome(appID, endpoint string, usePackaged bool) {
	a.devReachMu.Lock()
	defer a.devReachMu.Unlock()
	key := devReachCacheKey(appID, endpoint)
	ttl := 10 * time.Second
	if usePackaged {
		ttl = 4 * time.Second
	}
	a.devReachCache[key] = devReachOutcome{usePackaged: usePackaged, until: time.Now().Add(ttl)}
}

func shouldProbeDevHTTPEndpoint(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	host := strings.ToLower(strings.Trim(parsed.Hostname(), "[]"))
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasPrefix(host, "127.")
}

// httpDevEndpointReachable reports whether an http(s) dev server responds (HEAD, then GET).
func httpDevEndpointReachable(endpoint string) bool {
	u := strings.TrimSpace(endpoint)
	if u == "" {
		return false
	}
	if _, err := url.Parse(u); err != nil {
		return false
	}
	client := &http.Client{Timeout: 700 * time.Millisecond}
	try := func(method string) bool {
		req, err := http.NewRequest(method, u, nil)
		if err != nil {
			return false
		}
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		if method == http.MethodGet {
			_, _ = io.Copy(io.Discard, resp.Body)
		}
		return resp.StatusCode < 500
	}
	if try(http.MethodHead) {
		return true
	}
	return try(http.MethodGet)
}

func packagedWebEntryURL(pkg *packages.PackageInfo) string {
	if pkg == nil || pkg.Manifest == nil {
		return ""
	}
	entry := strings.TrimSpace(pkg.Manifest.WebEntry)
	if entry == "" {
		return ""
	}
	return packageAssetURL(pkg, entry)
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.ctx, a.cancel = context.WithCancel(ctx)
	a.initLogging()
	a.processManager.SetLogDir(a.packageLogDir)
	a.logInfo("startup", "Talos startup initiated")

	a.permissions = security.NewPermissions(func(appID, scope, reason string) (bool, string) {
		event := map[string]string{
			"app_id": appID,
			"scope":  scope,
			"reason": reason,
		}
		runtime.EventsEmit(a.ctx, "permissions:request", event)
		return false, security.MsgPendingHostApproval
	})
	a.loadPermissionGrants()
	a.grantLaunchpadAllPermissions()
	a.scopeManager = security.NewScopeManager(a.packagesDir, a.permissions)

	a.hub.SetStateHooks(
		func(appID string, data []byte) error {
			a.stateStore.Save(appID, data)
			return nil
		},
		func(appID string) ([]byte, bool, error) {
			data, ok := a.stateStore.Load(appID)
			return data, ok, nil
		},
	)
	a.hub.SetPermissionRequestHook(func(appID, scope, reason string) (bool, string, error) {
		granted, msg := a.permissions.Request(appID, scope, reason)
		if granted {
			_ = a.savePermissionGrants()
		}
		return granted, msg, nil
	})
	a.hub.SetResolvePathHook(func(appID, relativePath string) (string, bool, error) {
		resolved, err := a.ResolveScopedPath(appID, relativePath)
		if err != nil {
			return "", false, err
		}
		return resolved, true, nil
	})

	if err := a.hub.Start(); err != nil {
		runtime.LogErrorf(a.ctx, "hub start failed: %v", err)
		a.logError("hub", fmt.Sprintf("hub start failed: %v", err))
	} else {
		a.logInfo("hub", "hub started")
	}

	a.discovery = packages.NewDiscovery(a.packagesDir, func(evt packages.DiscoveryEvent) {
		a.handlePackageEvent(evt)
		runtime.EventsEmit(a.ctx, "packages:event", evt)
	})

	go func() {
		if err := a.discovery.Start(a.ctx); err != nil {
			runtime.LogErrorf(a.ctx, "package discovery stopped: %v", err)
			a.logError("packages", fmt.Sprintf("package discovery stopped: %v", err))
		}
	}()

	go a.ensureLaunchpadOrQuit()

	if p, err := a.LoadUserPrefs(); err == nil {
		a.devMu.Lock()
		a.prefsDeveloperMode = p.DeveloperMode
		a.devMu.Unlock()
	}
}

func (a *App) shutdown(_ context.Context) {
	a.logInfo("shutdown", "Talos shutdown requested")
	if a.cancel != nil {
		a.cancel()
	}
	a.processManager.StopAll()
	a.hub.Stop()
}

// ListPackages returns current package state keyed by package id.
func (a *App) ListPackages() map[string]*packages.PackageInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := make(map[string]*packages.PackageInfo, len(a.packages))
	for id, pkg := range a.packages {
		cp := *pkg
		out[id] = &cp
	}
	return out
}

// StartPackage starts a package process by id.
func (a *App) StartPackage(packageID string) error {
	a.mu.RLock()
	pkg := a.packages[packageID]
	a.mu.RUnlock()

	if pkg == nil {
		a.logError("package", fmt.Sprintf("start failed, package %s not found", packageID))
		return errors.New("package not found")
	}

	dataDir := filepath.Join(pkg.DirPath, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		a.logError("package", fmt.Sprintf("create data dir failed for %s: %v", packageID, err))
		return fmt.Errorf("create app data dir failed: %w", err)
	}

	env := map[string]string{
		"TALOS_APP_ID":       pkg.Manifest.ID,
		"TALOS_APP_DATA_DIR": dataDir,
		"TALOS_HUB_SOCKET":   a.hub.SocketURL(),
	}
	if a.sqlStore != nil {
		if dsn, _, err := a.sqlStore.Provision(packageID); err == nil {
			env["TALOS_SQL_DSN"] = dsn
		} else {
			a.logError("minisql", fmt.Sprintf("provision sql for %s: %v", packageID, err))
		}
	}
	if err := a.processManager.Start(a.ctx, pkg, env); err != nil {
		a.logError("package", fmt.Sprintf("start failed for %s: %v", packageID, err))
		return err
	}
	a.logInfo("package", fmt.Sprintf("started %s (log: %s)", packageID, filepath.Join(a.packageLogDir, packageID+".log")))
	go a.promptDeclaredNetworkPermissions(pkg)

	if a.effectiveDevelopmentEnabled() && pkg.Manifest.Development != nil && len(pkg.Manifest.Development.Command) > 0 {
		a.clearDevResolvedURL(packageID)
		manifestDev := strings.TrimSpace(pkg.Manifest.Development.URL)
		waitForInitialResolve := manifestDev == ""
		resolvedFirst := make(chan struct{}, 1)
		devOpts := &process.DevStartOptions{
			ManifestDevURL: manifestDev,
			OnResolvedURL: func(resolved string) {
				aligned := process.AlignDiscoveredDevURL(manifestDev, resolved)
				a.setDevResolvedURL(packageID, aligned)
				if waitForInitialResolve {
					select {
					case resolvedFirst <- struct{}{}:
					default:
					}
				}
				runtime.EventsEmit(a.ctx, "package:dev-url", map[string]string{
					"app_id": packageID,
					"url":    aligned,
				})
			},
		}
		if err := a.processManager.StartDev(a.ctx, pkg, pkg.Manifest.Development.Command, devOpts); err != nil {
			a.logError("package", fmt.Sprintf("dev server start failed for %s: %v", packageID, err))
		} else {
			a.logInfo("package", fmt.Sprintf("dev server started for %s", packageID))
			if waitForInitialResolve {
				select {
				case <-resolvedFirst:
				case <-time.After(3 * time.Second):
				}
			}
		}
	}
	// Next manifest read (e.g. GetInstalledApps after launch) should re-probe dev loopback, not reuse a stale packaged fallback from before Vite was ready.
	a.clearDevReachCacheForApp(packageID)
	return nil
}

func (a *App) promptDeclaredNetworkPermissions(pkg *packages.PackageInfo) {
	if pkg == nil || pkg.Manifest == nil || a.permissions == nil || a.ctx == nil {
		return
	}
	appID := pkg.Manifest.ID
	for _, scope := range pkg.Manifest.Permissions {
		scope = strings.TrimSpace(scope)
		if !strings.HasPrefix(scope, "net:") {
			continue
		}
		if a.permissions.HasDecision(appID, scope) {
			continue
		}
		_, _, _ = a.RequestPermission(appID, scope, "Requested by manifest at app launch")
	}
}

// StopPackage stops a running package process by id.
func (a *App) StopPackage(packageID string) error {
	a.clearDevResolvedURL(packageID)
	_ = a.processManager.StopDev(packageID)
	err := a.processManager.Stop(packageID)
	if err != nil {
		a.logError("package", fmt.Sprintf("stop failed for %s: %v", packageID, err))
		return err
	}
	a.clearCachedSidecarPort(packageID)
	if remErr := a.removeAPIPortFileForApp(packageID); remErr != nil {
		a.logError("package", fmt.Sprintf("remove api-port for %s: %v", packageID, remErr))
	}
	a.logInfo("package", fmt.Sprintf("stopped %s", packageID))
	return nil
}

// RunningPackageIDs returns currently running package ids.
func (a *App) RunningPackageIDs() []string {
	return a.processManager.RunningIDs()
}

// HubSocketURL exposes the central hub transport endpoint.
func (a *App) HubSocketURL() string {
	return a.hub.SocketURL()
}

// GrantPermission grants app scope permission from host UI flow.
func (a *App) GrantPermission(appID, scope string) {
	a.permissions.Set(appID, scope, true)
	a.permissions.CompletePendingDecision(appID, scope, true, "granted by user")
	_ = a.savePermissionGrants()
	_ = a.appendPermissionAudit("grant", appID, scope, true, "granted by user")
}

// DenyPermission stores an explicit deny decision.
func (a *App) DenyPermission(appID, scope string) {
	a.permissions.Set(appID, scope, false)
	a.permissions.CompletePendingDecision(appID, scope, false, "denied by user")
	_ = a.savePermissionGrants()
	_ = a.appendPermissionAudit("deny", appID, scope, false, "denied by user")
}

// RevokePermission clears a scope so the next SDK request can prompt again.
func (a *App) RevokePermission(appID, scope string) {
	if appID == launchpadPackageID {
		return
	}
	a.permissions.Clear(appID, scope)
	a.permissions.CompletePendingDecision(appID, scope, false, "revoked")
	_ = a.savePermissionGrants()
	_ = a.appendPermissionAudit("revoke", appID, scope, false, "revoked")
}

// ListPermissionEntries returns persisted permission rows for settings UI.
func (a *App) ListPermissionEntries() []PermissionEntry {
	exp := a.permissions.Export()
	out := make([]PermissionEntry, 0, 32)
	for appID, scopes := range exp {
		if appID == launchpadPackageID {
			continue
		}
		for scope, granted := range scopes {
			if scope == security.ScopeFSData {
				continue
			}
			out = append(out, PermissionEntry{AppID: appID, Scope: scope, Granted: granted})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].AppID != out[j].AppID {
			return out[i].AppID < out[j].AppID
		}
		return out[i].Scope < out[j].Scope
	})
	return out
}

// ListPermissionAudit returns the last N entries from permission_audit.jsonl (newest last).
func (a *App) ListPermissionAudit(maxEntries int) []PermissionAuditEntry {
	if maxEntries <= 0 {
		maxEntries = 200
	}
	if maxEntries > 5000 {
		maxEntries = 5000
	}
	raw, err := os.ReadFile(a.permissionAuditPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	if len(lines) > maxEntries {
		lines = lines[len(lines)-maxEntries:]
	}
	out := make([]PermissionAuditEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e PermissionAuditEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (a *App) appendPermissionAudit(action, appID, scope string, granted bool, message string) error {
	a.auditMu.Lock()
	defer a.auditMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(a.permissionAuditPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(a.permissionAuditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	e := PermissionAuditEntry{
		TS:      time.Now().Format(time.RFC3339Nano),
		Action:  action,
		AppID:   appID,
		Scope:   scope,
		Granted: granted,
		Message: message,
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

// SaveAppState stores app state in host memory.
func (a *App) SaveAppState(appID string, data []byte) {
	a.stateStore.Save(appID, data)
}

// LoadAppState loads app state from host memory.
func (a *App) LoadAppState(appID string) []byte {
	data, _ := a.stateStore.Load(appID)
	return data
}

// ResolveScopedPath enforces fs:data default scoping.
func (a *App) ResolveScopedPath(appID, relativePath string) (string, error) {
	a.mu.RLock()
	pkg := a.packages[appID]
	a.mu.RUnlock()
	if pkg == nil {
		return "", errors.New("package not found")
	}
	return a.scopeManager.ResolvePath(pkg.DirName, appID, relativePath)
}

// WriteScopedText writes UTF-8 text to an app-scoped file path validated by host policy.
func (a *App) WriteScopedText(appID, relativePath, text string) error {
	resolved, err := a.ResolveScopedPath(appID, relativePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o700); err != nil {
		return err
	}
	return os.WriteFile(resolved, []byte(text), 0o600)
}

// ReadScopedText reads UTF-8 text from an app-scoped file path validated by host policy.
func (a *App) ReadScopedText(appID, relativePath string) (map[string]any, error) {
	resolved, err := a.ResolveScopedPath(appID, relativePath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{"found": false, "text": ""}, nil
		}
		return nil, err
	}
	return map[string]any{"found": true, "text": string(data)}, nil
}

// IframePostToHost receives iframe messages and forwards them to host listeners.
func (a *App) IframePostToHost(appID, event string, payload string) {
	envelope := map[string]string{
		"app_id":  appID,
		"event":   event,
		"payload": payload,
	}
	runtime.EventsEmit(a.ctx, "iframe:from", envelope)
}

// HostPostToIframe emits a message for a specific app iframe.
func (a *App) HostPostToIframe(appID, event string, payload string) {
	envelope := map[string]string{
		"app_id":  appID,
		"event":   event,
		"payload": payload,
	}
	runtime.EventsEmit(a.ctx, "iframe:to", envelope)
}

// RouteMessage routes a string payload through the hub.
func (a *App) RouteMessage(sourceAppID, targetAppID, msgType, payload string) (string, error) {
	resp, err := a.hub.Route(a.ctx, &hubpb.RouteRequest{
		Message: &hubpb.Message{
			SourceAppId: sourceAppID,
			TargetAppId: targetAppID,
			Type:        msgType,
			Payload:     []byte(payload),
		},
	})
	if err != nil {
		return "", err
	}
	if resp.GetError() != "" {
		return "", errors.New(resp.GetError())
	}
	return string(resp.GetMessage().GetPayload()), nil
}

// BroadcastMessage sends a hub broadcast message.
func (a *App) BroadcastMessage(sourceAppID, msgType, payload string) (int32, error) {
	resp, err := a.hub.Broadcast(a.ctx, &hubpb.BroadcastRequest{
		Message: &hubpb.Message{
			SourceAppId: sourceAppID,
			Type:        msgType,
			Payload:     []byte(payload),
		},
	})
	if err != nil {
		return 0, err
	}
	return resp.GetRecipientCount(), nil
}

// RequestPermission asks host policy for an app scope permission.
func (a *App) RequestPermission(appID, scope, reason string) (bool, string, error) {
	granted, msg := a.permissions.Request(appID, scope, reason)
	if granted {
		_ = a.savePermissionGrants()
	}
	_ = a.appendPermissionAudit("request", appID, scope, granted, msg)
	return granted, msg, nil
}

// RequestPermissionDecision is a JS-friendly wrapper for iframe SDK transport.
func (a *App) RequestPermissionDecision(appID, scope, reason string) map[string]any {
	granted, message, err := a.RequestPermission(appID, scope, reason)
	out := map[string]any{
		"granted": granted,
		"message": message,
	}
	if err != nil {
		out["error"] = err.Error()
	}
	return out
}

// SaveAppStateText stores UTF-8 state payload for iframe SDK usage.
func (a *App) SaveAppStateText(appID, data string) {
	a.stateStore.Save(appID, []byte(data))
}

// LoadAppStateText loads state payload and returns UTF-8 string.
func (a *App) LoadAppStateText(appID string) string {
	return string(a.LoadAppState(appID))
}

// SaveAppStateBase64 stores binary payload encoded as base64.
func (a *App) SaveAppStateBase64(appID, b64 string) error {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	a.stateStore.Save(appID, raw)
	return nil
}

// LoadAppStateBase64 loads binary state and encodes it as base64.
func (a *App) LoadAppStateBase64(appID string) string {
	raw := a.LoadAppState(appID)
	if len(raw) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func (a *App) handlePackageEvent(evt packages.DiscoveryEvent) {
	if evt.PackageID == "" {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	switch evt.Type {
	case packages.EventAdded, packages.EventUpdated:
		if evt.Package != nil {
			a.packages[evt.PackageID] = evt.Package
			a.logInfo("packages", fmt.Sprintf("%s %s", evt.Type, evt.PackageID))
			a.hub.RegisterHandler(evt.PackageID, func(_ context.Context, msg *hubpb.Message) (*hubpb.Message, error) {
				// Placeholder local route target until apps register their own runtime handlers.
				return &hubpb.Message{
					SourceAppId: evt.PackageID,
					TargetAppId: msg.SourceAppId,
					Type:        "ack",
					Payload:     []byte("routed by host"),
					RequestId:   msg.RequestId,
				}, nil
			})
		}
	case packages.EventRemoved:
		delete(a.packages, evt.PackageID)
		_ = a.processManager.StopDev(evt.PackageID)
		_ = a.processManager.Stop(evt.PackageID)
		if a.sqlStore != nil {
			_ = a.sqlStore.Revoke(evt.PackageID)
		}
		a.clearDevResolvedURL(evt.PackageID)
		a.hub.UnregisterHandler(evt.PackageID)
		a.logInfo("packages", fmt.Sprintf("removed %s", evt.PackageID))
	}
}

func (a *App) loadPermissionGrants() {
	grants, err := security.LoadGrants(a.permissionsPath)
	if err != nil {
		runtime.LogErrorf(a.ctx, "load permission grants failed: %v", err)
		a.logError("permissions", fmt.Sprintf("load grants failed: %v", err))
		return
	}
	a.permissions.Import(grants)
	a.logInfo("permissions", "permission grants loaded")
}

func (a *App) grantLaunchpadAllPermissions() {
	if a.permissions == nil {
		return
	}
	a.permissions.Set(launchpadPackageID, "*", true)
	if err := a.savePermissionGrants(); err != nil {
		a.logError("permissions", fmt.Sprintf("failed to persist launchpad wildcard grant: %v", err))
		return
	}
	a.logInfo("permissions", "launchpad wildcard permissions granted")
}

func (a *App) savePermissionGrants() error {
	return security.SaveGrants(a.permissionsPath, a.permissions.Export())
}

func (a *App) initLogging() {
	if err := os.MkdirAll(a.packageLogDir, 0o755); err != nil {
		runtime.LogErrorf(a.ctx, "create logs directory failed: %v", err)
		return
	}
	a.logInfo("logging", fmt.Sprintf("logs initialized under %s", a.logsDir))
}

func (a *App) logInfo(source, message string) {
	a.writeLogLine("INFO", source, message)
}

func (a *App) logError(source, message string) {
	a.writeLogLine("ERROR", source, message)
}

func (a *App) writeLogLine(level, source, message string) {
	if a.ctx == nil {
		return
	}
	a.logMu.Lock()
	defer a.logMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(a.hostLogPath), 0o755); err != nil {
		runtime.LogErrorf(a.ctx, "create host log dir failed: %v", err)
		return
	}
	f, err := os.OpenFile(a.hostLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		runtime.LogErrorf(a.ctx, "open host log failed: %v", err)
		return
	}
	defer f.Close()

	line := fmt.Sprintf("%s [%s] [%s] %s", time.Now().Format(time.RFC3339), level, source, message)
	_, _ = fmt.Fprintln(f, line)
	runtime.EventsEmit(a.ctx, "logs:append", map[string]string{
		"source": source,
		"level":  level,
		"line":   line,
	})
}

func (a *App) ensureLaunchpadOrQuit() {
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.RLock()
		pkg := a.packages[launchpadPackageID]
		a.mu.RUnlock()
		if pkg != nil && pkg.Manifest != nil {
			a.logInfo("startup", "required launchpad package detected")
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	a.logError("startup", "required package app.launchpad is missing or invalid; quitting")
	runtime.EventsEmit(a.ctx, "fatal:error", map[string]string{
		"message": "Required package app.launchpad is missing or invalid.",
	})
	runtime.Quit(a.ctx)
}

// GetLogCatalog exposes runtime and package log sources.
func (a *App) GetLogCatalog() map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := map[string]string{
		"host": a.hostLogPath,
	}
	if _, err := os.Stat(a.devLogPath); err == nil {
		out[devServerLogName] = a.devLogPath
	}
	for id := range a.packages {
		out["package:"+id] = filepath.Join(a.packageLogDir, id+".log")
	}
	return out
}

// ReadLogTail returns up to maxLines from the selected log source.
func (a *App) ReadLogTail(source string, maxLines int) (string, error) {
	if maxLines <= 0 {
		maxLines = 120
	}
	if maxLines > 1000 {
		maxLines = 1000
	}

	path, err := a.resolveLogSourcePath(source)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	lines := make([]string, 0, maxLines)
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLines {
			lines = lines[1:]
		}
	}
	return strings.Join(lines, "\n"), nil
}

func (a *App) resolveLogSourcePath(source string) (string, error) {
	switch source {
	case "host":
		return a.hostLogPath, nil
	case devServerLogName:
		return a.devLogPath, nil
	}
	if strings.HasPrefix(source, "package:") {
		id := strings.TrimPrefix(source, "package:")
		if strings.TrimSpace(id) == "" {
			return "", errors.New("invalid package log source")
		}
		return filepath.Join(a.packageLogDir, id+".log"), nil
	}
	return "", fmt.Errorf("unknown log source %q", source)
}

// GetRuntimeInfo returns runtime flags useful for frontend dev tools.
func (a *App) GetRuntimeInfo() map[string]any {
	return map[string]any{
		"dev_mode":                     buildmode.DevelopmentAllowed(),
		"developer_mode":               a.GetDeveloperMode(),
		"effective_development":        a.effectiveDevelopmentEnabled(),
		"development_features_enabled": a.DevelopmentFeaturesEnabled(),
	}
}

// GetThemes returns available CSS themes from frontend public folder.
func (a *App) GetThemes() ([]ThemeInfo, error) {
	themesDir := filepath.Join(a.packagesDir, "Launchpad", "dist", "themes")
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return nil, err
	}
	out := make([]ThemeInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".css") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".css")
		out = append(out, ThemeInfo{Name: name, File: entry.Name()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// SaveUserPrefs persists user UI preferences.
func (a *App) SaveUserPrefs(prefs UserPrefs) error {
	path := filepath.Join(a.rootDir, "Temp", "user_prefs.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return err
	}
	a.devMu.Lock()
	a.prefsDeveloperMode = prefs.DeveloperMode
	a.devMu.Unlock()
	return nil
}

// LoadUserPrefs loads persisted UI preferences if available.
func (a *App) LoadUserPrefs() (UserPrefs, error) {
	path := filepath.Join(a.rootDir, "Temp", "user_prefs.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return UserPrefs{Theme: "minecraft", TabColors: map[string]string{}, DeveloperMode: false}, nil
		}
		return UserPrefs{}, err
	}

	var prefs UserPrefs
	if err := json.Unmarshal(raw, &prefs); err != nil {
		return UserPrefs{}, err
	}
	if prefs.TabColors == nil {
		prefs.TabColors = map[string]string{}
	}
	if strings.TrimSpace(prefs.Theme) == "" {
		prefs.Theme = "minecraft"
	}
	return prefs, nil
}

// GetStartupLaunchpad returns the required launchpad package for host UI startup.
func (a *App) GetStartupLaunchpad() (AppManifestView, error) {
	a.mu.RLock()
	pkg := a.packages[launchpadPackageID]
	a.mu.RUnlock()

	if pkg == nil || pkg.Manifest == nil {
		return AppManifestView{}, fmt.Errorf("required package %q not found", launchpadPackageID)
	}
	if strings.TrimSpace(pkg.Manifest.WebEntry) == "" {
		return AppManifestView{}, fmt.Errorf("required package %q has no web_entry", launchpadPackageID)
	}
	return a.manifestViewForPackage(pkg), nil
}

// GetInstalledApps returns app manifests suitable for launchpad installed grid.
func (a *App) GetInstalledApps() []AppManifestView {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := make([]AppManifestView, 0, len(a.packages))
	for _, pkg := range a.packages {
		if pkg.Manifest == nil {
			continue
		}
		out = append(out, a.manifestViewForPackage(pkg))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (a *App) setDevResolvedURL(appID, base string) {
	a.devResolvedMu.Lock()
	a.devResolvedURL[appID] = strings.TrimSpace(base)
	a.devResolvedMu.Unlock()
	a.clearDevReachCacheForApp(appID)
}

func (a *App) clearDevResolvedURL(appID string) {
	a.devResolvedMu.Lock()
	delete(a.devResolvedURL, appID)
	a.devResolvedMu.Unlock()
	a.clearDevReachCacheForApp(appID)
}

// effectiveDevelopmentEnabled is true when manifest development.* should be honored:
// machine env TALOS_DEV_MODE=1 or user enabled Developer mode in Settings.
func (a *App) effectiveDevelopmentEnabled() bool {
	if buildmode.DevelopmentAllowed() {
		return true
	}
	a.devMu.RLock()
	defer a.devMu.RUnlock()
	return a.prefsDeveloperMode
}

// SetDeveloperMode persists the Settings toggle and stops dev servers when disabling.
func (a *App) SetDeveloperMode(enabled bool) error {
	prefs, err := a.LoadUserPrefs()
	if err != nil {
		prefs = UserPrefs{Theme: "minecraft", TabColors: map[string]string{}}
	}
	prefs.DeveloperMode = enabled
	if err := a.SaveUserPrefs(prefs); err != nil {
		return err
	}
	if !enabled {
		a.processManager.StopAllDev()
		a.devResolvedMu.Lock()
		a.devResolvedURL = make(map[string]string)
		a.devResolvedMu.Unlock()
	}
	return nil
}

// GetDeveloperMode returns the persisted Settings flag (not including TALOS_DEV_MODE alone).
func (a *App) GetDeveloperMode() bool {
	a.devMu.RLock()
	defer a.devMu.RUnlock()
	return a.prefsDeveloperMode
}

// manifestDevIFrameURL returns iframe URL and bridge origins, applying a discovered dev-server URL when present.
func (a *App) manifestDevIFrameURL(pkg *packages.PackageInfo) (u string, origins []string, dev bool) {
	u, origins, dev = resolvePackageURL(pkg, a.effectiveDevelopmentEnabled())
	if !dev || pkg == nil || pkg.Manifest == nil {
		return u, origins, dev
	}
	final := u
	a.devResolvedMu.RLock()
	override := a.devResolvedURL[pkg.Manifest.ID]
	a.devResolvedMu.RUnlock()
	if override != "" {
		final = override
		parsed, err := url.Parse(override)
		if err == nil {
			extra := parsed.Scheme + "://" + parsed.Host
			origins = mergeOriginsUnique(origins, extra)
		}
	}
	// Command-only dev uses about:blank until log discovery fills devResolvedURL; then final is the live URL.
	if strings.TrimSpace(final) == "" || final == "about:blank" {
		return "about:blank", process.ExpandLoopbackOrigins(origins), dev
	}
	if !shouldProbeDevHTTPEndpoint(final) {
		return final, process.ExpandLoopbackOrigins(origins), dev
	}
	packaged := packagedWebEntryURL(pkg)
	if packaged == "" {
		return final, process.ExpandLoopbackOrigins(origins), dev
	}
	appID := pkg.Manifest.ID
	usePackaged, ok := a.cachedDevReachOutcome(appID, final)
	if !ok {
		usePackaged = !httpDevEndpointReachable(final)
		a.storeDevReachOutcome(appID, final, usePackaged)
		if usePackaged && a.ctx != nil {
			a.logInfo("package", fmt.Sprintf("%s: dev URL %q unreachable, using packaged web assets", appID, final))
		}
	}
	if usePackaged {
		return packaged, nil, false
	}
	return final, process.ExpandLoopbackOrigins(origins), dev
}

func mergeOriginsUnique(origins []string, extra string) []string {
	if strings.TrimSpace(extra) == "" {
		return origins
	}
	for _, o := range origins {
		if o == extra {
			return origins
		}
	}
	out := make([]string, 0, len(origins)+1)
	out = append(out, origins...)
	return append(out, extra)
}

func (a *App) manifestViewForPackage(pkg *packages.PackageInfo) AppManifestView {
	if pkg == nil || pkg.Manifest == nil {
		return AppManifestView{}
	}
	url, origins, dev := a.manifestDevIFrameURL(pkg)
	icon := resolveManifestIcon(pkg)
	hashPath := filepath.Join(a.hashDir(), pkg.Manifest.ID+".json")
	ts, err := packageinstall.EvaluateTrust(pkg.DirPath, hashPath, a.trustedKeysDir())
	trust := string(ts)
	if err != nil {
		trust = string(packageinstall.TrustUnknown)
	}
	desc := strings.TrimSpace(pkg.Manifest.Description)
	if desc == "" {
		desc = "Installed Tiny App"
	}
	storeURL := strings.TrimSpace(pkg.Manifest.StoreURL)
	return AppManifestView{
		ID:             pkg.Manifest.ID,
		Name:           pkg.Manifest.Name,
		Icon:           icon,
		URL:            url,
		Description:    desc,
		StoreURL:       storeURL,
		Category:       "installed",
		AllowedOrigins: origins,
		Development:    dev,
		TrustStatus:    trust,
	}
}

func (a *App) trustedKeysDir() string {
	return filepath.Join(a.rootDir, "Temp", "trusted_keys")
}

// packageToManifestView is used by tests that construct a synthetic App without trust evaluation.
func packageToManifestView(pkg *packages.PackageInfo) AppManifestView {
	if pkg == nil || pkg.Manifest == nil {
		return AppManifestView{}
	}
	url, origins, dev := resolvePackageURL(pkg, false)
	icon := resolveManifestIcon(pkg)
	desc := strings.TrimSpace(pkg.Manifest.Description)
	if desc == "" {
		desc = "Installed Tiny App"
	}
	storeURL := strings.TrimSpace(pkg.Manifest.StoreURL)
	return AppManifestView{
		ID:             pkg.Manifest.ID,
		Name:           pkg.Manifest.Name,
		Icon:           icon,
		URL:            url,
		Description:    desc,
		StoreURL:       storeURL,
		Category:       "installed",
		AllowedOrigins: origins,
		Development:    dev,
	}
}

// resolvePackageURL returns iframe URL, optional postMessage allowlist, and whether dev URL is active.
func resolvePackageURL(pkg *packages.PackageInfo, developmentFeaturesEnabled bool) (u string, allowed []string, devMode bool) {
	if pkg == nil || pkg.Manifest == nil {
		return "", nil, false
	}
	def := pkg.Manifest
	fileU := ""
	if strings.TrimSpace(def.WebEntry) != "" {
		webPath := filepath.Join(pkg.DirPath, def.WebEntry)
		if abs, err := filepath.Abs(webPath); err == nil {
			webPath = abs
		}
		fileU = packageAssetURL(pkg, def.WebEntry)
	}
	if !developmentFeaturesEnabled || def.Development == nil {
		return fileU, nil, false
	}
	d := def.Development
	devURL := strings.TrimSpace(d.URL)
	if len(d.Command) > 0 && devURL == "" {
		origins := make([]string, len(d.AllowedOrigins))
		copy(origins, d.AllowedOrigins)
		// With command-only dev config, wait for runtime URL discovery before loading iframe content.
		return "about:blank", origins, true
	}
	if devURL == "" {
		return fileU, nil, false
	}
	origins := make([]string, len(d.AllowedOrigins))
	copy(origins, d.AllowedOrigins)
	return devURL, origins, true
}

// fileURLToLocalPath converts a file:// URL to an OS path (including file:///… on Unix and file:///C:/… on Windows).
func fileURLToLocalPath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "file") {
		return ""
	}
	pathPart := u.Path
	if pathPart == "" {
		return ""
	}
	pathPart, err = url.PathUnescape(pathPart)
	if err != nil || pathPart == "" {
		return ""
	}
	if goruntime.GOOS == "windows" && len(pathPart) >= 3 && pathPart[0] == '/' && pathPart[2] == ':' {
		pathPart = pathPart[1:]
	}
	return filepath.Clean(pathPart)
}

func iconBytesToDataURL(bytes []byte, pathHint string) string {
	ext := strings.ToLower(filepath.Ext(pathHint))
	contentType := mime.TypeByExtension(ext)
	if strings.TrimSpace(contentType) == "" {
		contentType = http.DetectContentType(bytes)
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(bytes)
}

// pathWithinPackageDir reports whether target is equal to root or a path inside root (no "..").
func pathWithinPackageDir(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// manifestRelativeIconPath resolves an icon path relative to the package directory (where manifest.yaml lives).
// Leading "./", "/", or "\" are ignored so "dist/x.png", "/dist/x.png", and "./dist/x.png" all resolve the same.
func manifestRelativeIconPath(pkgDir, raw string) (string, bool) {
	pkgDir = filepath.Clean(pkgDir)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	rel := raw
	rel = strings.TrimPrefix(rel, "./")
	rel = strings.TrimLeft(rel, "/\\")
	rel = strings.TrimPrefix(rel, "./")
	if rel == "" {
		return "", false
	}
	joined := filepath.Join(pkgDir, filepath.FromSlash(filepath.ToSlash(rel)))
	joined = filepath.Clean(joined)
	if !pathWithinPackageDir(pkgDir, joined) {
		return "", false
	}
	return joined, true
}

func resolveManifestIcon(pkg *packages.PackageInfo) string {
	if pkg == nil || pkg.Manifest == nil {
		return ""
	}
	rawIcon := strings.TrimSpace(pkg.Manifest.Icon)
	if rawIcon == "" {
		return ""
	}
	if strings.HasPrefix(rawIcon, "data:") || strings.HasPrefix(rawIcon, "http://") || strings.HasPrefix(rawIcon, "https://") {
		return rawIcon
	}
	root := filepath.Clean(pkg.DirPath)

	if strings.HasPrefix(strings.ToLower(rawIcon), "file://") {
		p := fileURLToLocalPath(rawIcon)
		if p == "" {
			return rawIcon
		}
		p = filepath.Clean(p)
		// Only embed file:// icons that live under this package directory (same rule as relative paths).
		if !pathWithinPackageDir(root, p) {
			return rawIcon
		}
		bytes, err := os.ReadFile(p)
		if err != nil {
			return rawIcon
		}
		return iconBytesToDataURL(bytes, p)
	}

	iconPath, ok := manifestRelativeIconPath(root, rawIcon)
	if !ok {
		return rawIcon
	}
	bytes, err := os.ReadFile(iconPath)
	if err != nil {
		return rawIcon
	}
	return iconBytesToDataURL(bytes, iconPath)
}

// GetStoreApps returns placeholder store metadata.
func (a *App) GetStoreApps() []AppManifestView {
	return []AppManifestView{
		{ID: "store.obsidian", Name: "Obsidian", Icon: "📝", Description: "Knowledge base for markdown notes", Category: "store", StoreURL: "https://obsidian.md"},
		{ID: "store.figma", Name: "Figma", Icon: "🎨", Description: "Collaborative interface design", Category: "store", StoreURL: "https://figma.com"},
		{ID: "store.telegram", Name: "Telegram", Icon: "💬", Description: "Secure messaging and channels", Category: "store", StoreURL: "https://telegram.org"},
	}
}
