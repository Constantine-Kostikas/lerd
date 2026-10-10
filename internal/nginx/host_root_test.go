package nginx

import (
	"strings"
	"testing"
)

// A PHP on the host cannot open the VM path nginx serves from, so a site with a
// HostRoot sends that path as SCRIPT_FILENAME and DOCUMENT_ROOT instead.
func TestVhostSendsTheHostRootToANativePHP(t *testing.T) {
	for _, name := range []string{"vhost.conf.tmpl", "vhost-ssl.conf.tmpl"} {
		out := renderVhostForTest(t, name, VhostData{
			Domain:         "shop.test",
			Path:           "/mnt/c/Sites/My Shop",
			PublicDir:      "public",
			FPMContainer:   "host.containers.internal",
			CertDomain:     "shop.test",
			RequestTimeout: 60,
			HostRoot:       "C:/Sites/My Shop/public",
		})
		for _, want := range []string{
			`root "/mnt/c/Sites/My Shop/public";`,
			`fastcgi_param SCRIPT_FILENAME "C:/Sites/My Shop/public$fastcgi_script_name";`,
			`fastcgi_param DOCUMENT_ROOT "C:/Sites/My Shop/public";`,
		} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: want %q in:\n%s", name, want, out)
			}
		}
		if i, j := strings.Index(out, "include fastcgi_params;"), strings.Index(out, "fastcgi_param DOCUMENT_ROOT"); j < i {
			t.Errorf("%s: DOCUMENT_ROOT must follow fastcgi_params, which sends its own", name)
		}
	}
}

func TestVhostWithoutHostRootKeepsRealpathRoot(t *testing.T) {
	out := renderVhostForTest(t, "vhost.conf.tmpl", VhostData{
		Domain:       "shop.test",
		Path:         "/home/me/shop",
		PublicDir:    "public",
		FPMContainer: "lerd-php84-fpm",
	})
	if !strings.Contains(out, "fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;") {
		t.Errorf("SCRIPT_FILENAME changed for a containerised site:\n%s", out)
	}
	if strings.Contains(out, "fastcgi_param DOCUMENT_ROOT") {
		t.Errorf("DOCUMENT_ROOT overridden with no HostRoot:\n%s", out)
	}
}
