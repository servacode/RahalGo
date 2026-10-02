package server

// ══════════════════════════════════════════════════════════════════════
// **زرُّ «لدي مشكلة» من باب السائق والعمليات** (قرارُ المالك ٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// المحرّكُ يُختبر في `orders/problem_button_test.go`. **وهنا ما بين المحرّك
// والشاشة**: الحقولُ التي يقرؤها التطبيق، ورموزُ الردّ وأجسامُها، **وبابُ
// الموافقة على التعويض**، والطارئُ وطلباتُ السائق الأخرى.

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// call ينادي معالِجاً كما يناديه المسار — بمعرّفٍ ومستخدمٍ وأدوار.
func (f *driverFixture) call(h http.HandlerFunc, method, path, orderID, userID string,
	roles []string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	if orderID != "" {
		rc.URLParams.Add("id", orderID)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRoles, roles)
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

// problemOrderAt طلبٌ في حالٍ بعينها بيد هذا السائق — **بلا حدث وصول** (فلا انتظار).
func (f *driverFixture) problemOrderAt(t *testing.T, status, driverID string) string {
	t.Helper()
	id := f.dispatchingOrder(t, 20_000, 5_000)
	if _, err := f.pool.Exec(context.Background(), `
		UPDATE orders SET status = $2, driver_id = $3, accepted_at = now(),
		       driver_fee = delivery_fee,
		       picked_up_at = CASE WHEN $2 IN ('picked_up','on_the_way','at_dropoff') THEN now() END
		WHERE id = $1`, id, status, driverID); err != nil {
		t.Fatalf("تعذّر وضعُ الطلب في %s: %v", status, err)
	}
	return id
}

// armOps يركّب الإشعارات ومستخدمَ عملياتٍ يُعدّ ما يصله.
func (f *driverFixture) armOps(t *testing.T) string {
	t.Helper()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	f.srv.notify = notifications.New(f.pool, f.srv.hub, quiet)
	return testdb.NewUser(t, f.pool, "ops")
}

func (f *driverFixture) opsAlerts(t *testing.T, opsID, orderID string) []string {
	t.Helper()
	rows, err := f.pool.Query(context.Background(), `
		SELECT title || ' | ' || body FROM notifications
		WHERE user_id = $1 AND entity_id = $2 ORDER BY created_at`, opsID, orderID)
	if err != nil {
		t.Fatalf("قراءةُ التنبيهات: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

func (f *driverFixture) orderRow(t *testing.T, orderID string) (status string, driver *string) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(),
		`SELECT status, driver_id::text FROM orders WHERE id = $1`, orderID).
		Scan(&status, &driver); err != nil {
		t.Fatalf("قراءةُ الطلب: %v", err)
	}
	return status, driver
}

type reasonItem struct {
	Code           string `json:"code"`
	Fault          string `json:"fault"`
	Kind           string `json:"kind"`
	Closes         bool   `json:"closes"`
	AvailableInSec int64  `json:"available_in_sec"`
}

func (f *driverFixture) reasons(t *testing.T, driverID, query string) map[string]reasonItem {
	t.Helper()
	w := f.call(f.srv.handleFailReasons, http.MethodGet, "/driver/fail-reasons?"+query, "",
		driverID, []string{"driver"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("الأسبابُ ردّت %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data struct {
			Reasons []reasonItem `json:"reasons"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	out := map[string]reasonItem{}
	for _, r := range body.Data.Reasons {
		out[r.Code] = r
	}
	return out
}

// TestFailReasons_KindsClosesAndReports **كلُّ سببٍ يقول ما يفعله بالطلب.**
func TestFailReasons_KindsClosesAndReports(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]

	pickup := f.reasons(t, d, "at=at_pickup")
	if r := pickup["merchant_closed"]; r.Kind != "fail" || r.Closes || r.Fault != "merchant" {
		t.Errorf("merchant_closed = %+v — **عند المتجر لا يُغلق الطلب**", r)
	}
	if r, ok := pickup["merchant_not_ready"]; !ok || r.Kind != "report" || r.Closes {
		t.Errorf("merchant_not_ready = %+v — **بلاغٌ لا فشل** (قرارُ المالك)", r)
	}
	if _, ok := pickup["customer_absent"]; ok {
		t.Error("«الزبونُ غير موجود» يُعرض عند المتجر")
	}

	door := f.reasons(t, d, "at=at_dropoff")
	for _, code := range []string{"customer_absent", "customer_refused", "address_wrong", "driver_late"} {
		if r := door[code]; r.Kind != "fail" || !r.Closes {
			t.Errorf("%s = %+v — **عند الزبون يُغلق**", code, r)
		}
	}
	if r := door["customer_no_answer"]; r.Kind != "report" || r.Closes {
		t.Errorf("customer_no_answer = %+v — بلاغٌ للعمليات", r)
	}

	way := f.reasons(t, d, "at=on_the_way")
	for _, code := range []string{"customer_cancelled_by_phone", "customer_new_address"} {
		if r := way[code]; r.Kind != "report" || r.Closes {
			t.Errorf("%s = %+v في الطريق — بلاغٌ والطلبُ يبقى", code, r)
		}
	}
	for code, r := range way {
		if r.Kind != "report" {
			t.Errorf("في الطريق سببُ فشلٍ %q — **ولا يُغلق الطلبُ قبل الزبون**", code)
		}
	}
}

// TestFailReasons_DoorWaitCountdown **الشاشةُ تعرف كم بقي.**
func TestFailReasons_DoorWaitCountdown(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	orderID := f.problemOrderAt(t, "on_the_way", d)
	if _, err := f.srv.orders.Transition(context.Background(), d, []string{"driver"},
		orderID, orders.StAtDropoff, ""); err != nil {
		t.Fatalf("الوصول: %v", err)
	}
	door := f.reasons(t, d, "at=at_dropoff&order="+orderID)
	if r := door["customer_absent"]; r.AvailableInSec <= 0 || r.AvailableInSec > 300 {
		t.Errorf("customer_absent متاحٌ بعد %d ثانية — والمتوقّع بين ١ و٣٠٠", r.AvailableInSec)
	}
	if r := door["customer_refused"]; r.AvailableInSec != 0 {
		t.Errorf("customer_refused ينتظر %d — **والرفضُ لا ينتظر**", r.AvailableInSec)
	}

	// **والفشلُ قبل انقضائه ٤٠٩ بما بقي**.
	w := f.fail(d, orderID, "customer_absent", "")
	if w.Code != http.StatusConflict || errCode(t, w) != "door_wait" {
		t.Fatalf("الفشلُ بعد ثوانٍ ردّ %d %s — والمتوقّع 409 door_wait", w.Code, w.Body.String())
	}
	var body struct {
		Error struct {
			Details map[string]float64 `json:"details"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if left := body.Error.Details["remaining_sec"]; left <= 0 || left > 300 {
		t.Errorf("remaining_sec = %v في جسم الردّ: %s", left, w.Body.String())
	}
}

// TestDriverFail_WrongStageIs409 **سببٌ لا يخصّ المرحلة يُردّ من الباب.**
func TestDriverFail_WrongStageIs409(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	orderID := f.problemOrderAt(t, "at_pickup", d)
	w := f.fail(d, orderID, "customer_absent", "")
	if w.Code != http.StatusConflict || errCode(t, w) != "fail_reason_wrong_stage" {
		t.Fatalf("«الزبونُ غير موجود» عند المتجر ردّ %d %s", w.Code, w.Body.String())
	}
	if st, _ := f.orderRow(t, orderID); st != "at_pickup" {
		t.Errorf("تبدّلت الحالُ إلى %s", st)
	}
}

// TestStageReport_PerStatus **البلاغُ في مرحلته يُنبّه ولا يمسّ الطلب — وفي
// غيرها يُردّ.**
func TestStageReport_PerStatus(t *testing.T) {
	cases := []struct {
		status, code string
		ok           bool
	}{
		{"at_pickup", "merchant_not_ready", true},
		{"picked_up", "customer_cancelled_by_phone", true},
		{"on_the_way", "customer_new_address", true},
		{"at_dropoff", "customer_no_answer", true},
		{"at_pickup", "customer_no_answer", false},
		{"at_dropoff", "customer_cancelled_by_phone", false},
		{"assigned", "merchant_not_ready", false},
		{"at_dropoff", "customer_absent", false},
	}
	for _, c := range cases {
		t.Run(c.status+"/"+c.code, func(t *testing.T) {
			f := newDriverFixture(t, 1)
			ops := f.armOps(t)
			d := f.drivers[0]
			orderID := f.problemOrderAt(t, c.status, d)
			body := `{"code":"` + c.code + `","note":"ملاحظة"}`
			report := func() *httptest.ResponseRecorder {
				return f.call(f.srv.handleDriverReportOrStage, http.MethodPost,
					"/driver/orders/"+orderID+"/report", orderID, d, []string{"driver"}, body)
			}
			w := report()
			st, drv := f.orderRow(t, orderID)
			if st != c.status || drv == nil || *drv != d {
				t.Fatalf("تبدّل الطلبُ (%s · %v) — **والبلاغُ لا يمسّه**", st, drv)
			}
			alerts := f.opsAlerts(t, ops, orderID)
			if !c.ok {
				if w.Code != http.StatusConflict || errCode(t, w) != "report_wrong_stage" {
					t.Fatalf("بلاغٌ في غير مرحلته ردّ %d %s", w.Code, w.Body.String())
				}
				if len(alerts) != 0 {
					t.Errorf("نُبّهت العملياتُ ببلاغٍ مردود: %v", alerts)
				}
				return
			}
			if w.Code != http.StatusOK {
				t.Fatalf("البلاغُ ردّ %d: %s", w.Code, w.Body.String())
			}
			if len(alerts) != 1 {
				t.Fatalf("تنبيهاتُ العمليات %d والمتوقّع 1: %v", len(alerts), alerts)
			}
			// **والتكرارُ في دقيقةٍ ردٌّ ناجحٌ بلا تنبيهٍ ثانٍ.**
			if w2 := report(); w2.Code != http.StatusOK || !strings.Contains(w2.Body.String(), `"duplicate":true`) {
				t.Fatalf("التكرارُ ردّ %d: %s", w2.Code, w2.Body.String())
			}
			if n := len(f.opsAlerts(t, ops, orderID)); n != 1 {
				t.Errorf("التكرارُ نبّه ثانيةً: %d", n)
			}
		})
	}
}

// TestStageReport_OldAppSendsNotReadyAsFail **تطبيقٌ لم يُحدَّث يرسل «غيرُ جاهز»
// فشلاً — فيُسجَّل بلاغاً والسائقُ باقٍ على الطلب.**
func TestStageReport_OldAppSendsNotReadyAsFail(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	orderID := f.problemOrderAt(t, "at_pickup", d)
	if w := f.fail(d, orderID, "merchant_not_ready", ""); w.Code != http.StatusOK {
		t.Fatalf("ردّ %d: %s", w.Code, w.Body.String())
	}
	st, drv := f.orderRow(t, orderID)
	if st != "at_pickup" || drv == nil || *drv != d {
		t.Fatalf("(%s · %v) — **«غيرُ جاهز» كان يُحرّر السائقَ من الطلب**", st, drv)
	}
}

func (f *driverFixture) armCompensation(t *testing.T) string {
	t.Helper()
	f.srv.orders.SetSettings(settings.NewStore(f.pool))
	f.setSetting(t, "drivers.failed_compensation_percent", 50)
	return testdb.NewUser(t, f.pool, "admin")
}

func (f *driverFixture) compensate(orderID, adminID string, amount int64) *httptest.ResponseRecorder {
	return f.call(f.srv.handleCompensateDriver, http.MethodPost,
		"/admin/orders/"+orderID+"/compensate-driver", orderID, adminID, []string{"admin"},
		`{"amount":`+strconv.FormatInt(amount, 10)+`,"note":"وافقت العمليات"}`)
}

func (f *driverFixture) requestStatus(t *testing.T, orderID, driverID string) (status string, amount *int64) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(), `
		SELECT status, amount FROM driver_compensation_requests
		WHERE order_id = $1 AND driver_id = $2`, orderID, driverID).Scan(&status, &amount); err != nil {
		t.Fatalf("لا طلبَ تعويض: %v", err)
	}
	return status, amount
}

// TestCompensation_AfterOpsApproval_PaysOnce **لا مالَ لحظةَ الضغطة — ثمّ
// موافقةٌ تدفع مرّةً واحدة.**
func TestCompensation_AfterOpsApproval_PaysOnce(t *testing.T) {
	f := newDriverFixture(t, 1)
	admin := f.armCompensation(t)
	d := f.drivers[0]
	ctx := context.Background()
	orderID := f.problemOrderAt(t, "at_dropoff", d)

	if w := f.fail(d, orderID, "customer_refused", ""); w.Code != http.StatusOK {
		t.Fatalf("الفشلُ ردّ %d: %s", w.Code, w.Body.String())
	}
	if bal, _ := f.srv.wallet.Balance(ctx, d); bal != 0 {
		t.Fatalf("قُيّد للسائق %d لحظةَ الضغطة — **والتعويضُ بعد موافقة العمليات**", bal)
	}

	// **وتراه العملياتُ في قائمة الانتظار بمبلغه المقترَح.**
	lw := f.call(f.srv.handlePendingCompensations, http.MethodGet,
		"/admin/compensations/pending", "", admin, []string{"admin"}, "")
	var list struct {
		Data struct {
			Compensations []orders.CompensationRequest `json:"compensations"`
		} `json:"data"`
	}
	_ = json.Unmarshal(lw.Body.Bytes(), &list)
	var suggested int64 = -1
	for _, c := range list.Data.Compensations {
		if c.OrderID == orderID {
			suggested = c.SuggestedAmount
		}
	}
	if suggested != 2_500 {
		t.Fatalf("المقترَحُ في القائمة %d والمتوقّع 2500 (نصفُ ٥٬٠٠٠): %s", suggested, lw.Body.String())
	}

	if w := f.compensate(orderID, admin, suggested); w.Code != http.StatusOK {
		t.Fatalf("الموافقةُ ردّت %d: %s", w.Code, w.Body.String())
	}
	if bal, _ := f.srv.wallet.Balance(ctx, d); bal != 2_500 {
		t.Fatalf("بعد الموافقة رصيدُ السائق %d والمتوقّع 2500", bal)
	}
	if st, amt := f.requestStatus(t, orderID, d); st != "approved" || amt == nil || *amt != 2_500 {
		t.Errorf("الطلبُ (%s · %v) بعد الموافقة", st, amt)
	}
	// **والثانيةُ تُردّ** — لا تعويضَ مرّتين.
	if w := f.compensate(orderID, admin, suggested); w.Code != http.StatusConflict ||
		errCode(t, w) != "driver_already_compensated" {
		t.Fatalf("الموافقةُ الثانية ردّت %d: %s", w.Code, w.Body.String())
	}
	if bal, _ := f.srv.wallet.Balance(ctx, d); bal != 2_500 {
		t.Fatalf("عُوِّض مرّتين: %d", bal)
	}
	// **ولا رفضَ لما قُضي فيه.**
	rw := f.call(f.srv.handleRejectCompensation, http.MethodPost, "/x", orderID, admin,
		[]string{"admin"}, `{"note":"لا"}`)
	if rw.Code != http.StatusConflict || errCode(t, rw) != "compensation_not_pending" {
		t.Errorf("رفضُ المقضيّ ردّ %d: %s", rw.Code, rw.Body.String())
	}
}

// TestCompensation_MerchantFault_ApprovalPaysReleasedDriverAndOpensClaim **تعذّرُ
// المتجر لا يُفشل الطلب ويُحرّر السائق — والموافقةُ تصله وتفتح المطالبة.**
func TestCompensation_MerchantFault_ApprovalPaysReleasedDriverAndOpensClaim(t *testing.T) {
	f := newDriverFixture(t, 1)
	admin := f.armCompensation(t)
	d := f.drivers[0]
	ctx := context.Background()
	orderID := f.problemOrderAt(t, "at_pickup", d)

	if w := f.fail(d, orderID, "merchant_closed", ""); w.Code != http.StatusOK {
		t.Fatalf("التعذّرُ ردّ %d: %s", w.Code, w.Body.String())
	}
	if st, drv := f.orderRow(t, orderID); st != "accepted" || drv != nil {
		t.Fatalf("(%s · %v) — والمتوقّع العودةُ إلى المكتب بلا سائق", st, drv)
	}
	if w := f.compensate(orderID, admin, 2_500); w.Code != http.StatusOK {
		t.Fatalf("الموافقةُ ردّت %d: %s — **والطلبُ ليس failed ولا سائقَ عليه**", w.Code, w.Body.String())
	}
	if bal, _ := f.srv.wallet.Balance(ctx, d); bal != 2_500 {
		t.Fatalf("السائقُ المحرَّر لم يصله تعويضُه: %d", bal)
	}
	var claim int64
	if err := f.pool.QueryRow(ctx, `
		SELECT amount FROM disputes WHERE order_id = $1 AND party_role = 'merchant'`,
		orderID).Scan(&claim); err != nil || claim != 2_500 {
		t.Fatalf("المطالبةُ على المتجر (%d · %v) والمتوقّع 2500", claim, err)
	}
}

// TestCompensation_RejectClosesPending **والرفضُ قرارٌ يُكتب بسبب.**
func TestCompensation_RejectClosesPending(t *testing.T) {
	f := newDriverFixture(t, 1)
	admin := f.armCompensation(t)
	d := f.drivers[0]
	orderID := f.problemOrderAt(t, "at_dropoff", d)
	if w := f.fail(d, orderID, "address_wrong", ""); w.Code != http.StatusOK {
		t.Fatalf("الفشلُ ردّ %d", w.Code)
	}
	if w := f.call(f.srv.handleRejectCompensation, http.MethodPost, "/x", orderID, admin,
		[]string{"admin"}, `{"note":""}`); w.Code == http.StatusOK {
		t.Fatal("رُفض بلا سبب")
	}
	if w := f.call(f.srv.handleRejectCompensation, http.MethodPost, "/x", orderID, admin,
		[]string{"admin"}, `{"note":"العنوانُ صحيح — السائقُ لم يتّصل"}`); w.Code != http.StatusOK {
		t.Fatalf("الرفضُ ردّ %d: %s", w.Code, w.Body.String())
	}
	if st, _ := f.requestStatus(t, orderID, d); st != "rejected" {
		t.Errorf("الطلبُ %s بعد الرفض", st)
	}
	if bal, _ := f.srv.wallet.Balance(context.Background(), d); bal != 0 {
		t.Errorf("قُيّد مالٌ مع الرفض: %d", bal)
	}
}

// TestEmergency_NotReofferedToReporter **«لا يُعرض على السائق نفسِه ثانيةً».**
//
// **قِيس على التجهيز**: طارئٌ بعد الاستلام فعُرض الطلبُ على من أبلغ عنه —
// التحريرُ قبل إغلاق الدوام، والاستثناءُ لم يكن يشمل ما بعد الاستلام.
func TestEmergency_NotReofferedToReporter(t *testing.T) {
	f := newDriverFixture(t, 1)
	armRotation(t, f, 0)
	d := f.drivers[0]
	f.onShift(t, d, true)
	orderID := f.problemOrderAt(t, "picked_up", d)

	if code := f.emergency(t, d, orderID, `{"note":"حادث"}`); code >= 400 {
		t.Fatalf("الطارئُ ردّ %d", code)
	}
	var offered *string
	var onShift bool
	if err := f.pool.QueryRow(context.Background(), `
		SELECT o.offered_driver_id::text, u.on_shift
		FROM orders o, users u WHERE o.id = $1 AND u.id = $2`, orderID, d).
		Scan(&offered, &onShift); err != nil {
		t.Fatal(err)
	}
	if offered != nil && *offered == d {
		t.Fatal("عُرض الطلبُ على من أبلغ عن الطارئ نفسِه")
	}
	if onShift {
		t.Error("بقي على دوامه بعد الطارئ")
	}
}

// TestEmergency_OtherOrdersReleasedOrFlagged **طلباتُه الأخرى لا تُترك صامتة.**
func TestEmergency_OtherOrdersReleasedOrFlagged(t *testing.T) {
	f := newDriverFixture(t, 1)
	ops := f.armOps(t)
	d := f.drivers[0]
	hit := f.problemOrderAt(t, "picked_up", d)
	notYet := f.problemOrderAt(t, "assigned", d)
	carrying := f.problemOrderAt(t, "on_the_way", d)

	if code := f.emergency(t, d, hit, `{"note":"حادث"}`); code >= 400 {
		t.Fatalf("الطارئُ ردّ %d", code)
	}
	if st, drv := f.orderRow(t, notYet); st != "dispatching" || drv != nil {
		t.Errorf("الطلبُ الذي لم يُستلَم (%s · %v) — والمتوقّع تحريرُه", st, drv)
	}
	if st, drv := f.orderRow(t, carrying); st != "on_the_way" || drv == nil || *drv != d {
		t.Errorf("الطلبُ الذي معه (%s · %v) — **البضاعةُ معه فلا يُحرَّر**", st, drv)
	}
	alerts := strings.Join(f.opsAlerts(t, ops, hit), "\n")
	for _, id := range []string{notYet, carrying} {
		var n int64
		_ = f.pool.QueryRow(context.Background(), `SELECT number FROM orders WHERE id = $1`, id).Scan(&n)
		if !strings.Contains(alerts, "#"+strconv.FormatInt(n, 10)) {
			t.Errorf("التنبيهُ لا يذكر #%d: %q", n, alerts)
		}
	}
}

// TestEmergency_PickupNoteReachesNextDriver **كلمةُ الطارئ تصل التالي — بلا موقعٍ أيضاً.**
func TestEmergency_PickupNoteReachesNextDriver(t *testing.T) {
	f := newDriverFixture(t, 2)
	first, next := f.drivers[0], f.drivers[1]
	orderID := f.problemOrderAt(t, "picked_up", first)
	if code := f.emergency(t, first, orderID, `{"note":"تعطّلت"}`); code >= 400 {
		t.Fatalf("الطارئُ ردّ %d", code)
	}
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET status = 'assigned', driver_id = $2 WHERE id = $1`, orderID, next); err != nil {
		t.Fatal(err)
	}
	w := f.call(f.srv.handleDriverOrders, http.MethodGet, "/driver/orders", "", next, []string{"driver"}, "")
	var body struct {
		Data []struct {
			ID            string `json:"id"`
			PickupNote    string `json:"pickup_note"`
			PickupAddress string `json:"pickup_address"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("%v: %s", err, w.Body.String())
	}
	found := false
	for _, o := range body.Data {
		if o.ID != orderID {
			continue
		}
		found = true
		if o.PickupNote == "" || o.PickupAddress != o.PickupNote {
			t.Errorf("pickup_note=%q pickup_address=%q — **والعنوانُ يقول المتجرَ والبضاعةُ ليست فيه**",
				o.PickupNote, o.PickupAddress)
		}
	}
	if !found {
		t.Fatalf("الطلبُ لا يظهر للسائق التالي: %s", w.Body.String())
	}
	// **وحمولةُ الطلب للسائق تحملها أيضاً** (`AudienceDriver`).
	o, err := f.srv.orders.GetByID(context.Background(), orderID)
	if err != nil {
		t.Fatal(err)
	}
	if v := orders.ViewFor(orders.AudienceDriver, o); v["pickup_note"] == nil || v["pickup_note"] == "" {
		t.Errorf("pickup_note غائبٌ عن حمولة السائق: %v", v["pickup_note"])
	}
}
