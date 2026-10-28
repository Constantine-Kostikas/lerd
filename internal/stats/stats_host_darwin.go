//go:build darwin

package stats

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

// readHostProcesses reports lerd's launchd jobs, measured from `ps` since launchd
// keeps no per-job accounting, plus the containers. With no jobs it returns
// nothing so Read's own podman fallback is the only podman read.
func readHostProcesses() ([]ContainerStat, error) {
	list, err := hostOutput("launchctl", "list")
	if err != nil {
		return nil, nil
	}
	procs, err := hostOutput("ps", "-axo", "pid=,ppid=,rss=,%cpu=")
	if err != nil {
		return nil, nil
	}
	jobs := launchdRows(list, procs, hostTotalRAM())
	if len(jobs) == 0 {
		return nil, nil
	}
	containers, _ := readerFn()
	return mergeJobs(containers, jobs), nil
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
