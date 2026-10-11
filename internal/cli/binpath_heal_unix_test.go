//go:build !windows

package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// A Homebrew upgrade retires the keg the shims were written with, so every shim
// execs a path that is no longer there. Starting lerd points them back at the
// binary that is actually installed.
func TestHealShimBinaryPathsRepointsShimsAtTheLiveBinary(t *testing.T) {
	binDir, current := shimHealFixture(t)
	gone := "/home/linuxbrew/.linuxbrew/Cellar/lerd/1.31.0/bin/lerd"
	writeShim(t, binDir, "php", "#!/bin/sh\nLERD=\""+gone+"\"\n[ -x \"$LERD\" ] || LERD=lerd\nexec \"$LERD\" php \"$@\"\n")
	writeShim(t, binDir, "mysql", "#!/bin/sh\n# lerd client shim\nexec "+gone+" client-exec mysql \"$@\"\n")

	healShimBinaryPaths(current, binaryGone)

	for _, tool := range []string{"php", "mysql"} {
		shim := readShim(t, binDir, tool)
		if strings.Contains(shim, gone) {
			t.Errorf("%s shim still runs the retired %s:\n%s", tool, gone, shim)
		}
		if !strings.Contains(shim, current) {
			t.Errorf("%s shim does not run %s:\n%s", tool, current, shim)
		}
	}
}

// Paths that are not a lerd binary stay untouched, notably the composer.phar
// the composer shim falls back to.
func TestHealShimBinaryPathsKeepsNonBinaryPaths(t *testing.T) {
	binDir, current := shimHealFixture(t)
	gone := "/opt/homebrew/Cellar/lerd/1.31.0/bin/lerd"
	phar := filepath.Join(binDir, "composer.phar")
	writeShim(t, binDir, "composer", "#!/bin/sh\nLERD=\""+gone+"\"\nexec \"$LERD\" php "+phar+" \"$@\"\n")

	healShimBinaryPaths(current, binaryGone)

	shim := readShim(t, binDir, "composer")
	if !strings.Contains(shim, phar) {
		t.Errorf("composer shim lost the phar path %s:\n%s", phar, shim)
	}
	if !strings.Contains(shim, current) {
		t.Errorf("composer shim does not run %s:\n%s", current, shim)
	}
}
