import { describe, expect, it } from "vitest"
import type { TalosTransport } from "./types"
import { TalosClient } from "./index"

function mockTransport(): TalosTransport {
  return {
    async saveState() {},
    async loadState() {
      return null
    },
    async sendMessage() {
      return null
    },
    async requestPermission() {
      return { granted: true, message: "ok" }
    },
    async resolvePath(_appId, rel) {
      return `/scoped/${rel}`
    },
  }
}

describe("TalosClient", () => {
  it("delegates resolvePath with app id", async () => {
    const t = mockTransport()
    const c = new TalosClient("app.test", t)
    const p = await c.resolvePath("file.txt")
    expect(p).toBe("/scoped/file.txt")
  })
})
