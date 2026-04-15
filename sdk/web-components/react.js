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

/**
 * React 19 + WebKit: assigning a style object to custom elements can throw
 * "Attempted to assign to readonly property" on CSSStyleDeclaration. Host styles on a wrapper div.
 */
function hostWithOptionalStyleWrapper(tag, props) {
  if (!props) {
    return el(tag, null)
  }
  const { style, children, ...rest } = props
  if (style != null && style !== undefined) {
    return el("div", { style }, el(tag, rest, children))
  }
  return el(tag, rest, children)
}

export function TalosPanel(props) {
  return hostWithOptionalStyleWrapper("talos-panel", props)
}

export function TalosCard(props) {
  return hostWithOptionalStyleWrapper("talos-card", props)
}

export function TalosButton(props) {
  return hostWithOptionalStyleWrapper("talos-button", props)
}

export function TalosAlert(props) {
  return hostWithOptionalStyleWrapper("talos-alert", props)
}

export function TalosListRow(props) {
  return hostWithOptionalStyleWrapper("talos-list-row", props)
}
