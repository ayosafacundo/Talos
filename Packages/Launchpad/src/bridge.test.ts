// @vitest-environment jsdom

import { describe, expect, it } from "vitest";
import {
  BRIDGE_CHANNEL,
  ALLOWED_SDK_METHODS,
  buildBridgeResponse,
  isAllowedMethod,
  isMessageOriginAllowed,
  normalizeWebOrigin,
  parseBridgeRequest,
  resolveTrustedSender,
} from "./bridge";

describe("parseBridgeRequest", () => {
  it("accepts a minimal valid v1 envelope", () => {
    const raw = {
      type: "talos:sdk:req",
      channel: BRIDGE_CHANNEL,
      request_id: "rid-1",
      app_id: "app.demo",
      method: "loadState",
      params: {},
      bridge_token: "0123456789abcdef",
    };
    const p = parseBridgeRequest(raw);
    expect(p).not.toBeNull();
    expect(p!.requestId).toBe("rid-1");
    expect(p!.appId).toBe("app.demo");
    expect(p!.method).toBe("loadState");
    expect(p!.bridgeToken).toBe("0123456789abcdef");
    expect(p!.channel).toBe(BRIDGE_CHANNEL);
  });

  it("rejects wrong type, channel, token length, and non-object params", () => {
    expect(parseBridgeRequest({ type: "other" })).toBeNull();
    expect(
      parseBridgeRequest({
        type: "talos:sdk:req",
        channel: "evil",
        request_id: "r",
        app_id: "a",
        method: "m",
        params: {},
        bridge_token: "0123456789abcdef",
      }),
    ).toBeNull();
    expect(
      parseBridgeRequest({
        type: "talos:sdk:req",
        request_id: "r",
        app_id: "a",
        method: "m",
        params: {},
        bridge_token: "short",
      }),
    ).toBeNull();
    expect(
      parseBridgeRequest({
        type: "talos:sdk:req",
        request_id: "r",
        app_id: "a",
        method: "m",
        params: [],
        bridge_token: "0123456789abcdef",
      }),
    ).toBeNull();
  });
});

describe("buildBridgeResponse", () => {
  it("includes channel and type", () => {
    const r = buildBridgeResponse("rid", true, { x: 1 }, "");
    expect(r.channel).toBe(BRIDGE_CHANNEL);
    expect(r.type).toBe("talos:sdk:res");
    expect(r.request_id).toBe("rid");
    expect(r.ok).toBe(true);
    expect(r.result).toEqual({ x: 1 });
  });
});

describe("isAllowedMethod", () => {
  it("matches allowlist", () => {
    expect(isAllowedMethod("saveState")).toBe(true);
    expect(isAllowedMethod("not_a_method")).toBe(false);
    expect(ALLOWED_SDK_METHODS.has("broadcast")).toBe(true);
  });
});

describe("normalizeWebOrigin", () => {
  it("strips paths and trailing ambiguity for loopback", () => {
    expect(normalizeWebOrigin("http://127.0.0.1:5174/")).toBe("http://127.0.0.1:5174");
    expect(normalizeWebOrigin("http://127.0.0.1:5174")).toBe("http://127.0.0.1:5174");
    expect(normalizeWebOrigin("http://localhost:3000/foo")).toBe("http://localhost:3000");
  });
});

describe("isMessageOriginAllowed", () => {
  it("treats empty allowlist as allow-all", () => {
    expect(isMessageOriginAllowed("http://127.0.0.1:1", undefined)).toBe(true);
    expect(isMessageOriginAllowed("http://127.0.0.1:1", [])).toBe(true);
  });

  it("matches 127.0.0.1 vs localhost when both listed", () => {
    const allowed = ["http://127.0.0.1:5174", "http://localhost:5174"];
    expect(isMessageOriginAllowed("http://127.0.0.1:5174", allowed)).toBe(true);
    expect(isMessageOriginAllowed("http://localhost:5174", allowed)).toBe(true);
    expect(isMessageOriginAllowed("http://evil.com", allowed)).toBe(false);
  });
});

describe("resolveTrustedSender", () => {
  it("matches iframe contentWindow and token", () => {
    const iframe = document.createElement("iframe");
    document.body.appendChild(iframe);
    const childWin = iframe.contentWindow;
    if (!childWin) throw new Error("no contentWindow");
    iframe.dataset.talosBridgeToken = "0123456789abcdef";
    iframe.dataset.talosManifestId = "app.demo";

    const map: Record<string, HTMLIFrameElement | null> = { i1: iframe };
    const ok = resolveTrustedSender(childWin, map, "app.demo", "0123456789abcdef");
    expect(ok.ok).toBe(true);
    if (ok.ok) {
      expect(ok.trusted.manifestId).toBe("app.demo");
      expect(ok.trusted.bridgeToken).toBe("0123456789abcdef");
    }
  });

  it("rejects token and app id mismatches", () => {
    const iframe = document.createElement("iframe");
    document.body.appendChild(iframe);
    const childWin = iframe.contentWindow!;
    iframe.dataset.talosBridgeToken = "0123456789abcdef";
    iframe.dataset.talosManifestId = "app.demo";
    const map: Record<string, HTMLIFrameElement | null> = { i1: iframe };

    const badTok = resolveTrustedSender(childWin, map, "app.demo", "ffffffffffffffff");
    expect(badTok.ok).toBe(false);
    if (!badTok.ok) expect(badTok.reason).toBe("token_mismatch");

    const badApp = resolveTrustedSender(childWin, map, "other.app", "0123456789abcdef");
    expect(badApp.ok).toBe(false);
    if (!badApp.ok) expect(badApp.reason).toBe("app_id_mismatch");
  });
});
