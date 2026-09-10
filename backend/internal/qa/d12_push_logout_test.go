package qa

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/androidmap"
	"github.com/servacode/rahalgo/backend/internal/server"
)

// ══════════════════════════════════════════════════════════════════════
// **`D12` — رمزُ الدفع يبقى بعد الخروج**
// ══════════════════════════════════════════════════════════════════════
//
// **`Push.unregister` مكتوبةٌ ولا منادِيَ لها.** **وبابُ الخروج يمسح
// الجلسةَ محلّيّاً ثمّ يُبطلها في المحرّك** — **ولا يمسّ وجهةَ
// الدفع.** **فيبقى الجهازُ هدفاً لحسابٍ خرج منه**، **وهاتفٌ في بيتٍ
// يُظهر أسماءَ زبائنِ سائقٍ خرج.**
//
// # وليست خصوصيّةَ عميل
//
// **ولا يُغلَق هذا بأن يتجاهل التطبيقُ ما يصله** — **المحرّكُ ما زال
// يختار الجهازَ باسم الحساب القديم**، **والرسالةُ تصل شريطَ الإشعارات
// قبل أن يراها تطبيق.**

// loginPair دخولٌ حقيقيٌّ يردّ رمزَ الوصول ورمزَ التجديد.
func loginPair(t *testing.T, h *Harness, userID string) (access, refresh string) {
	t.Helper()
	raw, _ := issuedRefreshStrict(t, h, userID, "driver")
	got := h.POST("/api/v1/auth/refresh", "", map[string]any{"refresh_token": raw})
	if got.Code >= 400 {
		t.Fatalf("تعذّر الحصولُ على جلسةٍ حقيقيّة: %s", got)
	}
	return tokenField(got, "access_token"), tokenField(got, "refresh_token")
}

// registerDevice **يُسجّل عبر الباب الحقيقيّ** — لا بإدراجٍ في القاعدة.
func registerDevice(t *testing.T, h *Harness, access, token string) {
	t.Helper()
	got := h.POST("/api/v1/me/devices", access,
		map[string]any{"token": token, "platform": "android"})
	if got.Code >= 400 {
		t.Fatalf("تسجيلُ الجهاز (%s): %s", token, got)
	}
}

// deviceOwner صاحبُ الرمز الآن — وفارغٌ يعني لا صفَّ له.
func deviceOwner(t *testing.T, h *Harness, token string) string {
	t.Helper()
	var owner string
	err := h.Pool.QueryRow(ctxBG(),
		`SELECT user_id::text FROM device_tokens WHERE token = $1`, token).Scan(&owner)
	if err != nil {
		return ""
	}
	return owner
}

// logout **الخروجُ من الباب الحقيقيّ** — ويحمل رمزَ جهازه.
func logout(t *testing.T, h *Harness, refresh, device string) int {
	t.Helper()
	body := map[string]any{"refresh_token": refresh}
	if device != "" {
		body["device_token"] = device
	}
	return h.POST("/api/v1/auth/logout", "", body).Code
}

