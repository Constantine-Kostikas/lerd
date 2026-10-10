//go:build !windows

package nativephp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/geodro/lerd/internal/config"
)

// EnsureShim points ShimDir's php at the native CLI, replacing a stale link so
// a version switch or reinstall is picked up.
func EnsureShim(version, cliBinary string) (string, error) {
	dir := ShimDir(version)
	config.GuardRealWrite(dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	link := filepath.Join(dir, "php")
	if current, err := os.Readlink(link); err == nil && current == cliBinary {
		return dir, nil
	}
	if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err := os.Symlink(cliBinary, link); err != nil {
		return "", err
	}
	return dir, nil
}

func lookPathIn(name, pathList string) (string, error) {
	for _, dir := range filepath.SplitList(pathList) {
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s not found on the host PATH lerd uses (%s)", name, pathList)
}
