import { useMemo, useState } from "react"
import { createTalosApp } from "@talos/web-components"
import {
  TalosAlert,
  TalosButton,
  TalosCard,
  TalosListRow,
  TalosPanel,
} from "@talos/web-components/react"

const APP_ID = "app.example.ts"
const COUNTER_FILE = "counter.txt"

function App() {
  const talos = useMemo(() => createTalosApp(APP_ID), [])
  const [count, setCount] = useState(0)
  const [status, setStatus] = useState("Ready")

  async function saveCounter(): Promise<void> {
    try {
      await talos.saveText(COUNTER_FILE, String(count))
      const resolved = await talos.client.resolvePath(COUNTER_FILE)
      setStatus(`Saved counter=${count} to ${resolved}`)
    } catch (err) {
      setStatus(`Save failed: ${(err as Error).message}`)
    }
  }

  async function loadCounter(): Promise<void> {
    try {
      const read = await talos.loadText(COUNTER_FILE)
      if (!read.found) {
        setCount(0)
        setStatus("Counter file does not exist yet. Using 0.")
        return
      }
      const parsed = Number.parseInt(read.text.trim(), 10)
      if (Number.isNaN(parsed)) {
        setStatus(`Counter file is invalid ('${read.text.trim()}'). Using 0.`)
        setCount(0)
        return
      }
      setCount(parsed)
      setStatus(`Loaded counter=${parsed} from scoped text file`)
    } catch (err) {
      setStatus(`Load failed: ${(err as Error).message}`)
    }
  }

  return (
    <main className="app-root talos-p-4 talos-grid" style={{ minHeight: "100vh", placeItems: "center" }}>
      <TalosPanel style={{ width: "min(820px, 100%)" }}>
        <TalosAlert>
          <span slot="title">Talos Theme System Test App</span>
          This React app uses Talos tokens, runtime theme sync, and Talos Web Components only.
        </TalosAlert>

        <TalosCard>
          <h1 className="talos-text-primary talos-mb-2">Example TypeScript app (React)</h1>
          <p className="talos-text-secondary talos-mb-3">Counter: {count}</p>
          <div className="talos-flex talos-gap-2 talos-wrap">
            <TalosButton onClick={() => setCount((v) => v + 1)}>Increase counter</TalosButton>
            <TalosButton onClick={() => void saveCounter()}>Save counter to .txt</TalosButton>
            <TalosButton variant="ghost" onClick={() => void loadCounter()}>
              Load counter from .txt
            </TalosButton>
          </div>
        </TalosCard>

        <TalosCard>
          <TalosListRow>
            <span slot="leading">i</span>
            Operation status
            <span slot="meta">{status}</span>
            <span slot="trailing">{count}</span>
          </TalosListRow>
        </TalosCard>
      </TalosPanel>
    </main>
  )
}

export default App
