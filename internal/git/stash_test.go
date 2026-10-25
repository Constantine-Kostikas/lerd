package git

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The work comes back after the move, untracked files included, and the
// stash the user already had is left as it was.
func TestWithStash_putsTheWorkBack(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "a"), "theirs-to-keep")
	gitRun(t, dir, "stash", "push", "-q", "-m", "kept by the user")
	write(t, filepath.Join(dir, "a"), "local")
	write(t, filepath.Join(dir, "new"), "untracked")

	var during string
	_, err := WithStash(dir, "lerd: test", func() (string, error) {
		during = strings.TrimSpace(mustOut(t, dir, "status", "--porcelain"))
		return "moved", nil
	})
	if err != nil {
		t.Fatalf("WithStash: %v", err)
	}
	if during != "" {
		t.Errorf("tree not clean during the move: %q", during)
	}
	if readFile(t, filepath.Join(dir, "a")) != "local" || readFile(t, filepath.Join(dir, "new")) != "untracked" {
		t.Error("local work not put back")
	}
	if list := strings.TrimSpace(mustOut(t, dir, "stash", "list")); !strings.HasSuffix(list, "kept by the user") || strings.Count(list, "\n") != 0 {
		t.Errorf("stash list %q", list)
	}
}

// With nothing to set aside, git makes no entry, and the user's own must not be popped instead.
func TestWithStash_cleanTreeLeavesTheUsersStash(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "a"), "theirs-to-keep")
	gitRun(t, dir, "stash", "push", "-q", "-m", "kept by the user")

	if _, err := WithStash(dir, "lerd: test", func() (string, error) { return "", nil }); err != nil {
		t.Fatal(err)
	}
	if readFile(t, filepath.Join(dir, "a")) != "base" {
		t.Error("the user's stash was applied")
	}
	if !strings.Contains(mustOut(t, dir, "stash", "list"), "kept by the user") {
		t.Error("the user's stash is gone")
	}
}

// A move that fails leaves the work where it was.
func TestWithStash_failedMovePutsTheWorkBack(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "a"), "local")
	_, err := WithStash(dir, "lerd: test", func() (string, error) { return "", errors.New("no such branch") })
	if err == nil || !strings.Contains(err.Error(), "no such branch") {
		t.Fatalf("want the move's error, got %v", err)
	}
	if readFile(t, filepath.Join(dir, "a")) != "local" {
		t.Error("local work not put back")
	}
	if list := strings.TrimSpace(mustOut(t, dir, "stash", "list")); list != "" {
		t.Errorf("stash left behind: %q", list)
	}
}

func TestWithStash_conflictKeepsTheWorkStashed(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "a"), "local")
	_, err := WithStash(dir, "lerd: test", func() (string, error) {
		write(t, filepath.Join(dir, "a"), "moved")
		gitRun(t, dir, "commit", "-q", "-am", "moved")
		return "", nil
	})
	if err == nil || !strings.Contains(err.Error(), "a") || !strings.Contains(err.Error(), "lerd: test") {
		t.Fatalf("want the conflict named with the stash entry, got %v", err)
	}
	if !strings.Contains(mustOut(t, dir, "stash", "list"), "lerd: test") {
		t.Error("local work not kept in the stash")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "a")); statErr != nil {
		t.Error(statErr)
	}
}
