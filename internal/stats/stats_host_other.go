//go:build !linux && !darwin

package stats

// readHostProcesses is a no-op where lerd has neither systemd nor launchd units,
// so the dashboard there shows containers only.
func readHostProcesses() ([]ContainerStat, error) {
	return nil, nil
}
