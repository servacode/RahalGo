package server

// **موافقتان متزامنتان على سحبٍ واحد.**
//
// (فحصُ المشروع ٢٠٢٦-٠٨-٠٧، بقرار المالك: «نبدأ إذاً».)
//
// # المرض
//
// قرارُ السحب كان ثلاث خطواتٍ بلا معاملةٍ ولا قفل:
//
//	SELECT status FROM payout_requests WHERE id = $1   ← بلا FOR UPDATE
//	if status != "pending" { رفض }
//	wallet.Apply(-amount)                              ← معاملتُها الخاصّة
//	UPDATE payout_requests SET status = ...            ← معاملةٌ ثالثة
//
// **فموافقتان تقرآن «معلّق» كلتاهما وتمرّان الفحصَ كلتاهما** — فيُخصَم
// الرصيدُ مرّتين ويُصرف السحبُ مرّتين. **وقيدُ `balance >= 0` لا يوقفه إن
// كان الرصيدُ كافياً.**
//
// **ولا يظهر في أيّ سجلّ**: كلا النداءين يردّ ٢٠٠، وكلا القيدين صالح.
//
// # ولماذا اختبارُ تزامنٍ لا مراجعة
//
// **العينُ لا ترى سباقاً في شيفرةٍ كلُّ سطرٍ فيها صحيح.** والعطبُ في
// المسافة بين السطرين، لا في سطر.

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func TestDecidePayoutIsNotAppliedTwice(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	walletSvc := wallet.NewService(pool)
	srv := &Server{
		pg:       pool,
		logger:   quiet,
		hub:      realtime.NewHub(quiet),
		wallet:   walletSvc,
		settings: settings.NewStore(pool),
	}

	user := testdb.NewUser(t, pool, "driver")
	admin := testdb.NewUser(t, pool, "admin")

	// رصيدٌ يكفي الخصمَ مرّتين — **فلو كان بالكاد لَستر قيدُ القاعدة العطب.**
	const amount int64 = 50_000
	if _, err := walletSvc.Apply(ctx, user, 20*amount, "topup", "", "رصيدٌ للاختبار", nil); err != nil {
		t.Fatalf("تعذّر شحنُ الرصيد: %v", err)
	}

	var payoutID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO payout_requests (user_id, amount, status)
		VALUES ($1, $2, 'pending') RETURNING id`, user, amount).Scan(&payoutID); err != nil {
		t.Fatalf("تعذّر إنشاء طلب سحب: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM payout_requests WHERE id = $1`, payoutID)
	})

	before := balanceOf(t, pool, user)

	// **نداءان يبدآن معاً** — والحاجزُ يضمن أنّهما يقرآن الحالةَ قبل أن
	// يكتب أحدُهما.
	const n = 8
	var start sync.WaitGroup
	var done sync.WaitGroup
	start.Add(1)
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		done.Add(1)
		go func(i int) {
			defer done.Done()
			start.Wait()
			codes[i] = decidePayout(srv, admin, payoutID)
		}(i)
	}
	start.Done()
	done.Wait()

	after := balanceOf(t, pool, user)
	moved := before - after

	var applied int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'payout'`, payoutID).Scan(&applied); err != nil {
		t.Fatalf("تعذّر عدُّ القيود: %v", err)
	}

	if applied != 1 || moved != amount {
		t.Fatalf("**السحبُ صُرف %d مرّةً وخُصم %d** — والمنتظَر مرّةً واحدةً و%d.\n"+
			"ردودُ النداءين: %v\n"+
			"**والعلاجُ معاملةٌ واحدةٌ تلفّ القراءةَ والقيدَ والحالة، وقفلٌ "+
			"`FOR UPDATE` على صفّ الطلب.**",
			applied, moved, amount, codes)
	}

	// **وحالةُ الطلب تتبع المال** — لا تسبقه ولا تتخلّف عنه.
	var status string
	if err := pool.QueryRow(ctx,
		`SELECT status FROM payout_requests WHERE id = $1`, payoutID).Scan(&status); err != nil {
		t.Fatalf("تعذّرت قراءةُ الحالة: %v", err)
	}
	if status != "paid" {
		t.Fatalf("الحالةُ %q والمالُ خرج — **قيدٌ بلا أثرٍ في الطلب.**", status)
	}
}

func decidePayout(srv *Server, adminID, payoutID string) int {
	req := httptest.NewRequest(http.MethodPost, "/admin/payouts/"+payoutID+"/decide",
		strings.NewReader(`{"status":"paid","decision":"موافقة"}`))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", payoutID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, adminID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"admin", "finance"})
	w := httptest.NewRecorder()
	srv.handleDecidePayout(w, req.WithContext(ctx))
	return w.Code
}

func balanceOf(t *testing.T, pool *pgxpool.Pool, userID string) int64 {
	t.Helper()
	var b int64
	if err := pool.QueryRow(context.Background(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1`, userID).Scan(&b); err != nil {
		t.Fatalf("تعذّرت قراءةُ الرصيد: %v", err)
	}
	return b
}
