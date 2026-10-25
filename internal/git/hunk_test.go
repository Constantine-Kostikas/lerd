package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hunkFixture commits a five-line file and changes lines 1 and 5 in the working tree.
func hunkFixture(t *testing.T) string {
	t.Helper()
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "a"), "1\n2\n3\n4\n5\n")
	gitRun(t, dir, "commit", "-q", "-am", "lines")
	write(t, filepath.Join(dir, "a"), "one\n2\n3\n4\nfive\n")
	return dir
}

// shown is the hunk as the editor sends it, with both texts it showed.
func shown(t *testing.T, dir string, h Hunk, section string) Hunk {
	t.Helper()
	v, err := FileVersions(dir, section, "a")
	if err != nil {
		t.Fatal(err)
	}
	h.Original, h.Modified = v.Original, v.Modified
	return h
}

func index(t *testing.T, dir string) string {
	t.Helper()
	return mustOut(t, dir, "show", ":a")
}

func TestStageHunk_onlyThatBlock(t *testing.T) {
	dir := hunkFixture(t)
	if err := StageHunk(dir, "a", shown(t, dir, Hunk{OrigStart: 5, OrigEnd: 5, ModStart: 5, ModEnd: 5}, SectionUnstaged), false); err != nil {
		t.Fatalf("StageHunk: %v", err)
	}
	if got := index(t, dir); got != "1\n2\n3\n4\nfive\n" {
		t.Errorf("index %q", got)
	}
	if got := readFile(t, filepath.Join(dir, "a")); got != "one\n2\n3\n4\nfive\n" {
		t.Errorf("working file touched: %q", got)
	}
}

func TestStageHunk_unstageTakesTheBlockBackOut(t *testing.T) {
	dir := hunkFixture(t)
	gitRun(t, dir, "add", "a")
	// Staged compares HEAD with the index, so the block is numbered the same way.
	if err := StageHunk(dir, "a", shown(t, dir, Hunk{OrigStart: 1, OrigEnd: 1, ModStart: 1, ModEnd: 1}, SectionStaged), true); err != nil {
		t.Fatalf("unstage: %v", err)
	}
	if got := index(t, dir); got != "1\n2\n3\n4\nfive\n" {
		t.Errorf("index %q", got)
	}
}

func TestStageHunk_insertionAndDeletion(t *testing.T) {
	dir := hunkFixture(t)
	write(t, filepath.Join(dir, "a"), "1\n2\nnew\n3\n5\n")
	// Line 4 removed: no lines on the working side, after its line 4.
	if err := StageHunk(dir, "a", shown(t, dir, Hunk{OrigStart: 4, OrigEnd: 4, ModStart: 4, ModEnd: 0}, SectionUnstaged), false); err != nil {
		t.Fatalf("deletion: %v", err)
	}
	if got := index(t, dir); got != "1\n2\n3\n5\n" {
		t.Fatalf("after deletion %q", got)
	}
	// "new" added after line 2.
	if err := StageHunk(dir, "a", shown(t, dir, Hunk{OrigStart: 2, OrigEnd: 0, ModStart: 3, ModEnd: 3}, SectionUnstaged), false); err != nil {
		t.Fatalf("insertion: %v", err)
	}
	if got := index(t, dir); got != "1\n2\nnew\n3\n5\n" {
		t.Errorf("after insertion %q", got)
	}
}

// A block from a view of the file that has since changed is refused, even one
// whose own lines still match, such as a deletion the user has undone.
func TestStageHunk_refusesAStaleBlock(t *testing.T) {
	dir := hunkFixture(t)
	write(t, filepath.Join(dir, "a"), "one\n2\n3\nfive\n")
	stale := shown(t, dir, Hunk{OrigStart: 4, OrigEnd: 4, ModStart: 3, ModEnd: 0}, SectionUnstaged)
	write(t, filepath.Join(dir, "a"), "one\n2\n3\n4\nFIVE\n")

	err := StageHunk(dir, "a", stale, false)
	if err == nil || !strings.Contains(err.Error(), "changed since") {
		t.Fatalf("want a refusal, got %v", err)
	}
	if got := index(t, dir); got != "1\n2\n3\n4\n5\n" {
		t.Errorf("index changed: %q", got)
	}
}

