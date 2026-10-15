package server

// **طلباتُ السحب — قراراتُ المالك ٢٠٢٦-١٠-١٥.**
//
//	طلبٌ مفتوحٌ واحد      ← معلَّقٌ أو قيدُ الصرف، لا المعلَّقُ وحدَه
//	كلمةُ السرّ للمال     ← «تم الصرف» و«ارتدّ» وحدَهما؛ «بدأ الصرف» و«ارفض» و«فشل» بلا كلمة
//	دورُ الطالب           ← يُكتب على الطلب من التطبيق الذي طلب منه
//	المتاحُ للسحب         ← كشفُ المحفظة يقول المحجوزَ والمتاح
//	«مرتجع أو فاشل»       ← فلترٌ واحدٌ تفتحه البطاقةُ الرابعة

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestPAYOUTS1015_SecondWhileProcessingRefused **طلبٌ ثانٍ والأوّلُ قيدُ الصرف يُردّ.**
func TestPAYOUTS1015_SecondWhileProcessingRefused(t *testing.T) {
	f := newPayoutFixture(t)
	fin := f.financeUser(t)
	uid, id := f.newPayout(t, "driver", f.min)
	if w := f.decide(fin, financeRoles, id, map[string]any{"status": "processing"}); w.Code != 200 {
		t.Fatalf("بدءُ الصرف ردّ %d — %s", w.Code, w.Body.String())
	}
	w := f.request(uid, "driver", f.min)
	if w.Code != http.StatusConflict || errCode(t, w) != "payout_pending" {
		t.Fatalf("**طلبٌ ثانٍ مرّ والأوّلُ قيدُ الصرف**: %d — %s", w.Code, w.Body.String())
	}
	var open int
	_ = f.pool.QueryRow(context.Background(), `SELECT count(*) FROM payout_requests
		WHERE user_id = $1 AND status IN ('pending', 'processing')`, uid).Scan(&open)
	if open != 1 {
		t.Fatalf("للحساب %d طلباتٍ مفتوحة — يُنتظر واحد", open)
	}
}

// TestPAYOUTS1015_StepUpOnlyWhenMoneyMoves **«بدأ الصرف» و«ارفض» بلا كلمة سرّ، والصرفُ بها.**
//
// **عبر الوسيط الحقيقيّ** (`requireStepUp`) — لا المصنِّفِ وحدَه.
func TestPAYOUTS1015_StepUpOnlyWhenMoneyMoves(t *testing.T) {
	s := &Server{}
	passed := false
	h := s.requireStepUp(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		passed = true
		w.WriteHeader(http.StatusOK)
	}))
	const path = "/api/v1/admin/payouts/0b6f4f3e-8f1a-4c55-9b7e-2d0d6f1a9c11/decide"
	for status, wantStepUp := range map[string]bool{
		"processing": false, "rejected": false, "failed": false,
		"paid": true, "reversed": true, "": true, "bogus": true,
	} {
		passed = false
		body := `{"status":"` + status + `","amount":1000}`
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		gotStepUp := w.Code == http.StatusForbidden && strings.Contains(w.Body.String(), "step_up_required")
		if gotStepUp != wantStepUp || passed == wantStepUp {
			t.Errorf("الحال %q ⇒ طُلبت الكلمة=%v (يُنتظر %v) — %d %s",
				status, gotStepUp, wantStepUp, w.Code, w.Body.String())
		}
	}
}

