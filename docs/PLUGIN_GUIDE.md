# Plugin Guide

This guide shows how to build a Talos-ready package (binary, iframe, or both).

## 1. Create package layout

Use a package folder under `Packages/`:

```text
Packages/My-App/
├── manifest.yaml
├── dist/index.html
├── data/
└── bin/my-app-binary   # optional
```

## 2. Write `manifest.yaml`

Minimal:

```yaml
id: app.my.app
name: My App
web_entry: dist/index.html
multi_instance: false
```

Common fields:

- `id`, `name`, `version`, `icon`
- `web_entry` (iframe entry)
- `binary` (optional sidecar process)
- `permissions` (requested capabilities)
- `development` (dev-only iframe URL/command)

Full field semantics are in `docs/MANIFEST_SPEC.md`.

## 3. Implement app logic with SDK

### Go tiny app

- Use `sdk/go/talos`.
- Read environment values (`TALOS_APP_ID`, `TALOS_HUB_SOCKET`).
- Use hub RPCs for state, messages, permissions, and scoped paths.

### TypeScript iframe app

- Use `sdk/ts` bridge client.
- Ensure iframe bridge token propagation (`_talos_bt`) is preserved by your router.
- Handle request failures (permission denied, unsupported methods) explicitly in UI.

## 4. Development mode

For live frontend iteration, configure:

```yaml
development:
  command: ["npm", "run", "dev"]
  url: "http://127.0.0.1:5174"
  allowed_origins:
    - "http://127.0.0.1:5174"
    - "http://localhost:5174"
```

- Talos sets `TALOS_DEV_SERVER_PORT` from `development.url`.
- Prefer `strictPort: true` in Vite.
- Host can discover shifted loopback ports and emits `package:dev-url` so Launchpad refreshes.

## 5. Package trust and updates

- Install/update generates hash manifest entries under `Temp/package_hashes/`.
- Optional signatures use `.talos-signature` plus trusted publisher keys under `Temp/trusted_keys/`.
- In paranoid mode (`TALOS_PARANOID_TRUST=1`), tampered packages are blocked from launch.

## 6. Validation checklist

- Package appears in Launchpad.
- Binary starts/stops cleanly (if defined).
- `SaveState`/`LoadState` roundtrip works.
- Scoped data access works; external scope prompts when needed.
- Bridge messaging works for iframe apps.

## 7. Troubleshooting

- Discovery failures: verify manifest paths are relative and files exist.
- Bridge rejection: verify `allowed_origins`, token forwarding, and method names.
- Dev URL mismatch: confirm `development.url` loopback host/port and bundler port config.
