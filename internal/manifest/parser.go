package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const ManifestFileName = "manifest.yaml"

var (
	ErrMissingID   = errors.New("manifest: id is required")
	ErrMissingName = errors.New("manifest: name is required")
)

// Definition captures the expected schema for a package manifest.
type Definition struct {
	ID            string   `yaml:"id" json:"id"`
	Name          string   `yaml:"name" json:"name"`
	Version       string   `yaml:"version,omitempty" json:"version,omitempty"`
	Icon          string   `yaml:"icon,omitempty" json:"icon,omitempty"`
	Binary        string   `yaml:"binary,omitempty" json:"binary,omitempty"`
	WebEntry      string   `yaml:"web_entry,omitempty" json:"web_entry,omitempty"`
	Permissions   []string `yaml:"permissions,omitempty" json:"permissions,omitempty"`
	MultiInstance bool     `yaml:"multi_instance" json:"multi_instance"`
}

// Parse reads and validates a raw manifest.yaml payload.
func Parse(data []byte) (*Definition, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var def Definition
	if err := decoder.Decode(&def); err != nil {
		return nil, fmt.Errorf("manifest: decode failed: %w", err)
	}

	if err := def.Validate(); err != nil {
		return nil, err
	}

	return &def, nil
}

// ParseFile reads and validates a manifest file from disk.
func ParseFile(manifestPath string) (*Definition, error) {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("manifest: read %q failed: %w", manifestPath, err)
	}

	return Parse(raw)
}

// ParsePackageDir reads and validates the manifest from a package directory.
func ParsePackageDir(packageDir string) (*Definition, error) {
	manifestPath := filepath.Join(packageDir, ManifestFileName)
	def, err := ParseFile(manifestPath)
	if err != nil {
		return nil, err
	}

	return def, nil
}

// Validate enforces required fields and basic safety constraints.
func (m *Definition) Validate() error {
	m.ID = strings.TrimSpace(m.ID)
	m.Name = strings.TrimSpace(m.Name)
	m.Binary = filepath.Clean(strings.TrimSpace(m.Binary))
	m.Icon = strings.TrimSpace(m.Icon)
	m.WebEntry = strings.TrimSpace(m.WebEntry)

	if m.ID == "" {
		return ErrMissingID
	}
	if m.Name == "" {
		return ErrMissingName
	}
	if m.Binary == "." {
		m.Binary = ""
	}
	if m.Binary != "" && filepath.IsAbs(m.Binary) {
		return errors.New("manifest: binary must be a relative path")
	}
	if m.WebEntry == "" {
		m.WebEntry = "dist/index.html"
	}
	if filepath.IsAbs(m.WebEntry) {
		return errors.New("manifest: web_entry must be a relative path")
	}

	return nil
}
