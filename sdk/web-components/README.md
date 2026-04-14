# Talos Web Components

Framework-agnostic UI primitives for Tiny Apps.

## What this package provides

- `talos-card`
- `talos-panel`
- `talos-button`
- `talos-input`
- `talos-alert`
- `talos-list-row`

## Quick start

1. Install/link SDK packages in your app:

```bash
npm install @talos/sdk @talos/web-components
```

2. Include Talos styles using package aliases:

```css
@import "@talos/sdk/talos/tokens.css";
@import "@talos/sdk/talos/legacy-alias.css";
@import "@talos/sdk/talos/utilities.css";
@import "@talos/web-components/components.css";
```

3. Register components:

```html
<script type="module">
  import { registerTalosWebComponents } from "@talos/web-components";
  registerTalosWebComponents();
</script>
```

4. Use components:

```html
<talos-panel>
  <talos-alert>
    <span slot="title">Policy Notice</span>
    Theme variant is controlled by Talos host settings.
  </talos-alert>
  <talos-card>
    <talos-button>Approve</talos-button>
    <talos-button variant="ghost">Cancel</talos-button>
  </talos-card>
</talos-panel>
```

## Theming cookbook

### 1) Token-only theme override

Use token overrides for fast recolor:

```css
:root {
  --talos-color-accent: #5a8dff;
  --talos-color-bg-surface: #151a22;
  --talos-color-text-primary: #eef2ff;
}
```

### 2) Component-level customization via CSS variables

```css
:root {
  --talos-component-shadow: 0 6px 20px rgba(0, 0, 0, 0.28);
  --talos-component-success: #2ca66f;
  --talos-component-danger: #d55050;
}
```

### 3) Fine-grained styling with `::part`

```css
talos-card::part(base) {
  border-style: dashed;
}

talos-alert::part(title) {
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

talos-list-row::part(row):hover {
  transform: translateY(-1px);
}
```

### 4) Attribute-driven variants

```html
<talos-button variant="ghost" size="sm">Cancel</talos-button>
<talos-button tone="danger">Delete</talos-button>
<talos-alert tone="success">
  <span slot="title">Saved</span>
  Changes were persisted.
</talos-alert>
```

### 5) React wrapper usage

```tsx
import { TalosCard, TalosButton } from "@talos/web-components/react";

<TalosCard>
  <TalosButton tone="success">Approve</TalosButton>
</TalosCard>
```

## Notes

- These primitives are intentionally simple for Phase 1.
- Theme variants are applied through shared `--talos-*` tokens and host theme sync events.
