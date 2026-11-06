package qa

// ══════════════════════════════════════════════════════════════════════
// **السائقُ في طلبٍ بعدّة محطّات — وصولٌ واستلامٌ لكلّ متجر** (الدفعة P4)
// ══════════════════════════════════════════════════════════════════════
//
// **حمولةُ المحطّات** · **الاستلامُ يسوّي متجرَ المحطّة وحدَه** (قرارُ المالك ٣أ) · **ومن استلم
// محطّةً لا يترك الباقي** · **وشيلُ آخر محطّةٍ لم تُستلم يُكمل الاستلام** · **٤٢٦ لمن لا يعرف
// المحطّات** · **والطارئُ بين المحطّتين: سائقٌ آخرُ يستلم البضاعة، أو شيلُ الباقي** (٥أ) ·
// **والبضاعةُ للمكتب لا للمتجر** (٤أ) · **والمكتبُ يختار المتجرَ المذنب في التعويض** (٦أ).

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// ── أدواتٌ مشتركة ─────────────────────────────────────────────────────

// msStep **خطوةُ السائق بمحطّتها** — وفارغُ `stop` خطوةٌ بلا محطّة (تطبيقٌ قديم).
func msStep(h *Harness, oid string, drv *User, to, stop string) Res {
	body := map[string]any{"to": to}
	if stop != "" {
		body["stop"] = stop
	}
	return h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token, body)
}

// mustStep **خطوةٌ يُنتظر قبولُها** — ويردّ حالَ الطلب في الردّ.
func mustStep(t *testing.T, h *Harness, oid string, drv *User, to, stop string) string {
	t.Helper()
	r := msStep(h, oid, drv, to, stop)
	if r.Code != http.StatusOK {
		t.Fatalf("الخطوةُ %s للمحطّة %s: %s", to, stop, r)
	}
	st, _ := r.JSON()["status"].(string)
	return st
}

// stopsToCollect **محطّاتٌ حيّةٌ لم يستلمها هذا السائق** — بترتيبها.
func stopsToCollect(t *testing.T, h *Harness, oid, driverID string) []string {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT merchant_id::text FROM order_parts
		WHERE order_id = $1::uuid AND (status = 'accepted'
		      OR (status = 'picked_up' AND picked_up_by IS DISTINCT FROM $2::uuid))
		ORDER BY seq`, oid, driverID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	return out
}

// pickupAllStops **وصولٌ واستلامٌ لكلّ محطّةٍ باقية** — بنسخة السائق التي تعرف المحطّات.
func pickupAllStops(t *testing.T, h *Harness, oid string, drv *User) {
	t.Helper()
	for _, stop := range stopsToCollect(t, h, oid, drv.ID) {
		mustStep(t, h, oid, drv, "at_pickup", stop)
		mustStep(t, h, oid, drv, "picked_up", stop)
	}
}

// deliverMultiStop **يمشي طلباً حتّى التسليم بنسخة السائق التي تعرف المحطّات** — وصولٌ واستلامٌ لكلّ
// محطّةٍ حيّةٍ لم يستلمها هو (بترتيبها)، ثمّ الطريقُ والبابُ والتسليم.
//
// **متسامحٌ عمداً** (يستعمله اختبارُ مال المندوب أيضاً): **بلا سائقٍ يُسنَد سائقٌ جديد**؛ **وطلبُ
// المتجر الواحد يمشي كما يمشي اليوم** (`deliverOrder`)؛ **وطلبٌ استُلم سلفاً يُكمل من حيث هو.**
func deliverMultiStop(t *testing.T, s *msShop, oid string, drv ...*User) {
	t.Helper()
	var d *User
	if len(drv) > 0 && drv[0] != nil {
		d = drv[0]
	} else {
		d = s.h.driverOf(oid)
	}
	deliverByStops(t, s.h, oid, d)
}

// deliverByStops **كـ`deliverMultiStop` لاختبارٍ بلا `msShop`** — بسائقٍ مُسنَدٍ سلفاً.
func deliverByStops(t *testing.T, h *Harness, oid string, d *User) {
	t.Helper()
	var stops int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT stops FROM orders WHERE id = $1::uuid`, oid).Scan(&stops); err != nil {
		t.Fatal(err)
	}
	if stops <= 1 {
		deliverOrder(t, h, oid, d)
		return
	}
	if st := h.statusOf(oid); st == "assigned" || st == "at_pickup" {
		// **وطلبٌ أُسند خاماً قبل أن تجيب متاجرُه** (`driverOf`) — أقسامُه المعلَّقةُ تُقبل خاماً مثلَه.
		if _, err := h.Pool.Exec(ctxBG(), `UPDATE order_parts SET status = 'accepted', answered_at = now()
			WHERE order_id = $1::uuid AND status = 'pending'`, oid); err != nil {
			t.Fatal(err)
		}
		pickupAllStops(t, h, oid, d)
	}
	for _, to := range []string{"on_the_way", "at_dropoff"} {
		if h.statusOf(oid) == to {
			continue
		}
		if r := msStep(h, oid, d, to, ""); r.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s: %s", to, r)
		}
	}
	h.authorizeProofExempt(oid)
	if r := msStep(h, oid, d, "delivered", ""); r.Code >= 400 {
		t.Fatalf("التسليم: %s", r)
	}
}

