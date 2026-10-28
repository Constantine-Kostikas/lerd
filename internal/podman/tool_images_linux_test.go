package podman

import (
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

// Under systemd the default journald driver logs every listener line twice,
// once from the unit's output and once from podman itself.
func TestStripeListenExecStart_LogsEachLineOnce(t *testing.T) {
	got := StripeListenExecStart("lerd-stripe-shop", config.StripeAuth{APIKey: "sk_test_x"}, "https://shop.test/stripe/webhook")
	if !strings.Contains(got, " --network host --log-driver none ") {
		t.Errorf("ExecStart = %q, want --log-driver none", got)
	}
}
