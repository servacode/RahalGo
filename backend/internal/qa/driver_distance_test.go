package qa

// ══════════════════════════════════════════════════════════════════════
// **المسافة حسب طريق السائق** — قرار المالك ٢٠٢٦-١١-٠٥
// ══════════════════════════════════════════════════════════════════════
//
// «الطريق يحسب حسب طريق السائق مو ع كيفنا.» القاعدة المعتمدة
// (`D:/tmp/drivers_real.sql`) صارت دالةً واحدة في Go (`drivermonitor.IsDrive`) —
// **وهون تنقاس بالـSQL نفسه حرفاً** (`testdata/drivers_real_fns.sql`، أوراكل اختبار
// لا تنفيذٌ ثانٍ). والتنبيه صار بالقيادة لا بالرجفة، والطلب المتروك ومشوار الإرجاع
// صاروا «بطلب».

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
)

// rdPoint نقطة أثر بدقّتها وسرعتها (والفارغ NULL).
type rdPoint struct {
	at       time.Time
	lat, lng float64
	acc, spd *float64
}

func rdF(v float64) *float64 { return &v }

// rdPut يكتب نقاطاً في `driver_trail` مباشرة.
func rdPut(t *testing.T, h *Harness, driver string, pts []rdPoint) {
	t.Helper()
	for _, p := range pts {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO driver_trail (driver_id, recorded_at, lat, lng, accuracy_m, speed_mps)
			VALUES ($1::uuid, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`,
			driver, p.at, p.lat, p.lng, p.acc, p.spd); err != nil {
			t.Fatalf("كتابة نقطة الأثر: %v", err)
		}
	}
}

// rdMetersLat متر لكل درجة عرض.
const rdMetersLat = 6371000.0 * math.Pi / 180

// rdDrive يقود شمالاً `m` متر بسرعة `mps` بنقطة كل `step` من `from`.
func rdDrive(from rdPoint, m, mps float64, step time.Duration) []rdPoint {
	n := int(math.Round(m / (mps * step.Seconds())))
	out := make([]rdPoint, 0, n)
	p := from
	for i := 0; i < n; i++ {
		p = rdPoint{at: p.at.Add(step), lat: p.lat + mps*step.Seconds()/rdMetersLat, lng: p.lng,
			acc: rdF(6), spd: rdF(mps)}
		out = append(out, p)
	}
	return out
}

func rdLast(pts []rdPoint) rdPoint { return pts[len(pts)-1] }

// rdDayBase يومٌ قبل يومين بتوقيت دمشق عند العاشرة — داخل يوم حسابه.
func rdDayBase() (string, time.Time) {
	loc := drivermonitor.Loc()
	d := time.Now().In(loc).AddDate(0, 0, -2)
	base := time.Date(d.Year(), d.Month(), d.Day(), 10, 0, 0, 0, loc)
	return base.Format("2006-01-02"), base
}

type rdSummary struct {
	KmWithOrder      float64 `json:"km_with_order"`
	KmToPickup       float64 `json:"km_to_pickup"`
	KmNoOrder        float64 `json:"km_no_order"`
	KmTotal          float64 `json:"km_total"`
	DriveKmWithOrder float64 `json:"drive_km_with_order"`
	DriveKmToPickup  float64 `json:"drive_km_to_pickup"`
	DriveKmNoOrder   float64 `json:"drive_km_no_order"`
	DriveKmTotal     float64 `json:"drive_km_total"`
	NotDrivingKm     float64 `json:"not_driving_km"`
	Orders           int     `json:"orders"`
}

type rdDayResp struct {
	Day     string    `json:"day"`
	Today   string    `json:"today"`
	Alert   bool      `json:"alert"`
	Summary rdSummary `json:"summary"`
	Orders  []struct {
		ID         string `json:"id"`
		EndKind    string `json:"end_kind"`
		ToPickupM  int    `json:"to_pickup_m"`
		WithOrderM int    `json:"with_order_m"`
		Shared     bool   `json:"shared"`
	} `json:"orders"`
}

func rdFetch(t *testing.T, h *Harness, admin, driver, day string) rdDayResp {
	t.Helper()
	res := h.GET(fmt.Sprintf("/api/v1/admin/driver-monitor/%s?day=%s", driver, day), admin)
	if res.Code != 200 {
		t.Fatalf("يوم السائق: %s", res)
	}
	var env struct {
		Data rdDayResp `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatalf("فك الردّ: %v", err)
	}
	return env.Data
}

// rdEvent يكتب سطراً في سجلّ الطلب بحامله ووقته.
func rdEvent(t *testing.T, h *Harness, order, from, to, driver string, at time.Time) {
	t.Helper()
	var drv any
	if driver != "" {
		drv = driver
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_events (order_id, from_status, to_status, driver_id, created_at)
		VALUES ($1::uuid, $2, $3, $4::uuid, $5)`, order, from, to, drv, at); err != nil {
		t.Fatalf("سطر السجلّ %s→%s: %v", from, to, err)
	}
}

// TestRealDrive_EqualsValidatedSQL **Go = الـSQL المعتمد حرفاً**: يومٌ مصنوع فيه قيادة
// وزحف ورجفة ودقّة سيّئة وسرعة جوال فارغة وانقطاع وثلاثة طلبات (اثنان متداخلان) ⇒
// كل رقم من `DriverDay` يساوي `rg_real` إلى المليمتر.
func TestRealDrive_EqualsValidatedSQL(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	f := h.Factory()
	drv := f.Driver()
	m := f.Merchant()
	day, base := rdDayBase()

	rng := rand.New(rand.NewSource(5112026))
	p := rdPoint{at: base, lat: 35.62, lng: 38.71, acc: rdF(8), spd: rdF(0)}
	pts := []rdPoint{p}
	add := func(q []rdPoint) { pts = append(pts, q...); p = rdLast(pts) }
	// ١ قيادة للمتجر ٥ م/ث — ودقّةٌ تتبدّل: ٥ · ٣٥ (فوق ٣٠) · فارغة · ٦٠ (تُهمل كلياً)
	leg := rdDrive(p, 1500, 5, 10*time.Second)
	for i := range leg {
		switch i % 7 {
		case 1:
			leg[i].acc = rdF(35)
		case 3:
			leg[i].acc = nil
		case 5:
			leg[i].acc = rdF(60)
		}
		if i%4 == 2 {
			leg[i].spd = nil
		}
		if i%9 == 4 {
			leg[i].spd = rdF(1.2) // الجوال يقول بطيء
		}
	}
	add(leg)
	tAt := p.at
	// ٢ وقوف ٤ دقائق عند المتجر برجفة صغيرة (وقفة)
	for i := 1; i <= 4; i++ {
		add([]rdPoint{{at: p.at.Add(time.Minute), lat: p.lat + float64(i%2)*0.00002, lng: p.lng, acc: rdF(10), spd: rdF(0)}})
	}
	tPicked := p.at
	// ٣ قيادة بالطلب ٦ م/ث
	add(rdDrive(p, 2000, 6, 10*time.Second))
	tDelivered := p.at
	// ٤ رجفة الواقف ١٠ دقائق (٢٥–٤٥ م) — محسوبة وليست قيادة
	c := p
	for i := 1; i <= 10; i++ {
		off := (25 + float64(rng.Intn(20))) / rdMetersLat
		if i%2 == 0 {
			off = -off
		}
		add([]rdPoint{{at: c.at.Add(time.Duration(i) * time.Minute), lat: c.lat + off, lng: c.lng, acc: rdF(20), spd: rdF(0.3)}})
	}
	// ٥ زحف ٢ م/ث ثلاث دقائق
	add(rdDrive(p, 360, 2, 20*time.Second))
	// ٦ انقطاع ٨ دقائق
	add([]rdPoint{{at: p.at.Add(8 * time.Minute), lat: p.lat + 0.004, lng: p.lng, acc: rdF(9), spd: rdF(7)}})
	t2a := p.at
	// ٧ طلب ثانٍ: للمتجر ب ٧ م/ث، والثالث يُسند وهو بالطريق
	add(rdDrive(p, 1400, 7, 10*time.Second))
	t3a := p.at
	add(rdDrive(p, 700, 7, 10*time.Second))
	t2p := p.at
	add(rdDrive(p, 1800, 7, 10*time.Second))
	t2d := p.at
	add(rdDrive(p, 900, 7, 10*time.Second))
	t3p := p.at
	add(rdDrive(p, 1200, 7, 10*time.Second))
	t3d := p.at
	// ٨ قفزة مستحيلة تُهمل، ثم رجوع بلا طلب
	add([]rdPoint{{at: p.at.Add(5 * time.Second), lat: p.lat + 0.02, lng: p.lng, acc: rdF(5), spd: rdF(9)}})
	add(rdDrive(rdPoint{at: p.at, lat: p.lat - 0.02, lng: p.lng}, 1000, 8, 10*time.Second))
	rdPut(t, h, drv.ID, pts)

	dmOrder(t, h, drv.ID, m.ID, base, tPicked, tDelivered)
	dmOrder(t, h, drv.ID, m.ID, t2a, t2p, t2d)
	dmOrder(t, h, drv.ID, m.ID, t3a, t3p, t3d)
	_ = tAt

	// ── الأوراكل على اتصالٍ واحد (دوالّ pg_temp للجلسة) ──
	fns, err := os.ReadFile("testdata/drivers_real_fns.sql")
	if err != nil {
		t.Fatalf("قراءة الأوراكل: %v", err)
	}
	ctx := context.Background()
	conn, err := h.Pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, string(fns)); err != nil {
		t.Fatalf("تحميل الأوراكل: %v", err)
	}
	// **حدودُ يوم الحساب من القاعدة نفسها** (٩ ← ٩) — مستقلّةً عن Go.
	var from, to time.Time
	if err := h.Pool.QueryRow(ctx, `
		SELECT account_day_start($1::date), account_day_start($1::date + 1)`, day).Scan(&from, &to); err != nil {
		t.Fatal(err)
	}
	var kmTotal, kmWith, kmTo, kmNo, driveOrders, driveNo, driveNoMin, kmSlow float64
	if err := conn.QueryRow(ctx, `
		SELECT km_total, km_with, km_to, km_no, drive_orders, drive_no, drive_no_min, km_slow
		  FROM pg_temp.rg_real($1::uuid, $2, $3)`, drv.ID, from, to).
		Scan(&kmTotal, &kmWith, &kmTo, &kmNo, &driveOrders, &driveNo, &driveNoMin, &kmSlow); err != nil {
		t.Fatalf("الأوراكل: %v", err)
	}
	if kmTotal < 5 || driveOrders < 5 || kmSlow < 0.3 {
		t.Fatalf("اليوم المصنوع ضعيف: كل=%.3f طلبات=%.3f بطيء=%.3f", kmTotal, driveOrders, kmSlow)
	}

	got := rdFetch(t, h, admin.Token, drv.ID, day).Summary
	const eps = 1e-6 // كم = مليمتر
	check := func(name string, goV, sqlV float64) {
		if math.Abs(goV-sqlV) > eps {
			t.Errorf("%s: Go=%.9f و SQL=%.9f", name, goV, sqlV)
		}
	}
	check("km_total", got.KmTotal, kmTotal)
	check("km_with", got.KmWithOrder, kmWith)
	check("km_to", got.KmToPickup, kmTo)
	check("km_no", got.KmNoOrder, kmNo)
	check("drive_orders", got.DriveKmWithOrder+got.DriveKmToPickup, driveOrders)
	check("drive_no", got.DriveKmNoOrder, driveNo)
	check("slow_or_drift", got.NotDrivingKm, kmSlow)
	check("drive_total", got.DriveKmTotal, driveOrders+driveNo)
	t.Logf("SQL: كل=%.3f بطلب=%.3f بلا=%.3f بطيء=%.3f", kmTotal, driveOrders, driveNo, kmSlow)
}

// TestDriverMonitor_AlertUsesRealDriving **التنبيه بالقيادة لا بالرجفة**: ١٥ كم زحف
// ورجفة والعتبة ١٠ ⇒ لا تنبيه؛ و١٢ كم قيادة فعلية بلا طلب ⇒ تنبيه.
func TestDriverMonitor_AlertUsesRealDriving(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	f := h.Factory()
	h.Setting("drivers.monitor_km_alert", "10")
	day, base := rdDayBase()

	// زحف متعرّج: ١٥٠ م كل دقيقة ذهاباً وإياباً (٢٫٥ م/ث) — ١٠٠ مقطع = ١٥ كم محسوبة
	crawl := f.Driver()
	var zig []rdPoint
	for i := 0; i <= 100; i++ {
		lat := 35.70
		if i%2 == 1 {
			lat += 150 / rdMetersLat
		}
		zig = append(zig, rdPoint{at: base.Add(time.Duration(i) * time.Minute), lat: lat, lng: 38.80, acc: rdF(12), spd: rdF(2.5)})
	}
	rdPut(t, h, crawl.ID, zig)
	a := rdFetch(t, h, admin.Token, crawl.ID, day)
	if a.Summary.KmNoOrder < 14.5 {
		t.Fatalf("المحسوب بلا طلب = %.2f والمتوقَّع ~١٥", a.Summary.KmNoOrder)
	}
	if a.Alert || a.Summary.DriveKmNoOrder > 0.01 {
		t.Fatalf("زحف ورجفة طلّعوا تنبيه: alert=%v قيادة=%.2f", a.Alert, a.Summary.DriveKmNoOrder)
	}

	// قيادة فعلية ١٢ كم بلا طلب
	rider := f.Driver()
	rdPut(t, h, rider.ID, append([]rdPoint{{at: base, lat: 35.72, lng: 38.82, acc: rdF(6), spd: rdF(8)}},
		rdDrive(rdPoint{at: base, lat: 35.72, lng: 38.82}, 12000, 8, 10*time.Second)...))
	b := rdFetch(t, h, admin.Token, rider.ID, day)
	if !b.Alert || math.Abs(b.Summary.DriveKmNoOrder-12) > 0.05 {
		t.Fatalf("١٢ كم قيادة بلا طلب: alert=%v قيادة=%.2f", b.Alert, b.Summary.DriveKmNoOrder)
	}
}

// rdOrderRow طلبٌ بحامله الحاليّ وأزمنته — والسجلّ يُكتب بعده بيد الاختبار.
func rdOrderRow(t *testing.T, h *Harness, driver *string, merchant, status string, cols map[string]time.Time) string {
	t.Helper()
	cust := h.Customer()
	var oid string
	var drv any
	if driver != nil {
		drv = *driver
	}
	at := func(k string) any {
		if v, ok := cols[k]; ok {
			return v
		}
		return nil
	}
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
		                    payment_method, subtotal, delivery_fee, total,
		                    accepted_at, picked_up_at, delivered_at, closed_at, goods_handed_at, return_to)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, 'الرقة',
		        ST_SetSRID(ST_MakePoint(38.5110,35.5000),4326)::geography, 'cash', 10, 5, 15,
		        $5, $6, $7, $8, $9, $10)
		RETURNING id::text`, cust.ID, merchant, drv, status,
		at("accepted"), at("picked"), at("delivered"), at("closed"), at("handed"),
		func() any {
			if _, ok := cols["return"]; ok {
				return "office"
			}
			return nil
		}()).Scan(&oid); err != nil {
		t.Fatalf("إنشاء الطلب: %v", err)
	}
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM orders WHERE id = $1::uuid`, oid) })
	return oid
}

// TestDriverMonitor_ReleasedOrderIsOrderKm **طلبٌ تركه السائق**: قَبِل، ومشى ١ كم نحو
// المتجر، ثم ترك (والطلب صار لغيره) ⇒ الكيلو «للاستلام» لا «بلا طلب».
func TestDriverMonitor_ReleasedOrderIsOrderKm(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	f := h.Factory()
	d1, d2 := f.Driver(), f.Driver()
	m := f.Merchant()
	day, base := rdDayBase()

	start := rdPoint{at: base, lat: 35.74, lng: 38.84, acc: rdF(6), spd: rdF(6)}
	trail := append([]rdPoint{start}, rdDrive(start, 1000, 5, 10*time.Second)...)
	released := rdLast(trail).at
	// بعد الترك يمشي ٥٠٠ م بلا طلب
	trail = append(trail, rdDrive(rdLast(trail), 500, 5, 10*time.Second)...)
	rdPut(t, h, d1.ID, trail)

	d2id := d2.ID
	oid := rdOrderRow(t, h, &d2id, m.ID, "assigned", map[string]time.Time{"accepted": base.Add(-time.Minute)})
	rdEvent(t, h, oid, "dispatching", "assigned", d1.ID, base)
	rdEvent(t, h, oid, "assigned", "dispatching", d1.ID, released)
	rdEvent(t, h, oid, "dispatching", "assigned", d2.ID, released.Add(2*time.Minute))

	got := rdFetch(t, h, admin.Token, d1.ID, day)
	if math.Abs(got.Summary.DriveKmToPickup-1.0) > 0.02 {
		t.Errorf("للاستلام = %.3f كم والمتوقَّع ١ — الطلب المتروك صار «بلا طلب»", got.Summary.DriveKmToPickup)
	}
	if math.Abs(got.Summary.DriveKmNoOrder-0.5) > 0.02 {
		t.Errorf("بلا طلب = %.3f كم والمتوقَّع ٠٫٥", got.Summary.DriveKmNoOrder)
	}
	var row *struct {
		ID         string `json:"id"`
		EndKind    string `json:"end_kind"`
		ToPickupM  int    `json:"to_pickup_m"`
		WithOrderM int    `json:"with_order_m"`
		Shared     bool   `json:"shared"`
	}
	for i := range got.Orders {
		if got.Orders[i].ID == oid {
			row = &got.Orders[i]
		}
	}
	if row == nil {
		t.Fatalf("الطلب المتروك ما ظهر بطلبات اليوم: %+v", got.Orders)
	}
	if row.EndKind != "released" || math.Abs(float64(row.ToPickupM)-1000) > 20 {
		t.Errorf("سطر الطلب: %+v — المتوقَّع released و١٠٠٠ م للمتجر", *row)
	}
}

// TestDriverMonitor_ReturnTripIsOrderKm **مشوار إرجاع البضاعة جزء من الطلب**: فشل
// التسليم، فرجع ١ كم للمكتب وضغط «سلّمت البضاعة» ⇒ الكيلو «بطلب».
func TestDriverMonitor_ReturnTripIsOrderKm(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	f := h.Factory()
	drv := f.Driver()
	m := f.Merchant()
	day, base := rdDayBase()

	start := rdPoint{at: base, lat: 35.76, lng: 38.86, acc: rdF(6), spd: rdF(6)}
	trail := append([]rdPoint{start}, rdDrive(start, 600, 5, 10*time.Second)...)
	picked := rdLast(trail).at
	trail = append(trail, rdDrive(rdLast(trail), 900, 5, 10*time.Second)...)
	failed := rdLast(trail).at
	trail = append(trail, rdDrive(rdLast(trail), 1000, 5, 10*time.Second)...)
	handed := rdLast(trail).at
	trail = append(trail, rdDrive(rdLast(trail), 400, 5, 10*time.Second)...)
	rdPut(t, h, drv.ID, trail)

	id := drv.ID
	oid := rdOrderRow(t, h, &id, m.ID, "failed", map[string]time.Time{
		"accepted": base, "picked": picked, "closed": failed, "handed": handed, "return": {},
	})
	rdEvent(t, h, oid, "dispatching", "assigned", drv.ID, base)
	rdEvent(t, h, oid, "at_pickup", "picked_up", drv.ID, picked)
	rdEvent(t, h, oid, "at_dropoff", "failed", drv.ID, failed)

	got := rdFetch(t, h, admin.Token, drv.ID, day)
	// ٩٠٠ للزبون + ١٠٠٠ مشوار الإرجاع
	if math.Abs(got.Summary.DriveKmWithOrder-1.9) > 0.03 {
		t.Errorf("بطلب = %.3f كم والمتوقَّع ١٫٩ — مشوار الإرجاع صار «بلا طلب»", got.Summary.DriveKmWithOrder)
	}
	if math.Abs(got.Summary.DriveKmNoOrder-0.4) > 0.02 {
		t.Errorf("بلا طلب = %.3f كم والمتوقَّع ٠٫٤", got.Summary.DriveKmNoOrder)
	}
	for _, o := range got.Orders {
		if o.ID == oid && o.EndKind != "returned" {
			t.Errorf("نوع نهاية الطلب = %q والمتوقَّع returned", o.EndKind)
		}
	}
}

// TestDriverMonitor_StoreTransferKeepsHold **تحويلُ المتجر لا يقطع ولايةَ السائق**: قَبِل ومشى
// ١ كم للمتجر الأوّل، فحوّلت العمليّاتُ الطلبَ لمتجرٍ آخر — والطلبُ «مقبول» ومرّةً «عند المتجر» —
// فمشى ١٫٥ كم للجديد واستلم ومشى ٢ كم للزبون ⇒ ٢٫٥ للمتجر و٢ للزبون، **ولا شيء «بلا طلب».**
//
// والسجلُّ كما يكتبه بابُ التحويل حرفاً (`server/order_transfer.go`): ملاحظةٌ حالُها ما تغيّرت
// **بلا سائق**، وعند المتجر `at_pickup → assigned` **بلا سائق** — والسائقُ نفسُه ما زال ماسكَه.
func TestDriverMonitor_StoreTransferKeepsHold(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	f := h.Factory()
	m := f.Merchant()
	day, base := rdDayBase()
	for i, atPickup := range []bool{false, true} {
		name := "assigned"
		if atPickup {
			name = "at_pickup"
		}
		t.Run(name, func(t *testing.T) {
			drv := f.Driver()
			start := rdPoint{at: base.Add(time.Duration(i) * 3 * time.Hour), lat: 35.66 + float64(i)*0.1, lng: 38.75, acc: rdF(6), spd: rdF(5)}
			trail := append([]rdPoint{start}, rdDrive(start, 1000, 5, 10*time.Second)...)
			transfer := rdLast(trail).at
			trail = append(trail, rdDrive(rdLast(trail), 1500, 5, 10*time.Second)...)
			picked := rdLast(trail).at
			trail = append(trail, rdDrive(rdLast(trail), 2000, 5, 10*time.Second)...)
			delivered := rdLast(trail).at
			rdPut(t, h, drv.ID, trail)

			id := drv.ID
			oid := rdOrderRow(t, h, &id, m.ID, "delivered", map[string]time.Time{
				"accepted": start.at, "picked": picked, "delivered": delivered, "closed": delivered,
			})
			rdEvent(t, h, oid, "dispatching", "assigned", drv.ID, start.at)
			status := "assigned"
			if atPickup {
				rdEvent(t, h, oid, "assigned", "at_pickup", drv.ID, transfer.Add(-30*time.Second))
				status = "at_pickup"
			}
			// **كما يكتبه بابُ التحويل**: ملاحظةٌ بلا سائق، وعند المتجر رجوعٌ «في الطريق» بلا سائق.
			rdEvent(t, h, oid, status, status, "", transfer)
			if atPickup {
				rdEvent(t, h, oid, "at_pickup", "assigned", "", transfer)
			}
			rdEvent(t, h, oid, "assigned", "at_pickup", drv.ID, picked.Add(-30*time.Second))
			rdEvent(t, h, oid, "at_pickup", "picked_up", drv.ID, picked)
			rdEvent(t, h, oid, "at_dropoff", "delivered", drv.ID, delivered)

			got := rdFetch(t, h, admin.Token, drv.ID, day)
			if math.Abs(got.Summary.DriveKmToPickup-2.5) > 0.03 || math.Abs(got.Summary.DriveKmWithOrder-2.0) > 0.03 {
				t.Errorf("للمتجر = %.3f وللزبون = %.3f والمتوقَّع ٢٫٥ و٢ — التحويلُ قطع الولاية",
					got.Summary.DriveKmToPickup, got.Summary.DriveKmWithOrder)
			}
			if got.Summary.DriveKmNoOrder > 0.02 {
				t.Errorf("بلا طلب = %.3f والمتوقَّع ٠ — مشوارُ المتجر الجديد والزبون صار «بلا طلب»", got.Summary.DriveKmNoOrder)
			}
		})
	}
}

// TestDriverMonitor_StoreFailureEndsHold **«تعذّر» عند المتجر ينهي ولاية السائق** (Q4): قَبِل
// ومشى ٢ كم للمتجر، فأعادت العمليّاتُ الطلبَ للمكتب من بابها الحقيقيّ (`door-resolution` ⇒
// `orders/merchant_blocked.go`: `at_pickup → accepted` باسمه) ومشى ٣ كم بعدها، ثمّ نزل الطلب
// للطابور وسلّمه غيرُه ⇒ للسائق الأوّل ٢ كم «للمتجر» و٣ كم «بلا طلب».
func TestDriverMonitor_StoreFailureEndsHold(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	day, base := rdDayBase()

	oid, d1 := h.assignedOrder(t)
	d2 := h.NewUser("driver")
	rdEvent(t, h, oid, "dispatching", "assigned", d1.ID, base)
	if r := h.POST("/api/v1/driver/orders/"+oid+"/transition", d1.Token, map[string]any{"to": "at_pickup"}); r.Code >= 400 {
		t.Fatalf("وصل المتجر: %s", r)
	}
	if r := h.POST("/api/v1/admin/orders/"+oid+"/door-resolution", admin.Token, map[string]any{
		"action": "return_to_office", "fault": "merchant", "reason": "merchant_closed", "note": "المتجر مسكّر",
	}); r.Code >= 400 {
		t.Fatalf("تعذّر عند المتجر: %s", r)
	}
	var status string
	var holder *string
	var storeFailedBy *string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT o.status, o.driver_id::text,
		       (SELECT e.driver_id::text FROM order_events e
		         WHERE e.order_id = o.id AND e.from_status = 'at_pickup' AND e.to_status = 'accepted'
		         ORDER BY e.id DESC LIMIT 1)
		  FROM orders o WHERE o.id = $1::uuid`, oid).Scan(&status, &holder, &storeFailedBy); err != nil {
		t.Fatal(err)
	}
	if status != "accepted" || holder != nil || storeFailedBy == nil || *storeFailedBy != d1.ID {
		t.Fatalf("بعد «تعذّر»: حال=%s حامل=%v سطرُ السجلّ باسم=%v — المتوقَّع accepted بلا حامل وباسم السائق", status, holder, storeFailedBy)
	}

	start := rdPoint{at: base, lat: 35.68, lng: 38.77, acc: rdF(6), spd: rdF(5)}
	trail := append([]rdPoint{start}, rdDrive(start, 2000, 5, 10*time.Second)...)
	atStore := rdLast(trail).at
	failed := atStore.Add(time.Minute)
	last := rdLast(trail)
	trail = append(trail, rdPoint{at: failed, lat: last.lat, lng: last.lng, acc: rdF(6), spd: rdF(0)})
	trail = append(trail, rdDrive(rdLast(trail), 3000, 5, 10*time.Second)...)
	requeued := rdLast(trail).at.Add(5 * time.Minute)
	rdPut(t, h, d1.ID, trail)
	// **أزمنةُ ما كتبه البابان الحقيقيّان تُنقل إلى خطّ الفكسچر** — السطورُ نفسُها بحاملها.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE order_events SET created_at = CASE to_status WHEN 'at_pickup' THEN $2::timestamptz ELSE $3::timestamptz END
		 WHERE order_id = $1::uuid AND to_status IN ('at_pickup', 'accepted') AND from_status <> to_status`,
		oid, atStore, failed); err != nil {
		t.Fatal(err)
	}
	rdEvent(t, h, oid, "accepted", "dispatching", "", requeued)
	rdEvent(t, h, oid, "dispatching", "assigned", d2.ID, requeued.Add(time.Minute))
	rdEvent(t, h, oid, "at_dropoff", "delivered", d2.ID, requeued.Add(20*time.Minute))

	got := rdFetch(t, h, admin.Token, d1.ID, day)
	if math.Abs(got.Summary.DriveKmToPickup-2.0) > 0.03 {
		t.Errorf("للمتجر = %.3f كم والمتوقَّع ٢ — «تعذّر» ما أنهى الولاية", got.Summary.DriveKmToPickup)
	}
	if math.Abs(got.Summary.DriveKmNoOrder-3.0) > 0.03 {
		t.Errorf("بلا طلب = %.3f كم والمتوقَّع ٣ — ما بعد «تعذّر» انحسب على الطلب", got.Summary.DriveKmNoOrder)
	}
	for _, o := range got.Orders {
		if o.ID == oid && o.EndKind != "store_failed" {
			t.Errorf("نوع نهاية الطلب = %q والمتوقَّع store_failed", o.EndKind)
		}
	}
}
