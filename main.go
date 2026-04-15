package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:Packages/Launchpad/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Talos",
		Width:     1200,
		Height:    820,
		MinWidth:  900,
		MinHeight: 620,
		// Production builds disable the browser context menu unless this is set (needed for Inspect Element / debugging).
		EnableDefaultContextMenu: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
			Middleware: assetserver.ChainMiddleware(
				talosPackageMiddleware(app.packagesDir),
			),
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Printf("wails: %v", err)
	}
}
