package release

import (
	"context"
	"os"
	"path/filepath"
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

// TestFileNameSkipsArabicVersion **نسخةٌ كُتبت اسماً عربيّاً لا تملأ الاسمَ شُرَطاً.**
func TestFileNameSkipsArabicVersion(t *testing.T) {
	got := FileName(Public{Key: "driver", Version: "كابتن رحال غو", SHA256: "e1ec88746fa3aa"})
	if got != "rahalgo-driver-e1ec8874.apk" {
		t.Errorf("الاسم %q", got)
	}
	if got := FileName(Public{Key: "driver", Version: "0.4.0", SHA256: "e1ec88746fa3aa"}); got != "rahalgo-driver-0.4.0-e1ec8874.apk" {
		t.Errorf("الاسم %q", got)
	}
}
