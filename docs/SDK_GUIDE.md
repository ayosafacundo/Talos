# Talos SDK Guide

This guide explains how Tiny Apps talk to the Talos host through the SDK and gRPC hub.

## Architecture

Tiny Apps never talk to each other directly. All calls flow through the host hub:

1. Tiny App SDK client connects to `TALOS_HUB_SOCKET`.
2. SDK calls gRPC methods on `HubService`.
3. Host validates permissions and filesystem scope.
4. Host routes or persists data, then returns responses.

## SDK Locations

- Go SDK: `sdk/go/talos`
- TypeScript baseline SDK: `sdk/ts`
- Rust baseline SDK: `sdk/rust`

Go SDK is the most complete implementation right now.

TypeScript runtime demo is available at:

- Package: `Packages/Tiny TS Demo`
- Source: `examples/tinyapps/ts-demo/src/main.ts`

## Required Runtime Environment

Talos injects these env vars when it starts a tiny app process:

- `TALOS_APP_ID` - app identity from `manifest.yaml`.
- `TALOS_APP_DATA_DIR` - app-scoped writable data dir.
- `TALOS_HUB_SOCKET` - local hub transport endpoint.

## gRPC Methods Exposed by Host

Proto: `api/proto/talos/hub/v1/hub.proto`

- `Route` - request/response to a specific target app id.
- `Broadcast` - fanout message to active recipients.
- `SaveState` - persist serialized app state.
- `LoadState` - load previously stored state.
- `RequestPermission` - ask host for a scope grant.
- `ResolvePath` - resolve and validate scoped filesystem paths.

## Go SDK API (Current)

`sdk/go/talos/client.go`:

- `Dial(ctx, socketURL)` - connect to local hub.
- `SendMessage(ctx, sourceAppID, targetAppID, typ, payload)`
- `Broadcast(ctx, sourceAppID, typ, payload)`
- `SaveState(ctx, appID, data)`
- `LoadState(ctx, appID) (data, found, err)`
- `RequestPermission(ctx, appID, scope, reason)`
- `ResolvePath(ctx, appID, relativePath)`
- `WriteScopedFile(ctx, appID, relativePath, data)`
- `ReadScopedFile(ctx, appID, relativePath)`

## Filesystem Scoping Rules

Default allow:

- Writes/reads under `/Packages/[AppName]/data/...`.

Outside data scope:

- denied unless `fs:external` is granted.

Always use `ResolvePath` (or SDK helpers that call it) before direct filesystem access.

## Permission Flow

1. Tiny App calls `RequestPermission(scope, reason)`.
2. Host emits `permissions:request` event for UI.
3. User approves or denies in Talos UI.
4. Host persists decision to `Temp/permissions.json`.
5. Future requests may return immediately based on stored grant.

## State Flow

1. Tiny App serializes state bytes (JSON recommended for demo/debug).
2. App calls `SaveState`.
3. On next launch, app calls `LoadState`.
4. If `found=false`, app boots with defaults.

## Messaging Flow

Direct:

- `SendMessage` with target app id.

Broadcast:

- `Broadcast` for system-wide events (theme changes, lifecycle hints, etc.).

## Regenerating SDK/Proto Contracts

Whenever `hub.proto` changes:

```bash
make proto
make verify
```

## Recommended Tiny App Patterns

- Use short timeouts (`2s`-`5s`) per SDK call.
- Save state periodically and on process shutdown.
- Keep payloads compact and versioned.
- Treat permission denial as normal behavior.
- Use scoped file helpers for all local data IO.

## TypeScript Iframe Bridge Notes

The TypeScript demo uses a host bridge transport:

- Iframe sends `talos:sdk:req` with `channel: talos:sdk:v1` and `bridge_token` (from `_talos_bt` in the iframe URL).
- Host shell validates the sender iframe + token, then calls bound Go methods using the trusted manifest id.
- Host replies with `talos:sdk:res` (same channel).

Use `IframeBridgeTransport` from `@talos/sdk` or mirror the same envelope in plain JS. Styling: copy CSS from [`sdk/talos/`](../sdk/talos/) — see `docs/build-your-app/07-talos-ui-and-themes.md`.

Implemented bridge request methods:

- `saveState`
- `loadState`
- `requestPermission`
- `resolvePath`
- `sendMessage`
- `broadcast`
- `setContextMenuOptions`
- `clearContextMenuOptions`
- `openContextMenu`

When Launchpad users select a custom app menu item, the iframe receives:

- message type: `talos:context:action`
- payload: `{ channel, app_id, action_id, bridge_token }`
