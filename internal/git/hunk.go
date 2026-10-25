package git

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// Hunk is one change block as the diff editor numbers it, an end of 0 meaning no
// lines on that side, which then sit after its start. Original and Modified are
// the texts it showed, so a file that changed since is refused, not staged wrongly.
type Hunk struct {
	OrigStart int    `json:"orig_start"`
	OrigEnd   int    `json:"orig_end"`
	ModStart  int    `json:"mod_start"`
	ModEnd    int    `json:"mod_end"`
	Original  string `json:"original"`
	Modified  string `json:"modified"`
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
	// An added or deleted file has only one side, and a filtered one (LFS and the
	// like) is stored as something other than its text: both are staged whole.
	if f, ok := find(files, path); !ok || f.Status != "M" || hasFilter(dir, path) {
		return fmt.Errorf("%s can only be staged as a whole file", path)
	}
	v, err := FileVersions(dir, section, path)
	if err != nil {
		return err
	}
	if v.Binary || v.TooLarge || v.Unreadable {
		return fmt.Errorf("%s can only be staged as a whole file", path)
	}
	if withoutCR(v.Original) != withoutCR(h.Original) || withoutCR(v.Modified) != withoutCR(h.Modified) {
		return errors.New("the file changed since it was shown; reload and try again")
	}
	orig, mod := strings.Split(v.Original, "\n"), strings.Split(v.Modified, "\n")
	removed, okR := span(orig, h.OrigStart, h.OrigEnd)
	added, okA := span(mod, h.ModStart, h.ModEnd)
	if !okR || !okA {
		return errors.New("the file changed since it was shown; reload and try again")
	}
	// Staging puts working-tree lines on top of the index, so they go through the
	// file's conversions as git add would; unstaging only moves index and HEAD lines.
	if unstage {
		return writeIndex(dir, path, strings.Join(splice(mod, h.ModStart, h.ModEnd, removed), "\n"), false)
	}
	return writeIndex(dir, path, strings.Join(splice(orig, h.OrigStart, h.OrigEnd, added), "\n"), true)
}

// withoutCR drops carriage returns, which the editor may normalise away.
func withoutCR(s string) string { return strings.ReplaceAll(s, "\r", "") }

func hasFilter(dir, path string) bool {
	out, _ := Output(dir, "check-attr", "filter", "--", path)
	return !strings.HasSuffix(strings.TrimSpace(out), ": unspecified")
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

// writeIndex stores content as path's staged version, keeping its mode; converted
// applies the path's line-ending conversion, as staging from the working tree does.
func writeIndex(dir, path, content string, converted bool) error {
	entry, err := Output(dir, "ls-files", "-s", "--", path)
	mode, _, _ := strings.Cut(entry, " ")
	if err != nil || mode == "" {
		return fmt.Errorf("%s is not in the index", path)
	}
	args := []string{"hash-object", "-w", "--stdin", "--no-filters"}
	if converted {
		args = []string{"hash-object", "-w", "--stdin", "--path=" + path}
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(content)
	sha, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("storing the staged version: %w", err)
	}
	_, err = runQuiet(dir, "update-index", "--cacheinfo", mode+","+strings.TrimSpace(string(sha))+","+path)
	return err
}
