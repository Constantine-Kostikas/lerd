package ui

import (
	"net/http"

	"github.com/geodro/lerd/internal/cli"
)

// Seams so the handler can be exercised without a live systemd.
var (
	detectUnhealthyFn = cli.DetectUnhealthyWorkers
	stopWorkerUnitFn  = cli.StopWorkerUnit
	clearURLFileFn    = cli.ClearStoppedWorkerURLFile
)

// stopFailure is one unit that would not go down.
type stopFailure struct {
	Unit  string `json:"unit"`
	Error string `json:"error"`
}

// handleWorkersStop puts every worker the health detector is currently
// reporting down for good. Healing a worker that keeps crashing only buys the
// next crash, so the banner needs the other answer too: stop is the same
// teardown the site's own toggle performs, and a disabled unit is one the
// detector deliberately leaves alone afterwards.
func handleWorkersStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	unhealthy, err := detectUnhealthyFn()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	stopped := 0
	failures := []stopFailure{}
	for _, u := range unhealthy {
		if err := stopWorkerUnitFn(u.Unit, u.Worker); err != nil {
			failures = append(failures, stopFailure{Unit: u.Unit, Error: err.Error()})
			continue
		}
		clearURLFileFn(u.Site, u.Path, u.Worker)
		stopped++
	}
	writeJSON(w, map[string]any{
		"ok":       len(failures) == 0,
		"stopped":  stopped,
		"failures": failures,
	})
}
