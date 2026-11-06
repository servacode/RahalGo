package qa

// ══════════════════════════════════════════════════════════════════════
// **طلباتُ الواتساب** — قرارُ المالك ٢٠٢٦-١١-٠٤ ثمّ ٢٠٢٦-١١-٠٥
// ══════════════════════════════════════════════════════════════════════
//
// **الموظّفُ يكتب طلباً وصله على الواتساب فينزل للسائقين طلباً نظاميّاً.** والطلبُ «لدي
// توصيلة» بعلامة `origin='whatsapp'`: **بلا متجرٍ مربوطٍ ولا زبون**، والمستلِمُ يدفع الأجرةَ
// نقداً، **والبضاعةُ خارجَ الدفتر.**
//
// **والمالُ يُقاس من الصفوف** — وثوابتُ الدفتر تُصوَّر قبل السيناريو ويُطلب ألّا يستجدّ فيها خرق.

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// waFx **منطقةٌ ومتجرٌ من متاجرنا ونقطةُ مستلِم** — وأجرةُ «طلب واتساب» ٥٠٠ بحصّة ١٠٪ (مفاتيحُه هو
// منذ ملحق المالك ٢٠٢٦-١١-٠٥، كقيم «لدي توصيلة» في `newMDFx`).
type waFx struct {
	md mdFx
	// storeLat/storeLng **متجرٌ غيرُ تابع** داخلَ المنطقة.
	storeLat, storeLng float64
}

func newWAFx(t *testing.T, h *Harness, name string) waFx {
	t.Helper()
	m := newMDFx(t, h, name)
	h.Setting("delivery.whatsapp_base_fee", "500")
	h.Setting("delivery.whatsapp_by_distance", "false")
	h.Setting("delivery.share_whatsapp", "10")
	return waFx{md: m, storeLat: m.z.Lat - 0.003, storeLng: m.z.Lng}
}

// typed **طلبٌ من متجرٍ غير تابع** — بضاعتُه ٢٤٠٠.
func (w waFx) typed() orders.WhatsAppOrderInput {
	slat, slng := w.storeLat, w.storeLng
	lat, lng := w.md.dropLat, w.md.dropLng
	goods := int64(2400)
	return orders.WhatsAppOrderInput{
		StoreName: "فلافل الأمير", StorePhone: "0933444555", StoreAddress: "شارع تل أبيض",
		StoreLat: &slat, StoreLng: &slng,
		Details: "2 شاورما دبل بدون ثوم", GoodsAmount: &goods,
		RecipientName: "أبو أحمد", RecipientPhone: "+963935111222",
		AddressText: "حي الرميلة جانب الجامع", Lat: &lat, Lng: &lng,
		DriverNote: "اتصل قبل الوصول",
	}
}

