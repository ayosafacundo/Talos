# Startup to Launchpad: Under-the-Hood Flow

This document describes exactly how Talos goes from process start to showing Launchpad in an iframe.

## 1) Binary boot and Launchpad embedding

1. `main.go` starts.
2. `Packages/Launchpad/dist` is embedded via:
   - `//go:embed all:Packages/Launchpad/dist`
3. Wails starts with:
   - `OnStartup: app.startup`
   - `OnShutdown: app.shutdown`
   - `Bind: []interface{}{ app }`
4. Because `App` is bound, frontend JS can call Go methods through `window.go.main.App`.

## 2) Backend startup (`app.startup`)

When Wails calls `app.startup(ctx)`, Talos initializes backend services in this order:

1. Creates cancellable app context.
2. Creates permissions manager with an event-emitting prompt hook.
   - emits `permissions:request` runtime event for host UI decisions.
3. Loads persisted permission grants from `Temp/permissions.json`.
4. Creates filesystem scope manager.
5. Registers hub hooks for:
   - state save/load,
   - permission requests,
   - scoped path resolution.
6. Starts central gRPC hub (`a.hub.Start()`).
7. Starts package discovery watcher (`a.discovery.Start(...)`) in a goroutine.
8. Discovery callback does two things for each event:
   - `a.handlePackageEvent(evt)` updates in-memory package registry (`a.packages`),
   - `runtime.EventsEmit(a.ctx, "packages:event", evt)` notifies host UI.

## 3) Package discovery internals

`internal/packages/discovery.go` handles both initial scan and live updates:

- Ensures `Packages/` exists and watches it.
- For each package directory, tries to parse `manifest.yaml`.
- Validation path:
  1. parse + validate manifest (`internal/manifest/parser.go`)
  2. ensure `web_entry` exists on disk (`filepath.Join(packageDir, def.WebEntry)`)
- If valid, emits `added` or `updated`.
- If removed/renamed, emits `removed`.

Important behavior:

- `web_entry` defaults to `dist/index.html` if omitted.
- launchpad and any web-only app can work without `binary`.

## 4) Frontend startup sequence

The embedded frontend is built from `Packages/Launchpad/` and bootstraps through `Packages/Launchpad/src/App.tsx`.

In `App.tsx`, startup is:

1. `start()` runs.
2. `bootstrap()` loads themes and validates launchpad with:
   - `window.go.main.App.GetStartupLaunchpad()`
3. It then loads installed/store package catalogs.
4. Event listeners are registered for SDK messages, package updates, permissions, and log updates.
5. Launchpad UI is shown as the root panel (not inside a launchpad iframe).

## 5) How Launchpad is selected and shown

Launchpad id is hardcoded in the frontend:

- `const LAUNCHPAD_ID = "app.launchpad"`

Startup contract flow:

1. frontend calls `GetStartupLaunchpad()`.
2. backend returns success only if `app.launchpad` exists with valid `web_entry`.
3. if contract fails:
   - frontend surfaces startup error,
   - backend also emits fatal error and quits the app.
4. if contract succeeds:
   - frontend continues and renders Launchpad as root UI.

## 6) How backend computes Launchpad URL

`app.go` method `GetStartupLaunchpad()`:

1. Reads `a.packages["app.launchpad"]`.
2. Verifies package and manifest exist.
3. Verifies `manifest.web_entry` is not empty.
4. Returns `AppManifestView` from `packageToManifestView(...)`.

`packageToManifestView(...)` builds:

- `URL = "file://" + filepath.Join(pkg.DirPath, pkg.Manifest.WebEntry)`

That URL is still used as canonical package URL metadata, but the embedded frontend already runs Launchpad directly and uses package URLs when opening other apps in iframes.

## 7) Why Launchpad appears first

- App state defaults `launchpadVisible = true`.
- Startup validates launchpad before normal UI interactions.
- App iframes are only shown after user launches/focuses non-launchpad apps.

So the first meaningful content is the Launchpad panel itself if the package exists and is valid.

## 8) Live update path after startup

When package files change:

1. discovery emits event.
2. backend emits `packages:event` to frontend.
3. frontend listener does:
   - reload package catalogs,
   - refresh active app iframe URLs for changed non-launchpad packages.

This is the mechanism that lets package frontends refresh without restarting Talos.

## 9) Launchpad package requirements checklist

For Talos startup to show Launchpad correctly:

- directory exists: `Packages/Launchpad/`
- manifest exists: `Packages/Launchpad/manifest.yaml`
- manifest id is exactly: `app.launchpad`
- manifest `web_entry` points to a real file (default expected path: `dist/index.html`)

## 10) Existing tests for this flow

`app_launchpad_test.go` verifies:

- missing launchpad fails startup contract,
- valid launchpad returns launchpad startup payload and URL,
- launchpad appears in installed package listing.

Run:

```bash
go test ./...
```

