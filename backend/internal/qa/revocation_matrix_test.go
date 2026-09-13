package qa

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **مصفوفةُ الإبطال — ومن أُبطل لا يُوصَل إليه** (`SEC`، ٢٠٢٦-٠٩-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// # العقد
//
// **وحدثٌ قُصد به إبطالُ جلسةٍ أو عائلةٍ أو حسابٍ يجب أن يمنع فعلاً**:
// **رمزُ الوصول** · **رمزُ التجديد** · **ووجهةُ الدفع.**
//
// **والثالثةُ هي التي كانت متروكة**: **وجهاتُ الدفع تُنتقى بـ`user_id`
// وحدَه** (`push.tokensOf` و`delivery.go`) — **لا بصلاحيّةِ جلسةٍ ولا
// بحالِ حساب.** **فمن أُخرج إخراجاً شاملاً، أو أُعيدت كلمتُه، أو حُظر،
// أو حُذف حسابُه — يبقى جهازُه يستقبل إشعاراتِ رحّال غو الخاصّة.**
//
// **وذاك انكشافُ خصوصيّةٍ لا إزعاج**: **الإشعارُ يحمل حالَ طلبٍ واسمَ
// متجرٍ ومبلغاً** — **على جهازٍ قُصد قطعُه.**
//
// # وحدُّ السياسة مقصود
//
//	خروجُ الجلسة الحاليّة   وجهةُ هذه العائلة وحدَها
//	إخراجٌ شامل            كلُّ وجهاتِ الحساب
//	إعادةُ كلمةٍ (إدارة)     كلُّ وجهاتِ الحساب
//	تبديلُ كلمةٍ بنفسه      وجهاتُ العائلات المقطوعة — **وتبقى عائلتُه**
//	حظرٌ                  كلُّ وجهاتِ الحساب
//	حذفٌ إداريٌّ (`deleted`)  كلُّ وجهاتِ الحساب
//	حذفٌ ذاتيٌّ             كلُّ وجهاتِ الحساب
//
// **ولا تُحذف وجهةُ جلسةٍ تقول السياسةُ إنّها باقية** — **وتبديلُ
// الكلمة بنفسه يُبقي عائلتَه** (`XG-40`)، **فوجهتُها تبقى معها.**

// plantOTP **يزرع رمزاً بالبصمة التي يقرؤها المحرّك** ويعيد نصَّه.
//
// **والرموزُ تُخزَّن مبصومةً بـ`HMAC-SHA256`** (`identity.hashOTP`) —
// **فلا تُقرأ من القاعدة.** **والعُدّةُ تعرف السرَّ نفسَه**، فتُبصَم
// هنا كما تُبصَم هناك. **ولا يُخترَع بابٌ للاختبار في المنتَج.**
func plantOTP(t *testing.T, h *Harness, userID, purpose string) string {
	t.Helper()
	var phone string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT phone FROM users WHERE id = $1::uuid`, userID).Scan(&phone); err != nil {
		t.Fatalf("هاتفُ الحساب: %v", err)
	}
	code := "424242"
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte(phone + ":" + code))
	sum := hex.EncodeToString(mac.Sum(nil))
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO otp_codes (phone, code_hash, purpose, expires_at)
		VALUES ($1, $2, $3, now() + interval '10 minutes')`,
		phone, sum, purpose); err != nil {
		t.Fatalf("زرعُ الرمز: %v", err)
	}
	return code
}

