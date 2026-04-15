import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './talos/tokens.css'
import './talos/legacy-alias.css'
import './talos/utilities.css'
import './index.css'
import App from './App'

const rootEl = document.getElementById('root')!
createRoot(rootEl).render(
  import.meta.env.DEV ? (
    <StrictMode>
      <App />
    </StrictMode>
  ) : (
    <App />
  ),
)
