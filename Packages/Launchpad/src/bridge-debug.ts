/**
 * Opt-in Talos shell diagnostics (postMessage / iframe). Enable with
 * `localStorage.setItem("talos_debug_bridge", "1")` or `globalThis.__TALOS_DEBUG_BRIDGE__ = true`.
 *
 * See sdk/ts/src/bridge-debug.ts for classifying loopback "connection refused" in dev.
 */

export function talosBridgeDebugEnabled(): boolean {
  try {
    const g = globalThis as { __TALOS_DEBUG_BRIDGE__?: boolean };
    if (g.__TALOS_DEBUG_BRIDGE__ === true) return true;
    if (typeof localStorage !== "undefined" && localStorage.getItem("talos_debug_bridge") === "1") {
      return true;
    }
  } catch {
    /* ignore */
  }
  return false;
}

export function talosBridgeDebugLog(direction: string, data: Record<string, unknown>): void {
  if (!talosBridgeDebugEnabled()) return;
  try {
    console.info(`[talos-bridge] ${direction}`, data);
  } catch {
    /* ignore */
  }
}
