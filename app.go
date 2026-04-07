package main

import (
	"context"
	"errors"
	"path/filepath"
	"sync"

	"Talos/internal/hub"
	"Talos/internal/packages"
	"Talos/internal/process"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx    context.Context
	cancel context.CancelFunc

	rootDir     string
	packagesDir string

	discovery      *packages.Discovery
	processManager *process.Manager
	hub            *hub.Server

	mu       sync.RWMutex
	packages map[string]*packages.PackageInfo
}

// NewApp creates a new App application struct
func NewApp() *App {
	rootDir := "."
	packagesDir := filepath.Join(rootDir, "Packages")
	hubServer := hub.NewServer(hub.DefaultSocketURL())

	return &App{
		rootDir:        rootDir,
		packagesDir:    packagesDir,
		processManager: process.NewManager(),
		hub:            hubServer,
		packages:       make(map[string]*packages.PackageInfo),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.ctx, a.cancel = context.WithCancel(ctx)

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

	if err := a.processManager.Start(a.ctx, pkg); err != nil {
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
		}
	case packages.EventRemoved:
		delete(a.packages, evt.PackageID)
		_ = a.processManager.Stop(evt.PackageID)
	}
}
