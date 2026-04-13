import { useEffect, useMemo, useRef, useState } from "react";
import {
  BroadcastMessage,
  DenyPermission,
  DevelopmentFeaturesEnabled,
  GetInstalledApps,
  GetThemes,
  GrantPermission,
  InstallPackageFromGitHub,
  InstallPackageFromURL,
  ListPermissionEntries,
  ListRepositoryPackages,
  LoadUserPrefs,
  PickZipAndInstall,
  SaveUserPrefs,
  GetStoreApps,
  GetStartupLaunchpad,
  LoadAppStateBase64,
  RequestPermissionDecision,
  ResolveScopedPath,
  RevokePermission,
  RouteMessage,
  SaveAppStateBase64,
  StartPackage,
  StopPackage,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import type { main } from "../wailsjs/go/models";
import {
  BRIDGE_CHANNEL,
  buildBridgeResponse,
  isAllowedMethod,
  parseBridgeRequest,
  resolveTrustedSender,
} from "./bridge";
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

type SettingsTab = "themes" | "components" | "permissions" | "about";

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

function withReload(url: string): string {
  if (!url) return "";
  const sep = url.includes("?") ? "&" : "?";
  return `${url}${sep}_talos_reload=${Date.now()}`;
}

function withBridgeToken(url: string, token: string): string {
  if (!url || !token) return url;
  const sep = url.includes("?") ? "&" : "?";
  return `${url}${sep}_talos_bt=${encodeURIComponent(token)}`;
}

function newBridgeToken(): string {
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`;
}

function postMessageTargetOrigin(event: MessageEvent): string {
  const o = event.origin;
  if (!o || o === "null") return "*";
  return o;
}

function normalizeWebOrigin(raw: string): string {
  const s = String(raw || "").trim();
  if (!s) return "";
  try {
    return new URL(s).origin;
  } catch {
    return s.replace(/\/$/, "");
  }
}

/** When the manifest lists http(s) origins, require incoming postMessage to match. */
function isMessageOriginAllowed(eventOrigin: string, allowed?: string[]): boolean {
  if (!allowed || allowed.length === 0) return true;
  const o = eventOrigin === "null" ? "" : normalizeWebOrigin(eventOrigin);
  if (!o) return false;
  return allowed.some((a) => normalizeWebOrigin(a) === o);
}

/** Target for host replies: use sender origin when allowlisted, else legacy behavior. */
function replyPostMessageTarget(event: MessageEvent, allowed?: string[]): string {
  if (!allowed || allowed.length === 0) {
    return postMessageTargetOrigin(event);
  }
  const o = event.origin;
  if (!o || o === "null") return "*";
  if (!isMessageOriginAllowed(o, allowed)) return "*";
  return normalizeWebOrigin(o);
}

/** Target origin when the parent posts into an app iframe (tighten when URL has a real origin). */
function postMessageTargetForIframe(iframe: HTMLIFrameElement): string {
  try {
    const u = new URL(iframe.src);
    if (u.origin && u.origin !== "null") return u.origin;
  } catch {
    /* ignore */
  }
  return "*";
}

function postMessageTargetForAppInstance(instance: AppInstance, iframe: HTMLIFrameElement): string {
  const allowed = instance.allowed_origins;
  if (allowed && allowed.length > 0) {
    try {
      const u = new URL(iframe.src);
      if (u.origin && u.origin !== "null") {
        const no = normalizeWebOrigin(u.origin);
        if (allowed.some((a) => normalizeWebOrigin(a) === no)) {
          return no;
        }
      }
    } catch {
      /* ignore */
    }
    return normalizeWebOrigin(allowed[0]);
  }
  return postMessageTargetForIframe(iframe);
}

function isLikelyAssetPath(value: string): boolean {
  return value.includes("/") || value.startsWith("./") || value.startsWith("../");
}

function iconURLForApp(app?: main.AppManifestView): string {
  if (!app) return "";
  const icon = String(app.icon || "");
  if (!icon) return "";
  if (icon.startsWith("http://") || icon.startsWith("https://") || icon.startsWith("file://") || icon.startsWith("data:")) {
    return icon;
  }
  if (!isLikelyAssetPath(icon)) return "";
  const appURL = String(app.url || "");
  if (!appURL.startsWith("file://")) return "";

  const marker = "/dist/";
  const idx = appURL.indexOf(marker);
  if (idx >= 0) {
    const packageRoot = appURL.slice(0, idx);
    return `${packageRoot}/${icon.replace(/^\/+/, "")}`;
  }
  try {
    return new URL(icon, appURL).toString();
  } catch {
    return "";
  }
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
  const [installUiEnabled, setInstallUiEnabled] = useState(false);
  const [installMessage, setInstallMessage] = useState("");
  const [installBusy, setInstallBusy] = useState(false);
  const [urlInstall, setUrlInstall] = useState("");
  const [ghOwner, setGhOwner] = useState("");
  const [ghRepo, setGhRepo] = useState("");
  const [ghRef, setGhRef] = useState("");
  const [repoBrowse, setRepoBrowse] = useState<main.RemotePackageDescriptor[]>([]);
  const [appContextOptions, setAppContextOptions] = useState<Record<string, AppMenuOption[]>>({});
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
  const packagesEventDebounceRef = useRef<number | undefined>(undefined);

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

  async function reloadPermissionEntries(): Promise<void> {
    const rows = await ListPermissionEntries();
    setPermissionEntries(rows ?? []);
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
    setActiveApps((prev) => {
      const next = prev.filter((app) => app.id !== instanceId);
      setFocusedAppId((prevFocused) => {
        if (prevFocused !== instanceId) return prevFocused;
        return next[0]?.id ?? null;
      });
      setLaunchpadVisible((prevVisible) => prevVisible || next.length === 0);
      return next;
    });
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
    await StartPackage(manifest.id);
    setActiveApps((prev) => {
      const existing = prev.find((app) => app.manifestId === manifest.id);
      if (existing) {
        setFocusedAppId(existing.id);
        setLaunchpadVisible(false);
        return prev;
      }
      const bridgeToken = newBridgeToken();
      const origins = manifest.allowed_origins;
      const next: AppInstance = {
        id: `${manifest.id}:${Date.now()}`,
        manifestId: manifest.id,
        name: manifest.name,
        icon: manifest.icon,
        url: withBridgeToken(withReload(manifest.url ?? ""), bridgeToken),
        bridgeToken,
        allowed_origins: origins && origins.length > 0 ? [...origins] : undefined,
        iframeEpoch: 0,
      };
      setFocusedAppId(next.id);
      setLaunchpadVisible(false);
      return [...prev, next];
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
      void reloadPermissionEntries();
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
        if (import.meta.env.DEV) {
          console.warn("[talos bridge] rejected:", trust.reason, { app_id: parsed.appId });
        }
        respond(false, null, `bridge rejected: ${trust.reason}`, parsed.requestId);
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
        void DevelopmentFeaturesEnabled().then((v) => setInstallUiEnabled(!!v));
        void ListRepositoryPackages().then((rows) => setRepoBrowse(rows ?? []));
      } catch (error) {
        if (mounted) setStartupError(String(error));
      }
    };

    const onDismissMenu = (): void => hideContextMenu();

    let offPackages: (() => void) | undefined;
    let pollTimer: number | undefined;
    const wailsDeadline = Date.now() + 120_000;

    const attachWails = (): boolean => {
      if (!wailsShellReady()) return false;
      void bootstrap();
      window.addEventListener("message", onMessage);
      window.addEventListener("click", onDismissMenu);
      window.addEventListener("scroll", onDismissMenu, true);
      offPackages = EventsOn("packages:event", (evt: Record<string, unknown>) => {
        if (packagesEventDebounceRef.current !== undefined) {
          window.clearTimeout(packagesEventDebounceRef.current);
        }
        packagesEventDebounceRef.current = window.setTimeout(() => {
          packagesEventDebounceRef.current = undefined;
          void (async () => {
            const installed = await reloadCatalog();
            const pid = String(evt?.package_id ?? "");
            if (!pid) {
              return;
            }
            setActiveApps((prev) =>
              prev.map((app) => {
                if (app.manifestId !== pid) {
                  return app;
                }
                const manifest = installed.find((m) => m.id === pid);
                const nextUrl = manifest
                  ? withBridgeToken(withReload(manifest.url ?? ""), app.bridgeToken)
                  : withReload(app.url);
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
    };
  }, [installedApps, launchableApps, storeApps, activeApps]);

  async function selectTheme(name: string): Promise<void> {
    applyTheme(name);
    setCurrentTheme(name);
    const prefs = await LoadUserPrefs();
    await SaveUserPrefs({
      ...prefs,
      theme: name,
    });
  }

  return (
    <main className="shell">
      <aside className="sidebar">
        <button
          className={`icon-btn ${launchpadVisible ? "active" : ""}`}
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
              key={app.id}
              className={`icon-btn ${focusedAppId === app.id ? "active" : ""}`}
              onClick={() => focusApp(app.id)}
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
          className={`icon-btn ${settingsVisible ? "active" : ""}`}
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
              <h2>Installed</h2>
              <div className="list">
                {launchableApps.map((app) => (
                  <button
                    key={app.id}
                    className="app-card"
                    onClick={() => void launchApp(app)}
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
                      <small>{app.description || "Installed Tiny App"}</small>
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
                {repoBrowse.length === 0
                  ? "No remote packages listed yet (stub registry)."
                  : `${repoBrowse.length} package(s) from repositories.`}
              </p>
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
                <iframe
                  key={`${app.id}:${app.iframeEpoch}`}
                  src={app.url}
                  title={app.name}
                  data-talos-manifest-id={app.manifestId}
                  data-talos-bridge-token={app.bridgeToken}
                  ref={(node) => {
                    iframeRefs.current[app.id] = node;
                  }}
                  style={{ display: focusedAppId === app.id ? "block" : "none" }}
                  sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
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
                    <button type="button" className="ui-button perm-refresh" onClick={() => void reloadPermissionEntries()}>
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
                                      void RevokePermission(row.app_id, row.scope).then(() => reloadPermissionEntries());
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

                {settingsTab === "about" && (
                  <div>
                    <h3>About Talos Launchpad</h3>
                    <p>
                      Launchpad is the root frontend package. It lists installed apps, launches
                      app iframes, and brokers SDK bridge messages to the host.
                    </p>
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
          onClick={() => setPermissionPrompt(null)}
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
            <p className="perm-hint">
              The first SDK call may have completed before you answer here. After Allow or Deny, retry the action in the app.
            </p>
            <div className="perm-actions">
              <button
                type="button"
                className="ui-button"
                onClick={() => {
                  void GrantPermission(permissionPrompt.app_id, permissionPrompt.scope)
                    .then(() => reloadPermissionEntries())
                    .finally(() => setPermissionPrompt(null));
                }}
              >
                Allow
              </button>
              <button
                type="button"
                className="ui-button"
                onClick={() => {
                  void DenyPermission(permissionPrompt.app_id, permissionPrompt.scope)
                    .then(() => reloadPermissionEntries())
                    .finally(() => setPermissionPrompt(null));
                }}
              >
                Deny
              </button>
              <button type="button" className="ui-button" onClick={() => setPermissionPrompt(null)}>
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