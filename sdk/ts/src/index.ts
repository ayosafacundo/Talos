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

// TalosClient is an SDK wrapper intended for tiny app usage.
// A transport adapter can target grpc-web, native node gRPC, or bridge IPC.
export class TalosClient {
  private readonly appId: string
  private readonly transport: TalosTransport

  constructor(appId: string, transport: TalosTransport) {
    this.appId = appId
    this.transport = transport
  }

  async saveState(data: Uint8Array): Promise<void> {
    await this.transport.saveState(this.appId, data)
  }

  async loadState(): Promise<Uint8Array | null> {
    return this.transport.loadState(this.appId)
  }

  async sendMessage(targetID: string, payload: Uint8Array): Promise<Uint8Array | null> {
    return this.transport.sendMessage(targetID, payload)
  }

  async requestPermission(scope: string, reason = ""): Promise<PermissionResult> {
    return this.transport.requestPermission(scope, reason)
  }

  async resolvePath(relativePath: string): Promise<string> {
    return this.transport.resolvePath(this.appId, relativePath)
  }

  async setContextMenuOptions(options: ContextMenuOption[]): Promise<void> {
    if (!this.transport.setContextMenuOptions) {
      throw new Error("transport does not support context menu options")
    }
    await this.transport.setContextMenuOptions(this.appId, options)
  }

  async clearContextMenuOptions(): Promise<void> {
    if (!this.transport.clearContextMenuOptions) {
      throw new Error("transport does not support context menu options")
    }
    await this.transport.clearContextMenuOptions(this.appId)
  }

  async openContextMenu(x?: number, y?: number): Promise<void> {
    if (!this.transport.openContextMenu) {
      throw new Error("transport does not support context menu opening")
    }
    await this.transport.openContextMenu(this.appId, x, y)
  }
}
