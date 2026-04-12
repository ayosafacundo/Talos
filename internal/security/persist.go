package security

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func SaveGrants(path string, grants map[string]map[string]bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(grants, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func LoadGrants(path string) (map[string]map[string]bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]map[string]bool{}, nil
		}
		return nil, err
	}

	grants := make(map[string]map[string]bool)
	if len(raw) == 0 {
		return grants, nil
	}
	if err := json.Unmarshal(raw, &grants); err != nil {
		return nil, err
	}
	return grants, nil
}