// waEngineCreate **يُنشئ طلبَ واتسابٍ من المحرّك** — في معاملةٍ ثمّ ما بعد التثبيت.
func waEngineCreate(t *testing.T, h *Harness, actorID string, in orders.WhatsAppOrderInput) *orders.WhatsAppCreated {
	t.Helper()
	ctx := context.Background()
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	c, err := h.Orders.CreateWhatsAppOrderIn(ctx, tx, actorID, in)
	if err != nil {
		t.Fatalf("إنشاءُ طلب الواتساب: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Orders.AfterWhatsAppOrder(ctx, c.ID); err != nil {
		t.Fatalf("ما بعد الإنشاء: %v", err)
	}
	return c
}

// waLedgerOf **قيودُ الطلب بأنواعها** — من الدفتر لا من الردود.
func waLedgerOf(t *testing.T, h *Harness, orderID string) map[string]int64 {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT kind, sum(amount)::bigint FROM wallet_transactions
		 WHERE ref = $1 GROUP BY kind`, orderID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var k string
		var v int64
		if err := rows.Scan(&k, &v); err != nil {
			t.Fatal(err)
		}
		out[k] = v
	}
	return out
}

func waCashOf(t *testing.T, h *Harness, orderID string) int64 {
	t.Helper()
	var n int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0)::bigint FROM driver_cash_entries
		 WHERE ref = $1 AND kind = 'order_collection'`, orderID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestWhatsAppFeeFromStorePoint **الأجرةُ كالتوصيلات من المتجر إلى المستلِم — بمفاتيح «طلب واتساب»**
// (قرارُ المالك ٢٠٢٦-١١-٠٥: «ليها خيارات مثل الاجرة وحصة المنصة»)، **وعرضُ السعر والإنشاءُ بدالّةٍ
// واحدة.** ومفاتيحُ «لدي توصيلة» وأجرتُها الثابتةُ لا تمسّه.
//
// **وعلى شيفرة المراحل ١–٥** كان يقرأ مفاتيحَ «لدي توصيلة» وحصّتَها — فيخرج ٦٠٠ لا ١٢٥.
func TestWhatsAppFeeFromStorePoint(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-FEE")
	// **«لدي توصيلة» برقمٍ آخر** — لا يُقرأ هنا.
	h.Setting("delivery.merchant_base_fee", "600")
	h.Setting("delivery.merchant_by_distance", "false")
	h.Setting("delivery.share_merchant", "50")
	// **أساسٌ ١٠٠ ولكلّ كيلومترٍ ٢٥** — والمسافةُ هنا أقلُّ من كيلومتر ⇒ ١٢٥.
	h.Setting("delivery.whatsapp_base_fee", "100")
	h.Setting("delivery.whatsapp_per_km", "25")
	h.Setting("delivery.whatsapp_by_distance", "true")
	h.Setting("delivery.whatsapp_max_fee", "0")
	quote := func(from string) map[string]any {
		t.Helper()
		r := h.GET(waURL+"/quote?"+from+"&lat="+ftoa(w.md.dropLat)+"&lng="+ftoa(w.md.dropLng), w.md.admin.Token)
		if r.Code != http.StatusOK {
			t.Fatalf("عرضُ السعر: %d / %s", r.Code, r.Err())
		}
		return r.JSON()
	}
	typed := "pickup_lat=" + ftoa(w.storeLat) + "&pickup_lng=" + ftoa(w.storeLng)
	q := quote(typed)
	if q["fee"] != float64(125) || q["fee_basis"] != orders.FeeBasisDistance {
		t.Fatalf("**عرضُ السعر %v (%v) والمنتظَر 125 بمفاتيح «طلب واتساب»**", q["fee"], q["fee_basis"])
	}
	if d, _ := q["distance_m"].(float64); d <= 0 || d >= 1000 {
		t.Fatalf("المسافة %v — والمنتظَر دون الكيلومتر", q["distance_m"])
	}
	if q["driver_fee"] != float64(113) || q["platform_share"] != float64(12) {
		t.Fatalf("**الحصّة %v والسائق %v — المنتظَر 12 و113 بحصّة «طلب واتساب»** (الكسرُ للسائق)",
			q["platform_share"], q["driver_fee"])
	}
	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	if c.Fee != 125 || c.ComputedFee != 125 {
		t.Fatalf("**الإنشاءُ كتب %d وعرضُ السعر قال 125**", c.Fee)
	}

	// **وأجرةُ «لدي توصيلة» الثابتةُ لا تسبقه بعد اليوم.**
	h.Setting("delivery.merchant_fee", "300")
	if q2 := quote(typed); q2["fee"] != float64(125) {
		t.Fatalf("**أجرةُ «لدي توصيلة» الثابتةُ سبقت طلبَ الواتساب**: %v", q2["fee"])
	}
	// **وحصّتُه مبلغاً ثابتاً.**
	h.Setting("delivery.share_whatsapp_mode", `"fixed"`)
	h.Setting("delivery.share_whatsapp", "25")
	if q3 := quote(typed); q3["platform_share"] != float64(25) || q3["driver_fee"] != float64(100) {
		t.Fatalf("**الحصّةُ الثابتة**: %v / %v", q3["platform_share"], q3["driver_fee"])
	}
	// **ومتجرٌ من متاجرنا يُسعَّر من موقعه** — بالدالّة نفسِها.
	if q4 := quote("store_id=" + w.md.fx.M.ID); q4["fee"] == nil || q4["fee_basis"] != orders.FeeBasisDistance {
		t.Fatalf("عرضُ السعر لمتجرٍ من متاجرنا: %v", q4)
	}
}

// TestWAORD004_DeliveredMovesOnlyTheFee **التسليمُ لا يحرّك إلّا الأجرة.**
//
//	السائق  +٤٥٠ (الأجرةُ ناقصَ الحصّة)   · صندوقُه +٥٠٠ (الأجرةُ نقداً)
//	الخزينة +٥٠                            · البضاعةُ ٢٤٠٠ لا أثرَ لها في أيّ دفتر
func TestWAORD004_DeliveredMovesOnlyTheFee(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-004")
	base := mdaBaseline(t, h)

	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	var kind, origin, status, payer, mode string
	var merchant, customer, createdBy *string
	var subtotal, fee, driverFee, total, cash, waived int64
	var collect *int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT kind, origin, status, fee_payer, delivery_mode,
		       merchant_id::text, customer_id::text, created_by::text,
		       subtotal, delivery_fee, driver_fee, total, cash_due, promo_delivery_waived, collect_amount
		  FROM orders WHERE id = $1`, c.ID).
		Scan(&kind, &origin, &status, &payer, &mode, &merchant, &customer, &createdBy,
			&subtotal, &fee, &driverFee, &total, &cash, &waived, &collect); err != nil {
		t.Fatal(err)
	}
	if kind != "merchant_delivery" || origin != "whatsapp" || status != "dispatching" ||
		payer != "recipient" || mode != "send" {
		t.Fatalf("**شكلُ الطلب**: %s/%s/%s/%s/%s", kind, origin, status, payer, mode)
	}
	if merchant != nil || customer != nil {
		t.Fatalf("**طلبُ الواتساب رُبط بمتجرٍ أو زبون**: %v / %v", merchant, customer)
	}
	if createdBy == nil || *createdBy != w.md.admin.ID {
		t.Fatalf("**من كتبه لم يُكتب**: %v", createdBy)
	}
	if subtotal != 0 || fee != 500 || total != 500 || cash != 500 || driverFee != 450 || waived != 0 {
		t.Fatalf("**المال**: مجموع %d · أجرة %d · إجماليّ %d · نقد %d · سائق %d · إعفاء %d",
			subtotal, fee, total, cash, driverFee, waived)
	}
	if collect == nil || *collect != 2400 {
		t.Fatalf("**ثمنُ البضاعة للعرض** %v والمنتظَر 2400", collect)
	}
	var birth string
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT from_status || '>' || to_status || ':' || note FROM order_events
		 WHERE order_id = $1 ORDER BY created_at, id LIMIT 1`, c.ID).Scan(&birth)
	if birth != ">dispatching:طلب واتساب" {
		t.Fatalf("**لحظةُ الميلاد** %q", birth)
	}

	drv := h.driverOf(c.ID)
	deliverOrder(t, h, c.ID, drv)
	if st := h.statusOf(c.ID); st != "delivered" {
		t.Fatalf("الحال %s", st)
	}
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("**أجرُ السائق %d والمنتظَر 450**", got)
	}
	if got := waCashOf(t, h, c.ID); got != 500 {
		t.Fatalf("**نقدُ الصندوق %d والمنتظَر 500 — البضاعةُ لا تدخله**", got)
	}
	led := waLedgerOf(t, h, c.ID)
	if led["platform_profit"] != 50 || led["driver_earning"] != 450 || len(led) != 2 {
		t.Fatalf("**الدفتر**: %v — المنتظَر أجرُ السائق 450 وحصّةُ المنصّة 50 لا غير", led)
	}
	mdaNoNewViolations(t, h, base)
}

