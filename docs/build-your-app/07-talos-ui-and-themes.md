# 07 — Talos UI, tokens, and themes

Talos ships a small **design-token** layer plus optional **utility classes** (Tailwind-like ergonomics) so tiny apps can style consistently with the host. The host Launchpad loads tokens and utilities from its bundle; **iframe apps must ship the same CSS files** in their own `dist/` because browser iframes do not inherit the parent document’s stylesheets.

## Layers

| Layer | Purpose |
|--------|---------|
| **tokens** (`tokens.css`) | Canonical CSS variables: `--talos-color-*`, `--talos-radius-*`, spacing, shadows. |
| **legacy-alias** (`legacy-alias.css`) | Maps older Launchpad names (`--bg-primary`, …) to Talos tokens for existing host CSS. |
| **utilities** (`utilities.css`) | Composable classes: `talos-bg-primary`, `talos-text-muted`, `talos-rounded-md`, … |
| **Presets** (optional linked theme) | Small files under Launchpad `public/themes/*.css` that override **only** `--talos-*` values (e.g. `minecraft.css`, `dark.css`). |

## Using tokens and utilities in a tiny app

1. Copy the three files from [`sdk/talos/`](../../sdk/talos/) into your package, e.g. `Packages/My App/dist/talos/`.
2. Reference them from `index.html` **before** your own CSS:

```html
<link rel="stylesheet" href="talos/tokens.css" />
<link rel="stylesheet" href="talos/legacy-alias.css" />
<link rel="stylesheet" href="talos/utilities.css" />
```

3. Use either utilities on elements:

```html
<div class="talos-bg-surface talos-text-primary talos-p-4 talos-rounded-md talos-border">
  Hello
</div>
```

…or raw tokens:

```css
.panel {
  background: var(--talos-color-bg-surface);
  color: var(--talos-color-text-primary);
  border-radius: var(--talos-radius-md);
}
```

## Presets vs full themes

- A **preset** is a short `:root { --talos-*: … }` file selected in Launchpad settings. It does not redefine layout; it only recolors tokens.
- To add a preset, place `Packages/Launchpad/public/themes/<name>.css`, rebuild Launchpad (`make frontend-build` or `npm run build` in `Packages/Launchpad`), and pick it in **Settings → Themes**.

## Host theme changes and iframes

Changing the host theme updates Launchpad only. To react in an app, listen for host `postMessage` events you define (for example via hub `Broadcast`) and swap classes or variables in your iframe document.

## Keeping `sdk/talos` in sync

After editing token or utility source under `Packages/Launchpad/src/talos/`, run:

```bash
make talos-sync-css
```

`make frontend-build` runs this automatically.

## Iframe bridge (SDK) v1

SDK `postMessage` requests must include:

- `channel`: `talos:sdk:v1`
- `bridge_token`: value from the `_talos_bt` query parameter the host adds to your iframe URL

See [05-sdk-and-host-bridge.md](05-sdk-and-host-bridge.md) for the method list. Use [`@talos/sdk`](../../sdk/ts/README.md) `IframeBridgeTransport` in TypeScript when possible.