// TestPAYOUTS1015_RoleRecordedFromApp **حسابٌ بدورين يطلب من تطبيق المتجر ⇒ الطلبُ «متجر».**
//
// **والاشتقاقُ القديمُ يقول «سائق»** (أوّلُ الأدوار أبجديّاً) — وذاك ما كان يُعرض.
func TestPAYOUTS1015_RoleRecordedFromApp(t *testing.T) {
	f := newPayoutFixture(t)
	uid := f.user(t, "driver", 4*f.min)
	if _, err := f.pool.Exec(context.Background(),
		`INSERT INTO user_roles (user_id, role_code) VALUES ($1, 'merchant')`, uid); err != nil {
		t.Fatalf("دورٌ ثانٍ: %v", err)
	}
	body := `{"amount":` + itoa64(f.min) + `,"note":"من تطبيق المتجر"}`
	req := httptest.NewRequest(http.MethodPost, "/me/payouts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), ctxUserID, uid)
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver", "merchant"})
	ctx = context.WithValue(ctx, chi.RouteCtxKey, chi.NewRouteContext())
	ctx = identity.WithClient(ctx, "android-merchant")
	w := httptest.NewRecorder()
	f.srv.handleCreatePayout(w, req.WithContext(ctx))
	if w.Code != http.StatusCreated {
		t.Fatalf("الطلب ردّ %d — %s", w.Code, w.Body.String())
	}
	var role *string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT role FROM payout_requests WHERE user_id = $1`, uid).Scan(&role); err != nil {
		t.Fatalf("قراءةُ الدور: %v", err)
	}
	if role == nil || *role != "merchant" {
		t.Fatalf("**الدورُ المكتوبُ على الطلب %v — يُنتظر merchant**", role)
	}
	// **والقائمةُ وفلترُها يقرآنه** — لا الاشتقاقَ القديم.
	list := f.adminList(t, "role=merchant&status=pending&per_page=100")
	found := false
	for _, p := range list {
		if p.UserID == uid {
			found = p.UserRole == "merchant"
		}
	}
	if !found {
		t.Fatalf("**الطلبُ غائبٌ عن فلتر «متجر» أو بدورٍ آخر**: %+v", list)
	}
}

// TestPAYOUTS1015_FailedOrReversedFilter **البطاقةُ الرابعة تفتح المرتجعَ والفاشلَ معاً.**
func TestPAYOUTS1015_FailedOrReversedFilter(t *testing.T) {
	f := newPayoutFixture(t)
	fin := f.financeUser(t)
	_, failed := f.newPayout(t, "driver", f.min)
	if w := f.decide(fin, financeRoles, failed, map[string]any{"status": "failed", "decision": "رقم خطأ"}); w.Code != 200 {
		t.Fatalf("الفشل ردّ %d — %s", w.Code, w.Body.String())
	}
	_, pending := f.newPayout(t, "driver", f.min)
	got := map[string]string{}
	for _, p := range f.adminList(t, "status=failed_or_reversed&per_page=100") {
		got[p.ID] = p.Status
		if p.Status != "failed" && p.Status != "reversed" {
			t.Errorf("حالُ %q في فلتر «مرتجع أو فاشل»", p.Status)
		}
	}
	if got[failed] != "failed" {
		t.Errorf("**الفاشلُ غائبٌ عن فلتر «مرتجع أو فاشل»**")
	}
	if _, ok := got[pending]; ok {
		t.Errorf("المعلَّقُ ظهر في فلتر «مرتجع أو فاشل»")
	}
}

// TestPAYOUTS1015_WalletStatementHasAvailable **كشفُ المحفظة يقول المحجوزَ والمتاح.**
func TestPAYOUTS1015_WalletStatementHasAvailable(t *testing.T) {
	f := newPayoutFixture(t)
	uid, _ := f.newPayout(t, "driver", f.min) // الرصيدُ ٤×الحدّ، والمحجوزُ الحدّ
	req := httptest.NewRequest(http.MethodGet, "/my/wallet", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, uid)
	w := httptest.NewRecorder()
	f.srv.handleMyWallet(w, req.WithContext(ctx))
	if w.Code != 200 {
		t.Fatalf("الكشف ردّ %d — %s", w.Code, w.Body.String())
	}
	var out struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	res, okR := out.Data["reserved"].(float64)
	avail, okA := out.Data["available"].(float64)
	if !okR || !okA {
		t.Fatalf("**الكشفُ بلا `reserved`/`available`**: %s", w.Body.String())
	}
	if int64(res) != f.min || int64(avail) != 3*f.min {
		t.Fatalf("المحجوز %v والمتاح %v — يُنتظر %d و%d", res, avail, f.min, 3*f.min)
	}
}

// adminList قائمةُ طلبات السحب كما تقرؤها اللوحة.
func (f *payoutFixture) adminList(t *testing.T, query string) []payout {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/admin/payouts?"+query, nil)
	ctx := context.WithValue(req.Context(), ctxUserID, testdb.NewUser(t, f.pool, "finance"))
	w := httptest.NewRecorder()
	f.srv.handleAdminPayouts(w, req.WithContext(ctx))
	if w.Code != 200 {
		t.Fatalf("القائمة ردّت %d — %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			Payouts []payout `json:"payouts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("ردٌّ غير صالح: %v", err)
	}
	return out.Data.Payouts
}
