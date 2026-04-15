# 02 - Package Layout and Manifest

## Recommended Layout

Use **ASCII folder names without spaces** (e.g. `My-App` or `MyApp`). Spaces under `Packages/` can break Go tooling if any npm dependency ships `.go` files; the Talos repo runs `scripts/ensure-npm-go-modules.sh` after installs to isolate `node_modules`, but avoiding spaces is still recommended.

```text
Packages/My-App/
├── manifest.yaml
├── dist/
│   └── index.html
├── data/
│   └── .gitkeep
└── bin/
    └── my-app-binary   (optional)
```

## Minimal `manifest.yaml`

```yaml
id: app.my.app
name: My App
web_entry: dist/index.html
multi_instance: false
```

## Full Example

```yaml
id: app.my.app
name: My App
version: "0.1.0"
icon: dist/icon.png # png, gif, jpg, webp, svg are supported
binary: bin/my-app-binary
web_entry: dist/index.html
permissions:
  - fs:external
  - net:internet
multi_instance: false
```

## Field Notes

- `id`: unique package id, stable across versions.
- `name`: display name in Launchpad.
- `web_entry`: relative path to iframe entry HTML.
- `binary`: optional relative path to executable sidecar.
- `permissions`: requested capabilities (host policy still controls grants).
- `multi_instance`: whether multiple UI instances are allowed.

## Development vs production iframe (`development`)

For local work, you can point the iframe at a dev server (Vite, etc.) when **Development mode** is turned on for your app’s folder under `Packages/` in Launchpad Settings, or when you run a **non-production** Talos build with `TALOS_DEV_MODE=1` (e.g. `make dev`). **Release** binaries ignore `TALOS_DEV_MODE` and use only per-folder Settings toggles (default off = packaged `/talos-pkg/` assets).

```yaml
web_entry: dist/index.html

development:
  command: ["npm", "run", "dev"]
  url: "http://127.0.0.1:5174"
  allowed_origins:
    - "http://127.0.0.1:5174"
    - "http://localhost:5174"
```

- `development.url`: required when `development.command` is set; must be `http` or `https` on loopback (`localhost`, `127.0.0.1`, `::1`, or `127.*`). The host uses this URL as the initial iframe target and as a hint for discovery.
- `development.command`: optional argv list (first element is the binary name; no shell). If omitted, only `development.url` is used (you start the dev server yourself).
- `development.allowed_origins`: origins allowed for the iframe bridge; must include the origin of `development.url`.

**Host behavior (dev mode):** Talos injects **`TALOS_DEV_SERVER_PORT`** (from the port in `development.url`) into the dev command environment. Prefer wiring Vite (or similar) to that port and enabling **`strictPort: true`** so the real listen port cannot drift silently. If it does drift (for example “port in use, trying another one”), the host scans dev-server output and probes nearby ports on **both** `127.0.0.1` and `localhost`, then **rewrites** the discovered URL to use your manifest’s loopback hostname with the **actual** listen port. Bridge allowlists also gain the alternate loopback hostname for the same port when needed. The Wails event **`package:dev-url`** is emitted with `app_id` and `url` so the UI can refresh installed-app metadata.

## Validation Constraints

- Paths must be relative (not absolute).
- `web_entry` defaults to `dist/index.html` if omitted.
- `web_entry` file must exist or discovery rejects package.

