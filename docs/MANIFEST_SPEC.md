# Manifest Specification

Talos package manifests are YAML files at `Packages/<PackageName>/manifest.yaml`.

## Required fields

- `id` (string): globally unique package id (`app.vendor.name` style recommended).
- `name` (string): display name in Launchpad.

## Optional fields

- `version` (string): semantic-ish version for display and update workflows.
- `description` (string): Launchpad description text.
- `icon` (string): relative path to icon asset (png/gif/jpg/webp/svg).
- `web_entry` (string): relative path to iframe HTML entry. Defaults to `dist/index.html` when omitted.
- `binary` (string): relative executable path for sidecar process.
- `store_url` (string): external store/reference URL.
- `permissions` (array[string]): requested capabilities (host policies still gate grant/deny).
- `multi_instance` (bool): allow multiple UI instances.

## `development` block (optional; per-folder Development mode in Settings, or `TALOS_DEV_MODE` in source builds)

```yaml
development:
  command: ["npm", "run", "dev"]
  url: "http://127.0.0.1:5174"
  allowed_origins:
    - "http://127.0.0.1:5174"
    - "http://localhost:5174"
```

- `command` (array[string], optional): executable argv list used to start dev server.
- `url` (string, required when `command` is set): initial iframe URL hint.
- `allowed_origins` (array[string], recommended): allowed postMessage origins for bridge.

### Development constraints

- `development.url` must be `http`/`https` on loopback (`localhost`, `127.0.0.1`, `::1`, or `127.*`).
- `development.url` is optional when `development.command` is present; Talos can discover the runtime URL from logs/probing.
- The host honors `development` when **Development mode** is enabled in Launchpad Settings **for that package’s folder** under `Packages/`, or when running a **non-production** build with **`TALOS_DEV_MODE=1`** (e.g. `make dev`). Otherwise only packaged `web_entry` URLs (`/talos-pkg/...`) are used. **Release** binaries (`wails build -tags=production`) do not read `TALOS_DEV_MODE`.
- Host exports `TALOS_DEV_SERVER_PORT` to `development.command`.

## Host environment for package `binary` processes

When a package declares `binary`, the host starts it with environment including:

- `TALOS_APP_ID`, `TALOS_APP_DATA_DIR`, `TALOS_HUB_SOCKET`
- `TALOS_PACKAGE_DEVELOPMENT` — `1` when Launchpad **package development** is enabled for this app’s folder (Settings toggle, or source `TALOS_DEV_MODE=1` on non-production builds); `0` otherwise. Sidecars can surface this as “development” vs “production” packaging.
- `TALOS_SQL_DSN` — SQLite connection string for this app’s isolated database file under the Talos data directory (per-app file; host-provisioned).

## Path and validation rules

- All manifest paths (`web_entry`, `binary`, `icon`) must be relative.
- Absolute paths are rejected.
- `web_entry` must resolve to an existing file for package discovery success.
- If `binary` is set, it must resolve to a file (not directory) and be executable for runtime use.

## Trust metadata

Trust state is not authored in manifest; it is computed at install/runtime from hash/signature verification and surfaced to the UI.

## Theming metadata (Phase 1 behavior)

Asset-driven theme selection is currently host-driven and does not require manifest fields. Tiny Apps receive theme updates from host runtime events and should use Talos component/token assets.

## Example full manifest

```yaml
id: app.my.app
name: My App
version: "1.0.0"
description: Example Talos package
icon: dist/icon.png
web_entry: dist/index.html
binary: bin/my-app-binary
permissions:
  - fs:external
  - net:internet
multi_instance: false
development:
  command: ["npm", "run", "dev"]
  url: "http://127.0.0.1:5174"
  allowed_origins:
    - "http://127.0.0.1:5174"
    - "http://localhost:5174"
```
