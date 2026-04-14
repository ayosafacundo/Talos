export const TALOS_THEME_CHANNEL = "talos:theme:v1" as const
export const TALOS_THEME_UPDATE_TYPE = "talos:theme:update" as const

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
  onThemeUpdate?: (payload: TalosThemeUpdatePayload) => void
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

function ensureStylesheet(id: string, href: string): void {
  let el = document.getElementById(id) as HTMLLinkElement | null
  if (!el) {
    el = document.createElement("link")
    el.rel = "stylesheet"
    el.id = id
    document.head.appendChild(el)
  }
  if (el.href !== href) {
    el.href = href
  }
}

export function applyTalosThemeRuntime(
  payload: TalosThemeUpdatePayload,
  options?: TalosThemeRuntimeOptions,
): void {
  const root = options?.root ?? document.documentElement
  root.dataset.talosTheme = payload.theme_name
  root.dataset.talosThemeVariant = payload.variant_id
  if (payload.theme_href) {
    ensureStylesheet("talos-theme-preset", payload.theme_href)
  }
  if (payload.components_css_href) {
    ensureStylesheet(options?.componentsLinkId ?? "talos-components-theme-variant", payload.components_css_href)
  }
  options?.onThemeUpdate?.(payload)
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
        components_css_href: String(q.get("_talos_components_href") || ""),
      }, options)
    }
  } catch {
    // Ignore malformed URL state.
  }

  const onMessage = (event: MessageEvent): void => {
    if (!isThemePayload(event.data)) return
    const payload = { ...event.data }
    if (payload.theme_href?.startsWith("/") && event.origin && event.origin !== "null") {
      payload.theme_href = `${event.origin}${payload.theme_href}`
    }
    if (payload.components_css_href?.startsWith("/") && event.origin && event.origin !== "null") {
      payload.components_css_href = `${event.origin}${payload.components_css_href}`
    }
    applyTalosThemeRuntime(payload, options)
  }
  window.addEventListener("message", onMessage)
  return () => window.removeEventListener("message", onMessage)
}
