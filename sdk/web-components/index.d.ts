import type { ScopedTextReadResult, TalosClient } from "@talos/sdk"

export type TalosAppRuntime = {
  client: TalosClient
  registerTalosWebComponents: () => void
  themeReady: Promise<void>
  dispose: () => void
  ensureFilesystemPermission: (relativePath?: string) => Promise<void>
  saveText: (relativePath: string, text: string) => Promise<void>
  loadText: (relativePath: string) => Promise<ScopedTextReadResult>
}

export function registerTalosWebComponents(): void
export function createTalosApp(appID: string): TalosAppRuntime
