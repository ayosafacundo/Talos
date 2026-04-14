import { createElement, useEffect, useMemo, useState } from "react"
import { createTalosApp } from "./index.js"

export function useTalosApp(appID) {
  const app = useMemo(() => createTalosApp(appID), [appID])
  const [themeReady, setThemeReady] = useState(false)

  useEffect(() => {
    let active = true
    void app.themeReady.then(() => {
      if (active) setThemeReady(true)
    })
    return () => {
      active = false
    }
  }, [app])

  useEffect(() => () => app.dispose(), [app])
  return { ...app, themeReady }
}

function el(tag, props, ...children) {
  return createElement(tag, props, ...children)
}

export function TalosPanel(props) {
  return el("talos-panel", props, props?.children)
}

export function TalosCard(props) {
  return el("talos-card", props, props?.children)
}

export function TalosButton(props) {
  return el("talos-button", props, props?.children)
}

export function TalosAlert(props) {
  return el("talos-alert", props, props?.children)
}

export function TalosListRow(props) {
  return el("talos-list-row", props, props?.children)
}
