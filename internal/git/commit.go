package git

import (
	"errors"
	"strings"
)

// Stage adds one file's change to the index, a deletion included.
func Stage(dir, file string) error {
	_, err := runQuiet(dir, "add", "-A", "--", file)
	return err
}

// Unstage takes one file's change back out of the index, leaving the file as it is.
func Unstage(dir, file string) error {
	_, err := runQuiet(dir, "restore", "--staged", "--", file)
	return err
}

// CommitStaged records what is staged, or rewrites the last commit with it when
// amend is set. Hooks run as they would at a terminal, without one to prompt.
func CommitStaged(dir, message string, amend bool) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", errors.New("a commit needs a message")
	}
	args := []string{"commit", "-q", "-m", message}
	if amend {
		args = append(args, "--amend")
	}
	return runQuiet(dir, args...)
}

// LastMessage is the full message of the commit HEAD points at, for amending it.
func LastMessage(dir string) (string, error) {
	out, err := Output(dir, "log", "-1", "--format=%B")
	return strings.TrimSpace(out), err
}
