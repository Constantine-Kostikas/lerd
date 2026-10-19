package serviceops

import "github.com/geodro/lerd/internal/podman"

// activationID names the service container's current run: its launchd job only
// runs `podman run -d` and exits, so the container is what restarts.
func activationID(unit string) string { return podman.ContainerRunID(unit) }
