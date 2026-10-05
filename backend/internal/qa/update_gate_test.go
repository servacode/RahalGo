package qa

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/release"
)

// ══════════════════════════════════════════════════════════════════════
// **التحديثُ تلقائيٌّ من الملفّ المرفوع** (شكوى المالك ٢٠٢٦-١٠-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// «رفعتُ التطبيقات الجديدة وحدّدتُ الإصدارات وما طلب منّي جوّالي التحديث»
// — **لأنّ الهاتفَ بلا دخولٍ لا ينادي إلّا `‎/public/*`، وكانت كلُّها
// مفتوحةً للقديم.** **و«بدون ما ظلّ أبدّل إصدارات وأرقام».**

// updHead ترويستا تطبيقِ زبونٍ بنسخةٍ بعينها — كما يرسلهما أندرويد.
func updHead(kind, ver string) map[string]string {
	return map[string]string{"X-RahalGo-Client": "android-" + kind, "X-RahalGo-Version": ver}
}

// updIsolate **يحفظ مفاتيحَ التطبيق ليُرجعها** — والقاعدةُ مشتركةٌ بين الفحوص.
func updIsolate(h *Harness, app string) {
	h.Setting("app.min_version."+app, "0")
	h.Setting(release.AutoForceKey(app), "true")
	h.Setting(release.VersionCodeKey(app), "0")
	h.Setting(release.VersionKey(app), `""`)
	h.Setting(release.ApkKey(app), `""`)
	h.Setting("launch.customer_browse", "true")
}

// updPlantFile **ملفُّ حزمةٍ حقيقيُّ الصيغة على القرص** — كأنّه رُفع قبل اليوم.
func updPlantFile(t *testing.T, h *Harness, app, pkg string, code uint32, name string) {
	t.Helper()
	dir := filepath.Join(h.MediaDir, "app")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := "qa-upd-" + uniq("f") + ".apk"
	if err := os.WriteFile(filepath.Join(dir, file), release.BuildTestAPK(pkg, code, name), 0o644); err != nil {
		t.Fatal(err)
	}
	h.Setting(release.ApkKey(app), jstr(file))
}

// updSetting **قيمةُ إعدادٍ نصّاً كما في القاعدة** — أو فراغ.
func updSetting(h *Harness, sql, key string) string {
	var v *string
	_ = h.Pool.QueryRow(context.Background(), sql, key).Scan(&v)
	if v == nil {
		return ""
	}
	return *v
}

