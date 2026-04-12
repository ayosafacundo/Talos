# Talos Dashboard — Svelte 5 Architecture & Design System

## 1. Project Structure

```
talos/
├── src/
│   ├── app.html
│   ├── app.css                    # Global styles + Tailwind directives
│   ├── routes/
│   │   └── +layout.svelte         # Shell: Sidebar + Viewport
│   ├── lib/
│   │   ├── state/
│   │   │   ├── apps.svelte.ts     # $state for activeApps, focusedAppId
│   │   │   └── theme.svelte.ts    # $state for themeConfig, theme loading
│   │   ├── components/
│   │   │   ├── Sidebar.svelte
│   │   │   ├── SidebarIcon.svelte
│   │   │   ├── ContextMenu.svelte
│   │   │   ├── Launchpad.svelte
│   │   │   ├── InventorySlot.svelte
│   │   │   ├── StoreCarousel.svelte
│   │   │   ├── StoreCard.svelte
│   │   │   ├── IframeStack.svelte
│   │   │   ├── AppFrame.svelte
│   │   │   ├── Settings.svelte
│   │   │   ├── SettingsNav.svelte
│   │   │   ├── ThemeCard.svelte
│   │   │   └── McButton.svelte
│   │   ├── types/
│   │   │   └── index.ts
│   │   └── wails/
│   │       └── bridge.ts          # Wails Go function bindings
│   └── themes/
│       ├── variables.css           # Base design tokens
│       ├── dark.css                # Nether theme overrides
│       └── light.css               # Overworld theme overrides
├── tailwind.config.ts
├── svelte.config.js
└── vite.config.ts
```

## 2. State Management (Svelte 5 Runes)

### `apps.svelte.ts`

```ts
import type { AppInstance, AppManifest } from '$lib/types';

// Reactive state via Runes
let activeApps = $state<AppInstance[]>([]);
let focusedAppId = $state<string | null>(null);
let launchpadVisible = $state(true);
let settingsVisible = $state(false);

// Derived
let focusedApp = $derived(activeApps.find(a => a.id === focusedAppId) ?? null);
let hasActiveApps = $derived(activeApps.length > 0);

export function launchApp(manifest: AppManifest) {
  const existing = activeApps.find(a => a.manifestId === manifest.id);
  if (existing) {
    focusedAppId = existing.id;
  } else {
    const instance: AppInstance = {
      id: crypto.randomUUID(),
      manifestId: manifest.id,
      name: manifest.name,
      icon: manifest.icon,
      url: manifest.url,
      tabColor: null,
    };
    activeApps = [...activeApps, instance];
    focusedAppId = instance.id;
  }
  launchpadVisible = false;
  settingsVisible = false;
}

export function closeApp(id: string) {
  activeApps = activeApps.filter(a => a.id !== id);
  if (focusedAppId === id) {
    focusedAppId = activeApps[activeApps.length - 1]?.id ?? null;
    if (!focusedAppId) launchpadVisible = true;
  }
}

export function focusApp(id: string) {
  focusedAppId = id;
  launchpadVisible = false;
  settingsVisible = false;
}

export function setTabColor(id: string, color: string) {
  activeApps = activeApps.map(a =>
    a.id === id ? { ...a, tabColor: color } : a
  );
}

export function toggleLaunchpad() {
  launchpadVisible = !launchpadVisible;
  if (launchpadVisible) settingsVisible = false;
}

export function toggleSettings() {
  settingsVisible = !settingsVisible;
  if (settingsVisible) launchpadVisible = false;
}

// Export reactive getters
export { activeApps, focusedAppId, focusedApp, launchpadVisible, settingsVisible, hasActiveApps };
```

### `theme.svelte.ts`

