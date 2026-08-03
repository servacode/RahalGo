package server

// **الرمزُ المصنَّفُ يُغني عن النصّ — واختبارٌ يُثبته.**
//
// # الحادثة
//
// شاشةُ السائق كانت تعرض **حقلَ نصٍّ حرّ** لسبب التعذّر، **والخادمُ يرفض كلَّ
// ما ليس رمزاً من `FailReasons`.** فكلُّ ضغطةٍ على «تعذّر التسليم» تعود بخطأ،
// **والطلبُ لا يتحرّك**: يبقى زرُّ «سلّمتُ الطلب» ظاهراً وطلبٌ فاشلٌ قائم.
//
// شهده المالكُ في شاشته (٢٠٢٦-٠٨-٠٣): «رغم أنني قلت تعذّر التسليم ما زال زرّ
// سلّمت الطلب يظهر والطلب قائم».
//
// # ثمّ حارسٌ ثانٍ تحته
//
// ولمّا صارت الشاشةُ تُرسل الرمز، **ظهر حارسٌ آخر يطلب النصَّ أيضاً**:
//
//	if requiresReason[req.To] && note == "" { … }
//
// **فيُطلَب شيئان عن واقعةٍ واحدة** — وهو خلافُ ما قرّرناه في `failreasons.go`
// بالحرف: «تفصيلُ ما وقع يبقى في نصٍّ **اختياريّ** بجانب السبب».
//
// **ومن أُلزم بالكتابة كتب حرفاً ليمرّ** — وقد وقع فعلاً: `#1004` سببُه
// المحفوظ «1». **وإلزامٌ يُنتج ضجيجاً أسوأُ من لا إلزام**: الأوّل يُقرأ فيُصدَّق.
//
// # ولذلك يُنادى المسارُ هنا
//
// اختباران توأمان: **رمزٌ بلا نصٍّ يمرّ**، **ونصٌّ بلا رمزٍ يُردّ.** ولو عاد
// الحارسُ يوماً إلى ما كان، سقط أوّلُهما.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// atDropoffOrder طلبٌ نقديٌّ بلغ باب الزبون بيد هذا السائق — الحالُ الذي
// يُضغط فيه «تعذّر التسليم».
func (f *driverFixture) atDropoffOrder(t *testing.T, driverID string) string {
	t.Helper()
	id := f.dispatchingOrder(t, 20000, 5000)
	if _, err := f.pool.Exec(context.Background(), `
		UPDATE orders SET status = 'at_dropoff', driver_id = $2, accepted_at = now()
		WHERE id = $1`, id, driverID); err != nil {
		t.Fatalf("تعذّر وضعُ الطلب عند باب الزبون: %v", err)
	}
	return id
}

// fail ينادي المعالِج الحقيقي كما يناديه المسار.
func (f *driverFixture) fail(driverID, orderID, reason, note string) *httptest.ResponseRecorder {
	body := `{"to":"failed","reason":"` + reason + `","note":"` + note + `"}`
	req := httptest.NewRequest(http.MethodPost,
		"/driver/orders/"+orderID+"/transition", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", orderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, driverID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})

	w := httptest.NewRecorder()
	f.srv.handleDriverTransition(w, req.WithContext(ctx))
	return w
}

// TestDriverFail_CodedReasonNeedsNoNote **سببٌ مختارٌ يكفي — ولا تُطلب كتابة.**
func TestDriverFail_CodedReasonNeedsNoNote(t *testing.T) {
	f := newDriverFixture(t, 1)
	driver := f.drivers[0]
	orderID := f.atDropoffOrder(t, driver)

	w := f.fail(driver, orderID, "customer_absent", "")

	if w.Code != http.StatusOK {
		t.Fatalf("اختار السائقُ «الزبون غير موجود» ولم يكتب شيئاً — فردّ الخادمُ %d (%s)\n"+
			"والتفصيلُ اختياريٌّ بنصّ `failreasons.go`: «القائمةُ تُصنّف والنصُّ يشرح»\n"+
			"الردّ: %s", w.Code, errCode(t, w), w.Body.String())
	}

	// **والذنبُ يُشتقّ من القائمة لا من تقدير أحد** — وهو ما يقرّر التعويض.
	var status, fault, failReason string
	if err := f.pool.QueryRow(context.Background(), `
		SELECT status, COALESCE(fault, ''), COALESCE(fail_reason, '')
		FROM orders WHERE id = $1`, orderID).Scan(&status, &fault, &failReason); err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب: %v", err)
	}
	if status != "failed" {
		t.Fatalf("الطلبُ بقي %q بعد التعذّر — والسائقُ يرى زرَّ «سلّمتُ الطلب» قائماً", status)
	}
	if failReason != "customer_absent" {
		t.Fatalf("الرمزُ المحفوظ %q لا «customer_absent» — ولا يُقاس ما لا يُصنَّف", failReason)
	}
	if fault != "customer" {
		t.Fatalf("الذنبُ %q لا «customer» — والذنبُ هو من يقرّر التعويض", fault)
	}
}

// TestDriverFail_FreeTextAloneRejected **ونصٌّ بلا رمزٍ يُردّ** — كي لا يعود
// الحقلُ الحرُّ من حيث خرج.
func TestDriverFail_FreeTextAloneRejected(t *testing.T) {
	f := newDriverFixture(t, 1)
	driver := f.drivers[0]
	orderID := f.atDropoffOrder(t, driver)

	w := f.fail(driver, orderID, "", "الزبون لم يفتح الباب")

	if w.Code == http.StatusOK {
		t.Fatal("مرّ نصٌّ حرٌّ بلا رمزٍ مصنَّف — **ونصٌّ لا يُعدّ ولا يُقاس**، " +
			"ولا يُشتقّ منه ذنبٌ فلا يُعرف من يتحمّل")
	}
	if code := errCode(t, w); code != "bad_fail_reason" {
		t.Fatalf("رُدّ بـ%q لا «bad_fail_reason» — والرسالةُ يقرؤها السائق", code)
	}
}