// setMethod **طريقةُ تسوية المتجر** — قبل إنشاء الطلب (تُلقط في بنوده).
func setMethod(t *testing.T, h *Harness, m *Merchant, method string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET settlement_method = $2 WHERE id = $1::uuid`,
		m.ID, method); err != nil {
		t.Fatal(err)
	}
}

// acceptedMS **طلبٌ من المتجرين قبله المتجران** (وضعُ «المتاجر تدير») — ومعه سائقٌ مُسنَد.
func acceptedMS(t *testing.T, s *msShop) (string, *User) {
	t.Helper()
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
	return oid, s.h.driverOf(oid)
}

// driverListed **الطلبُ كما في قائمة السائق.**
func driverListed(t *testing.T, h *Harness, drv *User, oid string) map[string]any {
	t.Helper()
	r := h.GET("/api/v1/driver/orders", drv.Token)
	if r.Code != http.StatusOK {
		t.Fatalf("قائمةُ السائق: %s", r)
	}
	var env struct {
		Data []map[string]any `json:"data"`
	}
	var list []map[string]any
	if json.Unmarshal(r.Body, &env) == nil && env.Data != nil {
		list = env.Data
	} else {
		_ = json.Unmarshal(r.Body, &list)
	}
	for _, o := range list {
		if o["id"] == oid {
			return o
		}
	}
	t.Fatalf("الطلبُ ليس في قائمة السائق")
	return nil
}

func stopOf(o map[string]any, mid string) map[string]any {
	list, _ := o["stops"].([]any)
	for _, v := range list {
		if m, ok := v.(map[string]any); ok && strings.EqualFold(m["merchant_id"].(string), mid) {
			return m
		}
	}
	return nil
}

type msSettle struct {
	n      int
	amount int64
	pct    *float64
	state  string
}

func settleOf(t *testing.T, h *Harness, oid, mid string) msSettle {
	t.Helper()
	var r msSettle
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*), COALESCE(max(amount), 0), max(commission_percent)::float8, COALESCE(max(state), '')
		FROM merchant_settlements WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, mid).
		Scan(&r.n, &r.amount, &r.pct, &r.state); err != nil {
		t.Fatal(err)
	}
	return r
}

func partStatusBy(t *testing.T, h *Harness, oid, mid string) (status, by string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT status, COALESCE(picked_up_by::text, '') FROM order_parts
		WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, mid).Scan(&status, &by); err != nil {
		t.Fatal(err)
	}
	return status, by
}

func lineOf(t *testing.T, h *Harness, oid, mid string) string {
	t.Helper()
	var id string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT id::text FROM order_items WHERE order_id = $1::uuid AND merchant_id = $2::uuid LIMIT 1`, oid, mid).
		Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// clearCode **يمحو كودَ التسليم بعد فحصه** — فيُسلَّم الطلبُ بلا إملاء الزبون.
func clearCode(t *testing.T, h *Harness, oid string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET delivery_code = NULL WHERE id = $1::uuid`, oid); err != nil {
		t.Fatal(err)
	}
}

// ── الاختبارات ───────────────────────────────────────────────────────

// TestMultiStop_DriverStopsPayloadIncludesWalletStore **حمولةُ المحطّات** — `T12` (G-10).
//
// **كان السائقُ يرى متجراً واحداً ورقماً واحداً يدفعه** — ومتجرُ المحفظة يُقرأ كأنّه يُدفع.
func TestMultiStop_DriverStopsPayloadIncludesWalletStore(t *testing.T) {
	s := newMSShop(t, "merchants")
	setMethod(t, s.h, s.mA, "cash")
	setMethod(t, s.h, s.mB, "wallet")
	oid, drv := acceptedMS(t, s)

	o := driverListed(t, s.h, drv, oid)
	a, b := stopOf(o, s.mA.ID), stopOf(o, s.mB.ID)
	if a == nil || b == nil {
		t.Fatalf("**المحطّتان غائبتان من الحمولة**: %v", o["stops"])
	}
	if a["pays_by"] != "cash" || a["pay_amount"].(float64) != 900 {
		t.Errorf("محطّةُ النقد: pays_by=%v pay_amount=%v — يُنتظر cash و900 (١٠٠٠ − ١٠٪)", a["pays_by"], a["pay_amount"])
	}
	if b["pays_by"] != "wallet" || b["pay_amount"].(float64) != 0 {
		t.Errorf("**محطّةُ المحفظة يُدفع لها**: pays_by=%v pay_amount=%v — يُنتظر wallet و0", b["pays_by"], b["pay_amount"])
	}
	if items, _ := a["items"].([]any); len(items) != 1 {
		t.Errorf("أصنافُ محطّة الأوّل %v — يُنتظر صنفُه وحدَه", a["items"])
	}
	if o["next_stop"] != s.mA.ID {
		t.Errorf("next_stop=%v — يُنتظر الأوّل", o["next_stop"])
	}
	if a["mine"] != false || a["handover"] != false || a["status"] != "accepted" {
		t.Errorf("حالُ المحطّة الأولى: %v", a)
	}
	if o["merchant_name"] != a["name"] {
		t.Errorf("**الحقولُ المسطّحةُ ليست المحطّةَ التالية**: %v ≠ %v", o["merchant_name"], a["name"])
	}

	// **وبعد استلام الأولى صارت التاليةُ الثانية** — والردُّ يحمل المحطّات.
	mustStep(t, s.h, oid, drv, "at_pickup", s.mA.ID)
	r := msStep(s.h, oid, drv, "picked_up", s.mA.ID)
	if r.Code != http.StatusOK {
		t.Fatal(r)
	}
	if r.JSON()["next_stop"] != s.mB.ID {
		t.Errorf("ردُّ الاستلام: next_stop=%v — يُنتظر الثاني", r.JSON()["next_stop"])
	}
	o = driverListed(t, s.h, drv, oid)
	if stopOf(o, s.mA.ID)["mine"] != true || o["merchant_name"] != stopOf(o, s.mB.ID)["name"] {
		t.Errorf("بعد الأولى: mine=%v merchant_name=%v", stopOf(o, s.mA.ID)["mine"], o["merchant_name"])
	}
}

// TestMultiStop_PerStopPickupSettlesThatStoreOnly **الاستلامُ يسوّي متجرَ المحطّة وحدَه** — `T13`
// (قرارُ المالك ٣أ): **ونسبتُه في صفّه، وكودٌ واحدٌ عند آخر محطّة.**
func TestMultiStop_PerStopPickupSettlesThatStoreOnly(t *testing.T) {
	s := newMSShop(t, "merchants")
	s.h.Setting("delivery.code_required", "true")
	base := financialBaseline(t, s.h)
	setMethod(t, s.h, s.mA, "cash")
	setMethod(t, s.h, s.mB, "wallet")
	oid, drv := acceptedMS(t, s)

	if st := mustStep(t, s.h, oid, drv, "at_pickup", s.mA.ID); st != "at_pickup" {
		t.Fatalf("أوّلُ وصولٍ: الحالُ %q — يُنتظر at_pickup", st)
	}
	if st := mustStep(t, s.h, oid, drv, "picked_up", s.mA.ID); st != "at_pickup" {
		t.Fatalf("**استلامُ المحطّة الأولى نقل الطلبَ كلَّه**: %q", st)
	}
	a := settleOf(t, s.h, oid, s.mA.ID)
	if a.n != 1 || a.amount != 900 || a.state != "cash_paid" || a.pct == nil || *a.pct != 10 {
		t.Errorf("صفُّ الأوّل بعد محطّته: %+v — يُنتظر صفٌّ واحدٌ ٩٠٠ مدفوعٌ بنسبة ١٠", a)
	}
	if b := settleOf(t, s.h, oid, s.mB.ID); b.n != 0 {
		t.Errorf("**سُوّي الثاني قبل محطّته**: %+v", b)
	}
	var code *string
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT delivery_code FROM orders WHERE id = $1::uuid`, oid).Scan(&code)
	if code != nil {
		t.Errorf("**كودُ التسليم قبل آخر محطّة**")
	}

	mustStep(t, s.h, oid, drv, "at_pickup", s.mB.ID)
	if st := mustStep(t, s.h, oid, drv, "picked_up", s.mB.ID); st != "picked_up" {
		t.Fatalf("آخرُ محطّة: الحالُ %q — يُنتظر picked_up", st)
	}
	b := settleOf(t, s.h, oid, s.mB.ID)
	if b.n != 1 || b.amount != 400 || b.pct == nil || *b.pct != 20 {
		t.Errorf("صفُّ الثاني: %+v — يُنتظر ٤٠٠ بنسبة ٢٠", b)
	}
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT delivery_code FROM orders WHERE id = $1::uuid`, oid).Scan(&code)
	if code == nil {
		t.Errorf("لا كودَ تسليمٍ بعد آخر محطّة")
	}
	var codes int
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_events
		WHERE order_id = $1::uuid AND to_status = 'picked_up' AND merchant_id IS NULL`, oid).Scan(&codes)
	if codes != 1 {
		t.Errorf("أحداثُ «استُلم» للطلب %d — يُنتظر واحد", codes)
	}
	clearCode(t, s.h, oid)
	deliverMultiStop(t, s, oid, drv)
	if st := s.h.statusOf(oid); st != "delivered" {
		t.Fatalf("الحال %q", st)
	}
	if a2 := settleOf(t, s.h, oid, s.mA.ID); a2.n != 1 || a2.amount != 900 {
		t.Errorf("صفُّ الأوّل بعد التسليم: %+v", a2)
	}
	assertNewViolations(t, s.h, base)
}

