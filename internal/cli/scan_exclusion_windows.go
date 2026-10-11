package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/geodro/lerd/internal/config"
	"github.com/geodro/lerd/internal/feedback"
)

// Defender scans a file the first time anything opens it, which on a PHP
// project is thousands of files and seconds on the first request, whether
// php-cgi opens them natively or p9-serve does for a container.

type scanExclusionState struct {
	Excluded []string `json:"excluded,omitempty"`
	Declined []string `json:"declined,omitempty"`
}

// Seams so tests stay off Defender and UAC.
var (
	scanExclusionInteractive = isInteractive
	defenderActive           = func() bool {
		out, err := defenderQuery("(Get-MpComputerStatus).RealTimeProtectionEnabled")
		return err == nil && strings.TrimSpace(out) == "True"
	}
	// defenderExclusionPaths is empty for a non-admin, who Defender shows
	// "N/A: Must be an administrator..." instead of the list.
	defenderExclusionPaths = func() []string {
		out, err := defenderQuery("(Get-MpPreference).ExclusionPath")
		if err != nil {
			return nil
		}
		var paths []string
		for _, line := range strings.Split(out, "\n") {
			if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "N/A") {
				paths = append(paths, line)
			}
		}
		return paths
	}
	confirmScanExclusion = func(dir string) bool {
		feedback.Line("Windows Defender scans every PHP file the first time it is opened, which makes the first request after a change take seconds.")
		return feedback.Confirm(fmt.Sprintf("Exclude %s from Defender scanning? Windows will ask for administrator permission", dir), true)
	}
)

// offerScanExclusion asks once per folder to exclude it from Defender, and
// keeps the answer so a project inside an excluded or declined folder is not
// asked about again.
func offerScanExclusion(dir string) {
	if !scanExclusionInteractive() {
		return
	}
	st := loadScanExclusionState()
	if pathCovered(dir, st.Excluded) || pathCovered(dir, st.Declined) {
		return
	}
	if !defenderActive() || pathCovered(dir, defenderExclusionPaths()) {
		return
	}
	if !confirmScanExclusion(dir) {
		st.Declined = append(st.Declined, dir)
		saveScanExclusionState(st)
		return
	}
	if err := runElevated(addScanExclusionScript(dir)); err != nil {
		feedback.Warn("could not exclude %s from Defender (%v); from an elevated PowerShell run: Add-MpPreference -ExclusionPath %s", dir, err, psQuote(dir))
		return
	}
	st.Excluded = append(st.Excluded, dir)
	saveScanExclusionState(st)
	feedback.Line("excluded " + feedback.Val(dir) + " from Defender scanning")
}

func addScanExclusionScript(dir string) string {
	return "Add-MpPreference -ExclusionPath " + psQuote(dir) + " -ErrorAction Stop"
}

func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// pathCovered reports whether dir is one of roots or inside one, the way
// Windows compares paths: case-insensitively.
func pathCovered(dir string, roots []string) bool {
	d := strings.ToLower(filepath.Clean(dir)) + `\`
	for _, r := range roots {
		if r == "" {
			continue
		}
		root := strings.TrimSuffix(strings.ToLower(filepath.Clean(r)), `\`) + `\`
		if strings.HasPrefix(d, root) {
			return true
		}
	}
	return false
}

func defenderQuery(script string) (string, error) {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", encodePowerShell(script)).Output()
	return string(out), err
}

func scanExclusionStatePath() string {
	return filepath.Join(config.DataDir(), "defender-exclusions.json")
}

func loadScanExclusionState() scanExclusionState {
	var st scanExclusionState
	if data, err := os.ReadFile(scanExclusionStatePath()); err == nil {
		_ = json.Unmarshal(data, &st)
	}
	return st
}

func saveScanExclusionState(st scanExclusionState) {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	path := scanExclusionStatePath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, data, 0o644)
}
