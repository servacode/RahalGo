package qa

import "testing"

// TestDriverLandmarksToday **معالمُه اليوم من المطلوب** (قرارُ المالك ٢٠٢٦-١١-٠٥) — سطرُ «معالمك
// اليوم: ٢ من ٥» فوق صفحة «أضف معلماً»، **والمكرَّرُ لا يزيده.**
func TestDriverLandmarksToday(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	get := func() (int, int) {
		t.Helper()
		res := h.GET("/api/v1/driver/landmarks/today", drv.Token)
		n, okN := res.JSON()["today"].(float64)
		q, okQ := res.JSON()["quota"].(float64)
		if res.Code != 200 || !okN || !okQ {
			t.Fatalf("**لا عدَّ لمعالم اليوم في تطبيق السائق**: %s", res)
		}
		return int(n), int(q)
	}
	if n, q := get(); n != 0 || q != 5 {
		t.Fatalf("قبل الإرسال: %d من %d", n, q)
	}
	name := uniq("جامع العدّ ")
	if r := lmPost(h, drv.Token, name, "mosque", 34.7310, 40.3310); r.Code != 200 {
		t.Fatalf("المعلم: %s", r)
	}
	if r := lmPost(h, drv.Token, name, "mosque", 34.7310, 40.3310); r.Code != 409 {
		t.Fatalf("المكرَّر: %s", r)
	}
	if n, q := get(); n != 1 || q != 5 {
		t.Fatalf("بعد جديدٍ ومكرَّر: %d من %d والمنتظَر ١ من ٥", n, q)
	}
	// **ولغير السائق لا باب.**
	if res := h.GET("/api/v1/driver/landmarks/today", h.NewUser("customer").Token); res.Code != 403 {
		t.Fatalf("زبونٌ قرأ عدَّ معالم السائق: %s", res)
	}
}
