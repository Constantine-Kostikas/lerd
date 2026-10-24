package cli

import (
	"strings"
	"testing"
)

func TestCheckWSLSystemd(t *testing.T) {
	origWSL, origBooted := installIsWSL, installSystemdBooted
	t.Cleanup(func() { installIsWSL, installSystemdBooted = origWSL, origBooted })

	cases := []struct {
		name          string
		isWSL, booted bool
		wantErr       bool
	}{
		{"wsl without systemd", true, false, true},
		{"wsl with systemd", true, true, false},
		{"plain linux", false, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			installIsWSL = func() bool { return c.isWSL }
			installSystemdBooted = func() bool { return c.booted }
			err := checkWSLSystemd()
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "lerd wsl:setup") {
				t.Errorf("error does not point at wsl:setup: %v", err)
			}
		})
	}
}
