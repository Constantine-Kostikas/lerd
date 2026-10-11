//go:build !windows

package cli

import (
	"testing"
)

// The host binary is preferred whenever it is there: it carries the user's own
// ngrok configuration, which a container cannot see.
func TestNgrokRunnerPrefersTheHostBinary(t *testing.T) {
	dir := t.TempDir()
	writeFakeExe(t, dir+"/ngrok")
	t.Setenv("PATH", dir)

	runner, err := ngrokRunnerFor("token-abc")
	if err != nil {
		t.Fatalf("ngrokRunnerFor() error: %v", err)
	}
	if runner.container {
		t.Fatal("host ngrok is installed, the container should not have been chosen")
	}
}