```ts
import type { ThemeConfig } from '$lib/types';
// Wails bridge for filesystem access
import { GetThemes, SaveUserPrefs } from '$lib/wails/bridge';

let themeConfig = $state<ThemeConfig>({
  name: 'default',
  file: 'variables.css',
});
let availableThemes = $state<ThemeConfig[]>([]);

// Load themes from /themes directory via Wails
export async function loadThemes() {
  availableThemes = await GetThemes();
}

export async function applyTheme(theme: ThemeConfig) {
  themeConfig = theme;
  // Dynamically swap the <link> tag or inject CSS variables
  const link = document.getElementById('theme-link') as HTMLLinkElement;
  if (link) link.href = `/themes/${theme.file}`;
  await SaveUserPrefs({ theme: theme.name });
}

// Inject theme vars into iframe via postMessage
export function injectThemeToIframe(iframe: HTMLIFrameElement) {
  const vars = getComputedStyle(document.documentElement);
  const themeVars: Record<string, string> = {};
  const keys = [
    '--bg-primary', '--bg-secondary', '--text-primary',
    '--text-accent', '--accent-color', '--font-family',
  ];
  keys.forEach(k => { themeVars[k] = vars.getPropertyValue(k).trim(); });
  iframe.contentWindow?.postMessage({ type: 'TALOS_THEME', vars: themeVars }, '*');
}

export { themeConfig, availableThemes };
```

## 3. Types

```ts
// src/lib/types/index.ts

export interface AppManifest {
  id: string;
  name: string;
  icon: string;        // URL or local path
  url: string;         // Iframe src
  description: string;
  category: 'installed' | 'store';
  price?: string;
}

export interface AppInstance {
  id: string;          // Unique runtime ID
  manifestId: string;  // Reference to AppManifest.id
  name: string;
  icon: string;
  url: string;
  tabColor: string | null;
}

export interface ThemeConfig {
  name: string;
  file: string;        // CSS filename in /themes
  preview?: {
    bg: string;
    sidebar: string;
    accent: string;
  };
}

export interface UserPrefs {
  theme: string;
  tabColors: Record<string, string>;
}
```

## 4. Component Architecture

### Sidebar.svelte (key logic)

```svelte
<script lang="ts">
  import { activeApps, focusedAppId, toggleLaunchpad, toggleSettings, focusApp, launchpadVisible, settingsVisible } from '$lib/state/apps.svelte';
  import SidebarIcon from './SidebarIcon.svelte';
  import ContextMenu from './ContextMenu.svelte';

  let contextMenuPos = $state({ x: 0, y: 0, visible: false, appId: '' });

  function onTabContext(e: MouseEvent, appId: string) {
    e.preventDefault();
    contextMenuPos = { x: e.clientX, y: e.clientY, visible: true, appId };
  }
</script>

<nav class="sidebar">
  <SidebarIcon icon="grid" active={launchpadVisible} onclick={toggleLaunchpad} />

  <div class="sidebar-tabs">
    {#each activeApps as app (app.id)}
      <SidebarIcon
        icon={app.icon}
        active={focusedAppId === app.id}
        tabColor={app.tabColor}
        showIndicator={focusedAppId === app.id}
        onclick={() => focusApp(app.id)}
        oncontextmenu={(e) => onTabContext(e, app.id)}
      />
    {/each}
  </div>

  <SidebarIcon icon="settings" active={settingsVisible} onclick={toggleSettings} class="mt-auto" />
</nav>

{#if contextMenuPos.visible}
  <ContextMenu {...contextMenuPos} />
{/if}
```

### IframeStack.svelte (critical — never unmount)

```svelte
<script lang="ts">
  import { activeApps, focusedAppId } from '$lib/state/apps.svelte';
  import { injectThemeToIframe } from '$lib/state/theme.svelte';

  // Theme injection on mount
  $effect(() => {
    document.querySelectorAll<HTMLIFrameElement>('.app-frame').forEach(iframe => {
      iframe.addEventListener('load', () => injectThemeToIframe(iframe), { once: true });
    });
  });
</script>

<div class="iframe-stack">
  {#each activeApps as app (app.id)}
    <iframe
      src={app.url}
      class="app-frame"
      class:active={focusedAppId === app.id}
      data-app-id={app.id}
      title={app.name}
      sandbox="allow-scripts allow-same-origin allow-forms"
    ></iframe>
  {/each}
</div>

<style>
  .iframe-stack { position: absolute; inset: 0; z-index: 30; }
  .app-frame {
    position: absolute; inset: 0;
    width: 100%; height: 100%;
    border: none;
    /* CRITICAL: toggle visibility, never unmount */
    display: none;
  }
  .app-frame.active { display: block; }
</style>
```