// TestMultiStop_CannotReleaseOrCancelAfterOneStop **من استلم محطّةً لا يترك الباقي** — `T14` (G5).
func TestMultiStop_CannotReleaseOrCancelAfterOneStop(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid, drv := acceptedMS(t, s)
	mustStep(t, s.h, oid, drv, "at_pickup", s.mA.ID)
	// **«المتجرُ مغلق» عند محطّةٍ وفي الطلب قسمٌ حيٌّ غيرُه** (G8) — المكتبُ يشيل القسمَ أو يحوّله.
	blocked := orders.DoorResolution{Action: orders.DoorReturnToOffice, Fault: orders.FaultMerchant,
		Note: "المحل مسكّر", Reason: "merchant_closed"}
	if _, err := s.h.Orders.ResolveDoor(ctxBG(), s.admin.ID, []string{"admin"}, oid, blocked, nil); !errors.Is(err, orders.ErrUsePart) {
		t.Errorf("إنهاءُ المكتب عند المحطّة الأولى: %v — يُنتظر use_part", err)
	}
	mustStep(t, s.h, oid, drv, "picked_up", s.mA.ID)

	checks := []struct {
		label string
		r     Res
	}{
		{"تركُ السائق", s.h.POST("/api/v1/driver/orders/"+oid+"/release", drv.Token,
			map[string]any{"reason": "bike_broken", "note": "تعطلت"})},
		{"إلغاءُ الأدمن", s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
			map[string]any{"to": "cancelled", "note": "اختبار"})},
		{"إعادتُه للطابور", s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
			map[string]any{"to": "dispatching", "note": "اختبار"})},
	}
	for _, c := range checks {
		if c.r.Code != http.StatusConflict || c.r.Err() != "stops_in_hand" {
			t.Errorf("%s: %s — يُنتظر 409 stops_in_hand", c.label, c.r)
		}
	}
	if _, err := s.h.Orders.ResolveDoor(ctxBG(), s.admin.ID, []string{"admin"}, oid, blocked, nil); !errors.Is(err, orders.ErrStopsInHand) {
		t.Errorf("إنهاءُ المكتب والبضاعةُ معه: %v — يُنتظر stops_in_hand", err)
	}
	if st := s.h.statusOf(oid); st != "at_pickup" {
		t.Errorf("الحالُ تبدّل: %q", st)
	}
}

