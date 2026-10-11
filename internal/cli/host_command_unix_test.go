//go:build !windows

package cli

import (
	"testing"
)

// The bundled runtime a project ships is not always a full PHP build: the Linux
// php-bin NativePHP ships carries neither posix nor pcntl, and Jump's websocket
// bridge is Workerman, which calls both unguarded and dies on the first line.
func TestMissingPHPExtensions_reportsWhatTheDeclaredBinaryLacks(t *testing.T) {
	bin := stubPHPModules(t, "[PHP Modules]\nCore\njson\nposix\nstandard\n")

	got := missingPHPExtensions(bin, []string{"posix", "pcntl"})

	if len(got) != 1 || got[0] != "pcntl" {
		t.Errorf("missing = %v, want [pcntl]", got)
	}
}
