package services

import "github.com/geodro/lerd/internal/nativephp"

// nativePoolUnits hands the native php-cgi pools to the unit manager. A pool is
// a service unit written from its argv directly, with no unit text to quote a
// Windows path through, and restarted whenever php-cgi's master exits.
type nativePoolUnits struct{ *windowsServiceManager }

func (nativePoolUnits) WritePool(name string, args, env []string) (bool, error) {
	return writeDef(name, unitDef{Kind: kindService, Args: args, Restart: keepAliveAlways, Env: env})
}

func init() { nativephp.PoolUnits = nativePoolUnits{&windowsServiceManager{}} }