// TestMultiStop_RemoveLastUnpickedCompletesPickup **شيلُ آخر محطّةٍ لم تُستلم يُكمل الاستلام** — `T15`.
func TestMultiStop_RemoveLastUnpickedCompletesPickup(t *testing.T) {
	s := newMSShop(t, "merchants")
	s.h.Setting("delivery.code_required", "true")
	base := financialBaseline(t, s.h)
	oid, drv := acceptedMS(t, s)
	mustStep(t, s.h, oid, drv, "at_pickup", s.mA.ID)
	mustStep(t, s.h, oid, drv, "picked_up", s.mA.ID)
	if r := s.partPost(s.admin.Token, oid, s.mB.ID, "remove",
		map[string]any{"reason": "المحل سكّر", "fault": "platform"}); r.Code != http.StatusOK {
		t.Fatalf("شيلُ الثاني: %s", r)
	}
	if st := s.h.statusOf(oid); st != "picked_up" {
		t.Fatalf("**بقي الطلبُ عند المحطّات وكلُّ ما بقي مع السائق**: %q", st)
	}
	var code *string
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT delivery_code FROM orders WHERE id = $1::uuid`, oid).Scan(&code)
	if code == nil {
		t.Errorf("لا كودَ تسليم")
	}
	clearCode(t, s.h, oid)
	deliverMultiStop(t, s, oid, drv)
	if st := s.h.statusOf(oid); st != "delivered" {
		t.Fatalf("الحال %q", st)
	}
	assertNewViolations(t, s.h, base)
}

// TestMultiStop_StopLessCallsGet426 **نسخةٌ لا تعرف المحطّات** — `T16` (D1): خطوةٌ بلا محطّة ٤٢٦،
// والقبولُ ٤٢٦ لنسخةٍ دون نسخة المحطّات، **وإعادةُ «استلمت» للمحطّة نفسِها لا تكتب شيئاً،
// و«استلمت» قبل «وصلت» ٤٠٩.**
func TestMultiStop_StopLessCallsGet426(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
	if st := waitStatusIs(t, s.h, oid, "dispatching", 3*time.Second); st != "dispatching" {
		t.Fatalf("الطلبُ لم ينزل: %q", st)
	}
	// **وأدنى نسخةٍ تنزل** — فيبقى حزامُ القبول وحدَه بين النسخة القديمة والطلب.
	s.h.Setting("app.min_version.driver", "0")
	drv := s.f.Driver(OnShift())
	t.Cleanup(func() {
		_, _ = s.h.Pool.Exec(ctxBG(), `UPDATE users SET on_shift = false WHERE id = $1::uuid`, drv.ID)
	})
	old := s.h.Call("POST", "/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil, driverApp("22"))
	if old.Code != http.StatusUpgradeRequired || old.Err() != "update_required" {
		t.Fatalf("**قبولُ نسخةٍ لا تعرف المحطّات**: %s — يُنتظر 426", old)
	}
	if r := s.h.Call("POST", "/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil,
		driverApp(multiStopBuild)); r.Code != http.StatusOK {
		t.Fatalf("قبولُ النسخة الجديدة: %s", r)
	}
	if r := msStep(s.h, oid, drv, "at_pickup", ""); r.Code != http.StatusUpgradeRequired || r.Err() != "update_required" {
		t.Errorf("«وصلت» بلا محطّة: %s — يُنتظر 426", r)
	}
	mustStep(t, s.h, oid, drv, "at_pickup", s.mA.ID)
	if r := msStep(s.h, oid, drv, "picked_up", ""); r.Code != http.StatusUpgradeRequired {
		t.Errorf("«استلمت» بلا محطّة: %s — يُنتظر 426", r)
	}
	if r := msStep(s.h, oid, drv, "picked_up", s.mB.ID); r.Code != http.StatusConflict || r.Err() != "stop_not_arrived" {
		t.Errorf("«استلمت» الثانية قبل وصولها: %s — يُنتظر 409 stop_not_arrived", r)
	}
	mustStep(t, s.h, oid, drv, "picked_up", s.mA.ID)
	var events int
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_events WHERE order_id = $1::uuid`, oid).Scan(&events)
	// **الإعادةُ بالمحطّة نفسِها** — ردٌّ ٢٠٠ ولا شيءَ يُكتب، والثانيةُ لم تُمسّ.
	if r := msStep(s.h, oid, drv, "picked_up", s.mA.ID); r.Code != http.StatusOK {
		t.Errorf("إعادةُ «استلمت» الأولى: %s — يُنتظر 200", r)
	}
	if r := msStep(s.h, oid, drv, "at_pickup", s.mA.ID); r.Code != http.StatusOK {
		t.Errorf("إعادةُ «وصلت» الأولى: %s — يُنتظر 200", r)
	}
	var after int
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_events WHERE order_id = $1::uuid`, oid).Scan(&after)
	if after != events {
		t.Errorf("**الإعادةُ كتبت أحداثاً**: %d ← %d", events, after)
	}
	if st, _ := partStatusBy(t, s.h, oid, s.mB.ID); st != "accepted" {
		t.Errorf("**الإعادةُ استلمت الثانية**: %q", st)
	}
	if a := settleOf(t, s.h, oid, s.mA.ID); a.n != 1 {
		t.Errorf("صفوفُ الأوّل %d", a.n)
	}
	// **ومحطّةٌ ليست في الطلب** ٤٠٤.
	if r := msStep(s.h, oid, drv, "at_pickup", "00000000-0000-0000-0000-000000000000"); r.Code != http.StatusNotFound {
		t.Errorf("محطّةٌ غريبة: %s", r)
	}
	deliverMultiStop(t, s, oid, drv)
}

// TestMultiStop_CommissionChangeBetweenStops **نسبةُ الأوّل تُجمَّد عند محطّته** — `T24` (M1).
func TestMultiStop_CommissionChangeBetweenStops(t *testing.T) {
	s := newMSShop(t, "merchants")
	base := financialBaseline(t, s.h)
	setMethod(t, s.h, s.mA, "cash")
	setMethod(t, s.h, s.mB, "cash")
	oid, drv := acceptedMS(t, s)
	mustStep(t, s.h, oid, drv, "at_pickup", s.mA.ID)
	mustStep(t, s.h, oid, drv, "picked_up", s.mA.ID)
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE merchants SET commission_percent = 5 WHERE id = $1::uuid`,
		s.mA.ID); err != nil {
		t.Fatal(err)
	}
	pickupAllStops(t, s.h, oid, drv)
	deliverMultiStop(t, s, oid, drv)
	if st := s.h.statusOf(oid); st != "delivered" {
		t.Fatalf("**تبديلُ النسبة بين المحطّتين أوقف التسليم**: %q", st)
	}
	if a := settleOf(t, s.h, oid, s.mA.ID); a.amount != 900 || a.pct == nil || *a.pct != 10 {
		t.Errorf("صفُّ الأوّل: %+v — يُنتظر ٩٠٠ بنسبة ١٠", a)
	}
	var pc int64
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT platform_commission FROM orders WHERE id = $1::uuid`, oid).Scan(&pc)
	if pc != 100+100 {
		t.Errorf("عمولةُ المنصّة %d — يُنتظر ٢٠٠ (١٠٠ من الأوّل بنسبته المجمّدة + ١٠٠ من الثاني)", pc)
	}
	assertNewViolations(t, s.h, base)
}

// TestMultiStop_CollectedStoreLinesFrozen **بنودُ متجرٍ استُلم لا تُمسّ** — `T25` (D4 · M8 · C2).
func TestMultiStop_CollectedStoreLinesFrozen(t *testing.T) {
	s := newMSShop(t, "merchants")
	base := financialBaseline(t, s.h)
	oid, drv := acceptedMS(t, s)
	mustStep(t, s.h, oid, drv, "at_pickup", s.mA.ID)
	mustStep(t, s.h, oid, drv, "picked_up", s.mA.ID)
	ops := s.h.NewUser("operations")
	lineA, lineB := lineOf(t, s.h, oid, s.mA.ID), lineOf(t, s.h, oid, s.mB.ID)

	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/items/"+lineA+"/unavailable", s.admin.Token, nil); r.Code < 400 {
		t.Errorf("**صنفُ متجرٍ استُلم «غير متوفّر»**: %s", r)
	}
	if r := s.partPost(s.admin.Token, oid, s.mA.ID, "remove",
		map[string]any{"reason": "اختبار", "fault": "platform"}); r.Code != http.StatusConflict {
		t.Errorf("شيلُ قسمٍ استُلم: %s", r)
	}
	other := s.f.Merchant()
	if r := s.partPost(s.admin.Token, oid, s.mA.ID, "transfer",
		map[string]any{"to_merchant_id": other.ID}); r.Code != http.StatusConflict {
		t.Errorf("تحويلُ قسمٍ استُلم: %s", r)
	}
	// **والمكتبُ يعدّل بنودَ متجرٍ لم يُستلم** — ولا يعدّل بنودَ المستلَم.
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/edit", ops.Token, map[string]any{
		"reason": "زاد الزبون", "lines": []map[string]any{{"item_id": lineA, "qty": 2}},
	}); r.Code < 400 {
		t.Errorf("**المكتبُ عدّل بندَ متجرٍ استُلم**: %s", r)
	}
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/edit", ops.Token, map[string]any{
		"reason": "زاد الزبون", "lines": []map[string]any{{"item_id": lineB, "qty": 2}},
	}); r.Code != http.StatusOK {
		t.Errorf("**المكتبُ لا يعدّل بندَ متجرٍ لم يُستلم**: %s", r)
	}
	pickupAllStops(t, s.h, oid, drv)
	if b := settleOf(t, s.h, oid, s.mB.ID); b.amount != 800 {
		t.Errorf("صفُّ الثاني بعد مضاعفة بنده %d — يُنتظر ٨٠٠", b.amount)
	}
	deliverMultiStop(t, s, oid, drv)
	if st := s.h.statusOf(oid); st != "delivered" {
		t.Fatalf("الحال %q", st)
	}
	assertNewViolations(t, s.h, base)
}

// TestMultiStop_StopRemovedAfterArrivalKeepsBaseDriverFee **محطّةٌ انشالت بعد وصول السائق** — `T27`:
// **أجرُ السائق الأساسيُّ وحدَه** (قرارُ المالك ١ب) **ولا طلبَ تعويض** — والمالُ متّسق.
func TestMultiStop_StopRemovedAfterArrivalKeepsBaseDriverFee(t *testing.T) {
	s := newMSShop(t, "merchants")
	s.far(t)
	base := financialBaseline(t, s.h)
	oid, drv := acceptedMS(t, s)
	before := moneyOf(t, s.h, oid)
	if before.SourcesFee <= 0 {
		t.Fatalf("المتجران ليسا متباعدين: %+v", before)
	}
	mustStep(t, s.h, oid, drv, "at_pickup", s.mA.ID)
	mustStep(t, s.h, oid, drv, "picked_up", s.mA.ID)
	mustStep(t, s.h, oid, drv, "at_pickup", s.mB.ID)
	if r := s.partPost(s.admin.Token, oid, s.mB.ID, "remove",
		map[string]any{"reason": "المحل رفض يسلّم", "fault": "merchant"}); r.Code != http.StatusOK {
		t.Fatalf("الشيل: %s", r)
	}
	after := moneyOf(t, s.h, oid)
	want := orders.DriverFeeWithShare(before.DeliveryFee-before.SourcesFee, before.SharePct, before.ShareFixed)
	if after.SourcesFee != 0 || after.DriverFee != want {
		t.Errorf("بعد الشيل: sources_fee=%d driver_fee=%d — يُنتظر 0 و%d (الأساسيّ وحدَه)",
			after.SourcesFee, after.DriverFee, want)
	}
	var comps int
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM driver_compensation_requests WHERE order_id = $1::uuid`,
		oid).Scan(&comps)
	if comps != 0 {
		t.Errorf("**طلبُ تعويضٍ عن محطّةٍ انشالت**: %d", comps)
	}
	if st := s.h.statusOf(oid); st != "picked_up" {
		t.Fatalf("الحال %q — يُنتظر picked_up", st)
	}
	deliverMultiStop(t, s, oid, drv)
	var paid int64
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND user_id = $2::uuid AND kind = 'driver_earning' AND amount > 0`, oid, drv.ID).Scan(&paid)
	if paid != want {
		t.Errorf("أجرُ السائق المقيَّد %d — يُنتظر %d", paid, want)
	}
	assertNewViolations(t, s.h, base)
}

// emergencyOf **بلاغُ الطارئ المفتوح للطلب** — معرّفُه ومرحلتُه.
func emergencyOf(t *testing.T, h *Harness, oid string) (id, stage string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text, stage FROM driver_emergencies
		WHERE order_id = $1::uuid AND status = 'open' ORDER BY created_at LIMIT 1`, oid).Scan(&id, &stage); err != nil {
		t.Fatalf("لا طارئَ مفتوح: %v", err)
	}
	return id, stage
}

