# Tiny App Initialization Guide

This is a full walkthrough to create and run Tiny Apps with Talos (Go binary app and TypeScript iframe app).

## 1) Create Package Folder Layout

Inside `Packages/`, create your app folder:

```text
Packages/My Tiny App/
├── manifest.yaml
├── bin/
│   └── my-tiny-app           # built executable
├── data/                     # app-scoped persistent files
└── dist/
    └── index.html            # iframe content
```

## 2) Write `manifest.yaml`

Minimum required fields:

```yaml
id: app.my.tiny
name: My Tiny App
binary: bin/my-tiny-app
multi_instance: false
```

Recommended full example:

```yaml
id: app.my.tiny
name: My Tiny App
version: "0.1.0"
icon: dist/icon.gif
binary: bin/my-tiny-app
web_entry: dist/index.html
permissions:
  - fs:external
  - net:internet
multi_instance: false
```

### Optional: dev-server iframe (developer mode only)

When iterating on the web UI, you can load a local dev server instead of the built `dist/` tree. Turn on **Development mode** for your package folder in Launchpad Settings, or run a source build with `TALOS_DEV_MODE=1` (for example `make dev`). Installed release binaries ignore `TALOS_DEV_MODE` and use per-folder Settings only.

```yaml
web_entry: dist/index.html
development:
  url: "http://127.0.0.1:5174"
  allowed_origins:
    - "http://127.0.0.1:5174"
```

When Talos runs `development.command`, it sets **`TALOS_DEV_SERVER_PORT`** from the port in `development.url` (for example `5174`). Point your bundler at it so the listen port matches the manifest, for example in `vite.config.ts`:

```ts
export default defineConfig({
  server: {
    port: Number(process.env.TALOS_DEV_SERVER_PORT) || 5174,
    strictPort: true,
  },
});
```

Using **`strictPort: true`** makes Vite exit if the port is taken instead of silently picking another port, which would disagree with `development.url`. If Vite still prints a different port (for example “Port 5173 is in use, trying another one…”), the host **parses dev-server logs** and **probes nearby HTTP ports** on both `127.0.0.1` and `localhost`, then **rewrites the discovered URL** to match your manifest hostname while keeping the real listen port. Bridge `allowed_origins` are expanded to include both loopback hostnames for the same port when needed.

#### Talos shell URL vs your app’s Vite URL

When you run the host with `wails dev` or `make dev`, the console often says something like: navigate to **`http://localhost:34115`** (port varies) to use **Wails bindings** (`window.go.main.App`, …). That URL is the **Talos desktop shell** (Launchpad), not your Tiny App’s dev server.

Your package’s Vite (for example `http://localhost:5174/`) is meant to load **inside Talos** in the app iframe. Use the **Talos window** at the Wails URL for full host + hub + iframe behavior. Opening only the raw Vite URL in a browser will not expose the host Go API.

Benign noise from the embedded webview (for example `NeedDebuggerBreak` / VM trap lines in the terminal) can usually be ignored while debugging.

See `docs/build-your-app/02-package-layout-and-manifest.md` for `development.command` and validation rules.

### Optional: Talos asset-driven UI components

To reduce per-app styling effort, use Talos shared component assets:

- Theme contract: `docs/ASSET_DRIVEN_THEMES.md`
- Host/tokens/components guidance: `docs/build-your-app/07-talos-ui-and-themes.md`
- Web components package: `sdk/web-components/`
- Scaffold example: `Packages/Example TS App/`

## 3) Create Tiny App Source

Create app source under your package (for example `Packages/My Tiny App/src/main.go`) and use:

- `TALOS_APP_ID`
- `TALOS_HUB_SOCKET`
- `sdk/go/talos`

Core boot sequence:

1. Dial hub.
2. Load previous state.
3. Request optional permissions.
4. Do scoped IO via `ResolvePath` or helper methods.
5. Save state periodically and on shutdown.

## 4) Build Binary Into Package

Example command:

```bash
go build -o "Packages/My Tiny App/bin/my-tiny-app" ./Packages/My\ Tiny\ App/src
chmod +x "Packages/My Tiny App/bin/my-tiny-app"
```

For the included example apps, build from repo root with your own commands (see `docs/DEVELOPMENT.md` — e.g. `go build` / `cargo` / `npm` under each package).

## 5) Run Talos and Start App

1. Start Talos:

```bash
wails dev
```

2. In Talos UI, verify your package appears under discovered packages.
3. Click **Start**.
4. Observe logs from the tiny app in terminal output.

## 6) Handle Permission Requests

When your tiny app calls `RequestPermission`, Talos emits a host event and surfaces request actions in UI:

- **Allow** -> grant persisted
- **Deny** -> deny persisted

Persisted grants are stored in:

- `Temp/permissions.json`

## 7) Validate SDK Features

Use this checklist:

- [ ] `LoadState` returns prior state after restart.
- [ ] `SaveState` succeeds during runtime and on shutdown.
- [ ] `WriteScopedFile` and `ReadScopedFile` work under `data/`.
- [ ] Escaping `data/` is denied unless `fs:external` is granted.
- [ ] `SendMessage` receives expected response.
- [ ] `Broadcast` returns recipient count.

## 8) Common Troubleshooting

- **App not discovered**
  - Check `manifest.yaml` field names exactly.
  - Ensure `binary` is relative and points to an executable file.
- **Binary fails to start**
  - Confirm executable permission (`chmod +x`).
  - Confirm architecture/OS matches host.
- **Hub connection fails**
  - Ensure app reads `TALOS_HUB_SOCKET`.
  - Ensure Talos host is running.
- **Permission stays denied**
  - Re-trigger request from tiny app after allowing in host.
  - Check `Temp/permissions.json` update.
- **Scoped file error**
  - Use SDK helpers (they call host `ResolvePath`).
  - Verify relative path does not attempt directory escape.

## 9) Reference Example Apps

Included reference Go app:

- Source: `Packages/Example Go App/src/main.go`
- Package: `Packages/Example Go App`
- Build: `go build` from `Packages/Example Go App/src` into `bin/` (see `docs/DEVELOPMENT.md`)

Included reference Rust app:

- Source: `Packages/Example Rust App/src/main.rs`
- Package: `Packages/Example Rust App`
- Build: `cargo build --release` (see `docs/DEVELOPMENT.md`)

## 10) TypeScript Example (Iframe Bridge)

Included TypeScript example app:

- Source: `Packages/Example TS App/src/App.tsx`
- Package: `Packages/Example TS App`
- Build: `npm --prefix "Packages/Example TS App" run build`

How it works:

1. App runs inside iframe via `dist/index.html`.
2. TS sends `talos:sdk:req` messages to parent window.
3. Host resolves request through bound Go APIs.
4. Host returns `talos:sdk:res`.

This demo validates:

- iframe-host bridge flow
- state save/load via host
- permission request path
- route/broadcast call path
