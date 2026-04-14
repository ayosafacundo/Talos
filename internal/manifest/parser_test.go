package manifest

import "testing"

func TestParse_ValidManifest(t *testing.T) {
	t.Parallel()

	raw := []byte(`
id: app.calculator
name: Calculator
version: "1.0.0"
icon: web/icon.png
binary: bin/calculator
web_entry: web/index.html
permissions:
  - fs:data
  - net:deny
multi_instance: true
`)

	def, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if def.ID != "app.calculator" {
		t.Fatalf("expected id app.calculator, got %q", def.ID)
	}
	if !def.MultiInstance {
		t.Fatalf("expected multi_instance=true")
	}
}

func TestParse_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	raw := []byte(`
name: No ID
binary: bin/noid
`)

	_, err := Parse(raw)
	if err == nil {
		t.Fatalf("expected Parse() error")
	}
	if err != ErrMissingID {
		t.Fatalf("expected ErrMissingID, got %v", err)
	}
}

func TestParse_AbsoluteBinaryRejected(t *testing.T) {
	t.Parallel()

	raw := []byte(`
id: app.abs
name: Absolute Binary
binary: /usr/bin/something
`)

	_, err := Parse(raw)
	if err == nil {
		t.Fatalf("expected Parse() error")
	}
}

func TestParse_DevelopmentBlockValid(t *testing.T) {
	t.Parallel()

	raw := []byte(`
id: app.dev
name: Dev App
web_entry: dist/index.html
development:
  command: ["npm", "run", "dev"]
  url: "http://127.0.0.1:5174"
  allowed_origins:
    - "http://127.0.0.1:5174"
`)

	def, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if def.Development == nil || def.Development.URL == "" {
		t.Fatal("expected development block")
	}
}

func TestParse_DevelopmentCommandRequiresURL(t *testing.T) {
	t.Parallel()

	raw := []byte(`
id: app.dev
name: Dev App
web_entry: dist/index.html
development:
  command: ["npm", "run", "dev"]
`)

	def, err := Parse(raw)
	if err != nil {
		t.Fatalf("expected command-only development block to be valid, got %v", err)
	}
	if def.Development == nil || len(def.Development.Command) == 0 {
		t.Fatal("expected development command to be preserved")
	}
}

func TestParse_DevelopmentURLMustBeLoopback(t *testing.T) {
	t.Parallel()

	raw := []byte(`
id: app.dev
name: Dev App
web_entry: dist/index.html
development:
  url: "http://example.com/"
  allowed_origins:
    - "http://example.com"
`)

	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected error for non-loopback development url")
	}
}
