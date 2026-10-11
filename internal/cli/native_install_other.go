//go:build !windows

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/geodro/lerd/internal/config"
)

func tarBinary() string { return "tar" }

// placeNativeBuild moves the build's two static binaries into BinDir.
func placeNativeBuild(stage, version string) error {
	if err := os.MkdirAll(config.BinDir(), 0o755); err != nil {
		return err
	}
	for _, name := range []string{"php-native-" + version, "php-native-fpm-" + version} {
		src := filepath.Join(stage, name)
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("the php %s build is missing %s", version, name)
		}
		dest := filepath.Join(config.BinDir(), name)
		if err := os.Rename(src, dest); err != nil {
			return fmt.Errorf("installing %s: %w", name, err)
		}
		if err := os.Chmod(dest, 0o755); err != nil {
			return err
		}
	}
	return nil
}
