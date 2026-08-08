package server

// **طلبُ انضمامٍ حُوِّل إلى متجرٍ لا يُردّ بعدها.**
//
// (شهده المالك ٢٠٢٦-٠٨-٠٨: «لا يجوز أن يبقى زرُّ الرفض بعد قبول».)
//
// # المرض
//
// `handleAdminLeadStatus` تكتب الحالةَ بلا شرطٍ على ما كانت:
//
//	UPDATE merchant_leads SET status = 'rejected' ... WHERE id = $1
//
// **ولا ذكرَ لحالتها السابقة.** فطلبٌ حُوِّل — **والمتجرُ قائمٌ يبيع
// وصاحبُه يدخل بوّابتَه** — يُوسَم «مرفوضاً» بضغطة.
//
// # وأثرُه يخرج من الشاشة
//
// **يُرسَل إشعارٌ إلى المندوب**: «رُدّ طلبُ الانضمام — طيف» مع سببِ ردٍّ
// كتبه أحد. **فيقرأ أنّ فرصتَه ضاعت** وهي لم تضِع: المتجرُ يعمل، والنسبةُ
// له، **ويُحسب في هدفه الشهريّ.**
//
// **ويُخصَم من عدّ «سُجّل» في لوحته** — فيظنّ نفسَه أقلَّ إنجازاً ممّا هو.
//
// # ولا يُصلحه إخفاءُ الزرّ وحدَه
//
// **الشاشةُ تمنع اليدَ والخادمُ يمنع الفعل.** ومن نادى المسارَ مباشرةً —
// أو فتح لوحتين وضغط في القديمة قبل أن تُحدَّث — **مرّ.**

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestLeadRejectAfterConvertIsRefused(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := realtime.NewHub(quiet)

	srv := &Server{
		pg:     pool,
		logger: quiet,
		hub:    hub,
		notify: notifications.New(pool, hub, quiet),
	}

	admin := testdb.NewUser(t, pool, "admin")
	rep := testdb.NewUser(t, pool, "sales")

	var leadID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchant_leads (store_name, owner_name, phone, sales_rep_user_id, status)
		VALUES ('متجرُ اختبارِ الردّ', 'صاحبُه', '+963900000911', $1, 'converted')
		RETURNING id`, rep).Scan(&leadID); err != nil {
		t.Fatalf("تعذّر إنشاء طلبِ انضمامٍ محوَّل: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchant_leads WHERE id = $1`, leadID)
	})

	// **وإشعاراتُ المندوب تُعَدّ قبل وبعد** — فالضررُ الذي يخرج من الشاشة
	// إشعارٌ يقرؤه إنسان.
	countNotifs := func() int {
		var n int
		_ = pool.QueryRow(context.Background(),
			`SELECT count(*) FROM notifications WHERE user_id = $1`, rep).Scan(&n)
		return n
	}
	before := countNotifs()

	req := httptest.NewRequest(http.MethodPost, "/admin/leads/"+leadID+"/status",
		strings.NewReader(`{"status":"rejected","note":"ردٌّ بعد التحويل"}`))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", leadID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	req = req.WithContext(context.WithValue(req.Context(), ctxUserID, admin))

	rec := httptest.NewRecorder()
	srv.handleAdminLeadStatus(rec, req)

	if rec.Code < 400 {
		t.Errorf("**قُبل ردُّ طلبٍ حُوِّل** — والمتجرُ قائم. الردّ %d", rec.Code)
	}

	var status string
	if err := pool.QueryRow(ctx,
		`SELECT status FROM merchant_leads WHERE id = $1`, leadID).Scan(&status); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if status != "converted" {
		t.Errorf("**الحالةُ صارت %q** — وطلبٌ حُوّل يبقى محوَّلاً", status)
	}

	if after := countNotifs(); after != before {
		t.Errorf("**أُرسل إشعارٌ للمندوب أنّ فرصتَه رُدَّت** — وهي لم تُردّ (%d ← %d)", before, after)
	}
}
