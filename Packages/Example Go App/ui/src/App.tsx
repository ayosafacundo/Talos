import { useMemo, useState } from "react"
import { createTalosApp } from "@talos/web-components"
import {
  TalosAlert,
  TalosButton,
  TalosCard,
  TalosListRow,
  TalosPanel,
} from "@talos/web-components/react"

const APP_ID = "app.example.go"
const STATUS_FILE = "example_go_status.json"

function App() {
  const talos = useMemo(() => createTalosApp(APP_ID), [])
  const [hubState, setHubState] = useState<string>("—")
  const [sidecarJSON, setSidecarJSON] = useState<string>("—")
  const [uiNote, setUiNote] = useState<string>("Ready")

  async function refreshHubState(): Promise<void> {
    try {
      const raw = await talos.client.loadState()
      if (!raw || raw.byteLength === 0) {
        setHubState("(empty — Go saves a timestamp here on each launch)")
        setUiNote("Hub state loaded (empty).")
        return
      }
      setHubState(new TextDecoder().decode(raw))
      setUiNote("Hub state loaded from SaveState / LoadState (Go ↔ gRPC).")
    } catch (err) {
      setUiNote(`Hub state failed: ${(err as Error).message}`)
    }
  }

  async function refreshSidecarStatus(): Promise<void> {
    try {
      const read = await talos.loadText(STATUS_FILE)
      if (!read.found) {
        setSidecarJSON("(file not found yet — waiting for Go sidecar)")
        setUiNote("Status file missing; ensure the binary is running.")
        return
      }
      setSidecarJSON(read.text.trim())
      setUiNote("Read scoped JSON written by the Go process (ReadScopedFile / WriteScopedFile on host).")
    } catch (err) {
      setUiNote(`Status read failed: ${(err as Error).message}`)
    }
  }

  async function appendHubStateFromUI(): Promise<void> {
    try {
      const stamp = new Date().toISOString()
      const prev = await talos.client.loadState()
      let base = ""
      if (prev && prev.byteLength > 0) {
        base = new TextDecoder().decode(prev).trimEnd() + "\n"
      }
      const next = base + `ui-touch@${stamp}\n`
      await talos.client.saveState(new TextEncoder().encode(next))
      setUiNote(`Appended a line to hub state (${stamp}).`)
      await refreshHubState()
    } catch (err) {
      setUiNote(`saveState from UI failed: ${(err as Error).message}`)
    }
  }

  return (
    <main className="app-root talos-p-4 talos-grid" style={{ minHeight: "100vh", placeItems: "center" }}>
      <TalosPanel style={{ width: "min(900px, 100%)" }}>
        <TalosAlert>
          <span slot="title">Go + Talos SDK</span>
          The <strong>Go binary</strong> uses the official gRPC client: structured errors,{" "}
          <code className="talos-text-secondary">context</code> cancellation, JSON over the hub, and a ticker loop. This{" "}
          <strong>React UI</strong> uses the same Talos web components and theme tokens as the TypeScript example — bridging
          Go and TypeScript through hub state and scoped files.
        </TalosAlert>

        <TalosCard>
          <h1 className="talos-text-primary talos-mb-2">Example Go app</h1>
          <p className="talos-text-secondary talos-mb-3">
            Sidecar demonstrates LoadState, SaveState, RequestPermission, SendMessage, Broadcast, ResolvePath, ReadScopedFile,
            WriteScopedFile, and AppendPackageLog (visible when Development mode is on for this package).
          </p>
          <div className="talos-flex talos-gap-2 talos-wrap">
            <TalosButton onClick={() => void refreshHubState()}>Load hub state</TalosButton>
            <TalosButton onClick={() => void refreshSidecarStatus()}>Load sidecar status JSON</TalosButton>
            <TalosButton variant="ghost" onClick={() => void appendHubStateFromUI()}>
              Append line from UI (saveState)
            </TalosButton>
          </div>
        </TalosCard>

        <TalosCard>
          <TalosListRow>
            <span slot="leading">◎</span>
            Hub state (UTF-8)
            <span slot="meta" className="talos-text-secondary" style={{ maxWidth: "min(420px, 45vw)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
              {hubState}
            </span>
          </TalosListRow>
        </TalosCard>

        <TalosCard>
          <h2 className="talos-text-primary talos-mb-2 talos-text-body-sm" style={{ fontWeight: 600 }}>
            Sidecar status (<code>{STATUS_FILE}</code>)
          </h2>
          <pre
            className="talos-text-secondary talos-m-0 talos-p-3"
            style={{
              fontSize: "0.8rem",
              lineHeight: 1.45,
              overflow: "auto",
              maxHeight: "280px",
              borderRadius: "var(--talos-radius-md, 12px)",
              border: "1px solid var(--talos-component-border)",
              background: "var(--talos-color-bg-secondary, rgba(0,0,0,0.04))",
            }}
          >
            {sidecarJSON}
          </pre>
        </TalosCard>

        <TalosCard>
          <TalosListRow>
            <span slot="leading">i</span>
            UI status
            <span slot="meta">{uiNote}</span>
          </TalosListRow>
        </TalosCard>
      </TalosPanel>
    </main>
  )
}

export default App