// TestMultiStop_EmergencyBetweenStops **حادثٌ بين المحطّتين** — `T29` (D3 · C1، قرارُ المالك ٥أ):
// **طارئٌ «بعد الاستلام» بلا تحرير** · **«كمّل بسائق تاني»: يستلم بضاعةَ الأوّل منه ثمّ الثانية،
// وسلفةُ الأوّل تُردّ عند التسليم** · **و«شيل المحطّات الباقية» يمضي بما استُلم.**
func TestMultiStop_EmergencyBetweenStops(t *testing.T) {
	s := newMSShop(t, "merchants")
	base := financialBaseline(t, s.h)
	setMethod(t, s.h, s.mA, "cash")
	setMethod(t, s.h, s.mB, "cash")

	// ── ١ · سائقٌ آخرُ يستلم من الأوّل ─────────────────────────────────
	oid, x := acceptedMS(t, s)
	mustStep(t, s.h, oid, x, "at_pickup", s.mA.ID)
	mustStep(t, s.h, oid, x, "picked_up", s.mA.ID)
	r := s.h.POST("/api/v1/driver/orders/"+oid+"/emergency", x.Token,
		map[string]any{"kind": "accident", "note": "حادث", "lat": raqqaLat + 0.01, "lng": raqqaLng})
	if r.Code >= 400 || r.JSON()["released"] != false {
		t.Fatalf("الطارئ: %s — يُنتظر بلا تحرير", r)
	}
	eid, stage := emergencyOf(t, s.h, oid)
	if stage != "after_pickup" {
		t.Errorf("مرحلةُ الطارئ %q — يُنتظر after_pickup", stage)
	}
	var hasOverride bool
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT pickup_override IS NOT NULL FROM orders WHERE id = $1::uuid`, oid).
		Scan(&hasOverride)
	if !hasOverride || s.h.statusOf(oid) != "at_pickup" {
		t.Fatalf("بعد الطارئ: نقطةُ استلامٍ بديلة=%v والحال %q", hasOverride, s.h.statusOf(oid))
	}
	detail := s.h.GET("/api/v1/admin/emergencies/"+eid, s.admin.Token)
	if ord, _ := detail.JSON()["order"].(map[string]any); ord == nil || ord["goods_in_hand"] != true {
		t.Errorf("**غرفةُ الطوارئ لا تقول «بيده بضاعة»**: %s", detail)
	}
	if r := s.h.POST("/api/v1/admin/emergencies/"+eid+"/outcome", s.admin.Token,
		map[string]any{"outcome": "redispatch", "note": "سائق تاني"}); r.Code != http.StatusOK {
		t.Fatalf("كمّل بسائق تاني: %s", r)
	}
	if st := s.h.statusOf(oid); st != "dispatching" {
		t.Fatalf("بعد «كمّل بسائق تاني» الحال %q", st)
	}
	if st, by := partStatusBy(t, s.h, oid, s.mA.ID); st != "picked_up" || by != x.ID {
		t.Errorf("قسمُ الأوّل بعد الطارئ: %s بيد %s — يُنتظر picked_up بيد الأوّل", st, by)
	}
	y := s.h.driverOf(oid)
	o := driverListed(t, s.h, y, oid)
	if a := stopOf(o, s.mA.ID); a == nil || a["handover"] != true || o["next_stop"] != s.mA.ID {
		t.Errorf("**المحطّةُ الأولى ليست «تسليماً» للجديد**: %v next=%v", a, o["next_stop"])
	}
	mustStep(t, s.h, oid, y, "at_pickup", s.mA.ID)
	mustStep(t, s.h, oid, y, "picked_up", s.mA.ID)
	if st, by := partStatusBy(t, s.h, oid, s.mA.ID); st != "picked_up" || by != y.ID {
		t.Errorf("بعد التسليم: %s بيد %s", st, by)
	}
	if a := settleOf(t, s.h, oid, s.mA.ID); a.n != 1 {
		t.Errorf("**محطّةُ التسليم سوّت الأوّل ثانيةً**: %+v", a)
	}
	deliverMultiStop(t, s, oid, y)
	if st := s.h.statusOf(oid); st != "delivered" {
		t.Fatalf("الحال %q", st)
	}
	var advance int64
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		WHERE source = 'advance_return' AND ref = $1`, oid).Scan(&advance)
	if advance != 900 {
		t.Errorf("سلفةُ الأوّل المردودة %d — يُنتظر ٩٠٠ (ما دفعه للمتجر الأوّل)", advance)
	}

	// ── ٢ · شيلُ المحطّات الباقية ──────────────────────────────────────
	oid2, x2 := acceptedMS(t, s)
	mustStep(t, s.h, oid2, x2, "at_pickup", s.mA.ID)
	mustStep(t, s.h, oid2, x2, "picked_up", s.mA.ID)
	if r := s.h.POST("/api/v1/driver/orders/"+oid2+"/emergency", x2.Token,
		map[string]any{"kind": "accident", "note": "حادث"}); r.Code >= 400 {
		t.Fatal(r)
	}
	eid2, _ := emergencyOf(t, s.h, oid2)
	if r := s.h.POST("/api/v1/admin/emergencies/"+eid2+"/outcome", s.admin.Token,
		map[string]any{"outcome": "drop_remaining", "note": "ما في وقت"}); r.Code != http.StatusOK {
		t.Fatalf("شيلُ الباقي: %s", r)
	}
	if st := s.h.statusOf(oid2); st != "picked_up" {
		t.Fatalf("بعد شيل الباقي الحال %q — يُنتظر picked_up", st)
	}
	if st, _ := partStatusBy(t, s.h, oid2, s.mB.ID); st != "removed" {
		t.Errorf("قسمُ الثاني %q", st)
	}
	// **ثمّ مخارجُ ما بعد الاستلام كما هي** — سائقٌ آخر من موضع الأوّل.
	if r := s.h.POST("/api/v1/admin/emergencies/"+eid2+"/outcome", s.admin.Token,
		map[string]any{"outcome": "redispatch", "note": "سائق تاني"}); r.Code != http.StatusOK {
		t.Fatalf("بعد الشيل «كمّل بسائق تاني»: %s", r)
	}
	y2 := s.h.driverOf(oid2)
	for _, to := range []string{"at_pickup", "picked_up"} {
		if r := msStep(s.h, oid2, y2, to, s.mA.ID); r.Code != http.StatusOK {
			t.Fatalf("الجديدُ %s: %s", to, r)
		}
	}
	deliverMultiStop(t, s, oid2, y2)
	if st := s.h.statusOf(oid2); st != "delivered" {
		t.Fatalf("الحال %q", st)
	}
	assertNewViolations(t, s.h, base)
}

