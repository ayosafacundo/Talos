# Integration test tiers

## Tier 1 (automated, CI)

- **Go hub:** `bash scripts/run_integration_hub.sh` → [`internal/hub/integration_grpc_test.go`](../../internal/hub/integration_grpc_test.go) (`-tags=integration`).
- Covers multi-app routing, broadcast, state isolation, permission hooks, resolve-path denial, and SDK dial over a Unix socket.

## Tier 2 (optional / manual)

- Run the Talos binary with a fixed `Packages/` fixture (two manifests + dist stubs) and scripted UI actions. Suitable for nightly or release QA; not required for every PR.

## Tier 3 (optional / high cost)

- Browser-driven E2E (e.g. Playwright) against `wails dev` or a packaged build. Deferred until Tier 1 gaps are closed.
