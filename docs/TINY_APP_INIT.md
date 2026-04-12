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

## 3) Create Tiny App Source

Create app source under `examples/tinyapps/my-tiny-app/main.go` and use:

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
go build -o "Packages/My Tiny App/bin/my-tiny-app" ./examples/tinyapps/my-tiny-app
chmod +x "Packages/My Tiny App/bin/my-tiny-app"
```

For the included demo app:

```bash
make tiny-demo-build
```

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

## 9) Reference Demo

Included reference tiny app:

- Source: `examples/tinyapps/go-demo/main.go`
- Package: `Packages/Tiny Go Demo`
- Build: `make tiny-demo-build`

## 10) TypeScript Demo (Iframe Bridge)

Included TypeScript demo app:

- Source: `examples/tinyapps/ts-demo/src/main.ts`
- Package: `Packages/Tiny TS Demo`
- Build: `make tiny-ts-demo-build`

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
