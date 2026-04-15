import { talosBridgeDebugEnabled } from "./bridge-debug"
import { parseShellOriginFromSearch } from "./iframe-bridge"
import { normalizeLoopbackThemeHref } from "./theme-origin"

export const TALOS_THEME_CHANNEL = "talos:theme:v1" as const
export const TALOS_THEME_UPDATE_TYPE = "talos:theme:update" as const

export const TALOS_TOKENS_LINK_ID = "talos-tokens" as const

export type TalosThemeUpdatePayload = {
  channel: typeof TALOS_THEME_CHANNEL
  type: typeof TALOS_THEME_UPDATE_TYPE
  theme_name: string
  variant_id: string
  theme_href?: string
  tokens_href?: string
  components_css_href?: string
}

export type TalosThemeRuntimeOptions = {
  root?: HTMLElement
  componentsLinkId?: string
  tokensLinkId?: string
  onThemeUpdate?: (payload: TalosThemeUpdatePayload) => void
  /** Called when a theme `<link>` fails to load (network / wrong origin). */
  onStylesheetError?: (detail: { id: string; href: string }) => void
}

function shellOriginFromIframeSearch(): string {
  try {
    return parseShellOriginFromSearch(window.location.search)
  } catch {
    return ""
  }
}

function normalizeThemeHrefs(
  payload: TalosThemeUpdatePayload,
  shellOrigin: string,
): TalosThemeUpdatePayload {
  const out = { ...payload }
  if (out.theme_href) {
    out.theme_href = normalizeLoopbackThemeHref(out.theme_href, shellOrigin)
  }
  if (out.tokens_href) {
    out.tokens_href = normalizeLoopbackThemeHref(out.tokens_href, shellOrigin)
  }
  if (out.components_css_href) {
    out.components_css_href = normalizeLoopbackThemeHref(out.components_css_href, shellOrigin)
  }
  return out
}

function isThemePayload(data: unknown): data is TalosThemeUpdatePayload {
  if (!data || typeof data !== "object") return false
  const obj = data as Record<string, unknown>
  return (
    obj.channel === TALOS_THEME_CHANNEL &&
    obj.type === TALOS_THEME_UPDATE_TYPE &&
    typeof obj.theme_name === "string" &&
    typeof obj.variant_id === "string"
  )
}

function ensureStylesheet(id: string, href: string, options?: TalosThemeRuntimeOptions): void {
  let el = document.getElementById(id) as HTMLLinkElement | null
  if (!el) {
    el = document.createElement("link")
    el.rel = "stylesheet"
    el.id = id
    document.head.appendChild(el)
  }
  try {
    const next = new URL(href, document.baseURI).href
    if (el.href !== next) {
      el.href = href
    }
  } catch {
    if (el.getAttribute("href") !== href) {
      el.setAttribute("href", href)
    }
  }
  const onErr = (): void => {
    const resolved = el?.href || href
    const detail = { id, href: resolved }
    options?.onStylesheetError?.(detail)
    if (talosBridgeDebugEnabled()) {
      console.warn("[@talos/sdk] theme stylesheet failed to load", detail)
    }
  }
  el.onerror = onErr
}

export function applyTalosThemeRuntime(
  payload: TalosThemeUpdatePayload,
  options?: TalosThemeRuntimeOptions,
): void {
  const shellOrigin = shellOriginFromIframeSearch()
  const p = normalizeThemeHrefs(payload, shellOrigin)
  const root = options?.root ?? document.documentElement
  root.dataset.talosTheme = p.theme_name
  root.dataset.talosThemeVariant = p.variant_id
  if (p.theme_href) {
    ensureStylesheet("talos-theme-preset", p.theme_href, options)
  }
  if (p.tokens_href) {
    ensureStylesheet(options?.tokensLinkId ?? TALOS_TOKENS_LINK_ID, p.tokens_href, options)
  }
  if (p.components_css_href) {
    ensureStylesheet(options?.componentsLinkId ?? "talos-components-theme-variant", p.components_css_href, options)
  }
  options?.onThemeUpdate?.(p)
}

export function bindTalosThemeRuntime(options?: TalosThemeRuntimeOptions): () => void {
  try {
    const q = new URLSearchParams(window.location.search)
    const themeName = String(q.get("_talos_theme") || "").trim()
    const variantID = String(q.get("_talos_theme_variant") || "").trim()
    if (themeName && variantID) {
      applyTalosThemeRuntime({
        channel: TALOS_THEME_CHANNEL,
        type: TALOS_THEME_UPDATE_TYPE,
        theme_name: themeName,
        variant_id: variantID,
        theme_href: String(q.get("_talos_theme_href") || ""),
        tokens_href: String(q.get("_talos_tokens_href") || ""),
        components_css_href: String(q.get("_talos_components_href") || ""),
      }, options)
    }
  } catch {
    // Ignore malformed URL state.
  }

  const onMessage = (event: MessageEvent): void => {
    if (!isThemePayload(event.data)) return
    const raw = event.data as Record<string, unknown>
    // Plain object (not spread of possibly frozen structured-clone payloads) so WebKit allows href rewrites.
    const payload: TalosThemeUpdatePayload = {
      channel: TALOS_THEME_CHANNEL,
      type: TALOS_THEME_UPDATE_TYPE,
      theme_name: String(raw.theme_name ?? ""),
      variant_id: String(raw.variant_id ?? ""),
      theme_href: raw.theme_href != null ? String(raw.theme_href) : undefined,
      tokens_href: raw.tokens_href != null ? String(raw.tokens_href) : undefined,
      components_css_href: raw.components_css_href != null ? String(raw.components_css_href) : undefined,
    }
    if (payload.theme_href?.startsWith("/") && event.origin && event.origin !== "null") {
      payload.theme_href = `${event.origin}${payload.theme_href}`
    }
    if (payload.tokens_href?.startsWith("/") && event.origin && event.origin !== "null") {
      payload.tokens_href = `${event.origin}${payload.tokens_href}`
    }
    if (payload.components_css_href?.startsWith("/") && event.origin && event.origin !== "null") {
      payload.components_css_href = `${event.origin}${payload.components_css_href}`
    }
    applyTalosThemeRuntime(payload, options)
  }
  window.addEventListener("message", onMessage)
  return () => window.removeEventListener("message", onMessage)
}
