import fs from "node:fs"
import path from "node:path"
import type { Plugin } from "vite"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"

/** Strip crossorigin from emitted index.html so wails:// /talos-pkg/ loads in embedded WebKit. */
function stripCrossoriginForTalosWebView(): Plugin {
  return {
    name: "talos-strip-crossorigin",
    apply: "build",
    writeBundle(options) {
      const dir = options.dir
      if (!dir) return
      const htmlPath = path.join(dir, "index.html")
      if (!fs.existsSync(htmlPath)) return
      const html = fs.readFileSync(htmlPath, "utf8")
      const next = html.replace(/\s+crossorigin(?:="[^"]*"|='[^']*'|)?/gi, "")
      if (next !== html) fs.writeFileSync(htmlPath, next, "utf8")
    },
  }
}

export default defineConfig({
  base: "./",
  plugins: [react(), stripCrossoriginForTalosWebView()],
  resolve: {
    alias: {
      react: path.resolve(__dirname, "node_modules/react"),
      "react-dom": path.resolve(__dirname, "node_modules/react-dom"),
    },
  },
  server: {
    host: "127.0.0.1",
    port: Number(process.env.TALOS_DEV_SERVER_PORT) || 5175,
  },
  build: {
    outDir: "../dist",
    emptyOutDir: true,
    rollupOptions: {
      input: "index.html",
      output: {
        entryFileNames: "app.js",
        assetFileNames: "app.css",
      },
    },
  },
})
