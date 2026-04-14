package repository

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTP_List(t *testing.T) {
	t.Parallel()

	const body = `[{"id":"app.x","name":"X","install_url":"https://example.com/a.zip"}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	got, err := NewHTTP(srv.URL).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "app.x" || got[0].InstallURL != "https://example.com/a.zip" {
		t.Fatalf("%+v", got)
	}
}

func TestHTTP_ListHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("maintenance"))
	}))
	t.Cleanup(srv.Close)

	_, err := NewHTTP(srv.URL).List(context.Background())
	if err == nil || !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("expected HTTP 503 error, got %v", err)
	}
}

func TestHTTP_ListInvalidJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"bad":true}`))
	}))
	t.Cleanup(srv.Close)

	_, err := NewHTTP(srv.URL).List(context.Background())
	if err == nil || !strings.Contains(err.Error(), "invalid catalog JSON") {
		t.Fatalf("expected invalid JSON error, got %v", err)
	}
}

type repoFlakyRoundTripper struct {
	failCount int
	calls     int
}

func (f *repoFlakyRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	f.calls++
	if f.calls <= f.failCount {
		return nil, &net.DNSError{Err: "temporary", Name: "catalog.invalid", IsTemporary: true}
	}
	body := io.NopCloser(strings.NewReader(`[{"id":"app.x","name":"X"}]`))
	return &http.Response{StatusCode: http.StatusOK, Body: body, Header: make(http.Header)}, nil
}

func TestHTTP_ListRetriesTransientNetworkErrors(t *testing.T) {
	t.Parallel()

	rt := &repoFlakyRoundTripper{failCount: 2}
	h := NewHTTP("https://catalog.invalid/catalog.json")
	h.Client = &http.Client{Transport: rt}
	got, err := h.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "app.x" {
		t.Fatalf("unexpected rows: %+v", got)
	}
	if rt.calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", rt.calls)
	}
}

func TestHTTP_ListOfflineErrorMessage(t *testing.T) {
	t.Parallel()

	h := NewHTTP("https://catalog.invalid/catalog.json")
	h.Client = &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, &net.DNSError{Err: "no such host", Name: "catalog.invalid"}
		}),
	}
	_, err := h.List(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "offline or DNS issue") {
		t.Fatalf("expected actionable network hint, got %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
