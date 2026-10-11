//go:build !windows

package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRemoveOmarchyGlance_RemovesAnInstalledPlugin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".config/omarchy/plugins/sh.lerd.glance"), 0o755); err != nil {
		t.Fatal(err)
	}
	calls := stubGlanceRemove(t)

	if found, err := removeOmarchyGlance(); !found || err != nil {
		t.Fatalf("removeOmarchyGlance = %v, %v, want a removal", found, err)
	}
	want := [][]string{{"omarchy-plugin-remove", "sh.lerd.glance", "--yes"}}
	if !reflect.DeepEqual(*calls, want) {
		t.Errorf("calls = %v, want %v", *calls, want)
	}
}
