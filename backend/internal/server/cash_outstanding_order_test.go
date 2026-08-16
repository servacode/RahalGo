package server

// **والأقدمُ أوّلاً — القِدَمُ هو الإشارة لا المقدار.**
//
// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
//
// **مكتوبٌ في رأس الشاشة**: «خمسون ألفاً قُبضت قبل ساعة عملٌ يجري، وخمسون
// ألفاً منذ أسبوعٍ مسألةٌ أخرى · **فالعمودُ الذي يُنظر إليه أوّلاً هو
// (منذ متى) لا (كم)**» — **والترتيبُ كان بالمقدار.**
//
// **فسائقٌ يحمل عشرين ألفاً منذ ثمانية أيام يهبط تحت من قبض مئتي ألفٍ قبل
// ساعة** — والأوّلُ هو المسألة والثاني عملٌ يجري. **والشاشةُ تلوّن قِدَمَه
// بالأحمر وتدفنه في الأسفل.**
//
// **وهي عائلةُ عطب الطوارئ بعينها**: الوثيقةُ تقول قاعدةً والشيفرةُ تفعل
// عكسَها.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCashOutstanding_OldestFirst(t *testing.T) {
	f := newDriverFixture(t, 2)
	ctx := context.Background()
	oldSmall, freshBig := f.drivers[0], f.drivers[1]

	// **وقاعدةُ الفحص مشتركة** — تُخلى قبل القياس، **وفحصٌ يقيس ما خلّفه
	// غيرُه لا يقيس شيئا.**
	if _, err := f.pool.Exec(ctx, `DELETE FROM driver_cash_entries`); err != nil {
		t.Fatalf("تعذّر الإخلاء: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM driver_cash_entries`)
	})

	hold := func(driver string, amount int64, daysAgo int) {
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO driver_cash_entries (driver_id, amount, kind, created_at)
			VALUES ($1, $2, 'order_collection', now() - ($3::int || ' days')::interval)`,
			driver, amount, daysAgo); err != nil {
			t.Fatalf("تعذّر القيد: %v", err)
		}
	}
	// **عشرون ألفاً منذ ثمانية أيام** — وهو المسألة.
	hold(oldSmall, 20_000, 8)
	// **ومئتا ألفٍ قبضها اليوم** — وهو عملٌ يجري.
	hold(freshBig, 200_000, 0)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	c := context.WithValue(req.Context(), ctxRoles, []string{"admin"})
	w := httptest.NewRecorder()
	f.srv.handleCashOutstanding(w, req.WithContext(c))
	if w.Code != 200 {
		t.Fatalf("ردَّ %d — %s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			Total   int64 `json:"total"`
			Holders []struct {
				DriverID string `json:"driver_id"`
				Held     int64  `json:"held"`
			} `json:"holders"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v", err)
	}

	if len(env.Data.Holders) != 2 {
		t.Fatalf("%d حاملاً لا 2", len(env.Data.Holders))
	}
	if env.Data.Holders[0].DriverID != oldSmall {
		t.Fatal("تصدّر الأكبرُ مبلغاً — **والقِدَمُ هو الإشارة لا المقدار**، " +
			"والشاشةُ تلوّن قِدَمَ الأوّل بالأحمر ثمّ تدفنه أسفلَ الثاني")
	}
	// **والمجموعُ على الكلّ** — ولا سقفَ يقصّ هنا.
	if env.Data.Total != 220_000 {
		t.Fatalf("المجموعُ %d لا 220000", env.Data.Total)
	}
}
