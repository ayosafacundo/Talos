# Talos Status

This file tracks implementation maturity in a release-oriented format.

## Legend

- `implemented` - usable now
- `in_progress` - partially implemented, not yet production-ready
- `planned` - not started or intentionally deferred

## Current Snapshot

### Host Core

- `implemented` Wails host boot, lifecycle, and frontend binding (`main.go`, `app.go`)
- `implemented` package discovery with fsnotify and manifest parsing (`internal/packages`, `internal/manifest`)
- `implemented` process lifecycle manager (spawn/stop, env injection) (`internal/process`)

### IPC / Hub

- `implemented` local-only gRPC hub with protobuf contracts (`api/proto/talos/hub/v1/hub.proto`)
- `implemented` route + broadcast messaging
- `implemented` state RPCs (`SaveState`, `LoadState`)
- `implemented` permission and scoped path RPCs (`RequestPermission`, `ResolvePath`)

### Security + Persistence

- `implemented` permission request flow and host-side grant/deny handling
- `implemented` persisted grants in `Temp/permissions.json`
- `implemented` default filesystem scoping to package `data/` with external permission gate
- `implemented` permission settings tab (list grants, revoke/clear scope, recent request log, allow/deny modal on `permissions:request`)

### Frontend

- `implemented` package list + start/stop controls
- `implemented` event feed for discovery and bridge events
- `implemented` permission request action controls (allow/deny)
- `implemented` iframe bridge v1 (`talos:sdk:v1`, per-instance `bridge_token`, trusted `app_id` from iframe `postMessage` source)

### SDKs

- `implemented` Go SDK (`sdk/go/talos`) with hub calls and scoped file helpers
- `implemented` TypeScript `IframeBridgeTransport` (v1 bridge + `_talos_bt`)
- `implemented` Rust gRPC client over UDS (`Client::dial`, hub RPC parity with Go)

### Tiny App Demos

- `implemented` Go tiny app demo (`examples/tinyapps/go-demo`, `Packages/Tiny Go Demo`)
- `implemented` TypeScript iframe demo (`examples/tinyapps/ts-demo`, `Packages/Tiny TS Demo`)
- `in_progress` multi-app integration scenarios beyond baseline demos

### Build and Dev Experience

- `implemented` Makefile workflow (`proto`, `verify`, demos, `dev`, `app-build`)
- `implemented` full doc set (`DEVELOPMENT_FULL`, `SDK_GUIDE`, `TINY_APP_INIT`)
- `in_progress` cleanup/release hygiene across generated artifacts and alternative frontend tracks

## In-Progress Focus Areas

### 1) Iframe Bridge Hardening

Follow-ups:

- optional stricter `postMessage` targetOrigin where the runtime exposes a non-null origin
- optional hub integration test running Talos headless with a real UDS path

### 2) TS and Rust SDK Runtime Completion

Follow-ups:

- automated integration test in CI with `TALOS_TEST_SOCKET` against `wails dev`
- optional Node-side gRPC transport for non-iframe TS binaries

### 3) Permission UX and Policy Completeness

Follow-ups:

- async permission RPC so the first SDK attempt can block until the user answers (currently retry after Allow/Deny)
- richer audit log export

### 4) Integration/E2E Validation

Intended:

- confidence that host, SDKs, demos, and policy enforcement work together under realistic scenarios.

Needs:

- end-to-end scenarios with multiple concurrent tiny apps
- cross-feature regression suite (state, route, broadcast, fs scoping, permission transitions)

## Exit Signals for Phase 2

- iframe bridge hardened and policy-validated
- TS/Rust transport layers functionally complete with tests
- permission lifecycle complete (grant/deny/revoke UX)
- multi-app integration suite passing reliably
