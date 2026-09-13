package qa

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/release"
)

// ══════════════════════════════════════════════════════════════════════
// **مركزُ التنزيل الرسميّ — ولا يَعِد بما لا يملك** (`DLC`، ٢٠٢٦-٠٩-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// # القاعدةُ التي تُحرَس
//
// **لا أثرَ معتمَدٌ = لا زرَّ تنزيل.** **والغيابُ يُغلق ولا يُخمّن.**
//
// # وما وُجد عند الجرد
//
// **آليّةٌ لتطبيقٍ واحد** (`platform.app_file` ومسارُ `‎/public/app`) —
// **ومفتاحٌ واحدٌ لأربعةِ ملفّاتٍ يعني أنّ رفعَ تطبيقِ السائق يمحو
// تطبيقَ الزبون.**
//
// **وصفحةُ `/app` تعرض زرَّ متجرِ جوجل برابطٍ مكتوبٍ في شيفرتها**
// **و`platform.app_url` فارغٌ في الإنتاج والتجهيز** — **فوعدٌ بمتجرٍ
// لا يُعرَف أنّه فُتح.**

// dlcGet نداءٌ عامٌّ بلا توثيقٍ — **ولا حسابَ يُشترَط لتنزيل تطبيق.**
func dlcGet(t *testing.T, h *Harness, path string) Res {
	t.Helper()
	return h.GET(path, "")
}

// dlcApps يقرأ سجلَّ التوزيع كما يُرسَل للناس.
func dlcApps(t *testing.T, h *Harness) map[string]map[string]any {
	t.Helper()
	got := dlcGet(t, h, "/api/v1/public/releases")
	if got.Code != http.StatusOK {
		t.Fatalf("سجلُّ التوزيع لا يُقرأ: %s", got)
	}
	var body struct {
		Data struct {
			Apps []map[string]any `json:"apps"`
		} `json:"data"`
	}
	if err := json.Unmarshal(got.Body, &body); err != nil {
		t.Fatalf("جسمُ السجلّ: %v — %s", err, got)
	}
	out := map[string]map[string]any{}
	for _, a := range body.Data.Apps {
		key, _ := a["key"].(string)
		out[key] = a
	}
	return out
}