### Launchpad.svelte (always in DOM)

```svelte
<script lang="ts">
  import { launchpadVisible } from '$lib/state/apps.svelte';
  // Launchpad is ALWAYS rendered. Visibility toggled via CSS.
</script>

<div class="launchpad" class:visible={launchpadVisible}>
  <div class="launchpad-inner">
    <h1 class="launchpad-title">Talos</h1>
    <slot name="installed" />
    <slot name="store" />
  </div>
</div>

<style>
  .launchpad {
    position: absolute; inset: 0;
    visibility: hidden; opacity: 0;
    transition: opacity 200ms, visibility 200ms;
    z-index: 50;
  }
  .launchpad.visible { visibility: visible; opacity: 1; }
</style>
```

## 5. Wails Integration

### `bridge.ts`

```ts
// Auto-generated Wails bindings
// In production, these are generated by `wails generate`

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetThemes(): Promise<ThemeConfig[]>;
          SaveUserPrefs(prefs: UserPrefs): Promise<void>;
          GetInstalledApps(): Promise<AppManifest[]>;
          GetStoreApps(): Promise<AppManifest[]>;
        };
      };
    };
  }
}

export const GetThemes = () => window.go.main.App.GetThemes();
export const SaveUserPrefs = (prefs: UserPrefs) => window.go.main.App.SaveUserPrefs(prefs);
export const GetInstalledApps = () => window.go.main.App.GetInstalledApps();
export const GetStoreApps = () => window.go.main.App.GetStoreApps();
```

## 6. Theme Injection into Iframes

Tiny Apps can opt into Talos theming by listening for theme messages:

```js
// Inside a Tiny App's iframe
window.addEventListener('message', (e) => {
  if (e.data?.type === 'TALOS_THEME') {
    const root = document.documentElement;
    Object.entries(e.data.vars).forEach(([key, value]) => {
      root.style.setProperty(key, value as string);
    });
  }
});
```

Talos's own styles are protected via:
- Shadow DOM isolation (optional)
- Scoped CSS within Svelte components
- Iframe `sandbox` attribute restricting parent access

## 7. Design Token Reference

| Token | Purpose | Default |
|---|---|---|
| `--bg-primary` | Main background | `#2d2d2d` |
| `--bg-secondary` | Card/panel bg | `#3d3d3d` |
| `--bg-overlay` | Launchpad overlay | `rgba(0,0,0,0.8)` |
| `--text-primary` | Body text | `#d4d4d4` |
| `--text-accent` | Highlighted text | `#ffcc00` |
| `--accent-color` | Buttons, indicators | `#5b8731` |
| `--sidebar-bg` | Sidebar background | `#1a1a1a` |
| `--sidebar-width` | Sidebar width | `56px` |
| `--slot-bg` | Inventory slot fill | `#8b8b8b` |
| `--slot-border-light` | 3D border (top/left) | `#c6c6c6` |
| `--slot-border-dark` | 3D border (bot/right) | `#373737` |
| `--font-family` | Global font | `Press Start 2P` |
| `--border-radius` | Global radius | `0px` (Minecraft = no radius) |

## 8. Key Architectural Decisions

1. **Iframes are never unmounted** — only `display:none` toggled. This preserves app state (video playback, form inputs, WebSocket connections).

2. **Launchpad is always in DOM** — uses `visibility: hidden` + `opacity: 0` for instant show/hide with no re-render cost.

3. **All colors via CSS variables** — zero hardcoded colors in components. Swap a theme file = full reskin.

4. **Theme injection via postMessage** — iframes receive theme vars but can override them. Parent styles are isolated.

5. **Wails bridge is async** — all Go calls return Promises. State updates happen in Svelte's reactive system after resolution.

6. **Minecraft aesthetic** — `border-radius: 0`, beveled 3D borders on all interactive elements, `image-rendering: pixelated`, Press Start 2P font.
