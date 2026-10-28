//go:build darwin

package stats

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

// readHostProcesses reports the containers via podman plus lerd's launchd jobs:
// its daemons, native PHP-FPM masters and host-side workers. launchd keeps no
// per-job accounting, so each job is measured from `ps` over its process tree.
// CPU is ps's decaying average rather than a delta, which is close enough here.
func readHostProcesses() ([]ContainerStat, error) {
	rows, _ := readerFn()
	skip := map[string]bool{}
	for _, r := range rows {
		skip[r.Name] = true
	}
	jobs, err := hostOutput("launchctl", "list")
	if err != nil {
		return rows, nil
	}
	procs, err := hostOutput("ps", "-axo", "pid=,ppid=,rss=,%cpu=")
	if err != nil {
		return rows, nil
	}
	return append(rows, launchdRows(jobs, procs, hostTotalRAM(), skip)...), nil
}

func hostOutput(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return string(out), err
}

// hostTotalRAM is the Mac's physical memory, the denominator for host-job rows.
func hostTotalRAM() int64 {
	out, err := hostOutput("sysctl", "-n", "hw.memsize")
	if err != nil {
		return 0
	}
	n, _ := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	return n
}
