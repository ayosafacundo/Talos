import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { MutableRefObject } from "react";
import {
  BroadcastMessage,
  DenyPermission,
  DevelopmentFeaturesEnabled,
  GetDeveloperMode,
  SetDeveloperMode,
  GetInstalledApps,
  GetThemes,
  GrantPermission,
  InstallPackageFromGitHub,
  InstallPackageFromURL,
  CheckForUpdates,
  DefaultUpdateChannelURL,
  ListPermissionAudit,
  ListPermissionEntries,
  ListRepositoryPackages,
  LoadUserPrefs,
  PackageLocalHTTP,
  ParanoidPackageTrust,
  PickZipAndInstall,
  SaveUserPrefs,
  GetStoreApps,
  GetStartupLaunchpad,
  LoadAppStateBase64,
  ReadScopedText,
  RequestPermissionDecision,
  ResolveScopedPath,
  RevokePermission,
  RouteMessage,
  SaveAppStateBase64,
  StartPackage,
  StopPackage,
  WriteScopedText,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import type { main } from "../wailsjs/go/models";
import {
  BRIDGE_CHANNEL,
  buildBridgeResponse,
  isAllowedMethod,
  isMessageOriginAllowed,
  parseBridgeRequest,
  postMessageTargetForAppInstance,
  replyPostMessageTarget,
  resolveTrustedSender,
} from "./bridge";
import { talosBridgeDebugLog } from "./bridge-debug";
import { wailsShellReady } from "./wails-env";

const LAUNCHPAD_ID = "app.launchpad";

type AppInstance = {
  id: string;
  manifestId: string;
  name: string;
  icon?: string;
  url: string;
  bridgeToken: string;
  /** When set (dev http iframe), bridge replies and host→iframe posts use this allowlist. */
  allowed_origins?: string[];
  /** Bumped when the package is rescanned so the iframe remounts with a fresh URL. */
  iframeEpoch: number;
};

type SettingsTab = "themes" | "components" | "permissions" | "developer" | "about";

type PermissionHistoryItem = {
  ts: number;
  app_id: string;
  scope: string;
  reason: string;
};
type ContextMenuScope = "launchpad-card" | "app-tab" | "app-view";

type AppMenuOption = {
  id: string;
  label: string;
};

type ContextMenuItem = {
  id: string;
  label: string;
  disabled?: boolean;
  action: () => void | Promise<void>;
};

type ThemeVariantManifest = {
  schema_version: number;
  theme_name: string;
  variant_id: string;
  display_name: string;
  components_css_href?: string;
  notes?: string;
};

const FALLBACK_THEME_VARIANTS: Record<string, ThemeVariantManifest> = {
  minecraft: {
    schema_version: 1,
    theme_name: "minecraft",
    variant_id: "core.minecraft",
    display_name: "Minecraft Core",
    components_css_href: "/theme-assets/core.minecraft.components.css",
  },
  dark: {
    schema_version: 1,
    theme_name: "dark",
    variant_id: "core.dark",
    display_name: "Dark Core",
    components_css_href: "/theme-assets/core.dark.components.css",
  },
  light: {
    schema_version: 1,
    theme_name: "light",
    variant_id: "core.light",
    display_name: "Light Core",
    components_css_href: "/theme-assets/core.light.components.css",
  },
};

async function loadThemeVariant(themeName: string): Promise<ThemeVariantManifest> {
  const key = String(themeName || "").trim() || "minecraft";
  try {
    const resp = await fetch(`./theme-assets/${encodeURIComponent(key)}.json`, { cache: "no-cache" });
    if (resp.ok) {
      const data = await resp.json() as ThemeVariantManifest;
      if (data?.theme_name && data?.variant_id) {
        return data;
      }
    }
  } catch {
    // Fallback below.
  }
  return FALLBACK_THEME_VARIANTS[key] || FALLBACK_THEME_VARIANTS.minecraft;
}

function withReload(url: string): string {
  if (!url) return "";
  const sep = url.includes("?") ? "&" : "?";
  return `${url}${sep}_talos_reload=${Date.now()}`;
}

function withBridgeToken(url: string, token: string): string {
  if (!url || !token) return url;
  const sep = url.includes("?") ? "&" : "?";
  let out = `${url}${sep}_talos_bt=${encodeURIComponent(token)}`;
  // WebKit/Wails often misreports ancestorOrigins for iframe→parent postMessage; shell origin is explicit.
  if (typeof window !== "undefined") {
    const shell = window.location.origin;
    if (shell) {
      out += `&_talos_shell_origin=${encodeURIComponent(shell)}`;
    }
  }
  return out;
}

function withThemeSnapshot(url: string, themeName: string, variant: ThemeVariantManifest): string {
  if (!url) return url;
  const sep = url.includes("?") ? "&" : "?";
  const origin = typeof window !== "undefined" ? window.location.origin : "";
  const themeHref = `${origin}/themes/${encodeURIComponent(themeName)}.css`;
  const tokensHref = `${origin}/talos/tokens.css`;
  const componentsHref = variant.components_css_href
    ? `${origin}${variant.components_css_href}`
    : `${origin}/talos/components.css`;
  return `${url}${sep}_talos_theme=${encodeURIComponent(themeName)}&` +
    `_talos_theme_variant=${encodeURIComponent(variant.variant_id)}&` +
    `_talos_theme_href=${encodeURIComponent(themeHref)}&` +
    `_talos_tokens_href=${encodeURIComponent(tokensHref)}&` +
    `_talos_components_href=${encodeURIComponent(componentsHref)}`;
}

/** Drop query so bridge/theme params are rebuilt cleanly (avoids duplicate _talos_* keys). */
function stripUrlForIframeBase(url: string): string {
  const u = String(url || "").trim();
  if (!u) return "";
  try {
    const parsed = new URL(u);
    parsed.search = "";
    return parsed.toString();
  } catch {
    const q = u.indexOf("?");
    return q === -1 ? u : u.slice(0, q);
  }
}

function buildAppIframeUrl(
  catalogUrl: string,
  bridgeToken: string,
  themeName: string,
  variant: ThemeVariantManifest,
): string {
  const base = String(catalogUrl || "").trim();
  if (!base || !bridgeToken) return base;
  return withThemeSnapshot(
    withBridgeToken(withReload(base), bridgeToken),
    themeName,
    variant,
  );
}

function newBridgeToken(): string {
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`;
}

function isLikelyAssetPath(value: string): boolean {
  return value.includes("/") || value.startsWith("./") || value.startsWith("../");
}

function iconURLForApp(app?: main.AppManifestView): string {
  if (!app) return "";
  const icon = String(app.icon || "");
  if (!icon) return "";
  const lower = icon.toLowerCase();
  if (
    lower.startsWith("http://") ||
    lower.startsWith("https://") ||
    lower.startsWith("file://") ||
    lower.startsWith("data:")
  ) {
    return icon;
  }
  if (!isLikelyAssetPath(icon)) return "";
  const appURL = String(app.url || "");
  const isFile = appURL.startsWith("file://");
  const isPkg = appURL.startsWith("/talos-pkg/");
  if (!isFile && !isPkg) return "";

  const marker = "/dist/";
  const idx = appURL.indexOf(marker);
  if (idx >= 0) {
    const packageRoot = appURL.slice(0, idx);
    return `${packageRoot}/${icon.replace(/^\/+/, "")}`;
  }
  try {
    const base = isFile ? appURL : `https://local.invalid${appURL}`;
    const resolved = new URL(icon, base);
    if (isPkg) {
      return `${resolved.pathname}${resolved.search}${resolved.hash}`;
    }
    return resolved.toString();
  } catch {
    return "";
  }
}

