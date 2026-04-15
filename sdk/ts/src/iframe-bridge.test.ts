import { describe, expect, it } from "vitest"
import {
  computeParentPostMessageTarget,
  parseShellOriginFromSearch,
} from "./iframe-bridge"

describe("computeParentPostMessageTarget", () => {
  it("uses first ancestor that is not the iframe (Wails + Vite)", () => {
    expect(
      computeParentPostMessageTarget("http://127.0.0.1:5175", [
        "wails://wails.localhost:34115",
        "http://127.0.0.1:5175",
      ]),
    ).toBe("wails://wails.localhost:34115")
  })

  it("handles self listed before parent (quirky order)", () => {
    expect(
      computeParentPostMessageTarget("http://127.0.0.1:5175", [
        "http://127.0.0.1:5175",
        "wails://wails.localhost:34115",
      ]),
    ).toBe("wails://wails.localhost:34115")
  })

  it("single real parent", () => {
    expect(
      computeParentPostMessageTarget("http://127.0.0.1:5175", ["wails://wails.localhost:34115"]),
    ).toBe("wails://wails.localhost:34115")
  })

  it("falls back to star when every entry matches self", () => {
    expect(computeParentPostMessageTarget("http://127.0.0.1:5175", ["http://127.0.0.1:5175"])).toBe("*")
  })

  it("falls back to star when list empty", () => {
    expect(computeParentPostMessageTarget("http://127.0.0.1:5175", [])).toBe("*")
  })

  it("falls back to star when self origin is empty (skip ancestor heuristic)", () => {
    expect(
      computeParentPostMessageTarget("", ["http://127.0.0.1:5175", "wails://wails.localhost:34115"]),
    ).toBe("*")
  })
})

describe("parseShellOriginFromSearch", () => {
  it("decodes shell origin from Launchpad-injected query", () => {
    const q =
      "?_talos_bt=xxxxxxxxxxxxxxxxxxxx&_talos_shell_origin=" +
      encodeURIComponent("wails://wails.localhost:34115")
    expect(parseShellOriginFromSearch(q)).toBe("wails://wails.localhost:34115")
  })
})
