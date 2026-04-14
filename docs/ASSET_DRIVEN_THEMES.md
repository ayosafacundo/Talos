# Asset-Driven Themes

Talos asset-driven themes let app developers focus on logic while users control visual style.

## Scope

- **In scope:** component-level visual variants, host-to-iframe theme sync, and shared Talos Web Components.
- **Out of scope (Phase 1):** arbitrary per-app layout engines that fully replace app information architecture.

## Design goals

1. Developers write app logic once against stable Talos components.
2. Users can change visual style globally without app-specific restyling work.
3. Themes are packageable assets with explicit versioned contracts.
4. Existing token-only apps remain supported during migration.

## Contracts

### 1) Runtime event contract

Host broadcasts a theme payload to iframes via `postMessage`:

```json
{
  "channel": "talos:theme:v1",
  "type": "talos:theme:update",
  "theme_name": "dark",
  "variant_id": "core.dark",
  "tokens_href": "talos/tokens.css",
  "components_css_href": "talos/components.css"
}
```

Notes:

- `theme_name` is the selected host theme.
- `variant_id` maps to the component variant bundle.
- Tiny apps may ignore unknown fields for forward compatibility.

### 2) Theme variant manifest

Theme assets define variant metadata in JSON:

```json
{
  "schema_version": 1,
  "theme_name": "dark",
  "variant_id": "core.dark",
  "display_name": "Dark Core",
  "components_css_href": "/theme-assets/core.dark.components.css",
  "notes": "High-contrast dark surfaces for dashboard-heavy apps."
}
```

### 3) Web Components API stability

Talos Web Components expose:

- stable custom-element names (`talos-card`, `talos-button`, etc.)
- stable attributes (`variant`, `tone`, `size`, `disabled`)
- stable slots (`icon`, default content, optional suffix/prefix where relevant)
- stable CSS custom properties using `--talos-*`

Breaking changes require a component package major version bump.

## Package/versioning conventions

- Theme manifest IDs: `<namespace>.<theme>` (example: `core.dark`)
- Component package versions: semver (`0.x` while evolving, `1.0+` when stable)
- Schema version field required in every theme asset manifest

## Compatibility policy

- Token-only apps stay supported.
- Component-first apps become the recommended onboarding path.
- Host loads fallback token behavior when variant assets are unavailable.

## Security and isolation notes

- Theme updates are visual metadata only (no executable code evaluation from theme JSON).
- Host remains source-of-truth for allowed runtime channels and bridge policy.
- Apps must continue to pass bridge token and allowed origin checks independently of visual theme.

## Ready-to-use component plan

This list is the target catalog app developers can rely on directly.

### Available now (Phase 1)

- `talos-panel`: section container with themed surface
- `talos-card`: content container with slots (`header`, default, `footer`)
- `talos-button`: action button with `variant`, `size`, `tone`
- `talos-input`: basic text input
- `talos-alert`: callout with `tone` and title slot
- `talos-list-row`: row primitive with leading/meta/trailing slots

### Next wave (Phase 1.5)

- `talos-switch`: boolean toggle
- `talos-select`: themed select/dropdown
- `talos-segmented-control`: compact multi-option toggle
- `talos-dialog`: modal frame + action footer slots
- `talos-toast`: lightweight transient notification
- `talos-badge`: status chip / metadata marker

### App layout set (Phase 2)

- `talos-page`: page shell wrapper for consistent spacing zones
- `talos-page-header`: title/actions/secondary metadata layout
- `talos-sidebar-layout`: responsive left-nav + content structure
- `talos-toolbar`: action bar with slot regions
- `talos-empty-state`: standardized empty/result states

### Data display set (Phase 2)

- `talos-table`: table shell with sortable header slots
- `talos-stat-card`: KPI/value block
- `talos-tabs`: tab navigation + panel container
- `talos-accordion`: collapsible grouped sections
- `talos-progress`: progress bar + labels

### Form set (Phase 2)

- `talos-textarea`
- `talos-checkbox`
- `talos-radio-group`
- `talos-field`: label/help/error wrapper
- `talos-form-actions`: sticky/inline action row

### Delivery criteria per component

- Stable custom-element name and documented attributes
- Exposed `::part` contract for visual overrides
- Token/CSS-variable surface documented with examples
- Keyboard + focus behavior baseline
- Unit tests + one themed example in docs
