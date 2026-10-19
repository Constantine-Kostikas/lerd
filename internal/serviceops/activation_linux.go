package serviceops

import "github.com/geodro/lerd/internal/systemd"

// activationID is systemd's invocation id, new on every start of the unit.
func activationID(unit string) string { return systemd.DBusInvocationID(unit) }
