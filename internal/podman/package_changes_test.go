package podman

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitApkPackage(t *testing.T) {
	for in, want := range map[string][2]string{
		"brotli-libs-1.2.1-r0":      {"brotli-libs", "1.2.1-r0"},
		"libstdc++-14.2.0-r6":       {"libstdc++", "14.2.0-r6"},
		"php85-pecl-redis-6.2.0-r1": {"php85-pecl-redis", "6.2.0-r1"},
	} {
		name, ver, ok := splitApkPackage(in)
		if !ok || name != want[0] || ver != want[1] {
			t.Errorf("splitApkPackage(%q) = %q, %q, %v", in, name, ver, ok)
		}
	}
	for _, in := range []string{"", "musl", "WARNING:", "foo-1.0"} {
		if _, _, ok := splitApkPackage(in); ok {
			t.Errorf("splitApkPackage(%q) accepted a line that is not a package", in)
		}
	}
}

func TestPackageChanges(t *testing.T) {
	before := map[string]string{"brotli-libs": "1.2.0-r1", "nodejs": "24.18.1-r0", "gone": "1.0-r0", "zlib": "1.3.1-r2"}
	after := map[string]string{"brotli-libs": "1.2.1-r0", "nodejs": "24.22.1-r0", "zlib": "1.3.1-r2", "new": "2.0-r0"}
	want := []string{
		"brotli-libs 1.2.0-r1 → 1.2.1-r0",
		"gone 1.0-r0 removed",
		"new 2.0-r0 added",
		"nodejs 24.18.1-r0 → 24.22.1-r0",
	}
	if got := packageChanges(before, after); !reflect.DeepEqual(got, want) {
		t.Errorf("packageChanges = %q, want %q", got, want)
	}
}

func TestWritePackageChangesSaysWhenNothingMoved(t *testing.T) {
	var b strings.Builder
	writePackageChanges(&b, nil)
	if b.String() != "  No package changes.\n" {
		t.Errorf("got %q", b.String())
	}
}
