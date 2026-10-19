package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	nodeDet "github.com/geodro/lerd/internal/node"
	"github.com/geodro/lerd/internal/podman"
)

// containerNodeBin is the musl build of cwd's Node version when one is on
// disk, so node and npm in a PHP container match the Node lerd runs for the
// site on the host instead of whatever Alpine ships; "" when there is none.
func containerNodeBin(cwd string) string {
	if cwd == "" {
		return ""
	}
	spec, _ := nodeDet.DetectVersion(cwd)
	return nodeDet.MuslBinFor(spec)
}

// withContainerNode puts cwd's Node in front of the image's PATH entries in
// path, which starts with podman.ContainerPath or carries it after a prefix.
func withContainerNode(cwd, path string) string {
	bin := containerNodeBin(cwd)
	if bin == "" {
		return path
	}
	return strings.Replace(path, podman.ContainerPath, bin+":"+podman.ContainerPath, 1)
}

// ensureContainerNode fetches the musl build of the exact version the host
// runs for cwd, once. lerd shell calls it: it is interactive, so the one-off
// download has someone to wait for it, and the exec shims never block on it.
func ensureContainerNode(cwd string, w io.Writer) {
	spec, _ := nodeDet.DetectVersion(cwd)
	mgr := nodeDet.Active()
	if spec == "" || !mgr.Available() {
		return
	}
	out, err := mgr.Command(spec, "node", []string{"-p", "process.version"}).Output()
	if err != nil {
		return // the host has no Node for this site either, so there is nothing to match
	}
	version := strings.TrimSpace(string(out))
	if nodeDet.MuslBinFor(strings.TrimPrefix(version, "v")) != "" {
		return
	}
	fmt.Fprintf(w, "Fetching Node %s for the PHP container, once\n", version)
	if _, err := nodeDet.MuslBin(context.Background(), version, w); err != nil {
		fmt.Fprintf(w, "  ! the shell keeps the image's own Node: %v\n", err)
	}
}
