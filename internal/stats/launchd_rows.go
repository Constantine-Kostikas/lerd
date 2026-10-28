package stats

import (
	"strconv"
	"strings"
)

// launchdRows turns `launchctl list` and `ps -axo pid=,ppid=,rss=,%cpu=` output
// into one row per running lerd launchd job, each charged with its whole process
// tree so an FPM master carries its pool. Names in skip are already reported as
// containers; their launchd job only wraps podman and would double count.
func launchdRows(launchctlOut, psOut string, totalRAM int64, skip map[string]bool) []ContainerStat {
	type proc struct {
		rss int64
		cpu float64
	}
	procs := map[int]proc{}
	children := map[int][]int{}
	for _, line := range strings.Split(psOut, "\n") {
		f := strings.Fields(line)
		if len(f) != 4 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		rss, err3 := strconv.ParseInt(f[2], 10, 64)
		cpu, err4 := strconv.ParseFloat(f[3], 64)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			continue
		}
		procs[pid] = proc{rss: rss * 1024, cpu: cpu}
		children[ppid] = append(children[ppid], pid)
	}

	var rows []ContainerStat
	for _, line := range strings.Split(launchctlOut, "\n") {
		f := strings.Fields(line)
		if len(f) != 3 {
			continue
		}
		name, ok := lerdJobName(f[2])
		if !ok || skip[name] {
			continue
		}
		pid, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		if _, ok := procs[pid]; !ok {
			continue
		}
		row := ContainerStat{Name: name, MemLimit: totalRAM}
		for stack := []int{pid}; len(stack) > 0; {
			p := stack[len(stack)-1]
			stack = append(stack[:len(stack)-1], children[p]...)
			row.MemBytes += procs[p].rss
			row.CPUPercent += procs[p].cpu
		}
		if totalRAM > 0 {
			row.MemPercent = float64(row.MemBytes) / float64(totalRAM) * 100
		}
		rows = append(rows, row)
	}
	return rows
}

// lerdJobName maps a launchd label to the row name the dashboard shows: daemons
// are labelled com.lerd.lerd-ui, native FPM masters plain lerd-native-php84.
func lerdJobName(label string) (string, bool) {
	name := strings.TrimPrefix(label, "com.lerd.")
	return name, strings.HasPrefix(name, "lerd-")
}
