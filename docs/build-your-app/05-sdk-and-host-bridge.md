# 05 - SDK and Host Bridge

## Two Integration Paths

### A) Binary SDK (Go, later TS/Rust parity)

Use gRPC through Talos hub:

- route/broadcast messaging
- state save/load
- permission requests
- scoped path resolution

Main Go SDK path: `sdk/go/talos`.

### B) Iframe JS Bridge

For frontend-only apps, use `postMessage` with:

- request type: `talos:sdk:req`
- response type: `talos:sdk:res`

Methods supported by Launchpad host layer include:

- `saveState`
- `loadState`
- `requestPermission`
- `resolvePath`
- `sendMessage`
- `broadcast`
- `getInstalledApps`
- `getStoreApps`
- `launchApp`
- `setContextMenuOptions`
- `clearContextMenuOptions`
- `openContextMenu`

## Bridge Envelope (Example)

```js
window.parent.postMessage(
  {
    type: "talos:sdk:req",
    request_id: crypto.randomUUID(),
    app_id: "app.my.app",
    method: "loadState",
    params: {}
  },
  "*"
);
```

Listen for `talos:sdk:res` in your app to receive results.

## App Context Menu Integration

Apps can inject custom context actions while they are active in Launchpad.

### Register options

```js
window.parent.postMessage(
  {
    type: "talos:sdk:req",
    request_id: crypto.randomUUID(),
    app_id: "app.my.app",
    method: "setContextMenuOptions",
    params: {
      options: [
        { id: "refresh", label: "Refresh Data" },
        { id: "export", label: "Export Report" }
      ]
    }
  },
  "*"
);
```

### Handle selected action

Launchpad posts selected custom actions back to the app iframe:

- event type: `talos:context:action`
- payload fields: `app_id`, `action_id`

```js
window.addEventListener("message", (event) => {
  const msg = event.data;
  if (!msg || msg.type !== "talos:context:action") return;
  if (msg.action_id === "refresh") {
    // run app-specific action
  }
});
```

### Open menu from inside the app

Your app can request Launchpad to open the context menu at cursor coordinates:

```js
window.parent.postMessage(
  {
    type: "talos:sdk:req",
    request_id: crypto.randomUUID(),
    app_id: "app.my.app",
    method: "openContextMenu",
    params: { x: 320, y: 240 }
  },
  "*"
);
```

## Filesystem Safety Model

- Default allowed scope: your package `data/` directory.
- Outside that scope requires permission (`fs:external`).
- Use host path resolution APIs before direct file IO.

