//go:build !windows

package cli

import (
	"testing"
)

// TestRunBun_SuccessReturnsNil confirms the happy path still returns nil.
func TestRunBun_SuccessReturnsNil(t *testing.T) {
	if err := runBun(t.TempDir(), "/bin/sh", []string{"-c", "exit 0"}, false); err != nil {
		t.Fatalf("expected nil on success, got %v", err)
	}
}
