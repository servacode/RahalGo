package server

// **مهمّاتُ السائق ما لم يُغلَق — لا أكثر.**
//
// # قرارُ المالك (٢٠٢٦-٠٨-٠٤)
//
// «مهمّاتٌ تفشل التسليم لا يجب أن تبقى بالمهام لدى السائق — **خلص، تُعتبر
// مغلقةً منتهية**».
//
// # وما كان
//
// كان `failed` يبقى في القائمة حتى تُحسم بضاعتُه — `returned_at` أو
// `goods_settled_to`. **والمخرجان مسدودان**: زرُّ السائق لمتاجرِ الاسترداد
// وحدَها، وزرّا الإدارة يردّان `410`. **فمتجرٌ لا يستردّ يترك بطاقةً لا تُغلق
// أبداً** — تتراكم في شاشةٍ يقرؤها بيدٍ واحدةٍ على درّاجة.
//
// # ولماذا اختبارٌ
//
// الشرطُ سطرٌ في `WHERE`، **وعودتُه لا تكسر شيئاً ولا تظهر في أيّ خطأ** —
// تظهر بطاقةٌ زائدةٌ في شاشةٍ لا يراها من يكتب الاستعلام.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// driverOrders ينادي القائمةَ كما تناديها شاشةُ السائق.
func (f *driverFixture) driverOrders(t *testing.T, driverID string) []map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/driver/orders", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, driverID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})
	w := httptest.NewRecorder()
	f.srv.handleDriverOrders(w, req.WithContext(ctx))

	if w.Code != http.StatusOK {
		t.Fatalf("القائمةُ ردّت %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("ردٌّ لا يُقرأ: %v — %s", err, w.Body.String())
	}
	return body.Data
}

// TestDriverOrders_FailedLeavesTheList **ما فشل خرج** — ولو بقيت بضاعتُه.
func TestDriverOrders_FailedLeavesTheList(t *testing.T) {
	f := newDriverFixture(t, 1)
	driver := f.drivers[0]
	orderID := f.atDropoffOrder(t, driver)

	// **قبل**: مهمّةٌ قائمةٌ عند باب الزبون.
	before := f.driverOrders(t, driver)
	if len(before) != 1 {
		t.Fatalf("الطلبُ عند باب الزبون ولم يظهر في مهامّه: %d", len(before))
	}

	if w := f.fail(driver, orderID, "customer_absent", ""); w.Code != http.StatusOK {
		t.Fatalf("تعذّرَ الإفشال: %d — %s", w.Code, w.Body.String())
	}

	// **وبعد**: لا شيء. **ولم تُحسم البضاعةُ عمداً** — `returned_at`
	// و`goods_settled_to` فارغان، وهو الحالُ الذي كان يُبقيه ظاهراً.
	after := f.driverOrders(t, driver)
	if len(after) != 0 {
		t.Fatalf("بقيت %d مهمّةٍ بعد التعذّر — **والطلبُ مُغلَق** (`closed_at`)، "+
			"وبطاقةٌ لا فعلَ فيها تزاحم ما فيه فعل.\n"+
			"الشرطُ في `handleDriverOrders`: `o.closed_at IS NULL` وحدَه.", len(after))
	}

	// **والأثرُ باقٍ في القاعدة** — الخروجُ من الشاشة ليس محواً من الدفاتر.
	var status, failReason string
	var closed bool
	if err := f.pool.QueryRow(context.Background(), `
		SELECT status, COALESCE(fail_reason, ''), closed_at IS NOT NULL
		FROM orders WHERE id = $1`, orderID).Scan(&status, &failReason, &closed); err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب: %v", err)
	}
	if status != "failed" || failReason != "customer_absent" || !closed {
		t.Fatalf("الأثرُ ناقص: status=%q reason=%q closed=%v", status, failReason, closed)
	}
}

// TestDriverOrders_OpenWorkStays **وما لم يُغلق يبقى** — كي لا يُقرأ الحارسُ
// السابقُ «أخفِ كلَّ شيء».
func TestDriverOrders_OpenWorkStays(t *testing.T) {
	f := newDriverFixture(t, 1)
	driver := f.drivers[0]
	f.atDropoffOrder(t, driver)

	if got := f.driverOrders(t, driver); len(got) != 1 {
		t.Fatalf("مهمّةٌ مفتوحةٌ ولم تظهر: %d", len(got))
	}
}
