//go:build windows

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/geodro/lerd/internal/nativephp"
)

// tarBinary is Windows' own bsdtar. Git's GNU tar can come first on PATH, and
// it reads the "C:" of an archive path as a remote host.
func tarBinary() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		return "tar"
	}
	return filepath.Join(root, "System32", "tar.exe")
}

// placeNativeBuild installs the official build directory and the ini that
// turns its extensions on. Windows will not rename a directory holding a
// running php-cgi.exe, so the pool comes down first; callers restart it.
func placeNativeBuild(stage, version string) error {
	src := filepath.Join(stage, "php-native-"+version)
	if _, err := os.Stat(filepath.Join(src, "php-cgi.exe")); err != nil {
		return fmt.Errorf("the php %s build is missing php-cgi.exe", version)
	}
	if nativephp.Loaded(version) {
		if err := nativephp.Stop(version); err != nil {
			return err
		}
	}
	if err := replaceDir(src, nativephp.BuildDir(version)); err != nil {
		return fmt.Errorf("installing php %s: %w", version, err)
	}
	if err := replaceDir(filepath.Join(stage, "conf.d"), nativephp.BuildConfDir(version)); err != nil {
		return fmt.Errorf("installing the ini for php %s: %w", version, err)
	}
	return nil
}

// replaceDir swaps src in for dest, keeping the old copy until the new one is
// in place so a failure leaves the previous build serving.
func replaceDir(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	old := dest + ".old"
	os.RemoveAll(old)
	if _, err := os.Stat(dest); err == nil {
		if err := os.Rename(dest, old); err != nil {
			return err
		}
	}
	if err := os.Rename(src, dest); err != nil {
		os.Rename(old, dest) //nolint:errcheck
		return err
	}
	return os.RemoveAll(old)
}
