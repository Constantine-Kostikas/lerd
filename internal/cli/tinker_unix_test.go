//go:build !windows

package cli

import (
	"reflect"
	"testing"

	"github.com/geodro/lerd/internal/podman"
)

func TestTinkerEnvArgs(t *testing.T) {
	got := tinkerEnvArgs("/home/u/site", "/home/u", "/home/u/.config/composer")
	want := []string{
		"--env", "HOME=/home/u",
		"--env", "COMPOSER_HOME=/home/u/.config/composer",
		"--env", "PATH=/home/u/site/vendor/bin:" + podman.ContainerPath + ":/home/u/.config/composer/vendor/bin",
		"--env", "NO_COLOR=1",
		"--env", "TERM=dumb",
		"--env", "PSYSH_TRUST_PROJECT=1",
		"--env", "LERD_DUMP_PASSTHROUGH=1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("env args mismatch.\n got:  %v\n want: %v", got, want)
	}
}
