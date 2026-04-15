import { bindTalosThemeRuntime, IframeBridgeTransport, TalosClient } from "../ts/src/index.ts"
import { registerTalosWebComponents } from "./talos-web-components.js"

function normalizeRelativePath(path) {
  return String(path || "").replace(/\\/g, "/").trim()
}

function isOutsideDataScope(path) {
  const rel = normalizeRelativePath(path)
  if (!rel) return false
  if (rel.startsWith("/")) return true
  return rel.split("/").some((part) => part === "..")
}

function pathDirectory(path) {
  const rel = normalizeRelativePath(path)
  if (!rel) return "."
  const idx = rel.lastIndexOf("/")
  return idx <= 0 ? rel : rel.slice(0, idx)
}

async function ensureExternalPermission(client, relativePath, operation) {
  if (!isOutsideDataScope(relativePath)) return
  const dir = pathDirectory(relativePath)
  const reason = `App requests ${operation} outside scoped data folder (path: ${dir}).`
  const decision = await client.requestPermission("fs:external", reason)
  if (!decision.granted) {
    throw new Error(`Filesystem permission denied: ${decision.message || "user denied"}`)
  }
}

export function createTalosApp(appID) {
  const client = new TalosClient(appID, new IframeBridgeTransport(appID))
  let resolveThemeReady
  const themeReady = new Promise((resolve) => {
    resolveThemeReady = resolve
  })
  const disposeThemeRuntime = bindTalosThemeRuntime({
    onThemeUpdate() {
      if (resolveThemeReady) {
        resolveThemeReady()
        resolveThemeReady = null
      }
    },
  })
  window.addEventListener("pagehide", disposeThemeRuntime, { once: true })

  return {
    client,
    registerTalosWebComponents,
    themeReady,
    dispose() {
      disposeThemeRuntime()
    },
    async ensureFilesystemPermission(relativePath = "data/") {
      await ensureExternalPermission(client, relativePath, "filesystem access")
    },
    async saveText(relativePath, text) {
      await ensureExternalPermission(client, relativePath, "write")
      await client.writeScopedText(relativePath, String(text))
    },
    async loadText(relativePath) {
      await ensureExternalPermission(client, relativePath, "read")
      return client.readScopedText(relativePath)
    },
    /** Proxies to the package binary loopback API through the Talos host (path must start with /api/). */
    packageLocalHttp(method, path, body = "") {
      return client.packageLocalHttp(method, path, body)
    },
  }
}

export { registerTalosWebComponents }
