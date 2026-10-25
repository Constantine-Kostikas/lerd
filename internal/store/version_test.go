package store

import (
	"os"
	"path/filepath"
	"testing"
)

// ── extractMajorFromConstraint ──────────────────────────────────────────────

func TestExtractMajorFromConstraint(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"^11.0", "11"},
		{"~7.1", "7"},
		{">=10.0 <11.0", "10"},
		{"^5.0|^6.0", "5"},
		{"11.*", "11"},
		{"^8.2", "8"},
		{"*", ""},
		{"", ""},
	}

	for _, c := range cases {
		got := extractMajorFromConstraint(c.input)
		if got != c.want {
			t.Errorf("extractMajorFromConstraint(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// ── DetectFrameworkVersion ───────────────────────────────────────────────────

func TestDetectFrameworkVersion(t *testing.T) {
	dir := t.TempDir()
	composerJSON := `{
		"require": {
			"php": "^8.2",
			"laravel/framework": "^11.0"
		}
	}`
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0o644) //nolint:errcheck

	got := DetectFrameworkVersion(dir, "laravel/framework")
	if got != "11" {
		t.Errorf("DetectFrameworkVersion() = %q, want %q", got, "11")
	}
}

func TestDetectFrameworkVersion_DevPackage(t *testing.T) {
	dir := t.TempDir()
	composerJSON := `{
		"require": {},
		"require-dev": {
			"symfony/framework-bundle": "~7.1"
		}
	}`
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0o644) //nolint:errcheck

	got := DetectFrameworkVersion(dir, "symfony/framework-bundle")
	if got != "7" {
		t.Errorf("DetectFrameworkVersion() = %q, want %q", got, "7")
	}
}

func TestDetectFrameworkVersion_NotFound(t *testing.T) {
	dir := t.TempDir()
	composerJSON := `{"require": {"foo/bar": "^1.0"}}`
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0o644) //nolint:errcheck

	got := DetectFrameworkVersion(dir, "laravel/framework")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestDetectFrameworkVersion_NoFile(t *testing.T) {
	dir := t.TempDir()
	got := DetectFrameworkVersion(dir, "laravel/framework")
	if got != "" {
		t.Errorf("expected empty for missing file, got %q", got)
	}
}

func TestDetectFrameworkVersion_FlexRequire(t *testing.T) {
	dir := t.TempDir()
	composerJSON := `{
		"require": {"symfony/flex": "^2.10"},
		"flex-require": {"symfony/framework-bundle": "*"},
		"extra": {"symfony": {"require": "7.4.*"}}
	}`
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0o644) //nolint:errcheck

	got := DetectFrameworkVersionWithKey(dir, "symfony/framework-bundle", "extra.symfony.require", "flex-require")
	if got != "7" {
		t.Errorf("DetectFrameworkVersionWithKey() = %q, want %q", got, "7")
	}
}

// The store resolves a version the way the rest of lerd does, so a project whose
// constraint spans two majors is placed on the one composer installed.
func TestDetectFrameworkVersion_prefersTheLock(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "composer.json"),
		[]byte(`{"require":{"laravel/framework":"^11.0 || ^12.0"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "composer.lock"),
		[]byte(`{"packages":[{"name":"laravel/framework","version":"v12.4.1"}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := DetectFrameworkVersion(dir, "laravel/framework"); got != "12" {
		t.Errorf("DetectFrameworkVersion() = %q, want 12", got)
	}
}
