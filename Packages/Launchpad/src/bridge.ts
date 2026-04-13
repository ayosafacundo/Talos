/** Host-side iframe bridge: validation and allowlists for talos:sdk postMessage. */

export const BRIDGE_CHANNEL = "talos:sdk:v1" as const;

export const ALLOWED_SDK_METHODS = new Set([
  "saveState",
  "loadState",
  "requestPermission",
  "resolvePath",
  "sendMessage",
  "broadcast",
  "getInstalledApps",
  "getStoreApps",
  "launchApp",
  "setContextMenuOptions",
  "clearContextMenuOptions",
  "openContextMenu",
]);

const MAX_REQUEST_ID_LEN = 128;
const MAX_APP_ID_LEN = 256;
const MAX_METHOD_LEN = 64;

export type TrustedSender = {
  manifestId: string;
  instanceId: string;
  bridgeToken: string;
};

function isPlainObject(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === "object" && !Array.isArray(v);
}

/** Parse incoming postMessage data; returns null if not a valid v1 request. */
export function parseBridgeRequest(raw: unknown): {
  channel: string;
  requestId: string;
  appId: string;
  method: string;
  params: Record<string, unknown>;
  bridgeToken: string;
} | null {
  if (!isPlainObject(raw)) return null;
  const type = raw.type;
  if (type !== "talos:sdk:req") return null;

  const channel = String(raw.channel || "");
  const requestId = String(raw.request_id || "").trim();
  const appId = String(raw.app_id || "").trim();
  const method = String(raw.method || "").trim();
  const params = raw.params;
  const bridgeToken = String(raw.bridge_token || "").trim();

  if (channel !== "" && channel !== BRIDGE_CHANNEL) return null;
  if (requestId.length === 0 || requestId.length > MAX_REQUEST_ID_LEN) return null;
  if (appId.length === 0 || appId.length > MAX_APP_ID_LEN) return null;
  if (method.length === 0 || method.length > MAX_METHOD_LEN) return null;
  if (!isPlainObject(params)) return null;
  if (bridgeToken.length < 16) return null;

  return { channel: channel || BRIDGE_CHANNEL, requestId, appId, method, params, bridgeToken };
}

export function isAllowedMethod(method: string): boolean {
  return ALLOWED_SDK_METHODS.has(method);
}

export type BridgeRejectReason =
  | "not_bridge_message"
  | "invalid_envelope"
  | "unknown_method"
  | "source_mismatch"
  | "token_mismatch"
  | "app_id_mismatch";

export function resolveTrustedSender(
  eventSource: Window | null,
  iframeByInstanceId: Record<string, HTMLIFrameElement | null | undefined>,
  msgAppId: string,
  bridgeToken: string,
): { ok: true; trusted: TrustedSender } | { ok: false; reason: BridgeRejectReason } {
  if (!eventSource) {
    return { ok: false, reason: "source_mismatch" };
  }

  for (const [instanceId, iframe] of Object.entries(iframeByInstanceId)) {
    if (!iframe) continue;
    if (iframe.contentWindow !== eventSource) continue;

    const expectedToken = iframe.dataset.talosBridgeToken || "";
    const manifestId = iframe.dataset.talosManifestId || "";
    if (!manifestId || !expectedToken) {
      return { ok: false, reason: "source_mismatch" };
    }
    if (expectedToken !== bridgeToken) {
      return { ok: false, reason: "token_mismatch" };
    }
    if (msgAppId !== manifestId) {
      return { ok: false, reason: "app_id_mismatch" };
    }
    return {
      ok: true,
      trusted: { manifestId, instanceId, bridgeToken },
    };
  }

  return { ok: false, reason: "source_mismatch" };
}

export function buildBridgeResponse(
  requestId: string,
  ok: boolean,
  result: unknown,
  error: string,
): Record<string, unknown> {
  return {
    channel: BRIDGE_CHANNEL,
    type: "talos:sdk:res",
    request_id: requestId,
    ok,
    result,
    error,
  };
}
