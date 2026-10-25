package git

import (
	"fmt"
	"strings"
)

// WithStash sets uncommitted work aside, untracked files included, runs move and
// puts it back on top, touching only its own entry, found by commit, so the user's
// stashes stay. A failed move gets it back; a conflicting one leaves it stashed.
func WithStash(dir, label string, move func() (string, error)) (string, error) {
	before := stashTop(dir)
	if _, err := runQuiet(dir, "stash", "push", "--include-untracked", "-m", label); err != nil {
		return "", fmt.Errorf("setting local changes aside: %w", err)
	}
	ours := stashTop(dir)
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

func stashTop(dir string) string {
	out, _ := Output(dir, "rev-parse", "--verify", "-q", "refs/stash")
	return strings.TrimSpace(out)
}

func popStash(dir, sha, label string) error {
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
	if _, err := runQuiet(dir, "stash", "pop", fmt.Sprintf("stash@{%d}", n)); err == nil {
		return nil
	}
	files, _ := Output(dir, "diff", "--name-only", "--diff-filter=U")
	where := strings.Join(strings.Fields(files), ", ")
	if where == "" {
		where = "files the move also brought"
	}
	return fmt.Errorf("your local changes conflict with %s. They are kept in the stash as %q: resolve the conflicts, then run git stash drop", where, label)
}
