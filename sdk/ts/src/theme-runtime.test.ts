// @vitest-environment jsdom
import { describe, expect, it } from "vitest"
import { applyTalosThemeRuntime } from "./theme-runtime"

describe("theme runtime", () => {
  it("applies theme markers and component stylesheet", () => {
    const root = document.createElement("div")
    const payload = {
      channel: "talos:theme:v1" as const,
      type: "talos:theme:update" as const,
      theme_name: "dark",
      variant_id: "core.dark",
      theme_href: "http://127.0.0.1:5173/themes/dark.css",
      components_css_href: "http://127.0.0.1:5174/theme-assets/core.dark.components.css",
    }
    applyTalosThemeRuntime(payload, {
      root,
      componentsLinkId: "talos-theme-css",
    })
    expect(root.dataset.talosTheme).toBe("dark")
    expect(root.dataset.talosThemeVariant).toBe("core.dark")
    const link = document.getElementById("talos-theme-css") as HTMLLinkElement
    expect(link).toBeTruthy()
    expect(link.getAttribute("href")).toBe("http://127.0.0.1:5174/theme-assets/core.dark.components.css")
    const themeLink = document.getElementById("talos-theme-preset") as HTMLLinkElement
    expect(themeLink).toBeTruthy()
    expect(themeLink.getAttribute("href")).toBe("http://127.0.0.1:5173/themes/dark.css")
  })

  it("hydrates from URL snapshot on bind", async () => {
    window.history.replaceState(
      {},
      "",
      "?_talos_theme=dark&_talos_theme_variant=core.dark&_talos_theme_href=http%3A%2F%2F127.0.0.1%3A34115%2Fthemes%2Fdark.css",
    )
    const { bindTalosThemeRuntime } = await import("./theme-runtime")
    const dispose = bindTalosThemeRuntime()
    expect(document.documentElement.dataset.talosTheme).toBe("dark")
    expect(document.documentElement.dataset.talosThemeVariant).toBe("core.dark")
    dispose()
  })
})
