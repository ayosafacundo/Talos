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