// refreshFamily **عائلةٌ برمز تجديدٍ حقيقيٍّ** — ويعيد نصَّه وعائلتَه.
//
// **والخروجُ يوثَّق برمز التجديد نفسِه** (`Logout`)، **فلا يكفي صفٌّ
// ببصمةٍ مخترعة.**
func refreshFamily(t *testing.T, h *Harness, userID, client string) (raw, sid string) {
	t.Helper()
	raw = "qa-refresh-" + uniq("r")
	sum := sha256.Sum256([]byte(raw))
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', $3)
		RETURNING session_id::text`,
		userID, hex.EncodeToString(sum[:]), client).Scan(&sid); err != nil {
		t.Fatalf("صفُّ التجديد: %v", err)
	}
	return raw, sid
}

// devRow يسجّل وجهةَ دفعٍ لعائلةِ جلسةٍ بعينها — بالمِلكيّة لا بالرمز.
//
// **والتسجيلُ بالقاعدة لا بالمسار**: **مسارُ `POST /me/devices` يوثَّق
// برمز وصولٍ ولا يعرف العائلةَ إلّا منه** — **وهذه الفحوصُ تحتاج
// عائلاتٍ مصنوعةً بعينها.** (وهو نمطُ `deliverySession` نفسُه.)
func devRow(t *testing.T, h *Harness, userID, sid, token string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO device_tokens (user_id, session_id, token, platform, app, last_seen_at)
		VALUES ($1::uuid, $2::uuid, $3, 'android', 'customer', now())`,
		userID, sid, token); err != nil {
		t.Fatalf("صفُّ الوجهة: %v", err)
	}
}

