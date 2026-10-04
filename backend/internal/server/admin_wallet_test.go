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
//
// **ومنذ ٢٠٢٦-١٠-٠٤ صارت الحركةُ طلباً** يقترحه موظّفٌ ويوافق عليه غيرُه — فـ`apply`
// هنا خطوتان: اقتراحٌ ثمّ موافقة، **والقيدُ عند الموافقة وحدَها.** والسحبُ لم يعد
// من هنا (بابُه صفحةُ السحب).

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type adminWalletFixture struct {
	pool     *pgxpool.Pool
	srv      *Server
	admin    string
	approver string
}

func newAdminWalletFixture(t *testing.T) *adminWalletFixture {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := realtime.NewHub(quiet)
	walletSvc := wallet.NewService(pool)
	return &adminWalletFixture{
		pool: pool,
		srv: &Server{
			pg:       pool,
			logger:   quiet,
			hub:      hub,
			wallet:   walletSvc,
			orders:   orders.NewService(pool, nil, walletSvc, nil, nil, quiet),
			settings: settings.NewStore(pool),
			notify:   notifications.New(pool, hub, quiet),
		},
		admin:    testdb.NewUser(t, pool, "admin"),
		approver: testdb.NewUser(t, pool, "admin"),
	}
}

func (f *adminWalletFixture) call(actor, method, path string, params map[string]string, body string,
	h func(http.ResponseWriter, *http.Request), headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rc := chi.NewRouteContext()
	for k, v := range params {
		rc.URLParams.Add(k, v)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, actor)
	ctx = context.WithValue(ctx, ctxRoles, []string{"admin", "finance"})
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

// apply **يقترح ثمّ يوافق غيرُ المقترِح** — ويُرجع رمزَ الخطوة التي حسمت.
func (f *adminWalletFixture) apply(target, kind string, amount int64, debit bool) int {
	body := `{"amount":` + itoa64(amount) + `,"kind":"` + kind + `","debit":` +
		map[bool]string{true: "true", false: "false"}[debit] + `,"note":"تسويةُ اختبار"}`
	w := f.call(f.admin, http.MethodPost, "/admin/users/"+target+"/wallet",
		map[string]string{"id": target}, body, f.srv.handleAdminWalletApply,
		map[string]string{"Idempotency-Key": "k-" + target + "-" + kind + itoa64(amount)})
	if w.Code >= 400 {
		return w.Code
	}
	var res struct {
		Data struct {
			RequestID string `json:"request_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	a := f.call(f.approver, http.MethodPost, "/admin/wallet-requests/"+res.Data.RequestID+"/approve",
		map[string]string{"id": res.Data.RequestID}, `{}`, f.srv.handleDecideWalletRequest(true), nil)
	if a.Code == http.StatusOK {
		return http.StatusOK
	}
	return a.Code
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

	t.Run("النوعُ من قائمةٍ مغلقة — والسحبُ ليس منها", func(t *testing.T) {
		uid := testdb.NewUser(t, f.pool, "driver")
		for _, kind := range []string{"order_payment", "commission", "refund", "", "TOPUP", "drop", "payout"} {
			if code := f.apply(uid, kind, 1000, false); code < 400 {
				t.Fatalf("قُبل النوعُ %q برمز %d — **والقائمةُ ثلاثةٌ لا غير.**", kind, code)
			}
		}
		if b := f.balance(t, uid); b != 0 {
			t.Fatalf("تحرّك الرصيدُ %d بأنواعٍ مرفوضة", b)
		}
	})

	// **ورصيدٌ قائمٌ قبل التجربة عمداً** — بلا الفحص يصير الشحنُ السالبُ خصماً صامتاً.
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
		if code := f.apply(uid, "adjustment", 9000, true); code < 400 {
			t.Fatalf("قُبل خصمٌ يتجاوز الرصيدَ برمز %d — **وهو مالٌ لا وجودَ له.**", code)
		}
		if b := f.balance(t, uid); b != 5000 {
			t.Fatalf("تبدّل الرصيدُ إلى %d بعد خصمٍ مرفوض", b)
		}
	})
}
