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

// TestSettleGoods_PlatformAbsorbs المتجرُ يُجبَر بما كان سيقبضه — لا أكثر.
//
// **هذا اختبارُ مبلغٍ لا اختبارُ مسار.** «المنصةُ تتحمّل كاملاً» تقبل قراءتين:
// أن يُدفع للمتجر ثمنُ البضاعة كاملاً، أو ما كان سيقبضه لو نجح الطلب. والفرقُ
// بينهما عمولةُ المنصة — **وبالقراءة الأولى يربح المتجرُ من فشلٍ أكثر ممّا
// يربح من نجاح**، وهو حافزٌ مقلوب.
//
// فالمعتمد: البضاعةُ ناقصَ العمولة، بالمعادلة نفسها التي في تسوية التسليم.
func TestSettleGoods_PlatformAbsorbs(t *testing.T) {
	f := newAftermathFixture(t)

	// 100,000 بضاعة و10% عمولة → المتجر كان سيقبض 90,000
	orderID := f.failedOrder(t, 100_000)
	f.settleGoods(t, orderID, "platform", 200)

	if got := f.balance(t, f.owner); got != 90_000 {
		t.Errorf("رصيدُ المتجر = %d، والمتوقّع 90000 (البضاعة ناقصَ العمولة)", got)
	}
	// **ولا يُدفع مرّتين**: ضغطةٌ ثانية تُردّ.
	f.settleGoods(t, orderID, "platform", 409)
	if got := f.balance(t, f.owner); got != 90_000 {
		t.Errorf("دُفع مرّتين: الرصيد = %d", got)
	}
}

// TestSettleGoods_MerchantTookBack المتجرُ استردّ بضاعته — فلا قيد.
func TestSettleGoods_MerchantTookBack(t *testing.T) {
	f := newAftermathFixture(t)
	orderID := f.failedOrder(t, 100_000)
	f.settleGoods(t, orderID, "merchant", 200)

	if got := f.balance(t, f.owner); got != 0 {
		t.Errorf("رصيدُ المتجر = %d، والمتوقّع 0 — أخذ بضاعته فلا مالَ له", got)
	}
}

// TestSettleGoods_OnlyFailedOrders التسويةُ لطلبٍ فشل لا لغيره.
//
// **وإلّا صارت باباً خلفياً**: تسويةُ بضاعةٍ على طلبٍ مُسلَّم تدفع للمتجر مرّةً
// ثانيةً عن بيعةٍ قُبضت.
func TestSettleGoods_OnlyFailedOrders(t *testing.T) {
	f := newAftermathFixture(t)
	orderID := f.orderWithStatus(t, "delivered", 100_000)
	f.settleGoods(t, orderID, "platform", 409)

	if got := f.balance(t, f.owner); got != 0 {
		t.Errorf("دُفع عن طلبٍ مُسلَّم: الرصيد = %d", got)
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

// settleGoods ينادي المعالِج الحقيقي ويتحقّق من رمز الردّ.
func (f *aftermathFixture) settleGoods(t *testing.T, orderID, to string, wantStatus int) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/admin/orders/"+orderID+"/settle-goods",
		strings.NewReader(`{"to":"`+to+`"}`))
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", orderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, f.owner)
	ctx = context.WithValue(ctx, ctxRoles, []string{"ops"})

	w := httptest.NewRecorder()
	f.srv.handleSettleGoods(w, req.WithContext(ctx))
	if w.Code != wantStatus {
		t.Fatalf("الرمز %d والمتوقّع %d — %s", w.Code, wantStatus, w.Body.String())
	}
}
