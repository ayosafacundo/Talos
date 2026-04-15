import { describe, expect, it } from "vitest"
import { normalizeLoopbackThemeHref } from "./theme-origin"

describe("normalizeLoopbackThemeHref", () => {
  it("rewrites loopback href to match http shell origin", () => {
    expect(
      normalizeLoopbackThemeHref(
        "http://127.0.0.1:5173/themes/dark.css",
        "http://localhost:5173",
      ),
    ).toBe("http://localhost:5173/themes/dark.css")
  })

  it("preserves path query and hash", () => {
    expect(
      normalizeLoopbackThemeHref(
        "http://127.0.0.1:8080/talos/tokens.css?v=1#x",
        "http://localhost:8080",
      ),
    ).toBe("http://localhost:8080/talos/tokens.css?v=1#x")
  })

  it("no-op for wails shell origin", () => {
    const href = "http://127.0.0.1:5173/themes/dark.css"
    expect(normalizeLoopbackThemeHref(href, "wails://wails.localhost:34115")).toBe(href)
  })

  it("no-op when href is not loopback", () => {
    const href = "https://example.com/t.css"
    expect(normalizeLoopbackThemeHref(href, "http://localhost:5173")).toBe(href)
  })

  it("no-op when shell or href empty", () => {
    expect(normalizeLoopbackThemeHref("http://127.0.0.1:1/x", "")).toBe("http://127.0.0.1:1/x")
    expect(normalizeLoopbackThemeHref("", "http://localhost:1")).toBe("")
  })
})
