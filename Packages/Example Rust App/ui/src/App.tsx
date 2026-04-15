import { useMemo, useState } from "react"
import { createTalosApp } from "@talos/web-components"
import {
  TalosAlert,
  TalosButton,
  TalosCard,
  TalosListRow,
  TalosPanel,
} from "@talos/web-components/react"

const APP_ID = "app.example.rust"
const STATUS_FILE = "example_rust_status.json"

function iframeDeliveryMode(): { label: string; detail: string } {
  const href = window.location.href
  if (href.includes("/talos-pkg/")) {
    return {
      label: "Packaged web asset",
      detail: "Iframe URL is served from Talos package storage (typical production delivery).",
    }
  }
  try {
    const host = window.location.hostname
    if (host === "127.0.0.1" || host === "localhost" || host === "::1") {
      return {
        label: "Dev server (loopback)",
        detail: "Iframe is loading a local Vite dev URL — requires package development enabled in Launchpad.",
      }
    }
  } catch {
    /* ignore */
  }
  return { label: "Other origin", detail: href }
}

function App() {
  const talos = useMemo(() => createTalosApp(APP_ID), [])
  const delivery = useMemo(() => iframeDeliveryMode(), [])
  const [hubState, setHubState] = useState<string>("—")
  const [sidecarJSON, setSidecarJSON] = useState<string>("—")
  const [packageDevFromSidecar, setPackageDevFromSidecar] = useState<string>("—")
  const [uiNote, setUiNote] = useState<string>("Ready")

  async function refreshHubState(): Promise<void> {
    try {
      const raw = await talos.client.loadState()
      if (!raw || raw.byteLength === 0) {
        setHubState("(empty)")
        setUiNote("Hub state empty.")
        return
      }
      setHubState(new TextDecoder().decode(raw))
      setUiNote("Loaded hub state (Rust SaveState / LoadState).")
    } catch (err) {
      setUiNote(`Hub state failed: ${(err as Error).message}`)
    }
  }

  async function refreshSidecarStatus(): Promise<void> {
    try {
      const read = await talos.loadText(STATUS_FILE)
      if (!read.found) {
        setSidecarJSON("(waiting for Rust binary — file not found yet)")
        setPackageDevFromSidecar("—")
        setUiNote("Status JSON missing; start the package sidecar.")
        return
      }
      setSidecarJSON(read.text.trim())
      try {
        const parsed = JSON.parse(read.text) as { talos_package_development?: boolean }
        if (typeof parsed.talos_package_development === "boolean") {
          setPackageDevFromSidecar(parsed.talos_package_development ? "development (1)" : "production (0)")
        } else {
          setPackageDevFromSidecar("(no talos_package_development in JSON)")
        }
      } catch {
        setPackageDevFromSidecar("(invalid JSON)")
      }
      setUiNote("Loaded scoped JSON from the async Rust sidecar.")
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
      setUiNote(`Appended hub state line (${stamp}).`)
      await refreshHubState()
    } catch (err) {
      setUiNote(`saveState failed: ${(err as Error).message}`)
    }
  }

  return (
    <main className="app-root talos-p-4 talos-grid" style={{ minHeight: "100vh", placeItems: "center" }}>
      <TalosPanel style={{ width: "min(920px, 100%)" }}>
        <TalosAlert>
          <span slot="title">Rust + Talos SDK</span>
          The sidecar uses <strong>async</strong> gRPC (Tonic), <strong>serde</strong> for JSON status, and{" "}
          <strong>tokio::select!</strong> for shutdown. Check the badge below: <strong>Launchpad package development</strong>{" "}
          comes from <code className="talos-text-secondary">TALOS_PACKAGE_DEVELOPMENT</code> on the binary; the iframe row
          reflects how this UI is <em>served</em> (packaged vs dev server).
        </TalosAlert>

        <TalosCard>
          <h1 className="talos-text-primary talos-mb-2">Example Rust app</h1>
          <p className="talos-text-secondary talos-mb-3">
            Rust showcases hub RPCs plus scoped file I/O helpers and package SDK logging (when development is on for this
            package in Launchpad Settings).
          </p>
          <div className="talos-flex talos-gap-2 talos-wrap">
            <TalosButton onClick={() => void refreshHubState()}>Load hub state</TalosButton>
            <TalosButton onClick={() => void refreshSidecarStatus()}>Load sidecar status</TalosButton>
            <TalosButton variant="ghost" onClick={() => void appendHubStateFromUI()}>
              Append hub state from UI
            </TalosButton>
          </div>
        </TalosCard>

        <TalosCard>
          <TalosListRow>
            <span slot="leading">⚙</span>
            Launchpad package development (sidecar env)
            <span slot="meta">{packageDevFromSidecar}</span>
          </TalosListRow>
          <TalosListRow>
            <span slot="leading">⎚</span>
            Iframe delivery — {delivery.label}
            <span slot="meta" className="talos-text-secondary" style={{ maxWidth: "min(420px, 48vw)", textAlign: "end" }}>
              {delivery.detail}
            </span>
          </TalosListRow>
        </TalosCard>

        <TalosCard>
          <TalosListRow>
            <span slot="leading">◎</span>
            Hub state
            <span slot="meta" className="talos-text-secondary" style={{ maxWidth: "min(400px, 42vw)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
              {hubState}
            </span>
          </TalosListRow>
        </TalosCard>

        <TalosCard>
          <h2 className="talos-text-primary talos-mb-2 talos-text-body-sm" style={{ fontWeight: 600 }}>
            Sidecar JSON (<code>{STATUS_FILE}</code>)
          </h2>
          <pre
            className="talos-text-secondary talos-m-0 talos-p-3"
            style={{
              fontSize: "0.8rem",
              lineHeight: 1.45,
              overflow: "auto",
              maxHeight: "300px",
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
