package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVendorBinExists(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "vendor", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "pest"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(binDir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		bin  string
		want bool
	}{
		{"existing executable", "pest", true},
		{"missing", "phpunit", false},
		{"directory not bin", "subdir", false},
		{"empty name", "", false},
		{"path traversal rejected", "../bin/pest", false},
		{"nested name rejected", "foo/bar", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := VendorBinExists(dir, c.bin); got != c.want {
				t.Errorf("VendorBinExists(%q) = %v, want %v", c.bin, got, c.want)
			}
		})
	}

	// Empty cwd is rejected.
	if VendorBinExists("", "pest") {
		t.Error("VendorBinExists with empty cwd should return false")
	}

	// Project without vendor/bin returns false.
	empty := t.TempDir()
	if VendorBinExists(empty, "pest") {
		t.Error("VendorBinExists on project without vendor/bin should return false")
	}
}
