package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── removeMarkedBlock ────────────────────────────────────────────────────────

const testMarker = "# Added by Lerd installer"

func TestRemoveMarkedBlock_removesMarkerAndNextLine(t *testing.T) {
	tmp := t.TempDir()
	rc := filepath.Join(tmp, ".bashrc")

	content := "existing line\n" +
		testMarker + "\n" +
		`export PATH="/home/user/.local/share/lerd/bin:$PATH"` + "\n" +
		"another line\n"

	os.WriteFile(rc, []byte(content), 0644)
	removeMarkedBlock(rc, testMarker, 1)

	got, _ := os.ReadFile(rc)
	if strings.Contains(string(got), testMarker) {
		t.Error("marker line should have been removed")
	}
	if strings.Contains(string(got), "lerd/bin") {
		t.Error("PATH export line should have been removed")
	}
	if !strings.Contains(string(got), "existing line") {
		t.Error("unrelated lines should be preserved")
	}
	if !strings.Contains(string(got), "another line") {
		t.Error("lines after the block should be preserved")
	}
}

func TestRemoveMarkedBlock_noMarker_noChange(t *testing.T) {
	tmp := t.TempDir()
	rc := filepath.Join(tmp, ".bashrc")

	content := "line one\nline two\n"
	os.WriteFile(rc, []byte(content), 0644)
	removeMarkedBlock(rc, testMarker, 1)

	got, _ := os.ReadFile(rc)
	if string(got) != content {
		t.Errorf("file should be unchanged, got:\n%s", got)
	}
}

func TestRemoveMarkedBlock_missingFile_noError(t *testing.T) {
	// Must not panic or return an error — the function is best-effort.
	removeMarkedBlock("/tmp/lerd-test-nonexistent-file-xyz", testMarker, 1)
}

func TestRemoveMarkedBlock_markerAtEndOfFile(t *testing.T) {
	tmp := t.TempDir()
	rc := filepath.Join(tmp, ".zshrc")

	content := "source ~/.profile\n" + testMarker + "\n"
	os.WriteFile(rc, []byte(content), 0644)
	removeMarkedBlock(rc, testMarker, 1)

	got, _ := os.ReadFile(rc)
	if strings.Contains(string(got), testMarker) {
		t.Error("marker should have been removed")
	}
	if !strings.Contains(string(got), "source ~/.profile") {
		t.Error("preceding lines should be preserved")
	}
}

func TestRemoveMarkedBlock_onlyMarker(t *testing.T) {
	tmp := t.TempDir()
	rc := filepath.Join(tmp, ".bashrc")

	os.WriteFile(rc, []byte(testMarker+"\n"), 0644)
	removeMarkedBlock(rc, testMarker, 1)

	got, _ := os.ReadFile(rc)
	if strings.Contains(string(got), testMarker) {
		t.Error("marker should have been removed from single-line file")
	}
}

// ── removeShellEntry ─────────────────────────────────────────────────────────

func TestRemoveShellEntry_noRcFiles_noError(t *testing.T) {
	// Point HOME at an empty dir — no rc files exist, should not panic.
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	removeShellEntry() // must not panic
}

// ── removeInstalledBinaries ──────────────────────────────────────────────────

// mkbin creates an executable stand-in at path.
func mkbin(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func TestRemoveInstalledBinaries_removesLerdAndTray(t *testing.T) {
	dir := t.TempDir()
	self := filepath.Join(dir, "lerd")
	tray := filepath.Join(dir, "lerd-tray")
	mkbin(t, self)
	mkbin(t, tray)

	removeInstalledBinaries(self)

	if _, err := os.Stat(self); !os.IsNotExist(err) {
		t.Errorf("lerd binary still present")
	}
	if _, err := os.Stat(tray); !os.IsNotExist(err) {
		t.Errorf("lerd-tray still present: uninstall must not leave a tray a desktop entry can launch")
	}
}

func TestRemoveInstalledBinaries_missingTrayIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	self := filepath.Join(dir, "lerd")
	mkbin(t, self)

	removeInstalledBinaries(self)

	if _, err := os.Stat(self); !os.IsNotExist(err) {
		t.Errorf("lerd binary still present")
	}
}

func TestRemoveInstalledBinaries_leavesUnrelatedNeighbours(t *testing.T) {
	dir := t.TempDir()
	self := filepath.Join(dir, "lerd")
	other := filepath.Join(dir, "lerdfoo")
	mkbin(t, self)
	mkbin(t, other)

	removeInstalledBinaries(self)

	if _, err := os.Stat(other); err != nil {
		t.Errorf("removed an unrelated neighbour %s: %v", other, err)
	}
}

// ── removeDataDir ────────────────────────────────────────────────────────────

// undeletableDir builds a tree os.RemoveAll cannot clear: the inner directory
// has no write bit, which is what a subuid-owned service tree looks like to us.
func undeletableDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "data")
	sub := filepath.Join(dir, "redis")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "dump.rdb"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sub, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sub, 0755) })
	return dir
}

