# Talos TypeScript SDK

Transport-based client for tiny apps. Use **`IframeBridgeTransport`** inside the Wails iframe: it reads `_talos_bt` from the page URL and sends `talos:sdk:v1` envelopes to the host.

```ts
import { TalosClient, IframeBridgeTransport } from "@talos/sdk";

const client = new TalosClient("app.my.id", new IframeBridgeTransport("app.my.id"));
await client.requestPermission("fs:external", "reason");
```

See [`docs/build-your-app/05-sdk-and-host-bridge.md`](../../docs/build-your-app/05-sdk-and-host-bridge.md) and [`docs/build-your-app/07-talos-ui-and-themes.md`](../../docs/build-your-app/07-talos-ui-and-themes.md).

## API (`TalosClient`)

- `saveState` / `loadState`
- `sendMessage`
- `requestPermission`
- `resolvePath`
- `setContextMenuOptions` / `clearContextMenuOptions` / `openContextMenu` (when transport supports)

Run `npx tsc -p tsconfig.json` in this folder to typecheck (requires `devDependencies.typescript`).
