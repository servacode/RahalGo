package server

// **طارئٌ بعد الاستلام: البضاعةُ مع السائق، والثاني يجب ألّا يذهب إلى المتجر.**
//
// # المسألة
//
// قبل الاستلام الطعامُ في المتجر، **فمن يلتقط الطلبَ بعده يذهب إليه كما كان.**
// وبعد الاستلام الطعامُ مع المصاب حيث وقف — **ومن ذهب إلى المطعم استلم طلباً
// ثانياً من مطبخٍ حضّر واحداً**، فتُدفع البضاعةُ مرّتين، والمتجرُ قبض مرّةً
// واحدة. **ويبقى الطلبُ الأوّل في الشارع.**
//
// # والثغرةُ كانت في الشرط
//
//	if afterPickup && hasPoint { … }
//
// **فإن رُفض إذنُ الموقع** — وهو الغالبُ على متصفّح سطح المكتب، وواردٌ على
// الهاتف داخلَ بناء — **لم يُكتب شيءٌ إطلاقاً**، ويذهب الثاني إلى المتجر.
//
// **والطارئُ نفسُه لا يُردّ لغياب الموقع** وهو صواب: «طارئٌ يُردّ لأن الموقعَ
// لم يُقرأ طارئٌ ضاع». **لكنّ البضاعةَ حينها كانت بلا عنوانٍ ولا كلمة.**
//
// **وكلمةٌ بلا نقطةٍ خيرٌ من نقطةٍ لم تُلتقط.**

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// orderAtPickedUp طلبٌ بيد السائق **وبضاعتُه خرجت من المتجر** — وهي اللحظةُ
// التي يصير فيها الطارئُ مسألةَ بضاعةٍ لا مسألةَ إسناد.
func (f *driverFixture) orderAtPickedUp(t *testing.T, driverID string) string {
	t.Helper()
	id := f.dispatchingOrder(t, 50_000, 8_000)
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET driver_id = $2, status = 'picked_up', picked_up_at = now()
		 WHERE id = $1`, id, driverID); err != nil {
		t.Fatalf("تعذّر وضعُ الطلب في picked_up: %v", err)
	}
	return id
}

func (f *driverFixture) emergency(t *testing.T, driverID, orderID, body string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/driver/orders/"+orderID+"/emergency", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", orderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, driverID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})

	w := httptest.NewRecorder()
	f.srv.handleDriverEmergency(w, req.WithContext(ctx))
	return w.Code
}

// pickupNote يقرأ ما سيقرؤه السائقُ التالي.
func (f *driverFixture) pickupNote(t *testing.T, orderID string) (string, bool) {
	t.Helper()
	var note string
	var hasPoint bool
	if err := f.pool.QueryRow(context.Background(),
		`SELECT pickup_override_note, pickup_override IS NOT NULL
		 FROM orders WHERE id = $1`, orderID).Scan(&note, &hasPoint); err != nil {
		t.Fatalf("قراءة نقطة الاستلام البديلة: %v", err)
	}
	return note, hasPoint
}

// TestEmergencyAfterPickup_WithoutLocation_StillWarns **بلا موقعٍ تبقى كلمة.**
func TestEmergencyAfterPickup_WithoutLocation_StillWarns(t *testing.T) {
	f := newDriverFixture(t, 1)
	driverID := f.drivers[0]
	orderID := f.orderAtPickedUp(t, driverID)

	if code := f.emergency(t, driverID, orderID, `{"note":"انكسرت الدرّاجة"}`); code >= 400 {
		t.Fatalf("رُدّ الطارئ %d — **وطارئٌ يُردّ طارئٌ ضاع**", code)
	}

	note, hasPoint := f.pickupNote(t, orderID)
	if hasPoint {
		t.Error("كُتبت نقطةٌ ولم يُرسَل موقع")
	}
	if strings.TrimSpace(note) == "" {
		t.Fatal("لا كلمةَ ولا نقطة — **والثاني سيذهب إلى متجرٍ سلّم بضاعتَه وقبض ثمنَها**")
	}
	if !strings.Contains(note, "المتجر") {
		t.Errorf("الكلمةُ لا تنهاه عن المتجر: %q", note)
	}
}

// TestEmergencyAfterPickup_WithLocation_SetsPoint **وبموقعٍ تُكتب النقطة.**
func TestEmergencyAfterPickup_WithLocation_SetsPoint(t *testing.T) {
	f := newDriverFixture(t, 1)
	driverID := f.drivers[0]
	orderID := f.orderAtPickedUp(t, driverID)

	body := `{"lat":35.9531,"lng":39.0082,"note":"عطل"}`
	if code := f.emergency(t, driverID, orderID, body); code >= 400 {
		t.Fatalf("رُدّ الطارئ %d", code)
	}

	note, hasPoint := f.pickupNote(t, orderID)
	if !hasPoint {
		t.Error("أُرسل الموقعُ ولم تُكتب النقطة — **والثاني لا يعرف أين يذهب**")
	}
	if strings.TrimSpace(note) == "" {
		t.Error("نقطةٌ بلا كلمة — والسائقُ يرى دبّوساً ولا يعرف لماذا")
	}
}