// ══════════════════════════════════════════════════════════════════════
// **DLC1 · الأربعةُ معلَنةٌ وحالُها مغلقٌ افتراضاً**
// ══════════════════════════════════════════════════════════════════════
func TestDLC1_AllFourDeclaredAndClosedByDefault(t *testing.T) {
	h := New(t)
	apps := dlcApps(t, h)

	if len(apps) != 4 {
		t.Fatalf("DLC1 التطبيقاتُ %d لا ٤: %v", len(apps), apps)
	}
	for _, want := range []string{"customer", "driver", "merchant", "rep"} {
		a, ok := apps[want]
		if !ok {
			t.Fatalf("DLC1 **%s غائبٌ من السجلّ**", want)
		}
		t.Logf("DLC1 %-9s حالٌ=%v · متجرٌ=%q · تنزيلٌ=%q",
			want, a["status"], a["play_url"], a["download_url"])
		if a["status"] != string(release.StatusUnavailable) {
			t.Errorf("DLC1 **%s حالُه %v ولا إعدادَ له** — "+
				"**والغيابُ يُغلق**", want, a["status"])
		}
		if a["download_url"] != "" || a["play_url"] != "" {
			t.Errorf("DLC1 **%s يَعِد برابطٍ بلا إعداد**: %v · %v",
				want, a["play_url"], a["download_url"])
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **DLC2 · ومفتاحٌ مجهولٌ لا يُحلّ إلى ملفّ**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو حدُّ الأمان**: **لا اسمَ ملفٍّ يأتي من الطلب** — **والمفتاحُ
// من قائمةٍ مغلقةٍ من أربعة.**
func TestDLC2_UnknownKeyAndTraversalRefused(t *testing.T) {
	h := New(t)
	for _, k := range []string{
		"ghost", "admin", "..", "%2e%2e", "..%2f..%2fetc%2fpasswd",
		"customer.apk", "driver/../../etc/passwd",
	} {
		got := dlcGet(t, h, "/api/v1/public/app/"+k)
		t.Logf("DLC2 %-28q ⇒ %d", k, got.Code)
		if got.Code == http.StatusOK {
			t.Errorf("DLC2 **مفتاحٌ مجهولٌ خُدم**: %q", k)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **DLC3 · أثرٌ معتمَدٌ ⇒ زرٌّ يصفه بعينه**
// ══════════════════════════════════════════════════════════════════════
//
// **ويُزرَع أثرٌ حقيقيٌّ على القرص** — **ويُقاس أنّ الحجمَ والبصمةَ
// المعروضين هما حجمُ الملفّ وبصمتُه لا رقمان في إعداد.**
func TestDLC3_RegisteredArtifactIsDescribedExactly(t *testing.T) {
	h := New(t)
	body, sum := dlcPlant(t, h, "driver", "1.2.3")

	apps := dlcApps(t, h)
	d := apps["driver"]
	t.Logf("DLC3 حالٌ=%v · حجمٌ=%v · بصمةٌ=%v… · نسخةٌ=%v",
		d["status"], d["size_bytes"], firstN(d["sha256"], 12), d["version"])

	if d["status"] != string(release.StatusDirect) {
		t.Errorf("DLC3 **الحالُ %v والأثرُ مزروع**", d["status"])
	}
	if int64(d["size_bytes"].(float64)) != int64(len(body)) {
		t.Errorf("DLC3 **الحجمُ المعروضُ %v والملفُّ %d**",
			d["size_bytes"], len(body))
	}
	if d["sha256"] != sum {
		t.Errorf("DLC3 **البصمةُ المعروضةُ لا تصف الملفَّ**")
	}
	if d["version"] != "1.2.3" {
		t.Errorf("DLC3 النسخةُ %v — والمكتوبةُ 1.2.3", d["version"])
	}
	if d["download_url"] != "/api/v1/public/app/driver" {
		t.Errorf("DLC3 مسارُ التنزيل %v", d["download_url"])
	}

	// **والملفُّ يُخدَم فعلاً — بلا توثيقٍ وباسمٍ يقول ما هو.**
	got := dlcGet(t, h, "/api/v1/public/app/driver")
	if got.Code != http.StatusOK {
		t.Fatalf("DLC3 **الأثرُ المعتمَدُ لا يُخدَم**: %s", got)
	}
	if ct := got.Head.Get("Content-Type"); ct != "application/vnd.android.package-archive" {
		t.Errorf("DLC3 نوعُ المحتوى %q", ct)
	}
	if cd := got.Head.Get("Content-Disposition"); !strings.Contains(cd, "rahalgo-driver-1.2.3.apk") {
		t.Errorf("DLC3 **اسمُ التنزيل لا يحمل هويّتَه**: %q", cd)
	}
	if string(got.Body) != string(body) {
		t.Errorf("DLC3 **المخدومُ ليس الملفَّ المزروع**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **DLC4 · وإعدادٌ يشير إلى ملفٍّ غيرِ موجودٍ يُغلق**
// ══════════════════════════════════════════════════════════════════════
//
// **ومن اكتفى بوجود قيمةٍ في الإعداد أظهر زرّاً يعطي ٤٠٤** —
// **فيُسأل القرصُ نفسُه.**
func TestDLC4_MissingFileClosesTheDoor(t *testing.T) {
	h := New(t)
	h.Setting(release.ApkKey("rep"), jstr("ghost-artifact.apk"))

	apps := dlcApps(t, h)
	r := apps["rep"]
	t.Logf("DLC4 حالٌ=%v · تنزيلٌ=%q", r["status"], r["download_url"])
	if r["status"] != string(release.StatusUnavailable) || r["download_url"] != "" {
		t.Errorf("DLC4 **إعدادٌ بلا ملفٍّ فتح زرّاً**: %v", r)
	}
	if got := dlcGet(t, h, "/api/v1/public/app/rep"); got.Code == http.StatusOK {
		t.Errorf("DLC4 **خُدم ملفٌّ غيرُ موجود**: %s", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **DLC5 · تنزيلُ الزبون المباشرُ مقفَلٌ ولو زُرع الأثر**
// ══════════════════════════════════════════════════════════════════════
//
// **وقرارُ المالك ٢٠٢٦-٠٩-١٣**: **لا ملفَّ زبونٍ مباشرٌ قبل إثبات
// التوافق** (هويّةُ الحزمة · التوقيعُ · التحديثُ في الاتّجاهين).
//
// **والقفلُ في الحساب لا في الشاشة** — **ومن أقفله في الشاشة وحدَها
// تركه مكشوفاً بمسارٍ مباشر.**
func TestDLC5_CustomerDirectIsLockedByPolicy(t *testing.T) {
	h := New(t)
	dlcPlant(t, h, "customer", "1.0.6")

	apps := dlcApps(t, h)
	c := apps["customer"]
	t.Logf("DLC5 حالٌ=%v · تنزيلٌ=%q · حجمٌ=%v",
		c["status"], c["download_url"], c["size_bytes"])

	if c["download_url"] != "" {
		t.Errorf("DLC5 **رابطُ تنزيلٍ مباشرٍ للزبون ظهر** — " +
			"**والإثباتُ لم يُغلَق بعد**")
	}
	if got := dlcGet(t, h, "/api/v1/public/app/customer"); got.Code == http.StatusOK {
		t.Errorf("DLC5 **خُدم ملفُّ الزبون من المسار المباشر**: %d", got.Code)
	}
	if release.CustomerDirectAllowed {
		t.Error("DLC5 **الثابتُ مفتوحٌ** — ولا إثباتَ توافقٍ مسجَّل")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **DLC6 · ورابطُ المتجر لا يُقبل إلّا متجراً**
// ══════════════════════════════════════════════════════════════════════
//
// **وقيمةٌ تكتبها يدٌ في إعداد تصير زرّاً في صفحةٍ عامّة** —
// **و`javascript:` أو موقعٌ غريبٌ في زرٍّ رسميٍّ خطرٌ لا خطأُ عرض.**
func TestDLC6_OnlyPlayStoreLinksAreAccepted(t *testing.T) {
	h := New(t)
	for _, bad := range []string{
		"javascript:alert(1)",
		"http://play.google.com/store/apps/details?id=com.rahalgo.customer",
		"https://evil.example/store",
		"https://play.google.com.evil.example/x",
	} {
		h.Setting(release.PlayURLKey("customer"), jstr(bad))
		c := dlcApps(t, h)["customer"]
		t.Logf("DLC6 %-64q ⇒ حالٌ=%v", bad, c["status"])
		if c["play_url"] != "" {
			t.Errorf("DLC6 **رابطٌ غيرُ متجرٍ قُبل**: %q", bad)
		}
	}
	good := "https://play.google.com/store/apps/details?id=com.rahalgo.customer"
	h.Setting(release.PlayURLKey("customer"), jstr(good))
	c := dlcApps(t, h)["customer"]
	if c["play_url"] != good || c["status"] != string(release.StatusPlay) {
		t.Errorf("DLC6 **رابطُ المتجر الصحيحُ لم يُقبل**: %v", c)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **DLC7 · ولا يُكتَب إعدادٌ غيرُ إعدادِ أثرٍ من بابِ الرفع**
// ══════════════════════════════════════════════════════════════════════
//
// **والمفتاحُ صار يأتي في الطلب** — **ومكتبُ كتابةٍ في الإعدادات بمفتاحٍ
// حرٍّ يكتب أيَّ إعدادٍ بملفّ.**
func TestDLC7_UploadKeyIsAClosedList(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	for _, k := range []string{
		"launch.customer_orders", "platform.name", "release.ghost.apk",
		"app.max_file_mb",
	} {
		got := h.POST("/api/v1/admin/app-file?key="+k, admin.Token, map[string]any{})
		t.Logf("DLC7 %-26q ⇒ %d", k, got.Code)
		if got.Code == http.StatusCreated {
			t.Errorf("DLC7 **كُتب إعدادٌ خارجَ القائمة**: %q", k)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **DLC8 · ومعرّفاتُ الحزم تُطابق الغرادلَ حرفاً**
// ══════════════════════════════════════════════════════════════════════
//
// **ومن بدّل المعرّفَ في أندرويد ولم يبدّله في السجلّ نشر صفحةً تقول
// معرّفاً غيرَ الذي في الجهاز** — **ولا يُكتشَف إلّا بشكوى.**
func TestDLC8_PackageIdsMatchGradle(t *testing.T) {
	root := repoRoot(t)
	re := regexp.MustCompile(`applicationId\s*=\s*"([^"]+)"`)
	for _, a := range release.Apps {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(a.GradlePath)))
		if err != nil {
			t.Fatalf("DLC8 لم أجد غرادلَ %s: %v", a.Key, err)
		}
		mm := re.FindStringSubmatch(string(b))
		if mm == nil {
			t.Fatalf("DLC8 **لا معرّفَ حزمةٍ في %s**", a.GradlePath)
		}
		t.Logf("DLC8 %-9s غرادل=%s · سجلّ=%s", a.Key, mm[1], a.PackageID)
		if mm[1] != a.PackageID {
			t.Errorf("DLC8 **افتراقُ هويّة**: الغرادلُ %q والسجلُّ %q",
				mm[1], a.PackageID)
		}
	}
}

// dlcPlant **يزرع أثراً حقيقيّاً على القرص ويسجّله** — ويعيد جسمَه وبصمتَه.
func dlcPlant(t *testing.T, h *Harness, key, version string) ([]byte, string) {
	t.Helper()
	dir := filepath.Join(h.MediaDir, "app")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("مجلَّدُ الآثار: %v", err)
	}
	// **وأوّلُ أربعةِ بايتاتٍ بصمةُ أرشيف** — كما يشترط بابُ الرفع.
	body := append([]byte{'P', 'K', 3, 4}, []byte("qa-"+key+"-"+uniq("a"))...)
	name := "qa-" + key + "-" + uniq("f") + ".apk"
	if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
		t.Fatalf("كتابةُ الأثر: %v", err)
	}
	h.Setting(release.ApkKey(key), jstr(name))
	if version != "" {
		h.Setting(release.VersionKey(key), jstr(version))
	}
	return body, hexSum(body)
}

// jstr **قيمةُ إعدادٍ نصّيّةٌ بصيغة `jsonb`** — والعمودُ يقبل JSON لا نصّاً.
func jstr(v string) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// hexSum بصمةُ جسمٍ كما يحسبها المحرّك.
func hexSum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// firstN أوّلُ حروفٍ من قيمةٍ — للسجلّ لا للحكم.
func firstN(v any, n int) string {
	s, _ := v.(string)
	if len(s) > n {
		return s[:n]
	}
	return s
}