// TestMultiStop_ReleaseWithGoodsInHand **«لدي مشكلة» وبيده بضاعةُ محطّة** — `T30` (D3).
func TestMultiStop_ReleaseWithGoodsInHand(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
	waitStatusIs(t, s.h, oid, "dispatching", 3*time.Second)
	drv := s.f.Driver(OnShift())
	t.Cleanup(func() {
		_, _ = s.h.Pool.Exec(ctxBG(), `UPDATE users SET on_shift = false WHERE id = $1::uuid`, drv.ID)
	})
	if r := s.h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); r.Code != http.StatusOK {
		t.Fatalf("القبول: %s", r)
	}
	mustStep(t, s.h, oid, drv, "at_pickup", s.mA.ID)
	mustStep(t, s.h, oid, drv, "picked_up", s.mA.ID)
	onShift := func() bool {
		var on bool
		_ = s.h.Pool.QueryRow(ctxBG(), `SELECT on_shift FROM users WHERE id = $1::uuid`, drv.ID).Scan(&on)
		return on
	}
	if r := s.h.POST("/api/v1/driver/orders/"+oid+"/release", drv.Token,
		map[string]any{"reason": "bike_broken", "note": "بنشر"}); r.Code != http.StatusConflict || r.Err() != "stops_in_hand" {
		t.Errorf("تركٌ بعطل: %s — يُنتظر 409 stops_in_hand", r)
	}
	if !onShift() {
		t.Errorf("**أُغلق دوامُه والتركُ مرفوض**")
	}
	r := s.h.POST("/api/v1/driver/orders/"+oid+"/release", drv.Token,
		map[string]any{"reason": "accident", "note": "حادث"})
	if r.Code != http.StatusOK || r.JSON()["released"] != false {
		t.Fatalf("تركٌ بحادث: %s — يُنتظر 200 بلا تحرير", r)
	}
	if st := s.h.statusOf(oid); st != "at_pickup" {
		t.Errorf("الحال %q — يُنتظر at_pickup (المكتبُ يقرّر)", st)
	}
	var holder string
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(driver_id::text, '') FROM orders WHERE id = $1::uuid`, oid).
		Scan(&holder)
	if holder != drv.ID {
		t.Errorf("**الطلبُ تُرك والبضاعةُ معه**: الحامل %q", holder)
	}
	if _, stage := emergencyOf(t, s.h, oid); stage != "after_pickup" {
		t.Errorf("مرحلةُ الطارئ %q", stage)
	}
	if onShift() {
		t.Errorf("دوامُه لم يُغلق بعد الحادث")
	}
}

// TestMultiStop_SecondStopArrival **الوصولُ إلى المحطّة الثانية** — `T33` (D7): **وقتُها، وأمرُ
// الإدارة يُمحى، وانتظارُ الباب منها لا من الأولى.**
func TestMultiStop_SecondStopArrival(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid, drv := acceptedMS(t, s)
	mustStep(t, s.h, oid, drv, "at_pickup", s.mA.ID)
	mustStep(t, s.h, oid, drv, "picked_up", s.mA.ID)
	// **وأحداثُ المحطّات لا تحجب «خطوةً ثبتت»** — الإعادةُ تقرأ أحداثَ الطلب وحدَها.
	if !s.h.Orders.RepeatsOwnTransition(ctxBG(), oid, drv.ID, "at_pickup") {
		t.Errorf("أوّلُ وصولٍ للطلب لا يُقرأ خطوتَه بعد حدث المحطّة")
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `
		UPDATE orders SET door_instruction = 'keep', door_instruction_note = 'أكمل', door_instruction_at = now()
		WHERE id = $1::uuid`, oid); err != nil {
		t.Fatal(err)
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `
		UPDATE order_events SET created_at = now() - interval '20 minutes'
		WHERE order_id = $1::uuid AND to_status = 'at_pickup' AND merchant_id IS NULL`, oid); err != nil {
		t.Fatal(err)
	}
	mustStep(t, s.h, oid, drv, "at_pickup", s.mB.ID)
	var arrived bool
	var instr string
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT p.arrived_at IS NOT NULL, o.door_instruction
		FROM order_parts p JOIN orders o ON o.id = p.order_id
		WHERE p.order_id = $1::uuid AND p.merchant_id = $2::uuid`, oid, s.mB.ID).Scan(&arrived, &instr)
	if !arrived || instr != "" {
		t.Errorf("الوصولُ إلى الثانية: arrived=%v instruction=%q", arrived, instr)
	}
	v := s.h.Orders.DoorViewOf(ctxBG(), oid)
	if v.WaitedMin < 0 || v.WaitedMin > 2 {
		t.Errorf("**انتظارُ الباب من المحطّة الأولى**: %d د — يُنتظر من الثانية (صفر)", v.WaitedMin)
	}
}

