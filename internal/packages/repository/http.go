package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// HTTP loads a JSON catalog from a base URL (GET baseURL, typically ending with /v1/catalog or full URL to JSON).
type HTTP struct {
	BaseURL string
	Client  *http.Client
}

// NewHTTP creates a catalog client. baseURL should be a full URL to a JSON document describing descriptors.
func NewHTTP(baseURL string) *HTTP {
	return &HTTP{
		BaseURL: strings.TrimSpace(baseURL),
		Client:  &http.Client{Timeout: 45 * time.Second},
	}
}

// remoteRow matches JSON from the catalog feed.
type remoteRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source,omitempty"`
	InstallURL string `json:"install_url,omitempty"`
}

// List fetches catalog entries. Expected JSON: [{"id":"...","name":"...","install_url":"https://..."}, ...]
func (h *HTTP) List(ctx context.Context) ([]Descriptor, error) {
	if h.BaseURL == "" {
		return nil, fmt.Errorf("repository: empty base url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.BaseURL, nil)
	if err != nil {
		return nil, err
	}
	cli := h.Client
	if cli == nil {
		cli = &http.Client{Timeout: 45 * time.Second}
	}
	resp, err := doWithRetry(ctx, cli, req, 3)
	if err != nil {
		if isNetworkError(err) {
			return nil, fmt.Errorf("repository: network error while loading catalog (offline or DNS issue): %w", err)
		}
		return nil, fmt.Errorf("repository: fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("repository: HTTP %d while loading catalog: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	var rows []remoteRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("repository: invalid catalog JSON: %w", err)
	}
	out := make([]Descriptor, 0, len(rows))
	for _, r := range rows {
		if strings.TrimSpace(r.ID) == "" {
			continue
		}
		out = append(out, Descriptor{
			ID:         r.ID,
			Name:       r.Name,
			Source:     r.Source,
			InstallURL: r.InstallURL,
		})
	}
	return out, nil
}

func doWithRetry(ctx context.Context, client *http.Client, req *http.Request, maxAttempts int) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt == maxAttempts || ctx.Err() != nil {
			break
		}
		wait := time.Duration(attempt*attempt) * 100 * time.Millisecond
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, lastErr
}

func isNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr)
}
