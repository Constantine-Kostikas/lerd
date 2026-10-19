package ui

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geodro/lerd/internal/cli"
)

func withStopSeams(t *testing.T, detect func() ([]cli.UnhealthyWorker, error), stop func(unit, label string) error) {
	t.Helper()
	oldDetect, oldStop, oldClear := detectUnhealthyFn, stopWorkerUnitFn, clearURLFileFn
	detectUnhealthyFn, stopWorkerUnitFn = detect, stop
	clearURLFileFn = func(string, string, string) {}
	t.Cleanup(func() { detectUnhealthyFn, stopWorkerUnitFn, clearURLFileFn = oldDetect, oldStop, oldClear })
}

// The banner's second action puts every worker it is reporting down for good,
// so a worker that keeps failing stops being restarted instead of being healed
// into the next crash.
func TestWorkersStopStopsEveryReportedUnit(t *testing.T) {
	var stopped []string
	withStopSeams(t,
		func() ([]cli.UnhealthyWorker, error) {
			return []cli.UnhealthyWorker{
				{Site: "shop", Worker: "queue", Unit: "lerd-queue-shop"},
				{Site: "shop", Worker: "vite", Unit: "lerd-vite-shop-feature"},
			}, nil
		},
		func(unit, _ string) error {
			stopped = append(stopped, unit)
			return nil
		})

	rec := httptest.NewRecorder()
	handleWorkersStop(rec, httptest.NewRequest(http.MethodPost, "/api/workers/stop", nil))

	var got struct {
		OK      bool `json:"ok"`
		Stopped int  `json:"stopped"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if !got.OK || got.Stopped != 2 {
		t.Errorf("want ok with 2 stopped, got %+v", got)
	}
	if len(stopped) != 2 || stopped[0] != "lerd-queue-shop" || stopped[1] != "lerd-vite-shop-feature" {
		t.Errorf("wrong units stopped: %v", stopped)
	}
}

// One unit refusing to go down must not hide the others that did, so the
// failures are reported alongside the count rather than as a bare error.
func TestWorkersStopReportsAFailureWithoutLosingTheRest(t *testing.T) {
	withStopSeams(t,
		func() ([]cli.UnhealthyWorker, error) {
			return []cli.UnhealthyWorker{
				{Site: "shop", Worker: "queue", Unit: "lerd-queue-shop"},
				{Site: "blog", Worker: "queue", Unit: "lerd-queue-blog"},
			}, nil
		},
		func(unit, _ string) error {
			if unit == "lerd-queue-blog" {
				return errors.New("unit file is read-only")
			}
			return nil
		})

	rec := httptest.NewRecorder()
	handleWorkersStop(rec, httptest.NewRequest(http.MethodPost, "/api/workers/stop", nil))

	var got struct {
		OK       bool `json:"ok"`
		Stopped  int  `json:"stopped"`
		Failures []struct {
			Unit  string `json:"unit"`
			Error string `json:"error"`
		} `json:"failures"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.OK {
		t.Error("a failed stop must not report ok")
	}
	if got.Stopped != 1 {
		t.Errorf("the healthy stop should still count, got %d", got.Stopped)
	}
	if len(got.Failures) != 1 || got.Failures[0].Unit != "lerd-queue-blog" {
		t.Errorf("want the blog unit reported, got %+v", got.Failures)
	}
}

func TestWorkersStopRejectsAGet(t *testing.T) {
	rec := httptest.NewRecorder()
	handleWorkersStop(rec, httptest.NewRequest(http.MethodGet, "/api/workers/stop", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}

// A dev server stopped from the banner must not leave its URL file pointing
// pages at it, and the file belongs to the checkout the unit ran in.
func TestWorkersStopClearsURLFileOfStoppedUnitsOnly(t *testing.T) {
	withStopSeams(t,
		func() ([]cli.UnhealthyWorker, error) {
			return []cli.UnhealthyWorker{
				{Site: "shop", Worker: "vite", Unit: "lerd-vite-shop-feature", Path: "/srv/shop/shop-feature"},
				{Site: "blog", Worker: "vite", Unit: "lerd-vite-blog", Path: "/srv/blog"},
			}, nil
		},
		func(unit, _ string) error {
			if unit == "lerd-vite-blog" {
				return errors.New("refused")
			}
			return nil
		})
	var cleared []string
	clearURLFileFn = func(site, dir, worker string) { cleared = append(cleared, site+" "+dir+" "+worker) }

	handleWorkersStop(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/workers/stop", nil))

	if len(cleared) != 1 || cleared[0] != "shop /srv/shop/shop-feature vite" {
		t.Errorf("want only the stopped worktree unit cleared, got %v", cleared)
	}
}