// TestWAORD005_OpsCancelBeforePickup **المكتبُ يُلغي قبل الاستلام — ولا مال.**
//
// **وعلى الشيفرة القديمة يسقط بـ٥٠٠**: فرعُ الإلغاء يضمّ المتاجرَ ضمّاً صلباً فيردّ «لا صفوف».
func TestWAORD005_OpsCancelBeforePickup(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-005")
	base := mdaBaseline(t, h)

	// ── من الطابور ─────────────────────────────────────────────────────
	c1 := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	r := h.POST("/api/v1/admin/orders/"+c1.ID+"/transition", w.md.admin.Token,
		map[string]any{"to": "cancelled", "note": "المستلم ألغى بالواتساب"})
	if r.Code != http.StatusOK {
		t.Fatalf("**الإلغاءُ من الطابور رُدّ**: %d / %s", r.Code, r.Err())
	}
	// ── وبعد الإسناد — والسائقُ يُخبَر ──────────────────────────────────
	c2 := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	drv := h.driverOf(c2.ID)
	r = h.POST("/api/v1/admin/orders/"+c2.ID+"/transition", w.md.admin.Token,
		map[string]any{"to": "cancelled", "note": "المتجر مسكّر"})
	if r.Code != http.StatusOK {
		t.Fatalf("**الإلغاءُ بعد الإسناد رُدّ**: %d / %s", r.Code, r.Err())
	}
	for _, id := range []string{c1.ID, c2.ID} {
		if st := h.statusOf(id); st != "cancelled" {
			t.Fatalf("الحال %s", st)
		}
		if led := waLedgerOf(t, h, id); len(led) != 0 {
			t.Fatalf("**مالٌ تحرّك على إلغاءٍ قبل الاستلام**: %v", led)
		}
		if got := waCashOf(t, h, id); got != 0 {
			t.Fatalf("**نقدٌ في الصندوق**: %d", got)
		}
	}
	var told int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2`, drv.ID, c2.ID).Scan(&told)
	if told == 0 {
		t.Fatalf("**السائقُ لم يُخبَر بأنّ الطلب أُخذ منه**")
	}
	mdaNoNewViolations(t, h, base)
}

// waToDoor **السائقُ يستلم ويصل باب المستلِم** — بانتقالات المجال الحقيقيّة.
func waToDoor(t *testing.T, h *Harness, id string, drv *User) {
	t.Helper()
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token,
			map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s رُدّ: %s", to, r)
		}
	}
}

// TestWAORD006_DoorFailureChargesNoStore **المستلِمُ لا يردّ بعد الاستلام ⇒ لا متجرَ يُحمَّل.**
//
// **والسائقُ يُطلب له تعويضٌ يوافق عليه إنسان** (سؤالُ المالك ٢، الافتراض). **وعلى الشيفرة
// القديمة يسقط**: «المتجرُ يدفع كأنّها سُلّمت» يخصم من محفظة متجرٍ لا وجودَ له.
func TestWAORD006_DoorFailureChargesNoStore(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-006")
	base := mdaBaseline(t, h)

	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	drv := h.driverOf(c.ID)
	waToDoor(t, h, c.ID, drv)
	r := h.POST("/api/v1/admin/orders/"+c.ID+"/door-resolution", w.md.admin.Token,
		map[string]any{"action": "return_to_office", "fault": "customer",
			"reason": "customer_absent", "note": "المستلم ما رد"})
	if r.Code != http.StatusOK {
		t.Fatalf("**العودةُ إلى المكتب رُدّت**: %d / %s", r.Code, r.Err())
	}
	if st := h.statusOf(c.ID); st != "failed" {
		t.Fatalf("الحال %s", st)
	}
	if led := waLedgerOf(t, h, c.ID); len(led) != 0 {
		t.Fatalf("**قيدٌ على طلبٍ فشل قبل أيّ موافقة**: %v", led)
	}
	var debts, pending int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM financial_obligations WHERE order_id = $1`, c.ID).Scan(&debts)
	if debts != 0 {
		t.Fatalf("**دينٌ كُتب على طلب واتساب**: %d", debts)
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM driver_compensation_requests
		WHERE order_id = $1 AND driver_id = $2 AND status = 'pending'`, c.ID, drv.ID).Scan(&pending)
	if pending != 1 {
		t.Fatalf("**لا طلبَ تعويضٍ معلَّقاً لسائقٍ قاد بلا ذنب**: %d", pending)
	}
	mdaNoNewViolations(t, h, base)
}

// TestWAORD007_StoreClosedCompensationIsApprovable **المتجرُ مسكّرٌ عند الاستلام ⇒ تعويضٌ
// يوافَق عليه — ولا نزاعَ على متجرٍ غيرِ مربوط.**
//
// **وعلى الشيفرة القديمة تسقط الموافقةُ كلُّها**: مطالبةٌ بلا طرفٍ تخرق `disputes_party_present`.
func TestWAORD007_StoreClosedCompensationIsApprovable(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-007")
	base := mdaBaseline(t, h)

	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	drv := h.driverOf(c.ID)
	if r := h.POST("/api/v1/driver/orders/"+c.ID+"/transition", drv.Token,
		map[string]any{"to": "at_pickup"}); r.Code >= 400 {
		t.Fatalf("عند المتجر: %s", r)
	}
	// **السائقُ يُبلّغ والمكتبُ يقرّر** (مساءَ ٢٠٢٦-١٠-٠٢) — «المتجر مسكّر» بلاغٌ لا إنهاء.
	if r := h.POST("/api/v1/driver/orders/"+c.ID+"/transition", drv.Token,
		map[string]any{"to": "failed", "reason": "merchant_closed"}); r.Code >= 400 {
		t.Fatalf("**بلاغُ «المتجر مسكّر» رُدّ**: %s", r)
	}
	if r := h.POST("/api/v1/admin/orders/"+c.ID+"/door-resolution", w.md.admin.Token,
		map[string]any{"action": "return_to_office", "fault": "merchant",
			"reason": "merchant_closed", "note": "المتجر مسكّر"}); r.Code >= 400 {
		t.Fatalf("**قرارُ المكتب عند المتجر رُدّ**: %d / %s", r.Code, r.Err())
	}
	// **والطلبُ حيٌّ عند المتجر** — المكتبُ يقرّر، والسائقُ ينتظر تعويضه.
	if st := h.statusOf(c.ID); st != "at_pickup" {
		t.Fatalf("الحال %s والمنتظَر at_pickup", st)
	}
	var compID, fault string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text, fault FROM driver_compensation_requests
		WHERE order_id = $1 AND status = 'pending'`, c.ID).Scan(&compID, &fault); err != nil {
		t.Fatalf("**لا طلبَ تعويض**: %v", err)
	}
	fin := h.NewUser("admin")
	if r := h.POST("/api/v1/admin/compensations/"+compID+"/approve", fin.Token,
		map[string]any{"amount": 200, "note": "مشوار ضاع"}); r.Code >= 400 {
		t.Fatalf("**الموافقةُ على التعويض سقطت**: %d / %s", r.Code, r.Err())
	}
	led := waLedgerOf(t, h, c.ID)
	if led["compensation"] != 200 || led["platform_expense"] != -200 {
		t.Fatalf("**قيودُ التعويض**: %v", led)
	}
	var disputes int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM disputes WHERE order_id = $1`, c.ID).Scan(&disputes)
	if disputes != 0 {
		t.Fatalf("**نزاعٌ على متجرٍ غيرِ مربوط**: %d", disputes)
	}
	mdaNoNewViolations(t, h, base)
}

// TestWAORD_FI06e_OnlyTheFeeMoves **الحارسُ يرى قيداً غريباً على طلب واتساب** — ولا يرى شيئاً
// على دورةٍ سليمة.
func TestWAORD_FI06e_OnlyTheFeeMoves(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-FI")
	base := financialBaseline(t, h, "FI-06.e")

	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	drv := h.driverOf(c.ID)
	deliverOrder(t, h, c.ID, drv)
	clean, err := fininv.Run(ctxBG(), h.Pool, "FI-06.e")
	if err != nil {
		t.Fatal(err)
	}
	if fresh := base.New(clean); len(fresh) > 0 {
		t.Fatalf("**دورةٌ سليمةٌ قُرئت خرقاً**: %v", fresh)
	}
	// **ومستحقُّ متجرٍ محقونٌ عليه يُمسَك.**
	owner := w.md.fx.M.Owner.ID
	var id int64
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO wallet_transactions (user_id, amount, kind, ref, note)
		VALUES ($1::uuid, 2400, 'merchant_earning', $2, 'FI-06.e self-test') RETURNING id`,
		owner, c.ID).Scan(&id); err != nil {
		t.Fatalf("الإفسادُ المقصود: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `DELETE FROM wallet_transactions WHERE id = $1`, id)
	})
	dirty, err := fininv.Run(ctxBG(), h.Pool, "FI-06.e")
	if err != nil {
		t.Fatal(err)
	}
	if len(base.New(dirty)) == 0 {
		t.Fatal("**مستحقُّ متجرٍ على طلب واتسابٍ لم يُمسَك**")
	}
}

