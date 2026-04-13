package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"Talos/internal/buildmode"
	"Talos/internal/packageinstall"
	"Talos/internal/packages/repository"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	errInstallEmptyPath = errors.New("install: path is empty")
	errNoContext        = errors.New("install: app context not ready")
)

// RemotePackageDescriptor is a row for the future “browse repositories” UI.
type RemotePackageDescriptor struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source,omitempty"`
}

// DevelopmentFeaturesEnabled is true when manifest development.* may be honored (dev shell only).
func (a *App) DevelopmentFeaturesEnabled() bool {
	return buildmode.DevelopmentAllowed()
}

// InstallPackageFromZipPath installs a local .zip into Packages/ after validation.
func (a *App) InstallPackageFromZipPath(localPath string) (string, error) {
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		return "", errInstallEmptyPath
	}
	id, _, err := packageinstall.InstallFromZipFile(localPath, a.packagesDir, a.hashDir())
	return id, err
}

// PickZipAndInstall opens a file dialog and installs the selected zip.
func (a *App) PickZipAndInstall() (string, error) {
	if a.ctx == nil {
		return "", errNoContext
	}
	sel, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Select Talos package (.zip)",
		Filters: []runtime.FileFilter{{DisplayName: "Zip archive", Pattern: "*.zip"}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(sel) == "" {
		return "", nil
	}
	return a.InstallPackageFromZipPath(sel)
}

// InstallPackageFromURL downloads a zip from https? and installs it.
func (a *App) InstallPackageFromURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errInstallEmptyPath
	}
	id, _, err := packageinstall.InstallFromHTTPURL(a.ctx, rawURL, a.packagesDir, a.hashDir())
	return id, err
}

// InstallPackageFromGitHub downloads the GitHub zipball for owner/repo@ref and installs it.
func (a *App) InstallPackageFromGitHub(owner, repo, ref string) (string, error) {
	id, _, err := packageinstall.InstallFromGitHubZipball(a.ctx, owner, repo, ref, a.packagesDir, a.hashDir())
	return id, err
}

// ListRepositoryPackages returns catalog entries (stub until registry exists).
func (a *App) ListRepositoryPackages() []RemotePackageDescriptor {
	ctx := context.Background()
	if a.ctx != nil {
		ctx = a.ctx
	}
	rows, err := repository.NewStub().List(ctx)
	if err != nil {
		return nil
	}
	out := make([]RemotePackageDescriptor, 0, len(rows))
	for _, r := range rows {
		out = append(out, RemotePackageDescriptor{ID: r.ID, Name: r.Name, Source: "stub"})
	}
	return out
}

func (a *App) hashDir() string {
	return filepath.Join(a.rootDir, "Temp", "package_hashes")
}
