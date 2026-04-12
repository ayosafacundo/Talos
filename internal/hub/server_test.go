package hub

import (
	"context"
	"testing"

	hubpb "Talos/api/proto/talos/hub/v1"
)

func TestRouteRequest(t *testing.T) {
	t.Parallel()

	s := NewServer("unix:///tmp/talos_hub_test.sock")
	s.RegisterHandler("app.b", func(_ context.Context, msg *hubpb.Message) (*hubpb.Message, error) {
		return &hubpb.Message{
			SourceAppId: "app.b",
			TargetAppId: msg.SourceAppId,
			Type:        "response",
			Payload:     []byte("ok"),
		}, nil
	})

	resp, err := s.Route(context.Background(), &hubpb.RouteRequest{
		Message: &hubpb.Message{
			SourceAppId: "app.a",
			TargetAppId: "app.b",
			Type:        "request",
			Payload:     []byte("ping"),
		},
	})
	if err != nil {
		t.Fatalf("Route() error: %v", err)
	}
	if string(resp.Message.Payload) != "ok" {
		t.Fatalf("expected payload ok, got %q", string(resp.Message.Payload))
	}
}

func TestBroadcast(t *testing.T) {
	t.Parallel()

	s := NewServer("unix:///tmp/talos_hub_test.sock")
	s.RegisterHandler("app.a", func(_ context.Context, _ *hubpb.Message) (*hubpb.Message, error) {
		return &hubpb.Message{}, nil
	})
	s.RegisterHandler("app.b", func(_ context.Context, _ *hubpb.Message) (*hubpb.Message, error) {
		return &hubpb.Message{}, nil
	})
	s.RegisterHandler("app.c", func(_ context.Context, _ *hubpb.Message) (*hubpb.Message, error) {
		return &hubpb.Message{}, nil
	})

	resp, err := s.Broadcast(context.Background(), &hubpb.BroadcastRequest{
		Message: &hubpb.Message{
			SourceAppId: "app.a",
			Type:        "broadcast",
		},
	})
	if err != nil {
		t.Fatalf("Broadcast() error: %v", err)
	}
	if resp.RecipientCount != 2 {
		t.Fatalf("expected 2 recipients, got %d", resp.RecipientCount)
	}
}

func TestStateHooks(t *testing.T) {
	t.Parallel()

	s := NewServer("unix:///tmp/talos_hub_test.sock")
	store := map[string][]byte{}
	s.SetStateHooks(
		func(appID string, data []byte) error {
			store[appID] = data
			return nil
		},
		func(appID string) ([]byte, bool, error) {
			data, ok := store[appID]
			return data, ok, nil
		},
	)

	_, err := s.SaveState(context.Background(), &hubpb.SaveStateRequest{
		AppId: "app.test",
		Data:  []byte("state"),
	})
	if err != nil {
		t.Fatalf("SaveState() error: %v", err)
	}

	loadResp, err := s.LoadState(context.Background(), &hubpb.LoadStateRequest{
		AppId: "app.test",
	})
	if err != nil {
		t.Fatalf("LoadState() error: %v", err)
	}
	if !loadResp.Found || string(loadResp.Data) != "state" {
		t.Fatalf("unexpected load state response: found=%v data=%q", loadResp.Found, string(loadResp.Data))
	}
}

func TestRequestPermissionHook(t *testing.T) {
	t.Parallel()

	s := NewServer("unix:///tmp/talos_hub_test.sock")
	s.SetPermissionRequestHook(func(appID, scope, reason string) (bool, string, error) {
		if appID == "app.test" && scope == "net:internet" {
			return true, "approved", nil
		}
		return false, "denied", nil
	})

	resp, err := s.RequestPermission(context.Background(), &hubpb.PermissionRequest{
		AppId:  "app.test",
		Scope:  "net:internet",
		Reason: "sync",
	})
	if err != nil {
		t.Fatalf("RequestPermission() error: %v", err)
	}
	if !resp.Granted {
		t.Fatalf("expected permission granted")
	}
}

func TestResolvePathHook(t *testing.T) {
	t.Parallel()

	s := NewServer("unix:///tmp/talos_hub_test.sock")
	s.SetResolvePathHook(func(appID, relativePath string) (string, bool, error) {
		if appID == "app.test" {
			return "/tmp/Packages/App/data/" + relativePath, true, nil
		}
		return "", false, nil
	})

	resp, err := s.ResolvePath(context.Background(), &hubpb.ResolvePathRequest{
		AppId:        "app.test",
		RelativePath: "cache/state.json",
	})
	if err != nil {
		t.Fatalf("ResolvePath() error: %v", err)
	}
	if !resp.Allowed {
		t.Fatalf("expected path allowed")
	}
}
