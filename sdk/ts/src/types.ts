export type PermissionResult = {
  granted: boolean
  message: string
}

export type ContextMenuOption = {
  id: string
  label: string
}

export type ScopedTextReadResult = {
  found: boolean
  text: string
}

export interface TalosTransport {
  saveState(appId: string, data: Uint8Array): Promise<void>
  loadState(appId: string): Promise<Uint8Array | null>
  sendMessage(targetID: string, payload: Uint8Array): Promise<Uint8Array | null>
  requestPermission(scope: string, reason?: string): Promise<PermissionResult>
  resolvePath(appId: string, relativePath: string): Promise<string>
  readScopedText?(appId: string, relativePath: string): Promise<ScopedTextReadResult>
  writeScopedText?(appId: string, relativePath: string, text: string): Promise<void>
  setContextMenuOptions?(appId: string, options: ContextMenuOption[]): Promise<void>
  clearContextMenuOptions?(appId: string): Promise<void>
  openContextMenu?(appId: string, x?: number, y?: number): Promise<void>
}
