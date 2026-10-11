package nativephp

import (
	"errors"
	"path/filepath"
	"strconv"
	"time"

	"github.com/geodro/lerd/internal/config"
)

// PoolUnits runs the php-cgi pools as lerd service units. Set by the services
// package, which owns the Windows unit manager and already imports this one.
var PoolUnits interface {
	WritePool(name string, args, env []string) (bool, error)
	Start(name string) error
	Stop(name string) error
	Restart(name string) error
	IsActive(name string) bool
	RemoveServiceUnit(name string) error
}

// cgiChildren is the fixed pool php-cgi forks at start. It has no ondemand
// mode, so this is what a version holds resident while it is up.
const cgiChildren = 8

// LogPath is the unit's log, where php-cgi's stderr lands.
func LogPath(version string) string {
	return filepath.Join(config.DataDir(), "logs", UnitLabel(version)+".log")
}

// poolArgs binds loopback only: nginx reaches it through gvproxy's
// host.containers.internal, which forwards to the host's 127.0.0.1.
func poolArgs(binary string, port int) []string {
	return []string{binary, "-b", "127.0.0.1:" + strconv.Itoa(port)}
}

func poolEnv(version string) []string {
	return []string{
		"PHP_INI_SCAN_DIR=" + IniScanDir(version),
		"PHP_FCGI_CHILDREN=" + strconv.Itoa(cgiChildren),
	}
}

// Ensure brings a version's pool up, restarting it only when its definition
// changed. A php.ini edit reaches it through Reload.
func Ensure(version string) error { return ensure(version, false) }

// Reload restarts a version's pool so a changed php.ini reaches every child.
func Reload(version string) error {
	if err := ensure(version, true); err != nil {
		return err
	}
	waitAccepting(func() bool { return Running(version) }, reloadSettleTimeout, time.Sleep)
	return nil
}

func ensure(version string, restart bool) error {
	if PoolUnits == nil {
		return errors.New("the native php pool has no unit manager on this platform")
	}
	binary := FPMBinaryPath(version)
	if err := EnsureInstalled(version, binary); err != nil {
		return err
	}
	if err := WriteOverrides(version); err != nil {
		return err
	}
	port, err := PortFor(version)
	if err != nil {
		return err
	}
	label := UnitLabel(version)
	changed, err := PoolUnits.WritePool(label, poolArgs(binary, port), poolEnv(version))
	if err != nil {
		return err
	}
	if changed || (restart && PoolUnits.IsActive(label)) {
		return PoolUnits.Restart(label)
	}
	return PoolUnits.Start(label)
}

// Stop takes a version's pool down and removes its unit. Killing the php-cgi
// master takes its children with it.
func Stop(version string) error {
	if PoolUnits == nil {
		return nil
	}
	label := UnitLabel(version)
	if err := PoolUnits.Stop(label); err != nil {
		return err
	}
	return PoolUnits.RemoveServiceUnit(label)
}

func unitLoaded(label string) bool { return PoolUnits != nil && PoolUnits.IsActive(label) }