func updUpload(t *testing.T, h *Harness, tok, app string, body []byte) Res {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "upload.apk")
	_, _ = fw.Write(body)
	_ = mw.Close()
	req, _ := http.NewRequest("POST",
		h.Srv.URL+"/api/v1/admin/app-file?key="+release.ApkKey(app), &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := h.Srv.Client().Do(req)
	if err != nil {
		t.Fatalf("الرفع: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return Res{Code: resp.StatusCode, Body: b, Head: resp.Header.Clone()}
}

// TestUpdateGate_OutdatedCustomerLoggedOut **زبونٌ قديمٌ بلا دخول يُطلب منه
// التحديث** — والحدُّ من الملفّ المرفوع لا من يد.
func TestUpdateGate_OutdatedCustomerLoggedOut(t *testing.T) {
	h := New(t)
	updIsolate(h, "customer")
	updPlantFile(t, h, "customer", "com.rahalgo.customer", 18, "1.2.1")

	old := updHead("customer", "17")
	// **أبوابُ التسوّق والإقلاع تُغلق** — ومنها `‎/public/platform`: هو أوّلُ
	// نداءٍ في التطبيقات الأربعة، **وبه تظهر الشاشةُ عند أوّل إقلاع.**
	for _, p := range []string{
		"/api/v1/public/home", "/api/v1/public/sections", "/api/v1/public/search?q=x",
		"/api/v1/public/offers", "/api/v1/public/platform", "/api/v1/public/cities",
	} {
		r := h.Call("GET", p, "", nil, old)
		if r.Code != http.StatusUpgradeRequired || r.Err() != "update_required" {
			t.Errorf("**%s لقديمٍ ردّ %d لا ٤٢٦**: %s", p, r.Code, r)
		}
	}
	// **وأبوابُ النجاة تبقى** — ما يحتاجه من يحدّث.
	for _, p := range []string{"/api/v1/public/releases", "/api/v1/public/identity", "/healthz"} {
		if r := h.Call("GET", p, "", nil, old); r.Code == http.StatusUpgradeRequired {
			t.Errorf("**بابُ النجاة %s أُغلق**: %s", p, r)
		}
	}
	if r := h.Call("GET", "/api/v1/public/app/customer", "", nil, old); r.Code != http.StatusOK {
		t.Errorf("**تنزيلُ الملفّ أُغلق على القديم**: %d", r.Code)
	}

	// **والمحدَّثُ يمرّ**، **والويبُ (بلا ترويسة) لا يمسّه شيء.**
	for _, head := range []map[string]string{updHead("customer", "18"), updHead("customer", "19"), nil} {
		for _, p := range []string{"/api/v1/public/home", "/api/v1/public/platform"} {
			if r := h.Call("GET", p, "", nil, head); r.Code != http.StatusOK {
				t.Errorf("**%s لـ%v ردّ %d**: %s", p, head, r.Code, r)
			}
		}
	}
	// **والرقمُ قُرئ من الملفّ القديم وكُتب.**
	if got := updSetting(h, `SELECT value::text FROM app_settings WHERE key = $1`,
		release.VersionCodeKey("customer")); got != "18" {
		t.Errorf("**رقمُ الملفّ القديم لم يُكتب**: %q", got)
	}
}

// TestUpdateGate_AutoForceOffFallsBackToManual **إطفاءُ الفرض يُعيد الرقمَ اليدويَّ وحدَه.**
func TestUpdateGate_AutoForceOffFallsBackToManual(t *testing.T) {
	h := New(t)
	updIsolate(h, "driver")
	updPlantFile(t, h, "driver", "com.rahalgo.driver", 7, "0.4.1")
	h.Setting(release.VersionCodeKey("driver"), "7")
	h.Setting(release.AutoForceKey("driver"), "false")

	v6 := updHead("driver", "6")
	if r := h.Call("GET", "/api/v1/public/platform", "", nil, v6); r.Code != http.StatusOK {
		t.Fatalf("**الفرضُ مطفأٌ والقديمُ رُدّ**: %s", r)
	}
	h.Setting("app.min_version.driver", "7")
	if r := h.Call("GET", "/api/v1/public/platform", "", nil, v6); r.Code != http.StatusUpgradeRequired {
		t.Fatalf("**الرقمُ اليدويُّ لم يُحترَم**: %s", r)
	}
	// **واليدويُّ حدٌّ إضافيٌّ فوق رقم الملفّ.**
	h.Setting(release.AutoForceKey("driver"), "true")
	h.Setting("app.min_version.driver", "9")
	if r := h.Call("GET", "/api/v1/public/platform", "", nil, updHead("driver", "8")); r.Code != http.StatusUpgradeRequired {
		t.Fatalf("**اليدويُّ الأعلى لم يُحترَم**: %s", r)
	}
	// **ولا رقمَ بلا ملفّ** — حُذف الملفُّ فلا يُحبس أحدٌ على ما لا يُنزَّل.
	h.Setting("app.min_version.driver", "0")
	h.Setting(release.ApkKey("driver"), `""`)
	if r := h.Call("GET", "/api/v1/public/platform", "", nil, v6); r.Code != http.StatusOK {
		t.Fatalf("**رقمٌ بلا ملفٍّ حبس القديم**: %s", r)
	}
}

// TestUpdateGate_UploadSetsVersionAndRejectsWrongPackage **الرفعُ يكتب الرقمَ
// والاسمَ من الملفّ، ويردّ حزمةً لا تخصّ الخانة.**
func TestUpdateGate_UploadSetsVersionAndRejectsWrongPackage(t *testing.T) {
	h := New(t)
	updIsolate(h, "rep")
	h.Setting(release.VersionKey("rep"), jstr("كابتن رحّال غو"))
	admin := h.NewUser("admin")

	for _, pkg := range []string{"com.rahalgo.driver", "com.rahalgo.rep.staging"} {
		r := updUpload(t, h, admin.Token, "rep", release.BuildTestAPK(pkg, 50, "9.0.0"))
		if r.Code != http.StatusBadRequest || r.Err() != "app_wrong_package" {
			t.Errorf("**%s قُبل في خانة المندوب**: %s", pkg, r)
		}
	}
	if got := updSetting(h, `SELECT value::text FROM app_settings WHERE key = $1`, release.ApkKey("rep")); got != `""` {
		t.Fatalf("**رفعٌ مردودٌ كتب ملفّاً**: %s", got)
	}

	r := updUpload(t, h, admin.Token, "rep", release.BuildTestAPK("com.rahalgo.rep", 10, "0.5.0"))
	if r.Code != http.StatusCreated {
		t.Fatalf("**الرفعُ الصحيحُ رُدّ**: %s", r)
	}
	t.Cleanup(func() {
		if name := updSetting(h, `SELECT value #>> '{}' FROM app_settings WHERE key = $1`, release.ApkKey("rep")); name != "" {
			_ = os.Remove(filepath.Join(h.MediaDir, "app", filepath.Base(name)))
		}
	})
	if got := updSetting(h, `SELECT value::text FROM app_settings WHERE key = $1`, release.VersionCodeKey("rep")); got != "10" {
		t.Errorf("**version_code=%s لا 10**", got)
	}
	if got := updSetting(h, `SELECT value #>> '{}' FROM app_settings WHERE key = $1`, release.VersionKey("rep")); got != "0.5.0" {
		t.Errorf("**الاسمُ اليدويُّ بقي**: %q", got)
	}
	// **والرفعُ وحدَه يُجبر** — بلا رقمٍ يُكتب بيد.
	if r := h.Call("GET", "/api/v1/public/platform", "", nil, updHead("rep", "9")); r.Code != http.StatusUpgradeRequired {
		t.Errorf("**نسخةُ ٩ لم يُطلب تحديثُها بعد رفع ١٠**: %s", r)
	}
	if r := h.Call("GET", "/api/v1/public/platform", "", nil, updHead("rep", "10")); r.Code != http.StatusOK {
		t.Errorf("**النسخةُ المرفوعةُ نفسُها رُدّت**: %s", r)
	}
	// **ورقمُ النسخة واسمُها لا يُحرَّران من اللوحة.**
	for _, k := range []string{release.VersionKey("rep"), release.VersionCodeKey("rep")} {
		var v any = "1.0"
		if k == release.VersionCodeKey("rep") {
			v = 99
		}
		if r := h.Call("PUT", "/api/v1/admin/settings/"+k, admin.Token, map[string]any{"value": v}, nil); r.Err() != "setting_read_only" {
			t.Errorf("**%s حُرِّر من اللوحة**: %s", k, r)
		}
	}
}
