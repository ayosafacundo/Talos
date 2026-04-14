package updates

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

// ChannelEntry describes one installable artifact from an update or catalog channel.
type ChannelEntry struct {
	AppID          string `json:"app_id"`
	Version        string `json:"version"`
	ArtifactURL    string `json:"artifact_url"`
	MinHostVersion string `json:"min_host_version,omitempty"`
	SignatureURL   string `json:"signature_url,omitempty"`
	Name           string `json:"name,omitempty"`
}

// FetchChannel downloads a JSON array of ChannelEntry from an HTTPS URL.
func FetchChannel(ctx context.Context, channelURL string) ([]ChannelEntry, error) {
	return fetchChannelWithClient(ctx, channelURL, &http.Client{Timeout: 60 * time.Second})
}

func fetchChannelWithClient(ctx context.Context, channelURL string, client *http.Client) ([]ChannelEntry, error) {
	channelURL = strings.TrimSpace(channelURL)
	if channelURL == "" {
		return nil, fmt.Errorf("updates: empty channel url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, channelURL, nil)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := doWithRetry(ctx, client, req, 3)
	if err != nil {
		if isNetworkError(err) {
			return nil, fmt.Errorf("updates: network error while fetching channel (offline or DNS issue): %w", err)
		}
		return nil, fmt.Errorf("updates: fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("updates: HTTP %d while fetching channel: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	var entries []ChannelEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("updates: invalid channel JSON: %w", err)
	}
	return entries, nil
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
		// Small bounded backoff to smooth flaky connections.
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
