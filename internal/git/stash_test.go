package git

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
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

// What was staged before is staged again after.
func TestWithStash_keepsWhatWasStaged(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "a"), "staged")
	gitRun(t, dir, "add", "a")
	write(t, filepath.Join(dir, "b"), "untracked")
	if _, err := WithStash(dir, "lerd: test", func() (string, error) { return "", nil }); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(mustOut(t, dir, "diff", "--cached", "--name-only")); got != "a" {
		t.Errorf("staged after %q", got)
	}
}

// An untracked file the move brought in cannot come back; git's reason is
// passed on, and dropping the stash is not suggested while it holds that copy.
func TestWithStash_untrackedCollisionKeepsGitsReason(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "new"), "mine")
	_, err := WithStash(dir, "lerd: test", func() (string, error) {
		write(t, filepath.Join(dir, "new"), "theirs")
		gitRun(t, dir, "add", "new")
		gitRun(t, dir, "commit", "-q", "-m", "theirs")
		return "", nil
	})
	if err == nil || !strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "drop") {
		t.Fatalf("want git's reason without drop advice, got %v", err)
	}
	if !strings.Contains(mustOut(t, dir, "stash", "list"), "lerd: test") {
		t.Error("the stash holding the file is gone")
	}
}

// Two worktrees share one stash list; each gets its own work back.
func TestWithStash_worktreesDoNotSwapWork(t *testing.T) {
	dir := commitFixture(t)
	other := filepath.Join(t.TempDir(), "wt")
	gitRun(t, dir, "worktree", "add", "-q", "-b", "side", other)
	write(t, filepath.Join(dir, "a"), "main work")
	write(t, filepath.Join(other, "a"), "side work")

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, d := range []string{dir, other} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = WithStash(d, "lerd: "+d, func() (string, error) { time.Sleep(50 * time.Millisecond); return "", nil })
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if readFile(t, filepath.Join(dir, "a")) != "main work" || readFile(t, filepath.Join(other, "a")) != "side work" {
		t.Error("work landed in the wrong checkout")
	}
}
