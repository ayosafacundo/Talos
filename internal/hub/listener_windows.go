//go:build windows

package hub

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"gopkg.in/natefinch/npipe.v2"
)

func listenLocal(socketURL string) (net.Listener, error) {
	if !strings.HasPrefix(socketURL, "npipe://") {
		return nil, fmt.Errorf("hub: unsupported windows socket url %q", socketURL)
	}

	path := strings.TrimPrefix(socketURL, "npipe://")
	if path == "" {
		return nil, errors.New("hub: empty named pipe path")
	}

	return npipe.Listen(path)
}
