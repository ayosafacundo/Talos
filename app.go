package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	hubpb "Talos/api/proto/talos/hub/v1"
	"Talos/internal/buildmode"
	"Talos/internal/hub"
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
	logMu    sync.Mutex
	auditMu  sync.Mutex

	permissionAuditPath string
}

type ThemeInfo struct {
	Name string `json:"name"`
	File string `json:"file"`
}

type UserPrefs struct {
	Theme     string            `json:"theme"`
	TabColors map[string]string `json:"tab_colors"`
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
}

const launchpadPackageID = "app.launchpad"
const devServerLogName = "launchpad-dev"

// NewApp creates a new App application struct
func NewApp() *App {
	rootDir := "."
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
	}
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
	if err := a.processManager.Start(a.ctx, pkg, env); err != nil {
		a.logError("package", fmt.Sprintf("start failed for %s: %v", packageID, err))
		return err
	}
	a.logInfo("package", fmt.Sprintf("started %s (log: %s)", packageID, filepath.Join(a.packageLogDir, packageID+".log")))

	if buildmode.DevelopmentAllowed() && pkg.Manifest.Development != nil && len(pkg.Manifest.Development.Command) > 0 {
		if err := a.processManager.StartDev(a.ctx, pkg, pkg.Manifest.Development.Command); err != nil {
			a.logError("package", fmt.Sprintf("dev server start failed for %s: %v", packageID, err))
		} else {
			a.logInfo("package", fmt.Sprintf("dev server started for %s", packageID))
		}
	}
	return nil
}

// StopPackage stops a running package process by id.
func (a *App) StopPackage(packageID string) error {
	_ = a.processManager.StopDev(packageID)
	err := a.processManager.Stop(packageID)
	if err != nil {
		a.logError("package", fmt.Sprintf("stop failed for %s: %v", packageID, err))
		return err
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
		_ = a.processManager.Stop(evt.PackageID)
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
		"dev_mode": strings.TrimSpace(os.Getenv("TALOS_DEV_MODE")) == "1",
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
	return os.WriteFile(path, raw, 0o600)
}

// LoadUserPrefs loads persisted UI preferences if available.
func (a *App) LoadUserPrefs() (UserPrefs, error) {
	path := filepath.Join(a.rootDir, "Temp", "user_prefs.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return UserPrefs{Theme: "minecraft", TabColors: map[string]string{}}, nil
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
	return packageToManifestView(pkg), nil
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
		out = append(out, packageToManifestView(pkg))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func packageToManifestView(pkg *packages.PackageInfo) AppManifestView {
	url, origins, dev := resolvePackageURL(pkg)
	icon := resolveManifestIcon(pkg)
	return AppManifestView{
		ID:             pkg.Manifest.ID,
		Name:           pkg.Manifest.Name,
		Icon:           icon,
		URL:            url,
		Description:    "Installed Tiny App",
		Category:       "installed",
		AllowedOrigins: origins,
		Development:    dev,
	}
}

// resolvePackageURL returns iframe URL, optional postMessage allowlist, and whether dev URL is active.
func resolvePackageURL(pkg *packages.PackageInfo) (u string, allowed []string, devMode bool) {
	if pkg == nil || pkg.Manifest == nil {
		return "", nil, false
	}
	def := pkg.Manifest
	fileU := ""
	if strings.TrimSpace(def.WebEntry) != "" {
		fileU = "file://" + filepath.Join(pkg.DirPath, def.WebEntry)
	}
	if !buildmode.DevelopmentAllowed() || def.Development == nil {
		return fileU, nil, false
	}
	d := def.Development
	devURL := strings.TrimSpace(d.URL)
	if devURL == "" {
		return fileU, nil, false
	}
	origins := make([]string, len(d.AllowedOrigins))
	copy(origins, d.AllowedOrigins)
	return devURL, origins, true
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
	relativeIcon := strings.TrimLeft(rawIcon, "/\\")
	iconPath := filepath.Clean(filepath.Join(pkg.DirPath, relativeIcon))
	packageRoot := filepath.Clean(pkg.DirPath)
	if !strings.HasPrefix(iconPath, packageRoot+string(filepath.Separator)) && iconPath != packageRoot {
		return rawIcon
	}
	bytes, err := os.ReadFile(iconPath)
	if err != nil {
		return rawIcon
	}
	ext := strings.ToLower(filepath.Ext(iconPath))
	contentType := mime.TypeByExtension(ext)
	if strings.TrimSpace(contentType) == "" {
		contentType = http.DetectContentType(bytes)
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(bytes)
}

// GetStoreApps returns placeholder store metadata.
func (a *App) GetStoreApps() []AppManifestView {
	return []AppManifestView{
		{ID: "store.obsidian", Name: "Obsidian", Icon: "📝", Description: "Knowledge base for markdown notes", Category: "store", StoreURL: "https://obsidian.md"},
		{ID: "store.figma", Name: "Figma", Icon: "🎨", Description: "Collaborative interface design", Category: "store", StoreURL: "https://figma.com"},
		{ID: "store.telegram", Name: "Telegram", Icon: "💬", Description: "Secure messaging and channels", Category: "store", StoreURL: "https://telegram.org"},
	}
}
