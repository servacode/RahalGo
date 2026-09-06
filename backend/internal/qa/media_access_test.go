package qa

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **`D13` — سردُ `/media/` مكشوفٌ وفيه إثباتُ التسليم**
// ══════════════════════════════════════════════════════════════════════
//
// # ما هو مقيسٌ اليوم
//
// **`http.FileServer(http.Dir(...))` بلا حارس**: **يسرد المجلَّداتِ**
// حين لا `index.html`، **ويخدم كلَّ ملفٍّ لأيّ أحدٍ بلا توكن.**
//
// # وأصنافُ الوسائط تسعةٌ في القاعدة
//
// **سبعةٌ عامّةٌ بطبيعتها**: شعارُ المتجر · الصنفُ · القسمُ · اللافتةُ ·
// شعارُ المنصّة · خلفيّةُ الدخول · خلفيّةُ الموقع — **وخلفيّةُ الدخول
// تُعرَض قبل أن يملك أحدٌ توكناً أصلاً.**
//
// **واثنان شخصيّان**: `avatar` و`delivery_proof` — **وإثباتُ التسليم
// صورةُ بابِ بيتٍ ومعه إحداثيّاتُه ووقتُه.**
//
// # ولا تُحجَب بالترويسة
//
// **المصادقةُ ترويسةُ `Bearer` وحدَها** — **ووسمُ `<img>` لا يحمل
// ترويسة.** **فمن حجب الوسائطَ بالتوكن كسر عرضَ الصور في الويب
// وأندرويد معاً.**
//
// **فالتوقيعُ**: رابطٌ موقَّعٌ محدودُ الأجل يُصدره الخادمُ مع الردّ
// المُصرَّح به أصلاً — **والسرُّ في الخادم لا في العميل.**

