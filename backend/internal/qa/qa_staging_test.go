package qa

// **بابُ جلسةِ QA لا وجودَ له خارج التجهيز** — سلامةُ الإنتاج.
//
// **المسارُ لا يُسجَّل إلّا حين `APP_ENV=staging` و`RAHALGO_STAGING=1`**
// (`server.go`)، **وحارسٌ ثانٍ في المعالِج.** فبيئةُ الاختبار (ليست
// تجهيزاً) يجب أن تردّ `404` كأنّه غيرُ موجود — لا جلسةَ تُصدَر.

import "testing"

func TestQAStaging_DisabledOutsideStaging(t *testing.T) {
	h := New(t)
	// **بلا توثيق** — المسارُ عامٌّ لو وُجد؛ والمنتظَرُ ألّا يوجد أصلاً.
	got := h.POST("/api/v1/qa/session", "", map[string]any{})
	if got.Code != 404 {
		t.Fatalf("**بابُ جلسةِ QA ظهر خارج التجهيز** — رمز=%d (يُنتظر 404): %s", got.Code, got)
	}
	// **وبذّارُ QA كذلك** — شاهدُ الإحالة (Batch 4) يمرّ به، فيجب ألّا
	// يوجد أصلاً في الإنتاج: لا صرفَ مكافأةٍ ولا مساسَ بالمحفظة.
	for _, kind := range []string{"referral_witness", "report_against_customer", "report_cleanup"} {
		seed := h.POST("/api/v1/qa/seed", "", map[string]any{"kind": kind})
		if seed.Code != 404 {
			t.Fatalf("**بذّارُ QA (%s) ظهر خارج التجهيز** — رمز=%d (يُنتظر 404): %s", kind, seed.Code, seed)
		}
	}
}
