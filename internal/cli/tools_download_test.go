package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"testing"

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
