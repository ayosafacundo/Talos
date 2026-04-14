# Phase 4 — developer experience, trust polish, and isolation

Phase 3 (distribution and trust) is summarized in [STATUS.md](STATUS.md) and [PHASE3.md](PHASE3.md). Phase 4 groups post–Phase 3 work into epics with exit criteria.

## Goals

1. **Reliable local development** — Dev servers started via `development.command` match the iframe: `TALOS_DEV_SERVER_PORT`, log-based URL discovery, HTTP probing on both `127.0.0.1` and `localhost`, **canonical loopback alignment** with `development.url`, **bridge origin expansion** for localhost ↔ 127.0.0.1, process-group teardown, and `package:dev-url` for the Launchpad shell.
2. **Trust and distribution polish** — Smoother catalog/update error handling, optional paranoid trust adoption, clearer publisher workflows.
3. **Security and isolation (longer horizon)** — Stronger iframe and capability boundaries; explore OS-level sandboxing as a separate track (see [PHASE2_PREP.md](PHASE2_PREP.md)).
4. **Testing depth** — Optional headless `wails dev` smoke and UI E2E beyond hub integration tests ([STATUS.md](STATUS.md) Phase 2 follow-ups).

## Non-goals (for Phase 4 baseline)

- Replacing the Wails desktop shell with a different host.
- Full remote registry product (HTTP catalog exists; deep multi-tenant registry is out of scope unless explicitly prioritized).

## Epics and backlog

| Area | Deliverable |
|------|-------------|
| DevEx | Document Wails shell URL vs Tiny App Vite; env + strict port; discovery + alignment ([TINY_APP_INIT.md](TINY_APP_INIT.md)) |
| DevEx | Launchpad reacts to `package:dev-url` (implemented in host + shell) |
| Trust | Catalog/update retries, offline messaging, actionable errors |
| Trust | Publisher docs for signing + catalog entries |
| Isolation | Threat model note for iframe bridge; sandbox spike design doc |
| QA | Optional Playwright or headless Wails smoke |
| Observability | Host terminal should print structured app-load failure reasons (network/connectivity, iframe load error, bridge/origin rejection, manifest/dev-url mismatch) with full breakdown for debugging |

## Completed in tree (reference)

- Process-group / `taskkill /T` teardown for dev and app binaries (`internal/process/kill_tree_*.go`).
- Dev URL discovery (`internal/process/devurl.go`), `AlignDiscoveredDevURL`, `ExpandLoopbackOrigins`, dual-host HTTP probe.
- Host state `devResolvedURL` + `package:dev-url` event (`app.go`).

## Exit criteria (release-oriented)

- **DevEx:** Documented `development` workflow; loopback hostname alignment; no orphan `npm`/`node` trees after host shutdown on supported platforms under normal exit.
- **Trust polish:** Catalog/update failures are actionable in the UI or logs; trust states remain understandable with [PHASE3.md](PHASE3.md) semantics.
- **Isolation:** Documented threat model updates for iframe bridge; sandbox epic backlog with MVP vs stretch explicitly split.
- **Verification:** `make verify` remains green; tests cover dev URL parsing and loopback helpers.

## References

- Dev manifest rules: [build-your-app/02-package-layout-and-manifest.md](build-your-app/02-package-layout-and-manifest.md)
- Tiny app bootstrap: [TINY_APP_INIT.md](TINY_APP_INIT.md)
- Startup flow: [dev/STARTUP_TO_LAUNCHPAD.md](dev/STARTUP_TO_LAUNCHPAD.md)
- Iframe threat model: [dev/IFRAME_THREAT_MODEL.md](dev/IFRAME_THREAT_MODEL.md)