// TestMultiStop_ReclaimSkipsCollectedPart **كانسُ الإسناد الصامت لا ينزع طلباً استُلمت منه محطّة**
// — `T38` (D13).
func TestMultiStop_ReclaimSkipsCollectedPart(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid, drv := acceptedMS(t, s)
	mustStep(t, s.h, oid, drv, "at_pickup", s.mA.ID)
	mustStep(t, s.h, oid, drv, "picked_up", s.mA.ID)
	s.h.Setting("drivers.assignment_mode", `"rotation"`)
	s.h.Setting("drivers.direct_assign", "true")
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE orders SET status = 'assigned',
		offer_expires_at = now() - interval '1 minute' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatal(err)
	}
	s.h.Orders.SweepExpiredOffers(ctxBG())
	var holder string
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(driver_id::text, '') FROM orders WHERE id = $1::uuid`, oid).
		Scan(&holder)
	if st := s.h.statusOf(oid); st != "assigned" || holder != drv.ID {
		t.Errorf("**نُزع الطلبُ والبضاعةُ مع سائقه**: %q بيد %q", st, holder)
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE orders SET offer_expires_at = NULL, status = 'at_pickup'
		WHERE id = $1::uuid`, oid); err != nil {
		t.Fatal(err)
	}
}

// TestMultiStoreReturnRefused **بضاعةُ متجرين لا تعود لمتجر** — `T0f` (قرارُ المالك ٤أ، PD-6/PD-7):
// الإرجاعُ إلى المتجر مرفوض، وزرُّ السائق «أعدتُ البضاعة» ٤٠٩، **وحسمُها «إلى المتجر» مرفوض.**
func TestMultiStoreReturnRefused(t *testing.T) {
	s := newMSShop(t, "merchants")
	for _, m := range []*Merchant{s.mA, s.mB} {
		if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE merchants SET accepts_returns = true WHERE id = $1::uuid`,
			m.ID); err != nil {
			t.Fatal(err)
		}
	}
	oid, drv := acceptedMS(t, s)
	pickupAllStops(t, s.h, oid, drv)
	for _, to := range []string{"on_the_way", "at_dropoff"} {
		mustStep(t, s.h, oid, drv, to, "")
	}
	_, err := s.h.Orders.ResolveDoor(ctxBG(), s.admin.ID, []string{"admin"}, oid, orders.DoorResolution{
		Action: orders.DoorReturnToOffice, Fault: orders.FaultCustomer, Note: "الزبون رفض", ReturnTo: orders.ReturnToStore,
	}, nil)
	if !errors.Is(err, orders.ErrReturnStoreRefused) {
		t.Errorf("**الإرجاعُ إلى المتجر قُبل لبضاعة متجرين**: %v", err)
	}
	if _, err := s.h.Orders.ResolveDoor(ctxBG(), s.admin.ID, []string{"admin"}, oid, orders.DoorResolution{
		Action: orders.DoorReturnToOffice, Fault: orders.FaultCustomer, Note: "الزبون رفض",
	}, nil); err != nil {
		t.Fatalf("الإرجاعُ إلى المكتب: %v", err)
	}
	if r := s.h.POST("/api/v1/driver/orders/"+oid+"/return", drv.Token, nil); r.Code != http.StatusConflict {
		t.Errorf("**«أعدتُ البضاعة» لمتجرٍ من متجرين**: %s — يُنتظر 409", r)
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE orders SET return_to = NULL WHERE id = $1::uuid`, oid); err != nil {
		t.Fatal(err)
	}
	if err := s.h.Orders.SettleGoods(ctxBG(), oid, orders.GoodsToMerchant, s.admin.ID, 0); !errors.Is(err, orders.ErrReturnStoreRefused) {
		t.Errorf("**حسمُ البضاعة «إلى المتجر»**: %v — يُنتظر رفضاً", err)
	}
}

