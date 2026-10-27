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

// Work that clashes with the moved code stays in the stash, and the tree is
// left as the move made it rather than full of conflict markers.
func TestWithStash_conflictKeepsTheWorkStashed(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "a"), "local")
	msg, err := WithStash(dir, "lerd: test", func() (string, error) {
		write(t, filepath.Join(dir, "a"), "moved")
		gitRun(t, dir, "commit", "-q", "-am", "moved")
		return "moved", nil
	})
	var kept *StashKept
	if !errors.As(err, &kept) || strings.Join(kept.Files, ",") != "a" || kept.Label != "lerd: test" {
		t.Fatalf("want the clashing file named with the stash entry, got %v", err)
	}
	if msg != "moved" {
		t.Errorf("the move's message %q was dropped", msg)
	}
	if !strings.Contains(mustOut(t, dir, "stash", "list"), "lerd: test") {
		t.Error("local work not kept in the stash")
	}
	if readFile(t, filepath.Join(dir, "a")) != "moved" || strings.TrimSpace(mustOut(t, dir, "status", "--porcelain")) != "" {
		t.Error("the tree was not left as the move made it")
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
// passed on, and the stash keeps that copy.
func TestWithStash_untrackedCollisionKeepsGitsReason(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "new"), "mine")
	write(t, filepath.Join(dir, "a"), "local")
	_, err := WithStash(dir, "lerd: test", func() (string, error) {
		write(t, filepath.Join(dir, "new"), "theirs")
		gitRun(t, dir, "add", "new")
		gitRun(t, dir, "commit", "-q", "-m", "theirs")
		return "", nil
	})
	var kept *StashKept
	if !errors.As(err, &kept) || !strings.Contains(kept.Reason, "already exists") {
		t.Fatalf("want git's reason, got %v", err)
	}
	if !strings.Contains(mustOut(t, dir, "stash", "list"), "lerd: test") {
		t.Error("the stash holding the file is gone")
	}
	if readFile(t, filepath.Join(dir, "new")) != "theirs" || strings.TrimSpace(mustOut(t, dir, "status", "--porcelain")) != "" {
		t.Error("the tree was not left as the move made it")
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

// A pop that fails on a tree already holding other work takes back only its
// own files; the rest stays as it was.
func TestPopStash_failureKeepsOtherWork(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "b"), "base")
	gitRun(t, dir, "add", "b")
	gitRun(t, dir, "commit", "-q", "-m", "b")
	write(t, filepath.Join(dir, "a"), "stashed")
	gitRun(t, dir, "stash", "push", "-q", "-m", "lerd: test")
	sha := strings.TrimSpace(mustOut(t, dir, "rev-parse", "refs/stash"))
	write(t, filepath.Join(dir, "a"), "moved")
	gitRun(t, dir, "commit", "-q", "-am", "moved")
	write(t, filepath.Join(dir, "b"), "other work")

	var kept *StashKept
	if err := PopStash(dir, sha, "lerd: test"); !errors.As(err, &kept) {
		t.Fatalf("want the stash kept, got %v", err)
	}
	if readFile(t, filepath.Join(dir, "b")) != "other work" {
		t.Error("other work was thrown away")
	}
	if readFile(t, filepath.Join(dir, "a")) != "moved" {
		t.Error("the stash's own file was not taken back")
	}
}

// Work made on top since the entry was kept is what makes git refuse the pop,
// and it must survive the clean up, untracked files too.
func TestPopStash_refusalKeepsNewerWork(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "a"), "stashed")
	write(t, filepath.Join(dir, "new"), "stashed")
	gitRun(t, dir, "stash", "push", "-q", "--include-untracked", "-m", "lerd: test")
	sha := strings.TrimSpace(mustOut(t, dir, "rev-parse", "refs/stash"))
	write(t, filepath.Join(dir, "a"), "newer")
	write(t, filepath.Join(dir, "new"), "newer")

	var kept *StashKept
	if err := PopStash(dir, sha, "lerd: test"); !errors.As(err, &kept) {
		t.Fatalf("want the stash kept, got %v", err)
	}
	if readFile(t, filepath.Join(dir, "a")) != "newer" || readFile(t, filepath.Join(dir, "new")) != "newer" {
		t.Error("newer work was thrown away")
	}
}

// A site linked below the repository root still gets a clean tree back: git
// names the stash's files from the root, not from the site's folder.
func TestPopStash_conflictInSubfolderSite(t *testing.T) {
	root := commitFixture(t)
	site := filepath.Join(root, "apps", "shop")
	if err := os.MkdirAll(site, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(site, "a"), "base")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-q", "-m", "shop")
	write(t, filepath.Join(site, "a"), "stashed")
	write(t, filepath.Join(site, "new"), "stashed")
	gitRun(t, site, "stash", "push", "-q", "--include-untracked", "-m", "lerd: test")
	sha := strings.TrimSpace(mustOut(t, site, "rev-parse", "refs/stash"))
	write(t, filepath.Join(site, "a"), "moved")
	gitRun(t, site, "commit", "-q", "-am", "moved")

	var kept *StashKept
	if err := PopStash(site, sha, "lerd: test"); !errors.As(err, &kept) || len(kept.Files) == 0 {
		t.Fatalf("want the clash kept in the stash, got %v", err)
	}
	if st := strings.TrimSpace(mustOut(t, root, "status", "--porcelain")); st != "" {
		t.Errorf("tree left dirty: %q", st)
	}
}
