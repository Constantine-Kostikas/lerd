package stats

import "testing"

const launchctlFixture = `PID	Status	Label
88367	0	lerd-native-php84
1881	0	com.lerd.lerd-tray
-	0	com.lerd.lerd-nginx
1098	0	com.lerd.lerd-ui
500	0	com.lerd.lerd-redis
700	0	com.apple.speech.speechdatainstallerd
`

// pid ppid rss(KB) %cpu
const psFixture = `    1     0  1000  0.0
88367     1  5000  0.5
88400 88367  20000  10.0
88401 88400  1000  1.5
 1881     1  3000  0.0
 1098     1  40000  2.0
  500     1  9999  9.9
  700     1  7777  7.7
`

func TestLaunchdRows(t *testing.T) {
	rows := launchdRows(launchctlFixture, psFixture, gb(16), map[string]bool{"lerd-redis": true})
	got := map[string]ContainerStat{}
	for _, r := range rows {
		got[r.Name] = r
	}
	if len(got) != 3 {
		t.Fatalf("rows = %+v, want php84, tray and ui only", rows)
	}

	php := got["lerd-native-php84"]
	if php.MemBytes != 26000*1024 {
		t.Errorf("php84 mem = %d, want master plus all descendants", php.MemBytes)
	}
	if php.CPUPercent != 12.0 {
		t.Errorf("php84 cpu = %v, want 12", php.CPUPercent)
	}
	if php.MemLimit != gb(16) {
		t.Errorf("php84 limit = %d, want host RAM", php.MemLimit)
	}
	if got["lerd-ui"].MemBytes != 40000*1024 {
		t.Errorf("ui mem = %d", got["lerd-ui"].MemBytes)
	}
}

func TestLaunchdRowsSkipsUnknownPID(t *testing.T) {
	rows := launchdRows("123\t0\tcom.lerd.lerd-ui\n", "1 0 10 0.0\n", gb(1), nil)
	if len(rows) != 0 {
		t.Fatalf("rows = %+v, want none for a pid ps does not list", rows)
	}
}
