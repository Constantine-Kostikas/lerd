package git

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// Hunk is one change block as the diff editor numbers it, an end of 0 meaning no
// lines on that side, which then sit after its start. Removed and Added are the
// lines it showed, so a file that changed since is refused, not staged wrongly.
type Hunk struct {
	OrigStart int      `json:"orig_start"`
	OrigEnd   int      `json:"orig_end"`
	ModStart  int      `json:"mod_start"`
	ModEnd    int      `json:"mod_end"`
	Removed   []string `json:"removed"`
	Added     []string `json:"added"`
}

// StageHunk moves one block of a modified file's changes into the index, or
// with unstage back out of it, leaving the file's other changes where they are.
func StageHunk(dir, path string, h Hunk, unstage bool) error {
	section := SectionUnstaged
	if unstage {
		section = SectionStaged
	}
	files, err := sectionFiles(dir, section)
	if err != nil {
		return err
	}
	// An added or deleted file has only one side; it is staged whole.
	if f, ok := find(files, path); !ok || f.Status != "M" {
		return fmt.Errorf("%s can only be staged as a whole file", path)
	}
	v, err := FileVersions(dir, section, path)
	if err != nil {
		return err
	}
	if v.Binary || v.TooLarge || v.Unreadable {
		return fmt.Errorf("%s can only be staged as a whole file", path)
	}
	orig, mod := strings.Split(v.Original, "\n"), strings.Split(v.Modified, "\n")
	removed, okR := span(orig, h.OrigStart, h.OrigEnd)
	added, okA := span(mod, h.ModStart, h.ModEnd)
	if !okR || !okA || !sameLines(removed, h.Removed) || !sameLines(added, h.Added) {
		return errors.New("the file changed since it was shown; reload and try again")
	}
	// Staging puts the block on top of the index; unstaging puts HEAD's lines back in its place.
	next := splice(orig, h.OrigStart, h.OrigEnd, added)
	if unstage {
		next = splice(mod, h.ModStart, h.ModEnd, removed)
	}
	return writeIndex(dir, path, strings.Join(next, "\n"))
}

// sameLines compares without the \r of a CRLF file, which the editor leaves off its lines.
func sameLines(file, shown []string) bool {
	return slices.EqualFunc(file, shown, func(a, b string) bool { return strings.TrimSuffix(a, "\r") == b })
}

// span returns the lines start..end, or none for an end of 0, and whether they exist.
func span(lines []string, start, end int) ([]string, bool) {
	if end == 0 {
		return []string{}, start >= 0 && start <= len(lines)
	}
	if start < 1 || end < start || end > len(lines) {
		return nil, false
	}
	return lines[start-1 : end], true
}

func splice(lines []string, start, end int, with []string) []string {
	from, to := start-1, end
	if end == 0 {
		from, to = start, start
	}
	return slices.Concat(lines[:from], with, lines[to:])
}

// writeIndex stores content as path's staged version, keeping its mode.
func writeIndex(dir, path, content string) error {
	entry, err := Output(dir, "ls-files", "-s", "--", path)
	mode, _, _ := strings.Cut(entry, " ")
	if err != nil || mode == "" {
		return fmt.Errorf("%s is not in the index", path)
	}
	cmd := exec.Command("git", "hash-object", "-w", "--stdin")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(content)
	sha, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("storing the staged version: %w", err)
	}
	_, err = runQuiet(dir, "update-index", "--cacheinfo", mode+","+strings.TrimSpace(string(sha))+","+path)
	return err
}
