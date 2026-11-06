package release

import (
	"context"
	"os"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

// fakeStore مخزنٌ في الذاكرة — بلا قاعدةٍ ولا خادم.
type fakeStore struct {
	dir     string
	vals    map[string]string
	staging bool
}

func (f fakeStore) GetString(_ context.Context, key string) string { return f.vals[key] }
func (f fakeStore) ArtifactDir() string                            { return f.dir }
func (f fakeStore) Staging() bool                                  { return f.staging }

// TestCustomerDirectOnProductionToo **رابطُ الزبون المباشرُ في الإنتاج والتجهيز معاً.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠١: «نعم رابط مباشر أيضاً».) **وكان الإنتاجُ مقفَلاً
// بالثابت** حتّى أُغلق إثباتُ التوافق ٢٠٢٦-١٠-٠٥ — انظر `CustomerDirectAllowed`.
func TestCustomerDirectOnProductionToo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "c.apk"), []byte("PK\x03\x04apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	vals := map[string]string{ApkKey("customer"): "c.apk"}
	app, _ := Find("customer")

	for _, staging := range []bool{false, true} {
		got := ResolveOne(context.Background(), fakeStore{dir: dir, vals: vals, staging: staging}, app)
		if got.DownloadURL == "" || got.Status != StatusDirect {
			t.Errorf("تجهيز=%v: رابطُ الزبون المباشرُ لم يظهر — %+v", staging, got)
		}
	}
}

// TestFileNameClean **اسمٌ نظيفٌ بلا بصمة، وعربيٌّ في filename*.**
func TestFileNameClean(t *testing.T) {
	sha := "e1ec88746fa3aa"
	cases := []struct{ key, ver, ascii, ar string }{
		{"driver", "0.9.0", "RahalGo-Captain-0.9.0.apk", "كابتن رحال غو 0.9.0.apk"},
		{"merchant", "0.13.0", "RahalGo-Store-0.13.0.apk", "متجر رحال غو 0.13.0.apk"},
		{"rep", "0.10.0", "RahalGo-Agent-0.10.0.apk", "وكيل رحال غو 0.10.0.apk"},
		{"customer", "1.10.0", "RahalGo-1.10.0.apk", "رحال غو 1.10.0.apk"},
		{"driver", "", "RahalGo-Captain.apk", "كابتن رحال غو.apk"},
		{"driver", "كابتن رحال غو", "RahalGo-Captain.apk", "كابتن رحال غو.apk"},
		{"driver", "1/../x", "RahalGo-Captain.apk", "كابتن رحال غو.apk"},
	}
	for _, c := range cases {
		p := Public{Key: c.key, Version: c.ver, SHA256: sha}
		if got := FileName(p); got != c.ascii {
			t.Errorf("لاتيني %q بدل %q", got, c.ascii)
		}
		if got := DisplayFileName(p); got != c.ar {
			t.Errorf("عربي %q بدل %q", got, c.ar)
		}
		cd := ContentDisposition(p)
		want := `attachment; filename="` + c.ascii + `"; filename*=UTF-8''` + url.PathEscape(c.ar)
		if cd != want || strings.Contains(cd, sha[:8]) {
			t.Errorf("ترويسة %q", cd)
		}
	}
}
