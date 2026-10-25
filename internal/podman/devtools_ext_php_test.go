package podman

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// runDevtoolsExt runs body with the compiled lerd_devtools capturing queries,
// returning the lines and connections it sent.
func runDevtoolsExt(t *testing.T, body string, dropEach bool, exts ...string) ([]string, int) {
	t.Helper()
	php, args, dir, sink := devtoolsExtSetup(t, body, dropEach, exts...)
	if out, err := exec.Command(php, append(args, filepath.Join(dir, "probe.php"))...).CombinedOutput(); err != nil {
		t.Fatalf("php run failed: %v\n%s", err, out)
	}
	return sink.result()
}

// devtoolsExtSetup writes body to dir/probe.php and returns the php arguments
// that load lerd_devtools against a capture socket. Skipped unless php can load
// the extension, which holds in lerd's FPM image and not on a CI runner.
func devtoolsExtSetup(t *testing.T, body string, dropEach bool, exts ...string) (string, []string, string, *capture) {
	t.Helper()
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("php not installed")
	}
	args := []string{"-n", "-d", "auto_prepend_file=", "-d", "extension=lerd_devtools"}
	for _, e := range exts {
		args = append(args, "-d", "extension="+e)
	}
	// Match the module line: a failed load prints a warning naming it too.
	if out, _ := exec.Command(php, append(args, "-m")...).Output(); !slices.Contains(strings.Split(string(out), "\n"), "lerd_devtools") {
		t.Skip("php cannot load lerd_devtools; run inside lerd's FPM image")
	}

	dir := t.TempDir()
	flag := filepath.Join(dir, "enabled.flag")
	if err := os.WriteFile(flag, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// lerd's own php wrapper runs in a container that can't see this dir.
	if out, _ := exec.Command(php, "-n", "-r", "echo file_exists("+phpQuote(flag)+") ? 'Y' : 'N';").Output(); !strings.Contains(string(out), "Y") {
		t.Skip("php cannot read host files (containerised/sandboxed wrapper); native php needed")
	}
	if err := os.WriteFile(filepath.Join(dir, "probe.php"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sink := listenCapture(t, dropEach)
	args = append(args,
		"-d", "lerd.devtools_host=unix://"+sink.sock,
		"-d", "lerd.devtools_kinds=query",
		"-d", "lerd.devtools_flag="+flag,
		"-d", "lerd.assets_dir="+dir)
	return php, args, dir, sink
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

// The CLI exits right after its one request, which would close the socket
// anyway, so this serves requests from a long-lived php -S instead.
func TestDevtoolsExt_ClosesConnectionAtRequestEnd(t *testing.T) {
	php, args, dir, sink := devtoolsExtSetup(t, `<?php
(new PDO('sqlite::memory:'))->query('select 1');
echo 'ok';
`, false)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	srv := exec.Command(php, append(args, "-S", addr, "-t", dir)...)
	if err := srv.Start(); err != nil {
		t.Fatalf("php -S: %v", err)
	}
	t.Cleanup(func() { _ = srv.Process.Kill(); _ = srv.Wait() })

	var resp *http.Response
	for i := 0; i < 50; i++ {
		if resp, err = http.Get("http://" + addr + "/probe.php"); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET probe: %v", err)
	}
	resp.Body.Close()

	lines, conns := sink.result()
	sink.mu.Lock()
	ended := sink.ended
	sink.mu.Unlock()
	if queryLines(t, lines) != 1 || conns != 1 {
		t.Fatalf("got %d lines over %d connections, want 1 over 1: %v", len(lines), conns, lines)
	}
	if ended != 1 {
		t.Errorf("connection still open after the request ended, with php still running")
	}
}
