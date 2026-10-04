package server

// **قراراتُ المالك ٢٠٢٦-١٠-٠٤ — قسمُ «الطلبات»، في أبواب الخادم**
//
//	«لدي توصيلة» لا تُحوَّل لمتجرٍ آخر               البند ١١ (المشكلة ١١)
//	العمليّاتُ تحسم الوجهةَ ولا تكتب التعويض           البند ١٢ (المشكلة ٥)
//	الطارئُ بعد الاستلام يبقى لقرار المكتب            المشكلة ٢٦
//	العمليّاتُ تبلغ الطوارئ · وإعادةُ الحساب للمالك    البندان ٧ و١٥

import (
	"context"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestTransfer_MerchantDeliveryRefused **«لدي توصيلة» لا تُحوَّل** — ومتجرُها مُنشئُها.
//
// **وكانت البطاقةُ تستثني الطلبَ الخاصَّ وحدَه والمحرّكُ لا يفحص النوع** —
// وتوصيلةٌ بلا أصنافٍ يتبدّل متجرُها بلا مطابقة.
func TestTransfer_MerchantDeliveryRefused(t *testing.T) {
	f := newDriverFixture(t, 1)
	ops := testdb.NewUser(t, f.pool, "operations")
	id := f.deliveryOrderAt(t, "at_pickup", f.drivers[0], true, "merchant")

	var other string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO merchants (name, category_id, commission_percent)
		SELECT 'متجرٌ آخر لاختبار التوصيلة', category_id, 10 FROM merchants WHERE id = $1
		RETURNING id`, f.merchantID).Scan(&other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, other) })

	w := f.call(f.srv.handleTransferOrder, http.MethodPost, "/admin/orders/"+id+"/transfer", id,
		ops, []string{"operations", "ops"}, `{"merchant_id":"`+other+`","note":"المتجر مغلق"}`)
	if w.Code != http.StatusConflict || errCode(t, w) != "transfer_delivery_kind" {
		t.Fatalf("**تحويلُ «لدي توصيلة» مرّ أو رُدّ بغير سببه** — %d: %s", w.Code, w.Body.String())
	}
	var merchant string
	_ = f.pool.QueryRow(context.Background(), `SELECT merchant_id::text FROM orders WHERE id = $1`, id).Scan(&merchant)
	if merchant != f.merchantID {
		t.Fatalf("تبدّل متجرُ التوصيلة: %s", merchant)
	}
	w = f.call(f.srv.handleTransferCandidates, http.MethodGet, "/admin/orders/"+id+"/transfer-candidates", id,
		ops, []string{"operations"}, "")
	if w.Code != http.StatusConflict {
		t.Fatalf("مرشّحو تحويل التوصيلة عُرضوا — %d", w.Code)
	}
}

// TestGoods_OperationsCannotWriteCompensation **العمليّاتُ تقول أين البضاعة ولا تدفع.**
//
// **كان البابُ يقبل المبلغَ ممّن يملك `orders.intervene`** — فيدفع موظّفُ
// العمليّات من الخزينة. **وصار التعويضُ بابَ الماليّة** (`/goods/compensation`).
func TestGoods_OperationsCannotWriteCompensation(t *testing.T) {
	f := newDriverFixture(t, 1)
	ops := testdb.NewUser(t, f.pool, "operations")
	id := f.problemOrderAt(t, "at_dropoff", f.drivers[0])
	if _, err := f.pool.Exec(context.Background(), `UPDATE orders SET status = 'failed',
		closed_at = now(), return_to = 'store', goods_handed_at = now() WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	w := f.call(f.srv.handleGoods, http.MethodPost, "/admin/orders/"+id+"/goods", id,
		ops, []string{"operations"}, `{"to":"merchant","compensation":500000}`)
	if w.Code != http.StatusForbidden || errCode(t, w) != "goods_compensation_finance" {
		t.Fatalf("**العمليّاتُ كتبت تعويضَ المتجر** — %d: %s", w.Code, w.Body.String())
	}
	var settled *string
	_ = f.pool.QueryRow(context.Background(), `SELECT goods_settled_to FROM orders WHERE id = $1`, id).Scan(&settled)
	if settled != nil {
		t.Fatalf("حُسمت البضاعةُ مع الرفض: %s", *settled)
	}
}

// TestEmergencyAfterPickup_OfficeDecides **البضاعةُ مع المصاب — والمكتبُ يقرّر.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤، غرفةُ الطوارئ: «لا تُرسَل آليّاً إلى سائقٍ جديدٍ
// يأخذها من موضع الأوّل».) **وقبل الاستلام يُحرَّر كما كان.**
func TestEmergencyAfterPickup_OfficeDecides(t *testing.T) {
	f := newDriverFixture(t, 2)
	after := f.orderAtPickedUp(t, f.drivers[0])
	if code := f.emergency(t, f.drivers[0], after, `{"note":"حادث"}`); code >= 400 {
		t.Fatalf("رُدّ الطارئ %d", code)
	}
	var status string
	_ = f.pool.QueryRow(context.Background(), `SELECT status FROM orders WHERE id = $1`, after).Scan(&status)
	if status != "picked_up" {
		t.Fatalf("**أُرسلت البضاعةُ آليّاً إلى الطابور** — الحالُ %s والمكتبُ لم يقرّر", status)
	}

	before := f.problemOrderAt(t, "assigned", f.drivers[1])
	if code := f.emergency(t, f.drivers[1], before, `{"note":"عطل"}`); code >= 400 {
		t.Fatalf("رُدّ الطارئ %d", code)
	}
	_ = f.pool.QueryRow(context.Background(), `SELECT status FROM orders WHERE id = $1`, before).Scan(&status)
	if status != "dispatching" {
		t.Fatalf("طارئٌ قبل الاستلام لم يُحرّر الطلب — %s", status)
	}
}

// TestOrdersBoardGrants **العمليّاتُ والدعمُ يبلغان الطوارئ · وإعادةُ الحساب
// للمالك والأدمن وحدَهما** (البندان ٧ و١٥ — هجرة `0182`).
func TestOrdersBoardGrants(t *testing.T) {
	pool := testdb.Pool(t)
	has := func(role, capability string) bool {
		var ok bool
		if err := pool.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM role_capabilities
			WHERE role_code = $1 AND capability_code = $2)`, role, capability).Scan(&ok); err != nil {
			t.Fatal(err)
		}
		return ok
	}
	for _, c := range []struct {
		role, capability string
		want             bool
	}{
		{"operations", "emergencies.manage", true},
		{"customer_support", "emergencies.manage", true},
		{"owner_super_admin", "emergencies.manage", true},
		{"operations", "support.manage", false},
		{"operations", "finance.recompute", false},
		{"finance", "finance.recompute", false},
		{"owner_super_admin", "finance.recompute", true},
		{"admin", "finance.recompute", true},
	} {
		if got := has(c.role, c.capability); got != c.want {
			t.Errorf("`%s` يملك `%s` = %v والمتوقّع %v", c.role, c.capability, got, c.want)
		}
	}
}