// fakePodman puts a podman on PATH standing in for the user namespace. It is
// handed the one directory the test built and refuses anything else, so a
// mistake here can never widen into a path the test does not own.
func fakePodman(t *testing.T, owned, script string) {
	t.Helper()
	dir := t.TempDir()
	guard := "#!/bin/sh\ncase \"$4\" in\n  " + owned + ") ;;\n  *) echo \"refusing $4\" >&2; exit 2 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(dir, "podman"), []byte(guard+script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestRemoveDataDir_removesAPlainTree(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(filepath.Join(dir, "mysql"), 0755); err != nil {
		t.Fatal(err)
	}

	if kept := removeDataDir(dir); kept != "" {
		t.Errorf("removeDataDir = %q, want an empty string", kept)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("%s still present", dir)
	}
}

func TestRemoveDataDir_fallsBackToPodmanUnshare(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root removes the tree without the fallback")
	}
	dir := undeletableDir(t)
	fakePodman(t, dir, "chmod -R u+w \"$4\" && rm -rf \"$4\"\n")

	if kept := removeDataDir(dir); kept != "" {
		t.Errorf("removeDataDir = %q, want an empty string", kept)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("%s survived the podman unshare fallback", dir)
	}
}

// ── removeScriptInstalledBinaries ─────────────────────────────────────────────

// Nothing here may reach the binary the package manager is responsible for,
// which is the whole point of the guard it runs under.
func TestRemoveScriptInstalledBinaries_neverTouchesThePackagedBinary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	self := filepath.Join(binDir, "lerd")
	mkbin(t, self)

	removeScriptInstalledBinaries(self)

	if _, err := os.Stat(self); err != nil {
		t.Error("the running binary is the package manager's to remove, not ours")
	}
}

// ── removeServiceUnits ───────────────────────────────────────────────────────

// unitListMgr answers the unit listings the removal walks and records nothing
// else; the fake it embeds covers the rest of the interface.
type unitListMgr struct {
	*fakeServiceMgr
	containers []string
	services   []string
	timers     []string
}

func (m *unitListMgr) ListContainerUnits(string) []string { return m.containers }
func (m *unitListMgr) ListServiceUnits(string) []string   { return m.services }
func (m *unitListMgr) ListTimerUnits(string) []string     { return m.timers }

// A unit that is already failed when its file goes stays behind as failed and
// not-found, so an uninstalled lerd went on being listed by systemctl on a
// machine that no longer had one. The installer script has reset them all
// along; this path deleted the files, reloaded and stopped there. It does not
// cover a unit that fails after the uninstall has exited, which is a separate
// problem in how long the stop waits.
func TestRemoveServiceUnits_resetsEveryUnitItRemoved(t *testing.T) {
	swapMgr(t, &unitListMgr{
		fakeServiceMgr: &fakeServiceMgr{},
		containers:     []string{"lerd-mysql", "lerd-nginx"},
		services:       []string{"lerd-mysql", "lerd-ui"},
		timers:         []string{"lerd-backup.timer"},
	})

	var reset []string
	orig := resetFailedUnit
	resetFailedUnit = func(name string) { reset = append(reset, name) }
	t.Cleanup(func() { resetFailedUnit = orig })

	removeServiceUnits()

	want := []string{"lerd-mysql", "lerd-nginx", "lerd-ui", "lerd-backup"}
	if !equalStrings(reset, want) {
		t.Errorf("reset %v, want %v", reset, want)
	}
}

// A keep-data uninstall removes every unit, so a built-in service no site lists
// comes back only because uninstall recorded it. One removed since stays removed.
func TestDefaultPresetsToRefresh_restoresKeptServices(t *testing.T) {
	installed := func(unit string) bool { return unit == "lerd-mysql" }
	removed := func(n string) bool { return n == "mailpit" }
	kept := map[string]bool{"redis": true, "mailpit": true}

	got := defaultPresetsToRefresh([]string{"mailpit", "mysql", "postgres", "redis"}, installed, kept, removed)
	want := []string{"mysql", "redis"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("defaultPresetsToRefresh = %v, want %v", got, want)
	}
}

func TestInstalledDefaultServices_listsOnlyThoseWithAUnit(t *testing.T) {
	installed := func(unit string) bool { return unit == "lerd-redis" || unit == "lerd-mysql" }
	got := installedDefaultServices([]string{"mailpit", "mysql", "redis"}, installed)
	if strings.Join(got, ",") != "mysql,redis" {
		t.Fatalf("installedDefaultServices = %v", got)
	}
}

// A Windows install names its tray lerd-tray.exe, beside lerd.exe.
func TestRemoveInstalledBinaries_keepsTheBinarysExtensionForTheTray(t *testing.T) {
	dir := t.TempDir()
	self := filepath.Join(dir, "lerd.exe")
	tray := filepath.Join(dir, "lerd-tray.exe")
	mkbin(t, self)
	mkbin(t, tray)

	removeInstalledBinaries(self)

	if _, err := os.Stat(tray); !os.IsNotExist(err) {
		t.Errorf("lerd-tray.exe still present")
	}
}
