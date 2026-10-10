package nginx

import "testing"

func TestWindowsHostRootIsTheWindowsPathOfANativeSite(t *testing.T) {
	cases := []struct {
		path, public string
		native       bool
		want         string
	}{
		{`C:\Sites\shop`, "public", true, "C:/Sites/shop/public"},
		{"/mnt/d/work/shop", "public", true, "D:/work/shop/public"},
		{`C:\Sites\shop`, ".", true, "C:/Sites/shop"},
		{`C:\Sites\shop`, "public", false, ""},
	}
	for _, c := range cases {
		if got := hostRoot(c.path, c.public, c.native); got != c.want {
			t.Errorf("hostRoot(%q, %q, %v) = %q, want %q", c.path, c.public, c.native, got, c.want)
		}
	}
}
