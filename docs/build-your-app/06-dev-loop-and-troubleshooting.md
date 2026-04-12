# 06 - Dev Loop and Troubleshooting

## Recommended Development Loop

1. Build/update your package frontend into `Packages/<App>/dist`.
2. Run Talos:

```bash
make dev
```

3. Open Launchpad and start your app.
4. Watch live host/app logs in dev mode.
5. Iterate on package files and relaunch as needed.

## Useful Commands

- `make proto` - regenerate hub bindings after proto changes.
- `make verify` - run tests + build checks.
- `make launchpad-build` - rebuild Launchpad frontend package.
- `make app-build` - production app build.

## Common Problems

### App not listed in Launchpad

- confirm package under `Packages/`
- confirm `manifest.yaml` is valid
- confirm `web_entry` file exists

### App listed but fails to start

- for binary apps: ensure `binary` exists and is executable
- inspect logs in `Temp/logs/packages/<app_id>.log`

### Permission requests do not behave as expected

- check `Temp/permissions.json`
- deny/grant states persist across runs

### State not loading

- verify app uses same `app_id` consistently
- verify save/load calls complete without errors

## Pre-Release Checklist

- [ ] Manifest has stable `id`
- [ ] `dist/index.html` exists
- [ ] Optional binary is executable and correct architecture
- [ ] App handles denied permissions gracefully
- [ ] Data writes are scoped safely
- [ ] App launches and relaunches cleanly via Launchpad