// ══════════════════════════════════════════════════════════════════════
// **`D12-T1`+`T7` · الخروجُ يُنهي وجهةَ هذا الجهاز — والجلسةَ معها**
// ══════════════════════════════════════════════════════════════════════
func TestD12_LogoutRemovesThisDeviceBinding(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	access, refresh := loginPair(t, h, drv.ID)

	const tok = "d12-tok-one"
	registerDevice(t, h, access, tok)
	if got := deviceOwner(t, h, tok); got != drv.ID {
		t.Fatalf("**التسجيلُ لم يربط الرمزَ بصاحبه**: %q ≠ %q", got, drv.ID)
	}
	t.Logf("D12: قبل الخروج — صاحبُ الرمز=%s", tok)

	if code := logout(t, h, refresh, tok); code >= 400 {
		t.Fatalf("**الخروجُ رُدّ** (%d)", code)
	}

	if got := deviceOwner(t, h, tok); got != "" {
		t.Errorf("**رمزُ الدفع بقي مربوطاً بحسابٍ خرج** (%s) — **وذاك `D12`.**", got)
	}
	// ── `T7` · والجلسةُ أُبطلت كما كانت ───────────────────────────
	if res := h.GET(mePath, access); res.Code != 401 {
		t.Errorf("**الخروجُ لم يُبطل الجلسة** (%d)", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D12-T3`+`T10` · وجهازٌ ثانٍ للحساب نفسِه لا يُمَسّ**
// ══════════════════════════════════════════════════════════════════════
func TestD12_LogoutIsDeviceScoped(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	// **وجهازان يعنيان جلستين** — **لا رمزين في جلسةٍ واحدة**:
	// **الجلسةُ تركيبُ تطبيقٍ واحد، ورمزُها واحدٌ في اللحظة.**
	// **ورمزٌ ثانٍ من الجلسة نفسِها دورانٌ لا جهازٌ ثانٍ** —
	// **ويُنظَّف كلُّه عند الخروج** (دورةُ ٦٠).
	phone := d12Phone(t, h, drv.ID)
	accessOne, refreshOne := d12LoginAs(t, h, phone, "driver")
	accessTwo, _ := d12LoginAs(t, h, phone, "customer")

	const one, two = "d12-dev-1", "d12-dev-2"
	registerDeviceAs(t, h, accessOne, one, "android-driver")
	registerDeviceAs(t, h, accessTwo, two, "android-customer")

	if code := logout(t, h, refreshOne, one); code >= 400 {
		t.Fatalf("الخروج: %d", code)
	}

	if got := deviceOwner(t, h, one); got != "" {
		t.Errorf("**جهازُ الخروج بقي مربوطاً** (%s)", got)
	}
	if got := deviceOwner(t, h, two); got != drv.ID {
		t.Errorf("**خروجُ جهازٍ أسقط جهازاً آخرَ للحساب نفسِه**: %q — "+
			"**والنطاقُ جهازٌ لا حساب.**", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D12-T4` · وخروجٌ مكرَّرٌ لا يُغيّر شيئاً**
// ══════════════════════════════════════════════════════════════════════
func TestD12_LogoutIsIdempotent(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	access, refresh := loginPair(t, h, drv.ID)

	const tok = "d12-idem"
	registerDevice(t, h, access, tok)

	first := logout(t, h, refresh, tok)
	second := logout(t, h, refresh, tok)
	t.Logf("D12: خروجٌ أوّل=%d · ثانٍ=%d", first, second)
	if first >= 400 || second >= 400 {
		t.Errorf("**الخروجُ المكرَّرُ ليس صامتَ النجاح**: %d ثمّ %d", first, second)
	}
	if got := deviceOwner(t, h, tok); got != "" {
		t.Errorf("**الرمزُ بُعث بعد خروجٍ مكرَّر**: %s", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D12-T5` · وتبديلُ الحساب على الجهاز نفسِه**
// ══════════════════════════════════════════════════════════════════════
//
// **وخروجُ الأوّل المتأخّرُ لا يسرق رمزَ الثاني** — **والحذفُ مقيَّدٌ
// بصاحبه** (`token AND user_id`).
func TestD12_AccountSwitchOnSameDevice(t *testing.T) {
	h := New(t)
	_, a := activeOrderFor(t, h)
	_, b := activeOrderFor(t, h)
	// **وقِيس أنّ دخولاً ثانياً على العميل نفسِه يُبطل الأوّل**
	// (عائلاتٌ حيّةٌ = ١) — **فلا يملك حسابٌ عائلتين حيّتين على
	// تطبيقٍ واحد**، **و«خروجٌ متأخّرٌ يسرق» غيرُ بالغٍ من هذا
	// الباب**: **رمزُ تجديدٍ أُنفق يقف عند إبطال العائلة ولا يبلغ
	// الحذف.**
	//
	// **والبابُ الذي يبلغه فعلاً هو إلغاءُ التسجيل الصريح**
	// (`DELETE /me/devices`) — **وهو موثَّقٌ بجلسةٍ حيّة**:
	// **فيُقاس به قيدُ الصاحب.**
	accessA, refreshA := loginPair(t, h, a.ID)
	accessB, _ := loginPair(t, h, b.ID)

	const tok = "d12-shared-device"
	registerDevice(t, h, accessA, tok)
	if code := logout(t, h, refreshA, tok); code >= 400 {
		t.Fatalf("خروجُ الأوّل: %d", code)
	}
	if got := deviceOwner(t, h, tok); got != "" {
		t.Fatalf("**الأوّلُ خرج والرمزُ باقٍ له**: %s", got)
	}

	registerDevice(t, h, accessB, tok)
	if got := deviceOwner(t, h, tok); got != b.ID {
		t.Fatalf("**الثاني سجّل ولم يملك الرمزَ**: %q", got)
	}

	// ── والأوّلُ يعود بجلسةٍ حيّةٍ ويطلب إلغاءَ الرمز نفسِه ────────
	freshA, _ := loginPair(t, h, a.ID)
	del := h.Call("DELETE", "/api/v1/me/devices", freshA, map[string]any{"token": tok}, nil)
	t.Logf("D12: إلغاءُ الأوّلِ لرمزٍ يملكه الثاني ⇒ %d", del.Code)
	if got := deviceOwner(t, h, tok); got != b.ID {
		t.Errorf("**الأوّلُ ألغى رمزاً يملكه الثاني**: %q — "+
			"**والحذفُ مقيَّدٌ بصاحبه.**", got)
	}
}

func TestD12_SuspensionIsNotLogout(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	access, _ := loginPair(t, h, drv.ID)

	const tok = "d12-suspended"
	registerDevice(t, h, access, tok)
	suspend(t, h, drv.ID, "suspended")

	if got := deviceOwner(t, h, tok); got != drv.ID {
		t.Errorf("**التعليقُ ألغى تسجيلَ الجهاز** (%q) — "+
			"**والتعليقُ حالُ عملٍ لا خروج** (`XG-39`).", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D12-T2` · والمُرسِلُ نفسُه لا يعود يختار الجهاز**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُكتفى بغياب الصفّ** — **يُشغَّل مسارُ الإشعار الحقيقيّ
// ويُقرأ ما وصل الناقل.**
func TestD12_DispatcherNoLongerTargetsLoggedOutDevice(t *testing.T) {
	fake := NewFakePush()
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	f := h.Factory()
	owner := f.NewUserWith("customer")

	// **وجلستان لا رمزان في جلسة** — انظر `LogoutIsDeviceScoped`.
	phone := d12Phone(t, h, owner.ID)
	accessGone, refreshGone := d12LoginAs(t, h, phone, "driver")
	accessKept, _ := d12LoginAs(t, h, phone, "customer")
	const gone, kept = "d12-gone", "d12-kept"
	registerDeviceAs(t, h, accessGone, gone, "android-driver")
	registerDeviceAs(t, h, accessKept, kept, "android-customer")

	if code := logout(t, h, refreshGone, gone); code >= 400 {
		t.Fatalf("الخروج: %d", code)
	}

	got := h.POST("/api/v1/admin/users/"+owner.ID+"/wallet", admin.Token,
		map[string]any{"amount": 5000, "kind": "topup", "note": "D12"})
	if got.Code >= 400 {
		t.Fatalf("القيد: %s", got)
	}
	// **ومسارُ التسليم يُنادي لكلّ هدفٍ نداءً** — **فانتظارُ نداءٍ
	// واحدٍ يقرأ نصفَ التوزيع.** **فيُنتظَر الجهازُ الباقي حتّى
	// يصل**: **وصولُه دليلُ أنّ التوزيعَ تمّ**، **ثمّ يُسأل عن
	// الخارج.**
	seen := func() (goneSeen, keptSeen bool) {
		for _, c := range fake.Calls() {
			for _, tok := range c.Tokens {
				if tok == gone {
					goneSeen = true
				}
				if tok == kept {
					keptSeen = true
				}
			}
		}
		return
	}
	deadline := time.Now().Add(8 * time.Second)
	var sawGone, sawKept bool
	for time.Now().Before(deadline) {
		sawGone, sawKept = seen()
		if sawKept {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	// **ومهلةٌ قصيرةٌ بعد وصول الباقي** — **فلو كان الخارجُ في
	// الطريق لَوصل.**
	time.Sleep(500 * time.Millisecond)
	sawGone, sawKept = seen()
	for _, c := range fake.Calls() {
		t.Logf("  نداءٌ: الرموز=%v · العنوان=%q", c.Tokens, c.Title)
	}
	t.Logf("D12: المُرسِلُ اختار — الخارج=%t · الباقي=%t", sawGone, sawKept)
	if sawGone {
		t.Error("**المُرسِلُ ما زال يختار جهازاً خرج حسابُه** — **وذاك `D12`.**")
	}
	if !sawKept {
		t.Error("**الجهازُ الباقي لم يُختَر** — **والخروجُ أصمت جهازاً آخر.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **تكرارُ `D12`** — **مرّةٌ تُصادِف، والمئةُ تحكم**
// ══════════════════════════════════════════════════════════════════════

// d12Login **دخولٌ متكرّرٌ بكلمةٍ ضُبطت مرّة** — فلا يُعاد ضبطُها مئةً.
func d12Login(t *testing.T, h *Harness, phone string) (access, refresh string) {
	t.Helper()
	in := h.POST("/api/v1/auth/login", "", map[string]any{
		"phone": phone, "password": "Qa!Refresh-2026", "client": "driver",
	})
	if in.Code >= 400 {
		t.Fatalf("الدخول: %s", in)
	}
	return tokenField(in, "access_token"), tokenField(in, "refresh_token")
}

func d12Phone(t *testing.T, h *Harness, userID string) string {
	t.Helper()
	issuedRefreshStrict(t, h, userID, "driver") // **يضبط الكلمة مرّةً**
	var phone string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT phone FROM users WHERE id = $1::uuid`, userID).Scan(&phone); err != nil {
		t.Fatalf("الهاتف: %v", err)
	}
	return phone
}

func TestD12_StressLoginRegisterLogout(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	phone := d12Phone(t, h, drv.ID)

	stale := 0
	for i := 0; i < 100; i++ {
		access, refresh := d12Login(t, h, phone)
		tok := uniq("d12-loop")
		registerDevice(t, h, access, tok)
		if code := logout(t, h, refresh, tok); code >= 400 {
			t.Fatalf("الخروج في الجولة %d: %d", i, code)
		}
		if deviceOwner(t, h, tok) != "" {
			stale++
		}
	}
	if stale != 0 {
		t.Errorf("**وجهاتٌ بقيت بعد الخروج**: %d من 100", stale)
	}
}

func TestD12_StressTwoDevicesAndSwitch(t *testing.T) {
	h := New(t)
	_, a := activeOrderFor(t, h)
	_, b := activeOrderFor(t, h)
	phoneA := d12Phone(t, h, a.ID)
	phoneB := d12Phone(t, h, b.ID)

	otherDropped, stolen, staleOld := 0, 0, 0
	for i := 0; i < 100; i++ {
		accessA1, refreshA1 := d12LoginAs(t, h, phoneA, "driver")
		accessA2, _ := d12LoginAs(t, h, phoneA, "customer")
		one, two := uniq("d12-s1"), uniq("d12-s2")
		registerDeviceAs(t, h, accessA1, one, "android-driver")
		registerDeviceAs(t, h, accessA2, two, "android-customer")
		logout(t, h, refreshA1, one)
		if deviceOwner(t, h, one) != "" {
			staleOld++
		}
		if deviceOwner(t, h, two) != a.ID {
			otherDropped++
		}

		// ── وتبديلُ الحساب على الجهاز نفسِه ───────────────────────
		accessB, _ := d12LoginAs(t, h, phoneB, "driver")
		registerDeviceAs(t, h, accessB, one, "android-driver")
		// **خروجٌ متأخّرٌ من الأوّل بالرمز نفسِه** — لا يسرق.
		logout(t, h, refreshA1, one)
		if deviceOwner(t, h, one) != b.ID {
			stolen++
		}
	}
	if staleOld != 0 {
		t.Errorf("**وجهةُ الخروج بقيت**: %d", staleOld)
	}
	if otherDropped != 0 {
		t.Errorf("**جهازٌ آخرُ أُسقط**: %d", otherDropped)
	}
	if stolen != 0 {
		t.Errorf("**خروجٌ متأخّرٌ سرق رمزَ حسابٍ آخر**: %d", stolen)
	}
}

func TestD12_StressLogoutVsReRegisterRace(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	phone := d12Phone(t, h, drv.ID)

	survived := 0
	for i := 0; i < 100; i++ {
		access, refresh := d12Login(t, h, phone)
		tok := uniq("d12-race")
		registerDevice(t, h, access, tok)

		r := Race(t, DefaultRaceTimeout,
			Actor{Name: "الخروج", Do: func(context.Context) any {
				return h.POST("/api/v1/auth/logout", "",
					map[string]any{"refresh_token": refresh, "device_token": tok})
			}},
			Actor{Name: "إعادةُ التسجيل", Do: func(context.Context) any {
				return h.POST("/api/v1/me/devices", access,
					map[string]any{"token": tok, "platform": "android"})
			}},
		)
		if r.TimedOut {
			t.Fatalf("السباقُ عَلِق — %s", r)
		}
		// **والمقيسُ حالٌ مستقرّة**: **إن بقيت الوجهةُ فقد كتبها
		// تسجيلٌ سبق الحذفَ بجلسةٍ كانت حيّةً حينَه** — **ويموت
		// بالزمن.** **والممنوعُ أن تبقى وقد تمّ الخروجُ بعده.**
		if deviceOwner(t, h, tok) != "" {
			// **يُعاد الخروجُ بعد استقرار السباق** — فإن بقيت
			// بعده فهي وجهةٌ حيّةٌ لحسابٍ خرج.
			logout(t, h, refresh, tok)
			if deviceOwner(t, h, tok) != "" {
				survived++
			}
		}
	}
	if survived != 0 {
		t.Errorf("**وجهةٌ نجت من خروجٍ تامّ**: %d من 100", survived)
	}
}

func TestD12_StressIdempotentLogout(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	phone := d12Phone(t, h, drv.ID)

	bad := 0
	for i := 0; i < 50; i++ {
		access, refresh := d12Login(t, h, phone)
		tok := uniq("d12-idem")
		registerDevice(t, h, access, tok)
		for k := 0; k < 3; k++ {
			if logout(t, h, refresh, tok) >= 400 {
				bad++
			}
		}
		if deviceOwner(t, h, tok) != "" {
			bad++
		}
	}
	if bad != 0 {
		t.Errorf("**الخروجُ المكرَّرُ ليس مستقرّاً**: %d", bad)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ودليلُ الجهاز يبقى** — **حارسُ وجودٍ لا حارسُ سلوك**
// ══════════════════════════════════════════════════════════════════════
//
// **والسلوكُ مقيسٌ أعلاه بالمسار الحقيقيّ** — **وهذا يمنع أن يُحذف
// نداءُ العميل فتبقى حزمةُ Go خضراءَ والسجلُّ يقول `D12` مُغلَق.**
func TestD12_ClientSendsDeviceTokenOnLogout(t *testing.T) {
	must := func(path string) string {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("**دليلُ `D12` غائبٌ**: %s — %v", path, err)
		}
		return string(b)
	}
	api := must("../../../mobile/shared/src/main/kotlin/com/rahalgo/shared/auth/AuthApi.kt")
	if !strings.Contains(api, `"device_token" to deviceToken`) {
		t.Error("**نداءُ الخروج لا يحمل رمزَ الجهاز** — **وذاك `D12`.**")
	}
	for _, f := range []string{
		"../../../mobile/ui/src/main/kotlin/com/rahalgo/ui/AuthViewModel.kt",
		"../../../mobile/app-driver/src/main/kotlin/com/rahalgo/driver/home/HomeViewModel.kt",
	} {
		src := must(f)
		if !strings.Contains(src, "Push.currentToken()") {
			t.Errorf("**بابُ خروجٍ لا يقرأ رمزَ جهازه**: %s", f)
		}
		if !strings.Contains(src, "backend.auth.logout(refresh, device)") {
			t.Errorf("**بابُ خروجٍ لا يُرسل رمزَ جهازه**: %s", f)
		}
	}
	var automated, onDevice int
	for _, c := range androidmap.Cases {
		for _, r := range c.Registers {
			if r != "D12" {
				continue
			}
			if c.Automation == androidmap.Automated {
				automated++
				if c.Result != androidmap.Pass {
					t.Errorf("**حالةٌ آليّةٌ لـ`D12` ليست ناجحة**: %s ⇒ %s", c.ID, c.Result)
				}
			} else {
				onDevice++
			}
		}
	}
	t.Logf("D12: حالاتٌ آليّةٌ=%d · على جهازٍ=%d", automated, onDevice)
	if automated == 0 {
		t.Error("**لا حالةَ آليّةٌ مسجَّلةٌ لـ`D12`**")
	}
	if onDevice == 0 {
		t.Error("**حُذف قبولُ الجهاز لـ`D12`** — **والمحلّيُّ لا يُغني.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **`D12` · دورةُ ٦٠ — والخروجُ لا يتّكل على رمزٍ يُقرأ من الجهاز**
// ══════════════════════════════════════════════════════════════════════
//
// **ورمزُ `FCM` قد لا يُقرأ ساعةَ الخروج** — خدماتٌ معطّلةٌ أو شبكةٌ
// منقطعةٌ أو مهلةٌ انتهت. **وكان الخروجُ حينها يمضي ناجحاً والوجهةُ
// تبقى حيّةً حتّى يقتلها القِدَم بعد شهرين.**
//
// **وخروجٌ نجح لا يحتمل «سيُنظَّف لاحقاً».**

// ══════════════════════════════════════════════════════════════════════
// **الخروجُ بلا رمزِ جهازٍ يُنهي وجهةَ عائلته**
// ══════════════════════════════════════════════════════════════════════
func TestD12_LogoutWithoutDeviceTokenStillRemovesBinding(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	access, refresh := loginPair(t, h, drv.ID)

	const tok = "d12-no-client-token"
	registerDevice(t, h, access, tok)
	if got := deviceOwner(t, h, tok); got != drv.ID {
		t.Fatalf("**التسجيلُ لم يربط الرمز**: %q", got)
	}
	// **والمِلكيّةُ عائلةُ جلسةٍ لا حساب** — فتُقرأ ويُتحقَّق منها.
	var sid string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT coalesce(session_id::text, '') FROM device_tokens WHERE token = $1`,
		tok).Scan(&sid); err != nil {
		t.Fatalf("مِلكيّةُ الرمز: %v", err)
	}
	t.Logf("D12/60: مالكُ الرمز عائلةٌ=%t", sid != "")
	if sid == "" {
		t.Fatal("**التسجيلُ لم يكتب عائلةَ الجلسة** — " +
			"**فلا سلطةَ للخروج إلّا رمزُ الجهاز.**")
	}

	// ── والخروجُ بلا رمزِ جهازٍ البتّة ────────────────────────────
	if code := logout(t, h, refresh, ""); code >= 400 {
		t.Fatalf("**الخروجُ رُدّ** (%d)", code)
	}
	if got := deviceOwner(t, h, tok); got != "" {
		t.Errorf("**وجهةٌ بقيت بعد خروجٍ ناجحٍ بلا رمزِ جهاز** (%s) — "+
			"**والقِدَمُ ليس ضمانَ خروج.**", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **وجلسةٌ ثانيةٌ للحساب نفسِه لا تُمَسّ** — بلا رمزِ جهاز
// ══════════════════════════════════════════════════════════════════════
//
// **وعائلتان حيّتان تحتاجان تطبيقين** — **قِيس (دورةُ ٥٩) أنّ دخولاً
// ثانياً على العميل نفسِه يُبطل الأوّل.**
func TestD12_SessionScopedCleanupKeepsOtherSession(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	phone := d12Phone(t, h, drv.ID)

	accessDriver, refreshDriver := d12LoginAs(t, h, phone, "driver")
	accessCust, _ := d12LoginAs(t, h, phone, "customer")

	const tDriver, tCust = "d12-sess-driver", "d12-sess-customer"
	registerDeviceAs(t, h, accessDriver, tDriver, "android-driver")
	registerDeviceAs(t, h, accessCust, tCust, "android-customer")

	if code := logout(t, h, refreshDriver, ""); code >= 400 {
		t.Fatalf("الخروج: %d", code)
	}
	if got := deviceOwner(t, h, tDriver); got != "" {
		t.Errorf("**وجهةُ الجلسة الخارجة بقيت**: %s", got)
	}
	if got := deviceOwner(t, h, tCust); got != drv.ID {
		t.Errorf("**خروجُ جلسةٍ أسقط وجهةَ جلسةٍ أخرى للحساب نفسِه**: %q", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ودورانُ الرمز في الجلسة نفسِها يُنظَّف كلُّه**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُقرأ من الجهاز إلّا آخرُ رمز** — **فحذفٌ بالرمز يترك ما
// سبقه.** **والمِلكيّةُ تطالهما.**
func TestD12_TokenRotationWithinSessionIsFullyCleaned(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	access, refresh := loginPair(t, h, drv.ID)

	const first, second = "d12-rot-1", "d12-rot-2"
	registerDevice(t, h, access, first)
	registerDevice(t, h, access, second)

	// **ويُرسَل آخرُ رمزٍ وحدَه** — كما يفعل جهازٌ دوّر رمزَه.
	if code := logout(t, h, refresh, second); code >= 400 {
		t.Fatalf("الخروج: %d", code)
	}
	if got := deviceOwner(t, h, second); got != "" {
		t.Errorf("**الرمزُ الأخيرُ بقي**: %s", got)
	}
	if got := deviceOwner(t, h, first); got != "" {
		t.Errorf("**رمزٌ دُوّر قبله بقي وجهةً لحسابٍ خرج**: %s — "+
			"**والجهازُ لا يعرفه ليُرسله.**", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والمُرسِلُ لا يختار شيئاً بعد خروجٍ بلا رمزِ جهاز**
// ══════════════════════════════════════════════════════════════════════
func TestD12_DispatcherSilentAfterTokenlessLogout(t *testing.T) {
	fake := NewFakePush()
	h := NewWith(t, server.WithPushTransport(fake))
	admin := h.NewUser("admin")
	f := h.Factory()
	owner := f.NewUserWith("customer")
	phone := d12Phone(t, h, owner.ID)

	accessDriver, refreshDriver := d12LoginAs(t, h, phone, "driver")
	accessCust, _ := d12LoginAs(t, h, phone, "customer")
	const gone, kept = "d12-fi-gone", "d12-fi-kept"
	registerDeviceAs(t, h, accessDriver, gone, "android-driver")
	registerDeviceAs(t, h, accessCust, kept, "android-customer")

	if code := logout(t, h, refreshDriver, ""); code >= 400 {
		t.Fatalf("الخروج: %d", code)
	}

	got := h.POST("/api/v1/admin/users/"+owner.ID+"/wallet", admin.Token,
		map[string]any{"amount": 5000, "kind": "topup", "note": "D12/60"})
	if got.Code >= 400 {
		t.Fatalf("القيد: %s", got)
	}
	seen := func() (g, k bool) {
		for _, c := range fake.Calls() {
			for _, tk := range c.Tokens {
				if tk == gone {
					g = true
				}
				if tk == kept {
					k = true
				}
			}
		}
		return
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if _, k := seen(); k {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	sawGone, sawKept := seen()
	t.Logf("D12/60: المُرسِلُ — الخارج=%t · الباقي=%t", sawGone, sawKept)
	if sawGone {
		t.Error("**المُرسِلُ اختار وجهةَ جلسةٍ خرجت بلا رمزِ جهاز**")
	}
	if !sawKept {
		t.Error("**الجلسةُ الباقيةُ لم تُختَر**")
	}
}

// d12LoginAs **دخولٌ بعميلٍ بعينه** — **وعائلتان حيّتان تحتاجان
// تطبيقين**: دخولٌ ثانٍ على العميل نفسِه يُبطل الأوّل.
func d12LoginAs(t *testing.T, h *Harness, phone, client string) (access, refresh string) {
	t.Helper()
	in := h.Call("POST", "/api/v1/auth/login", "", map[string]any{
		"phone": phone, "password": "Qa!Refresh-2026", "client": client,
	}, map[string]string{"X-RahalGo-Client": "android-" + client})
	if in.Code >= 400 {
		t.Fatalf("الدخولُ بـ%s: %s", client, in)
	}
	return tokenField(in, "access_token"), tokenField(in, "refresh_token")
}

// registerDeviceAs تسجيلٌ بترويسةِ عميلٍ بعينها — **فالتطبيقُ منها.**
func registerDeviceAs(t *testing.T, h *Harness, access, token, clientHeader string) {
	t.Helper()
	got := h.Call("POST", "/api/v1/me/devices", access,
		map[string]any{"token": token, "platform": "android"},
		map[string]string{"X-RahalGo-Client": clientHeader})
	if got.Code >= 400 {
		t.Fatalf("تسجيلُ الجهاز (%s): %s", token, got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **تكرارُ دورةِ ٦٠ — والمِلكيّةُ عائلةُ جلسة**
// ══════════════════════════════════════════════════════════════════════

func TestD12_StressTokenlessLogout(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	phone := d12Phone(t, h, drv.ID)

	stale, rotationLeft := 0, 0
	for i := 0; i < 100; i++ {
		access, refresh := d12LoginAs(t, h, phone, "driver")
		first, second := uniq("d12-tl-a"), uniq("d12-tl-b")
		registerDeviceAs(t, h, access, first, "android-driver")
		// **ودورانُ رمزٍ في الجلسة نفسِها** — **والجهازُ لا يعرف
		// إلّا آخرَه.**
		registerDeviceAs(t, h, access, second, "android-driver")

		// **ولا رمزَ جهازٍ البتّة** — تعذّرت قراءتُه.
		if code := logout(t, h, refresh, ""); code >= 400 {
			t.Fatalf("الخروج في الجولة %d: %d", i, code)
		}
		if deviceOwner(t, h, second) != "" {
			stale++
		}
		if deviceOwner(t, h, first) != "" {
			rotationLeft++
		}
	}
	if stale != 0 {
		t.Errorf("**وجهاتٌ بقيت بعد خروجٍ بلا رمزِ جهاز**: %d", stale)
	}
	if rotationLeft != 0 {
		t.Errorf("**رموزٌ دُوّرت بقيت وجهاتٍ لحسابٍ خرج**: %d", rotationLeft)
	}
}

func TestD12_StressTwoSessionsAndClaim(t *testing.T) {
	h := New(t)
	_, a := activeOrderFor(t, h)
	_, b := activeOrderFor(t, h)
	phoneA := d12Phone(t, h, a.ID)
	phoneB := d12Phone(t, h, b.ID)

	otherDropped, stolen, staleGone := 0, 0, 0
	for i := 0; i < 100; i++ {
		accessA1, refreshA1 := d12LoginAs(t, h, phoneA, "driver")
		accessA2, _ := d12LoginAs(t, h, phoneA, "customer")
		gone, kept := uniq("d12-2s-g"), uniq("d12-2s-k")
		registerDeviceAs(t, h, accessA1, gone, "android-driver")
		registerDeviceAs(t, h, accessA2, kept, "android-customer")

		// ── والجلسةُ الجديدةُ تطالب بالرمز قبل خروج القديمة ──────
		accessB, _ := d12LoginAs(t, h, phoneB, "driver")
		registerDeviceAs(t, h, accessB, gone, "android-driver")

		logout(t, h, refreshA1, gone)
		if deviceOwner(t, h, gone) != b.ID {
			stolen++
		}
		if deviceOwner(t, h, kept) != a.ID {
			otherDropped++
		}

		// ── ثمّ خروجٌ لجلسةٍ لم يُطالَب برمزها ───────────────────
		accessC, refreshC := d12LoginAs(t, h, phoneA, "driver")
		solo := uniq("d12-2s-s")
		registerDeviceAs(t, h, accessC, solo, "android-driver")
		logout(t, h, refreshC, "")
		if deviceOwner(t, h, solo) != "" {
			staleGone++
		}
	}
	if stolen != 0 {
		t.Errorf("**خروجُ جلسةٍ قديمةٍ سرق مِلكيّةَ جلسةٍ جديدة**: %d", stolen)
	}
	if otherDropped != 0 {
		t.Errorf("**جلسةٌ أخرى للحساب نفسِه أُسقطت**: %d", otherDropped)
	}
	if staleGone != 0 {
		t.Errorf("**وجهةٌ بقيت بعد خروجٍ بلا رمز**: %d", staleGone)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **وحقنُ فشلٍ بين الإبطال والحذف — لا خروجَ نصفَ ناجح**
// ══════════════════════════════════════════════════════════════════════
//
// **والعقدُ أنّ خروجاً نجح يعني الاثنين معاً** — **فإن تعذّر حذفُ
// الوجهة لم تُبطَل الجلسةُ أيضاً**: **يُعيد صاحبُه المحاولة، ولا
// يبقى بجلسةٍ ميّتةٍ ووجهةٍ حيّة.**
func TestD12_FailureBetweenRevokeAndCleanupIsAtomic(t *testing.T) {
	h := New(t)
	_, drv := activeOrderFor(t, h)
	access, refresh := loginPair(t, h, drv.ID)

	const tok = "d12-fi-atomic"
	registerDevice(t, h, access, tok)

	// **والفشلُ يقع على حذف الوجهة داخلَ معاملة الخروج.**
	fp := h.ArmAny("d12-cleanup", "device_tokens", "DELETE")
	code := logout(t, h, refresh, "")
	t.Logf("D12/60: الخروجُ مع فشلِ الحذف ⇒ %d", code)
	fp.MustFire(t)
	fp.Disarm()

	if code < 400 {
		t.Errorf("**رُدّ الخروجُ ناجحاً والحذفُ سقط** (%d) — "+
			"**فهو خروجٌ نصفُ ناجح.**", code)
	}
	// ── ولا شيءَ ثُبِّت: الوجهةُ باقيةٌ والجلسةُ حيّة ─────────────
	if got := deviceOwner(t, h, tok); got != drv.ID {
		t.Errorf("**الوجهةُ حُذفت رغم سقوط المعاملة**: %q", got)
	}
	if res := h.GET(mePath, access); res.Code != 200 {
		t.Errorf("**الجلسةُ أُبطلت رغم سقوط المعاملة** (%d) — "+
			"**فالإبطالُ ثُبِّت والحذفُ لا.**", res.Code)
	}

	// ── وإعادةُ المحاولة بعد زوال العطب تُتمّ الاثنين ─────────────
	if again := logout(t, h, refresh, ""); again >= 400 {
		t.Fatalf("**إعادةُ الخروج بعد زوال العطب رُدّت** (%d)", again)
	}
	if got := deviceOwner(t, h, tok); got != "" {
		t.Errorf("**الوجهةُ بقيت بعد خروجٍ ناجح**: %s", got)
	}
	if res := h.GET(mePath, access); res.Code != 401 {
		t.Errorf("**الجلسةُ لم تُبطَل بعد خروجٍ ناجح** (%d)", res.Code)
	}
}
