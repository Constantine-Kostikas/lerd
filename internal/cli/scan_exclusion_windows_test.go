package cli

import (
	"errors"
	"strings"
	"testing"
)

type scanExclusionFake struct {
	interactive bool
	active      bool
	existing    []string
	confirm     bool
	elevateErr  error
	asked       int
	scripts     []string
}

func useScanExclusionFake(t *testing.T, f *scanExclusionFake) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	prevI, prevA, prevE, prevC, prevR := scanExclusionInteractive, defenderActive, defenderExclusionPaths, confirmScanExclusion, runElevated
	t.Cleanup(func() {
		scanExclusionInteractive, defenderActive, defenderExclusionPaths, confirmScanExclusion, runElevated = prevI, prevA, prevE, prevC, prevR
	})
	scanExclusionInteractive = func() bool { return f.interactive }
	defenderActive = func() bool { return f.active }
	defenderExclusionPaths = func() []string { return f.existing }
	confirmScanExclusion = func(string) bool { f.asked++; return f.confirm }
	runElevated = func(script string) error { f.scripts = append(f.scripts, script); return f.elevateErr }
}

func TestOfferScanExclusion_addsTheFolderOnceConfirmed(t *testing.T) {
	f := &scanExclusionFake{interactive: true, active: true, confirm: true}
	useScanExclusionFake(t, f)

	offerScanExclusion(`C:\Sites`)
	offerScanExclusion(`C:\Sites\app`)

	if f.asked != 1 || len(f.scripts) != 1 {
		t.Fatalf("asked %d times, elevated %d times, want 1 and 1", f.asked, len(f.scripts))
	}
	if want := "Add-MpPreference -ExclusionPath 'C:\\Sites'"; !strings.Contains(f.scripts[0], want) {
		t.Errorf("script = %q, want it to contain %q", f.scripts[0], want)
	}
}

func TestOfferScanExclusion_remembersADecline(t *testing.T) {
	f := &scanExclusionFake{interactive: true, active: true, confirm: false}
	useScanExclusionFake(t, f)

	offerScanExclusion(`C:\Sites`)
	offerScanExclusion(`c:\sites\app`)

	if f.asked != 1 || len(f.scripts) != 0 {
		t.Errorf("asked %d times, elevated %d times, want 1 and 0", f.asked, len(f.scripts))
	}
}

func TestOfferScanExclusion_asksAgainWhenElevationFails(t *testing.T) {
	f := &scanExclusionFake{interactive: true, active: true, confirm: true, elevateErr: errors.New("cancelled")}
	useScanExclusionFake(t, f)

	offerScanExclusion(`C:\Sites`)
	offerScanExclusion(`C:\Sites`)

	if f.asked != 2 {
		t.Errorf("asked %d times, want 2: a cancelled UAC prompt is not a decision", f.asked)
	}
}

func TestOfferScanExclusion_staysQuiet(t *testing.T) {
	cases := map[string]scanExclusionFake{
		"no terminal":             {interactive: false, active: true, confirm: true},
		"defender off":            {interactive: true, active: false, confirm: true},
		"already excluded":        {interactive: true, active: true, confirm: true, existing: []string{`C:\`}},
		"excluded by an ancestor": {interactive: true, active: true, confirm: true, existing: []string{`C:\Users\me`}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := c
			useScanExclusionFake(t, &f)
			offerScanExclusion(`C:\Users\me\Sites`)
			if f.asked != 0 || len(f.scripts) != 0 {
				t.Errorf("asked %d times, elevated %d times, want neither", f.asked, len(f.scripts))
			}
		})
	}
}

func TestPathCovered(t *testing.T) {
	cases := []struct {
		dir   string
		roots []string
		want  bool
	}{
		{`C:\Sites\app`, []string{`C:\Sites`}, true},
		{`c:\sites\APP\`, []string{`C:\Sites\`}, true},
		{`C:\Sites`, []string{`C:\Sites`}, true},
		{`C:\Sites2\app`, []string{`C:\Sites`}, false},
		{`C:\Sites`, []string{`C:\Sites\app`}, false},
		{`C:\Sites`, []string{""}, false},
	}
	for _, c := range cases {
		if got := pathCovered(c.dir, c.roots); got != c.want {
			t.Errorf("pathCovered(%q, %q) = %v, want %v", c.dir, c.roots, got, c.want)
		}
	}
}

func TestAddScanExclusionScript_quotesThePath(t *testing.T) {
	got := addScanExclusionScript(`C:\Users\o'neil\Sites`)
	if want := `Add-MpPreference -ExclusionPath 'C:\Users\o''neil\Sites'`; !strings.Contains(got, want) {
		t.Errorf("script = %q, want it to contain %q", got, want)
	}
}
