//go:build !windows

package cli

import (
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

func TestRunShareTool_setShowReset(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("PATH", fakeBin(t, "ngrok"))

	if err := runShareTool(nil, []string{"ngrok"}); err != nil {
		t.Fatalf("setting ngrok: %v", err)
	}
	cfg, err := config.LoadGlobal()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if cfg.Share.DefaultTool != "ngrok" {
		t.Errorf("DefaultTool = %q, want ngrok", cfg.Share.DefaultTool)
	}

	// Showing the default also has to say how to change it, otherwise the command
	// reports a value with no way to act on it.
	out := captureStdout(t, func() {
		if err := runShareTool(nil, nil); err != nil {
			t.Errorf("showing: %v", err)
		}
	})
	if !strings.Contains(out, "ngrok") {
		t.Errorf("show output %q does not report the current tool", out)
	}
	if !strings.Contains(out, "lerd share:tool") {
		t.Errorf("show output %q does not say how to change it", out)
	}

	if err := runShareTool(nil, []string{"auto"}); err != nil {
		t.Fatalf("resetting: %v", err)
	}
	cfg, err = config.LoadGlobal()
	if err != nil {
		t.Fatalf("reloading config: %v", err)
	}
	if cfg.Share.DefaultTool != "" {
		t.Errorf("DefaultTool = %q after auto, want it cleared", cfg.Share.DefaultTool)
	}
}
