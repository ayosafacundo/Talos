# Build Your App for Talos

This documentation explains how to build a Tiny App that runs inside Talos, from folder layout to SDK calls and debugging.

## Read Order

1. `docs/build-your-app/01-how-talos-loads-apps.md`
2. `docs/build-your-app/02-package-layout-and-manifest.md`
3. `docs/build-your-app/03-build-a-web-app.md`
4. `docs/build-your-app/04-add-a-go-sidecar-binary.md`
5. `docs/build-your-app/05-sdk-and-host-bridge.md`
6. `docs/build-your-app/06-dev-loop-and-troubleshooting.md`
7. `docs/build-your-app/07-talos-ui-and-themes.md`

## What You Build

A Talos app is a package under `Packages/<Your App>/` with:

- a `manifest.yaml`
- a web entry file (usually `dist/index.html`)
- optional binary (`bin/...`) if you need a long-running process
- optional persistent storage under `data/`

Talos discovers packages at runtime and exposes them in Launchpad.

