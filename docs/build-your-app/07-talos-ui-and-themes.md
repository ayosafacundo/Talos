# 07 — Talos UI, tokens, components, and themes

Talos ships a design-token layer plus optional utility classes and an evolving component layer. The host Launchpad loads host theme assets, while iframe apps should include Talos UI assets in their own bundle because iframes do not inherit parent stylesheets.

## Layers

| Layer | Purpose |
|--------|---------|
| **tokens** (`tokens.css`) | Canonical CSS variables: `--talos-color-*`, `--talos-radius-*`, spacing, shadows. |
| **legacy-alias** (`legacy-alias.css`) | Maps older Launchpad names (`--bg-primary`, …) to Talos tokens for existing host CSS. |
| **utilities** (`utilities.css`) | Composable classes: `talos-bg-primary`, `talos-text-muted`, `talos-rounded-md`, … |
| **components** (`components.css` + web components) | Stable component primitives (`talos-card`, `talos-button`, …) designed for theme variants. |
| **Presets** (optional linked theme) | Small files under Launchpad `public/themes/*.css` that override **only** `--talos-*` values (e.g. `minecraft.css`, `dark.css`). |
| **Variant manifests** (`public/theme-assets/*.json`) | Maps selected host theme to component variant assets for Tiny Apps. |

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
- A **variant theme** maps host theme selection to component-level visual variants (buttons/cards/inputs/panels). This is the asset-driven path for shared app styling.

See [ASSET_DRIVEN_THEMES.md](../ASSET_DRIVEN_THEMES.md).
Component catalog roadmap: [ASSET_DRIVEN_THEMES.md#ready-to-use-component-plan](../ASSET_DRIVEN_THEMES.md#ready-to-use-component-plan)

## Host theme changes and iframes

Changing the host theme updates Launchpad and can also notify Tiny Apps through runtime theme events. Apps using Talos component/runtime helpers can apply updates without full page reload.

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
