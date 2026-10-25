package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/geodro/lerd/internal/config"
	"github.com/geodro/lerd/internal/download"
	"github.com/geodro/lerd/internal/tools"
)

func pinnedManifest(digest string) *tools.Manifest {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	return &tools.Manifest{Tools: map[string]tools.Tool{"php-native-8.4": {
		Version: "8.4.27",
		URL:     "https://github.com/lerd-env/php/releases/download/php-{version}",
		Assets:  map[string]string{platform: "lerd-php-{version}.tar.gz"},
		Digests: map[string]string{platform: digest},
	}}}
}

// A build republished under the same version gets a new checksum while the
// cached pins still carry the old one, so the download is retried once against
// freshly fetched pins instead of failing every install until the cache expires.
func TestDownloadRefetchesPinsOnChecksumMismatch(t *testing.T) {
	origVerified, origRefresh := verifiedDownload, refreshPins
	defer func() { verifiedDownload, refreshPins = origVerified, origRefresh }()

	var tried []string
	verifiedDownload = func(_ context.Context, _, _ string, _ os.FileMode, digest string, _ io.Writer) error {
		tried = append(tried, digest)
		if digest == "old" {
			return fmt.Errorf("%w: got new, expected old", download.ErrChecksumMismatch)
		}
		return nil
	}
	refreshPins = func(context.Context) *tools.Manifest { return pinnedManifest("new") }

	p := &pinnedTools{m: pinnedManifest("old")}
	if _, err := p.downloadCtx(context.Background(), "php-native-8.4", "dest", 0o644, io.Discard); err != nil {
		t.Fatal(err)
	}
	if len(tried) != 2 || tried[1] != "new" {
		t.Errorf("tried digests %v, want old then new", tried)
	}
}

// When the fresh pins carry the same checksum the bytes really are wrong, so
// the mismatch is reported rather than downloaded a second time.
func TestDownloadKeepsAMismatchTheRefreshDoesNotExplain(t *testing.T) {
	origVerified, origRefresh := verifiedDownload, refreshPins
	defer func() { verifiedDownload, refreshPins = origVerified, origRefresh }()

	calls := 0
	verifiedDownload = func(context.Context, string, string, os.FileMode, string, io.Writer) error {
		calls++
		return fmt.Errorf("%w: got bad, expected old", download.ErrChecksumMismatch)
	}
	refreshPins = func(context.Context) *tools.Manifest { return pinnedManifest("old") }

	p := &pinnedTools{m: pinnedManifest("old")}
	if _, err := p.downloadCtx(context.Background(), "php-native-8.4", "dest", 0o644, io.Discard); err == nil {
		t.Fatal("expected the mismatch to be reported")
	}
	if calls != 1 {
		t.Errorf("downloaded %d times, want 1", calls)
	}
}

// fakeNativeTarball writes a gzipped build holding what unpackNativePHP needs.
func fakeNativeTarball(t *testing.T, dest, version string) {
	t.Helper()
	f, err := os.Create(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "modules/", Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"php-native-" + version, "php-native-fpm-" + version} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: 0}); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gz.Close()
}

// A refreshed pin can name a newer patch than the one the install set out to
// fetch, and the build on disk has to be recorded as the one that arrived.
func TestNativeInstallRecordsThePatchTheRefreshFetched(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := os.MkdirAll(config.DataDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	origVerified, origRefresh := verifiedDownload, refreshPins
	defer func() { verifiedDownload, refreshPins = origVerified, origRefresh }()

	verifiedDownload = func(_ context.Context, _, dest string, _ os.FileMode, digest string, _ io.Writer) error {
		if digest == "old" {
			return fmt.Errorf("%w: got new, expected old", download.ErrChecksumMismatch)
		}
		fakeNativeTarball(t, dest, "8.4")
		return nil
	}
	fresh := pinnedManifest("new")
	tool := fresh.Tools["php-native-8.4"]
	tool.Version = "8.4.28"
	fresh.Tools["php-native-8.4"] = tool
	refreshPins = func(context.Context) *tools.Manifest { return fresh }

	got, err := installNativePHP(&pinnedTools{m: pinnedManifest("old")}, "8.4", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if got != "8.4.28" {
		t.Errorf("recorded patch %q, want the refreshed 8.4.28", got)
	}
}
