package nativephp

import (
	"strings"
	"testing"
)

// Every module ships under the same file name for every PHP it was built
// against, so a shared directory would leave the last version installed
// deciding which xdebug all the others load.
func TestModulesDirIsPerVersion(t *testing.T) {
	if ModulesDir("8.4") == ModulesDir("8.5") {
		t.Fatalf("8.4 and 8.5 share a modules directory: %s", ModulesDir("8.4"))
	}
	if !strings.Contains(ModulesDir("8.4"), "8.4") {
		t.Errorf("modules dir does not name its version: %s", ModulesDir("8.4"))
	}
}
