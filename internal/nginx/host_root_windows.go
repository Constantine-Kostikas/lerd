package nginx

import (
	"path/filepath"

	"github.com/geodro/lerd/internal/hostpath"
)

// hostRoot is a native site's document root as php-cgi on Windows opens it.
// nginx reaches the project at its /mnt/<drive> path inside the VM, which does
// not exist for a PHP running on the host. Forward slashes, which PHP accepts
// and nginx's quoting leaves alone.
func hostRoot(sitePath, publicDir string, native bool) string {
	if !native {
		return ""
	}
	return filepath.ToSlash(filepath.Join(hostpath.FromVM(sitePath), publicDir))
}
