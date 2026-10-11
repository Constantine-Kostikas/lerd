//go:build !windows

package cli

import (
	"os/exec"
	"testing"
	"time"
)

// A tunnel that outlived its lerd-ui is actually killed.
func TestReapKillsSurvivingTunnel(t *testing.T) {
	withTunnelRecordFile(t)

	cmd := exec.Command("/bin/sleep", "37")
	cmd.SysProcAttr = tunnelSysProcAttr()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }() //nolint:errcheck

	recordTunnel(cmd.Process.Pid, cmd.Args)
	ReapOrphanTunnels()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		cmd.Process.Kill() //nolint:errcheck
		t.Fatal("orphaned tunnel process was not killed by the reap")
	}
}
