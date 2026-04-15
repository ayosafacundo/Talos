package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Talos/internal/manifest"
	"Talos/internal/packages"
)

func TestAppendPackageSDKLog_NoOpWhenDevOff(t *testing.T) {
	t.Setenv("TALOS_DEV_MODE", "")
	dir := t.TempDir()
	app := NewApp()
	app.rootDir = dir
	app.packageSDKLogDir = filepath.Join(dir, "Temp", "logs", "packages", "sdk")
	pkg := &packages.PackageInfo{
		DirName: "Pkg",
		DirPath: filepath.Join(dir, "Packages", "Pkg"),
		Manifest: &manifest.Definition{
			ID: "app.sdk.off",
		},
	}
	app.mu.Lock()
	app.packages["app.sdk.off"] = pkg
	app.mu.Unlock()
	if err := app.appendPackageSDKLog("app.sdk.off", "INFO", "x"); err != nil {
		t.Fatal(err)
	}
	_, err := os.Stat(filepath.Join(app.packageSDKLogDir, "app.sdk.off.log"))
	if err == nil {
		t.Fatal("expected no log file when development is off")
	}
}

func TestAppendPackageSDKLog_WritesWhenPerDirDevOn(t *testing.T) {
	t.Setenv("TALOS_DEV_MODE", "")
	dir := t.TempDir()
	app := NewApp()
	app.rootDir = dir
	app.packageSDKLogDir = filepath.Join(dir, "Temp", "logs", "packages", "sdk")
	app.devMu.Lock()
	app.prefsDevModeByDir["Pkg"] = true
	app.devMu.Unlock()
	pkg := &packages.PackageInfo{
		DirName: "Pkg",
		DirPath: filepath.Join(dir, "Packages", "Pkg"),
		Manifest: &manifest.Definition{
			ID: "app.sdk.on",
		},
	}
	app.mu.Lock()
	app.packages["app.sdk.on"] = pkg
	app.mu.Unlock()
	if err := app.appendPackageSDKLog("app.sdk.on", "INFO", "hello"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(app.packageSDKLogDir, "app.sdk.on.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "hello") || !strings.Contains(string(b), "INFO") {
		t.Fatalf("log content: %q", string(b))
	}
}
