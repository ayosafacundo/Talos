/**
 * Browser iframe transport for Talos host bridge (postMessage v1).
 * Reads `_talos_bt` from the page URL (injected when the host loads the iframe).
 *
 * See docs/dev/iframe-bridge.md for origin and allowlist rules shared with the host.
 */

import { talosBridgeDebugLog } from "./bridge-debug"
import type { ContextMenuOption, PermissionResult, TalosTransport } from "./types";

export const BRIDGE_CHANNEL = "talos:sdk:v1" as const;

let warnedBridgeTokenWithoutShellOrigin = false

function normalizeOrigin(raw: string): string {
  const s = String(raw || "").trim()
  if (!s) return ""
  try {
    return new URL(s).origin
  } catch {
    return s.replace(/\/$/, "")
  }
}

/**
 * Computes targetOrigin for iframe → parent postMessage.
 * MDN: ancestorOrigins is ordered parent → root. Some engines append the iframe origin; using
 * `item(length - 1)` picked the wrong entry under Wails (e.g. http://127.0.0.1:5175) so the
 * shell (wails://…) rejected the message.
 */
export function computeParentPostMessageTarget(selfOrigin: string, ancestorOrigins: readonly string[]): string {
  const selfO = normalizeOrigin(selfOrigin)
  // Empty iframe origin (opaque / exotic contexts): ancestor scan can pick the wrong entry.
  if (!selfO) {
    return "*"
  }
  for (const o of ancestorOrigins) {
    if (!o || o === "null") continue
    if (normalizeOrigin(o) !== selfO) return o
  }
  return "*"
}

/** Parse `_talos_shell_origin` from a query string (used in tests). */
export function parseShellOriginFromSearch(search: string): string {
  try {
    const q = new URLSearchParams(search.startsWith("?") ? search.slice(1) : search)
    return String(q.get("_talos_shell_origin") || "").trim()
  } catch {
    return ""
  }
}

function shellOriginFromQuery(): string {
  try {
    return parseShellOriginFromSearch(window.location.search)
  } catch {
    return ""
  }
}

/** Prefer the real parent origin; fall back to "*" (e.g. file:// or empty ancestorOrigins). */
export function parentPostMessageTarget(): string {
  const fromShell = shellOriginFromQuery()
  const bt = bridgeTokenFromLocation()
  if (!fromShell && bt.length >= 16 && !warnedBridgeTokenWithoutShellOrigin) {
    warnedBridgeTokenWithoutShellOrigin = true
    try {
      console.warn(
        "[@talos/sdk] iframe bridge: _talos_bt is set but _talos_shell_origin is missing. " +
          "Host should inject both (see Talos Launchpad withBridgeToken). Falling back to ancestorOrigins or *.",
      )
    } catch {
      /* ignore */
    }
  }
  if (fromShell) {
    talosBridgeDebugLog("iframe_to_parent_target", {
      target: fromShell,
      via: "_talos_shell_origin",
    })
    return fromShell
  }
  let selfOrigin = ""
  try {
    selfOrigin = window.location.origin
  } catch {
    return "*"
  }
  const list: string[] = []
  try {
    const ao = (window.location as unknown as { ancestorOrigins?: DOMStringList }).ancestorOrigins
    if (ao && ao.length > 0) {
      for (let i = 0; i < ao.length; i++) {
        const o = ao.item(i)
        if (o) list.push(o)
      }
    }
  } catch {
    return "*"
  }
  const target = computeParentPostMessageTarget(selfOrigin, list)
  talosBridgeDebugLog("iframe_to_parent_target", {
    target,
    selfOrigin,
    ancestorOrigins: list,
    hasShellOriginQuery: Boolean(fromShell),
  })
  return target
}

export function bridgeTokenFromLocation(): string {
  try {
    const q = new URLSearchParams(window.location.search);
    return String(q.get("_talos_bt") || "").trim();
  } catch {
    return "";
  }
}

export class IframeBridgeTransport implements TalosTransport {
  private readonly appId: string
  private readonly pending = new Map<
    string,
    { resolve: (v: unknown) => void; reject: (e: Error) => void }
  >()
  private readonly bridgeToken: string
  private readonly onMessage: (event: MessageEvent) => void

  constructor(appId: string, bridgeToken?: string) {
    this.appId = appId
    const token = (bridgeToken ?? "").trim() || bridgeTokenFromLocation()
    if (token.length < 16) {
      throw new Error(
        "Talos iframe bridge: missing _talos_bt in URL (host must inject bridge token when opening the app)",
      )
    }
    this.bridgeToken = token
    this.onMessage = (event: MessageEvent): void => {
      const msg = event.data as Record<string, unknown> | null
      if (!msg || msg.type !== "talos:sdk:res") return
      if (msg.channel && msg.channel !== BRIDGE_CHANNEL) return
      const requestId = String(msg.request_id || "")
      const pending = this.pending.get(requestId)
      if (!pending) return
      this.pending.delete(requestId)
      if (!msg.ok) {
        pending.reject(new Error(String(msg.error || "sdk request failed")))
        return
      }
      pending.resolve(msg.result ?? {})
    }
    window.addEventListener("message", this.onMessage)
  }

