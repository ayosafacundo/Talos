import type { ContextMenuOption, PermissionResult, ScopedTextReadResult, TalosTransport } from "./types"

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

  async readScopedText(relativePath: string): Promise<ScopedTextReadResult> {
    if (!this.transport.readScopedText) {
      throw new Error("transport does not support scoped text reads")
    }
    return this.transport.readScopedText(this.appId, relativePath)
  }

  async writeScopedText(relativePath: string, text: string): Promise<void> {
    if (!this.transport.writeScopedText) {
      throw new Error("transport does not support scoped text writes")
    }
    await this.transport.writeScopedText(this.appId, relativePath, text)
  }

  async packageLocalHttp(method: string, path: string, body = ""): Promise<{
    status: number
    content_type: string
    body: string
  }> {
    if (!this.transport.packageLocalHttp) {
      throw new Error("transport does not support package local HTTP")
    }
    return this.transport.packageLocalHttp(this.appId, method, path, body)
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
