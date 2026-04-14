# Talos Architecture

Talos is a local-first desktop host that discovers package manifests, launches package binaries, and renders package web UIs in iframes.

## Core components

- Host shell: Wails app lifecycle and frontend bindings (`main.go`, `app.go`).
- Package subsystem: discovery, manifest validation, and package install paths (`internal/packages`, `internal/packageinstall`).
- Process manager: package binary and dev-server process lifecycle (`internal/process`).
- Hub IPC: gRPC over local sockets/pipes for app-to-host/app-to-app RPC (`internal/hub`).
- Launchpad frontend: package UX, bridge mediation, permissions, and settings (`Packages/Launchpad`).

## Startup flow

1. Host starts and initializes directories (`Packages`, `Temp`, logs, trust stores).
2. Package discovery scans manifests and emits package events.
3. Launchpad loads installed apps and renders package entries.
4. User action starts package binary (if configured) and loads iframe `web_entry` (or dev URL in dev mode).
5. Tiny app bridge calls are routed by Launchpad to host APIs.

## IPC model

- Tiny binaries connect to the host hub via Unix domain socket (Linux/macOS) or named pipes (Windows).
- Hub supports route/broadcast messaging, state RPCs, permission requests, and scoped path resolution.
- TypeScript iframe apps use `postMessage` bridge v1 envelope; Launchpad validates sender + token + origin before host dispatch.

## Trust and updates pipeline

- Install/update writes a hash manifest (`Temp/package_hashes/<app_id>.json`).
- Optional detached Ed25519 signature (`.talos-signature`) is verified against trusted publisher keys (`Temp/trusted_keys/*.pub`).
- Trust statuses (`ok`, `unsigned`, `signed_ok`, `signed_invalid`, `tampered`) are surfaced in Launchpad.
- Update and repository feeds are fetched from operator-provided HTTPS JSON endpoints.

## Isolation model

- Package frontend runs in iframe sandbox.
- Host permissions gate sensitive scopes (`fs:external`, `net:internet`, etc.).
- Default file operations are scoped to package `data/` unless an explicit grant exists.
- Bridge calls are constrained by allowlisted methods, per-instance bridge tokens, and `allowed_origins`.

See `docs/dev/IFRAME_THREAT_MODEL.md` for Phase 4 threat details.