// waDriverOrder **طلبٌ من قائمة السائق كما يقرؤها تطبيقُه** — حقولُ بطاقته.
type waDriverOrder struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Origin         string   `json:"origin"`
	MerchantName   string   `json:"merchant_name"`
	MerchantPhone  *string  `json:"merchant_phone"`
	PickupAddress  string   `json:"pickup_address"`
	NavLat         *float64 `json:"nav_lat"`
	NavLng         *float64 `json:"nav_lng"`
	RecipientPhone string   `json:"recipient_phone"`
	CustomerName   string   `json:"customer_name"`
	ParcelNote     string   `json:"parcel_note"`
	DriverNote     string   `json:"driver_note"`
	CollectAmount  *int64   `json:"collect_amount"`
	CashDue        int64    `json:"cash_due"`
	DeliveryFee    int64    `json:"delivery_fee"`
	FeePayer       string   `json:"fee_payer"`
	DeliveryMode   string   `json:"delivery_mode"`
}

func waDriverList(t *testing.T, r Res) []waDriverOrder {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("قائمةُ السائق: %d / %s", r.Code, r.Err())
	}
	var env struct {
		Data []waDriverOrder `json:"data"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("فكُّ قائمة السائق: %v", err)
	}
	return env.Data
}

// TestWAORD002_OldDriverAppReadsTypedStore **التطبيقُ القديمُ يقرأ المتجرَ المكتوب حيث اعتاد.**
//
// **بلا تحديثٍ للتطبيق**: اسمُ المتجر ورقمُه وعنوانُه في `merchant_name`/`merchant_phone`/
// `pickup_address`، ونقطتُه في `nav_*`. **وعلى شيفرة المرحلة الأولى** يصل الاسمُ فارغاً
// والرقمُ بلا قيمة — فلا يعرف السائقُ من أين يشتري ولا بمن يتّصل.
func TestWAORD002_OldDriverAppReadsTypedStore(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-002")
	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	drv := h.driverOf(c.ID)

	var got *waDriverOrder
	for _, o := range waDriverList(t, h.GET("/api/v1/driver/orders", drv.Token)) {
		if o.ID == c.ID {
			o := o
			got = &o
		}
	}
	if got == nil {
		t.Fatalf("**طلبُ الواتساب غائبٌ عن قائمة سائقه**")
	}
	if got.MerchantName != "فلافل الأمير" || got.MerchantPhone == nil || *got.MerchantPhone != "0933444555" ||
		got.PickupAddress != "شارع تل أبيض" {
		t.Fatalf("**المتجرُ المكتوبُ لا يصل السائق**: اسم %q · رقم %v · عنوان %q",
			got.MerchantName, got.MerchantPhone, got.PickupAddress)
	}
	if got.NavLat == nil || got.NavLng == nil ||
		math.Abs(*got.NavLat-w.storeLat) > 1e-6 || math.Abs(*got.NavLng-w.storeLng) > 1e-6 {
		t.Fatalf("**الملاحةُ لا تقود إلى المتجر**: %v,%v والمنتظَر %v,%v",
			got.NavLat, got.NavLng, w.storeLat, w.storeLng)
	}
	if got.Origin != "whatsapp" || got.Kind != "merchant_delivery" || got.DeliveryMode != "send" ||
		got.FeePayer != "recipient" {
		t.Fatalf("**الأصلُ والنوع**: %q / %q / %q / %q", got.Origin, got.Kind, got.DeliveryMode, got.FeePayer)
	}
	// **ولا رقمَ للمستلِم عند السائق أبداً** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٧) — ولو بعد الإسناد.
	if got.RecipientPhone != "" || got.CustomerName != "أبو أحمد" ||
		got.ParcelNote != "2 شاورما دبل بدون ثوم" || got.DriverNote != "اتصل قبل الوصول" {
		t.Fatalf("**المستلِمُ وما يشتريه**: %+v", got)
	}
	if got.CollectAmount == nil || *got.CollectAmount != 2400 || got.CashDue != 500 || got.DeliveryFee != 500 {
		t.Fatalf("**ما يدفعه وما يقبضه**: بضاعة %v · نقد %d · أجرة %d", got.CollectAmount, got.CashDue, got.DeliveryFee)
	}
}

// TestWAORD013_DriverMonitorCountsIt **مراقبةُ السائقين تعدّه نافذةَ طلب** — باسم متجره.
//
// **وهي الغايةُ كلُّها** (قرارُ المالك ٢٠٢٦-١١-٠٤): مشوارُ طلب الواتساب يُقرأ «مع طلب» لا «بلا طلب».
func TestWAORD013_DriverMonitorCountsIt(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-013")
	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	drv := h.driverOf(c.ID)
	deliverOrder(t, h, c.ID, drv)

	now := time.Now()
	ws, err := drivermonitor.LoadTenures(ctxBG(), h.Pool, drv.ID, now.Add(-time.Hour), now.Add(time.Minute), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 1 || ws[0].OrderID != c.ID {
		t.Fatalf("**نوافذُ السائق %d — والمنتظَر نافذةُ طلب الواتساب**: %+v", len(ws), ws)
	}
	if ws[0].Store != "فلافل الأمير" {
		t.Fatalf("**النافذةُ بلا اسم متجر**: %q", ws[0].Store)
	}
}

// TestWAORD016_OpsViewAndOfferSayWhatsApp **اللوحةُ تقول «طلب واتساب» ومن كتبه، والسائقُ يُعرض
// عليه باسمه.**
func TestWAORD016_OpsViewAndOfferSayWhatsApp(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-016")
	// **سائقٌ بالدوام عند المتجر المكتوب** — يراه في طابوره فيُعرض عليه.
	w.md.f.Driver(OnShift(), LocationAt(w.storeLat, w.storeLng, time.Now()))
	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())

	r := h.GET("/api/v1/admin/orders/"+c.ID, w.md.admin.Token)
	if r.Code != http.StatusOK {
		t.Fatalf("الطلبُ في اللوحة: %d / %s", r.Code, r.Err())
	}
	o := r.JSON()
	if inner, ok := o["order"].(map[string]any); ok {
		o = inner
	}
	if o["origin"] != "whatsapp" || o["merchant_name"] != "فلافل الأمير" ||
		o["pickup_address"] != "شارع تل أبيض" || o["merchant_phone"] != "0933444555" {
		t.Fatalf("**اللوحةُ لا تقرأ طلبَ الواتساب**: أصل %v · متجر %v · عنوان %v · رقم %v",
			o["origin"], o["merchant_name"], o["pickup_address"], o["merchant_phone"])
	}
	if o["created_by_name"] != w.md.admin.Name {
		t.Fatalf("**من كتبه لا يُقرأ**: %v والمنتظَر %q", o["created_by_name"], w.md.admin.Name)
	}
	var offers int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE entity_id = $1 AND title IN ('طلب واتساب جديد بانتظارك', 'طلب واتساب أُسند إليك')`,
		c.ID).Scan(&offers)
	if offers == 0 {
		t.Fatalf("**العرضُ على السائق لا يقول «طلب واتساب»**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **أبوابُ اللوحة** — `/admin/whatsapp-orders` بقدرة `orders.whatsapp`
// ══════════════════════════════════════════════════════════════════════

const waURL = "/api/v1/admin/whatsapp-orders"

// body **حمولةُ النموذج لمتجرٍ غير تابع** — كما ترسلها اللوحة.
func (w waFx) body() map[string]any {
	return map[string]any{
		"store_name": "فلافل الأمير", "store_phone": "0933444555", "store_address": "شارع تل أبيض",
		"store_lat": w.storeLat, "store_lng": w.storeLng,
		"details": "2 شاورما دبل بدون ثوم", "goods_amount": 2400, "amount_currency": "SYP",
		"recipient_name": "أبو أحمد", "recipient_phone": "0935111222",
		"address_text": "حي الرميلة جانب الجامع", "lat": w.md.dropLat, "lng": w.md.dropLng,
		"driver_note": "اتصل قبل الوصول",
	}
}

func waPost(t *testing.T, h *Harness, tok string, body map[string]any) Res {
	t.Helper()
	return h.POSTKey(waURL, tok, uniq("wa"), body)
}

func waCreated(t *testing.T, r Res) string {
	t.Helper()
	if r.Code != http.StatusCreated {
		t.Fatalf("**الإنشاءُ رُدّ**: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	if id == "" {
		t.Fatalf("الإنشاءُ بلا معرّف: %s", r)
	}
	if n, _ := r.JSON()["number"].(float64); n <= 0 {
		t.Fatalf("الإنشاءُ بلا رقم: %s", r)
	}
	return id
}

// TestWAORD001_CreateGoesStraightToDrivers **موظّفُ العمليّات يكتب الطلبَ فينزل للسائقين — بلا موافقة.**
//
// **وعلى الشيفرة القديمة: لا باب** (٤٠٤).
func TestWAORD001_CreateGoesStraightToDrivers(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-001")
	drv := w.md.f.Driver(OnShift(), LocationAt(w.storeLat, w.storeLng, time.Now()))
	ops := h.NewUser("operations")

	id := waCreated(t, waPost(t, h, ops.Token, w.body()))
	var origin, status, payer, createdBy string
	var merchant, customer *string
	var total, cash int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT origin, status, fee_payer, merchant_id::text, customer_id::text, created_by::text,
		       total, cash_due
		  FROM orders WHERE id = $1`, id).
		Scan(&origin, &status, &payer, &merchant, &customer, &createdBy, &total, &cash); err != nil {
		t.Fatal(err)
	}
	if origin != "whatsapp" || payer != "recipient" || merchant != nil || customer != nil ||
		createdBy != ops.ID || total != 500 || cash != 500 {
		t.Fatalf("**الصفّ**: %s/%s · متجر %v · زبون %v · كاتب %s · %d/%d",
			origin, payer, merchant, customer, createdBy, total, cash)
	}
	if status != "dispatching" && status != "assigned" {
		t.Fatalf("**الحال %s — والمنتظَر الطابور بلا موافقة**", status)
	}
	// **والسائقُ القريبُ يراه** — في طابوره، أو في طلباته إن أُسند مباشرةً.
	if !containsID(h.GET("/api/v1/driver/queue", drv.Token), id) &&
		!containsID(h.GET("/api/v1/driver/orders", drv.Token), id) {
		t.Fatalf("**السائقُ القريبُ لا يرى طلبَ الواتساب**")
	}
	var audits int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log
		WHERE action = 'ops.whatsapp_order_create' AND entity_id = $1`, id).Scan(&audits)
	if audits != 1 {
		t.Fatalf("**الإنشاءُ بلا أثرٍ في سجلّ الأحداث**: %d", audits)
	}
	// **وآخرُ الطلبات في الصفحة** — والتلميحُ برقم المستلِم.
	list := h.GET(waURL+"?open=1&q=0935111222", ops.Token)
	if list.Code != http.StatusOK || !containsID(list, id) {
		t.Fatalf("**القائمةُ لا تُظهر الطلبَ لرقمه**: %d / %s", list.Code, list.Err())
	}
}

// TestWAORD003_RegisteredStoreIsCopiedNotLinked **متجرٌ من متاجرنا يُنسخ ولا يُربط** — لا يراه في
// توصيلاته، ولا يُخبَر، ولا يُلغيه.
func TestWAORD003_RegisteredStoreIsCopiedNotLinked(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-003")
	body := w.body()
	for _, k := range []string{"store_name", "store_phone", "store_address", "store_lat", "store_lng"} {
		delete(body, k)
	}
	body["store_id"] = w.md.fx.M.ID

	// **والبحثُ يجده** — بعضُ اسمه يكفي.
	st := h.GET(waURL+"/stores?q="+url.QueryEscape(w.md.fx.M.Name), w.md.admin.Token)
	if st.Code != http.StatusOK || !strings.Contains(string(st.Body), w.md.fx.M.ID) {
		t.Fatalf("**البحثُ لا يجد المتجر**: %d / %s", st.Code, st.Body)
	}

	id := waCreated(t, waPost(t, h, w.md.admin.Token, body))
	var merchant, pickupMerchant *string
	var name string
	var dist float64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT o.merchant_id::text, o.pickup_merchant_id::text, o.pickup_name,
		       ST_Distance(o.pickup_override, m.location)
		  FROM orders o JOIN merchants m ON m.id = $2::uuid WHERE o.id = $1`, id, w.md.fx.M.ID).
		Scan(&merchant, &pickupMerchant, &name, &dist); err != nil {
		t.Fatal(err)
	}
	if merchant != nil || pickupMerchant == nil || *pickupMerchant != w.md.fx.M.ID ||
		name != w.md.fx.M.Name || dist > 1 {
		t.Fatalf("**المتجرُ لم يُنسخ أو رُبط**: متجر %v · أثر %v · اسم %q · بُعد %v", merchant, pickupMerchant, name, dist)
	}
	if containsID(h.GET(w.md.url("/deliveries"), w.md.fx.Tok), id) {
		t.Fatalf("**طلبُ الواتساب ظهر في «توصيلاتي» عند المتجر**")
	}
	if r := h.POST("/api/v1/merchant/deliveries/"+id+"/cancel", w.md.fx.Tok, map[string]any{}); r.Code < 400 {
		t.Fatalf("**المتجرُ ألغى طلبَ واتسابٍ نُسخ اسمُه عليه**: %d", r.Code)
	}
	var told int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2`, w.md.fx.M.Owner.ID, id).Scan(&told)
	if told != 0 {
		t.Fatalf("**صاحبُ المتجر أُخبر بطلبٍ لم يرسله**: %d", told)
	}
}

// TestWAORD005b_CancelRouteBeforePickupOnly **«إلغاء الطلب» من صفحة الواتساب** — بسببٍ ومن يملك
// القدرةَ وحدَها، وقبل الاستلام وحدَه، ولطلب الواتساب وحدَه.
func TestWAORD005b_CancelRouteBeforePickupOnly(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-005b")
	capRole(t, h, "qa_wa_only", authz.OrdersWhatsApp)
	_, tok := capUser(t, h, "qa_wa_only")

	id := waCreated(t, waPost(t, h, tok, w.body()))
	if r := h.POST(waURL+"/"+id+"/cancel", tok, map[string]any{"reason": ""}); r.Code != http.StatusBadRequest {
		t.Fatalf("**إلغاءٌ بلا سبب**: %d", r.Code)
	}
	r := h.POST(waURL+"/"+id+"/cancel", tok, map[string]any{"reason": "الزبون ألغى بالواتساب"})
	if r.Code != http.StatusOK {
		t.Fatalf("**الإلغاءُ رُدّ**: %d / %s", r.Code, r.Err())
	}
	if st := h.statusOf(id); st != "cancelled" {
		t.Fatalf("الحال %s", st)
	}
	var audits int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log
		WHERE action = 'ops.whatsapp_order_cancel' AND entity_id = $1`, id).Scan(&audits)
	if audits != 1 {
		t.Fatalf("**الإلغاءُ بلا أثر**: %d", audits)
	}

	// **وبعد الاستلام لا** — الحلُّ من بطاقة الطلب.
	id2 := waCreated(t, waPost(t, h, tok, w.body()))
	drv := h.driverOf(id2)
	for _, to := range []string{"at_pickup", "picked_up"} {
		if r := h.POST("/api/v1/driver/orders/"+id2+"/transition", drv.Token, map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("%s: %s", to, r)
		}
	}
	if r := h.POST(waURL+"/"+id2+"/cancel", tok, map[string]any{"reason": "متأخر"}); r.Err() != "wa_cancel_after_pickup" {
		t.Fatalf("**إلغاءٌ بعد الاستلام**: %d / %s", r.Code, r.Err())
	}
	// **وموظّفُ العمليّات يُلغيه من بابه العامّ قبل الاستلام** — المكتبُ منشئُه، فلا يسري عليه
	// «بعد التحويل لا تُلغي العمليّات».
	ops := h.NewUser("operations")
	id3 := waCreated(t, waPost(t, h, ops.Token, w.body()))
	if r := h.POST("/api/v1/admin/orders/"+id3+"/transition", ops.Token,
		map[string]any{"to": "cancelled", "note": "طلب مكرر"}); r.Code != http.StatusOK {
		t.Fatalf("**العمليّاتُ لا تُلغي طلبَ واتسابٍ كتبته**: %d / %s", r.Code, r.Err())
	}
	// **ولا يُلغى من هنا ما ليس طلبَ واتساب.**
	md := w.md.create(t, h, "recipient", uniq("wa5b"))
	if r := h.POST(waURL+"/"+md+"/cancel", tok, map[string]any{"reason": "تجربة"}); r.Err() != "wa_not_whatsapp" {
		t.Fatalf("**ألغى «لدي توصيلة» من باب الواتساب**: %d / %s", r.Code, r.Err())
	}
}

// TestWAORD008_FeeOverrideBoundsAndAudit **الأجرةُ تُعدَّل بحدّيها، والمحسوبةُ والمكتوبةُ في السجلّ.**
func TestWAORD008_FeeOverrideBoundsAndAudit(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-008")

	b := w.body()
	b["fee"] = 700
	id := waCreated(t, waPost(t, h, w.md.admin.Token, b))
	var fee, driverFee, cash int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT delivery_fee, driver_fee, cash_due FROM orders WHERE id = $1`, id).Scan(&fee, &driverFee, &cash); err != nil {
		t.Fatal(err)
	}
	if fee != 700 || driverFee != 630 || cash != 700 {
		t.Fatalf("**الأجرةُ المعدَّلة**: %d · سائق %d · نقد %d — المنتظَر 700 · 630 · 700", fee, driverFee, cash)
	}
	var details string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT details::text FROM audit_log
		WHERE action = 'ops.whatsapp_order_create' AND entity_id = $1`, id).Scan(&details)
	if !strings.Contains(details, `"computed_fee": 500`) || !strings.Contains(details, `"fee": 700`) {
		t.Fatalf("**السجلُّ لا يحمل الأجرتين**: %s", details)
	}

	b["fee"] = 5000
	r := waPost(t, h, w.md.admin.Token, b)
	if r.Code != http.StatusUnprocessableEntity || r.Err() != "wa_fee_out_of_range" {
		t.Fatalf("**أجرةٌ فوق الحدّ**: %d / %s", r.Code, r.Err())
	}
	errBody, _ := r.JSON()["error"].(map[string]any)
	if d, _ := errBody["details"].(map[string]any); d == nil || d["min"] != "10" || d["max"] != "1000" {
		t.Fatalf("**الحدّان لا يُقالان**: %s", r.Body)
	}

	// **وحصّةٌ ثابتةٌ تبلغ الأجرةَ تُرسل سائقاً بلا أجر.**
	h.Setting("delivery.share_whatsapp", "100")
	h.Setting("delivery.share_whatsapp_mode", `"fixed"`)
	b["fee"] = 80
	if r := waPost(t, h, w.md.admin.Token, b); r.Err() != "wa_fee_below_share" {
		t.Fatalf("**أجرةٌ دون الحصّة**: %d / %s", r.Code, r.Err())
	}
}

// TestWAORD009_Permission **بقدرته وحدَها** — بلاها ٤٠٣، ودورٌ مخصَّصٌ يُمنَحها يكتب.
func TestWAORD009_Permission(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-009")
	fin := h.NewUser("finance")
	if r := waPost(t, h, fin.Token, w.body()); r.Code != http.StatusForbidden {
		t.Fatalf("**الماليّةُ أنشأت طلبَ واتساب**: %d", r.Code)
	}
	if r := h.GET(waURL+"/stores?q=qa", fin.Token); r.Code != http.StatusForbidden {
		t.Fatalf("**الماليّةُ بحثت في المتاجر**: %d", r.Code)
	}
	capRole(t, h, "qa_wa_role", authz.OrdersWhatsApp)
	_, tok := capUser(t, h, "qa_wa_role")
	waCreated(t, waPost(t, h, tok, w.body()))
}

// TestWAORD010_Validation **ما ينقص يُقال برمزه.**
func TestWAORD010_Validation(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-010")
	tok := w.md.admin.Token
	with := func(mut func(b map[string]any)) map[string]any {
		b := w.body()
		mut(b)
		return b
	}
	inactive := w.md.f.Merchant(func(s *merchantSpec) { s.status = "inactive" })
	noPin := w.md.f.Merchant()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET location = NULL WHERE id = $1`, noPin.ID); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		body map[string]any
		code string
	}{
		{"بلا متجر", with(func(b map[string]any) { delete(b, "store_name"); delete(b, "store_lat"); delete(b, "store_lng") }), "wa_store_required"},
		{"متجران", with(func(b map[string]any) { b["store_id"] = w.md.fx.M.ID }), "wa_store_conflict"},
		{"متجرٌ موقوف", with(func(b map[string]any) {
			delete(b, "store_name")
			delete(b, "store_lat")
			delete(b, "store_lng")
			b["store_id"] = inactive.ID
		}), "wa_store_unavailable"},
		{"متجرٌ بلا موقع", with(func(b map[string]any) {
			delete(b, "store_name")
			delete(b, "store_lat")
			delete(b, "store_lng")
			b["store_id"] = noPin.ID
		}), "wa_store_unavailable"},
		// **والتفاصيلُ واسمُ المستلِم وعنوانُه اختياريّةٌ منذ ٢٠٢٦-١١-٠٦** — `TestWAORD040`؛ ورقمُه وحدَه إلزاميّ.
		{"بلا رقم المستلم", with(func(b map[string]any) { b["recipient_phone"] = "" }), "invalid_phone"},
		// **وبلا نقطةٍ يُنشأ بالعنوان — بأجرةٍ يكتبها الموظّف** (ملحقُ المالك ٢٠٢٦-١١-٠٥، ب).
		{"بلا نقطة المستلم ولا أجرة", with(func(b map[string]any) { delete(b, "lat"); delete(b, "lng") }), "wa_fee_required"},
		{"نصفُ نقطة", with(func(b map[string]any) { delete(b, "lng") }), "bad_point"},
		// **ورمزُ الرقم الفاسد رمزُ «لدي توصيلة» نفسُه** — لا يُخترَع ثانٍ.
		{"رقمٌ فاسد", with(func(b map[string]any) { b["recipient_phone"] = "12" }), "invalid_phone"},
	} {
		r := waPost(t, h, tok, c.body)
		if r.Code < 400 || r.Err() != c.code {
			t.Errorf("**%s**: %d / %s — المنتظَر %s", c.name, r.Code, r.Err(), c.code)
		}
	}
	// **ونقطةٌ خارجَ التغطية بحرّاسها القائمة** — لا رمزٌ جديد.
	far := with(func(b map[string]any) { b["lat"] = w.md.z.Lat + 0.5 })
	if r := waPost(t, h, tok, far); r.Code < 400 || strings.HasPrefix(r.Err(), "wa_") {
		t.Errorf("**خارجَ التغطية**: %d / %s", r.Code, r.Err())
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM orders WHERE origin = 'whatsapp'
		AND created_at > now() - interval '5 minutes' AND recipient_name = 'أبو أحمد'
		AND pickup_name = 'فلافل الأمير' AND address_text = ''`).Scan(&n)
	if n != 0 {
		t.Fatalf("**طلبٌ ناقصٌ كُتب**")
	}
}

// TestWAORD011_Idempotent **ضغطتان بالمفتاح نفسِه طلبٌ واحد.**
func TestWAORD011_Idempotent(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-011")
	key := uniq("wa11")
	first := h.POSTKey(waURL, w.md.admin.Token, key, w.body())
	id := waCreated(t, first)
	again := h.POSTKey(waURL, w.md.admin.Token, key, w.body())
	if again.Code != http.StatusCreated || !again.Replay() {
		t.Fatalf("**الإعادةُ لم تُقرأ إعادة**: %d · replay=%v", again.Code, again.Replay())
	}
	if got, _ := again.JSON()["id"].(string); got != id {
		t.Fatalf("**الإعادةُ أنشأت طلباً ثانياً**: %s ≠ %s", got, id)
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log
		WHERE action = 'ops.whatsapp_order_create' AND actor_user_id = $1::uuid`, w.md.admin.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("**طلبان بمفتاحٍ واحد**: %d", n)
	}
}

