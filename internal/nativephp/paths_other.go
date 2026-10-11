//go:build !windows

package nativephp

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/geodro/lerd/internal/config"
)

// moduleExt is the suffix of the loadable extensions a native build ships.
const moduleExt = ".so"

// BinaryPath is where lerd keeps the native PHP-FPM for a version. Suffixed
// with the minor because a project's composer.lock was resolved against one,
// and deliberately not "php": that name in BinDir is the shim into the
// container, and a static binary written over it would send every php on the
// machine to a PHP with no lerd behind it.
func BinaryPath(version string) string {
	return filepath.Join(config.BinDir(), "php-native-"+version)
}

// FPMBinaryPath is where lerd keeps the native PHP-FPM for a version. Separate
// from BinaryPath because FPM and CLI are different SAPIs and different builds:
// FPM serves requests, the CLI runs artisan, composer and tinker.
func FPMBinaryPath(version string) string {
	return filepath.Join(config.BinDir(), "php-native-fpm-"+version)
}

// ListInstalled returns the PHP versions with a usable native runtime on disk,
// sorted. A version needs both SAPIs: the CLI runs artisan, composer and the
// workers, and FPM serves requests, so a half-installed version would list as
// available and then fail at whichever half is missing.
func ListInstalled() []string {
	entries, err := os.ReadDir(config.BinDir())
	if err != nil {
		return nil
	}
	cli, fpm := map[string]bool{}, map[string]bool{}
	for _, e := range entries {
		if v, ok := strings.CutPrefix(e.Name(), "php-native-fpm-"); ok {
			fpm[v] = true
			continue
		}
		if v, ok := strings.CutPrefix(e.Name(), "php-native-"); ok {
			cli[v] = true
		}
	}
	var out []string
	for v := range cli {
		if fpm[v] {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

func isExecutable(info fs.FileInfo) bool { return info.Mode()&0111 != 0 }

// The static builds compile their extensions in, so there is no extension
// directory to point at and no ini of the build's own to scan.
func buildIniDirs(string) []string { return nil }

func buildIni(string) string { return "" }
