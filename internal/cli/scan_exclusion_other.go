//go:build !windows

package cli

// offerScanExclusion only does something on Windows, where it offers a
// Defender exclusion.
func offerScanExclusion(string) {}
