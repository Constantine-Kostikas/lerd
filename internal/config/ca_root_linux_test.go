package config

import (
	"path/filepath"
	"testing"
)

// Resolved as mkcert resolves it, so an install keeps the CA mkcert made.
func TestCARootDir(t *testing.T) {
	t.Setenv("CAROOT", "")
	t.Setenv("XDG_DATA_HOME", "/data")
	if got, _ := CARootDir(); got != filepath.Join("/data", "mkcert") {
		t.Errorf("CARootDir() = %q, want /data/mkcert", got)
	}
	t.Setenv("CAROOT", "/custom")
	if got, _ := CARootDir(); got != "/custom" {
		t.Errorf("CARootDir() = %q, want CAROOT to win", got)
	}
}
