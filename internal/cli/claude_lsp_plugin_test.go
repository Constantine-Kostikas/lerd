package cli

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func recordClaudePlugin(t *testing.T, fail string) *[]string {
	t.Helper()
	var calls []string
	prev := claudePlugin
	claudePlugin = func(args ...string) ([]byte, error) {
		call := strings.Join(args, " ")
		calls = append(calls, call)
		if fail != "" && strings.HasPrefix(call, fail) {
			return []byte("boom"), errors.New("exit status 1")
		}
		return nil, nil
	}
	t.Cleanup(func() { claudePlugin = prev })
	return &calls
}

// Enabling global MCP also hands Claude Code the PHP language server, which it
// only takes through its own plugin CLI.
func TestWriteGlobalMCPConfigsInstallsTheLSPPluginInClaude(t *testing.T) {
	calls := recordClaudePlugin(t, "")
	if err := writeGlobalMCPConfigs(t.TempDir(), false); err != nil {
		t.Fatal(err)
	}
	want := []string{"marketplace add lerd-env/claude-plugins", "install --scope user lerd-php@lerd"}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("claude plugin calls = %q, want %q", *calls, want)
	}
}

// A second enable finds the marketplace already added, which the CLI reports
// as a failure; the install must still run and the enable still succeed.
func TestWriteGlobalMCPConfigsInstallsTheLSPPluginWhenTheMarketplaceExists(t *testing.T) {
	calls := recordClaudePlugin(t, "marketplace add")
	if err := writeGlobalMCPConfigs(t.TempDir(), false); err != nil {
		t.Fatal(err)
	}
	if got := (*calls)[len(*calls)-1]; got != "install --scope user lerd-php@lerd" {
		t.Fatalf("last claude plugin call = %q, want the install", got)
	}
}

func TestRemoveGlobalAISkillsTakesTheLSPPluginBackOut(t *testing.T) {
	calls := recordClaudePlugin(t, "")
	if err := RemoveGlobalAISkills(t.TempDir(), false); err != nil {
		t.Fatal(err)
	}
	want := []string{"uninstall --scope user lerd-php@lerd", "marketplace remove lerd"}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("claude plugin calls = %q, want %q", *calls, want)
	}
}
