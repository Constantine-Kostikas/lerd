package cli

import (
	"testing"
)

func stubGlanceRemove(t *testing.T) *[][]string {
	t.Helper()
	var calls [][]string
	prev := runGlanceRemove
	runGlanceRemove = func(args ...string) error {
		calls = append(calls, args)
		return nil
	}
	t.Cleanup(func() { runGlanceRemove = prev })
	return &calls
}

func TestRemoveOmarchyGlance_NothingWithoutThePlugin(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	calls := stubGlanceRemove(t)

	if found, _ := removeOmarchyGlance(); found {
		t.Error("removeOmarchyGlance claimed a removal with no plugin on disk")
	}
	if len(*calls) != 0 {
		t.Errorf("calls = %v, want none", *calls)
	}
}
