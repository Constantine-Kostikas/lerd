package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/phpantom"
)

// lerd lsp speaks the protocol on stdout, so the editor or agent must see the
// server's bytes and nothing else, with the server rooted where it was started.
func TestRunLSPHandsStdioToTheServerInTheProjectDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	bin := phpantom.BinPath()
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\npwd -P\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin+".version", []byte(phpantom.Version), 0o644); err != nil {
		t.Fatal(err)
	}
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := runLSP(context.Background(), project, strings.NewReader("ping"), &out, &errOut); err != nil {
		t.Fatalf("runLSP: %v (stderr %q)", err, errOut.String())
	}
	if want := project + "\nping"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
}
