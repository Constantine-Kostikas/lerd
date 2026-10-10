package nativephp

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/geodro/lerd/internal/config"
)

// UnitLabel names the job supervising a version's native pool.
func UnitLabel(version string) string {
	return "lerd-native-php" + strings.ReplaceAll(version, ".", "")
}

// ConfPath is the generated php-fpm.conf for a version's native listener.
func ConfPath(version string) string {
	return filepath.Join(config.RunDir(), "native", version, "php-fpm.conf")
}

// reloadSettleTimeout bounds the wait for a restarted pool to start accepting.
const reloadSettleTimeout = 10 * time.Second

// reloadSettleInterval is how often the pool is probed while settling.
const reloadSettleInterval = 50 * time.Millisecond

// waitAccepting polls until the pool accepts or the budget runs out, reporting
// whether it came up. tick is injected so a test spends no real time.
func waitAccepting(accepting func() bool, budget time.Duration, tick func(time.Duration)) bool {
	for waited := time.Duration(0); ; waited += reloadSettleInterval {
		if accepting() {
			return true
		}
		if waited >= budget {
			return false
		}
		tick(reloadSettleInterval)
	}
}

// writeIfChanged writes body to path when it differs from what is there,
// reporting whether it did. Callers restart on the back of that, so rewriting
// identical content must not look like a change.
func writeIfChanged(path, body string) (bool, error) {
	existing, err := os.ReadFile(path)
	if err == nil && string(existing) == body {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return false, err
	}
	return true, nil
}
