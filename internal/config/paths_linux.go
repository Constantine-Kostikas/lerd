package config

import (
	"os"
	"path/filepath"
)

// LaunchAgentsDir is empty on Linux, whose unit dirs follow the XDG vars.
func LaunchAgentsDir() string { return "" }

// ProvidedEnvDir is where env_provider output is kept: under XDG_RUNTIME_DIR,
// which is tmpfs, so secrets never reach disk. Empty when the session has no
// runtime dir; the feature is off there.
func ProvidedEnvDir() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "lerd", "env")
}

// defaultCARootDir is mkcert's CAROOT under the XDG data directory.
func defaultCARootDir() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "mkcert"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "mkcert"), nil
}
