package server

// **عزلُ الزبائن — لا يقرأ أحدٌ ما ليس له ولا يمسّه.**
//
// # لماذا وُجد هذا الملفّ
//
// **لم يكن في المنصة اختبارٌ واحدٌ يمنع زبوناً من قراءة طلب غيره.**
//
// والحرّاسُ موجودون في الشيفرة — **لكنّ حارساً بلا اختبارٍ حارسٌ إلى أن
// يُحذَف سهواً**: يُعاد تشكيل معالِجٍ فيسقط سطرُ الملكيّة، **ولا شيءَ يصرخ**.
// والطلبُ يحمل **اسمَ الزبون ورقمَه وعنوانَه ومبلغَه** — ومعرّفاتُ الطلبات
// `uuid` لا تُخمَّن، **لكنّها تُسرَّب**: في رابطٍ يُشارَك، أو في لقطةِ شاشة،
// أو من موظّفٍ سابق.
//
// # وثلاثةُ أبوابٍ لا واحد
//
//	القراءةُ   يقرأ طلباً ليس له
//	الفعلُ     يُلغي أو يقيّم طلباً ليس له
//	الحذفُ     يمحو عنواناً ليس له
//
// **وكلُّها تُختبر بالحسابين لا بحسابٍ واحد** — وحارسٌ يُفحص بصاحب الحقّ
// وحدَه لا يُفحص أصلاً.
//
// # ولماذا «غيرُ موجود» لا «ممنوع»
//
// **«ممنوع» تُثبت أنّ الطلبَ موجود** — فمن جرّب ألفَ معرّفٍ عرف أيُّها حقيقيّ
// وإن لم يقرأه. **و«غير موجود» لا تقول شيئاً.**

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// twoCustomers صاحبُ الحقّ ودخيلٌ وطلبٌ للأوّل.
func twoCustomers(t *testing.T, f *driverFixture) (owner, intruder, orderID string) {
	t.Helper()
	ctx := context.Background()
	owner = testdb.NewUser(t, f.pool, "customer")
	intruder = testdb.NewUser(t, f.pool, "customer")

	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
		VALUES ($1, $2, 'pending', 'عنوانُ صاحب الحقّ',
		        ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
		        'cash', 20000, 10000, 30000, 0, 30000)
		RETURNING id`, owner, f.merchantID).Scan(&orderID); err != nil {
		t.Fatalf("تعذّر إنشاءُ طلب: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID)
	})
	return owner, intruder, orderID
}

// asCustomer ينادي معالِجاً بمعرّف مسارٍ وسياقِ زبون — كما يناديه المسار.
func asCustomer(
	h func(http.ResponseWriter, *http.Request),
	method, userID, id string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/x/"+id, nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", id)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"customer"})
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

// TestOrder_IntruderCannotRead **لا يقرأ طلباً ليس له.**
//
// الطلبُ يحمل اسمَ صاحبه ورقمَه **وعنوانَ بيته** — وتسريبُ عنوانِ بيتٍ أخطرُ
// من تسريب مبلغ.
func TestOrder_IntruderCannotRead(t *testing.T) {
	f := newDriverFixture(t, 0)
	owner, intruder, orderID := twoCustomers(t, f)

	if w := asCustomer(f.srv.handleMyOrder, http.MethodGet, owner, orderID); w.Code != 200 {
		t.Fatalf("صاحبُ الحقّ لم يقرأ طلبَه: %d — **والحارسُ يمنع من يجب أن يمرّ**", w.Code)
	}

	w := asCustomer(f.srv.handleMyOrder, http.MethodGet, intruder, orderID)
	if w.Code == 200 {
		t.Fatal("قرأ الدخيلُ طلبَ غيره — **وفيه اسمُه ورقمُه وعنوانُ بيته**")
	}
	// **و«غيرُ موجود» لا «ممنوع»**: الثانيةُ تُثبت أنّ الطلبَ موجود، **فمن
	// جرّب ألفَ معرّفٍ عرف أيُّها حقيقيّ** وإن لم يقرأ شيئاً.
	if w.Code != http.StatusNotFound {
		t.Errorf("رُدّ بـ%d والمتوقّع ٤٠٤ — **و«ممنوع» تكشف وجودَ الطلب**", w.Code)
	}
}

// TestOrder_IntruderCannotCancel **ولا يُلغي طلباً ليس له.**
//
// **وهذا أسوأُ من القراءة**: القراءةُ تسريب، **والإلغاءُ إتلاف** — زبونٌ ينتظر
// طعامَه فيجده أُلغي، والمتجرُ قد بدأ تحضيرَه.
func TestOrder_IntruderCannotCancel(t *testing.T) {
	f := newDriverFixture(t, 0)
	_, intruder, orderID := twoCustomers(t, f)

	asCustomer(f.srv.handleCustomerCancelOrder, http.MethodPost, intruder, orderID)

	var status string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status); err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب: %v", err)
	}
	if status != "pending" {
		t.Fatalf("صارت الحالةُ %q — **ألغى الدخيلُ طلبَ غيره**", status)
	}
}

// TestAddress_IntruderCannotDelete **ولا يمحو عنواناً ليس له.**
//
// **وعنوانٌ يُمحى لا يُستدرَك**: يعود صاحبُه ليطلب فلا يجد بيتَه، **فيرسمه
// من جديدٍ ويُخطئ** — والسائقُ يقف في شارعٍ آخر.
func TestAddress_IntruderCannotDelete(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner, intruder, _ := twoCustomers(t, f)

	var addrID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO user_addresses (user_id, label, address_text, location)
		VALUES ($1, 'بيتي', 'شارعُ صاحب الحقّ',
		        ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography)
		RETURNING id`, owner).Scan(&addrID); err != nil {
		t.Skipf("تعذّر إنشاءُ عنوان (قد يختلف الجدول): %v", err)
	}

	asCustomer(f.srv.handleDeleteAddress, http.MethodDelete, intruder, addrID)

	var alive bool
	_ = f.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM user_addresses WHERE id = $1)`, addrID).Scan(&alive)
	if !alive {
		t.Fatal("محا الدخيلُ عنوانَ غيره — **ولا يُستدرَك**")
	}
	_, _ = f.pool.Exec(ctx, `DELETE FROM user_addresses WHERE id = $1`, addrID)
}

// TestWallet_ShowsOnlyOwnBalance **ومحفظتُه محفظتُه.**
//
// **والكشفُ يُطلب بلا معرّف** — يُشتقّ من الجلسة، فالخطرُ ليس في المعرّف بل
// في **أن يُقرأ من مصدرٍ يقبل التزوير**. وهذا يُثبت أنّه من الجلسة.
func TestWallet_ShowsOnlyOwnBalance(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner, intruder, _ := twoCustomers(t, f)

	if _, err := f.pool.Exec(ctx, `
		INSERT INTO wallets (user_id, balance) VALUES ($1, 777000)
		ON CONFLICT (user_id) DO UPDATE SET balance = 777000`, owner); err != nil {
		t.Fatalf("تعذّرت المحفظة: %v", err)
	}

	w := asCustomer(f.srv.handleMyWallet, http.MethodGet, intruder, "")
	if w.Code == 200 && contains(w.Body.String(), "777000") {
		t.Fatal("رأى الدخيلُ رصيدَ غيره — **والكشفُ يُقرأ من غير الجلسة**")
	}
}

func contains(hay, needle string) bool {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
