//go:build !windows

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveShellEntry_bashrc(t *testing.T) {
	tmp := t.TempDir()

	// Simulate a home directory with a .bashrc containing the Lerd PATH block.
	bashrc := filepath.Join(tmp, ".bashrc")
	os.WriteFile(bashrc, []byte(
		"# existing config\n"+
			"# Added by Lerd installer\n"+
			`export PATH="/home/user/.local/share/lerd/bin:$PATH"`+"\n",
	), 0644)

	// Point HOME at the temp dir so removeShellEntry reads our fake rc files.
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	removeShellEntry()

	got, _ := os.ReadFile(bashrc)
	if strings.Contains(string(got), "Added by Lerd installer") {
		t.Error("Lerd marker should have been removed from .bashrc")
	}
	if strings.Contains(string(got), "lerd/bin") {
		t.Error("Lerd PATH export should have been removed from .bashrc")
	}
	if !strings.Contains(string(got), "# existing config") {
		t.Error("pre-existing config should be preserved")
	}
}

func TestRemoveShellEntry_fishConfig(t *testing.T) {
	tmp := t.TempDir()
	fishDir := filepath.Join(tmp, ".config", "fish", "conf.d")
	os.MkdirAll(fishDir, 0755)

	fishConf := filepath.Join(fishDir, "lerd.fish")
	os.WriteFile(fishConf, []byte(
		"# Added by Lerd installer\n"+
			"fish_add_path /home/user/.local/share/lerd/bin\n",
	), 0644)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	removeShellEntry()

	got, _ := os.ReadFile(fishConf)
	if strings.Contains(string(got), "Added by Lerd installer") {
		t.Error("Lerd marker should have been removed from fish config")
	}
}

// TestRemoveShellEntry_removesAllLerdInstallerMarkers pins the fix for a
// user-reported regression: `lerd install` (the Go binary) writes two
// extra marker blocks beyond `install.sh`'s "# Added by Lerd installer"
// — "# Lerd" (PATH export) and "# Lerd completions" (fpath + autoload).
// Uninstall used to match only the first marker, leaving the other two
// behind on every install path that went through `lerd install` (which
// is the common case).
func TestRemoveShellEntry_removesAllLerdInstallerMarkers(t *testing.T) {
	tmp := t.TempDir()
	zshrc := filepath.Join(tmp, ".zshrc")
	os.WriteFile(zshrc, []byte(
		"alias gco='git checkout'\n"+
			"\n"+
			"# Added by Lerd installer\n"+
			`export PATH="/h/u/.local/bin:$PATH"`+"\n"+
			"\n"+
			"# Lerd\n"+
			`export PATH="/h/u/.local/share/lerd/bin:$PATH"`+"\n"+
			"\n"+
			"# Lerd completions\n"+
			"fpath=(/h/u/.local/share/zsh/site-functions $fpath)\n"+
			"autoload -Uz compinit && compinit\n"+
			"\n"+
			"alias home='cd /h/u'\n",
	), 0644)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	removeShellEntry()

	got, _ := os.ReadFile(zshrc)
	for _, leftover := range []string{
		"Added by Lerd installer",
		"# Lerd\n",
		"# Lerd completions",
		"share/lerd/bin",
		"share/zsh/site-functions",
		"compinit",
	} {
		if strings.Contains(string(got), leftover) {
			t.Errorf("expected %q to be removed; remaining content:\n%s", leftover, got)
		}
	}
	for _, preserved := range []string{"git checkout", "cd /h/u"} {
		if !strings.Contains(string(got), preserved) {
			t.Errorf("expected %q to be preserved", preserved)
		}
	}
}

// TestRemoveShellEntry_fishFileDeletedWhenEmpty pins that an entirely
// lerd-owned fish config file is removed (not left as an empty
// conf.d entry that fish keeps sourcing on every shell start).
func TestRemoveShellEntry_fishFileDeletedWhenEmpty(t *testing.T) {
	tmp := t.TempDir()
	fishDir := filepath.Join(tmp, ".config", "fish", "conf.d")
	os.MkdirAll(fishDir, 0755)
	fishConf := filepath.Join(fishDir, "lerd.fish")
	os.WriteFile(fishConf, []byte(
		"\n# Added by Lerd installer\nfish_add_path /h/u/.local/bin\n",
	), 0644)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	removeShellEntry()

	if _, err := os.Stat(fishConf); !os.IsNotExist(err) {
		body, _ := os.ReadFile(fishConf)
		t.Errorf("expected lerd.fish to be removed; still exists with content:\n%s", body)
	}
}

// TestRemoveShellEntry_fishFileKeptWhenNonEmpty pins the inverse: if
// the user added their own content to lerd.fish beyond our markers, we
// strip our blocks but leave the file alone.
func TestRemoveShellEntry_fishFileKeptWhenNonEmpty(t *testing.T) {
	tmp := t.TempDir()
	fishDir := filepath.Join(tmp, ".config", "fish", "conf.d")
	os.MkdirAll(fishDir, 0755)
	fishConf := filepath.Join(fishDir, "lerd.fish")
	os.WriteFile(fishConf, []byte(
		"# Added by Lerd installer\nfish_add_path /h/u/.local/bin\n\n"+
			"# user-added: alias for personal use\nalias myls 'ls -la'\n",
	), 0644)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	removeShellEntry()

	body, err := os.ReadFile(fishConf)
	if err != nil {
		t.Fatalf("file should have been kept: %v", err)
	}
	if strings.Contains(string(body), "Added by Lerd installer") {
		t.Errorf("expected lerd marker removed, got:\n%s", body)
	}
	if !strings.Contains(string(body), "myls") {
		t.Errorf("user content lost; got:\n%s", body)
	}
}

func TestRemoveDataDir_reportsTheDirectoryWhenTheFallbackFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root removes the tree without the fallback")
	}
	dir := undeletableDir(t)
	fakePodman(t, dir, "exit 1\n")

	if kept := removeDataDir(dir); kept != dir {
		t.Errorf("removeDataDir = %q, want %q so the uninstall can say so", kept, dir)
	}
}

// The package-managed guard exists so lerd never deletes a file out of a Cellar
// or an rpm's file list. It said nothing about ~/.local/bin, which no package
// manager owns, so a machine carrying both installs kept a script-installed
// lerd on PATH pointing at a data directory the same run had just deleted.
func TestRemoveScriptInstalledBinaries_clearsTheInstallDirPair(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	lerd := filepath.Join(binDir, "lerd")
	tray := filepath.Join(binDir, "lerd-tray")
	mkbin(t, lerd)
	mkbin(t, tray)

	removeScriptInstalledBinaries("/usr/bin/lerd")

	for _, p := range []string{lerd, tray} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s survived, so a broken lerd stays on PATH", p)
		}
	}
}
