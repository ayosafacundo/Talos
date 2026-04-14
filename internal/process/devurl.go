package process

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Common dev-server log lines (Vite, webpack-dev-server, etc.).
var devLogURLPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)Local:\s*(https?://[^\s]+)`),
	regexp.MustCompile(`(?i)➜\s+Local:\s*(https?://[^\s]+)`),
	regexp.MustCompile(`(?i)On Your Network:\s*(https?://[^\s]+)`),
	regexp.MustCompile(`(https?://127\.0\.0\.1:\d+/)`),
	regexp.MustCompile(`(https?://localhost:\d+/)`),
	regexp.MustCompile(`(https?://\[::1\]:\d+/)`),
}

// AlignDiscoveredDevURL rewrites a discovered dev-server URL to use the manifest URL's
// loopback hostname while keeping the discovered listen port. This keeps the iframe and
// manifest consistent when Vite prints `localhost` but `development.url` uses `127.0.0.1`
// (or the reverse).
func AlignDiscoveredDevURL(manifestURL, discovered string) string {
	discovered = strings.TrimSpace(discovered)
	m := strings.TrimSpace(manifestURL)
	if m == "" || discovered == "" {
		return discovered
	}
	mu, err1 := url.Parse(m)
	du, err2 := url.Parse(discovered)
	if err1 != nil || err2 != nil {
		return discovered
	}
	if !isLoopbackHost(mu.Hostname()) || !isLoopbackHost(du.Hostname()) {
		return discovered
	}
	dp := du.Port()
	if dp == "" {
		return discovered
	}
	scheme := mu.Scheme
	if scheme == "" {
		scheme = "http"
	}
	host := mu.Hostname()
	return scheme + "://" + net.JoinHostPort(host, dp) + "/"
}

// ExpandLoopbackOrigins adds the alternate loopback hostname (127.0.0.1 ↔ localhost)
// for each http(s) origin that has an explicit port, so postMessage bridge checks pass
// regardless of which hostname the dev server printed.
func ExpandLoopbackOrigins(origins []string) []string {
	seen := make(map[string]struct{}, len(origins)*2)
	out := make([]string, 0, len(origins)*2)

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		key := strings.TrimSuffix(s, "/")
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}

	for _, o := range origins {
		add(o)
		u, err := url.Parse(o)
		if err != nil || u.Port() == "" {
			continue
		}
		h := strings.ToLower(u.Hostname())
		if !isLoopbackHost(h) {
			continue
		}
		port := u.Port()
		scheme := u.Scheme
		if scheme == "" {
			scheme = "http"
		}
		switch h {
		case "127.0.0.1", "::1":
			if h == "::1" {
				add(scheme + "://127.0.0.1:" + port)
				add(scheme + "://localhost:" + port)
			} else {
				add(scheme + "://localhost:" + port)
			}
		case "localhost":
			add(scheme + "://127.0.0.1:" + port)
		}
	}
	return out
}

// ParseDevServerBaseURL scans combined dev-server stdout/stderr for a loopback URL.
func ParseDevServerBaseURL(log string) string {
	for _, re := range devLogURLPatterns {
		m := re.FindStringSubmatch(log)
		if len(m) < 2 {
			continue
		}
		raw := strings.TrimSpace(m[1])
		raw = strings.TrimSuffix(raw, "\x1b[0m") // strip ANSI tail if present
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		if !isLoopbackHost(u.Hostname()) {
			continue
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			continue
		}
		out := u.Scheme + "://" + u.Host
		if !strings.HasSuffix(out, "/") {
			out += "/"
		}
		return out
	}
	return ""
}

func isLoopbackHost(h string) bool {
	switch strings.ToLower(h) {
	case "localhost", "::1":
		return true
	}
	if ip := net.ParseIP(h); ip != nil && ip.IsLoopback() {
		return true
	}
	return strings.HasPrefix(h, "127.")
}

// ProbeHTTPAroundManifest tries GET http(s)://host:port/ for ports near the manifest URL's port.
// For loopback manifests it probes both the manifest hostname and the alternate (127.0.0.1 ↔ localhost).
func ProbeHTTPAroundManifest(ctx context.Context, manifestURL string, span int) string {
	u, err := url.Parse(strings.TrimSpace(manifestURL))
	if err != nil || u.Hostname() == "" {
		return ""
	}
	host := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		return ""
	}
	center, err := strconv.Atoi(portStr)
	if err != nil {
		return ""
	}
	hosts := []string{host}
	if isLoopbackHost(host) {
		switch strings.ToLower(host) {
		case "127.0.0.1":
			hosts = append(hosts, "localhost")
		case "localhost":
			hosts = append(hosts, "127.0.0.1")
		}
	}
	scheme := u.Scheme
	if scheme == "" {
		scheme = "http"
	}
	client := &http.Client{Timeout: 450 * time.Millisecond}
	for delta := -span; delta <= span; delta++ {
		p := center + delta
		if p < 1 || p > 65535 {
			continue
		}
		for _, h := range hosts {
			tryURL := scheme + "://" + net.JoinHostPort(h, strconv.Itoa(p)) + "/"
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, tryURL, nil)
			if err != nil {
				continue
			}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			_, _ = io.CopyN(io.Discard, resp.Body, 512)
			_ = resp.Body.Close()
			if resp.StatusCode < 500 {
				return tryURL
			}
		}
	}
	return ""
}
