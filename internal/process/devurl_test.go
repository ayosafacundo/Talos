package process

import (
	"strings"
	"testing"
)

func TestParseDevServerBaseURL_Vite(t *testing.T) {
	log := `
  VITE v5.0.0  ready in 120 ms

  ➜  Local:   http://127.0.0.1:5173/
  ➜  Network: use --host to expose
`
	got := ParseDevServerBaseURL(log)
	if got != "http://127.0.0.1:5173/" {
		t.Fatalf("got %q", got)
	}
}

func TestParseDevServerBaseURL_LocalPlain(t *testing.T) {
	log := `Local: http://localhost:3000/foo`
	got := ParseDevServerBaseURL(log)
	if !strings.HasPrefix(got, "http://localhost:3000") {
		t.Fatalf("got %q", got)
	}
}

func TestParseDevServerBaseURL_IgnoresNonLoopback(t *testing.T) {
	log := `Local: http://192.168.1.1:8080/`
	got := ParseDevServerBaseURL(log)
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestAlignDiscoveredDevURL_LocalhostTo127(t *testing.T) {
	got := AlignDiscoveredDevURL("http://127.0.0.1:5173/", "http://localhost:5174/")
	want := "http://127.0.0.1:5174/"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAlignDiscoveredDevURL_127ToLocalhostManifest(t *testing.T) {
	got := AlignDiscoveredDevURL("http://localhost:5173/", "http://127.0.0.1:5174/")
	want := "http://localhost:5174/"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExpandLoopbackOrigins(t *testing.T) {
	out := ExpandLoopbackOrigins([]string{"http://127.0.0.1:5174"})
	if len(out) != 2 {
		t.Fatalf("want 2 origins, got %v", out)
	}
	hasLocal := false
	has127 := false
	for _, o := range out {
		if o == "http://127.0.0.1:5174" {
			has127 = true
		}
		if o == "http://localhost:5174" {
			hasLocal = true
		}
	}
	if !has127 || !hasLocal {
		t.Fatalf("missing twin: %v", out)
	}
}