// devAlive **أتبقى الوجهةُ هدفاً للدفع؟** — بالشرط الذي ينتقي به الدفعُ نفسُه.
func devAlive(t *testing.T, h *Harness, token string) bool {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM device_tokens
		 WHERE token = $1 AND last_seen_at > now() - interval '60 days'`,
		token).Scan(&n); err != nil {
		t.Fatalf("عدُّ الوجهة: %v", err)
	}
	return n > 0
}

// secVerdict يقرأ الثلاثةَ معاً لعائلةٍ واحدة.
type secVerdict struct {
	access bool // **أيمرّ رمزُ الوصول؟**
	live   int  // صفوفُ التجديد الحيّة
	push   bool // **أتبقى وجهةُ الدفع؟**
}

func secRead(t *testing.T, h *Harness, tok, sid, dev string) secVerdict {
	t.Helper()
	return secVerdict{
		access: h.GET(mePath, tok).Code == http.StatusOK,
		live:   liveRows(t, h, sid),
		push:   devAlive(t, h, dev),
	}
}

// secTwoFamilies حسابٌ بعائلتين، لكلٍّ وجهتُها.
func secTwoFamilies(t *testing.T, h *Harness, role string) (
	uid string, tokA, sidA, devA, tokB, sidB, devB string) {
	t.Helper()
	u := h.NewUser(role)
	tokA, sidA = deliverySession(t, h, u.ID, "customer", role)
	tokB, sidB = deliverySession(t, h, u.ID, "web", role)
	devA, devB = "sec-"+uniq("a"), "sec-"+uniq("b")
	devRow(t, h, u.ID, sidA, devA)
	devRow(t, h, u.ID, sidB, devB)
	return u.ID, tokA, sidA, devA, tokB, sidB, devB
}

// ══════════════════════════════════════════════════════════════════════
// **SEC1 · خروجُ الجلسة الحاليّة — عائلتُها وحدَها**
// ══════════════════════════════════════════════════════════════════════
//
// **والثانيةُ تبقى بكلّ شيء** — **ومن خرج من هاتفه لا يُخرَج من حاسبه.**
func TestSEC1_CurrentLogoutTouchesOneFamilyOnly(t *testing.T) {
	h := New(t)
	u := h.NewUser("customer")
	raw, sidA := refreshFamily(t, h, u.ID, "customer")
	tokA := h.TokenWithSession(u.ID, sidA, "customer")
	tokB, sidB := deliverySession(t, h, u.ID, "web", "customer")
	devA, devB := "sec-"+uniq("a"), "sec-"+uniq("b")
	devRow(t, h, u.ID, sidA, devA)
	devRow(t, h, u.ID, sidB, devB)

	if got := h.POST("/api/v1/auth/logout", "",
		map[string]any{"refresh_token": raw}); got.Code >= 400 {
		t.Fatalf("SEC1 تعذّر الخروج: %s", got)
	}

	a := secRead(t, h, tokA, sidA, devA)
	b := secRead(t, h, tokB, sidB, devB)
	t.Logf("SEC1 أ=%+v · ب=%+v", a, b)

	if a.access || a.live != 0 || a.push {
		t.Errorf("SEC1 **الخروجُ لم يُغلق عائلتَه**: %+v", a)
	}
	if !b.access || b.live != 1 || !b.push {
		t.Errorf("SEC1 **الخروجُ مسّ عائلةً أخرى**: %+v", b)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **SEC2 · إخراجٌ شاملٌ — والوجهاتُ كلُّها معه**
// ══════════════════════════════════════════════════════════════════════
func TestSEC2_LogoutAllClearsEveryDestination(t *testing.T) {
	h := New(t)
	uid, tokA, sidA, devA, tokB, sidB, devB := secTwoFamilies(t, h, "customer")

	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/users/"+uid+"/logout-all", admin.Token,
		map[string]any{}); got.Code >= 400 {
		t.Fatalf("SEC2 تعذّر الإخراجُ الشامل: %s", got)
	}

	a := secRead(t, h, tokA, sidA, devA)
	b := secRead(t, h, tokB, sidB, devB)
	t.Logf("SEC2 أ=%+v · ب=%+v", a, b)

	for name, v := range map[string]secVerdict{"أ": a, "ب": b} {
		if v.access || v.live != 0 {
			t.Errorf("SEC2 **العائلةُ %s بقيت بعد الإخراج الشامل**: %+v", name, v)
		}
		if v.push {
			t.Errorf("SEC2 **وجهةُ %s تبقى تستقبل إشعاراً خاصّاً بعد إخراجٍ شامل**", name)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **SEC3 · إعادةُ كلمةٍ إداريّةٌ — إبطالٌ وقطعُ وجهات**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو عقدُ `R13`** — **والوجهةُ هي الشقُّ الذي كان ناقصاً منه.**
func TestSEC3_AdminResetClearsEveryDestination(t *testing.T) {
	h := New(t)
	uid, tokA, sidA, devA, tokB, sidB, devB := secTwoFamilies(t, h, "driver")

	if got := resetPassword(t, h, uid, "Qa!Sec3-2026"); got.Code >= 400 {
		t.Fatalf("SEC3 تعذّرت الإعادة: %s", got)
	}

	a := secRead(t, h, tokA, sidA, devA)
	b := secRead(t, h, tokB, sidB, devB)
	t.Logf("SEC3 أ=%+v · ب=%+v", a, b)

	for name, v := range map[string]secVerdict{"أ": a, "ب": b} {
		if v.access || v.live != 0 {
			t.Errorf("SEC3 **العائلةُ %s بقيت بعد إعادة الكلمة**: %+v", name, v)
		}
		if v.push {
			t.Errorf("SEC3 **وجهةُ %s تبقى بعد إعادة الكلمة** — "+
				"**ومن سُرق حسابُه يبقى جهازُ السارق يستقبل**", name)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **SEC5 · حظرٌ — إبطالٌ شاملٌ وقطعُ وجهات**
// ══════════════════════════════════════════════════════════════════════
//
// **وحالُ أمنٍ استثنائيّةٌ بعقد `XG-39`** — **فلا تُترَك لها وجهة.**
func TestSEC5_BlockedAccountReceivesNothing(t *testing.T) {
	h := New(t)
	uid, tokA, sidA, devA, tokB, sidB, devB := secTwoFamilies(t, h, "customer")

	admin := h.NewUser("admin")
	if got := h.PATCH("/api/v1/admin/users/"+uid, admin.Token,
		map[string]any{"status": "blocked", "status_reason": "SEC5"}); got.Code >= 400 {
		t.Fatalf("SEC5 تعذّر الحظر: %s", got)
	}

	a := secRead(t, h, tokA, sidA, devA)
	b := secRead(t, h, tokB, sidB, devB)
	t.Logf("SEC5 أ=%+v · ب=%+v", a, b)

	for name, v := range map[string]secVerdict{"أ": a, "ب": b} {
		if v.access {
			t.Errorf("SEC5 **المحظورُ ما زال يمرّ بعائلته %s**", name)
		}
		if v.live != 0 {
			t.Errorf("SEC5 **صفُّ تجديدٍ حيٌّ للمحظور في %s**: %d", name, v.live)
		}
		if v.push {
			t.Errorf("SEC5 **وجهةُ %s تبقى لمحظور** — **والحظرُ حالُ أمن**", name)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **SEC6 · حذفٌ إداريٌّ — أو لا بابَ له أصلاً**
// ══════════════════════════════════════════════════════════════════════
//
// **و`deleted` لا تُصنع من باب الإدارة** (`XG-39`: مدخلاتُه `active` و
// `suspended` و`blocked` لا غير) — **فيُقاس ذلك صراحةً لا يُفترَض.**
//
// **ومن وسّع البابَ غداً سقط هذا الفحصُ ونبّه أنّ مصفوفةَ الإبطال
// تحتاج فرعاً سادساً.**
func TestSEC6_AdminCannotMintDeletedState(t *testing.T) {
	h := New(t)
	u := h.NewUser("customer")
	admin := h.NewUser("admin")

	got := h.PATCH("/api/v1/admin/users/"+u.ID, admin.Token,
		map[string]any{"status": "deleted"})
	t.Logf("SEC6 الردُّ %d: %s", got.Code, got)

	if got.Code < 400 {
		t.Errorf("SEC6 **بابُ الإدارة صنع `deleted`** — " +
			"**فمصفوفةُ الإبطال تحتاج فرعاً لهذا المسار.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **SEC7 · الحذفُ الذاتيُّ — `XG-49` بحارسٍ دائم**
// ══════════════════════════════════════════════════════════════════════
//
// **وكانت فجوةَ دليلٍ لا عيبَ منتَج**: **السلوكُ قائمٌ في المصدر ولا
// اختبارَ واحدٌ يمرّ به** — **فمن نزع الإبطالَ غداً لم يسقط له شيء.**
//
// **ويُمشى بالمسار الحقيقيّ**: **طلبُ رمزٍ ⇒ تأكيدٌ بالرمز ⇒ تجريد** —
// **والرمزُ يُقرأ من القاعدة كما يقرؤه المسارُ نفسُه، ولا يُخترَع بابٌ
// للاختبار.**
func TestSEC7_SelfDeleteRevokesEverything(t *testing.T) {
	h := New(t)
	uid, tokA, sidA, devA, tokB, sidB, devB := secTwoFamilies(t, h, "customer")

	// **والطلبُ بجلستِه هو** — فالحذفُ الذاتيُّ لا يُنادى من غيره.
	if got := h.POST("/api/v1/auth/account/delete/request", tokA,
		map[string]any{}); got.Code >= 400 {
		t.Fatalf("SEC7 تعذّر طلبُ الحذف: %s", got)
	}
	code := plantOTP(t, h, uid, "delete")
	if got := h.POST("/api/v1/auth/account/delete/confirm", tokA,
		map[string]any{"code": code}); got.Code >= 400 {
		t.Fatalf("SEC7 تعذّر تأكيدُ الحذف: %s", got)
	}

	a := secRead(t, h, tokA, sidA, devA)
	b := secRead(t, h, tokB, sidB, devB)
	t.Logf("SEC7 أ=%+v · ب=%+v", a, b)

	for name, v := range map[string]secVerdict{"أ": a, "ب": b} {
		if v.access {
			t.Errorf("SEC7 **المحذوفُ ما زال يمرّ بعائلته %s**", name)
		}
		if v.live != 0 {
			t.Errorf("SEC7 **صفُّ تجديدٍ حيٌّ لحسابٍ محذوف في %s**: %d", name, v.live)
		}
		if v.push {
			t.Errorf("SEC7 **وجهةُ %s تبقى لحسابٍ حُذف بطلب صاحبه**", name)
		}
	}

	// **ولا يُبعَث بالتجديد** — **وهو الشقُّ الذي يسأل عنه العقد.**
	var status string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT status FROM users WHERE id = $1::uuid`, uid).Scan(&status); err != nil {
		t.Fatalf("حالُ الحساب: %v", err)
	}
	if status != "deleted" {
		t.Errorf("SEC7 الحالُ %q — والمنتظَرُ محذوفاً", status)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **SEC4 · تبديلُ المرء كلمتَه — عائلتُه تبقى بوجهتها**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو عقدُ `XG-40` حرفاً**: **تبقى العائلةُ التي نفّذت التغيير وتُقطَع
// البواقي** — **ووجهةُ الباقيةِ تبقى معها.**
//
// **والحدُّ مقصود**: **من حذف الوجهاتِ كلَّها هنا أسكت إشعاراتِ من لم
// يُخرَج** — وهو عكسُ ما طُلب.
func TestSEC4_SelfPasswordChangeKeepsItsOwnDestination(t *testing.T) {
	h := New(t)
	uid, tokA, sidA, devA, tokB, sidB, devB := secTwoFamilies(t, h, "customer")

	// **وكلمةٌ قائمةٌ تُثبَت أوّلاً** — فالتبديلُ يسأل عن الحاليّة.
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT 1 FROM users WHERE id = $1::uuid`, uid).Scan(new(int)); err != nil {
		t.Fatalf("الحساب: %v", err)
	}
	if got := h.POST("/api/v1/auth/password", tokA,
		map[string]any{"password": "Qa!Sec4-2026"}); got.Code >= 400 {
		t.Fatalf("SEC4 تعذّر التبديل: %s", got)
	}

	a := secRead(t, h, tokA, sidA, devA)
	b := secRead(t, h, tokB, sidB, devB)
	t.Logf("SEC4 المنفِّذة=%+v · الأخرى=%+v", a, b)

	if !a.access || a.live != 1 {
		t.Errorf("SEC4 **العائلةُ المنفِّذةُ قُطعت** — وعقدُ `XG-40` يُبقيها: %+v", a)
	}
	if !a.push {
		t.Errorf("SEC4 **وجهةُ العائلة الباقيةِ حُذفت** — " +
			"**فمن بدّل كلمتَه أُسكتت إشعاراتُه وهو لم يُخرَج.**")
	}
	if b.access || b.live != 0 {
		t.Errorf("SEC4 **العائلةُ الأخرى بقيت**: %+v", b)
	}
	if b.push {
		t.Errorf("SEC4 **وجهةُ العائلة المقطوعةِ تبقى** — " +
			"**وجلسةُ المتطفّل قُطعت وجهازُه يستقبل.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **SEC8 · الاستعادةُ الذاتيّةُ برمز — وحدُّها مقيسٌ لا مفترض**
// ══════════════════════════════════════════════════════════════════════
//
// **والمسارُ غيرُ مسار الإدارة**: `ConfirmPasswordReset` تكتب الكلمةَ
// **ثمّ تُصدر جلسةً** — **و`issueFor` تُبطل عائلاتِ نوعِ العميل نفسِه
// وحدَها** (`revokeClientSessions`، هجرة `0099`).
//
// **وهذا الفحصُ يقيس الحدَّ ويُثبّته** — **فمن وسّعه أو ضيّقه غداً
// يُنبَّه**، **ولا يُخترَع عقدٌ ثالثٌ للاستعادة.**
func TestSEC8_SelfServiceResetScopeIsMeasured(t *testing.T) {
	h := New(t)
	uid, tokA, sidA, devA, tokB, sidB, devB := secTwoFamilies(t, h, "customer")
	var phone string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT phone FROM users WHERE id = $1::uuid`, uid).Scan(&phone); err != nil {
		t.Fatalf("هاتفُ الحساب: %v", err)
	}

	code := plantOTP(t, h, uid, "reset")
	got := h.POST("/api/v1/auth/password/reset/confirm", "", map[string]any{
		"phone": phone, "code": code, "password": "Qa!Sec8-2026"})
	if got.Code >= 400 {
		t.Fatalf("SEC8 تعذّرت الاستعادة: %s", got)
	}

	a := secRead(t, h, tokA, sidA, devA)
	b := secRead(t, h, tokB, sidB, devB)
	t.Logf("SEC8 عميلٌ=%+v · ويبٌ=%+v", a, b)

	// **والعائلاتُ كلُّها تُقطَع** — **والاستعادةُ بابُ من فقد حسابَه،
	// فلا تُترَك للمتطفّل عائلةٌ من نوعٍ آخر.** (`R13` · `F-30`.)
	for name, v := range map[string]secVerdict{"العميل": a, "الويب": b} {
		if v.access || v.live != 0 {
			t.Errorf("SEC8 **عائلةُ %s بقيت بعد استعادةٍ ذاتيّة**: %+v", name, v)
		}
		if v.push {
			t.Errorf("SEC8 **وجهةُ %s تبقى بعد استعادةٍ ذاتيّة**", name)
		}
	}
}
