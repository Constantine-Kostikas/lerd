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

func index(t *testing.T, dir string) string {
	t.Helper()
	return mustOut(t, dir, "show", ":a")
}

func TestStageHunk_onlyThatBlock(t *testing.T) {
	dir := hunkFixture(t)
	if err := StageHunk(dir, "a", Hunk{OrigStart: 5, OrigEnd: 5, ModStart: 5, ModEnd: 5, Removed: []string{"5"}, Added: []string{"five"}}, false); err != nil {
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
	if err := StageHunk(dir, "a", Hunk{OrigStart: 1, OrigEnd: 1, ModStart: 1, ModEnd: 1, Removed: []string{"1"}, Added: []string{"one"}}, true); err != nil {
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
	if err := StageHunk(dir, "a", Hunk{OrigStart: 4, OrigEnd: 4, ModStart: 4, ModEnd: 0, Removed: []string{"4"}}, false); err != nil {
		t.Fatalf("deletion: %v", err)
	}
	if got := index(t, dir); got != "1\n2\n3\n5\n" {
		t.Fatalf("after deletion %q", got)
	}
	// "new" added after line 2.
	if err := StageHunk(dir, "a", Hunk{OrigStart: 2, OrigEnd: 0, ModStart: 3, ModEnd: 3, Added: []string{"new"}}, false); err != nil {
		t.Fatalf("insertion: %v", err)
	}
	if got := index(t, dir); got != "1\n2\nnew\n3\n5\n" {
		t.Errorf("after insertion %q", got)
	}
}

// A block the file no longer has is refused rather than staged in the wrong place.
func TestStageHunk_refusesAStaleBlock(t *testing.T) {
	dir := hunkFixture(t)
	err := StageHunk(dir, "a", Hunk{OrigStart: 5, OrigEnd: 5, ModStart: 5, ModEnd: 5, Removed: []string{"5"}, Added: []string{"FIVE"}}, false)
	if err == nil || !strings.Contains(err.Error(), "changed since") {
		t.Fatalf("want a refusal, got %v", err)
	}
	if got := index(t, dir); got != "1\n2\n3\n4\n5\n" {
		t.Errorf("index changed: %q", got)
	}
}

func TestStageHunk_onlyModifiedFiles(t *testing.T) {
	dir := hunkFixture(t)
	if err := os.Remove(filepath.Join(dir, "a")); err != nil {
		t.Fatal(err)
	}
	if err := StageHunk(dir, "a", Hunk{OrigStart: 1, OrigEnd: 5, ModStart: 0, ModEnd: 0, Removed: []string{"1", "2", "3", "4", "5"}}, false); err == nil {
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
	if err := StageHunk(dir, "a", Hunk{OrigStart: 1, OrigEnd: 1, ModStart: 1, ModEnd: 1, Removed: []string{"1"}, Added: []string{"one"}}, false); err != nil {
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
	if err := StageHunk(dir, "a", Hunk{OrigStart: 1, OrigEnd: 1, ModStart: 1, ModEnd: 1, Removed: []string{"1"}, Added: []string{"one"}}, false); err != nil {
		t.Fatal(err)
	}
	if got := index(t, dir); got != "one\r\n2\r\n" {
		t.Errorf("index %q", got)
	}
}
