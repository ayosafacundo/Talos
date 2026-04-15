// @vitest-environment jsdom
import { afterEach, describe, expect, it } from "vitest"
import { applyTalosThemeRuntime, TALOS_TOKENS_LINK_ID } from "./theme-runtime"

afterEach(() => {
  window.history.replaceState({}, "", "/")
})

describe("theme runtime", () => {
  it("applies theme markers, tokens, and component stylesheet", () => {
    const root = document.createElement("div")
    const payload = {
      channel: "talos:theme:v1" as const,
      type: "talos:theme:update" as const,
      theme_name: "dark",
      variant_id: "core.dark",
      theme_href: "http://127.0.0.1:5173/themes/dark.css",
      tokens_href: "http://127.0.0.1:5173/talos/tokens.css",
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
    const tokensLink = document.getElementById(TALOS_TOKENS_LINK_ID) as HTMLLinkElement
    expect(tokensLink).toBeTruthy()
    expect(tokensLink.getAttribute("href")).toBe("http://127.0.0.1:5173/talos/tokens.css")
  })

  it("rewrites loopback theme hrefs using _talos_shell_origin", () => {
    window.history.replaceState(
      {},
      "",
      "?_talos_shell_origin=" + encodeURIComponent("http://localhost:5173"),
    )
    const root = document.createElement("div")
    applyTalosThemeRuntime(
      {
        channel: "talos:theme:v1",
        type: "talos:theme:update",
        theme_name: "dark",
        variant_id: "core.dark",
        theme_href: "http://127.0.0.1:5173/themes/dark.css",
        tokens_href: "http://127.0.0.1:5173/talos/tokens.css",
      },
      { root },
    )
    const themeLink = document.getElementById("talos-theme-preset") as HTMLLinkElement
    expect(themeLink.getAttribute("href")).toBe("http://localhost:5173/themes/dark.css")
    const tokensLink = document.getElementById(TALOS_TOKENS_LINK_ID) as HTMLLinkElement
    expect(tokensLink.getAttribute("href")).toBe("http://localhost:5173/talos/tokens.css")
  })

  it("calls onStylesheetError when link fires error", () => {
    const errors: { id: string; href: string }[] = []
    const root = document.createElement("div")
    applyTalosThemeRuntime(
      {
        channel: "talos:theme:v1",
        type: "talos:theme:update",
        theme_name: "dark",
        variant_id: "core.dark",
        theme_href: "http://127.0.0.1:9/__nonexistent__.css",
      },
      {
        root,
        onStylesheetError: (d) => {
          errors.push(d)
        },
      },
    )
    const themeLink = document.getElementById("talos-theme-preset") as HTMLLinkElement
    themeLink?.dispatchEvent(new Event("error"))
    expect(errors.length).toBe(1)
    expect(errors[0].id).toBe("talos-theme-preset")
  })

  it("hydrates from URL snapshot on bind", async () => {
    window.history.replaceState(
      {},
      "",
      "?_talos_theme=dark&_talos_theme_variant=core.dark" +
        "&_talos_theme_href=http%3A%2F%2F127.0.0.1%3A34115%2Fthemes%2Fdark.css" +
        "&_talos_tokens_href=http%3A%2F%2F127.0.0.1%3A34115%2Ftalos%2Ftokens.css",
    )
    const { bindTalosThemeRuntime } = await import("./theme-runtime")
    const dispose = bindTalosThemeRuntime()
    expect(document.documentElement.dataset.talosTheme).toBe("dark")
    expect(document.documentElement.dataset.talosThemeVariant).toBe("core.dark")
    const tokensLink = document.getElementById(TALOS_TOKENS_LINK_ID) as HTMLLinkElement
    expect(tokensLink?.getAttribute("href")).toBe("http://127.0.0.1:34115/talos/tokens.css")
    dispose()
  })
})
