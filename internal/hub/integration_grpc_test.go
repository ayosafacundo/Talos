//go:build integration

package hub

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	hubpb "Talos/api/proto/talos/hub/v1"
	"Talos/sdk/go/talos"
)

func integrationSocketURL(t *testing.T) string {
	t.Helper()
	if u := os.Getenv("TALOS_TEST_SOCKET"); u != "" {
		return u
	}
	return "unix://" + filepath.Join(t.TempDir(), "talos_integration.sock")
}

func TestIntegrationHubGRPCRouteAndBroadcast(t *testing.T) {
	url := integrationSocketURL(t)
	s := NewServer(url)
	s.RegisterHandler("app.b", func(_ context.Context, msg *hubpb.Message) (*hubpb.Message, error) {
		return &hubpb.Message{
			SourceAppId: "app.b",
			TargetAppId: msg.SourceAppId,
			Type:        "reply",
			Payload:     []byte("pong"),
			RequestId:   msg.RequestId,
		}, nil
	})
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(s.Stop)

	ctx := context.Background()
	cli, err := talos.Dial(ctx, url)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })

	got, err := cli.SendMessage(ctx, "app.a", "app.b", "ping", []byte("x"))
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if string(got.GetPayload()) != "pong" {
		t.Fatalf("payload: %q", got.GetPayload())
	}

	s.RegisterHandler("app.c", func(_ context.Context, _ *hubpb.Message) (*hubpb.Message, error) {
		return &hubpb.Message{}, nil
	})
	n, err := cli.Broadcast(ctx, "app.a", "hi", []byte("z"))
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if n != 2 {
		t.Fatalf("recipient count: %d", n)
	}
}

func TestIntegrationHubMultiAppScenario(t *testing.T) {
	url := integrationSocketURL(t)
	s := NewServer(url)
	store := map[string][]byte{}
	s.SetStateHooks(
		func(appID string, data []byte) error {
			store[appID] = data
			return nil
		},
		func(appID string) ([]byte, bool, error) {
			d, ok := store[appID]
			return d, ok, nil
		},
	)
	s.SetPermissionRequestHook(func(appID, scope, _ string) (bool, string, error) {
		if appID == "app.a" && scope == "fs:external" {
			return true, "approved", nil
		}
		return false, "denied", nil
	})
	s.SetResolvePathHook(func(appID, relativePath string) (string, bool, error) {
		if appID == "app.a" {
			return filepath.Join("/tmp/Packages/AppA/data", relativePath), true, nil
		}
		return "", false, nil
	})
	s.RegisterHandler("app.b", func(_ context.Context, msg *hubpb.Message) (*hubpb.Message, error) {
		return &hubpb.Message{
			SourceAppId: "app.b",
			TargetAppId: msg.SourceAppId,
			Type:        "ack",
			Payload:     []byte("routed"),
			RequestId:   msg.RequestId,
		}, nil
	})
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(s.Stop)

	ctx := context.Background()
	cli, err := talos.Dial(ctx, url)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })

	if err := cli.SaveState(ctx, "app.a", []byte("state-a")); err != nil {
		t.Fatalf("SaveState a: %v", err)
	}
	if err := cli.SaveState(ctx, "app.b", []byte("state-b")); err != nil {
		t.Fatalf("SaveState b: %v", err)
	}
	da, fa, err := cli.LoadState(ctx, "app.a")
	if err != nil || !fa || string(da) != "state-a" {
		t.Fatalf("LoadState a: err=%v found=%v data=%q", err, fa, da)
	}
	db, fb, err := cli.LoadState(ctx, "app.b")
	if err != nil || !fb || string(db) != "state-b" {
		t.Fatalf("LoadState b: err=%v found=%v data=%q", err, fb, db)
	}

	msg, err := cli.SendMessage(ctx, "app.a", "app.b", "t", []byte("ping"))
	if err != nil || string(msg.GetPayload()) != "routed" {
		t.Fatalf("route: err=%v payload=%q", err, msg.GetPayload())
	}

	n, err := cli.Broadcast(ctx, "app.a", "evt", []byte("b"))
	if err != nil || n != 1 {
		t.Fatalf("broadcast: err=%v n=%d", err, n)
	}

	ok, m, err := cli.RequestPermission(ctx, "app.a", "fs:external", "need")
	if err != nil || !ok || m != "approved" {
		t.Fatalf("permission a: err=%v ok=%v m=%q", err, ok, m)
	}
	ok2, _, err := cli.RequestPermission(ctx, "app.b", "fs:external", "need")
	if err != nil || ok2 {
		t.Fatalf("permission b: err=%v ok=%v", err, ok2)
	}

	p, err := cli.ResolvePath(ctx, "app.a", "x.txt")
	if err != nil || p != filepath.Join("/tmp/Packages/AppA/data", "x.txt") {
		t.Fatalf("resolve: err=%v p=%q", err, p)
	}
}
