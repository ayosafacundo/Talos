package updates

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchChannel(t *testing.T) {
	t.Parallel()

	want := []ChannelEntry{{AppID: "app.x", Version: "1.0.0", ArtifactURL: "https://example.com/a.zip"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(want)
	}))
	t.Cleanup(srv.Close)

	got, err := FetchChannel(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].AppID != "app.x" {
		t.Fatalf("got %+v", got)
	}
}

func TestFetchChannel_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("gateway down"))
	}))
	t.Cleanup(srv.Close)

	_, err := FetchChannel(context.Background(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("expected HTTP error, got %v", err)
	}
}

func TestFetchChannel_InvalidJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"bad":true}`))
	}))
	t.Cleanup(srv.Close)

	_, err := FetchChannel(context.Background(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "invalid channel JSON") {
		t.Fatalf("expected JSON error, got %v", err)
	}
}

type flakyRoundTripper struct {
	failCount int
	calls     int
}

func (f *flakyRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	f.calls++
	if f.calls <= f.failCount {
		return nil, &net.DNSError{Err: "temporary", Name: "example.invalid", IsTemporary: true}
	}
	body := io.NopCloser(strings.NewReader(`[{"app_id":"app.x","version":"1.0.0","artifact_url":"https://example.com/a.zip"}]`))
	return &http.Response{StatusCode: http.StatusOK, Body: body, Header: make(http.Header)}, nil
}

func TestFetchChannel_RetryOnTransientNetworkError(t *testing.T) {
	t.Parallel()

	rt := &flakyRoundTripper{failCount: 2}
	client := &http.Client{Transport: rt}
	got, err := fetchChannelWithClient(context.Background(), "https://example.invalid/channel.json", client)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].AppID != "app.x" {
		t.Fatalf("unexpected result: %+v", got)
	}
	if rt.calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", rt.calls)
	}
}

type alwaysFailRoundTripper struct{}

func (alwaysFailRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, &net.DNSError{Err: "no such host", Name: "nohost.invalid"}
}

func TestFetchChannel_OfflineErrorMessage(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: alwaysFailRoundTripper{}}
	_, err := fetchChannelWithClient(context.Background(), "https://nohost.invalid/channel.json", client)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "offline or DNS issue") {
		t.Fatalf("expected actionable network hint, got %v", err)
	}
}

func TestIsNetworkError(t *testing.T) {
	t.Parallel()
	if !isNetworkError(&net.DNSError{Err: "x"}) {
		t.Fatal("expected dns error to be network error")
	}
	if isNetworkError(errors.New("plain")) {
		t.Fatal("plain error should not be treated as network error")
	}
}
