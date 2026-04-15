# Talos Developer Guide

This directory is a practical, code-aligned manual for developing Talos without guessing where logic lives.

## Who this is for

Use this guide if you want to:

- run Talos locally,
- understand the architecture end to end,
- change startup/package loading behavior safely,
- extend host-to-tiny-app SDK behavior.

## Read this first

1. `docs/dev/CODEBASE_MAP.md` - where each subsystem lives.
2. `docs/dev/STARTUP_TO_LAUNCHPAD.md` - exact runtime flow from app start to launchpad iframe.
3. `docs/SDK_GUIDE.md` - tiny-app side usage.

## Local development workflow

From repo root:

```bash
make deps
make proto
make verify
make dev
```

Common targets:

- `make test` - run Go tests only.
- `make build` - compile Go packages only.
- `make app-build` - Talos + Launchpad only (proto + verify + `wails build -tags=production`).

Other folders under `Packages/` are built with your own tooling; see `docs/DEVELOPMENT.md` for optional example commands.

## Runtime model in one paragraph

Talos is a Go/Wails desktop host that now runs directly through the Launchpad frontend built from `frontend/` into `Packages/Launchpad/dist`. On startup, Go creates a central IPC hub, starts package discovery on `Packages/`, and exposes app APIs to JS through Wails bindings. The frontend validates that the required package `app.launchpad` exists and then uses Launchpad as the root UI that opens and switches to other app iframes. Package changes are pushed to the UI through Wails events so app iframes can refresh without restarting Talos.

## Source of truth files

- App entrypoint: `main.go`
- Host backend orchestration: `app.go`
- Launchpad-root frontend shell and iframe orchestration: `frontend/src/App.svelte`
- Package scanning: `internal/packages/discovery.go`
- Manifest parsing/validation: `internal/manifest/parser.go`
- Tiny app process lifecycle: `internal/process/manager.go`
- Hub service: `internal/hub/server.go`

