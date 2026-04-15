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

### Tiny App Examples

- `implemented` Example Go app (`Packages/Example Go App`)
- `implemented` Example Rust app (`Packages/Example Rust App`)
- `implemented` Example TypeScript iframe app (`Packages/Example TS App`)
- `implemented` multi-app hub integration coverage (`internal/hub/integration_grpc_test.go`, `make integration-hub`)

### Build and Dev Experience

- `implemented` Makefile workflow (`proto`, `verify`, `dev`, `app-build`; Talos + Launchpad only)
- `implemented` full doc set (`DEVELOPMENT_FULL`, `SDK_GUIDE`, `TINY_APP_INIT`)
- `implemented` CI aligned with `make verify` (see `.github/workflows/ci.yml`)

## Phase 2 follow-ups (non-blocking / stretch)

- Optional: headless `wails dev` smoke in CI.
- Optional: Node-side gRPC transport for non-iframe TypeScript binaries.
- Optional: Playwright (or similar) full UI E2E beyond hub integration tests.

**Sign-off checklist:** [docs/PHASE2_SIGNOFF.md](PHASE2_SIGNOFF.md)

## Phase 3 snapshot (distribution and trust)

- `implemented` (baseline) package hash manifests and install-time writes (`internal/packageinstall/hash.go`)
- `implemented` optional Ed25519 package signatures and trust evaluation (`internal/packageinstall/trust.go`, `Temp/trusted_keys/`)
- `implemented` update channel fetch + apply (`internal/updates`)
- `implemented` HTTP package catalog + Launchpad browse (`internal/packages/repository/http.go`, `TALOS_CATALOG_URL`)

See [docs/PHASE3.md](PHASE3.md) for operator notes.

## Phase 4 (post–Phase 3)

Roadmap and exit criteria: [docs/PHASE4.md](PHASE4.md).

## Phase 5 (documentation generation)

- `implemented` architecture guide: [docs/ARCHITECTURE.md](ARCHITECTURE.md)
- `implemented` plugin developer guide: [docs/PLUGIN_GUIDE.md](PLUGIN_GUIDE.md)
- `implemented` manifest reference: [docs/MANIFEST_SPEC.md](MANIFEST_SPEC.md)

## Phase 6 (final polish / release readiness)

- `implemented` release polish checklist: [docs/PHASE6_RELEASE_POLISH.md](PHASE6_RELEASE_POLISH.md)

## Exit Signals for Phase 2 (reference)

- iframe bridge hardened and policy-validated — see [docs/dev/iframe-bridge.md](dev/iframe-bridge.md)
- TS/Rust transport layers functionally complete with tests — see `sdk/ts`, `sdk/rust` READMEs
- permission lifecycle complete (grant/deny/revoke UX) — Launchpad + audit JSONL
- multi-app integration suite passing reliably — `make integration-hub`
