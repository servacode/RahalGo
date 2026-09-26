package qa

// ══════════════════════════════════════════════════════════════════════
// **جسورُ شهادةِ المندوب لا وجودَ لها خارج التجهيز** — سلامةُ الإنتاج
// ══════════════════════════════════════════════════════════════════════
//
// **المسارات لا تُسجَّل إلّا حين `APP_ENV=staging` و`RAHALGO_STAGING=1`**
// (`server.go`، داخل `if s.qaStagingEnabled()`). فبيئةُ الاختبار (ليست تجهيزاً)
// يجب أن تردَّ `404` على كلِّ جسرٍ — كأنّه غيرُ موجود. **وهو الشاهدُ السالبُ:
// لا جلسةُ مندوبٍ تُصدَر، ولا فعلَ أدمنٍ يُنفَّذ، من باب الاختبار في الإنتاج.**
//
// (الوجهُ الموجَب — حضورُها على التجهيز — يُشهَد حيّاً على `staging-api` أثناء
// الـE2E، حيث `qaStagingEnabled` صحيح؛ وتُثبته وحدةُ البوّابة `qaStagingEnabled`
// وحارسُ المصدر أنّها داخلَ تلك البوّابة وحدَها.)

import "testing"

func TestRepBridges_AbsentOutsideStaging(t *testing.T) {
	h := New(t)
	fakeID := "00000000-0000-0000-0000-000000000000"

	// جلسةُ المندوب — POST /qa/rep-session
	if got := h.POST("/api/v1/qa/rep-session", "", map[string]any{"slot": "a"}); got.Code != 404 {
		t.Errorf("**/qa/rep-session ظهر خارج التجهيز** — رمز=%d (يُنتظر 404): %s", got.Code, got)
	}

	// أفعالُ الأدمن — كلُّها خلف qaAdminActor، ويجب أن تكون غيرَ مسجَّلة.
	if got := h.POST("/api/v1/qa/leads/"+fakeID+"/status", "", map[string]any{"status": "converted"}); got.Code != 404 {
		t.Errorf("**/qa/leads/{id}/status ظهر خارج التجهيز** — رمز=%d (يُنتظر 404): %s", got.Code, got)
	}
	if got := h.POST("/api/v1/qa/payouts/"+fakeID+"/decide", "", map[string]any{"status": "paid"}); got.Code != 404 {
		t.Errorf("**/qa/payouts/{id}/decide ظهر خارج التجهيز** — رمز=%d (يُنتظر 404): %s", got.Code, got)
	}
	if got := h.PATCH("/api/v1/qa/users/"+fakeID, "", map[string]any{"status": "suspended", "status_reason": "qa"}); got.Code != 404 {
		t.Errorf("**/qa/users/{id} ظهر خارج التجهيز** — رمز=%d (يُنتظر 404): %s", got.Code, got)
	}
	if got := h.PATCH("/api/v1/qa/merchants/"+fakeID, "", map[string]any{"sales_rep_code": "X", "transfer_reason": "qa"}); got.Code != 404 {
		t.Errorf("**/qa/merchants/{id} ظهر خارج التجهيز** — رمز=%d (يُنتظر 404): %s", got.Code, got)
	}
	if got := h.POST("/api/v1/qa/app-file?key=release.rep.apk", "", map[string]any{}); got.Code != 404 {
		t.Errorf("**/qa/app-file ظهر خارج التجهيز** — رمز=%d (يُنتظر 404): %s", got.Code, got)
	}
}