  dispose(): void {
    window.removeEventListener("message", this.onMessage)
    this.pending.clear()
  }

  private call(method: string, params: Record<string, unknown>, timeoutMs = 30_000): Promise<unknown> {
    const requestId =
      typeof crypto !== "undefined" && crypto.randomUUID
        ? crypto.randomUUID()
        : `${Date.now()}-${Math.random().toString(36).slice(2)}`
    const req = {
      channel: BRIDGE_CHANNEL,
      type: "talos:sdk:req" as const,
      request_id: requestId,
      app_id: this.appId,
      bridge_token: this.bridgeToken,
      method,
      params,
    }
    return new Promise((resolve, reject) => {
      this.pending.set(requestId, { resolve, reject })
      const pmTarget = parentPostMessageTarget()
      talosBridgeDebugLog("iframe_to_parent_postMessage", { method, targetOrigin: pmTarget })
      window.parent.postMessage(req, pmTarget)
      if (timeoutMs > 0) {
        window.setTimeout(() => {
          if (!this.pending.has(requestId)) return
          this.pending.delete(requestId)
          reject(new Error(`timeout waiting for ${method}`))
        }, timeoutMs)
      }
    })
  }

  async saveState(_appId: string, data: Uint8Array): Promise<void> {
    void _appId
    const b64 = btoa(String.fromCharCode(...data))
    await this.call("saveState", { data_base64: b64 })
  }

  async loadState(_appId: string): Promise<Uint8Array | null> {
    void _appId
    const result = (await this.call("loadState", {})) as { data_base64?: string }
    const b64 = String(result.data_base64 || "")
    if (!b64) return null
    const raw = atob(b64)
    return Uint8Array.from(raw, (c) => c.charCodeAt(0))
  }

  async sendMessage(targetID: string, payload: Uint8Array): Promise<Uint8Array | null> {
    const text = new TextDecoder().decode(payload)
    const result = (await this.call("sendMessage", {
      target_id: targetID,
      payload: text,
      type: "pkg:msg",
    })) as { payload?: string }
    const out = String(result.payload || "")
    return new TextEncoder().encode(out)
  }

  async requestPermission(scope: string, reason?: string): Promise<PermissionResult> {
    const result = (await this.call("requestPermission", { scope, reason: reason ?? "" }, 0)) as {
      granted?: boolean
      message?: string
    }
    return {
      granted: Boolean(result.granted),
      message: String(result.message || ""),
    }
  }

  async resolvePath(appId: string, relativePath: string): Promise<string> {
    void appId
    const result = (await this.call("resolvePath", { relative_path: relativePath })) as {
      resolved_path?: string
    }
    return String(result.resolved_path || "")
  }

  async readScopedText(appId: string, relativePath: string): Promise<{ found: boolean; text: string }> {
    void appId
    const result = (await this.call("readScopedText", { relative_path: relativePath })) as {
      found?: boolean
      text?: string
    }
    return {
      found: Boolean(result.found),
      text: String(result.text || ""),
    }
  }

  async writeScopedText(appId: string, relativePath: string, text: string): Promise<void> {
    void appId
    await this.call("writeScopedText", { relative_path: relativePath, text })
  }

  async packageSdkLog(appId: string, level: string, message: string): Promise<void> {
    void appId
    await this.call("packageSdkLog", {
      level: String(level || "INFO").trim(),
      message: String(message || ""),
    })
  }

  /** Proxies GET/POST to the app's loopback sidecar via the host (path must start with /api/). */
  async packageLocalHttp(
    appId: string,
    method: string,
    path: string,
    body: string,
  ): Promise<{ status: number; content_type: string; body: string; body_base64?: string }> {
    void appId
    const result = (await this.call("packageLocalHttp", {
      method: String(method || "GET").toUpperCase(),
      path: String(path || ""),
      body: String(body || ""),
    })) as { status?: number; content_type?: string; body?: string; body_base64?: string }
    const b64 = String(result.body_base64 || "").trim()
    if (b64) {
      const raw = atob(b64)
      return {
        status: Number(result.status ?? 0),
        content_type: String(result.content_type || ""),
        body: raw,
        body_base64: b64,
      }
    }
    return {
      status: Number(result.status ?? 0),
      content_type: String(result.content_type || ""),
      body: String(result.body ?? ""),
    }
  }

  async setContextMenuOptions(appId: string, options: ContextMenuOption[]): Promise<void> {
    void appId
    await this.call("setContextMenuOptions", {
      options: options.map((o) => ({ id: o.id, label: o.label })),
    })
  }

  async clearContextMenuOptions(appId: string): Promise<void> {
    void appId
    await this.call("clearContextMenuOptions", {})
  }

  async openContextMenu(appId: string, x?: number, y?: number): Promise<void> {
    void appId
    await this.call("openContextMenu", { x: x ?? 0, y: y ?? 0 })
  }
}