// TestMultiStop_CompensationStoreChoice **تعويضُ السائق بذنب متجرٍ في طلبٍ بعدّة محطّات** — المكتبُ
// يختار المتجر (قرارُ المالك ٦أ): **بلا اختيارٍ ٤٠٩**، **وبه تُكتب المطالبةُ على المختار.**
func TestMultiStop_CompensationStoreChoice(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid, drv := acceptedMS(t, s)
	var id string
	if err := s.h.Pool.QueryRow(ctxBG(), `
		INSERT INTO driver_compensation_requests (kind, order_id, driver_id, fault, fail_reason, suggested_amount)
		VALUES ('driver', $1::uuid, $2::uuid, 'merchant', 'merchant_closed', 100) RETURNING id::text`,
		oid, drv.ID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	fin := s.h.NewUser("finance")
	if r := s.h.POST("/api/v1/admin/compensations/"+id+"/approve", fin.Token,
		map[string]any{"amount": 100, "note": "موافقة"}); r.Code != http.StatusConflict || r.Err() != "compensation_store_required" {
		t.Fatalf("موافقةٌ بلا متجر: %s — يُنتظر 409 compensation_store_required", r)
	}
	var paid int
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallet_transactions WHERE ref = $1 AND kind = 'compensation'`,
		oid).Scan(&paid)
	if paid != 0 {
		t.Fatalf("**دُفع التعويضُ والموافقةُ مرفوضة**")
	}
	if r := s.h.POST("/api/v1/admin/compensations/"+id+"/approve", fin.Token,
		map[string]any{"amount": 100, "note": "موافقة", "merchant_id": s.mB.ID}); r.Code != http.StatusOK {
		t.Fatalf("موافقةٌ على الثاني: %s", r)
	}
	var disputeStore, reqStore string
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT merchant_id::text FROM disputes WHERE order_id = $1::uuid
		AND party_role = 'merchant'`, oid).Scan(&disputeStore)
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(merchant_id::text, '') FROM driver_compensation_requests
		WHERE id = $1::uuid`, id).Scan(&reqStore)
	if disputeStore != s.mB.ID || reqStore != s.mB.ID {
		t.Errorf("المطالبةُ على %q وطلبُ التعويض على %q — يُنتظر الثاني", disputeStore, reqStore)
	}
	var warned string
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(merchant_id::text, '') FROM warnings WHERE order_id = $1::uuid
		AND merchant_id IS NOT NULL LIMIT 1`, oid).Scan(&warned)
	if warned != s.mB.ID {
		t.Errorf("الإنذارُ على %q — يُنتظر الثاني", warned)
	}
}
