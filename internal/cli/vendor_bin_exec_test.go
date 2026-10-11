package cli

import (
	"strings"
	"testing"
)

// ssh in the container ignores HOME and would only read root's known_hosts,
// so every exec route has to hand it the user's own file.
func TestContainerExecEnvArgs_PointsSSHAtTheUserKnownHosts(t *testing.T) {
	t.Setenv("HOME", "/home/u")
	args := strings.Join(containerExecEnvArgs("/home/u/site"), " ")
	want := "GIT_SSH_COMMAND=ssh -o UserKnownHostsFile='/home/u/.ssh/known_hosts'"
	if !strings.Contains(args, want) {
		t.Errorf("env args = %q, want %q", args, want)
	}
}
