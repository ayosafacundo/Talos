package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Talos/internal/manifest"
	"Talos/internal/packages"
)

func TestGetStartupLaunchpadMissing(t *testing.T) {
	app := NewApp()
	app.packages = map[string]*packages.PackageInfo{}

	_, err := app.GetStartupLaunchpad()
	if err == nil {
		t.Fatalf("expected error when launchpad package is missing")
	}
	if !strings.Contains(err.Error(), launchpadPackageID) {
		t.Fatalf("expected launchpad id in error, got: %v", err)
	}
}

func TestGetStartupLaunchpadReturnsLaunchpadManifest(t *testing.T) {
	app := NewApp()
	launchpadDir := filepath.Join("Packages", "Launchpad")
	app.packages = map[string]*packages.PackageInfo{
		launchpadPackageID: {
			DirPath: launchpadDir,
			Manifest: &manifest.Definition{
				ID:       launchpadPackageID,
				Name:     "Launchpad",
				WebEntry: "dist/index.html",
				Icon:     "🚀",
			},
		},
	}

	got, err := app.GetStartupLaunchpad()
	if err != nil {
		t.Fatalf("expected launchpad startup payload, got error: %v", err)
	}
	if got.ID != launchpadPackageID {
		t.Fatalf("expected id %q, got %q", launchpadPackageID, got.ID)
	}
	if got.Name != "Launchpad" {
		t.Fatalf("expected launchpad name, got %q", got.Name)
	}
	if !strings.HasSuffix(got.URL, filepath.ToSlash("Packages/Launchpad/dist/index.html")) &&
		!strings.HasSuffix(got.URL, "Packages\\Launchpad\\dist\\index.html") {
		t.Fatalf("expected launchpad URL suffix, got %q", got.URL)
	}
}

func TestGetInstalledAppsIncludesLaunchpad(t *testing.T) {
	app := NewApp()
	launchpadDir := filepath.Join("Packages", "Launchpad")
	app.packages = map[string]*packages.PackageInfo{
		launchpadPackageID: {
			DirPath: launchpadDir,
			Manifest: &manifest.Definition{
				ID:       launchpadPackageID,
				Name:     "Launchpad",
				WebEntry: "dist/index.html",
			},
		},
	}

	out := app.GetInstalledApps()
	if len(out) != 1 {
		t.Fatalf("expected one installed app, got %d", len(out))
	}
	if out[0].ID != launchpadPackageID {
		t.Fatalf("expected launchpad in installed apps, got %q", out[0].ID)
	}
}

func TestPackageToManifestViewResolvesGIFIconAsDataURL(t *testing.T) {
	t.Parallel()

	packageDir := t.TempDir()
	distDir := filepath.Join(packageDir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatalf("mkdir dist failed: %v", err)
	}
	gifPath := filepath.Join(distDir, "icon.gif")
	minimalGIF := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff!\xf9\x04\x00\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;")
	if err := os.WriteFile(gifPath, minimalGIF, 0o644); err != nil {
		t.Fatalf("write gif failed: %v", err)
	}

	pkg := &packages.PackageInfo{
		DirPath: packageDir,
		Manifest: &manifest.Definition{
			ID:      "app.test.gif",
			Name:    "GIF App",
			Icon:    "/dist/icon.gif",
			WebEntry: "dist/index.html",
		},
	}

	out := packageToManifestView(pkg)
	if !strings.HasPrefix(out.Icon, "data:image/gif;base64,") {
		t.Fatalf("expected gif data URL, got %q", out.Icon)
	}
}