function formatTrustStatus(ts: string): string {
  switch (ts) {
    case "ok":
      return "integrity OK";
    case "tampered":
      return "tampered";
    case "unsigned":
      return "unsigned";
    case "signed_ok":
      return "signed";
    case "signed_invalid":
      return "bad signature";
    case "unknown":
      return "unknown integrity";
    default:
      return ts;
  }
}

function userFacingFetchError(err: unknown): string {
  const msg = String(err || "");
  if (msg.includes("offline or DNS issue")) {
    return `${msg}\nHint: verify network connectivity and the configured HTTPS URL.`;
  }
  if (msg.includes("HTTP ")) {
    return `${msg}\nHint: check server status and endpoint path.`;
  }
  if (msg.includes("invalid channel JSON") || msg.includes("invalid catalog JSON")) {
    return `${msg}\nHint: validate that the endpoint returns a JSON array with expected fields.`;
  }
  return msg;
}

function themeHref(themeName: string): string {
  return `./themes/${themeName}.css`;
}

function ensureThemeLink(): HTMLLinkElement {
  let link = document.getElementById("talos-theme-link") as HTMLLinkElement | null;
  if (!link) {
    link = document.createElement("link");
    link.id = "talos-theme-link";
    link.rel = "stylesheet";
    document.head.appendChild(link);
  }
  return link;
}

function applyTheme(themeName: string): void {
  const link = ensureThemeLink();
  link.href = themeHref(themeName);
}

type TalosPackageIframeProps = {
  app: AppInstance;
  isFocused: boolean;
  iframeRefs: MutableRefObject<Record<string, HTMLIFrameElement | null>>;
  iframeLoadedEpochRef: MutableRefObject<Record<string, number>>;
  postThemeUpdate: (app: AppInstance) => void;
};

/** Own row so the iframe ref callback can stay referentially stable across parent re-renders (avoids spurious ref(null) from inline refs). */
function TalosPackageIframe({
  app,
  isFocused,
  iframeRefs,
  iframeLoadedEpochRef,
  postThemeUpdate,
}: TalosPackageIframeProps): React.ReactElement {
  const setRef = useCallback(
    (node: HTMLIFrameElement | null) => {
      iframeRefs.current[app.id] = node;
    },
    [app.id, app.manifestId, iframeRefs],
  );

  return (
    <iframe
      src={app.url}
      title={app.name}
      data-talos-manifest-id={app.manifestId}
      data-talos-bridge-token={app.bridgeToken}
      ref={setRef}
      style={{ display: isFocused ? "block" : "none" }}
      sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
      onLoad={() => {
        iframeLoadedEpochRef.current[app.id] = app.iframeEpoch;
        postThemeUpdate(app);
      }}
      onError={() => {
        iframeLoadedEpochRef.current[app.id] = -1;
        talosBridgeDebugLog("iframe_nav_error", { instanceId: app.id, src: app.url });
      }}
    />
  );
}

