# Phase 6 Release Polish Checklist

Use this checklist to close final release-candidate risk after Phases 3-5 deliverables are merged.

## Reliability

- [ ] Catalog and update fetch retries validated under transient network faults.
- [ ] Timeouts and error copy are actionable for offline/HTTP/malformed feed failures.
- [ ] Process shutdown leaves no expected orphan dev server or package child trees.

## UX consistency

- [ ] Launchpad trust badges and error banners use consistent wording.
- [ ] About/Settings update and repository guidance match `docs/PHASE3.md`.
- [ ] Permission and trust failure flows are understandable without reading logs.

## Performance

- [ ] Startup package discovery remains responsive on representative package counts.
- [ ] Update checks complete within acceptable latency on healthy network.
- [ ] UI app-switch flow remains smooth with multiple active iframes.

## Verification

- [ ] `make verify` green.
- [ ] `bash scripts/run_integration_hub.sh` green.
- [ ] Targeted tests green:
  - `go test ./internal/updates ./internal/packages/repository ./internal/process ./internal/packageinstall`
- [ ] Manual critical-path validation completed for Linux/macOS/Windows owners.

## Release candidate gate

- [ ] Two consecutive CI runs with no required job failures.
- [ ] Deferred-items list captured in release notes.
- [ ] Rollback notes documented for trust/update regressions.
