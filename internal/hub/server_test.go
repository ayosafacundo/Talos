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
