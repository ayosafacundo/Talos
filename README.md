# Talos

Talos is a local-first desktop host for modular "Tiny Apps", built with Go + Wails + Svelte.

## Development

- Backend + app shell notes: `docs/PHASE1.md`
- Command workflow and protobuf regeneration: `docs/DEVELOPMENT.md`
- Phase 2 readiness plan: `docs/PHASE2_PREP.md`

## Common Commands

```bash
make help
make proto
make verify
```

Run dev mode:

```bash
wails dev
```

Build production package:

```bash
wails build
```
