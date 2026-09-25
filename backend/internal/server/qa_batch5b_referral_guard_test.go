package server

import (
	"os"
	"strings"
	"testing"
)

// TestQAReferralUIKindsRegistered — نوعا شاهدِ الإحالة من الواجهة مسجَّلان،
// وكلاهما حالةٌ (يحلّ QA1 والمدعوّين بنفسِه، لا يلزمه استخراجٌ قبليّ).
func TestQAReferralUIKindsRegistered(t *testing.T) {
	for _, k := range []string{"referral_ui_arm", "referral_ui_cleanup"} {
		if !qaSeedAllowlist[k] {
			t.Fatalf("%s must be in qaSeedAllowlist", k)
		}
		if !qaStateSeed[k] {
			t.Fatalf("%s must be a state seed", k)
		}
	}
}

// TestQAReferralUIWitnessGuards **حرّاسُ قدرةِ شاهدِ الإحالة من الواجهة**
// (Batch 5). تُقاس من المصدرِ فلا يتّسع نطاقُها في تعديلٍ لاحق.
func TestQAReferralUIWitnessGuards(t *testing.T) {
	src, err := os.ReadFile("qa_batch5b_referral.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	s := string(src)

	// 1) كلا المعالجَين يسقط مغلقاً في الإنتاج.
	if n := strings.Count(s, "if !s.qaStagingEnabled() {"); n < 2 {
		t.Errorf("**حارسُ qaStagingEnabled ناقصٌ**: وجد %d من 2", n)
	}
	// 2) العكسُ متوازنٌ عبر المسار المُعتمَد — لا حقنَ محفظةٍ خام.
	if !strings.Contains(s, "s.qaReferralReverseWallet(") {
		t.Error("**لا يعكس الرصيدَ عبر qaReferralReverseWallet المتوازن**")
	}
	if strings.Contains(s, "wallet_transactions") || strings.Contains(s, "UPDATE wallets") {
		t.Error("**حقنٌ خامٌّ على المحفظة — ممنوع، يمرّ بالمسار المتوازن**")
	}
	// 3) التجهيلُ عبر مسار الحذف الحقيقيّ — لا حذفَ صفوفِ users خام.
	for _, m := range []string{"s.identity.QAIssueDeleteCode(", "s.identity.ConfirmAccountDeletion("} {
		if !strings.Contains(s, m) {
			t.Errorf("**لا يمرّ بمسار الحذف الحقيقيّ %s**", m)
		}
	}
	if strings.Contains(s, "DELETE FROM users") {
		t.Error("**حذفٌ خامٌّ لصفوف users — ممنوع، التجهيلُ عبر ConfirmAccountDeletion**")
	}
	// 4) الرتبُ تُحفَظ وتُستعاد — لا تبقى قيمةٌ اختباريّةٌ نافذة.
	if !strings.Contains(s, "qaRefUISaved") || !strings.Contains(s, "prevRewardOn") {
		t.Error("**لا حفظَ/استعادةَ لإعداداتِ الرتبِ ووضعِ الصرف**")
	}
	// 5) الداعي QA1 الثابتُ والمدعوّون أرقامٌ محجوزة — لا معرّفَ من الطلب.
	if !strings.Contains(s, "s.qaUserIDByPhone(ctx, qaStagingPhone)") {
		t.Error("**الداعي ليس QA1 الثابت**")
	}
	if !strings.Contains(s, "qaRefInviteePhonesUI()") {
		t.Error("**المدعوّون ليسوا الأرقامَ المحجوزةَ الثابتة**")
	}
	// 6) لا يُصدِر جلسةً/توكن.
	if strings.Contains(s, "IssueForUserID") || strings.Contains(s, "ActiveSessionID") {
		t.Error("**القدرةُ تُصدِر جلسةً/توكن — ممنوع**")
	}
}
