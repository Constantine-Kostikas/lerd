package serviceops

import (
	"time"

	"github.com/geodro/lerd/internal/podman"
)

// activationID is when the service's container last started: its launchd job
// only runs `podman run -d` and exits, so the container is what restarts.
func activationID(unit string) string {
	t, running := podman.ContainerStartedAt(unit)
	if !running {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}
