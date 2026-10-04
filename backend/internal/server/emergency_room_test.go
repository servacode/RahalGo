package server

// ══════════════════════════════════════════════════════════════════════
// **غرفةُ الطوارئ** — قراراتُ المالك ٢٠٢٦-١٠-٠٤
// ══════════════════════════════════════════════════════════════════════
//
// صندوقٌ واحدٌ بنوعٍ مخزَّن · الطارئُ قبل الاستلام فيه · عدّادٌ يحمرّ · صفحةٌ لكلّ طارئ ·
// خطواتُ حلٍّ بترتيبها والمالُ طلبُ تعويضٍ لا دفع · الزبونُ يُخبَر · طلباتُ المتجر المغلق ·
// و«تعطّلت درّاجتي» تحرّر طلباتِه الأخرى.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func (f *driverFixture) roomCustomer(t *testing.T, orderID string) string {
	t.Helper()
	var c string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT customer_id::text FROM orders WHERE id = $1`, orderID).Scan(&c); err != nil {
		t.Fatal(err)
	}
	return c
}

func (f *driverFixture) customerGot(t *testing.T, customer, body string) bool {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM notifications WHERE user_id = $1 AND body = $2`, customer, body).
		Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n > 0
}

func (f *driverFixture) roomRow(t *testing.T, orderID string) (id, kind, stage string) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(), `
		SELECT id::text, kind, stage FROM driver_emergencies
		 WHERE order_id = $1 AND status = 'open'`, orderID).Scan(&id, &kind, &stage); err != nil {
		t.Fatalf("لا طارئَ مفتوحٌ للطلب في الغرفة: %v", err)
	}
	return
}

func (f *driverFixture) admin(h http.HandlerFunc, method, id, user, body string) (int, map[string]any) {
	w := f.call(h, method, "/x", id, user, []string{"admin"}, body)
	var out struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out.Data
}

// TestRoom_ReleaseBeforePickup_EntersRoom_FreesOthers_TellsCustomer **«تعطّلت درّاجتي»
// قبل الاستلام تدخل الغرفة بنوعها، وتحرّر طلباتِه الأخرى غيرَ المستلَمة، ويُخبَر الزبون.**
func TestRoom_ReleaseBeforePickup_EntersRoom_FreesOthers_TellsCustomer(t *testing.T) {
	f := newDriverFixture(t, 1)
	f.armOps(t)
	d := f.drivers[0]
	f.onShift(t, d, true)
	a := f.problemOrderAt(t, "assigned", d)
	b := f.problemOrderAt(t, "assigned", d)
	c := f.problemOrderAt(t, "picked_up", d)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM driver_emergencies WHERE driver_id = $1`, d)
	})

	w := f.release(d, a, `{"reason":"bike_broken","note":"انكسر الجنزير"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("التركُ ردّ %d: %s", w.Code, w.Body.String())
	}
	_, kind, stage := f.roomRow(t, a)
	if kind != EmergencyBreakdown || stage != "before_pickup" {
		t.Fatalf("النوع %q والموضع %q — والمتوقّع عطلٌ قبل الاستلام", kind, stage)
	}
	if st, drv := f.orderRow(t, b); st != "dispatching" || drv != nil {
		t.Fatalf("طلبُه الآخرُ غيرُ المستلَم بقي معه (%s · %v)", st, drv)
	}
	if st, drv := f.orderRow(t, c); st != "picked_up" || drv == nil {
		t.Fatalf("المستلَمُ تحرّك (%s · %v) — والبضاعةُ معه", st, drv)
	}
	if !f.customerGot(t, f.roomCustomer(t, a), emergencyCustomerRedispatch) {
		t.Fatal("الزبونُ لم يُخبَر بأنّ سائقاً آخر يُرسَل")
	}
	if !f.customerGot(t, f.roomCustomer(t, b), emergencyCustomerRedispatch) {
		t.Fatal("زبونُ الطلب الآخر لم يُخبَر")
	}
}

