package talos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	hubpb "Talos/api/proto/talos/hub/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a lightweight SDK wrapper for tiny apps.
type Client struct {
	conn *grpc.ClientConn
	hub  hubpb.HubServiceClient
}

// Dial opens a connection to the Talos local hub endpoint.
func Dial(ctx context.Context, socketURL string) (*Client, error) {
	target, dialer, err := resolveDialTarget(socketURL)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.DialContext(
		ctx,
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("talos sdk: dial failed: %w", err)
	}

	return &Client{
		conn: conn,
		hub:  hubpb.NewHubServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) SendMessage(ctx context.Context, sourceAppID, targetAppID, typ string, payload []byte) (*hubpb.Message, error) {
	resp, err := c.hub.Route(ctx, &hubpb.RouteRequest{
		Message: &hubpb.Message{
			SourceAppId: sourceAppID,
			TargetAppId: targetAppID,
			Type:        typ,
			Payload:     payload,
		},
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.GetError()) != "" {
		return nil, fmt.Errorf("talos sdk: route error: %s", resp.GetError())
	}
	return resp.GetMessage(), nil
}

func (c *Client) Broadcast(ctx context.Context, sourceAppID, typ string, payload []byte) (int32, error) {
	resp, err := c.hub.Broadcast(ctx, &hubpb.BroadcastRequest{
		Message: &hubpb.Message{
			SourceAppId: sourceAppID,
			Type:        typ,
			Payload:     payload,
		},
	})
	if err != nil {
		return 0, err
	}
	return resp.GetRecipientCount(), nil
}

func (c *Client) SaveState(ctx context.Context, appID string, data []byte) error {
	resp, err := c.hub.SaveState(ctx, &hubpb.SaveStateRequest{
		AppId: appID,
		Data:  data,
	})
	if err != nil {
		return err
	}
	if !resp.GetOk() {
		return fmt.Errorf("talos sdk: save state failed: %s", resp.GetError())
	}
	return nil
}

func (c *Client) LoadState(ctx context.Context, appID string) ([]byte, bool, error) {
	resp, err := c.hub.LoadState(ctx, &hubpb.LoadStateRequest{
		AppId: appID,
	})
	if err != nil {
		return nil, false, err
	}
	if resp.GetError() != "" {
		return nil, false, fmt.Errorf("talos sdk: load state failed: %s", resp.GetError())
	}
	return resp.GetData(), resp.GetFound(), nil
}

func (c *Client) RequestPermission(ctx context.Context, appID, scope, reason string) (bool, string, error) {
	resp, err := c.hub.RequestPermission(ctx, &hubpb.PermissionRequest{
		AppId:  appID,
		Scope:  scope,
		Reason: reason,
	})
	if err != nil {
		return false, "", err
	}
	return resp.GetGranted(), resp.GetMessage(), nil
}

func (c *Client) ResolvePath(ctx context.Context, appID, relativePath string) (string, error) {
	resp, err := c.hub.ResolvePath(ctx, &hubpb.ResolvePathRequest{
		AppId:        appID,
		RelativePath: relativePath,
	})
	if err != nil {
		return "", err
	}
	if !resp.GetAllowed() {
		return "", fmt.Errorf("talos sdk: resolve path denied: %s", resp.GetError())
	}
	return resp.GetResolvedPath(), nil
}

// Log appends a line to the host SDK log file for appID when Development mode is enabled for that package (no-op on the host otherwise). level is e.g. INFO, WARN, ERROR, DEBUG.
func (c *Client) Log(ctx context.Context, appID, level, message string) error {
	resp, err := c.hub.AppendPackageLog(ctx, &hubpb.AppendPackageLogRequest{
		AppId:   appID,
		Level:   level,
		Message: message,
	})
	if err != nil {
		return err
	}
	if !resp.GetOk() {
		return fmt.Errorf("talos sdk: log failed: %s", resp.GetError())
	}
	return nil
}

func (c *Client) WriteScopedFile(ctx context.Context, appID, relativePath string, data []byte) error {
	path, err := c.ResolvePath(ctx, appID, relativePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (c *Client) ReadScopedFile(ctx context.Context, appID, relativePath string) ([]byte, error) {
	path, err := c.ResolvePath(ctx, appID, relativePath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}
