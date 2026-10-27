package git

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// StashKept is work that would not go back on top of the moved code. The tree is
// left as the move made it and the work stays in the stash under Label. Err is a
// later step's failure, so one error carries both.
type StashKept struct {
	Label string
	// SHA finds the entry again: labels repeat across pulls.
	SHA string
	// Files clashed with the moved code; Reason is git's refusal when none did.
	Files  []string
	Reason string
	Err    error
}

func (e *StashKept) Error() string {
	why := e.Reason
	if len(e.Files) > 0 {
		why = "they conflict with " + strings.Join(e.Files, ", ")
	}
	msg := fmt.Sprintf("your local changes could not be put back (%s) and are kept in the stash as %q", why, e.Label)
	if e.Err != nil {
		return e.Err.Error() + "; " + msg
	}
	return msg
}

func (e *StashKept) Unwrap() error { return e.Err }

// WithStash sets uncommitted work aside, untracked files included, runs move and
// puts it back on top, touching only its own entry, found by commit, so the user's
// stashes stay. A failed move gets it back; a clashing one leaves it stashed as a
// *StashKept, with the move's message.
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
		return msg, err
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

// PopStash puts back the entry WithStash kept, found by its commit. Work that
// still clashes stays stashed as a *StashKept, with the tree left as it was.
func PopStash(dir, sha, label string) error { return popStash(dir, sha, label) }

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
	kept := &StashKept{Label: label, SHA: sha, Reason: err.Error()}
	if files, _ := Output(dir, "diff", "--name-only", "--diff-filter=U"); strings.TrimSpace(files) != "" {
		kept.Files = strings.Fields(files)
	}
	if err := undoPop(dir, sha); err != nil {
		return fmt.Errorf("%v, and undoing the partial put back failed: %w", kept, err)
	}
	return kept
}

// undoPop takes a failed pop's partial changes back out of the files the stash
// holds, which git only touches when they were clean; other work stays.
func undoPop(dir, sha string) error {
	tracked, err := Output(dir, "diff", "-z", "--name-only", sha+"^1", sha)
	if err != nil {
		return err
	}
	for _, p := range nulSplit(tracked) {
		args := []string{"checkout", "-q", "HEAD", "--", p}
		if _, err := Output(dir, "cat-file", "-e", "HEAD:"+p); err != nil {
			args = []string{"rm", "-q", "-f", "--ignore-unmatch", "--", p}
		}
		if _, err := runQuiet(dir, args...); err != nil {
			return err
		}
	}
	// The untracked files the pop brought back; clean leaves tracked ones alone.
	untracked, _ := Output(dir, "ls-tree", "-r", "-z", "--name-only", sha+"^3")
	if paths := nulSplit(untracked); len(paths) > 0 {
		_, err = runQuiet(dir, append([]string{"clean", "-q", "-f", "--"}, paths...)...)
	}
	return err
}

func nulSplit(out string) []string {
	if out = strings.TrimRight(out, "\x00"); out == "" {
		return nil
	}
	return strings.Split(out, "\x00")
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
