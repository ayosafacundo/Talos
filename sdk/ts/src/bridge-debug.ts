/**
 * Opt-in diagnostics for Talos iframe / postMessage issues.
 *
 * Enable: `localStorage.setItem("talos_debug_bridge", "1")` then reload, or set
 * `globalThis.__TALOS_DEBUG_BRIDGE__ = true` before the iframe loads.
 *
 * Classifying "connection refused" to 127.0.0.1 in dev:
 * - Vite `@vite/client` or HMR WebSocket when HMR is enabled (mitigate with `server.hmr: false`).
 * - Main document or assets when the mini-app dev server is not running or the port is wrong.
 * - In-app `fetch`/`EventSource` to loopback when the sidecar API is down.
 */

export function talosBridgeDebugEnabled(): boolean {
  try {
    const g = globalThis as { __TALOS_DEBUG_BRIDGE__?: boolean }
    if (g.__TALOS_DEBUG_BRIDGE__ === true) return true
    if (typeof localStorage !== "undefined" && localStorage.getItem("talos_debug_bridge") === "1") {
      return true
    }
  } catch {
    /* ignore */
  }
  return false
}

export function talosBridgeDebugLog(direction: string, data: Record<string, unknown>): void {
  if (!talosBridgeDebugEnabled()) return
  try {
    console.info(`[talos-bridge] ${direction}`, data)
  } catch {
    /* ignore */
  }
}
