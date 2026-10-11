package nativephp

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/geodro/lerd/internal/config"
)

// moduleExt is the suffix of the loadable extensions a native build ships.
const moduleExt = ".dll"

// BuildDir holds a version's official windows.php.net build. It stays a
// directory rather than two binaries in BinDir: php.exe and php-cgi.exe load
// php8.dll, the bundled extensions and the libraries they need from beside
// themselves.
func BuildDir(version string) string {
	return filepath.Join(config.DataDir(), "native-php", version, "php")
}

// BuildConfDir holds the ini the build ships to enable its extensions.
func BuildConfDir(version string) string {
	return filepath.Join(config.DataDir(), "native-php", version, "conf.d")
}

// BinaryPath is the CLI of a version's native build.
func BinaryPath(version string) string {
	return filepath.Join(BuildDir(version), "php.exe")
}

// FPMBinaryPath is the php-cgi.exe that serves a version's requests. Windows
// has no PHP-FPM; php-cgi with PHP_FCGI_CHILDREN is its process manager.
func FPMBinaryPath(version string) string {
	return filepath.Join(BuildDir(version), "php-cgi.exe")
}

// ListInstalled returns the PHP versions whose build carries both the CLI and
// php-cgi, sorted.
func ListInstalled() []string {
	entries, err := os.ReadDir(filepath.Join(config.DataDir(), "native-php"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		v := e.Name()
		_, cliErr := os.Stat(BinaryPath(v))
		_, cgiErr := os.Stat(FPMBinaryPath(v))
		if e.IsDir() && cliErr == nil && cgiErr == nil {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// Windows has no execute bit; the file existing as a regular file is the test.
func isExecutable(info fs.FileInfo) bool { return info.Mode().IsRegular() }

// buildIniDirs scans the build's own ini first, so the extensions it ships are
// on before lerd's fragments configure them.
func buildIniDirs(version string) []string { return []string{BuildConfDir(version)} }

// buildIni points PHP at the build's ext directory, which only lerd knows: the
// default is C:\php\ext, wherever the build was unpacked.
func buildIni(version string) string {
	return "extension_dir=" + filepath.Join(BuildDir(version), "ext") + "\n"
}
