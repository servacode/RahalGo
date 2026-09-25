package server

import (
	"os"
	"strings"
	"testing"
)

// TestQAReferralWitnessCapabilityGuards **حرّاسُ قدرةِ شاهدِ الإحالة الحيّ**
// (Batch 4). **بنيةُ اختبارٍ ضيّقةٌ يجب ألّا تصير باباً واسعاً** — تُقاس من
// المصدرِ فلا تُنسى أو يتّسع نطاقُها في تعديلٍ لاحق. **وهي تمسّ المالَ**
// (مكافآتٌ ومحفظة)، فحراستُها أشدّ.
func TestQAReferralWitnessCapabilityGuards(t *testing.T) {
	src, err := os.ReadFile("qa_batch4_witness.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	s := string(src)

	// 1) حارسُ التجهيز — يسقط مغلقاً في الإنتاج (حارسٌ ثانٍ بعد المنادي).
	if !strings.Contains(s, "if !s.qaStagingEnabled() {") {
		t.Error("**حارسُ qaStagingEnabled غائب**")
	}
	// 2) هويّاتٌ ثابتةٌ وحدَها — لا معرّفَ مستخدمٍ من الطلب.
	if !strings.Contains(s, `qaRefInviterPhone = "+963900556000"`) {
		t.Error("**هاتفُ الداعي الثابتُ غائبٌ أو تغيّر**")
	}
	if strings.Contains(s, "chi.URLParam") ||
		strings.Contains(s, `Query().Get(`) ||
		strings.Contains(s, "req.UserID") || strings.Contains(s, "req.OrderID") ||
		strings.Contains(s, "req.ZoneID") {
		t.Error("**القدرةُ تقبل معرّفاً/إعداداً من الطلب — ممنوع**")
	}
	// 3) يمرّ بمسارِ الإحالةِ الحقيقيّ — لا حقنَ رصيدٍ بديلاً عن الصرف.
	for _, m := range []string{"s.referrals.Attach(", "s.referrals.SettleOnSignup(", "s.referrals.MyCode(", "s.wallet.Balance("} {
		if !strings.Contains(s, m) {
			t.Errorf("**لا يستدعي المسارَ الحقيقيَّ %s**", m)
		}
	}
	// 4) لا حقنَ رصيدٍ خامٍّ: لا UPDATE/INSERT على المحافظِ أو القيود.
	if strings.Contains(s, "INSERT INTO wallet") || strings.Contains(s, "UPDATE wallets") ||
		strings.Contains(s, "UPDATE wallet_transactions") {
		t.Error("**حقنُ رصيدٍ خامٌّ على المحافظِ — يجب أن يمرّ بالصرفِ الحقيقيّ**")
	}
	// 5) العكسُ متوازنٌ — قيدٌ مزدوجٌ (الداعي ↔ الخزينة) لا طرفٌ واحد.
	if !strings.Contains(s, "is_treasury") || strings.Count(s, "s.wallet.ApplyTxID(") < 2 {
		t.Error("**عكسُ المصروفِ ليس قيداً مزدوجاً متوازناً**")
	}
	// 6) لا يُصدِر توكن/جلسة — لا أدمن ولا سائق ولا متجر.
	if strings.Contains(s, "IssueForUserID") || strings.Contains(s, "ActiveSessionID") ||
		strings.Contains(s, "TokenFor") {
		t.Error("**القدرةُ تُصدِر جلسةً/توكن — ممنوع**")
	}
	// 7) الإعداداتُ تُحفظ وتُستعاد — لا يبقى إعدادٌ اختباريٌّ نافذاً.
	if !strings.Contains(s, "defer func()") || !strings.Contains(s, "prevInt") {
		t.Error("**لا استعادةَ مؤجّلةً للإعداداتِ السابقة**")
	}
	// 8) التنظيفُ محصورٌ في الهويّاتِ الثابتةِ — حذفٌ بمعرّفِ الداعي/المدعوّين لا كنسٌ عامّ.
	if !strings.Contains(s, "DELETE FROM referrals WHERE inviter_id = $1 OR invitee_id = ANY($2)") {
		t.Error("**تنظيفُ الإحالةِ ليس محصوراً في الهويّاتِ الثابتة**")
	}
	// 9) قيمُ المكافآتِ الخمسُ متمايزةٌ — كي يُقرأ كلُّ رتبةٍ على حدة.
	for _, v := range []string{"1000", "2000", "3000", "4000", "500"} {
		if !strings.Contains(s, v) {
			t.Errorf("**قيمةُ مكافأةٍ اختباريّةٌ متمايزةٌ غائبة: %s**", v)
		}
	}

	// ── شاهدُ «بلاغٌ ضدَّ الزبون» (report_against_customer / report_cleanup) ──
	// 10) سائقٌ ثابتٌ وحدَه — لا معرّفَ سائقٍ من الطلب.
	if !strings.Contains(s, `qaReportDriverPhone = "+963900556010"`) {
		t.Error("**هاتفُ السائقِ الثابتُ غائبٌ أو تغيّر**")
	}
	// 11) البلاغُ يمرّ بالخدمةِ الحقيقيّة — لا حقنَ تذكرةٍ خام.
	if !strings.Contains(s, "s.support.DriverReport(") {
		t.Error("**لا يمرّ البلاغُ بـ support.DriverReport الحقيقيّة**")
	}
	if strings.Contains(s, "INSERT INTO tickets") {
		t.Error("**حقنُ تذكرةٍ خامٌّ — يجب أن يمرّ بخدمةِ البلاغ**")
	}
	// 12) التجهيزُ محصورٌ في طلبٍ واحدٍ ويُستعاد — driver_id + closed_at عكوسان.
	if !strings.Contains(s, "restore()") {
		t.Error("**تجهيزُ الطلبِ (driver_id/closed_at) لا يُستعاد**")
	}
	// 13) التنظيفُ محصورٌ في تذاكرِ السائقِ الثابتِ وحدَه — لا كنسٌ عامّ.
	if !strings.Contains(s, "DELETE FROM tickets WHERE created_by = $1::uuid") {
		t.Error("**تنظيفُ البلاغِ ليس محصوراً في تذاكرِ السائقِ الثابت**")
	}
	// 14) زبونُ QA الثابتُ هو الهدفُ — لا معرّفَ زبونٍ من الطلب.
	if !strings.Contains(s, "qaStagingPhone") {
		t.Error("**هدفُ البلاغِ ليس زبونَ QA الثابت**")
	}

	// ── شاهدُ «ضاع الرد» (signup_confirm_abort) ─────────────────────────
	// 15) مقصورٌ على رقمِ QA ثابتٍ من قائمةِ OTP — لا يُسقَط ردُّ أيّ رقمٍ آخر.
	if !strings.Contains(s, "qaOTPPhones[phone]") {
		t.Error("**الإسقاطُ لا يُقصَر على أرقامِ QA المسموحة**")
	}
	// 16) مرّةً واحدةً ثمّ يُنظَّف نفسَه (يُفرَّغ الرقمُ المستهدَف).
	if !strings.Contains(s, `qaSignupAbort.phone = ""`) {
		t.Error("**الإسقاطُ ليس مرّةً واحدةً (لا يُنظَّف نفسَه)**")
	}
	// 17) الإسقاطُ اختطافُ اتصالٍ وإغلاقٌ — لا رمزَ HTTP يُقرأ نجاحاً/خطأً.
	if !strings.Contains(s, "http.Hijacker") || !strings.Contains(s, "conn.Close()") {
		t.Error("**الإسقاطُ ليس اختطافَ اتصالٍ وإغلاقاً**")
	}
	// 18) لا إنشاءَ حسابٍ خامٍّ في القدرة — المنطقُ الحقيقيّ في ConfirmSignup.
	if strings.Contains(s, "CreateCustomerWithPassword") || strings.Contains(s, "INSERT INTO users") {
		t.Error("**القدرةُ تُنشئ حساباً خامّاً — يجب أن يمرّ بـ ConfirmSignup**")
	}

	// ── الخطّافُ في المعالِج مطفأٌ في الإنتاج (fail-closed) ──────────────
	auth, err := os.ReadFile("auth_handlers.go")
	if err != nil {
		t.Fatalf("read auth_handlers.go: %v", err)
	}
	a := string(auth)
	// 19) الخطّافُ محروسٌ بـ qaStagingEnabled + مقصورٌ على رقمِ الطلب.
	if !strings.Contains(a, "s.qaStagingEnabled()") || !strings.Contains(a, "qaSignupConfirmAbortHit(req.Phone)") {
		t.Error("**خطّافُ الإسقاطِ غيرُ محروسٍ بـ qaStagingEnabled + رقمِ الطلب**")
	}
	ci := strings.Index(a, "s.identity.ConfirmSignup(")
	hi := strings.Index(a, "qaSignupConfirmAbortHit(")
	if ci < 0 || hi < 0 || hi < ci {
		t.Error("**الإسقاطُ ليس بعد ConfirmSignup (post-commit)**")
	}
}
