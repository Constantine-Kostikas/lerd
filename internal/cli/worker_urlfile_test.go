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

// The dashboard's per-worker toggles stop by site name alone, so an empty
// checkout means the registered site's own.
func TestClearStoppedWorkerURLFile_emptyDirUsesSitePath(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("XDG_DATA_HOME", xdg)
	if err := os.MkdirAll(filepath.Join(xdg, "lerd", "frameworks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveFramework(&config.Framework{
		Name: "testfw",
		Workers: map[string]config.FrameworkWorker{
			"reverb": {Command: "php artisan reverb:start", Health: &config.WorkerHealth{URLFile: "storage/reverb.url"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	urlFile := filepath.Join(dir, "storage", "reverb.url")
	if err := os.MkdirAll(filepath.Dir(urlFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(urlFile, []byte("http://127.0.0.1:8080"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.AddSite(config.Site{Name: "myapp", Path: dir, Domains: []string{"myapp.test"}, Framework: "testfw"}); err != nil {
		t.Fatal(err)
	}

	ClearStoppedWorkerURLFile("myapp", "", "reverb")

	if _, err := os.Stat(urlFile); !os.IsNotExist(err) {
		t.Errorf("url file still present after stop by site name: %v", err)
	}
}

// A custom container site has no framework; its workers, health included, live
// in .lerd.yaml custom_workers, which the stop cleanup must read like start does.
func TestClearStoppedWorkerURLFile_customContainerSite(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("XDG_DATA_HOME", xdg)
	dir := t.TempDir()
	lerdYAML := "custom_workers:\n  dev:\n    command: npm run dev\n    health:\n      url_file: dev.url\n"
	if err := os.WriteFile(filepath.Join(dir, ".lerd.yaml"), []byte(lerdYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	urlFile := filepath.Join(dir, "dev.url")
	if err := os.WriteFile(urlFile, []byte("http://127.0.0.1:5173"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.AddSite(config.Site{Name: "nodeapp", Path: dir, Domains: []string{"nodeapp.test"}, ContainerPort: 3000}); err != nil {
		t.Fatal(err)
	}

	ClearStoppedWorkerURLFile("nodeapp", "", "dev")

	if _, err := os.Stat(urlFile); !os.IsNotExist(err) {
		t.Errorf("custom worker's url file still present after stop: %v", err)
	}
}
