package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// TestReturn_ReversesMerchantEarning الإرجاعُ يستردّ ما دُفع — لا ما يُحسب.
//
// **المتجرُ قبض عند خروج البضاعة.** فإن عادت إليه **عاد معها مالُها** — فلا
// هو خسر ولا ربح. **ولو تُرك له المالُ والبضاعةُ معاً لربح من الفشل أكثرَ
// ممّا يربح من النجاح**، وهو حافزٌ مقلوب.
//
// **ويُعكس ما قُيّد فعلاً لا ما تقول المعادلة**: عمولتُه قد تكون تغيّرت منذ
// الاستلام، **وحسبةٌ جديدةٌ تعكس مبلغاً غيرَ الذي دُفع** فيبقى فرقٌ في محفظته
// بلا سبب.
func TestReturn_ReversesMerchantEarning(t *testing.T) {
	f := newAftermathFixture(t)
	orderID := f.paidThenFailed(t, 100_000)
	f.setReturns(t, true)

	if got := f.balance(t, f.owner); got != 90_000 {
		t.Fatalf("مستحقّ المتجر قبل الإرجاع = %d، والمتوقّع 90000", got)
	}
	f.returnOrder(t, orderID, 200)
	if got := f.balance(t, f.owner); got != 0 {
		t.Errorf("بعد الإرجاع = %d، والمتوقّع 0 — أخذ بضاعتَه وردّ ثمنَها", got)
	}
	// **ولا يُستردّ مرّتين.**
	f.returnOrder(t, orderID, 409)
	if got := f.balance(t, f.owner); got != 0 {
		t.Errorf("استُردّ مرّتين: %d", got)
	}
}

// TestReturn_RefusedMerchant متجرٌ لا يستردّ لا يُخصم منه.
//
// **والمنصةُ تتحمّل كاملاً** — وهي التي اشترت الطعامَ لحظةَ خروجه.
func TestReturn_RefusedMerchant(t *testing.T) {
	f := newAftermathFixture(t)
	orderID := f.paidThenFailed(t, 100_000)
	f.setReturns(t, false)

	f.returnOrder(t, orderID, 409)
	if got := f.balance(t, f.owner); got != 90_000 {
		t.Errorf("خُصم من متجرٍ لا يستردّ: %d", got)
	}
}

// ── العُدّة ──────────────────────────────────────────────────────────────

type aftermathFixture struct {
	*driverFixture
	owner string
}

func newAftermathFixture(t *testing.T) *aftermathFixture {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	walletSvc := wallet.NewService(pool)
	settingsStore := settings.NewStore(pool)
	cashboxSvc := cashbox.NewService(pool, settingsStore)

	base := &driverFixture{
		pool: pool,
		srv: &Server{
			pg:       pool,
			logger:   quiet,
			hub:      realtime.NewHub(quiet),
			cashbox:  cashboxSvc,
			wallet:   walletSvc,
			settings: settingsStore,
			orders:   orders.NewService(pool, nil, walletSvc, cashboxSvc, nil, quiet),
		},
	}
	ctx := context.Background()
	owner := testdb.NewUser(t, pool, "merchant")

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات في القاعدة: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, owner_user_id)
		VALUES ('متجر اختبار ما بعد الفشل', $1, 10, $2) RETURNING id`,
		categoryID, owner).Scan(&base.merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, base.merchantID)
	})
	base.drivers = append(base.drivers, testdb.NewUser(t, pool, "driver"))
	return &aftermathFixture{driverFixture: base, owner: owner}
}

func (f *aftermathFixture) orderWithStatus(t *testing.T, status string, subtotal int64) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
		VALUES ($1, $2, $3, $4, 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', $5, 10000, $5::bigint + 10000, 0, $5::bigint + 10000)
		RETURNING id`,
		testdb.NewUser(t, f.pool, "customer"), f.merchantID, f.drivers[0], status, subtotal).
		Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاء طلب: %v", err)
	}
	return id
}

func (f *aftermathFixture) failedOrder(t *testing.T, subtotal int64) string {
	t.Helper()
	return f.orderWithStatus(t, "failed", subtotal)
}

func (f *aftermathFixture) balance(t *testing.T, user string) int64 {
	t.Helper()
	v, err := f.srv.wallet.Balance(context.Background(), user)
	if err != nil {
		t.Fatalf("تعذّرت قراءة الرصيد: %v", err)
	}
	return v
}

// setReturns يضبط سياسةَ استرداد المتجر.
func (f *aftermathFixture) setReturns(t *testing.T, on bool) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE merchants SET accepts_returns = $2 WHERE id = $1`, f.merchantID, on); err != nil {
		t.Fatalf("تعذّر ضبط الاسترداد: %v", err)
	}
}

// paidThenFailed طلبٌ قُيّد مستحقُّ متجره ثمّ فشل — الحالُ الذي يلي الاستلام.
func (f *aftermathFixture) paidThenFailed(t *testing.T, subtotal int64) string {
	t.Helper()
	id := f.orderWithStatus(t, "failed", subtotal)
	ctx := context.Background()
	// **يُقيَّد كما يقيّده المحرّك** — بنوعه ومرجعه، فيجده الإرجاعُ ويعكسه.
	actor := f.owner
	if _, err := f.srv.wallet.Apply(ctx, f.owner, 90_000, "merchant_earning",
		id, "مستحقّ عن بضاعةٍ سُلّمت للسائق", &actor); err != nil {
		t.Fatalf("تعذّر قيدُ المستحقّ: %v", err)
	}
	return id
}

// returnOrder ينادي معالِجَ الإرجاع ويتحقّق من رمز الردّ.
func (f *aftermathFixture) returnOrder(t *testing.T, orderID string, wantStatus int) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/driver/orders/"+orderID+"/return", strings.NewReader("{}"))
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", orderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, f.drivers[0])
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})

	w := httptest.NewRecorder()
	f.srv.handleDriverReturn(w, req.WithContext(ctx))
	if w.Code != wantStatus {
		t.Fatalf("الرمز %d والمتوقّع %d — %s", w.Code, wantStatus, w.Body.String())
	}
}
