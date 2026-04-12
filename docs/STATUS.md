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
- `in_progress` permission UX policy hardening (revocation UX, rationale/history, richer policy controls)

### Frontend

- `implemented` package list + start/stop controls
- `implemented` event feed for discovery and bridge events
- `implemented` permission request action controls (allow/deny)
- `in_progress` robust iframe bridge hardening (strict origin/event policy and safer envelopes)

### SDKs

- `implemented` Go SDK (`sdk/go/talos`) with hub calls and scoped file helpers
- `in_progress` TypeScript SDK transport parity with Go SDK depth
- `in_progress` Rust SDK transport parity with Go SDK depth

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

Intended:

- Safe host-iframe message bus with strict validation, app isolation, and explicit event allowlists.

Needs:

- strict origin checks
- per-app channel/auth token strategy
- envelope schema validation and reject logging
- tighter separation of debug/test events from production channels

### 2) TS and Rust SDK Runtime Completion

Intended:

- TS and Rust SDKs with near-parity behavior to Go SDK for local IPC and scoped IO flows.

Needs:

- concrete transport implementations (not only baseline wrappers)
- stable retry/timeout and error normalization
- integration tests against running Talos host

### 3) Permission UX and Policy Completeness

Intended:

- complete permission lifecycle with clear prompts, persistence, and revocation semantics.

Needs:

- revocation UI
- richer permission metadata/history
- clearer denied-state UX and retry flows

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