// TestWAORD012_ListFilterSearchExportReport **طلبُ الواتساب طلبٌ نظاميٌّ في القوائم** — بفلتره، وباسم
// متجره في البحث، وبنوعه في الملفّ، وفي التقارير. **و«توصيلات المتاجر» لم تعد تشمله.**
func TestWAORD012_ListFilterSearchExportReport(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-012")
	tok := w.md.admin.Token
	wa := waEngineCreate(t, h, w.md.admin.ID, w.typed()).ID
	md := w.md.create(t, h, "recipient", uniq("wa12"))

	byKind := h.GET("/api/v1/admin/orders?kind=whatsapp&per_page=100", tok)
	if byKind.Code != http.StatusOK || !containsID(byKind, wa) || containsID(byKind, md) {
		t.Fatalf("**فلترُ «طلبات الواتساب»**: %d · واتساب %v · توصيلة %v",
			byKind.Code, containsID(byKind, wa), containsID(byKind, md))
	}
	byMD := h.GET("/api/v1/admin/orders?kind=merchant_delivery&per_page=100", tok)
	if byMD.Code != http.StatusOK || containsID(byMD, wa) || !containsID(byMD, md) {
		t.Fatalf("**«توصيلات المتاجر» تشمل طلبَ الواتساب**: واتساب %v · توصيلة %v",
			containsID(byMD, wa), containsID(byMD, md))
	}
	// **والبحثُ باسم المتجر المكتوب** — بحثُ الموظّفين.
	if s := h.GET("/api/v1/admin/orders?per_page=100&query="+url.QueryEscape("فلافل الأمير"), tok); !containsID(s, wa) {
		t.Fatalf("**البحثُ باسم المتجر المكتوب لا يجد طلبَ الواتساب**: %d", s.Code)
	}
	exp := h.GET("/api/v1/admin/orders/export?kind=whatsapp", tok)
	if exp.Code != http.StatusOK || !strings.Contains(string(exp.Body), "طلب واتساب") ||
		!strings.Contains(string(exp.Body), "فلافل الأمير") {
		t.Fatalf("**الملفُّ لا يقول «طلب واتساب» ولا اسمَ المتجر**: %d", exp.Code)
	}
	if rep := h.GET("/api/v1/admin/reports?kind=whatsapp", tok); rep.Code != http.StatusOK {
		t.Fatalf("**التقاريرُ ترفض نوعَ «طلب واتساب»**: %d / %s", rep.Code, rep.Err())
	}
}

