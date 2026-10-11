//go:build !windows

package nginx

// hostRoot is empty off Windows: a native PHP there sees the project at the
// same path nginx does.
func hostRoot(string, string, bool) string { return "" }
