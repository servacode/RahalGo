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

// TestCustomerDirectOnlyOnStaging **رابطُ الزبون المباشرُ على التجهيز وحدَه.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠١: «نعم رابط مباشر أيضاً».) **والإنتاجُ يبقى
// مقفَلاً بالثابت** حتّى يُغلَق إثباتُ توافق المتجر والمباشر.
func TestCustomerDirectOnlyOnStaging(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "c.apk"), []byte("PK\x03\x04apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	vals := map[string]string{ApkKey("customer"): "c.apk"}
	app, _ := Find("customer")

	prod := ResolveOne(context.Background(), fakeStore{dir: dir, vals: vals}, app)
	if prod.DownloadURL != "" || prod.Status != StatusUnavailable {
		t.Errorf("الإنتاج: رابطُ الزبون المباشرُ ظهر — %+v", prod)
	}

	stg := ResolveOne(context.Background(), fakeStore{dir: dir, vals: vals, staging: true}, app)
	if stg.DownloadURL == "" || stg.Status != StatusDirect {
		t.Errorf("التجهيز: رابطُ الزبون المباشرُ لم يظهر — %+v", stg)
	}
}