export default function App(): React.ReactElement {
  const [installedApps, setInstalledApps] = useState<main.AppManifestView[]>([]);
  const [storeApps, setStoreApps] = useState<main.AppManifestView[]>([]);
  const [activeApps, setActiveApps] = useState<AppInstance[]>([]);
  const [focusedAppId, setFocusedAppId] = useState<string | null>(null);
  const [launchpadVisible, setLaunchpadVisible] = useState(true);
  const [settingsVisible, setSettingsVisible] = useState(false);
  const [settingsTab, setSettingsTab] = useState<SettingsTab>("themes");
  const [themes, setThemes] = useState<main.ThemeInfo[]>([]);
  const [currentTheme, setCurrentTheme] = useState("minecraft");
  const [startupError, setStartupError] = useState("");
  const [permissionPrompt, setPermissionPrompt] = useState<{
    app_id: string;
    scope: string;
    reason: string;
  } | null>(null);
  const [permissionHistory, setPermissionHistory] = useState<PermissionHistoryItem[]>([]);
  const [permissionEntries, setPermissionEntries] = useState<main.PermissionEntry[]>([]);
  const [permissionAudit, setPermissionAudit] = useState<main.PermissionAuditEntry[]>([]);
  const [hostBanner, setHostBanner] = useState("");
  const [updateChannelURL, setUpdateChannelURL] = useState("");
  const [updateCheckResult, setUpdateCheckResult] = useState<string>("");
  const [installUiEnabled, setInstallUiEnabled] = useState(false);
  const [developerMode, setDeveloperMode] = useState(false);
  const [installMessage, setInstallMessage] = useState("");
  const [installBusy, setInstallBusy] = useState(false);
  const [urlInstall, setUrlInstall] = useState("");
  const [ghOwner, setGhOwner] = useState("");
  const [ghRepo, setGhRepo] = useState("");
  const [ghRef, setGhRef] = useState("");
  const [repoBrowse, setRepoBrowse] = useState<main.RemotePackageDescriptor[]>([]);
  const [repoBrowseError, setRepoBrowseError] = useState("");
  const [appContextOptions, setAppContextOptions] = useState<Record<string, AppMenuOption[]>>({});
  const [activeThemeVariant, setActiveThemeVariant] = useState<ThemeVariantManifest>(FALLBACK_THEME_VARIANTS.minecraft);
  const [contextMenuState, setContextMenuState] = useState<{
    visible: boolean;
    x: number;
    y: number;
    title: string;
    scope: ContextMenuScope;
    items: ContextMenuItem[];
  }>({
    visible: false,
    x: 0,
    y: 0,
    title: "",
    scope: "launchpad-card",
    items: [],
  });
  const iframeRefs = useRef<Record<string, HTMLIFrameElement | null>>({});
  /** Last `iframeEpoch` for which `onLoad` fired; `-1` = navigation error (skip theme push). */
  const iframeLoadedEpochRef = useRef<Record<string, number>>({});
  const packagesEventDebounceRef = useRef<number | undefined>(undefined);
  const activeAppsRef = useRef<AppInstance[]>([]);
  const focusedAppIdRef = useRef<string | null>(null);
  const launchpadVisibleRef = useRef(false);

  activeAppsRef.current = activeApps;
  focusedAppIdRef.current = focusedAppId;
  launchpadVisibleRef.current = launchpadVisible;

  const launchableApps = useMemo(
    () => installedApps.filter((app) => app.id !== LAUNCHPAD_ID),
    [installedApps],
  );
  const placeholderIconSource = useMemo(() => {
    const tinyGo = installedApps.find((app) => app.id === "app.tiny.go.demo");
    const tinyTs = installedApps.find((app) => app.id === "app.tiny.ts.demo");
    return iconURLForApp(tinyGo)
      || iconURLForApp(tinyTs)
      || "";
  }, [installedApps]);

  async function reloadCatalog(): Promise<main.AppManifestView[]> {
    const [installed, store] = await Promise.all([GetInstalledApps(), GetStoreApps()]);
    setInstalledApps(installed ?? []);
    setStoreApps(store ?? []);
    return installed ?? [];
  }

  async function reloadThemes(): Promise<void> {
    const [themeList, prefs] = await Promise.all([GetThemes(), LoadUserPrefs()]);
    setThemes(themeList ?? []);
    const selectedTheme = String(prefs?.theme || "minecraft");
    setCurrentTheme(selectedTheme);
    applyTheme(selectedTheme);
  }

  async function reloadPermissionsSettings(): Promise<void> {
    const [rows, audit] = await Promise.all([ListPermissionEntries(), ListPermissionAudit(200)]);
    setPermissionEntries(rows ?? []);
    setPermissionAudit(audit ?? []);
  }

  function hideContextMenu(): void {
    setContextMenuState((prev) => ({ ...prev, visible: false }));
  }

  function showContextMenu(
    event: React.MouseEvent,
    title: string,
    scope: ContextMenuScope,
    items: ContextMenuItem[],
  ): void {
    // With Wails EnableDefaultContextMenu, allow native menu (e.g. Inspect Element) when holding Ctrl/Meta/Shift.
    if (event.ctrlKey || event.metaKey || event.shiftKey) {
      return;
    }
    event.preventDefault();
    showContextMenuAt(event.clientX, event.clientY, title, scope, items);
  }

  function showContextMenuAt(
    x: number,
    y: number,
    title: string,
    scope: ContextMenuScope,
    items: ContextMenuItem[],
  ): void {
    if (items.length === 0) return;
    setContextMenuState({
      visible: true,
      x,
      y,
      title,
      scope,
      items,
    });
  }

  async function closeAppInstance(instanceId: string, manifestId: string): Promise<void> {
    const prev = activeAppsRef.current;
    const next = prev.filter((app) => app.id !== instanceId);
    const f = focusedAppIdRef.current;
    const lp = launchpadVisibleRef.current;
    setActiveApps(next);
    setFocusedAppId(f === instanceId ? next[0]?.id ?? null : f);
    setLaunchpadVisible(lp || next.length === 0);
    delete iframeLoadedEpochRef.current[instanceId];
    try {
      await StopPackage(manifestId);
    } catch (error) {
      console.warn(`failed to stop package ${manifestId}:`, error);
    }
  }

  function sendContextActionToFocusedApp(actionID: string): void {
    if (!focusedAppId) return;
    const focused = activeApps.find((app) => app.id === focusedAppId);
    if (!focused) return;
    const iframe = iframeRefs.current[focused.id];
    const bridgeToken = iframe?.dataset.talosBridgeToken ?? "";
    const target = iframe ? postMessageTargetForAppInstance(focused, iframe) : "*";
    iframe?.contentWindow?.postMessage(
      {
        channel: BRIDGE_CHANNEL,
        type: "talos:context:action",
        app_id: focused.manifestId,
        action_id: actionID,
        bridge_token: bridgeToken,
      },
      target,
    );
  }

  function appViewContextItems(instance: AppInstance): ContextMenuItem[] {
    const custom = appContextOptions[instance.manifestId] ?? [];
    const customItems: ContextMenuItem[] = custom.map((item) => ({
      id: `custom:${item.id}`,
      label: item.label,
      action: () => sendContextActionToFocusedApp(item.id),
    }));
    return [
      {
        id: "focus",
        label: "Focus App",
        action: () => focusApp(instance.id),
      },
      {
        id: "close",
        label: "Close App",
        action: () => closeAppInstance(instance.id, instance.manifestId),
      },
      ...customItems,
    ];
  }

  function focusApp(instanceId: string): void {
    setFocusedAppId(instanceId);
    setLaunchpadVisible(false);
    setSettingsVisible(false);
  }

  async function launchApp(manifest: main.AppManifestView): Promise<void> {
    const paranoid = await ParanoidPackageTrust();
    if (paranoid && String(manifest.trust_status || "") === "tampered") {
      setHostBanner("This package failed integrity verification (tampered). Reinstall or remove it.");
      return;
    }
    setHostBanner("");
    const prev = activeAppsRef.current;
    const existing = prev.find((app) => app.manifestId === manifest.id);
    if (existing) {
      setFocusedAppId(existing.id);
      setLaunchpadVisible(false);
      setSettingsVisible(false);
      return;
    }
    await StartPackage(manifest.id);
    let launchManifest = manifest;
    try {
      const installedNow = await GetInstalledApps();
      const fresh = (installedNow ?? []).find((m) => m.id === manifest.id);
      if (fresh) {
        launchManifest = fresh;
      }
    } catch {
      // Keep original manifest snapshot if catalog refresh fails.
    }
    const bridgeToken = newBridgeToken();
    const origins = launchManifest.allowed_origins;
    const next: AppInstance = {
      id: `${launchManifest.id}:${Date.now()}`,
      manifestId: launchManifest.id,
      name: launchManifest.name,
      icon: launchManifest.icon,
      url: buildAppIframeUrl(
        launchManifest.url ?? "",
        bridgeToken,
        currentTheme,
        activeThemeVariant,
      ),
      bridgeToken,
      allowed_origins: origins && origins.length > 0 ? [...origins] : undefined,
      iframeEpoch: 0,
    };
    setFocusedAppId(next.id);
    setLaunchpadVisible(false);
    setSettingsVisible(false);
    setActiveApps((p) => {
      if (p.some((a) => a.manifestId === launchManifest.id)) return p;
      return [...p, next];
    });
  }

  useEffect(() => {
    let offPerm: (() => void) | undefined;
    let pollTimer: number | undefined;
    const deadline = Date.now() + 120_000;

    const attach = (): boolean => {
      if (!wailsShellReady()) return false;
      offPerm = EventsOn("permissions:request", (data: Record<string, string>) => {
        const row: PermissionHistoryItem = {
          ts: Date.now(),
          app_id: String(data.app_id || ""),
          scope: String(data.scope || ""),
          reason: String(data.reason || ""),
        };
        setPermissionHistory((h) => [row, ...h].slice(0, 40));
        setPermissionPrompt({
          app_id: row.app_id,
          scope: row.scope,
          reason: row.reason,
        });
      });
      return true;
    };

    if (!attach()) {
      pollTimer = window.setInterval(() => {
        if (Date.now() > deadline) {
          if (pollTimer !== undefined) window.clearInterval(pollTimer);
          pollTimer = undefined;
          return;
        }
        if (attach() && pollTimer !== undefined) {
          window.clearInterval(pollTimer);
          pollTimer = undefined;
        }
      }, 50);
    }

    return () => {
      if (pollTimer !== undefined) window.clearInterval(pollTimer);
      if (typeof offPerm === "function") offPerm();
    };
  }, []);

  useEffect(() => {
    if (settingsVisible && settingsTab === "permissions") {
      void reloadPermissionsSettings();
    }
  }, [settingsVisible, settingsTab]);

  useEffect(() => {
    let mounted = true;
    const onMessage = async (event: MessageEvent): Promise<void> => {
      const raw = event?.data;
      if (!raw || typeof raw !== "object" || (raw as { type?: string }).type !== "talos:sdk:req") {
        return;
      }

      const parsed = parseBridgeRequest(raw);
      const sourceWindow = event.source as Window | null;

      let bridgeAllowedOrigins: string[] | undefined;

      const respond = (ok: boolean, result: unknown, error = "", requestId = ""): void => {
        const id = requestId || (parsed?.requestId ?? "");
        if (!id) return;
        sourceWindow?.postMessage(
          buildBridgeResponse(id, ok, result, error),
          replyPostMessageTarget(event, bridgeAllowedOrigins),
        );
      };

      if (!parsed) {
        if (import.meta.env.DEV) {
          console.warn("[talos bridge] invalid or legacy envelope (v1 channel + bridge_token required)", raw);
        }
        return;
      }

      const trust = resolveTrustedSender(
        sourceWindow,
        iframeRefs.current,
        parsed.appId,
        parsed.bridgeToken,
      );
      if (!trust.ok) {
        const reason = "reason" in trust ? trust.reason : "source_mismatch";
        if (import.meta.env.DEV) {
          console.warn("[talos bridge] rejected:", reason, { app_id: parsed.appId });
        }
        respond(false, null, `bridge rejected: ${reason}`, parsed.requestId);
        return;
      }

      bridgeAllowedOrigins = activeApps.find((a) => a.id === trust.trusted.instanceId)?.allowed_origins;
      if (!isMessageOriginAllowed(event.origin, bridgeAllowedOrigins)) {
        if (import.meta.env.DEV) {
          console.warn("[talos bridge] rejected: origin not in allowed_origins", {
            app_id: parsed.appId,
            origin: event.origin,
          });
        }
        respond(false, null, "bridge rejected: origin not allowed", parsed.requestId);
        return;
      }

      const manifestId = trust.trusted.manifestId;
      const method = parsed.method;
      const params = parsed.params;

      if (!isAllowedMethod(method)) {
        respond(false, null, `unsupported method: ${method}`, parsed.requestId);
        return;
      }

      try {
        if (method === "getInstalledApps") {
          respond(true, { apps: launchableApps }, "", parsed.requestId);
          return;
        }
        if (method === "getStoreApps") {
          respond(true, { apps: storeApps }, "", parsed.requestId);
          return;
        }
        if (method === "launchApp") {
          const targetId = String(params.app_id || "");
          const target = installedApps.find((app) => app.id === targetId);
          if (!target) {
            respond(false, null, `app not found: ${targetId}`, parsed.requestId);
            return;
          }
          await launchApp(target);
          respond(true, { launched: true, app_id: targetId }, "", parsed.requestId);
          return;
        }
        if (method === "saveState") {
          await SaveAppStateBase64(manifestId, String(params.data_base64 || ""));
          respond(true, { ok: true }, "", parsed.requestId);
          return;
        }
        if (method === "loadState") {
          const data = await LoadAppStateBase64(manifestId);
          respond(true, { data_base64: data, found: !!data }, "", parsed.requestId);
          return;
        }
        if (method === "requestPermission") {
          const out = await RequestPermissionDecision(
            manifestId,
            String(params.scope || ""),
            String(params.reason || ""),
          );
          respond(
            !(out as { error?: string })?.error,
            out,
            (out as { error?: string })?.error || "",
            parsed.requestId,
          );
          return;
        }
        if (method === "resolvePath") {
          const resolved = await ResolveScopedPath(manifestId, String(params.relative_path || ""));
          respond(true, { resolved_path: resolved }, "", parsed.requestId);
          return;
        }
        if (method === "readScopedText") {
          const out = await ReadScopedText(manifestId, String(params.relative_path || ""));
          respond(true, out, "", parsed.requestId);
          return;
        }
        if (method === "packageLocalHttp") {
          const httpMethod = String(params.method || "GET").trim().toUpperCase();
          const httpPath = String(params.path || "");
          const httpBody = String(params.body || "");
          const out = await PackageLocalHTTP(manifestId, httpMethod, httpPath, httpBody);
          respond(
            true,
            {
              status: out.status,
              content_type: out.content_type,
              body: out.body,
            },
            "",
            parsed.requestId,
          );
          return;
        }
        if (method === "writeScopedText") {
          await WriteScopedText(
            manifestId,
            String(params.relative_path || ""),
            String(params.text || ""),
          );
          respond(true, { ok: true }, "", parsed.requestId);
          return;
        }
        if (method === "sendMessage") {
          const payload = await RouteMessage(
            manifestId,
            String(params.target_id || ""),
            String(params.type || "pkg:msg"),
            String(params.payload || ""),
          );
          respond(true, { payload }, "", parsed.requestId);
          return;
        }
        if (method === "broadcast") {
          const recipients = await BroadcastMessage(
            manifestId,
            String(params.type || "pkg:broadcast"),
            String(params.payload || ""),
          );
          respond(true, { recipients }, "", parsed.requestId);
          return;
        }
        if (method === "setContextMenuOptions") {
          const incoming = Array.isArray(params.options) ? params.options : [];
          const normalized = incoming
            .map((entry: unknown) => {
              const rawEntry = entry as { id?: string; label?: string };
              return {
                id: String(rawEntry?.id || "").trim(),
                label: String(rawEntry?.label || "").trim(),
              };
            })
            .filter((entry: AppMenuOption) => entry.id !== "" && entry.label !== "")
            .slice(0, 20);
          setAppContextOptions((prev) => ({
            ...prev,
            [manifestId]: normalized,
          }));
          respond(true, { count: normalized.length }, "", parsed.requestId);
          return;
        }
        if (method === "clearContextMenuOptions") {
          setAppContextOptions((prev) => ({
            ...prev,
            [manifestId]: [],
          }));
          respond(true, { ok: true }, "", parsed.requestId);
          return;
        }
        if (method === "openContextMenu") {
          const target = activeApps.find((app) => app.manifestId === manifestId);
          if (!target) {
            respond(false, null, "app is not active", parsed.requestId);
            return;
          }
          const x = Number(params.x);
          const y = Number(params.y);
          const fallbackX = Math.round(window.innerWidth * 0.5);
          const fallbackY = Math.round(window.innerHeight * 0.5);
          showContextMenuAt(
            Number.isFinite(x) ? x : fallbackX,
            Number.isFinite(y) ? y : fallbackY,
            target.name,
            "app-view",
            appViewContextItems(target),
          );
          respond(true, { ok: true }, "", parsed.requestId);
          return;
        }
        respond(false, null, `unsupported method: ${method}`, parsed.requestId);
      } catch (error) {
        respond(false, null, String(error), parsed.requestId);
      }
    };

    const bootstrap = async (): Promise<void> => {
      try {
        const launchpad = await GetStartupLaunchpad();
        if (!launchpad || launchpad.id !== LAUNCHPAD_ID) {
          if (mounted) setStartupError("Required package app.launchpad is missing or invalid.");
          return;
        }
        await reloadCatalog();
        await reloadThemes();
        void GetDeveloperMode().then((v) => setDeveloperMode(!!v));
        void DevelopmentFeaturesEnabled().then((v) => setInstallUiEnabled(!!v));
        void ListRepositoryPackages()
          .then((rows) => {
            setRepoBrowse(rows ?? []);
            setRepoBrowseError("");
          })
          .catch((e: unknown) => {
            setRepoBrowse([]);
            setRepoBrowseError(userFacingFetchError(e));
          });
        void DefaultUpdateChannelURL().then((u) => setUpdateChannelURL(u ?? ""));
      } catch (error) {
        if (mounted) setStartupError(String(error));
      }
    };

    const onDismissMenu = (): void => hideContextMenu();

    let offPackages: (() => void) | undefined;
    let offDevURL: (() => void) | undefined;
    let pollTimer: number | undefined;
    const wailsDeadline = Date.now() + 120_000;

    const refreshInstalledAndIframeFor = (pid: string): void => {
      if (packagesEventDebounceRef.current !== undefined) {
        window.clearTimeout(packagesEventDebounceRef.current);
      }
      packagesEventDebounceRef.current = window.setTimeout(() => {
        packagesEventDebounceRef.current = undefined;
        void (async () => {
          const installed = await reloadCatalog();
          if (!pid) {
            return;
          }
          setActiveApps((prev) =>
            prev.map((app) => {
              if (app.manifestId !== pid) {
                return app;
              }
              const manifest = installed.find((m) => m.id === pid);
              const base = manifest?.url?.trim()
                ? manifest.url
                : stripUrlForIframeBase(app.url);
              const nextUrl = buildAppIframeUrl(base, app.bridgeToken, currentTheme, activeThemeVariant);
              const origins = manifest?.allowed_origins;
              return {
                ...app,
                iframeEpoch: app.iframeEpoch + 1,
                url: nextUrl,
                icon: manifest?.icon ?? app.icon,
                allowed_origins: origins && origins.length > 0 ? [...origins] : undefined,
              };
            }),
          );
        })();
      }, 300);
    };

    const attachWails = (): boolean => {
      if (!wailsShellReady()) return false;
      void bootstrap();
      window.addEventListener("message", onMessage);
      window.addEventListener("click", onDismissMenu);
      window.addEventListener("scroll", onDismissMenu, true);
      offPackages = EventsOn("packages:event", (evt: Record<string, unknown>) => {
        const pid = String(evt?.package_id ?? "");
        refreshInstalledAndIframeFor(pid);
      });
      offDevURL = EventsOn("package:dev-url", (evt: Record<string, unknown>) => {
        const pid = String(evt?.app_id ?? "");
        refreshInstalledAndIframeFor(pid);
      });
      return true;
    };

    if (!attachWails()) {
      pollTimer = window.setInterval(() => {
        if (Date.now() > wailsDeadline) {
          if (pollTimer !== undefined) window.clearInterval(pollTimer);
          pollTimer = undefined;
          if (mounted) {
            setStartupError(
              "Wails runtime not ready. Use the Talos dev window (not the raw Vite URL) or wait for the shell to inject bindings.",
            );
          }
          return;
        }
        if (attachWails() && pollTimer !== undefined) {
          window.clearInterval(pollTimer);
          pollTimer = undefined;
        }
      }, 50);
    }

    return () => {
      mounted = false;
      if (pollTimer !== undefined) window.clearInterval(pollTimer);
      if (packagesEventDebounceRef.current !== undefined) {
        window.clearTimeout(packagesEventDebounceRef.current);
        packagesEventDebounceRef.current = undefined;
      }
      window.removeEventListener("message", onMessage);
      window.removeEventListener("click", onDismissMenu);
      window.removeEventListener("scroll", onDismissMenu, true);
      if (typeof offPackages === "function") offPackages();
      if (typeof offDevURL === "function") offDevURL();
    };
  }, [installedApps, launchableApps, storeApps, activeApps]);

  async function selectTheme(name: string): Promise<void> {
    applyTheme(name);
    setCurrentTheme(name);
    const variant = await loadThemeVariant(name);
    setActiveThemeVariant(variant);
    const prefs = await LoadUserPrefs();
    await SaveUserPrefs({
      ...prefs,
      theme: name,
    });
  }

  useEffect(() => {
    void (async () => {
      const variant = await loadThemeVariant(currentTheme);
      setActiveThemeVariant(variant);
    })();
  }, [currentTheme]);

  function postThemeUpdateToApp(app: AppInstance): void {
    const iframe = iframeRefs.current[app.id];
    if (!iframe) {
      return;
    }
    if (!iframe.contentWindow) {
      return;
    }
    if (iframeLoadedEpochRef.current[app.id] !== app.iframeEpoch) {
      talosBridgeDebugLog("host_to_iframe_theme_skipped", {
        reason: "iframe_not_loaded_for_epoch",
        instanceId: app.id,
        epoch: app.iframeEpoch,
        loadedEpoch: iframeLoadedEpochRef.current[app.id],
      });
      return;
    }
    const shellOrigin = typeof window !== "undefined" ? window.location.origin : "";
    const componentsPath = activeThemeVariant.components_css_href || "/talos/components.css";
    const componentsAbs = componentsPath.startsWith("/")
      ? `${shellOrigin}${componentsPath}`
      : `${shellOrigin}/${componentsPath}`;
    const payload = {
      channel: "talos:theme:v1",
      type: "talos:theme:update",
      theme_name: currentTheme,
      variant_id: activeThemeVariant.variant_id,
      theme_href: `${shellOrigin}/themes/${encodeURIComponent(currentTheme)}.css`,
      tokens_href: `${shellOrigin}/talos/tokens.css`,
      components_css_href: componentsAbs,
    };
    try {
      // Theme payloads are only CSS URLs (no bridge secrets). `*` avoids WebKit mismatches when
      // iframe.src origin ≠ live document (e.g. dev server down → embedded error page).
      talosBridgeDebugLog("host_to_iframe_theme", {
        instanceId: app.id,
        iframeSrc: iframe.src,
        targetOrigin: "*",
      });
      iframe.contentWindow.postMessage(payload, "*");
    } catch {
      /* postMessage can fail for cross-origin or torn-down frames */
    }
  }

  useEffect(() => {
    for (const app of activeApps) {
      postThemeUpdateToApp(app);
    }
  }, [activeApps, currentTheme, activeThemeVariant]);

  // Launchpad/Settings are full-viewport panels; app-tab focus must not stay set while they are visible
  // (otherwise the sidebar can show an app as "active" while the viewport still shows Launchpad).
  useEffect(() => {
    if (!focusedAppId) return;
    if (!launchpadVisible && !settingsVisible) return;
    setFocusedAppId(null);
  }, [launchpadVisible, settingsVisible, focusedAppId]);

  function resolvePermissionPrompt(granted: boolean): void {
    if (!permissionPrompt) return;
    const appID = permissionPrompt.app_id;
    const scope = permissionPrompt.scope;
    const complete = granted ? GrantPermission(appID, scope) : DenyPermission(appID, scope);
    void complete
      .then(() => reloadPermissionsSettings())
      .finally(() => setPermissionPrompt(null));
  }

  return (
    <main className="shell">
      <aside className="sidebar">
        <button
          type="button"
          className={`icon-btn ${launchpadVisible ? "active" : ""}`}
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => {
            setFocusedAppId(null);
            setLaunchpadVisible(true);
            setSettingsVisible(false);
          }}
          title="Launchpad"
        >
          🚀
        </button>
        <div className="tabs">
          {activeApps.map((app) => (
            <button
              type="button"
              key={app.id}
              className={`icon-btn ${
                focusedAppId === app.id && !launchpadVisible && !settingsVisible ? "active" : ""
              }`}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => {
                focusApp(app.id);
              }}
              onContextMenu={(event) =>
                showContextMenu(event, app.name, "app-tab", appViewContextItems(app))
              }
              title={app.name}
            >
              {(() => {
                const iconSource = iconURLForApp({
                  id: app.manifestId,
                  name: app.name,
                  icon: String(app.icon || ""),
                  url: app.url,
                  description: "",
                  category: "installed",
                });
                if (!iconSource) {
                  return <span className="tab-icon-fallback show">{app.icon || "🧩"}</span>;
                }
                return (
                  <>
                    <img
                      className="tab-icon-image"
                      src={iconSource}
                      alt={app.name}
                      onError={(evt) => {
                        const img = evt.currentTarget;
                        img.style.display = "none";
                        const sibling = img.nextElementSibling as HTMLElement | null;
                        if (sibling) sibling.style.display = "grid";
                      }}
                    />
                    <span className="tab-icon-fallback">{app.icon || "🧩"}</span>
                  </>
                );
              })()}
            </button>
          ))}
        </div>
        <button
          type="button"
          className={`icon-btn ${settingsVisible ? "active" : ""}`}
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => {
            setFocusedAppId(null);
            setSettingsVisible((prev) => !prev);
            setLaunchpadVisible(false);
          }}
          title="Settings"
        >
          ⚙️
        </button>
      </aside>

      <section className="viewport">
        {startupError ? (
          <div className="error">{startupError}</div>
        ) : (
          <>
            <section className={`launchpad ${launchpadVisible ? "show" : "hide"}`}>
              <h1>Talos Launchpad</h1>
              {hostBanner ? <div className="host-banner">{hostBanner}</div> : null}
              <h2>Installed</h2>
              <div className="list">
                {launchableApps.map((app) => (
                  <button
                    key={app.id}
                    className="app-card"
                    onClick={() => {
                      void launchApp(app);
                    }}
                    onContextMenu={(event) =>
                      showContextMenu(event, app.name, "launchpad-card", [
                        {
                          id: "open",
                          label: "Open App",
                          action: () => launchApp(app),
                        },
                      ])
                    }
                  >
                    <span className="icon">
                      {(() => {
                        const iconSrc = iconURLForApp(app) || placeholderIconSource;
                        return iconSrc ? (
                          <img
                            className="app-icon-image"
                            src={iconSrc}
                            alt={app.name}
                            onError={(evt) => {
                              const img = evt.currentTarget;
                              img.style.display = "none";
                              const sibling = img.nextElementSibling as HTMLElement | null;
                              if (sibling) sibling.style.display = "grid";
                            }}
                          />
                        ) : null;
                      })()}
                      <span className={`app-icon-fallback ${!(iconURLForApp(app) || placeholderIconSource) ? "show" : ""}`}>
                        {app.icon || "🧩"}
                      </span>
                    </span>
                    <span>
                      <strong>{app.name}</strong>
                      <small>
                        {app.description || "Installed Tiny App"}
                        {app.trust_status ? (
                          <span className={`trust-badge trust-${app.trust_status}`}>
                            {" "}
                            · {formatTrustStatus(app.trust_status)}
                          </span>
                        ) : null}
                      </small>
                    </span>
                  </button>
                ))}
              </div>
              {installUiEnabled ? (
                <section className="add-package">
                  <h2>Add package</h2>
                  <p className="perm-hint">
                    Install from a local zip, an HTTPS URL to a zip archive, or a GitHub repository zipball (development
                    mode only).
                  </p>
                  {installMessage ? <div className="install-msg">{installMessage}</div> : null}
                  <div className="install-row">
                    <button
                      type="button"
                      className="ui-button install-btn"
                      disabled={installBusy}
                      onClick={() => {
                        setInstallBusy(true);
                        setInstallMessage("");
                        void PickZipAndInstall()
                          .then((id) => setInstallMessage(id ? `Installed package: ${id}` : "Cancelled."))
                          .catch((e: unknown) => setInstallMessage(String(e)))
                          .finally(() => setInstallBusy(false));
                      }}
                    >
                      Choose .zip…
                    </button>
                  </div>
                  <div className="install-row install-row-split">
                    <input
                      className="ui-input"
                      placeholder="https://example.com/app.zip"
                      value={urlInstall}
                      onChange={(e) => setUrlInstall(e.target.value)}
                    />
                    <button
                      type="button"
                      className="ui-button install-btn"
                      disabled={installBusy}
                      onClick={() => {
                        setInstallBusy(true);
                        setInstallMessage("");
                        void InstallPackageFromURL(urlInstall)
                          .then((id) => {
                            setInstallMessage(`Installed package: ${id}`);
                            setUrlInstall("");
                          })
                          .catch((e: unknown) => setInstallMessage(String(e)))
                          .finally(() => setInstallBusy(false));
                      }}
                    >
                      Install from URL
                    </button>
                  </div>
                  <div className="install-row install-row-github">
                    <input
                      className="ui-input"
                      placeholder="owner"
                      value={ghOwner}
                      onChange={(e) => setGhOwner(e.target.value)}
                    />
                    <input
                      className="ui-input"
                      placeholder="repo"
                      value={ghRepo}
                      onChange={(e) => setGhRepo(e.target.value)}
                    />
                    <input
                      className="ui-input"
                      placeholder="ref (branch or tag, optional)"
                      value={ghRef}
                      onChange={(e) => setGhRef(e.target.value)}
                    />
                    <button
                      type="button"
                      className="ui-button install-btn"
                      disabled={installBusy}
                      onClick={() => {
                        setInstallBusy(true);
                        setInstallMessage("");
                        void InstallPackageFromGitHub(ghOwner, ghRepo, ghRef)
                          .then((id) => {
                            setInstallMessage(`Installed package: ${id}`);
                            setGhOwner("");
                            setGhRepo("");
                            setGhRef("");
                          })
                          .catch((e: unknown) => setInstallMessage(String(e)))
                          .finally(() => setInstallBusy(false));
                      }}
                    >
                      Install from GitHub
                    </button>
                  </div>
                </section>
              ) : null}
              <h2>Repositories</h2>
              <p className="perm-hint">
                Set <code>TALOS_CATALOG_URL</code> to an HTTPS JSON feed (see docs/PHASE3.md). Otherwise the catalog is empty.
              </p>
              {repoBrowseError ? <pre className="update-check-out">{repoBrowseError}</pre> : null}
              <div className="list">
                {repoBrowse.map((p) => (
                  <div key={p.id} className="app-card repo-row">
                    <span className="icon">📦</span>
                    <span>
                      <strong>{p.name || p.id}</strong>
                      <small>
                        <code>{p.id}</code>
                        {p.source ? ` · ${p.source}` : ""}
                      </small>
                    </span>
                    {installUiEnabled && p.install_url ? (
                      <button
                        type="button"
                        className="ui-button install-btn"
                        disabled={installBusy}
                        onClick={() => {
                          setInstallBusy(true);
                          setInstallMessage("");
                          void InstallPackageFromURL(p.install_url!)
                            .then((id) => {
                              setInstallMessage(`Installed: ${id}`);
                              return reloadCatalog();
                            })
                            .catch((e: unknown) => setInstallMessage(String(e)))
                            .finally(() => setInstallBusy(false));
                        }}
                      >
                        Install
                      </button>
                    ) : null}
                  </div>
                ))}
              </div>
              <h2>Store</h2>
              <div className="list">
                {storeApps.map((app) => (
                  <a key={app.id} href={app.store_url || "#"} target="_blank" rel="noreferrer" className="app-card">
                    <span className="icon">{app.icon || "🛒"}</span>
                    <span>
                      <strong>{app.name}</strong>
                      <small>{app.description || "Store app"}</small>
                    </span>
                  </a>
                ))}
              </div>
            </section>

            <section
              className={`apps ${launchpadVisible || settingsVisible ? "hide" : "show"}`}
              onContextMenu={(event) => {
                const focused = activeApps.find((app) => app.id === focusedAppId);
                if (!focused) return;
                showContextMenu(event, focused.name, "app-view", appViewContextItems(focused));
              }}
            >
              {activeApps.map((app) => (
                <TalosPackageIframe
                  key={`${app.id}:${app.iframeEpoch}`}
                  app={app}
                  isFocused={focusedAppId === app.id}
                  iframeRefs={iframeRefs}
                  iframeLoadedEpochRef={iframeLoadedEpochRef}
                  postThemeUpdate={postThemeUpdateToApp}
                />
              ))}
            </section>

            <section className={`settings ${settingsVisible ? "show" : "hide"}`}>
              <div className="settings-content">
                <nav className="settings-tabs">
                  <button
                    className={`settings-tab ${settingsTab === "themes" ? "active" : ""}`}
                    onClick={() => setSettingsTab("themes")}
                  >
                    Themes
                  </button>
                  <button
                    className={`settings-tab ${settingsTab === "components" ? "active" : ""}`}
                    onClick={() => setSettingsTab("components")}
                  >
                    Components
                  </button>
                  <button
                    className={`settings-tab ${settingsTab === "permissions" ? "active" : ""}`}
                    onClick={() => setSettingsTab("permissions")}
                  >
                    Permissions
                  </button>
                  <button
                    className={`settings-tab ${settingsTab === "developer" ? "active" : ""}`}
                    onClick={() => setSettingsTab("developer")}
                  >
                    Developer
                  </button>
                  <button
                    className={`settings-tab ${settingsTab === "about" ? "active" : ""}`}
                    onClick={() => setSettingsTab("about")}
                  >
                    About
                  </button>
                </nav>

                {settingsTab === "themes" && (
                  <div>
                    <h3>Installed Themes</h3>
                    <div className="theme-list">
                      {themes.map((theme) => (
                        <button
                          key={theme.name}
                          className={`theme-card ${currentTheme === theme.name ? "active" : ""}`}
                          onClick={() => void selectTheme(theme.name)}
                        >
                          <strong>{theme.name}</strong>
                        </button>
                      ))}
                    </div>
                    <h4>How To Add A Theme Preset</h4>
                    <ol>
                      <li>Add `Packages/Launchpad/public/themes/&lt;name&gt;.css` with only <code>:root</code> overrides of <code>--talos-*</code> tokens.</li>
                      <li>Rebuild Launchpad (<code>make frontend-build</code> or <code>npm run build</code> in Launchpad).</li>
                      <li>Select the preset here. See <code>docs/build-your-app/07-talos-ui-and-themes.md</code>.</li>
                    </ol>
                  </div>
                )}

                {settingsTab === "components" && (
                  <div>
                    <h3>Component Showcase</h3>
                    <div className="showcase-grid">
                      <div className="showcase-card">
                        <h4>Text</h4>
                        <h1>H1 Header</h1>
                        <h2>H2 Header</h2>
                        <h3>H3 Header</h3>
                        <p><strong>Bold</strong>, <em>Italic</em>, <span>Span text</span></p>
                      </div>
                      <div className="showcase-card">
                        <h4>Sidebar Tabs</h4>
                        <div className="sample-tabs">
                          <button className="icon-btn">A</button>
                          <button className="icon-btn active">B</button>
                        </div>
                      </div>
                      <div className="showcase-card"><h4>Buttons</h4><button className="ui-button">Button</button></div>
                      <div className="showcase-card"><h4>Button Group</h4><div className="button-group"><button className="ui-button">One</button><button className="ui-button">Two</button></div></div>
                      <div className="showcase-card"><h4>Checkbox</h4><label><input type="checkbox" /> Checked</label></div>
                      <div className="showcase-card"><h4>Toggle Switch</h4><label className="switch"><input type="checkbox" defaultChecked /><span /></label></div>
                      <div className="showcase-card"><h4>Search Bar</h4><input className="ui-input" placeholder="Search..." /></div>
                      <div className="showcase-card"><h4>Text Input</h4><input className="ui-input" defaultValue="Text input" /></div>
                      <div className="showcase-card"><h4>Dropdown</h4><select className="ui-input"><option>Inactive</option><option>Active</option></select></div>
                      <div className="showcase-card"><h4>Slider</h4><input type="range" /></div>
                      <div className="showcase-card"><h4>Double Slider</h4><input type="range" defaultValue={30} /><input type="range" defaultValue={75} /></div>
                      <div className="showcase-card"><h4>Progress</h4><progress max={100} value={45} /></div>
                      <div className="showcase-card"><h4>Context Menu</h4><div className="context-demo"><button className="ui-button">Open</button><button className="ui-button">Rename</button><button className="ui-button">Delete</button></div></div>
                    </div>
                  </div>
                )}

                {settingsTab === "permissions" && (
                  <div>
                    <h3>Permission grants</h3>
                    <p className="perm-hint">
                      Revoke clears a scope so the next request can prompt again. If you Allow or Deny from the popup,
                      retry the action in the app if needed.
                    </p>
                    <button type="button" className="ui-button perm-refresh" onClick={() => void reloadPermissionsSettings()}>
                      Refresh list
                    </button>
                    <div className="perm-table-wrap">
                      <table className="perm-table">
                        <thead>
                          <tr>
                            <th>App</th>
                            <th>Scope</th>
                            <th>State</th>
                            <th />
                          </tr>
                        </thead>
                        <tbody>
                          {permissionEntries.length === 0 ? (
                            <tr>
                              <td colSpan={4}>No explicit grants or denies yet (excluding built-in data scope).</td>
                            </tr>
                          ) : (
                            permissionEntries.map((row) => (
                              <tr key={`${row.app_id}:${row.scope}`}>
                                <td><code>{row.app_id}</code></td>
                                <td><code>{row.scope}</code></td>
                                <td>{row.granted ? "allowed" : "denied"}</td>
                                <td>
                                  <button
                                    type="button"
                                    className="ui-button"
                                    onClick={() => {
                                      void RevokePermission(row.app_id, row.scope).then(() => reloadPermissionsSettings());
                                    }}
                                  >
                                    Revoke
                                  </button>
                                </td>
                              </tr>
                            ))
                          )}
                        </tbody>
                      </table>
                    </div>
                    <h3>Permission audit log</h3>
                    <p className="perm-hint">Persisted decisions from <code>Temp/logs/permission_audit.jsonl</code>.</p>
                    <div className="perm-table-wrap">
                      <table className="perm-table">
                        <thead>
                          <tr>
                            <th>Time</th>
                            <th>Action</th>
                            <th>App</th>
                            <th>Scope</th>
                            <th>Granted</th>
                          </tr>
                        </thead>
                        <tbody>
                          {permissionAudit.length === 0 ? (
                            <tr>
                              <td colSpan={5}>No audit entries yet.</td>
                            </tr>
                          ) : (
                            permissionAudit.map((row, i) => (
                              <tr key={`${row.ts}-${row.app_id}-${row.scope}-${i}`}>
                                <td className="perm-ts">{row.ts}</td>
                                <td>{row.action}</td>
                                <td><code>{row.app_id}</code></td>
                                <td><code>{row.scope}</code></td>
                                <td>{row.granted ? "yes" : "no"}</td>
                              </tr>
                            ))
                          )}
                        </tbody>
                      </table>
                    </div>
                    <h3>Recent permission requests</h3>
                    <ul className="perm-history">
                      {permissionHistory.length === 0 ? (
                        <li>None yet.</li>
                      ) : (
                        permissionHistory.map((h) => (
                          <li key={`${h.ts}-${h.app_id}-${h.scope}`}>
                            <span className="perm-ts">{new Date(h.ts).toLocaleString()}</span>{" "}
                            <code>{h.app_id}</code> → <code>{h.scope}</code>
                            {h.reason ? <span className="perm-reason"> — {h.reason}</span> : null}
                          </li>
                        ))
                      )}
                    </ul>
                  </div>
                )}

                {settingsTab === "developer" && (
                  <div>
                    <h3>Developer mode</h3>
                    <p className="perm-hint">
                      When enabled, installed packages that declare <code>development.command</code> can run dev servers (for example{" "}
                      <code>npm run dev</code>) and use loopback dev URLs instead of only bundled static assets under{" "}
                      <code>/talos-pkg/</code>. Commands come from package manifests—enable only for packages you trust.
                    </p>
                    <label className="dev-mode-toggle">
                      <input
                        type="checkbox"
                        checked={developerMode}
                        onChange={(e) => {
                          const v = e.target.checked;
                          void SetDeveloperMode(v)
                            .then(() => {
                              setDeveloperMode(v);
                              return reloadCatalog();
                            })
                            .then(() =>
                              DevelopmentFeaturesEnabled().then((en) => setInstallUiEnabled(!!en)),
                            );
                        }}
                      />
                      <span>Enable developer mode (manifest dev commands)</span>
                    </label>
                    <p className="perm-hint">
                      Machine override: set environment variable <code>TALOS_DEV_MODE=1</code> to enable the same behavior without this toggle
                      (useful for automation).
                    </p>
                  </div>
                )}

                {settingsTab === "about" && (
                  <div>
                    <h3>About Talos Launchpad</h3>
                    <p>
                      Launchpad is the root frontend package. It lists installed apps, launches
                      app iframes, and brokers SDK bridge messages to the host.
                    </p>
                    <h4>Update channel (optional)</h4>
                    <p className="perm-hint">
                      Set <code>TALOS_UPDATE_CHANNEL</code> to a JSON URL (array of updates), or paste a URL below.
                    </p>
                    <div className="install-row install-row-split">
                      <input
                        className="ui-input"
                        placeholder="https://example.com/talos-channel.json"
                        value={updateChannelURL}
                        onChange={(e) => setUpdateChannelURL(e.target.value)}
                      />
                      <button
                        type="button"
                        className="ui-button"
                        onClick={() => {
                          void CheckForUpdates(updateChannelURL).then((rows) => {
                            if (!rows?.length) {
                              setUpdateCheckResult("No entries or empty URL.");
                              return;
                            }
                            setUpdateCheckResult(
                              rows.map((r) => `${r.app_id} ${r.version} → ${r.artifact_url}`).join("\n"),
                            );
                          }).catch((e: unknown) => setUpdateCheckResult(userFacingFetchError(e)));
                        }}
                      >
                        Check for updates
                      </button>
                    </div>
                    {updateCheckResult ? (
                      <pre className="update-check-out">{updateCheckResult}</pre>
                    ) : null}
                  </div>
                )}
              </div>
            </section>
          </>
        )}
      </section>
      {permissionPrompt && (
        <div
          className="permission-modal-backdrop"
          role="presentation"
          onClick={() => resolvePermissionPrompt(false)}
        >
          <div
            className="permission-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="perm-dialog-title"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 id="perm-dialog-title">Permission request</h3>
            <p>
              App <code>{permissionPrompt.app_id}</code> requests scope <code>{permissionPrompt.scope}</code>
            </p>
            {permissionPrompt.reason ? (
              <p className="perm-reason-block">{permissionPrompt.reason}</p>
            ) : null}
            <div className="perm-actions">
              <button
                type="button"
                className="ui-button"
                onClick={() => resolvePermissionPrompt(true)}
              >
                Allow
              </button>
              <button
                type="button"
                className="ui-button"
                onClick={() => resolvePermissionPrompt(false)}
              >
                Deny
              </button>
              <button type="button" className="ui-button" onClick={() => resolvePermissionPrompt(false)}>
                Dismiss
              </button>
            </div>
          </div>
        </div>
      )}

      {contextMenuState.visible && (
        <div
          className="context-menu"
          style={{
            left: `${contextMenuState.x}px`,
            top: `${contextMenuState.y}px`,
          }}
          onClick={(event) => event.stopPropagation()}
        >
          <div className="context-menu-title">{contextMenuState.title}</div>
          {contextMenuState.items.map((item) => (
            <button
              key={item.id}
              className="context-menu-item"
              disabled={item.disabled}
              onClick={() => {
                void item.action();
                hideContextMenu();
              }}
            >
              {item.label}
            </button>
          ))}
        </div>
      )}
    </main>
  );
}