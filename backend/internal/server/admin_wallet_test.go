package server

// **القيدُ الإداريُّ المباشر — أقصرُ طريقٍ بين قرارٍ ومال.**
//
// (فحصُ التغطية ٢٠٢٦-٠٨-٠٨، بقرار المالك: «نعم ابدأ بالخمسة».)
//
// لا طلبَ ولا شكوى ولا نزاع: **رقمٌ ونوعٌ وضغطة.** فما يحرسه هو كلُّ ما
// يفصل بين تسويةٍ وخطأٍ لا يُستدرَك.
//
//	النوع      ← أربعةٌ لا غير، وما عداها يُرفض
//	المبلغ     ← موجبٌ دائماً، والاتّجاهُ من النوع لا من إشارة الرقم
//	السحبُ     ← `payout` يخصم ولو كُتب موجباً
//	التسوية    ← `adjustment` تودع، و`debit` تجعلها خصماً
//	المعرّف    ← ليس UUID فلا يُقيَّد شيء
//
// **والرصيدُ لا يهبط تحت الصفر** — قيدُ القاعدة يمنعه، والخطأُ يُترجَم.

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type adminWalletFixture struct {
	pool  *pgxpool.Pool
	srv   *Server
	admin string
}

func newAdminWalletFixture(t *testing.T) *adminWalletFixture {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := realtime.NewHub(quiet)
	return &adminWalletFixture{
		pool: pool,
		srv: &Server{
			pg:       pool,
			logger:   quiet,
			hub:      hub,
			wallet:   wallet.NewService(pool),
			settings: settings.NewStore(pool),
			notify:   notifications.New(pool, hub, quiet),
		},
		admin: testdb.NewUser(t, pool, "admin"),
	}
}

func (f *adminWalletFixture) apply(target, kind string, amount int64, debit bool) int {
	body := `{"amount":` + itoa64(amount) + `,"kind":"` + kind + `","debit":` +
		map[bool]string{true: "true", false: "false"}[debit] + `,"note":"تسويةُ اختبار"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+target+"/wallet",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", target)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, f.admin)
	ctx = context.WithValue(ctx, ctxRoles, []string{"admin", "finance"})
	w := httptest.NewRecorder()
	f.srv.handleAdminWalletApply(w, req.WithContext(ctx))
	return w.Code
}

func (f *adminWalletFixture) balance(t *testing.T, uid string) int64 {
	t.Helper()
	var b int64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1`, uid).Scan(&b); err != nil {
		t.Fatalf("تعذّرت قراءةُ الرصيد: %v", err)
	}
	return b
}

func TestAdminWalletApplyGuardsAndDirections(t *testing.T) {
	f := newAdminWalletFixture(t)

	t.Run("النوعُ من قائمةٍ مغلقة", func(t *testing.T) {
		uid := testdb.NewUser(t, f.pool, "driver")
		for _, kind := range []string{"order_payment", "commission", "refund", "", "TOPUP", "drop"} {
			if code := f.apply(uid, kind, 1000, false); code < 400 {
				t.Fatalf("قُبل النوعُ %q برمز %d — **والقائمةُ أربعةٌ لا غير.**", kind, code)
			}
		}
		if b := f.balance(t, uid); b != 0 {
			t.Fatalf("تحرّك الرصيدُ %d بأنواعٍ مرفوضة", b)
		}
	})

	// **ورصيدٌ قائمٌ قبل التجربة عمداً.**
	//
	// برصيدٍ صفرٍ يسقط السالبُ على قيد القاعدة فيُقرأ «مُنع» — **وهو ممنوعٌ
	// بغير ما نظنّ.** وبرصيدٍ قائمٍ يظهر الفرق: بلا الفحص **يصير الشحنُ
	// السالبُ خصماً صامتاً.**
	t.Run("لا صفرَ ولا سالب", func(t *testing.T) {
		uid := testdb.NewUser(t, f.pool, "driver")
		if code := f.apply(uid, "topup", 20000, false); code != http.StatusOK {
			t.Fatalf("رُفض الشحنُ التمهيديُّ برمز %d", code)
		}
		for _, amount := range []int64{0, -5000} {
			if code := f.apply(uid, "topup", amount, false); code < 400 {
				t.Fatalf("قُبل مبلغُ %d برمز %d — **وشحنٌ سالبٌ خصمٌ صامت.**", amount, code)
			}
		}
		if b := f.balance(t, uid); b != 20000 {
			t.Fatalf("تبدّل الرصيدُ إلى %d بمبلغٍ غيرِ صالح — والمنتظَر 20000", b)
		}
	})

	t.Run("معرّفٌ ليس UUID لا يُقيَّد", func(t *testing.T) {
		if code := f.apply("not-a-uuid", "topup", 1000, false); code != http.StatusNotFound {
			t.Fatalf("رمزٌ %d لمعرّفٍ غيرِ صالح — والمنتظَر ٤٠٤", code)
		}
	})

	// **والاتّجاهُ من النوع لا من إشارة الرقم** — والرقمُ موجبٌ دائماً.
	t.Run("الشحنُ يودع والسحبُ يخصم", func(t *testing.T) {
		uid := testdb.NewUser(t, f.pool, "driver")
		if code := f.apply(uid, "topup", 20000, false); code != http.StatusOK {
			t.Fatalf("رُفض الشحنُ برمز %d", code)
		}
		if b := f.balance(t, uid); b != 20000 {
			t.Fatalf("بعد الشحن: %d — والمنتظَر 20000", b)
		}
		if code := f.apply(uid, "payout", 5000, false); code != http.StatusOK {
			t.Fatalf("رُفض السحبُ برمز %d", code)
		}
		if b := f.balance(t, uid); b != 15000 {
			t.Fatalf("بعد السحب: %d — والمنتظَر 15000. **و`payout` يخصم ولو كُتب موجباً.**", b)
		}
	})

	t.Run("والتسويةُ تودع أو تخصم بعلامتها", func(t *testing.T) {
		uid := testdb.NewUser(t, f.pool, "driver")
		_ = f.apply(uid, "topup", 10000, false)
		if code := f.apply(uid, "adjustment", 3000, false); code != http.StatusOK {
			t.Fatalf("رُفضت تسويةُ الإيداع برمز %d", code)
		}
		if b := f.balance(t, uid); b != 13000 {
			t.Fatalf("بعد تسوية الإيداع: %d — والمنتظَر 13000", b)
		}
		if code := f.apply(uid, "adjustment", 3000, true); code != http.StatusOK {
			t.Fatalf("رُفضت تسويةُ الخصم برمز %d", code)
		}
		if b := f.balance(t, uid); b != 10000 {
			t.Fatalf("بعد تسوية الخصم: %d — والمنتظَر 10000", b)
		}
	})

	// **ولا يهبط الرصيدُ تحت الصفر** — قيدُ القاعدة يمنعه لا فحصٌ في الشيفرة.
	t.Run("لا رصيدَ سالب", func(t *testing.T) {
		uid := testdb.NewUser(t, f.pool, "driver")
		_ = f.apply(uid, "topup", 5000, false)
		if code := f.apply(uid, "payout", 9000, false); code < 400 {
			t.Fatalf("قُبل سحبٌ يتجاوز الرصيدَ برمز %d — **وهو مالٌ لا وجودَ له.**", code)
		}
		if b := f.balance(t, uid); b != 5000 {
			t.Fatalf("تبدّل الرصيدُ إلى %d بعد سحبٍ مرفوض", b)
		}
	})
}
