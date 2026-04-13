/** True when Wails has injected `window.runtime` and Go bindings (`window.go.main.App`). */
export function wailsShellReady(): boolean {
  if (typeof window === "undefined") return false;
  const w = window as unknown as {
    go?: { main?: { App?: unknown } };
    runtime?: { EventsOnMultiple?: unknown };
  };
  return Boolean(w.runtime?.EventsOnMultiple && w.go?.main?.App);
}
