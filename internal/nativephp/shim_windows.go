package nativephp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/geodro/lerd/internal/config"
)

// EnsureShim writes ShimDir's php.cmd, which runs the native CLI with the same
// ini scan the pool uses. A symlink would need Developer Mode or elevation, and
// cmd and PowerShell only resolve a bare php through a PATHEXT extension.
func EnsureShim(version, cliBinary string) (string, error) {
	dir := ShimDir(version)
	config.GuardRealWrite(dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	_, err := writeIfChanged(filepath.Join(dir, "php.cmd"), phpCmdShim(version, cliBinary))
	return dir, err
}

func phpCmdShim(version, cliBinary string) string {
	return "@echo off\r\n" +
		fmt.Sprintf("set \"PHP_INI_SCAN_DIR=%s\"\r\n", IniScanDir(version)) +
		fmt.Sprintf("\"%s\" %%*\r\n", cliBinary)
}

// lookPathIn resolves name the way cmd does: a path is taken as given, a bare
// name is tried with each PATHEXT extension in every directory of pathList.
func lookPathIn(name, pathList string) (string, error) {
	if strings.ContainsAny(name, `\/:`) {
		return name, nil
	}
	exts := filepath.SplitList(os.Getenv("PATHEXT"))
	if len(exts) == 0 {
		exts = []string{".com", ".exe", ".bat", ".cmd"}
	}
	if filepath.Ext(name) != "" {
		exts = append([]string{""}, exts...)
	}
	for _, dir := range filepath.SplitList(pathList) {
		for _, ext := range exts {
			p := filepath.Join(dir, name+ext)
			if info, err := os.Stat(p); err == nil && info.Mode().IsRegular() {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("%s not found on the host PATH lerd uses (%s)", name, pathList)
}
