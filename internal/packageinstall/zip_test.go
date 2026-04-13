package packageinstall

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZipSecure_RejectsZipSlip(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	w, err := zw.Create("../evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := ExtractZipSecure(zr, dest); err == nil {
		t.Fatal("expected ZipSlip error")
	}
}

func TestExtractZipSecure_AllowsManifestAndDist(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for _, name := range []string{"dist/", "dist/index.html", "manifest.yaml"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if name != "dist/" {
			if _, err := w.Write([]byte("<!doctype html><title>x</title>")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := ExtractZipSecure(zr, dest); err != nil {
		t.Fatalf("extract: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "manifest.yaml")); err != nil {
		t.Fatal(err)
	}
}
