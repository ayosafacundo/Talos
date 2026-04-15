# Iframe bridge (v1) — behavior matrix

The Launchpad host and [`sdk/ts/src/iframe-bridge.ts`](../../sdk/ts/src/iframe-bridge.ts) cooperate using `postMessage` with channel `talos:sdk:v1`. Shared helpers live in [`Packages/Launchpad/src/bridge.ts`](../../Packages/Launchpad/src/bridge.ts) (unit tests: `bridge.test.ts`).

## Identity and trust

- Each iframe instance receives a random `_talos_bt` query parameter (bridge token). Requests must include `bridge_token` matching the iframe’s `data-talos-bridge-token`.
- `app_id` in the envelope must match `data-talos-manifest-id` on the same iframe.
- The host resolves the sender `Window` against registered iframes (`resolveTrustedSender`). Unmatched senders are rejected.

## Origins (`file://` vs `http(s)`)

| Iframe URL | `allowed_origins` in manifest | Incoming `postMessage` | Host reply `targetOrigin` | Host → iframe posts |
|------------|--------------------------------|-------------------------|---------------------------|---------------------|
| `file://` | ignored | `event.origin` may be `"null"`; token + app id still required | Prefer concrete origin when possible; else `*` per browser rules | `file:` origin or `*` |
| `http(s)://` loopback | non-empty list | Must match `allowed_origins` after `normalizeWebOrigin` | Same origin as sender when allowlisted | First matching allowlist origin or iframe `src` origin |

When `allowed_origins` is **empty**, the host uses legacy behavior: replies use `event.origin` when set, otherwise `*`.

## SDK alignment

`IframeBridgeTransport` uses `parentPostMessageTarget()` for requests to the parent. The host’s reply target must remain compatible with the origins the iframe page can receive (see Vitest cases in `bridge.test.ts`).

## `parentPostMessageTarget` (iframe → shell)

`postMessage` from a dev iframe (e.g. `http://127.0.0.1:5175`) to the Wails shell must use the **shell’s origin** (e.g. `wails://wails.localhost:34115`), not the iframe origin.

Launchpad appends **`_talos_shell_origin=<encodeURIComponent(shellOrigin)>`** on every iframe URL in `withBridgeToken`; [`parentPostMessageTarget`](../../sdk/ts/src/iframe-bridge.ts) **reads that first**, so WebKit does not need a correct `ancestorOrigins` list. If the param is missing, the SDK falls back to scanning `ancestorOrigins` (first entry whose origin ≠ iframe) or `"*"`.

## SDK logging (`packageSdkLog`)

Iframe apps can append diagnostic lines through **`packageSdkLog`** with params `level` and `message`. The host writes to **`Temp/logs/packages/sdk/<app_id>.log`** only when **Development mode** is enabled for that package (same policy as manifest `development.*`). In production with dev mode off, the host accepts the request but performs no file write.

## Package loopback HTTP (`packageLocalHttp`)

Mini-apps with a `binary` sidecar often serve a local HTTP API on `127.0.0.1`. **Embedded WebViews must not connect to that URL directly** (connection refused, policy, and timing). Instead the iframe calls bridge method **`packageLocalHttp`**, which runs [`PackageLocalHTTP`](../../package_local_http.go) in the host: it resolves the sidecar port (see lifecycle below), forwards **GET/POST** only to paths under **`/api/`**, and returns status + body to the iframe.

### `api-port.txt` lifecycle (staleness and recovery)

- The sidecar writes the decimal TCP port to **`Packages/<DirName>/data/api-port.txt`** immediately after a successful `Listen` on `127.0.0.1`. It may use an ephemeral port (`:0`) or a fixed port when **`TALOS_API_PORT`** is set (for aligning bare Vite dev with a matching `VITE_*` port in the web bundle).
- The host keeps an **in-memory cache** of the last successful port per `app_id` (invalidated on dial failure or when the package stops).
- On each proxy call, the host **re-reads** `api-port.txt` when needed and **waits** (bounded backoff, on the order of several seconds) for the file to appear after process start, and **retries** dial failures while the sidecar process is still managed.
- When **`StopPackage`** runs, the host best-effort **removes** `api-port.txt` so the file does not outlive the child process.
- If a dial fails and the host no longer considers the sidecar running, it removes the stale file and returns a stable error **`package sidecar not running`** (distinct from **`package sidecar not ready`** when the file is missing or empty).

Structured host logs for failures use source `package-local-http` with `app_id`, `attempt`, `port_source` (`cache` vs `file`), and `err_class` (`dial` vs `non_dial`).

## `127.0.0.1` connection refused (dev)

Common causes:

- **Vite (or another dev server) not listening** on the URL in the manifest—e.g. **port already in use**, wrong port, or process not started.
- **Vite HMR / `@vite/client`** opening a WebSocket to loopback; mitigate with `server.hmr: false` in the mini-app’s Vite config when embedded.
- **Theme `<link>` URLs** pointing at the wrong loopback host/port (`127.0.0.1` vs `localhost`). When the host injects **`_talos_shell_origin`**, [`theme-runtime`](../../sdk/ts/src/theme-runtime.ts) **normalizes** http(s) loopback theme, tokens, and components hrefs to match that shell origin so stylesheet loads stay consistent.

Enable stylesheet diagnostics: `localStorage.setItem("talos_debug_bridge", "1")` (see [`bridge-debug`](../../sdk/ts/src/bridge-debug.ts)); optional `onStylesheetError` on [`bindTalosThemeRuntime`](../../sdk/ts/src/theme-runtime.ts) / `applyTalosThemeRuntime`.
