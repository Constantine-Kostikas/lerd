package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/geodro/lerd/internal/config"
	"github.com/spf13/cobra"
)

func findCmd(cmds []*cobra.Command, use string) *cobra.Command {
	for _, c := range cmds {
		if c.Use == use {
			return c
		}
	}
	return nil
}

func TestNewStripeCmds_ConfigCommandAndFlags(t *testing.T) {
	cmds := NewStripeCmds()

	cfg := findCmd(cmds, "stripe:config")
	if cfg == nil {
		t.Fatal("NewStripeCmds() must include a stripe:config command")
	}
	for _, f := range []string{"path", "secret-env-key", "project-name"} {
		if cfg.Flags().Lookup(f) == nil {
			t.Errorf("stripe:config missing --%s flag", f)
		}
	}

	// The listener command must also expose the persistence flags so the path
	// can be set inline when starting.
	listen := findCmd(cmds, "stripe:listen")
	if listen == nil {
		t.Fatal("NewStripeCmds() must include a stripe:listen command")
	}
	for _, f := range []string{"path", "secret-env-key", "api-key", "project-name"} {
		if listen.Flags().Lookup(f) == nil {
			t.Errorf("stripe:listen missing --%s flag", f)
		}
	}

	login := findCmd(cmds, "stripe:login")
	if login == nil || login.Flags().Lookup("project-name") == nil {
		t.Fatal("NewStripeCmds() must include stripe:login with --project-name")
	}
}

// The stop command used to be registered at the root under the name
// stripe:listen, which left two root commands sharing that name and made
// `lerd stripe:listen` resolve to whichever cobra walked into first.
func TestNewStripeCmds_ListenStopIsASubcommand(t *testing.T) {
	root := &cobra.Command{Use: "lerd"}
	for _, c := range NewStripeCmds() {
		root.AddCommand(c)
	}

	listen := 0
	for _, c := range root.Commands() {
		if c.Name() == "stripe:listen" {
			listen++
		}
	}
	if listen != 1 {
		t.Fatalf("root has %d commands named stripe:listen, want 1", listen)
	}

	start, _, err := root.Find([]string{"stripe:listen"})
	if err != nil {
		t.Fatalf("finding stripe:listen: %v", err)
	}
	if start.Flags().Lookup("api-key") == nil {
		t.Error("lerd stripe:listen must resolve to the starter, which carries --api-key")
	}

	stop, _, err := root.Find([]string{"stripe:listen", "stop"})
	if err != nil {
		t.Fatalf("finding stripe:listen stop: %v", err)
	}
	if stop.Name() != "stop" {
		t.Errorf("lerd stripe:listen stop resolved to %q, want the stop subcommand", stop.Name())
	}
}

func TestStripeLoginProject_PrefersSavedThenSiteName(t *testing.T) {
	if got := stripeLoginProject(nil); got != "default" {
		t.Errorf("outside a site = %q, want default", got)
	}
	dir := t.TempDir()
	site := &config.Site{Name: "shop", Path: dir}
	if got := stripeLoginProject(site); got != "shop" {
		t.Errorf("site without a project = %q, want shop", got)
	}
	if err := os.WriteFile(filepath.Join(dir, ".lerd.local.yaml"), []byte("stripe:\n  project_name: acme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := stripeLoginProject(site); got != "acme" {
		t.Errorf("site with a saved project = %q, want acme", got)
	}
}

// A login inside a git worktree belongs to the parent site, which runs the
// listener; outside any site there is no site to save it to.
func TestStripeLoginSite_ResolvesAWorktreeToItsParent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	sitePath := filepath.Join(tmp, "shop")
	checkout := filepath.Join(tmp, "shop-feature")
	meta := filepath.Join(sitePath, ".git", "worktrees", "feature")
	for _, d := range []string{meta, checkout} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(meta, "gitdir"), []byte(filepath.Join(checkout, ".git")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.AddSite(config.Site{Name: "shop", Path: sitePath, Domains: []string{"shop.test"}}); err != nil {
		t.Fatal(err)
	}
	if got := stripeLoginSite(checkout); got == nil || got.Name != "shop" {
		t.Errorf("stripeLoginSite(worktree) = %v, want the shop site", got)
	}
	if got := stripeLoginSite(sitePath); got == nil || got.Name != "shop" {
		t.Errorf("stripeLoginSite(site) = %v, want the shop site", got)
	}
	if got := stripeLoginSite(t.TempDir()); got != nil {
		t.Errorf("stripeLoginSite(unrelated) = %v, want nil", got)
	}
}
