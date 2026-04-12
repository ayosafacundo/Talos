export type PermissionResult = {
  granted: boolean
  message: string
}

export type ContextMenuOption = {
  id: string
  label: string
}

export interface TalosTransport {
  saveState(appId: string, data: Uint8Array): Promise<void>
  loadState(appId: string): Promise<Uint8Array | null>
  sendMessage(targetID: string, payload: Uint8Array): Promise<Uint8Array | null>
  requestPermission(scope: string, reason?: string): Promise<PermissionResult>
  resolvePath(appId: string, relativePath: string): Promise<string>
  setContextMenuOptions?(appId: string, options: ContextMenuOption[]): Promise<void>
  clearContextMenuOptions?(appId: string): Promise<void>
  openContextMenu?(appId: string, x?: number, y?: number): Promise<void>
}
