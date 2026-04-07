# Phase 1 Implementation Notes

This document captures what is implemented for Phase 1 and where each component lives.

## 1) Wails + Svelte Host Scaffold

- Host entrypoint: `main.go`
- Wails-bound host API and lifecycle: `app.go`
- Frontend shell: `frontend/src/App.svelte`

Current window configuration is fixed to `450x700` to align with the Talos SRS.

## 2) Package Discovery (`/Packages`)

- Service: `internal/packages/discovery.go`
- Manifest parser: `internal/manifest/parser.go`

Behavior:

- Watches `Packages/` with `fsnotify`.
- Detects package add/update/remove events.
- Parses `manifest.yaml` from each package.
- Emits Wails event `packages:event` to frontend with:
  - `added`
  - `updated`
  - `removed`
  - `error`

## 3) Process Lifecycle Manager

- Service: `internal/process/manager.go`

Behavior:

- Starts app binaries from manifest-defined relative path (typically `bin/...`).
- Tracks running processes by app id.
- Supports stop by app id and stop-all on shutdown.

## 4) Central gRPC Hub (Option B)

- Proto definition: `api/proto/talos/hub/v1/hub.proto`
- Generated stubs:
  - `api/proto/talos/hub/v1/hub.pb.go`
  - `api/proto/talos/hub/v1/hub_grpc.pb.go`
- Hub server: `internal/hub/server.go`
- Platform listeners:
  - Unix sockets: `internal/hub/listener_unix.go`
  - Windows named pipes: `internal/hub/listener_windows.go`

Behavior:

- Starts one local-only gRPC server in host process.
- Uses Unix domain socket on Linux/macOS.
- Uses named pipes on Windows.
- Supports:
  - `Route` RPC for request/response
  - `Broadcast` RPC for fanout

## 5) Frontend Integration

The frontend currently provides a thin Phase 1 control surface:

- Shows hub endpoint.
- Lists discovered packages.
- Start/Stop buttons for package processes.
- Live discovery event feed.

This is intentionally minimal and intended as a verification UI for Phase 1 backend services.
