package packages

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"

	"Talos/internal/manifest"
)

const (
	EventAdded   = "added"
	EventUpdated = "updated"
	EventRemoved = "removed"
	EventError   = "error"
)

// PackageInfo describes a discovered package.
type PackageInfo struct {
	DirName    string               `json:"dir_name"`
	DirPath    string               `json:"dir_path"`
	Manifest   *manifest.Definition `json:"manifest,omitempty"`
	ManifestOK bool                 `json:"manifest_ok"`
}

// DiscoveryEvent is emitted when package state changes.
type DiscoveryEvent struct {
	Type      string       `json:"type"`
	PackageID string       `json:"package_id,omitempty"`
	Package   *PackageInfo `json:"package,omitempty"`
	Error     string       `json:"error,omitempty"`
}

type EventHandler func(evt DiscoveryEvent)

// Discovery watches package manifests and emits normalized events.
type Discovery struct {
	rootDir string
	onEvent EventHandler

	mu       sync.RWMutex
	packages map[string]*PackageInfo
}

func NewDiscovery(rootDir string, onEvent EventHandler) *Discovery {
	return &Discovery{
		rootDir:  rootDir,
		onEvent:  onEvent,
		packages: make(map[string]*PackageInfo),
	}
}

// Snapshot returns currently known packages keyed by package id.
func (d *Discovery) Snapshot() map[string]*PackageInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	out := make(map[string]*PackageInfo, len(d.packages))
	for id, info := range d.packages {
		cp := *info
		out[id] = &cp
	}
	return out
}

// Start begins initial discovery and fsnotify watch loop.
func (d *Discovery) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("packages: watcher init failed: %w", err)
	}
	defer watcher.Close()

	if err := os.MkdirAll(d.rootDir, 0o755); err != nil {
		return fmt.Errorf("packages: create root dir failed: %w", err)
	}
	if err := watcher.Add(d.rootDir); err != nil {
		return fmt.Errorf("packages: watch root dir failed: %w", err)
	}

	if err := d.scanAll(watcher); err != nil {
		d.emit(DiscoveryEvent{Type: EventError, Error: err.Error()})
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			d.handleFsEvent(watcher, ev)
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			d.emit(DiscoveryEvent{Type: EventError, Error: err.Error()})
		}
	}
}

func (d *Discovery) scanAll(watcher *fsnotify.Watcher) error {
	entries, err := os.ReadDir(d.rootDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		packageDir := filepath.Join(d.rootDir, entry.Name())
		if err := d.watchPackageTree(watcher, packageDir); err != nil {
			d.emit(DiscoveryEvent{Type: EventError, Error: fmt.Sprintf("watch %q failed: %v", packageDir, err)})
		}
		d.syncPackage(packageDir)
	}
	return nil
}

func (d *Discovery) handleFsEvent(watcher *fsnotify.Watcher, ev fsnotify.Event) {
	cleanPath := filepath.Clean(ev.Name)
	if cleanPath == d.rootDir {
		return
	}

	rel, err := filepath.Rel(d.rootDir, cleanPath)
	if err != nil {
		return
	}
	if rel == "." || strings.HasPrefix(rel, "..") {
		return
	}

	parts := strings.Split(rel, string(os.PathSeparator))
	packageDir := filepath.Join(d.rootDir, parts[0])

	if len(parts) == 1 {
		if ev.Has(fsnotify.Create) {
			if isDir(packageDir) {
				_ = d.watchPackageTree(watcher, packageDir)
				d.syncPackage(packageDir)
			}
			return
		}
		if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
			d.removePackageByDir(packageDir)
			return
		}
	}

	if ev.Has(fsnotify.Create) && isDir(cleanPath) {
		_ = d.watchPackageTree(watcher, cleanPath)
	}

	if strings.EqualFold(filepath.Base(cleanPath), manifest.ManifestFileName) ||
		ev.Has(fsnotify.Create) || ev.Has(fsnotify.Write) || ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
		d.syncPackage(packageDir)
	}
}

func (d *Discovery) watchPackageTree(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !dirEntry.IsDir() {
			return nil
		}
		if addErr := watcher.Add(path); addErr != nil {
			d.emit(DiscoveryEvent{Type: EventError, Error: fmt.Sprintf("watch %q failed: %v", path, addErr)})
		}
		return nil
	})
}

func (d *Discovery) syncPackage(packageDir string) {
	def, err := manifest.ParsePackageDir(packageDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			d.removePackageByDir(packageDir)
			return
		}
		d.emit(DiscoveryEvent{
			Type:  EventError,
			Error: fmt.Sprintf("parse manifest in %q failed: %v", packageDir, err),
		})
		return
	}

	info := &PackageInfo{
		DirName:    filepath.Base(packageDir),
		DirPath:    packageDir,
		Manifest:   def,
		ManifestOK: true,
	}
	webEntryPath := filepath.Join(packageDir, def.WebEntry)
	if st, statErr := os.Stat(webEntryPath); statErr != nil || st.IsDir() {
		d.emit(DiscoveryEvent{
			Type:  EventError,
			Error: fmt.Sprintf("required web entry missing for %q: %s", def.ID, webEntryPath),
		})
		return
	}

	removedEvents := make([]DiscoveryEvent, 0, 1)
	d.mu.Lock()
	for existingID, existing := range d.packages {
		if filepath.Clean(existing.DirPath) == packageDir && existingID != def.ID {
			delete(d.packages, existingID)
			removedEvents = append(removedEvents, DiscoveryEvent{
				Type:      EventRemoved,
				PackageID: existingID,
				Package:   existing,
			})
		}
	}
	_, existed := d.packages[def.ID]
	d.packages[def.ID] = info
	d.mu.Unlock()

	for _, evt := range removedEvents {
		d.emit(evt)
	}

	evtType := EventAdded
	if existed {
		evtType = EventUpdated
	}

	d.emit(DiscoveryEvent{
		Type:      evtType,
		PackageID: def.ID,
		Package:   info,
	})
}

func (d *Discovery) removePackageByDir(packageDir string) {
	d.mu.Lock()
	var removed *DiscoveryEvent

	for id, info := range d.packages {
		if filepath.Clean(info.DirPath) != packageDir {
			continue
		}
		delete(d.packages, id)
		evt := DiscoveryEvent{
			Type:      EventRemoved,
			PackageID: id,
			Package:   info,
		}
		removed = &evt
		break
	}
	d.mu.Unlock()
	if removed != nil {
		d.emit(*removed)
	}
}

func (d *Discovery) emit(evt DiscoveryEvent) {
	if d.onEvent != nil {
		d.onEvent(evt)
	}
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
