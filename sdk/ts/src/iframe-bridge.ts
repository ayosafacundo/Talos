/**
 * Browser iframe transport for Talos host bridge (postMessage v1).
 * Reads `_talos_bt` from the page URL (injected when the host loads the iframe).
 */

import type { ContextMenuOption, PermissionResult, TalosTransport } from "./types";

export const BRIDGE_CHANNEL = "talos:sdk:v1" as const;

function bridgeTokenFromLocation(): string {
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

  private call(method: string, params: Record<string, unknown>): Promise<unknown> {
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
      window.parent.postMessage(req, "*")
      window.setTimeout(() => {
        if (!this.pending.has(requestId)) return
        this.pending.delete(requestId)
        reject(new Error(`timeout waiting for ${method}`))
      }, 30_000)
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
    const result = (await this.call("requestPermission", { scope, reason: reason ?? "" })) as {
      granted?: boolean
      message?: string
    }
    return {
      granted: Boolean(result.granted),
      message: String(result.message || ""),
    }
  }

  async resolvePath(appId: string, relativePath: string): Promise<string> {
    const result = (await this.call("resolvePath", { relative_path: relativePath })) as {
      resolved_path?: string
    }
    return String(result.resolved_path || "")
  }

  async setContextMenuOptions(appId: string, options: ContextMenuOption[]): Promise<void> {
    await this.call("setContextMenuOptions", {
      options: options.map((o) => ({ id: o.id, label: o.label })),
    })
  }

  async clearContextMenuOptions(appId: string): Promise<void> {
    await this.call("clearContextMenuOptions", {})
  }

  async openContextMenu(appId: string, x?: number, y?: number): Promise<void> {
    await this.call("openContextMenu", { x: x ?? 0, y: y ?? 0 })
  }
}