// With eol=crlf the index holds LF; a staged block goes through that conversion.
func TestStageHunk_convertsLineEndings(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, ".gitattributes"), "a text eol=crlf\n")
	write(t, filepath.Join(dir, "a"), "1\r\n2\r\n3\r\n")
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-q", "-m", "crlf")
	write(t, filepath.Join(dir, "a"), "one\r\n2\r\n3\r\n")
	v, err := FileVersions(dir, SectionUnstaged, "a")
	if err != nil {
		t.Fatal(err)
	}
	// Index and working tree differ in line endings, so every line is in the block.
	h := Hunk{OrigStart: 1, OrigEnd: 3, ModStart: 1, ModEnd: 3, Original: v.Original, Modified: v.Modified}
	if err := StageHunk(dir, "a", h, false); err != nil {
		t.Fatal(err)
	}
	if got := index(t, dir); got != "one\n2\n3\n" {
		t.Errorf("index %q", got)
	}
}

func TestStageHunk_filteredFilesStageWhole(t *testing.T) {
	dir := hunkFixture(t)
	write(t, filepath.Join(dir, ".gitattributes"), "a filter=lfs\n")
	err := StageHunk(dir, "a", shown(t, dir, Hunk{OrigStart: 1, OrigEnd: 1, ModStart: 1, ModEnd: 1}, SectionUnstaged), false)
	if err == nil || !strings.Contains(err.Error(), "whole file") {
		t.Fatalf("want a refusal, got %v", err)
	}
}

func TestStageHunk_onlyModifiedFiles(t *testing.T) {
	dir := hunkFixture(t)
	if err := os.Remove(filepath.Join(dir, "a")); err != nil {
		t.Fatal(err)
	}
	if err := StageHunk(dir, "a", Hunk{OrigStart: 1, OrigEnd: 5, ModStart: 0, ModEnd: 0}, false); err == nil {
		t.Fatal("want a refusal for a deleted file")
	}
}

// The file keeps its executable bit when a block of it is staged.
func TestStageHunk_keepsTheMode(t *testing.T) {
	dir := hunkFixture(t)
	// --chmod re-adds the file, so the edit is made again after the commit.
	gitRun(t, dir, "checkout", "--", "a")
	gitRun(t, dir, "update-index", "--chmod=+x", "a")
	gitRun(t, dir, "commit", "-q", "-m", "exec")
	write(t, filepath.Join(dir, "a"), "one\n2\n3\n4\nfive\n")
	if err := os.Chmod(filepath.Join(dir, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := StageHunk(dir, "a", shown(t, dir, Hunk{OrigStart: 1, OrigEnd: 1, ModStart: 1, ModEnd: 1}, SectionUnstaged), false); err != nil {
		t.Fatal(err)
	}
	if got := mustOut(t, dir, "ls-files", "-s", "a"); !strings.HasPrefix(got, "100755 ") {
		t.Errorf("mode lost: %q", got)
	}
}

// The editor shows a CRLF file's lines without their \r.
func TestStageHunk_crlf(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "a"), "1\r\n2\r\n")
	gitRun(t, dir, "commit", "-q", "-am", "crlf")
	write(t, filepath.Join(dir, "a"), "one\r\n2\r\n")
	if err := StageHunk(dir, "a", shown(t, dir, Hunk{OrigStart: 1, OrigEnd: 1, ModStart: 1, ModEnd: 1}, SectionUnstaged), false); err != nil {
		t.Fatal(err)
	}
	if got := index(t, dir); got != "one\r\n2\r\n" {
		t.Errorf("index %q", got)
	}
}

// The mode is read for that file alone, not for whatever its name matches as a pattern.
func TestStageHunk_modeOfThatFileOnly(t *testing.T) {
	dir := commitFixture(t)
	write(t, filepath.Join(dir, "aA.sh"), "x\n")
	write(t, filepath.Join(dir, "a[AB].sh"), "1\n2\n")
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "update-index", "--chmod=+x", "aA.sh")
	gitRun(t, dir, "commit", "-q", "-m", "two")
	write(t, filepath.Join(dir, "a[AB].sh"), "one\n2\n")
	v, err := FileVersions(dir, SectionUnstaged, "a[AB].sh")
	if err != nil {
		t.Fatal(err)
	}
	if err := StageHunk(dir, "a[AB].sh", Hunk{OrigStart: 1, OrigEnd: 1, ModStart: 1, ModEnd: 1, Original: v.Original, Modified: v.Modified}, false); err != nil {
		t.Fatal(err)
	}
	if got := mustOut(t, dir, "ls-files", "-s", "--", ":(literal)a[AB].sh"); !strings.HasPrefix(got, "100644 ") {
		t.Errorf("mode taken from another file: %q", got)
	}
}
