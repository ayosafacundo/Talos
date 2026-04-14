# Talos TypeScript SDK

Transport-based client for tiny apps. Use **`IframeBridgeTransport`** inside the Wails iframe: it reads `_talos_bt` from the page URL and sends `talos:sdk:v1` envelopes to the host.

```ts
import { TalosClient, IframeBridgeTransport } from "@talos/sdk";

const client = new TalosClient("app.my.id", new IframeBridgeTransport("app.my.id"));
await client.requestPermission("fs:external", "reason");
```

See [`docs/build-your-app/05-sdk-and-host-bridge.md`](../../docs/build-your-app/05-sdk-and-host-bridge.md) and [`docs/build-your-app/07-talos-ui-and-themes.md`](../../docs/build-your-app/07-talos-ui-and-themes.md).

## Theme runtime helper

Use `bindTalosThemeRuntime` to react to host-driven theme updates in iframe apps:

```ts
import { bindTalosThemeRuntime } from "@talos/sdk"

const disposeTheme = bindTalosThemeRuntime()
```

This listens for `talos:theme:v1` / `talos:theme:update` messages and updates root theme markers (`data-talos-theme`, `data-talos-theme-variant`). If `components_css_href` is provided, it keeps a linked variant stylesheet in sync.

## API (`TalosClient`)

- `saveState` / `loadState`
- `sendMessage`
- `requestPermission`
- `resolvePath`
- `setContextMenuOptions` / `clearContextMenuOptions` / `openContextMenu` (when transport supports)

Run `npx tsc -p tsconfig.json` in this folder to typecheck (requires `devDependencies.typescript`).
