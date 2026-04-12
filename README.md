# Talos

Talos is a local-first desktop host for modular "Tiny Apps", built with Go + Wails + Svelte.

## Development

- Backend + app shell notes: `docs/PHASE1.md`
- Command workflow and protobuf regeneration: `docs/DEVELOPMENT.md`
- Full development manual: `docs/DEVELOPMENT_FULL.md`
- Release-style status tracker: `docs/STATUS.md`
- Phase 2 readiness plan: `docs/PHASE2_PREP.md`
- SDK guide: `docs/SDK_GUIDE.md`
- Tiny app init guide: `docs/TINY_APP_INIT.md`

## Common Commands

```bash
make help
make proto
make verify
make tiny-demo-build
make tiny-ts-demo-build
make app-build
```

Run dev mode:

```bash
make dev
```

Build production package:

```bash
make app-build
```
