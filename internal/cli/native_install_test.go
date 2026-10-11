package cli

import (
	"testing"
)

func TestNativeUpdatePlan(t *testing.T) {
	const (
		older = "2026-09-21T06:00:00Z"
		newer = "2026-09-28T12:00:00Z"
	)
	cases := []struct {
		name               string
		pinned, installed  string
		publishedAt        string
		installedPublished string
		want               string
		wantUpdate         bool
	}{
		{"a newer patch is published", "8.4.24", "8.4.23", "", "", "8.4.24", true},
		{"already on the pinned patch", "8.4.24", "8.4.24", "", "", "8.4.24", false},
		{"nothing installed yet", "8.4.24", "", "", "", "8.4.24", true},
		// The pin is the only statement of what lerd publishes, so a machine
		// ahead of it is brought back to it rather than left on a build that is
		// no longer offered.
		{"installed ahead of the pin", "8.4.23", "8.4.24", "", "", "8.4.23", true},
		{"no pin at all", "", "8.4.24", "", "", "", false},
		// The same patch rebuilt to carry a newer collector.
		{"the patch was rebuilt", "8.4.24", "8.4.24", newer, older, "8.4.24", true},
		{"the same build", "8.4.24", "8.4.24", newer, newer, "8.4.24", false},
		{"installed predates the date", "8.4.24", "8.4.24", newer, "", "8.4.24", true},
		{"no date either side", "8.4.24", "8.4.24", "", "", "8.4.24", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, update := nativeUpdatePlan(tc.pinned, tc.installed, tc.publishedAt, tc.installedPublished)
			if got != tc.want || update != tc.wantUpdate {
				t.Errorf("nativeUpdatePlan(%q, %q) = (%q, %v), want (%q, %v)",
					tc.pinned, tc.installed, got, update, tc.want, tc.wantUpdate)
			}
		})
	}
}
