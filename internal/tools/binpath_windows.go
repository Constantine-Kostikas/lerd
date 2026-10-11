package tools

import (
	"strings"

	"github.com/geodro/lerd/internal/nativephp"
)

// platformBinPath places a native PHP build's CLI, which on Windows lives in
// the build's own directory rather than in BinDir, so its stamps sit beside it.
func platformBinPath(name string) (string, bool) {
	v, ok := strings.CutPrefix(name, "php-native-")
	if !ok {
		return "", false
	}
	return nativephp.BinaryPath(v), true
}
