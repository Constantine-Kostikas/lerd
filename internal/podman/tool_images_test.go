package podman

import (
	"strings"
	"testing"

	"github.com/geodro/lerd/internal/config"
)

// S3 buckets used to be provisioned by running the minio client image, until
// Docker Hub stopped answering for it on every tag and a fresh install could
// not create a bucket at all. lerd speaks S3 itself now, so no tool image may
// carry a client whose registry lerd does not control.
func TestToolImagesCarryNoS3Client(t *testing.T) {
	for _, img := range ToolImages() {
		if strings.Contains(img, "minio") || strings.Contains(img, "rclone") {
			t.Errorf("tool image %q is an S3 client; S3 is served by the compiled-in client", img)
		}
	}
}

// stripe-cli 1.51 made listen refuse to start without an event selection, and
// :latest moved onto it under every install; a listener has to name its events
// on an image that knows the flag.
func TestStripeListenExecStart_NamesItsEventsOnAPinnedImage(t *testing.T) {
	got := StripeListenExecStart("lerd-stripe-shop", config.StripeAuth{APIKey: "sk_test_x"}, "https://shop.test/stripe/webhook")
	for _, want := range []string{
		" run --rm --replace --name lerd-stripe-shop --network host ",
		StripeCLIImage + " listen --api-key sk_test_x --forward-to https://shop.test/stripe/webhook --skip-verify --all-snapshot",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ExecStart = %q, want it to contain %q", got, want)
		}
	}
	if strings.HasSuffix(StripeCLIImage, ":latest") {
		t.Errorf("StripeCLIImage = %q: an older image cached as :latest rejects --all-snapshot", StripeCLIImage)
	}
}

// A listener stopped before its container started leaves it in Created, where
// --rm never fires and lerd uninstall, which removes units, leaves it behind.
func TestStripeListenUnit_RemovesItsContainerOnStop(t *testing.T) {
	got := StripeListenUnit("shop", config.StripeAuth{APIKey: "sk_test_x"}, "https://shop.test/stripe/webhook")
	for _, want := range []string{
		"Description=Lerd Stripe Listener (shop)\n",
		"ExecStart=" + StripeListenExecStart("lerd-stripe-shop", config.StripeAuth{APIKey: "sk_test_x"}, "https://shop.test/stripe/webhook") + "\n",
		"ExecStopPost=-" + PodmanBin() + " rm -f --ignore lerd-stripe-shop\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("unit = %q, want it to contain %q", got, want)
		}
	}
}

// A saved Stripe CLI project is read from the host's config folder, mounted
// unlabelled so SELinux lets the container use a folder under home.
func TestStripeListenExecStart_SignsInWithASavedProject(t *testing.T) {
	auth := config.StripeAuth{ProjectName: "acme", ConfigDir: "/home/dev/.config/stripe"}
	got := StripeListenExecStart("lerd-stripe-shop", auth, "https://shop.test/stripe/webhook")
	want := PodmanBin() + " run --rm --replace --name lerd-stripe-shop --network host " +
		strings.Join(HostMountRunArgs(), " ") +
		" -v /home/dev/.config/stripe:/root/.config/stripe " +
		StripeCLIImage + " listen --project-name acme" +
		" --forward-to https://shop.test/stripe/webhook --skip-verify --all-snapshot"
	if got != want {
		t.Errorf("ExecStart =\n%s\nwant\n%s", got, want)
	}
}

// Login writes the saved project into the host's Stripe CLI config folder,
// the same file the listener later mounts read-only.
func TestStripeLoginArgs_SavesIntoTheHostConfig(t *testing.T) {
	got := strings.Join(StripeLoginArgs("/home/dev/.config/stripe", "acme"), " ")
	want := "run --rm -it " + strings.Join(HostMountRunArgs(), " ") +
		" -v /home/dev/.config/stripe:/root/.config/stripe " + StripeCLIImage + " login --project-name acme"
	if got != want {
		t.Errorf("args =\n%s\nwant\n%s", got, want)
	}
}
