package nativephp

import (
	"reflect"
	"strings"
	"testing"
)

type fakePoolUnits struct {
	args, env []string
	changed   bool
	active    bool
	calls     []string
}

func (f *fakePoolUnits) WritePool(name string, args, env []string) (bool, error) {
	f.calls = append(f.calls, "write "+name)
	f.args, f.env = args, env
	return f.changed, nil
}
func (f *fakePoolUnits) Start(name string) error   { f.calls = append(f.calls, "start"); return nil }
func (f *fakePoolUnits) Stop(name string) error    { f.calls = append(f.calls, "stop"); return nil }
func (f *fakePoolUnits) Restart(name string) error { f.calls = append(f.calls, "restart"); return nil }
func (f *fakePoolUnits) IsActive(string) bool      { return f.active }
func (f *fakePoolUnits) RemoveServiceUnit(string) error {
	f.calls = append(f.calls, "remove")
	return nil
}

func withFakePool(t *testing.T, f *fakePoolUnits) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	seedWindowsBuild(t, "8.4", "php-cgi.exe")
	prev := PoolUnits
	PoolUnits = f
	t.Cleanup(func() { PoolUnits = prev })
}

func TestWindowsEnsureRunsPHPCGIOnLoopback(t *testing.T) {
	f := &fakePoolUnits{}
	withFakePool(t, f)
	if err := Ensure("8.4"); err != nil {
		t.Fatal(err)
	}
	if want := []string{FPMBinaryPath("8.4"), "-b", "127.0.0.1:9484"}; !reflect.DeepEqual(f.args, want) {
		t.Errorf("args = %v, want %v", f.args, want)
	}
	env := strings.Join(f.env, "\n")
	for _, want := range []string{"PHP_INI_SCAN_DIR=" + IniScanDir("8.4"), "PHP_FCGI_CHILDREN="} {
		if !strings.Contains(env, want) {
			t.Errorf("env lacks %q: %v", want, f.env)
		}
	}
	if want := []string{"write lerd-native-php84", "start"}; !reflect.DeepEqual(f.calls, want) {
		t.Errorf("calls = %v, want %v", f.calls, want)
	}
}

func TestWindowsEnsureRestartsAChangedPool(t *testing.T) {
	f := &fakePoolUnits{changed: true, active: true}
	withFakePool(t, f)
	if err := Ensure("8.4"); err != nil {
		t.Fatal(err)
	}
	if got := f.calls[len(f.calls)-1]; got != "restart" {
		t.Errorf("last call = %s, want restart", got)
	}
}

func TestWindowsEnsureLeavesARunningPoolAlone(t *testing.T) {
	f := &fakePoolUnits{active: true}
	withFakePool(t, f)
	if err := Ensure("8.4"); err != nil {
		t.Fatal(err)
	}
	if got := f.calls[len(f.calls)-1]; got != "start" {
		t.Errorf("last call = %s, want start (a no-op on a live unit)", got)
	}
}

func TestWindowsStopRemovesTheUnit(t *testing.T) {
	f := &fakePoolUnits{}
	withFakePool(t, f)
	if err := Stop("8.4"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"stop", "remove"}; !reflect.DeepEqual(f.calls, want) {
		t.Errorf("calls = %v, want %v", f.calls, want)
	}
}
