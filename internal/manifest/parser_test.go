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
