package podman

import (
	"fmt"
	"strings"

	"github.com/geodro/lerd/internal/config"
)

// lerd runs a few public images as throw-away containers rather than as
// services: no quadlet names them and no container outlives the command, so
// nothing in lerd's config references them. They are listed here so cleanup can
// tell them apart from an image nobody wants, and so the refs live in one place
// instead of being repeated at each call site.
const (
	// ProbeImage backs the IPv6 network probe, which runs it with --pull never:
	// removing it downgrades the probe to inconclusive rather than pulling.
	ProbeImage = "alpine:latest"
)

// StripeCLIImage is the image a site's stripe listener worker runs. Pinned,
// because an older image cached as :latest rejects the --all-snapshot flag
// that stripe-cli 1.51 made mandatory.
const StripeCLIImage = "docker.io/stripe/stripe-cli:v1.53.1"

// StripeListenExecStart is the ExecStart of a site's stripe listener unit.
// --all-snapshot forwards every classic webhook event, which is what listen
// did before stripe-cli made choosing the events mandatory.
func StripeListenExecStart(containerName string, auth config.StripeAuth, forwardTo string) string {
	mount, signIn := "", "--api-key "+auth.APIKey
	if auth.ProjectName != "" {
		// Read-write and the whole folder: a login keeps its OAuth token in
		// credentials.json beside config.toml and refreshes it there.
		mount = strings.Join(HostMountRunArgs(), " ") + " -v " + auth.ConfigDir + ":/root/.config/stripe "
		signIn = "--project-name " + auth.ProjectName
	}
	return fmt.Sprintf("%s run --rm --replace --name %s --network host %s%s%s listen %s --forward-to %s --skip-verify --all-snapshot",
		PodmanBin(), containerName, UnitRunLogArgs(), mount, StripeCLIImage, signIn, forwardTo)
}

// ToolImages lists every image lerd runs as a throw-away container.
func ToolImages() []string {
	return []string{ProbeImage}
}

// StripeLoginArgs are the podman run args for `stripe login`, saving the
// login into the host's Stripe CLI config folder at configDir.
func StripeLoginArgs(configDir, projectName string) []string {
	args := append([]string{"run", "--rm", "-it"}, HostMountRunArgs()...)
	return append(args, "-v", configDir+":/root/.config/stripe", StripeCLIImage, "login", "--project-name", projectName)
}

// StripeListenUnit is a site's stripe listener service unit. ExecStopPost
// removes the container, because a stop that lands before it starts leaves it
// in Created, where --rm never fires.
func StripeListenUnit(siteName string, auth config.StripeAuth, forwardTo string) string {
	containerName := "lerd-stripe-" + siteName
	return fmt.Sprintf(`[Unit]
Description=Lerd Stripe Listener (%s)
After=network.target

[Service]
Type=simple
Restart=on-failure
RestartSec=5
ExecStart=%s
ExecStopPost=-%s rm -f --ignore %s

[Install]
WantedBy=default.target
`, siteName, StripeListenExecStart(containerName, auth, forwardTo), PodmanBin(), containerName)
}
