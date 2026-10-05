package release

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAPKManifest_RealProductionAndStaging **يُقرأ البيانُ من الملفّات
// الحقيقيّة المبنيّة للمالك** — وتُتخطّى إن لم تكن على الجهاز.
func TestAPKManifest_RealProductionAndStaging(t *testing.T) {
	cases := []struct {
		path, pkg, name string
		code            int64
	}{
		{"D:/apk/production-2026-10-06/rahalgo-drivers-0.4.1-v7.apk", "com.rahalgo.driver", "0.4.1", 7},
		{"D:/apk/production-2026-10-06/rahalgo-reps-0.4.1-v9.apk", "com.rahalgo.rep", "0.4.1", 9},
		{"D:/apk/production-2026-10-06/rahalgo-stores-0.5.1-v8.apk", "com.rahalgo.merchant", "0.5.1", 8},
	}
	ran := 0
	for _, c := range cases {
		m, ok := readFile(t, c.path)
		if !ok {
			continue
		}
		ran++
		t.Logf("%s ⇒ %+v", filepath.Base(c.path), m)
		if m.Package != c.pkg || m.VersionCode != c.code || m.VersionName != c.name {
			t.Errorf("%s ⇒ %+v، يُنتظر %s · %d · %s", filepath.Base(c.path), m, c.pkg, c.code, c.name)
		}
	}
	// **وملفّاتُ التجهيز**: الحزمةُ تنتهي بـ`.staging` ورقمُها موجب.
	st, _ := filepath.Glob("D:/apk/staging-2026-10-06/*.apk")
	for _, p := range st {
		m, ok := readFile(t, p)
		if !ok {
			continue
		}
		ran++
		t.Logf("%s ⇒ %+v", filepath.Base(p), m)
		if !strings.HasSuffix(m.Package, ".staging") || m.VersionCode <= 0 || m.VersionName == "" {
			t.Errorf("%s ⇒ %+v", filepath.Base(p), m)
		}
	}
	// **والزبونُ المرفوعُ من Play Console** — اسمُه عربيّ.
	if m, ok := readFile(t, "D:/apk/production-2026-10-06/رحال غو.apk"); ok {
		ran++
		t.Logf("customer ⇒ %+v", m)
		if m.Package != "com.rahalgo.customer" || m.VersionCode <= 0 || m.VersionName == "" {
			t.Errorf("customer ⇒ %+v", m)
		}
	}
	if ran == 0 {
		t.Skip("لا ملفّاتِ APK على هذا الجهاز")
	}
}

func readFile(t *testing.T, p string) (Manifest, bool) {
	t.Helper()
	f, err := os.Open(p)
	if err != nil {
		return Manifest{}, false
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		return Manifest{}, false
	}
	m, err := ReadAPKManifest(f, fi.Size())
	if err != nil {
		t.Errorf("%s: %v", p, err)
		return Manifest{}, false
	}
	return m, true
}

// TestAPKManifest_RejectsNonAPK **أرشيفٌ بلا بيانٍ أو ملفٌّ ليس أرشيفاً يُردّ.**
func TestAPKManifest_RejectsNonAPK(t *testing.T) {
	for name, body := range map[string][]byte{
		"empty":       {},
		"not-zip":     []byte("PK\x03\x04garbage"),
		"no-manifest": zipOf(t, map[string][]byte{"classes.dex": []byte("x")}),
		"text-manifest": zipOf(t, map[string][]byte{
			"AndroidManifest.xml": []byte(`<manifest package="com.rahalgo.driver"/>`)}),
		"truncated": zipOf(t, map[string][]byte{
			"AndroidManifest.xml": {3, 0, 8, 0, 200, 0, 0, 0, 1, 0}}),
	} {
		if _, err := ReadAPKManifest(bytes.NewReader(body), int64(len(body))); err == nil {
			t.Errorf("%s: قُبل", name)
		}
	}
}

// TestAPKManifest_Synthetic **بيانٌ مصنوعٌ بالصيغتين (UTF-8 وUTF-16) وبأسماءٍ
// مجرَّدةٍ يُطابَق فيها بمعرّف المورد.**
func TestAPKManifest_Synthetic(t *testing.T) {
	for _, u8 := range []bool{true, false} {
		for _, stripped := range []bool{false, true} {
			b := BuildTestAXML(u8, stripped, "com.rahalgo.rep", 12, "0.9.0")
			m, err := ParseAXMLManifest(b)
			if err != nil {
				t.Fatalf("utf8=%v stripped=%v: %v", u8, stripped, err)
			}
			if m.Package != "com.rahalgo.rep" || m.VersionCode != 12 || m.VersionName != "0.9.0" {
				t.Errorf("utf8=%v stripped=%v ⇒ %+v", u8, stripped, m)
			}
			// **وكلُّ قطعٍ في البيان يُردّ ولا يُذعر.**
			for i := 0; i < len(b); i++ {
				_, _ = ParseAXMLManifest(b[:i])
			}
		}
	}
}

func TestExpectedPackage(t *testing.T) {
	a, _ := Find("driver")
	if ExpectedPackage(a, false) != "com.rahalgo.driver" || ExpectedPackage(a, true) != "com.rahalgo.driver.staging" {
		t.Fatal(ExpectedPackage(a, false), ExpectedPackage(a, true))
	}
}

func zipOf(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for n, b := range files {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(b)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
