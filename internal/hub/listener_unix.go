//go:build !windows

package hub

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func listenLocal(socketURL string) (net.Listener, error) {
	if !strings.HasPrefix(socketURL, "unix://") {
		return nil, fmt.Errorf("hub: unsupported unix socket url %q", socketURL)
	}

	path := strings.TrimPrefix(socketURL, "unix://")
	if path == "" {
		return nil, errors.New("hub: empty unix socket path")
	}
	_ = os.Remove(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("hub: create socket dir failed: %w", err)
	}

	return net.Listen("unix", path)
}
