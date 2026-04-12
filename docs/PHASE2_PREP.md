# Phase 2 Plan and Progress

This document started as readiness planning and now tracks Phase 2 execution progress.

## Goal

Phase 2 introduces the Talos SDK and runtime enforcement surfaces:

- Go/Rust/TS client libraries
- FS scoping and permission gates
- Iframe bridge between Wails host and sandboxed app frontends

## Scope Boundary

What is included now:

- Interface and sequencing decisions
- Data contracts
- Delivery checklist and test strategy
- Progress snapshots for Phase 2 implementation

What is not fully complete yet:

- Full Rust and TS gRPC transport implementations
- Production-grade iframe policy/allowlist hardening

## Proposed Deliverable Order

1) **Define Hub SDK RPCs**

- Add/update proto contracts for:
  - `SaveState` / `LoadState`
  - `SendMessage` (already conceptually covered by Phase 1 hub routing)
  - `RequestPermission`
- Freeze a versioned RPC namespace before generating clients.

2) **Implement Go SDK first**

- Build a minimal host-aware client package in Go:
  - Connection bootstrap from hub endpoint
  - Typed wrappers around RPCs
  - Retry/error normalization for local IPC

3) **Generate TS and Rust SDK baselines**

- TS:
  - Start with generated/grpc-compatible client surface.
  - Wrap in tiny ergonomic API methods matching product language.
- Rust:
  - Define crate layout and transport strategy for UDS/Named Pipe parity.

4) **Host-side permission and scope enforcement**

- Add policy layer in host process for:
  - Default deny outside `/Packages/[App]/data/`
  - Elevation flow via `RequestPermission`
- Ensure enforcement sits in host boundary, not SDK only.

5) **Iframe bridge**

- Define host-to-iframe and iframe-to-host message schema.
- Add narrow, explicit event whitelist.
- Tie bridge identity to manifest `id` and running instance.

## Contracts to Finalize Before Coding

- `manifest.yaml` fields required for Phase 2:
  - app identity and version
  - requested permissions
  - data scope metadata (if any additional fields are needed)
- Permission model:
  - grant granularity (session vs persistent)
  - deny defaults
  - revocation semantics
- SDK compatibility policy:
  - semantic versioning
  - backward compatibility for RPC fields

## Test Strategy for Phase 2 (Planned)

- Unit:
  - SDK wrappers, serialization, and transport behavior
  - policy decision logic for permission/scoping
- Integration:
  - host + mock tiny app over local socket transport
  - permission request and deny/allow flows
- E2E:
  - iframe-host bridge event flow with at least one sample tiny app

## Phase 2 Start Criteria

- [x] Proto contract draft reviewed
- [x] Permission semantics agreed (initial in-memory model)
- [x] SDK package structure approved
- [x] Bridge event schema approved (initial)

## Exit Criteria for Phase 2

- [x] Go SDK usable by a tiny app demo
- [x] TS and Rust SDK baselines generated and documented
- [x] FS scoping enforcement validated with deny-by-default tests
- [ ] Iframe bridge passes host-app message exchange tests

## Progress Snapshot

- [x] Hub proto expanded with `SaveState`, `LoadState`, `RequestPermission`.
- [x] Generated Go gRPC stubs regenerated with `protoc`.
- [x] Host-side in-memory state store added.
- [x] Host-side permission service + request flow added.
- [x] Permission grants persisted to `Temp/permissions.json`.
- [x] FS scope manager added with deny-by-default behavior outside `data/`.
- [x] Hub `ResolvePath` RPC added for host-validated path resolution.
- [x] Go SDK wrapper added in `sdk/go/talos`.
- [x] TS SDK baseline wrapper added in `sdk/ts`.
- [x] Rust SDK baseline wrapper added in `sdk/rust`.
- [x] Initial Wails <-> iframe bridge event flow wired in host/frontend.
- [x] Go tiny app demo implemented in `examples/tinyapps/go-demo`.
- [x] TypeScript tiny app iframe demo implemented in `examples/tinyapps/ts-demo`.
- [x] SDK and tiny-app initialization docs added.
