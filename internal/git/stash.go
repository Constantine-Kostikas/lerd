package git

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// WithStash sets uncommitted work aside, untracked files included, runs move and
// puts it back on top, touching only its own entry, found by commit, so the user's
// stashes stay. A failed move gets it back; a conflicting one leaves it stashed.
func WithStash(dir, label string, move func() (string, error)) (string, error) {
	unlock := lockStash(dir)
	before := stashTop(dir)
	_, err := runQuiet(dir, "stash", "push", "--include-untracked", "-m", label)
	ours := stashTop(dir)
	unlock()
	if err != nil {
		return "", fmt.Errorf("setting local changes aside: %w", err)
	}
	// Nothing to set aside makes no entry, and the top one is still the user's.
	if ours == "" || ours == before {
		return move()
	}
	msg, moveErr := move()
	if err := popStash(dir, ours, label); err != nil {
		if moveErr != nil {
			return "", fmt.Errorf("%v; your local changes are kept in the stash as %q", moveErr, label)
		}
		return "", err
	}
	return msg, moveErr
}

// stashLocks serialise stash changes per repository: its worktrees share one
// stash list, so a push and the read of its entry must not interleave.
var stashLocks sync.Map

func lockStash(dir string) (unlock func()) {
	key := dir
	if out, err := Output(dir, "rev-parse", "--path-format=absolute", "--git-common-dir"); err == nil {
		key = filepath.Clean(strings.TrimSpace(out))
	}
	mu, _ := stashLocks.LoadOrStore(key, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	return mu.(*sync.Mutex).Unlock
}

func stashTop(dir string) string {
	out, _ := Output(dir, "rev-parse", "--verify", "-q", "refs/stash")
	return strings.TrimSpace(out)
}

func popStash(dir, sha, label string) error {
	defer lockStash(dir)()
	out, _ := Output(dir, "stash", "list", "--format=%H")
	n := -1
	for i, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == sha {
			n = i
		}
	}
	if n < 0 {
		return fmt.Errorf("the stash entry %q is gone; your local changes are not back", label)
	}
	ref := fmt.Sprintf("stash@{%d}", n)
	// --index brings back what was staged; git refuses it when the staged part
	// no longer applies, and then the changes come back unstaged instead.
	_, err := runQuiet(dir, "stash", "pop", "--index", ref)
	if err != nil && stashTop(dir) != "" && listed(dir, sha) && clean(dir) {
		_, err = runQuiet(dir, "stash", "pop", ref)
	}
	if err == nil {
		return nil
	}
	if files, _ := Output(dir, "diff", "--name-only", "--diff-filter=U"); strings.TrimSpace(files) != "" {
		where := strings.Join(strings.Fields(files), ", ")
		return fmt.Errorf("your local changes conflict with %s. They are kept in the stash as %q: resolve the conflicts, then run git stash drop", where, label)
	}
	return fmt.Errorf("your local changes could not all be put back, and are kept in the stash as %q: %v", label, err)
}

func listed(dir, sha string) bool {
	out, _ := Output(dir, "stash", "list", "--format=%H")
	return strings.Contains(out, sha)
}

// clean reports a tree a failed pop left untouched, so a second try starts fresh.
func clean(dir string) bool {
	out, _ := Output(dir, "status", "--porcelain")
	return strings.TrimSpace(out) == ""
}
