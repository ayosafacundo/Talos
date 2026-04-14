# Phase 2 sign-off checklist

Use this list with [docs/STATUS.md](STATUS.md) exit signals. Items reference where verification lives.

## Exit signals (all should be checked before declaring Phase 2 complete)

| Signal | Verification |
|--------|----------------|
| Iframe bridge hardened and policy-validated | [docs/dev/iframe-bridge.md](dev/iframe-bridge.md), [`Packages/Launchpad/src/bridge.test.ts`](../Packages/Launchpad/src/bridge.test.ts), Launchpad `allowed_origins` + token trust |
| TS/Rust transport layers functionally complete with tests | [`sdk/ts`](../sdk/ts/) (transport tests), [`sdk/rust`](../sdk/rust/) (Unix `Dial` + Windows caveat in README); optional Node gRPC deferred |
| Permission lifecycle complete (grant/deny/revoke UX) | Launchpad Settings → Permissions + **Permission audit** table; [`app.go`](../app.go) `appendPermissionAudit` on grant/deny/revoke |
| Multi-app integration suite passing reliably | `make integration-hub` → [`internal/hub/integration_grpc_test.go`](../internal/hub/integration_grpc_test.go); tiers: [docs/dev/INTEGRATION_TIERS.md](dev/INTEGRATION_TIERS.md) |

## CI

- **Green bar:** `make verify` (includes `production-gate`, Launchpad tests, `go test ./...`, `go build`).
- Hub socket tests: `bash scripts/run_integration_hub.sh` (optional `TALOS_TEST_SOCKET`).

## Deferred (not Phase 2 blockers)

- Headless `wails dev` smoke test.
- Node-side gRPC transport for non-iframe TypeScript binaries.
- Playwright-style full UI E2E (Tier 3).