// TestRoom_StepsInOrder_MoneyIsRequestNotPayment **الخطواتُ بترتيبها، و«السائقُ بخير» يرفع
// قفلَ الحادث، ومصيرُ الطلب يُقرَّر ويُخبَر الزبون، والمالُ طلبُ تعويضٍ لا قيد.**
func TestRoom_StepsInOrder_MoneyIsRequestNotPayment(t *testing.T) {
	f := newDriverFixture(t, 1)
	f.armOps(t)
	ops := testdb.NewUser(t, f.pool, "ops")
	d := f.drivers[0]
	ctx := context.Background()
	order := f.problemOrderAt(t, "picked_up", d)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM driver_emergencies WHERE driver_id = $1`, d)
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM driver_compensation_requests WHERE order_id = $1`, order)
	})

	w := f.call(f.srv.handleDriverEmergency, http.MethodPost, "/x", order, d, []string{"driver"},
		`{"lat":35.95,"lng":39.0,"note":"وقعت","kind":"accident"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("الطارئ ردّ %d: %s", w.Code, w.Body.String())
	}
	id, kind, stage := f.roomRow(t, order)
	if kind != EmergencyAccident || stage != "after_pickup" {
		t.Fatalf("النوع %q والموضع %q", kind, stage)
	}
	if !f.customerGot(t, f.roomCustomer(t, order), emergencyCustomerFollowing) {
		t.Fatal("الزبونُ لم يُخبَر بعد الطارئ")
	}

	// «تمّ» قبل الخطوات يُردّ.
	if code, _ := f.admin(f.srv.handleResolveEmergency, http.MethodPost, id, ops,
		`{"resolution":"x"}`); code != http.StatusConflict {
		t.Fatalf("أُغلق قبل خطواته (%d)", code)
	}
	if code, _ := f.admin(f.srv.handleAckEmergency, http.MethodPost, id, ops, `{}`); code != 200 {
		t.Fatalf("الاستلام %d", code)
	}
	if !f.srv.driverAccidentLocked(ctx, d) {
		t.Fatal("الحادثُ لم يقفل الدوام")
	}
	if code, out := f.admin(f.srv.handleEmergencyDriverOK, http.MethodPost, id, ops,
		`{"note":"اتّصلتُ به وهو بخير"}`); code != 200 || out["lock_cleared"] != true {
		t.Fatalf("السائقُ بخير ردّ %d %v", code, out)
	}
	if f.srv.driverAccidentLocked(ctx, d) {
		t.Fatal("القفلُ باقٍ بعد «السائقُ بخير»")
	}
	// مصيرُ الطلب بلا سببٍ يُردّ، وبه يُرسَل لسائقٍ آخر.
	if code, _ := f.admin(f.srv.handleEmergencyOutcome, http.MethodPost, id, ops,
		`{"outcome":"redispatch"}`); code != http.StatusBadRequest {
		t.Fatalf("مصيرٌ بلا سببٍ ردّ %d", code)
	}
	if code, out := f.admin(f.srv.handleEmergencyOutcome, http.MethodPost, id, ops,
		`{"outcome":"redispatch","note":"يأخذها سائقٌ من موضعه"}`); code != 200 {
		t.Fatalf("المصير ردّ %d %v", code, out)
	}
	if st, _ := f.orderRow(t, order); st != "dispatching" {
		t.Fatalf("الطلبُ %s لا في الطابور", st)
	}
	if !f.customerGot(t, f.roomCustomer(t, order), emergencyCustomerRedispatch) {
		t.Fatal("الزبونُ لم يُخبَر بالسائق الآخر")
	}
	// المال: طلبُ تعويضٍ معلَّقٌ بذنب المنصّة — ولا قيدَ في المحفظة.
	if code, out := f.admin(f.srv.handleEmergencyMoney, http.MethodPost, id, ops,
		`{"amount":3000,"note":"أجرةُ المشوار"}`); code != 200 {
		t.Fatalf("المال ردّ %d %v", code, out)
	}
	var status, fault string
	var suggested int64
	if err := f.pool.QueryRow(ctx, `
		SELECT r.status, r.fault, r.suggested_amount
		  FROM driver_emergencies e JOIN driver_compensation_requests r ON r.id = e.money_request_id
		 WHERE e.id = $1`, id).Scan(&status, &fault, &suggested); err != nil {
		t.Fatalf("لا طلبَ تعويضٍ مربوط: %v", err)
	}
	if status != "pending" || fault != "platform" || suggested != 3000 {
		t.Fatalf("الطلب (%s · %s · %d)", status, fault, suggested)
	}
	var paid int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'compensation'`, order).Scan(&paid)
	if paid != 0 {
		t.Fatal("دُفع مالٌ من الغرفة — والقرارُ للماليّة")
	}
	// سجلُّ الملاحظات يُضاف إليه ولا يُعدَّل.
	if code, _ := f.admin(f.srv.handleEmergencyNote, http.MethodPost, id, ops,
		`{"body":"اتّصل الزبونُ وطمأنّاه"}`); code != http.StatusCreated {
		t.Fatalf("الملاحظة %d", code)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE emergency_notes SET body = 'x' WHERE emergency_id = $1`, id); err == nil {
		t.Fatal("عُدّلت ملاحظة — والسجلُّ يُضاف إليه فقط")
	}
	if _, err := f.pool.Exec(ctx, `UPDATE users SET last_location =
		ST_SetSRID(ST_MakePoint(39.01, 35.96), 4326)::geography, last_location_at = now()
		WHERE id = $1`, d); err != nil {
		t.Fatal(err)
	}
	code, detail := f.admin(f.srv.handleEmergencyDetail, http.MethodGet, id, ops, "")
	if code != 200 || detail["next_step"] != "resolve" {
		t.Fatalf("الصفحة %d · الخطوةُ التالية %v", code, detail["next_step"])
	}
	if notes, _ := detail["notes"].([]any); len(notes) < 3 {
		t.Fatalf("الملاحظاتُ %d", len(notes))
	}
	if drv, _ := detail["driver"].(map[string]any); drv == nil || drv["lat"] == nil {
		t.Fatalf("لا موقعَ حيٌّ للسائق في الصفحة: %v", detail["driver"])
	}
	if code, _ := f.admin(f.srv.handleResolveEmergency, http.MethodPost, id, ops,
		`{"resolution":"أوصله سائقٌ آخر"}`); code != 200 {
		t.Fatalf("«تمّ» بعد الخطوات ردّ %d", code)
	}
}

// TestRoom_StoreClosure_CountStale_Halt **إغلاقُ المتجر يدخل الغرفة بطلباته، والاستلامُ في
// الموضعين، والعدّادُ يحمرّ بعد المهلة، وتوقّفُ المنصّة يُفتح ويُغلق بعودتها.**
func TestRoom_StoreClosure_CountStale_Halt(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	ops := testdb.NewUser(t, f.pool, "ops")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2 WHERE id = $1`,
		f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	waiting := f.dispatchingOrder(t, 10_000, 2_000)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM driver_emergencies WHERE kind = 'platform_halt'`)
	})

	w := f.call(f.srv.handleMerchantEmergency, http.MethodPost, "/x", f.merchantID, owner,
		[]string{"merchant"}, `{"closed":true}`)
	if w.Code != 200 {
		t.Fatalf("الإغلاق %d %s", w.Code, w.Body.String())
	}
	var id string
	if err := f.pool.QueryRow(ctx, `SELECT id::text FROM driver_emergencies
		WHERE merchant_id = $1 AND kind = 'store_closure' AND status = 'open'`, f.merchantID).
		Scan(&id); err != nil {
		t.Fatalf("الإغلاقُ لم يدخل الغرفة: %v", err)
	}
	code, detail := f.admin(f.srv.handleEmergencyDetail, http.MethodGet, id, ops, "")
	if code != 200 {
		t.Fatalf("الصفحة %d", code)
	}
	found := false
	for _, o := range detail["store_orders"].([]any) {
		if o.(map[string]any)["id"] == waiting {
			found = true
		}
	}
	if !found {
		t.Fatal("طلبُ المتجر المفتوح لا يظهر في الغرفة")
	}

	// العدّاد: يحمرّ بعد المهلة بلا مستلِم.
	if _, err := f.pool.Exec(ctx, `UPDATE driver_emergencies
		SET created_at = now() - interval '11 minutes' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	_, cnt := f.admin(f.srv.handleEmergencyCount, http.MethodGet, "", ops, "")
	if cnt["stale"].(float64) < 1 {
		t.Fatalf("العدّادُ لم يحمرّ: %v", cnt)
	}
	if code, _ := f.admin(f.srv.handleAckEmergency, http.MethodPost, id, ops, `{}`); code != 200 {
		t.Fatalf("الاستلام %d", code)
	}
	var merchantAcked bool
	_ = f.pool.QueryRow(ctx, `SELECT emergency_ack_at IS NOT NULL FROM merchants WHERE id = $1`,
		f.merchantID).Scan(&merchantAcked)
	if !merchantAcked {
		t.Fatal("الاستلامُ في الغرفة لم يُسكت شريطَ المتجر")
	}

	f.srv.recordPlatformHalt(ctx, true, "صيانة", ops)
	var halts int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM driver_emergencies
		WHERE kind = 'platform_halt' AND status = 'open'`).Scan(&halts)
	if halts != 1 {
		t.Fatalf("توقّفُ المنصّة %d صفّاً مفتوحاً", halts)
	}
	f.srv.recordPlatformHalt(ctx, false, "", ops)
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM driver_emergencies
		WHERE kind = 'platform_halt' AND status = 'open'`).Scan(&halts)
	if halts != 0 {
		t.Fatal("التوقّفُ بقي مفتوحاً بعد عودة المنصّة")
	}
}