// ══════════════════════════════════════════════════════════════════════
// **١ · لا يُسرَد مجلَّد**
// ══════════════════════════════════════════════════════════════════════
func TestD13_MediaDirectoryIsNotListable(t *testing.T) {
	h := New(t)
	for _, p := range []string{"/media/", "/media/2026/", "/media/2026/09/"} {
		got := h.GET(p, "")
		body := string(got.Body)
		t.Logf("%-20s ⇒ %d · %d بايت", p, got.Code, len(body))
		if got.Code == 200 && strings.Contains(body, "<pre>") {
			t.Errorf("**`%s` يسرد محتواه** — **ومن سرد عرف الأسماءَ فجلبها.** "+
				"(`D13`)", p)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · إثباتُ التسليم لا يُجلَب برابطٍ عارٍ**
// ══════════════════════════════════════════════════════════════════════
//
// **ويُرفَع إثباتٌ حقيقيٌّ ثمّ يُطلَب بلا توقيع.**
func TestD13_DeliveryProofNeedsSignedURL(t *testing.T) {
	h := New(t)
	path := uploadKind(t, h, "delivery_proof")
	bare := h.GET("/media/"+path, "")
	t.Logf("إثباتُ تسليمٍ برابطٍ عارٍ ⇒ %d · %d بايت", bare.Code, len(bare.Body))
	if bare.Code == 200 {
		t.Errorf("**إثباتُ التسليم يُجلَب بلا توقيع** — **صورةُ بابِ بيتٍ " +
			"لمن خمّن المسار.** (`D13`)")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · صورةُ الحساب كذلك**
// ══════════════════════════════════════════════════════════════════════
func TestD13_AvatarNeedsSignedURL(t *testing.T) {
	h := New(t)
	path := uploadKind(t, h, "avatar")
	bare := h.GET("/media/"+path, "")
	t.Logf("صورةُ حسابٍ برابطٍ عارٍ ⇒ %d", bare.Code)
	if bare.Code == 200 {
		t.Errorf("**صورةُ الحساب تُجلَب بلا توقيع** (`D13`)")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · والعامُّ يبقى عامّاً**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُحجَب الوسائطُ كلُّها جزافاً**: **خلفيّةُ الدخول تُعرَض قبل
// أن يملك أحدٌ توكناً** — **فحجبُها يقفل بابَ الدخول نفسَه.**
func TestD13_PublicMediaStaysPublic(t *testing.T) {
	h := New(t)
	for _, kind := range []string{"auth_background", "merchant_logo", "banner"} {
		path := uploadKind(t, h, kind)
		got := h.GET("/media/"+path, "")
		t.Logf("%-16s برابطٍ عارٍ ⇒ %d", kind, got.Code)
		if got.Code != 200 {
			t.Errorf("**`%s` صار محجوباً** (%d) — **وهو عامٌّ بطبيعته، "+
				"وحجبُه يكسر عرضاً مشروعاً.**", kind, got.Code)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · والموقَّعُ يُجلَب، والتوقيعُ المزوَّرُ أو المنتهي لا**
// ══════════════════════════════════════════════════════════════════════
func TestD13_SignedURLWorksAndForgeryDoesNot(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	path := uploadKind(t, h, "delivery_proof")

	// **والرابطُ الموقَّعُ يأتي من الردّ المُصرَّح به** — يُقرأ من نقطةٍ
	// تُصدره. **ولا يُصنَع في الاختبار**: **من صنعه بيده أثبت صنعَه.**
	signed := signedMediaURL(t, h, admin, path)
	if signed == "" {
		t.Skip("لا نقطةَ تُصدر رابطاً موقَّعاً بعد")
	}
	ok := h.GET(signed, "")
	t.Logf("موقَّعٌ ⇒ %d", ok.Code)
	if ok.Code != 200 {
		t.Errorf("**الرابطُ الموقَّعُ لا يُجلَب** (%d) — **وحمايةٌ تمنع "+
			"صاحبَ الحقّ ليست حماية.**", ok.Code)
	}

	forged := signed[:len(signed)-4] + "dead"
	bad := h.GET(forged, "")
	t.Logf("مزوَّرٌ ⇒ %d", bad.Code)
	if bad.Code == 200 {
		t.Error("**توقيعٌ مزوَّرٌ قُبل** — ولا توقيعَ إذاً")
	}
}

// uploadKind يكتب ملفَّ وسيطٍ حقيقيّاً بصنفٍ معلوم ويُرجع مسارَه النسبيّ.
//
// **ولا يُرفَع عبر النقطة**: **الرفعُ يشترط دوراً ومعالجةَ صورة**،
// **والمقصودُ هنا الإذنُ عند الجلب لا صحّةُ الرفع.**
func uploadKind(t *testing.T, h *Harness, kind string) string {
	t.Helper()
	rel := "2026/09/" + uniq("m") + ".jpg"
	full := filepath.Join(h.MediaDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("مجلَّدُ الوسيط: %v", err)
	}
	if err := os.WriteFile(full, []byte("JPEGDATA"), 0o644); err != nil {
		t.Fatalf("كتابةُ الوسيط: %v", err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO media (kind, path, thumb_path, width, height, bytes)
		VALUES ($1, $2, $3, 10, 10, 8)`, kind, rel, rel); err != nil {
		t.Fatalf("صفُّ الوسيط: %v", err)
	}
	return rel
}

// signedMediaURL يقرأ رابطاً موقَّعاً من ردٍّ مُصرَّحٍ به.
//
// **ولا يُصنَع التوقيعُ في الفحص** — **من صنعه بيده أثبت صنعَه لا
// أثبت الخادم.**
func signedMediaURL(t *testing.T, h *Harness, admin *User, path string) string {
	t.Helper()
	got := h.GET("/api/v1/admin/media/sign?path="+path, admin.Token)
	if got.Code != 200 {
		return ""
	}
	u, _ := got.JSON()["url"].(string)
	return u
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · وتوقيعٌ انتهى أجلُه أو عُبث بأجله لا يُقبَل**
// ══════════════════════════════════════════════════════════════════════
func TestD13_ExpiredAndTamperedSignatures(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	path := uploadKind(t, h, "delivery_proof")
	signed := signedMediaURL(t, h, admin, path)
	if signed == "" {
		t.Fatal("لا رابطَ موقَّع")
	}
	u, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("تحليلُ الرابط: %v", err)
	}
	q := u.Query()
	sig, exp := q.Get("sig"), q.Get("exp")

	// **أجلٌ مضى بتوقيعه الأصليّ.**
	past := "/media/" + path + "?exp=1&sig=" + sig
	// **أجلٌ مُطاوَلٌ بتوقيعٍ قديم** — **ومن قبله جعل الأجلَ زينة.**
	far := "/media/" + path + "?exp=99999999999&sig=" + sig
	// **مسارٌ بُدّل بتوقيع غيرِه.**
	other := uploadKind(t, h, "delivery_proof")
	swapped := "/media/" + other + "?exp=" + exp + "&sig=" + sig

	for name, u := range map[string]string{
		"أجلٌ مضى": past, "أجلٌ مُطاوَل": far, "مسارٌ مُبدَّل": swapped,
	} {
		got := h.GET(u, "")
		t.Logf("%-14s ⇒ %d", name, got.Code)
		if got.Code == 200 {
			t.Errorf("**%s قُبل** — والتوقيعُ لا يحرس", name)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٧ · واجتيازُ المسار مرفوضٌ نصّاً ومرمَّزاً**
// ══════════════════════════════════════════════════════════════════════
func TestD13_TraversalIsRefused(t *testing.T) {
	h := New(t)
	for _, p := range []string{
		"/media/../go.mod",
		"/media/..%2fgo.mod",
		"/media/%2e%2e/go.mod",
		"/media/2026/../../go.mod",
	} {
		got := h.GET(p, "")
		t.Logf("%-26s ⇒ %d", p, got.Code)
		if got.Code == 200 {
			t.Errorf("**`%s` خرج من المجلَّد** — واجتيازُ المسار مفتوح", p)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٨ · ومجهولُ النسب يُحجَب**
// ══════════════════════════════════════════════════════════════════════
//
// **ملفٌّ في المجلَّد بلا صفٍّ في الجدول** — **لا يُعرَف أعامٌّ هو أم
// شخصيّ**، **ومن خدمه بحجّة الجهل خدم كلَّ ما تسرّب إلى المجلَّد.**
func TestD13_UnknownMediaDefaultsToProtected(t *testing.T) {
	h := New(t)
	rel := "2026/09/" + uniq("orphan") + ".jpg"
	full := filepath.Join(h.MediaDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("مجلَّد: %v", err)
	}
	if err := os.WriteFile(full, []byte("JPEGDATA"), 0o644); err != nil {
		t.Fatalf("كتابة: %v", err)
	}
	got := h.GET("/media/"+rel, "")
	t.Logf("ملفٌّ بلا صفٍّ ⇒ %d", got.Code)
	if got.Code == 200 {
		t.Error("**ملفٌّ مجهولُ النسب خُدم** — **والافتراضُ يجب أن يكون الحجب**")
	}
}
