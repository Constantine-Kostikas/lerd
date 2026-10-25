package podman

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runDevtoolsExt runs body with the compiled lerd_devtools capturing queries,
// returning the lines and connections it sent. Skipped unless php can load the
// extension, which holds in lerd's FPM image and not on a CI runner.
func runDevtoolsExt(t *testing.T, body string, dropEach bool, exts ...string) ([]string, int) {
	t.Helper()
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("php not installed")
	}
	args := []string{"-n", "-d", "auto_prepend_file=", "-d", "extension=lerd_devtools"}
	for _, e := range exts {
		args = append(args, "-d", "extension="+e)
	}
	if out, _ := exec.Command(php, append(args, "-m")...).Output(); !strings.Contains(string(out), "lerd_devtools") {
		t.Skip("php cannot load lerd_devtools; run inside lerd's FPM image")
	}

	dir := t.TempDir()
	flag := filepath.Join(dir, "enabled.flag")
	script := filepath.Join(dir, "probe.php")
	if err := os.WriteFile(flag, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// lerd's own php wrapper runs in a container that can't see this dir.
	if out, _ := exec.Command(php, "-n", "-r", "echo file_exists("+phpQuote(flag)+") ? 'Y' : 'N';").Output(); !strings.Contains(string(out), "Y") {
		t.Skip("php cannot read host files (containerised/sandboxed wrapper); native php needed")
	}
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sink := listenCapture(t, dropEach)
	args = append(args,
		"-d", "lerd.devtools_host=unix://"+sink.sock,
		"-d", "lerd.devtools_kinds=query",
		"-d", "lerd.devtools_flag="+flag,
		"-d", "lerd.assets_dir="+dir,
		script)
	if out, err := exec.Command(php, args...).CombinedOutput(); err != nil {
		t.Fatalf("php run failed: %v\n%s", err, out)
	}
	return sink.result()
}

func queryLines(t *testing.T, lines []string) int {
	t.Helper()
	n := 0
	for _, l := range lines {
		var e struct{ Kind string }
		if err := json.Unmarshal([]byte(l), &e); err != nil {
			t.Fatalf("bad JSON line %q: %v", l, err)
		}
		if e.Kind == "query" {
			n++
		}
	}
	return n
}

const extQueries = `<?php
$db = new PDO('sqlite::memory:');
for ($i = 0; $i < 5; $i++) {
    $db->query('select ' . $i);
    usleep(SLEEP);
}
`

func TestDevtoolsExt_OneConnectionPerRequest(t *testing.T) {
	lines, conns := runDevtoolsExt(t, strings.Replace(extQueries, "SLEEP", "0", 1), false)
	if got := queryLines(t, lines); got != 5 {
		t.Fatalf("query events = %d, want 5: %v", got, lines)
	}
	if conns != 1 {
		t.Errorf("connections = %d, want 1 for the whole request", conns)
	}
}

func TestDevtoolsExt_ReopensAClosedConnection(t *testing.T) {
	lines, conns := runDevtoolsExt(t, strings.Replace(extQueries, "SLEEP", "100000", 1), true)
	if got := queryLines(t, lines); got != 5 {
		t.Fatalf("query events = %d, want all 5 despite the socket closing after each: %v", got, lines)
	}
	if conns != 5 {
		t.Errorf("connections = %d, want 5", conns)
	}
}

// A forked child must open its own connection; sharing the parent's would
// interleave the two processes' lines.
func TestDevtoolsExt_ForkedChildOpensItsOwnConnection(t *testing.T) {
	lines, conns := runDevtoolsExt(t, `<?php
if (!function_exists('pcntl_fork')) { echo 'SKIP'; exit; }
$db = new PDO('sqlite::memory:');
$db->query('select 1');
$pid = pcntl_fork();
$db->query('select 2');
if ($pid === 0) { exit(0); }
pcntl_waitpid($pid, $status);
$db->query('select 3');
`, false, "pcntl")
	if got := queryLines(t, lines); got != 4 {
		t.Fatalf("query events = %d, want 4 (1 before the fork, 1 from each side, 1 after): %v", got, lines)
	}
	if conns != 2 {
		t.Errorf("connections = %d, want 2, the parent's and the child's", conns)
	}
}
