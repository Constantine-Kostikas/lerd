//go:build !windows

package tools

func platformBinPath(string) (string, bool) { return "", false }
