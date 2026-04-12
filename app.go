package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	hubpb "Talos/api/proto/talos/hub/v1"
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

	mu       sync.RWMutex
	packages map[string]*packages.PackageInfo
}

// NewApp creates a new App application struct
func NewApp() *App {
	rootDir := "."
	packagesDir := filepath.Join(rootDir, "Packages")
	hubServer := hub.NewServer(hub.DefaultSocketURL())

	return &App{
		rootDir:         rootDir,
		packagesDir:     packagesDir,
		processManager:  process.NewManager(),
		hub:             hubServer,
		stateStore:      state.NewStore(),
		permissionsPath: filepath.Join(rootDir, "Temp", "permissions.json"),
		packages:        make(map[string]*packages.PackageInfo),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.ctx, a.cancel = context.WithCancel(ctx)

	a.permissions = security.NewPermissions(func(appID, scope, reason string) (bool, string) {
		event := map[string]string{
			"app_id": appID,
			"scope":  scope,
			"reason": reason,
		}
		runtime.EventsEmit(a.ctx, "permissions:request", event)
		return false, "pending host approval"
	})
	a.loadPermissionGrants()
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
	}

	a.discovery = packages.NewDiscovery(a.packagesDir, func(evt packages.DiscoveryEvent) {
		a.handlePackageEvent(evt)
		runtime.EventsEmit(a.ctx, "packages:event", evt)
	})

	go func() {
		if err := a.discovery.Start(a.ctx); err != nil {
			runtime.LogErrorf(a.ctx, "package discovery stopped: %v", err)
		}
	}()
}

func (a *App) shutdown(_ context.Context) {
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
		return errors.New("package not found")
	}

	dataDir := filepath.Join(pkg.DirPath, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create app data dir failed: %w", err)
	}

	env := map[string]string{
		"TALOS_APP_ID":       pkg.Manifest.ID,
		"TALOS_APP_DATA_DIR": dataDir,
		"TALOS_HUB_SOCKET":   a.hub.SocketURL(),
	}
	if err := a.processManager.Start(a.ctx, pkg, env); err != nil {
		return err
	}
	return nil
}

// StopPackage stops a running package process by id.
func (a *App) StopPackage(packageID string) error {
	return a.processManager.Stop(packageID)
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
	_ = a.savePermissionGrants()
}

// DenyPermission stores an explicit deny decision.
func (a *App) DenyPermission(appID, scope string) {
	a.permissions.Set(appID, scope, false)
	_ = a.savePermissionGrants()
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
	}
}

func (a *App) loadPermissionGrants() {
	grants, err := security.LoadGrants(a.permissionsPath)
	if err != nil {
		runtime.LogErrorf(a.ctx, "load permission grants failed: %v", err)
		return
	}
	a.permissions.Import(grants)
}

func (a *App) savePermissionGrants() error {
	return security.SaveGrants(a.permissionsPath, a.permissions.Export())
}