// TestWAORD014_NoFreeDeliveryClaim **عرضُ التوصيل المجّانيّ لا يمسّ طلبَ الواتساب** — لا يُعفى ولا
// يستهلك عرضَ متجر.
func TestWAORD014_NoFreeDeliveryClaim(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-014")
	_, _ = h.Pool.Exec(ctxBG(), `UPDATE free_delivery_campaigns SET active = false`)
	now := time.Now().UTC()
	var camp string
	if err := h.Pool.QueryRow(ctxBG(), `INSERT INTO free_delivery_campaigns (active, starts_at, ends_at, kinds)
		VALUES (true, $1, $2, ARRAY['standard', 'merchant']) RETURNING id::text`,
		now.Add(-time.Hour), now.Add(24*time.Hour)).Scan(&camp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `UPDATE free_delivery_campaigns SET active = false WHERE id = $1`, camp)
	})
	id := waCreated(t, waPost(t, h, w.md.admin.Token, w.body()))
	var waived, fee int64
	var claims int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT promo_delivery_waived, delivery_fee FROM orders WHERE id = $1`, id).Scan(&waived, &fee)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM free_delivery_claims WHERE order_id = $1`, id).Scan(&claims)
	if waived != 0 || fee != 500 || claims != 0 {
		t.Fatalf("**التوصيلُ المجّانيّ مسّ طلبَ الواتساب**: إعفاء %d · أجرة %d · مطالبات %d", waived, fee, claims)
	}
}
