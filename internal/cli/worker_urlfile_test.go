package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

// A stopped dev server's URL file keeps pointing pages at a server that is gone.
func TestClearWorkerURLFile_removesFileOnStop(t *testing.T) {
	dir := t.TempDir()
	hot := filepath.Join(dir, "public", "hot")
	if err := os.MkdirAll(filepath.Dir(hot), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hot, []byte("https://myapp.test/@lerd-vite/"), 0o644); err != nil {
		t.Fatal(err)
	}

	clearWorkerURLFile(dir, &config.WorkerHealth{URLFile: "public/hot"})

	if _, err := os.Stat(hot); !os.IsNotExist(err) {
		t.Errorf("url file still present after stop: %v", err)
	}
}

// The path can come from a project's own .lerd.yaml, so it may not reach
// outside the checkout.
func TestClearWorkerURLFile_refusesPathOutsideCheckout(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "site")
	outside := filepath.Join(root, "keep")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(root, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}

	clearWorkerURLFile(dir, &config.WorkerHealth{URLFile: "../keep"})
	clearWorkerURLFile(dir, &config.WorkerHealth{URLFile: outside})
	clearWorkerURLFile(dir, &config.WorkerHealth{URLFile: "link/keep"})

	if _, err := os.Stat(outside); err != nil {
		t.Errorf("file outside the checkout was removed: %v", err)
	}
}

func TestClearWorkerURLFile_noHealthIsNoop(t *testing.T) {
	clearWorkerURLFile(t.TempDir(), nil)
	clearWorkerURLFile(t.TempDir(), &config.WorkerHealth{})
}
