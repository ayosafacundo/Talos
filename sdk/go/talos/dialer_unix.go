//go:build !windows

package talos

import (
	"context"
	"fmt"
	"net"
	"strings"
)

func resolveDialTarget(socketURL string) (string, func(context.Context, string) (net.Conn, error), error) {
	if !strings.HasPrefix(socketURL, "unix://") {
		return "", nil, fmt.Errorf("talos sdk: unsupported socket url for unix platform: %q", socketURL)
	}

	path := strings.TrimPrefix(socketURL, "unix://")
	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "unix", path)
	}
	return "passthrough:///talos-hub", dialer, nil
}
