package tokenstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tuanp-github/unified-ai-proxy/internal/model"
)

// Load reads a token file. A missing file yields a nil set and nil error.
func Load(path string) (*model.TokenSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read token file %s: %w", path, err)
	}
	ts := &model.TokenSet{}
	if err := json.Unmarshal(data, ts); err != nil {
		return nil, fmt.Errorf("parse token file %s: %w", path, err)
	}
	return ts, nil
}

// Save writes a token file with 0600 permissions inside a 0700 directory.
func Save(path string, ts *model.TokenSet) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create token directory: %w", err)
	}
	data, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("chmod token file: %w", err)
	}
	return nil
}

// Exists reports whether a token file exists on disk.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
