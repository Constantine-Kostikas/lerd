package store

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/geodro/lerd/internal/config"
)

// DetectFrameworkVersion reads composer.json in dir and returns the major version
// of the given composer package. It searches require, require-dev, and any extra
// sections specified. If the constraint is unparseable (e.g. "*"), it falls back
// to the versionKey dot-path in composer.json.
func DetectFrameworkVersion(dir string, composerPackageName string, extraSections ...string) string {
	return DetectFrameworkVersionWithKey(dir, composerPackageName, "", extraSections...)
}

// DetectFrameworkVersionWithKey is like DetectFrameworkVersion but also accepts a
// versionKey (dot-path into composer.json) as a fallback when the constraint is "*".
// The answer comes from config, which reads what composer resolved before what
// the manifest asked for, so the store resolves a version the same way the rest
// of lerd does rather than from a second copy of the manifest walk.
func DetectFrameworkVersionWithKey(dir string, composerPackageName string, versionKey string, extraSections ...string) string {
	return config.DetectPackageMajor(dir, composerPackageName, versionKey, extraSections...)
}

// extractMajorFromConstraint extracts the major version from a composer constraint.
// Examples: "^11.0" -> "11", "~7.1" -> "7", ">=10.0 <11.0" -> "10", "^5.0|^6.0" -> "5"
func extractMajorFromConstraint(constraint string) string {
	for i := 0; i < len(constraint); i++ {
		b := constraint[i]
		if b >= '0' && b <= '9' {
			j := i
			for j < len(constraint) && constraint[j] >= '0' && constraint[j] <= '9' {
				j++
			}
			return constraint[i:j]
		}
	}
	return ""
}

// DetectVersionFromFile reads a file and extracts the major version using a regex pattern.
func DetectVersionFromFile(dir, relPath, pattern string) string {
	data, err := os.ReadFile(filepath.Join(dir, relPath))
	if err != nil {
		return ""
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}
	m := re.FindSubmatch(data)
	if len(m) < 2 {
		return ""
	}
	return extractMajorFromConstraint(string(m[1]))
}
