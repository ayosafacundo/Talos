# Phase 2 Readiness Plan (No Implementation Yet)

This document prepares the codebase for Phase 2 execution without starting feature work.

## Goal

Phase 2 introduces the Talos SDK and runtime enforcement surfaces:

- Go/Rust/TS client libraries
- FS scoping and permission gates
- Iframe bridge between Wails host and sandboxed app frontends

## Scope Boundary

What is included now:

- Interface and sequencing decisions
- Data contracts to implement next
- Delivery checklist and test strategy

What is explicitly not included now:

- SDK code
- Permission gate implementation
- Iframe bridge implementation
- FS enforcement logic

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

## Entry Criteria to Start Phase 2

- [ ] Proto contract draft reviewed
- [ ] Permission semantics agreed
- [ ] SDK package structure approved
- [ ] Bridge event schema approved

## Exit Criteria for Phase 2

- [ ] Go SDK usable by a tiny app demo
- [ ] TS and Rust SDK baselines generated and documented
- [ ] FS scoping enforcement validated with deny-by-default tests
- [ ] Iframe bridge passes host-app message exchange tests
