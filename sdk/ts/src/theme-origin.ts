/** Align loopback theme asset URLs with the shell origin from `_talos_shell_origin` (see iframe bridge). */

function isLoopbackHostname(host: string): boolean {
  const h = String(host || "").toLowerCase().replace(/^\[|\]$/g, "")
  return h === "localhost" || h === "127.0.0.1" || h === "::1"
}

/**
 * If both the shell origin (from query) and href are http(s) loopback, rewrite href to use the shell's
 * origin (scheme/host/port) so `<link>` loads succeed when manifest/snapshot used127.0.0.1 vs localhost.
 */
export function normalizeLoopbackThemeHref(href: string, shellOrigin: string): string {
  const raw = String(href || "").trim()
  const shell = String(shellOrigin || "").trim()
  if (!raw || !shell) return raw
  try {
    const s = new URL(shell)
    const h = new URL(raw)
    if (s.protocol !== "http:" && s.protocol !== "https:") return raw
    if (h.protocol !== "http:" && h.protocol !== "https:") return raw
    if (!isLoopbackHostname(s.hostname) || !isLoopbackHostname(h.hostname)) return raw
    return `${s.origin}${h.pathname}${h.search}${h.hash}`
  } catch {
    return raw
  }
}
