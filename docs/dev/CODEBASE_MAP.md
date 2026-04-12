# Talos Codebase Map

This is the "where is everything?" reference for day-to-day development.

## Top-level directories

- `api/proto/talos/hub/v1/`
  - gRPC protocol for host hub (`hub.proto` + generated Go files).
- `Packages/Launchpad/`
  - Wails frontend directory and Launchpad UI source (React + Vite).
  - builds into `Packages/Launchpad/dist`.
  - includes generated Wails JS bindings under `Packages/Launchpad/wailsjs/`.
- `internal/`
  - backend subsystems (manifest parser, discovery, process manager, hub, security, state).
- `Packages/`
  - runtime tiny app packages.
  - each package is expected to include `manifest.yaml` and a `web_entry` asset (default `dist/index.html`).
- `sdk/`
  - tiny app SDKs (Go/TS/Rust).
- `examples/`
  - tiny app sample code and build inputs.
- `docs/`
  - architecture and development docs.

## Core backend files

- `main.go`
  - Wails app bootstrap.
  - embeds Launchpad package dist from `Packages/Launchpad/dist`.
  - configures window, startup/shutdown hooks, and binds `App` methods to JS.

- `app.go`
  - main backend orchestration and Wails bridge surface.
  - owns:
    - package state (`a.packages`),
    - hub lifecycle,
    - process manager,
    - permissions/state logic,
  - methods called from frontend JS.
  - critical frontend bridge methods:
    - `ListPackages()`
    - `GetStartupLaunchpad()`
    - `StartPackage()`, `StopPackage()`
    - SDK bridge helpers (`RouteMessage`, `BroadcastMessage`, `ResolveScopedPath`, state and permission wrappers).

## Package system

- `internal/manifest/parser.go`
  - strict parsing/validation of `manifest.yaml`.
  - defaults `web_entry` to `dist/index.html`.
  - enforces relative paths for `binary` and `web_entry`.

- `internal/packages/discovery.go`
  - fsnotify watcher over `Packages/`.
  - initial scan + recursive watch of package trees.
  - emits normalized events: `added`, `updated`, `removed`, `error`.
  - validates `web_entry` exists before accepting package.

- `internal/process/manager.go`
  - starts/stops tiny app binaries.
  - handles web-only packages (`binary == ""`) as no-op for process start.
  - injects environment (`TALOS_APP_ID`, `TALOS_APP_DATA_DIR`, `TALOS_HUB_SOCKET`) when launching binaries.

## IPC, state, and security

- `internal/hub/server.go`
  - central gRPC service with route/broadcast/state/permission/path APIs.
  - each discovered package gets a host-side route handler registration in `app.go`.

- `internal/state/store.go`
  - in-memory app state store used by SDK save/load operations.

- `internal/security/permissions.go`
  - per-app scope grants.
  - `fs:data` is always considered granted.

- `internal/security/fs_scope.go`
  - resolves file paths under package `data/` by default.
  - rejects path escapes unless `fs:external` is granted.

## Frontend UI (what user sees)

- `Packages/Launchpad/src/App.tsx`
  - root UI shell running as Launchpad directly (no hostui layer).
  - renders Launchpad panel, app iframe stack, settings panel, and context menu.
  - routes iframe `postMessage` SDK calls to Go bridge methods.
  - listens to `packages:event` for live package/frontend refresh.
  - in dev mode, shows a live log panel fed by runtime and package logs.

- `Packages/Launchpad/vite.config.ts`
  - outputs build artifacts to `Packages/Launchpad/dist`.

## Launchpad package

- `Packages/Launchpad/manifest.yaml`
  - required package id is `app.launchpad`.
  - must expose `web_entry` (currently `dist/index.html`).

## Tests relevant to startup + launchpad

- `app_launchpad_test.go`
  - validates launchpad startup contract:
    - missing launchpad returns error,
    - valid launchpad returns startup view with URL,
    - launchpad is present in installed apps list.

