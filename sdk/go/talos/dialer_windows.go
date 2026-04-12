//go:build windows

package talos

import (
	"context"
	"fmt"
	"net"
	"strings"

	"gopkg.in/natefinch/npipe.v2"
)

func resolveDialTarget(socketURL string) (string, func(context.Context, string) (net.Conn, error), error) {
	if !strings.HasPrefix(socketURL, "npipe://") {
		return "", nil, fmt.Errorf("talos sdk: unsupported socket url for windows platform: %q", socketURL)
	}

	pipePath := strings.TrimPrefix(socketURL, "npipe://")
	dialer := func(_ context.Context, _ string) (net.Conn, error) {
		return npipe.Dial(pipePath)
	}
	return "passthrough:///talos-hub", dialer, nil
}
