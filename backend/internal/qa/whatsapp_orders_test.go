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
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// waFx **منطقةٌ ومتجرٌ من متاجرنا ونقطةُ مستلِم** — وأجرةُ «لدي توصيلة» ٥٠٠ بحصّة ١٠٪.
type waFx struct {
	md mdFx
	// storeLat/storeLng **متجرٌ غيرُ تابع** داخلَ المنطقة.
	storeLat, storeLng float64
}

func newWAFx(t *testing.T, h *Harness, name string) waFx {
	t.Helper()
	m := newMDFx(t, h, name)
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

// TestWhatsAppFeeFromStorePoint **الأجرةُ كالتوصيلات من المتجر إلى المستلِم** — والخاصّةُ
// تسبقها، **وعرضُ السعر والإنشاءُ بدالّةٍ واحدة.**
func TestWhatsAppFeeFromStorePoint(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-FEE")
	// **أساسٌ ١٠٠ ولكلّ كيلومترٍ ٢٥** — والمسافةُ هنا أقلُّ من كيلومتر ⇒ ١٢٥.
	h.Setting("delivery.merchant_base_fee", "100")
	h.Setting("delivery.merchant_per_km", "25")
	h.Setting("delivery.merchant_by_distance", "true")
	h.Setting("delivery.merchant_max_fee", "0")
	in := w.typed()
	q, err := h.Orders.QuoteWhatsAppOrder(ctxBG(), "", in.StoreLat, in.StoreLng, *in.Lat, *in.Lng)
	if err != nil {
		t.Fatalf("عرضُ السعر: %v", err)
	}
	if q.Fee != 125 || q.FeeBasis != orders.FeeBasisDistance || q.DistanceM <= 0 || q.DistanceM >= 1000 {
		t.Fatalf("**عرضُ السعر %d (%s، %d م) والمنتظَر 125 على مسافةٍ دون الكيلومتر**",
			q.Fee, q.FeeBasis, q.DistanceM)
	}
	if q.DriverFee != 113 || q.PlatformShare != 12 {
		t.Fatalf("**الحصّة %d والسائق %d — المنتظَر 12 و113** (الكسرُ للسائق)", q.PlatformShare, q.DriverFee)
	}
	c := waEngineCreate(t, h, w.md.admin.ID, in)
	if c.Fee != q.Fee || c.ComputedFee != q.Fee {
		t.Fatalf("**الإنشاءُ كتب %d وعرضُ السعر قال %d**", c.Fee, q.Fee)
	}

	// **والأجرةُ الخاصّةُ تسبق المسافة.**
	h.Setting("delivery.merchant_fee", "300")
	q2, err := h.Orders.QuoteWhatsAppOrder(ctxBG(), "", in.StoreLat, in.StoreLng, *in.Lat, *in.Lng)
	if err != nil {
		t.Fatal(err)
	}
	if q2.Fee != 300 || q2.FeeBasis != orders.FeeBasisFlat {
		t.Fatalf("**الأجرةُ الخاصّة لم تسبق**: %d (%s)", q2.Fee, q2.FeeBasis)
	}
	// **ومتجرٌ من متاجرنا يُسعَّر من موقعه** — بالدالّة نفسِها.
	q3, err := h.Orders.QuoteWhatsAppOrder(ctxBG(), w.md.fx.M.ID, nil, nil, *in.Lat, *in.Lng)
	if err != nil {
		t.Fatalf("عرضُ السعر لمتجرٍ من متاجرنا: %v", err)
	}
	if q3.Fee != 300 {
		t.Fatalf("متجرٌ من متاجرنا سُعّر %d", q3.Fee)
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
	if got.RecipientPhone != "+963935111222" || got.CustomerName != "أبو أحمد" ||
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
	ws, err := drivermonitor.LoadWindows(ctxBG(), h.Pool, drv.ID, now.Add(-time.Hour), now.Add(time.Minute))
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
