# Talos Full Development Guide

This document is the complete development reference for Talos. It consolidates setup, architecture, workflows, SDK usage, demos, packaging, and current implementation status.

## 1) Project Overview

Talos is a local-first desktop host for Tiny Apps.

- Host: Go + Wails
- UI shell: Svelte
- IPC: gRPC over local transports (Unix socket / Named Pipe)
- Tiny app packaging: `Packages/<App>/manifest.yaml` + `bin/` + `web/` + `data/`

Core design goals:

- local execution only
- app isolation
- permissioned access
- transport-level host mediation

## 2) Repository Map

Top-level directories:

- `api/` - protobuf contracts and generated Go stubs
- `internal/` - host internals (hub, discovery, process, security, state)
- `frontend/` - active Wails frontend
- `frontend_new/` - alternative/new frontend work (not currently the active Wails frontend)
- `sdk/` - Tiny App SDKs (`go`, `ts`, `rust`)
- `Packages/` - discoverable tiny app packages
- `docs/` - project docs

## 3) Toolchain Prerequisites

Required:

- Go
- Node.js + npm
- Wails CLI (`wails`)
- `protoc`
- `protoc-gen-go` and `protoc-gen-go-grpc`

Install Go proto plugins:

```bash
make install-tools
```

Install project deps:

```bash
make deps
```

## 4) Day-to-Day Commands

Main commands:

- `make proto` - regenerate protobuf and gRPC stubs
- `make verify` - run tests + go build + frontend build
- `make dev` - proto + Launchpad frontend, then `TALOS_DEV_MODE=1 wails dev`
- `make app-build` - proto + verify + `wails build -tags=production` (Talos + Launchpad only)

Bundled example apps under `Packages/` are built manually; see `docs/DEVELOPMENT.md`.

## 5) Host Runtime Architecture

### Wails App Layer

- `main.go` configures window and Wails runtime.
- `app.go` is the host orchestration layer exposed to frontend bindings.

### Package Discovery

- `internal/packages/discovery.go` watches `Packages/` via `fsnotify`.
- Manifest parser: `internal/manifest/parser.go`.

### Process Manager

- `internal/process/manager.go` starts/stops tiny app binaries.
- Host injects:
  - `TALOS_APP_ID`
  - `TALOS_APP_DATA_DIR`
  - `TALOS_HUB_SOCKET`

### gRPC Hub

- Proto: `api/proto/talos/hub/v1/hub.proto`
- Server: `internal/hub/server.go`
- Local transports:
  - unix socket on Linux/macOS
  - named pipe on Windows

### Security + State

- Permissions: `internal/security/permissions.go`
- Persisted grants: `internal/security/persist.go` -> `Temp/permissions.json`
- FS scope rules: `internal/security/fs_scope.go`
- App state store: `internal/state/store.go`

## 6) Tiny App Package Spec (Current)

Required for discovery/start:

- `manifest.yaml` with valid required fields
- executable path in `binary` (relative path)
- package directories as needed: `bin/`, `dist/`, `data/`

Typical package:

```text
Packages/My App/
├── manifest.yaml
├── bin/
├── data/
└── dist/
```

## 7) SDKs and Demos

### Go SDK

- Path: `sdk/go/talos`
- Primary implementation for direct hub communication.

### TypeScript SDK Baseline

- Path: `sdk/ts`
- Baseline transport abstraction; runtime demo currently uses iframe bridge transport.

### Rust SDK Baseline

- Path: `sdk/rust`
- Baseline API/trait shape.

### Included Example Apps

- Example Go app:
  - source: `Packages/Example Go App/src/main.go`
  - package: `Packages/Example Go App`
- Example Rust app:
  - source: `Packages/Example Rust App/src/main.rs`
  - package: `Packages/Example Rust App`
- Example TypeScript app:
  - source: `Packages/Example TS App/src/App.tsx`
  - package: `Packages/Example TS App`

## 8) Build and Packaging Pipeline

Recommended release build:

```bash
make app-build
```

This runs:

1. proto generation
2. Launchpad frontend build
3. verify steps
4. `wails build -tags=production`

Packaging output name is configured in `wails.json` as `Talos`.

## 9) Testing and Quality

Backend checks:

```bash
go test ./...
go build ./...
```

Frontend checks:

```bash
npm --prefix frontend run build
```

Combined:

```bash
make verify
```

## 10) Troubleshooting

- App missing in UI:
  - verify manifest is valid and uses expected fields
  - ensure package lives under `Packages/`
- Binary won’t start:
  - ensure executable exists and is executable
  - verify `binary` is a relative path in manifest
- Hub connectivity issues:
  - ensure Talos host is running
  - verify tiny app reads `TALOS_HUB_SOCKET`
- Permission issues:
  - inspect `Temp/permissions.json`
  - re-trigger request through UI if needed
- TS iframe example not updating:
  - rebuild the package (`npm --prefix "Packages/Example TS App" run build`) and restart Talos

## 11) Current Status Snapshot

Implemented:

- Phase 1 scaffolding, discovery, process manager, core hub
- Phase 2 baseline:
  - state save/load
  - permission request flow with persistence
  - filesystem scope enforcement
  - host-iframe bridge baseline
  - Go/TS/Rust SDK baselines
- Go, Rust, and TypeScript example tiny apps

Not finished / still maturing:

- full production-grade iframe bridge hardening (policy/allowlists)
- fully transport-complete TS and Rust SDK implementations equivalent to Go SDK depth
- broader integration/e2e coverage across multiple real tiny apps

## 12) Related Documentation

- `README.md`
- `docs/DEVELOPMENT.md`
- `docs/PHASE1.md`
- `docs/PHASE2_PREP.md`
- `docs/SDK_GUIDE.md`
- `docs/TINY_APP_INIT.md`
