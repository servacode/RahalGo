package server

// **الطارئُ يُقرأ بالأقدم · ويُغلق بكلمة · ويبقى بعد إغلاقه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٦ بعد فحصٍ طلبه.)
//
// # ثلاثةٌ تناقض ما كُتب في الشاشة نفسِها
//
// **«من سأل عنه بعد يومين لم يجد من يقول ماذا جرى»** — مكتوبةٌ في رأس
// الصفحة، **والشاشةُ تعرض المفتوحَ وحدَه**: فما إن يُغلق حتّى يختفي.
//
// **والترتيبُ كان بالأحدث** — فسائقٌ ضغط الزرَّ قبل أربعين دقيقةً **يهبط
// تحت من ضغطه الآن**. **وطابورُ مراجعة الأصناف يرتّب بالأقدم**، وصنفٌ
// ينتظر أهونُ من إنسانٍ ينتظر.
//
// **والسجلُّ كان يحفظ من أغلق ومتى ولا يحفظ ماذا فعل** — فبعد يومين تجد
// «أُغلقت بيد فلان» **ولا تجد ماذا جرى.**

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestEmergencies_OldestFirstResolvedTabAndResolution(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	driver := f.drivers[0]

	mk := func(note string, minutesAgo int) string {
		var id string
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO driver_emergencies (driver_id, note, created_at)
			VALUES ($1, $2, now() - ($3::int || ' minutes')::interval)
			RETURNING id::text`, driver, note, minutesAgo).Scan(&id); err != nil {
			t.Fatalf("تعذّر البلاغُ %q: %v", note, err)
		}
		return id
	}
	// **وقاعدةُ الفحص مشتركةٌ تتراكم فيها بقايا الجولات** — والعدُّ عامٌّ
	// لا لسائقٍ بعينه، **فتُخلى الطاولةُ قبل القياس**: **فحصٌ يقيس ما
	// خلّفه غيرُه لا يقيس شيئا.**
	if _, err := f.pool.Exec(ctx, `DELETE FROM driver_emergencies`); err != nil {
		t.Fatalf("تعذّر الإخلاء: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM driver_emergencies`)
	})
	// **الأقدمُ ينتظر أربعين دقيقة** — وهو من يجب أن يتصدّر.
	old := mk("أنتظر منذ أربعين دقيقة", 40)
	mk("ضغطتُ الآن", 1)

	type res struct {
		Data struct {
			Total       int `json:"total"`
			Emergencies []struct {
				ID         string `json:"id"`
				Note       string `json:"note"`
				Resolution string `json:"resolution"`
				ResolvedBy string `json:"resolved_by"`
			} `json:"emergencies"`
		} `json:"data"`
	}
	list := func(status string) res {
		req := httptest.NewRequest(http.MethodGet, "/x?status="+status, nil)
		c := context.WithValue(req.Context(), ctxUserID, driver)
		c = context.WithValue(c, ctxRoles, []string{"admin"})
		w := httptest.NewRecorder()
		f.srv.handleOpenEmergencies(w, req.WithContext(c))
		if w.Code != 200 {
			t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
		}
		var out res
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("ردٌّ لا يُفكّ: %v", err)
		}
		return out
	}

	// ── ١ · الأقدمُ أوّلاً ────────────────────────────────────────────
	open := list("open")
	if open.Data.Total != 2 {
		t.Fatalf("المفتوحُ %d لا 2", open.Data.Total)
	}
	if open.Data.Emergencies[0].ID != old {
		t.Fatalf("تصدّر الأحدثُ — **فمن ينتظر أربعين دقيقةً يهبط تحت من ضغط الآن**")
	}

	// ── ٢ · ويُغلق بكلمة ─────────────────────────────────────────────
	body := strings.NewReader(`{"resolution":"اتّصلنا به وأوصل الطلبَ سائقٌ آخر"}`)
	req := httptest.NewRequest(http.MethodPost, "/x", body)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", old)
	c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	c = context.WithValue(c, ctxUserID, driver)
	c = context.WithValue(c, ctxRoles, []string{"admin"})
	w := httptest.NewRecorder()
	f.srv.handleResolveEmergency(w, req.WithContext(c))
	if w.Code != 200 {
		t.Fatalf("تعذّر الإغلاق: %d — %s", w.Code, w.Body.String())
	}

	// ── ٣ · ولا يختفي — بل ينتقل ─────────────────────────────────────
	//
	// **وهو ما وُعدت به الشاشةُ ولم تفِ**: «من سأل بعد يومين».
	if again := list("open"); again.Data.Total != 1 {
		t.Fatalf("المفتوحُ بعد الإغلاق %d لا 1", again.Data.Total)
	}
	done := list("resolved")
	if done.Data.Total != 1 || done.Data.Emergencies[0].ID != old {
		t.Fatalf("المُعالَجُ %d — **فما إن يُغلق حتّى يختفي**", done.Data.Total)
	}
	// **وماذا جرى مكتوبٌ** — **وبلاغٌ يُقال «أُغلق» ولا يُقال كيف يُنسى.**
	if !strings.Contains(done.Data.Emergencies[0].Resolution, "سائقٌ آخر") {
		t.Fatalf("لا وصفَ للإغلاق: %q — **فتجد «أُغلقت بيد فلان» ولا تجد ماذا جرى**",
			done.Data.Emergencies[0].Resolution)
	}
	if done.Data.Emergencies[0].ResolvedBy == "" {
		t.Fatal("لا اسمَ لمن أغلق")
	}

	// **ولا يُغلق مرّتين** — والثانيةُ تردّ «غيرُ موجود» لا تُصمت.
	w2 := httptest.NewRecorder()
	f.srv.handleResolveEmergency(w2, req.WithContext(c))
	if w2.Code == 200 {
		t.Fatal("أُغلق بلاغٌ مُغلَقٌ مرّةً ثانية")
	}
}
