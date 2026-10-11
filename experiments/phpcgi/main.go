// Command phpcgi fires concurrent FastCGI requests at a php-cgi listener and
// reports which worker PID answered each one, to measure how php-cgi.exe
// handles concurrency on Windows. Experimental, not part of lerd.
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	typeBeginRequest = 1
	typeEndRequest   = 3
	typeParams       = 4
	typeStdin        = 5
	typeStdout       = 6
	typeStderr       = 7
)

func writeRecord(w io.Writer, typ uint8, id uint16, content []byte) error {
	pad := (8 - len(content)%8) % 8
	h := []byte{1, typ, byte(id >> 8), byte(id), byte(len(content) >> 8), byte(len(content)), byte(pad), 0}
	if _, err := w.Write(h); err != nil {
		return err
	}
	if _, err := w.Write(content); err != nil {
		return err
	}
	_, err := w.Write(make([]byte, pad))
	return err
}

func encodeLen(n int) []byte {
	if n < 128 {
		return []byte{byte(n)}
	}
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(n)|1<<31)
	return b
}

func encodeParams(p map[string]string) []byte {
	var out []byte
	for k, v := range p {
		out = append(out, encodeLen(len(k))...)
		out = append(out, encodeLen(len(v))...)
		out = append(out, k...)
		out = append(out, v...)
	}
	return out
}

// request runs one FastCGI request and returns the response body.
func request(addr, script, query string, timeout time.Duration) (string, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	w := bufio.NewWriter(conn)
	const id = 1
	if err := writeRecord(w, typeBeginRequest, id, []byte{0, 1, 0, 0, 0, 0, 0, 0}); err != nil {
		return "", err
	}
	params := encodeParams(map[string]string{
		"SCRIPT_FILENAME":   script,
		"REQUEST_METHOD":    "GET",
		"QUERY_STRING":      query,
		"SERVER_PROTOCOL":   "HTTP/1.1",
		"GATEWAY_INTERFACE": "CGI/1.1",
		"REQUEST_URI":       "/?" + query,
	})
	if err := writeRecord(w, typeParams, id, params); err != nil {
		return "", err
	}
	if err := writeRecord(w, typeParams, id, nil); err != nil {
		return "", err
	}
	if err := writeRecord(w, typeStdin, id, nil); err != nil {
		return "", err
	}
	if err := w.Flush(); err != nil {
		return "", err
	}
	r := bufio.NewReader(conn)
	var stdout, stderr strings.Builder
	for {
		h := make([]byte, 8)
		if _, err := io.ReadFull(r, h); err != nil {
			return stdout.String(), fmt.Errorf("reading header: %w", err)
		}
		n := int(h[4])<<8 | int(h[5])
		body := make([]byte, n+int(h[6]))
		if _, err := io.ReadFull(r, body); err != nil {
			return stdout.String(), fmt.Errorf("reading body: %w", err)
		}
		switch h[1] {
		case typeStdout:
			stdout.Write(body[:n])
		case typeStderr:
			stderr.Write(body[:n])
		case typeEndRequest:
			out := stdout.String()
			if i := strings.Index(out, "\r\n\r\n"); i >= 0 {
				out = out[i+4:]
			}
			if stderr.Len() > 0 {
				out += " [stderr: " + strings.TrimSpace(stderr.String()) + "]"
			}
			return strings.TrimSpace(out), nil
		}
	}
}

func main() {
	addr := flag.String("addr", "127.0.0.1:9484", "php-cgi FastCGI address")
	script := flag.String("script", "", "absolute path of the PHP script")
	conc := flag.Int("c", 8, "concurrent requests")
	total := flag.Int("n", 8, "total requests")
	query := flag.String("q", "", "query string")
	timeout := flag.Duration("timeout", 60*time.Second, "per-request timeout")
	flag.Parse()

	type result struct {
		body string
		dur  time.Duration
		err  error
	}
	results := make([]result, *total)
	jobs := make(chan int)
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < *conc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				t := time.Now()
				body, err := request(*addr, *script, *query, *timeout)
				results[j] = result{body, time.Since(t), err}
			}
		}()
	}
	for i := 0; i < *total; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	wall := time.Since(start)

	pids := map[string]int{}
	failed := 0
	var durs []time.Duration
	for i, r := range results {
		if r.err != nil {
			failed++
			fmt.Printf("req %2d  ERROR %v (%s)\n", i, r.err, r.body)
			continue
		}
		pids[r.body]++
		durs = append(durs, r.dur)
	}
	sort.Slice(durs, func(a, b int) bool { return durs[a] < durs[b] })
	fmt.Printf("requests=%d concurrency=%d ok=%d failed=%d wall=%s\n", *total, *conc, *total-failed, failed, wall.Round(time.Millisecond))
	if len(durs) > 0 {
		fmt.Printf("latency min=%s p50=%s max=%s\n", durs[0].Round(time.Millisecond), durs[len(durs)/2].Round(time.Millisecond), durs[len(durs)-1].Round(time.Millisecond))
	}
	keys := make([]string, 0, len(pids))
	for k := range pids {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Printf("distinct responders=%d\n", len(keys))
	for _, k := range keys {
		fmt.Printf("  %s x%d\n", k, pids[k])
	}
	if failed > 0 {
		os.Exit(1)
	}
}
