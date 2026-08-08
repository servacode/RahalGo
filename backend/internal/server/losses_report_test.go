package server

// **تقريرُ الخسائر لم يعمل قطّ.**
//
// (كشفه فحصٌ شاملٌ للمنصّة ٢٠٢٦-٠٨-٠٨ بطلب المالك.)
//
// # المرض
//
// الاستعلامُ يصل جدولَ المحافظ بجدول الحركات على **عمودين لا وجودَ لهما**:
// جدولُ المحافظ مفتاحُه صاحبُها، وجدولُ الحركات يحمل صاحبَها كذلك — **ولا
// معرّفَ محفظةٍ في أيٍّ منهما.**
//
// **فيسقط قبل أن يقرأ صفّاً**: الصفحةُ تردّ ٥٠٠ في كلّ مرّة، ومنذ كُتبت.
//
// # ولماذا لم يظهر
//
// **لا اختبارَ يفتح الصفحة**، ولا حارسَ يقارن الاستعلاماتِ بالمخطَّط.
// **وخطأُ عمودٍ لا يظهر عند البناء** — يظهر عند أوّل استعلامٍ حقيقيّ،
// وحزمةُ `server` كانت عند ٧٫٧٪ تغطية.
//
// **والخطأُ لا يُقرأ في الشاشة**: الردُّ «حدث خطأ غير متوقّع» — فمن فتحها
// ظنَّها بطءَ شبكةٍ وأعاد المحاولة.

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func TestLossesReportRuns(t *testing.T) {
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := realtime.NewHub(quiet)
	srv := &Server{
		pg:       pool,
		logger:   quiet,
		hub:      hub,
		wallet:   wallet.NewService(pool),
		settings: settings.NewStore(pool),
	}
	admin := testdb.NewUser(t, pool, "admin")

	req := httptest.NewRequest(http.MethodGet, "/admin/reports/losses?from=2020-01-01&to=2030-01-01", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, admin)
	ctx = context.WithValue(ctx, ctxRoles, []string{"admin", "finance"})
	w := httptest.NewRecorder()
	srv.handlePlatformLosses(w, req.WithContext(ctx))

	if w.Code != http.StatusOK {
		t.Fatalf("تقريرُ الخسائر ردّ %d — **واستعلامٌ يشير إلى عمودٍ لا وجودَ له "+
			"يسقط قبل أن يقرأ صفّاً.** الجسم: %s", w.Code, w.Body.String())
	}
}
