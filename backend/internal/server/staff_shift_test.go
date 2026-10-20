package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestNotStaff **أبوابُ الحساب الشخصيّة مغلقةٌ على موظّف المتجر** (٢٠٢٦-١٠-٢٠) —
// مفتوحةٌ لصاحب الحساب.
func TestNotStaff(t *testing.T) {
	s := &Server{}
	h := s.notStaff(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	call := func(staff string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/auth/password", nil)
		if staff != "" {
			req = req.WithContext(context.WithValue(req.Context(), ctxStaff, staff))
		}
		w := httptest.NewRecorder()
		h(w, req)
		return w
	}
	if w := call(""); w.Code != http.StatusOK {
		t.Fatalf("صاحبُ الحساب رُدّ: %d", w.Code)
	}
	w := call("staff-1")
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "staff_not_allowed") {
		t.Fatalf("الموظّفُ مرّ إلى بابٍ شخصيّ: %d %s", w.Code, w.Body.String())
	}
}

// TestOrderPhones_ShiftNumberOnWhatsApp **تحويلُ الطلب على واتساب يذهب لرقم الدوام**
// (٢٠٢٦-١٠-٢٠) — ومدخلٌ ناقصٌ يُردّ، و«دوام كامل» يعود إلى الرقم الأصليّ.
func TestOrderPhones_ShiftNumberOnWhatsApp(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	admin := testdb.NewUser(t, f.pool, "admin")
	put := func(body string) int {
		return f.call(f.srv.handleAdminPutOrderPhones, http.MethodPut, "/admin/merchants/x/order-phones",
			f.merchantID, admin, []string{"admin"}, body).Code
	}
	if code := put(`{"mode":"split","morning_phone":"0911111111","evening_phone":"bad","evening_from":"16:00"}`); code != http.StatusBadRequest {
		t.Fatalf("رقمٌ مسائيٌّ فاسدٌ قُبل: %d", code)
	}
	if code := put(`{"mode":"split","morning_phone":"0911111111","evening_phone":"0922222222","evening_from":"25:00"}`); code != http.StatusBadRequest {
		t.Fatalf("ساعةٌ فاسدةٌ قُبلت: %d", code)
	}
	// **والمسائيُّ من منتصف الليل** — فهو الدوامُ الآن أيّاً كانت الساعة.
	if code := put(`{"mode":"split","morning_phone":"0911111111","evening_phone":"0922222222","evening_from":"00:00"}`); code != http.StatusOK {
		t.Fatalf("الحفظُ ردّ %d", code)
	}
	order := f.dispatchingOrder(t, 20_000, 5_000)
	_, ph, err := f.srv.loadOrderMessage(ctx, order)
	if err != nil {
		t.Fatal(err)
	}
	if ph.Shift != "evening" || ph.WhatsApp != "+963922222222" {
		t.Fatalf("الطلبُ لم يذهب لرقم المسائيّ: %+v", ph)
	}
	if code := put(`{"mode":"full"}`); code != http.StatusOK {
		t.Fatalf("العودةُ إلى دوامٍ كامل ردّت %d", code)
	}
	if _, ph, _ := f.srv.loadOrderMessage(ctx, order); ph.Shift != "" || ph.WhatsApp == "+963922222222" {
		t.Fatalf("الدوامُ الكاملُ بقي على رقم المسائيّ: %+v", ph)
	}
}
