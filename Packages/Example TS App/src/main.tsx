import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import "@talos/web-components/react-types"
import { registerTalosWebComponents } from "@talos/web-components"
import App from "./App.tsx"
import "./index.css"

registerTalosWebComponents()

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
