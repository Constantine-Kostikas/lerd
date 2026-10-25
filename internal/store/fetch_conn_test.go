package store

import (
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

// A refresh fetches in parallel, and Go keeps only two idle connections per host,
// so a second wave would handshake again. The first wave is held until every
// request has arrived, so it opens one connection each and fills the pool.
func TestFetch_ReusesTheConnectionsAcrossParallelWaves(t *testing.T) {
	var conns, arrived atomic.Int64
	release := make(chan struct{})
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if arrived.Add(1) == FetchConcurrency {
			close(release)
		}
		<-release
		_, _ = w.Write([]byte("ok"))
	}))
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			conns.Add(1)
		}
	}
	srv.Start()
	defer srv.Close()

	c := &Client{BaseURL: srv.URL}
	wave := func() {
		var wg sync.WaitGroup
		for range FetchConcurrency {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := c.fetch("x.yaml"); err != nil {
					t.Errorf("fetch: %v", err)
				}
			}()
		}
		wg.Wait()
	}
	wave()
	first := conns.Load()
	wave()
	second := conns.Load() - first

	// Half a wave is well clear of both: a pooled second wave opens none or one,
	// an unpooled one opens most of eight.
	if second >= FetchConcurrency/2 {
		t.Errorf("the second wave opened %d connections of its own after the first opened %d, want fewer than %d: it is handshaking again rather than reusing the pool",
			second, first, FetchConcurrency/2)
	}
}
