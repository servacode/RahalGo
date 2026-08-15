package server

// **مرجعُ الكشف رقمُ طلبٍ يفتحه البحث — لا معرّفٌ يردّ فراغاً.**
//
// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٥.)
//
// # العطبُ الذي كان
//
// **الكشفُ كان يرسل `o.id::text`** — والشاشةُ تبني منه
// `‎/dashboard/orders?q=<معرّف>`، **وبحثُ الطلبات يطابق `number::text` أو
// هاتفَ الزبون لا المعرّف.**
//
// **فكلُّ سطرٍ في كشف سائقٍ أو متجرٍ يُضغط فيردّ «لا نتائج»** — ولا خطأ
// ولا سطرٌ في سجلّ: **شاشةٌ تعمل وتُجيب بالفراغ**، وهي أسوأُ من شاشةٍ
// تسقط لأنّ من ينظر يراها سليمة.
//
// **والفحصُ لا يقارن نصّاً بنصّ**: يأخذ المرجعَ من الكشف **ويبحث به في
// شاشة الطلبات فعلاً** — فلو تبدّل أحدُهما يوماً سقط هنا.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// TestFinancialsRef_OpensInOrdersSearch **ما يُرسَل مرجعاً يجده البحث.**
func TestFinancialsRef_OpensInOrdersSearch(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	customer, _, _ := twoCustomers(t, f)
	driver := f.drivers[0]

	// **طلبٌ سلّمه السائقُ بأجرة** — وهو ما يملأ «مستحقٌّ له» في كشفه.
	var orderID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text,
			dropoff, payment_method, subtotal, delivery_fee, total, cash_due,
			delivered_at, closed_at)
		VALUES ($1, $2, $3, 'delivered', 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 10000, 3000, 13000, 13000, now(), now())
		RETURNING id::text`, customer, f.merchantID, driver).Scan(&orderID); err != nil {
		t.Fatalf("تعذّر إنشاءُ طلب: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID)
	})

	var fin struct {
		Data struct {
			OwedTo struct {
				Items []struct {
					Ref string `json:"ref"`
				} `json:"items"`
			} `json:"owed_to"`
		} `json:"data"`
	}
	w := asCustomer(f.srv.handleAdminUserFinancials, http.MethodGet, driver, driver)
	if w.Code != 200 {
		t.Fatalf("ردَّ %d", w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &fin); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v — %s", err, w.Body.String())
	}

	if len(fin.Data.OwedTo.Items) == 0 {
		t.Fatal("لا بنودَ في «مستحقٌّ له» وقد سلّم طلباً بأجرة")
	}
	ref := fin.Data.OwedTo.Items[0].Ref
	if ref == "" {
		t.Fatal("مرجعٌ فارغ — **والسطرُ يُرسم بلا رابطٍ فلا يُفتح**")
	}

	// ══════════════════════════════════════════════════════════════════
	// **والمرجعُ يُبحث به كما تبحث الشاشة**
	// ══════════════════════════════════════════════════════════════════
	//
	// **ولو قُورن بنصٍّ متوقَّعٍ لَنجح الفحصُ ونجا العطب**: العبرةُ أن
	// **يجد البحثُ الطلب**، لا أن يُشبه المرجعُ شكلاً نتوقّعه.
	page, err := f.srv.orders.List(ctx, orders.ListFilter{Query: ref, Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("تعذّر البحث: %v", err)
	}
	if len(page.Orders) == 0 {
		t.Fatalf("البحثُ عن %q لا يجد شيئاً — **فيُضغط السطرُ فيردّ «لا نتائج»**", ref)
	}
	if page.Orders[0].ID != orderID {
		t.Fatalf("وجد طلباً آخر: %s لا %s", page.Orders[0].ID, orderID)
	}
}
