# Talos TypeScript SDK (Baseline)

This is the Phase 2 baseline wrapper for Tiny Apps written in TypeScript.

It intentionally uses a transport interface so the app can plug in:

- grpc-web transport
- Node gRPC transport
- host bridge transport

## Current API

- `saveState(data)`
- `loadState()`
- `sendMessage(targetID, payload)`
- `requestPermission(scope, reason?)`
