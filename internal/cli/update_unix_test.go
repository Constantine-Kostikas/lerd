//go:build !windows

package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadReleaseBinary_success(t *testing.T) {
	// Build a fake tar.gz to serve
	tmp := t.TempDir()
	makeFakeTarGz(t, tmp, "#!/bin/sh\necho lerd")

	archiveBytes, err := os.ReadFile(filepath.Join(tmp, "lerd_0.1.0_linux_amd64.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(archiveBytes)
	}))
	defer srv.Close()

	orig := githubDownloadBases
	githubDownloadBases = func() []string { return []string{srv.URL} }
	defer func() { githubDownloadBases = orig }()

	binary, cleanup, err := downloadReleaseBinary("v0.1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if _, err := os.Stat(binary); err != nil {
		t.Errorf("binary not found at %s: %v", binary, err)
	}
}

func TestCopyFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")

	if err := os.WriteFile(src, []byte("hello lerd"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst, 0755); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello lerd" {
		t.Errorf("got %q, want %q", got, "hello lerd")
	}

	info, _ := os.Stat(dst)
	if info.Mode() != 0755 {
		t.Errorf("mode = %v, want 0755", info.Mode())
	}
}
