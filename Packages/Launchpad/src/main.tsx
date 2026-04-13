import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './talos/tokens.css'
import './talos/legacy-alias.css'
import './talos/utilities.css'
import './index.css'
import App from './App'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
