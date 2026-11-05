package qa

// ══════════════════════════════════════════════════════════════════════
// **مراقبة حركة السائقين** — قرار المالك ٢٠٢٦-١١-٠٣
// ══════════════════════════════════════════════════════════════════════
//
// أثر يومي مخفَّف (`driver_trail`) + تحليل: كيلومترات بطلب وبلا طلب، وقفات عند
// متاجر بلا طلب، نمط «طلب من ورا ضهري»، انقطاع، وموقع مزيَّف. والأبواب بـ`drivers.monitor`.

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
)

// dmSpot موقع مسبوق بنقطة (lat,lng) — بقعة بعيدة عن بقية الاختبارات.
type dmSpot struct{ lat, lng float64 }

var (
	dmA = dmSpot{35.5000, 38.5000} // متجر أ
	dmC = dmSpot{35.5000, 38.5110} // الزبون ~١ كم شرق أ
	dmB = dmSpot{35.5135, 38.5110} // متجر ب ~١٫٥ كم شمال الزبون
	dmD = dmSpot{35.5135, 38.5250} // «عند الزبون الخفيّ» ~١٫٢٧ كم شرق ب
	dmP = dmSpot{35.5090, 38.5000} // بداية اليوم ~١ كم شمال أ
)

// dmTrail يبني أثرا بنقاط تُكتب في `driver_trail` مباشرة.
type dmTrail struct {
	t       *testing.T
	h       *Harness
	driver  string
	cur     time.Time
	pos     dmSpot
	written int
}

func (tr *dmTrail) put(at time.Time, p dmSpot, mocked bool) {
	tr.t.Helper()
	if _, err := tr.h.Pool.Exec(ctxBG(), `
		INSERT INTO driver_trail (driver_id, recorded_at, lat, lng, accuracy_m, speed_mps, mocked)
		VALUES ($1::uuid, $2, $3, $4, 5, $5, $6)
		ON CONFLICT DO NOTHING`, tr.driver, at, p.lat, p.lng, 5.0, mocked); err != nil {
		tr.t.Fatalf("كتابة نقطة الأثر: %v", err)
	}
	tr.written++
}

// walk يمشي بخط مستقيم إلى `to` خلال `dur` بنقطة كل `step`.
func (tr *dmTrail) walk(to dmSpot, dur, step time.Duration) {
	n := int(dur / step)
	from := tr.pos
	for i := 1; i <= n; i++ {
		f := float64(i) / float64(n)
		tr.put(tr.cur.Add(time.Duration(i)*step), dmSpot{from.lat + (to.lat-from.lat)*f, from.lng + (to.lng-from.lng)*f}, false)
	}
	tr.cur = tr.cur.Add(dur)
	tr.pos = to
}

// stand يقف مكانه `dur` بنبضة كل دقيقة (مع رجفة ~٣ م).
func (tr *dmTrail) stand(dur time.Duration) {
	for i := 1; i <= int(dur/time.Minute); i++ {
		j := 0.00003 * float64(i%2)
		tr.put(tr.cur.Add(time.Duration(i)*time.Minute), dmSpot{tr.pos.lat + j, tr.pos.lng}, false)
	}
	tr.cur = tr.cur.Add(dur)
}

// dmDay أمس بتوقيت دمشق عند الساعة ١٠:٠٠ — داخل يوم الحساب (٩ ← ٩) لتاريخ أمس.
func dmDay() (string, time.Time) {
	loc := drivermonitor.Loc()
	y := time.Now().In(loc).AddDate(0, 0, -1)
	base := time.Date(y.Year(), y.Month(), y.Day(), 10, 0, 0, 0, loc)
	return base.Format("2006-01-02"), base
}

// dmPlaceStore ينقل متجرا إلى موقع معيّن ويجعله الوحيد الفعّال بجواره.
func dmPlaceStore(t *testing.T, h *Harness, id string, p dmSpot) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants
		SET location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography WHERE id = $1::uuid`, id, p.lng, p.lat); err != nil {
		t.Fatalf("نقل المتجر: %v", err)
	}
}

// dmOrder طلب مسلَّم لسائق بأزمنة محددة (إسناد · استلام · تسليم).
func dmOrder(t *testing.T, h *Harness, driver, merchant string, assigned, picked, delivered time.Time) {
	t.Helper()
	cust := h.Customer()
	var oid string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
		                    payment_method, subtotal, delivery_fee, total,
		                    accepted_at, picked_up_at, delivered_at, closed_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'delivered', 'الرقة',
		        ST_SetSRID(ST_MakePoint(38.5110,35.5000),4326)::geography, 'cash', 10, 5, 15,
		        $4, $5, $6, $6)
		RETURNING id::text`, cust.ID, merchant, driver, assigned, picked, delivered).Scan(&oid); err != nil {
		t.Fatalf("إنشاء الطلب: %v", err)
	}
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM orders WHERE id = $1::uuid`, oid) })
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_events (order_id, from_status, to_status, driver_id, created_at)
		VALUES ($1::uuid, 'dispatching', 'assigned', $2::uuid, $3)`, oid, driver, assigned); err != nil {
		t.Fatalf("سطر الإسناد: %v", err)
	}
}

type dmDayResp struct {
	Alert   bool `json:"alert"`
	Summary struct {
		KmWithOrder float64 `json:"km_with_order"`
		KmToPickup  float64 `json:"km_to_pickup"`
		KmNoOrder   float64 `json:"km_no_order"`
		Patterns    int     `json:"patterns"`
		StoreStops  int     `json:"store_stops"`
		Gaps        int     `json:"gaps"`
		GapMinutes  int     `json:"gap_minutes"`
		Mocked      int     `json:"mocked"`
	} `json:"summary"`
	Path []struct {
		Class  string       `json:"class"`
		Coords [][2]float64 `json:"coords"`
	} `json:"path"`
	Events []struct {
		Kind     string `json:"kind"`
		Place    string `json:"place"`
		ToPlace  string `json:"to_place"`
		Minutes  int    `json:"minutes"`
		Returned bool   `json:"returned"`
		Flagged  bool   `json:"flagged"`
		Distance int    `json:"distance_m"`
		Count    int    `json:"count"`
	} `json:"events"`
}

func dmFetch(t *testing.T, h *Harness, admin, driver, day string) dmDayResp {
	t.Helper()
	res := h.GET(fmt.Sprintf("/api/v1/admin/driver-monitor/%s?day=%s", driver, day), admin)
	if res.Code != 200 {
		t.Fatalf("يوم السائق: %s", res)
	}
	var out dmDayResp
	var env struct {
		Data dmDayResp `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatalf("فك الردّ: %v", err)
	}
	out = env.Data
	return out
}

func within(v, lo, hi float64) bool { return v >= lo && v <= hi }

// dmBuildDay يبني اليوم الكامل: طلب حقيقي، ثم حركة بلا طلب، ثم «توصيلة من ورا الضهر» عند ب،
// ثم انقطاع وهو يمشي، ثم نقطتان مزيَّفتان.
//
// `withOrderCover=true` يضيف طلبا ثانيا يغطّي وقت الوقوف عند ب (للشاهد السلبي).
func dmBuildDay(t *testing.T, h *Harness, f *Factory, withCover bool) (driver, day string, storeB string) {
	t.Helper()
	day, base := dmDay()
	drv := f.Driver()
	mA, mB := f.Merchant(), f.Merchant()
	dmPlaceStore(t, h, mA.ID, dmA)
	dmPlaceStore(t, h, mB.ID, dmB)

	tr := &dmTrail{t: t, h: h, driver: drv.ID, cur: base, pos: dmP}
	tr.put(base, dmP, false)
	// **بسرعة درّاجة (~٤ م/ث)** — فالكيلومترات قيادة فعليّة لا زحف (قرار المالك ٢٠٢٦-١١-٠٥).
	// ١٠:٠٠ → ١٠:٠٤ إلى المتجر أ (طلب مسنَد منذ ١٠:٠٠)
	tr.walk(dmA, 4*time.Minute, 15*time.Second)
	// ١٠:٠٤ → ١٠:٠٧ استلام
	tr.stand(3 * time.Minute)
	// ١٠:٠٧ → ١٠:١١ إلى الزبون بالطلب
	tr.walk(dmC, 4*time.Minute, 15*time.Second)
	dmOrder(t, h, drv.ID, mA.ID, base, base.Add(4*time.Minute), base.Add(11*time.Minute))
	// ١٠:١١ → ١٠:١٧ إلى متجر ب بلا طلب
	tr.walk(dmB, 6*time.Minute, 15*time.Second)
	// ١٠:١٧ → ١٠:٢٣ يقف عند ب ست دقائق
	tr.stand(6 * time.Minute)
	// ١٠:٢٣ → ١٠:٢٨ إلى د (≥ ٨٠٠ م) ثم يقف ثلاث دقائق
	tr.walk(dmD, 5*time.Minute, 15*time.Second)
	tr.stand(3 * time.Minute)
	// ١٠:٣١ → ١٠:٣٦ يرجع إلى ب
	tr.walk(dmB, 5*time.Minute, 15*time.Second)
	if withCover {
		// طلب مسنَد قبل الوقوف عند ب ومسلَّم بعد العودة: كل ما سبق «بطلب».
		dmOrder(t, h, drv.ID, mB.ID, base.Add(12*time.Minute), base.Add(17*time.Minute), base.Add(40*time.Minute))
	}
	// انقطاع ٢٢ دقيقة وهو يمشي، ثم نقطتان مزيَّفتان
	g := tr.cur.Add(22 * time.Minute)
	tr.put(g, dmSpot{dmB.lat + 0.0002, dmB.lng}, true)
	tr.put(g.Add(30*time.Second), dmSpot{dmB.lat + 0.0003, dmB.lng}, true)
	return drv.ID, day, mB.Name
}

// TestDriverMonitor_ClassifiesDayAndCatchesBehindTheBackDelivery **اليوم الكامل**:
// كيلومترات كل صنف، النمط المشبوه عند ب، الانقطاع، المزيَّف — والمتجر الذي وقف عنده بالاسم.
func TestDriverMonitor_ClassifiesDayAndCatchesBehindTheBackDelivery(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	driver, day, storeB := dmBuildDay(t, h, h.Factory(), false)
	got := dmFetch(t, h, admin.Token, driver, day)

	s := got.Summary
	if !within(s.KmToPickup, 0.85, 1.15) {
		t.Errorf("كم إلى الاستلام = %.2f والمتوقَّع ~١", s.KmToPickup)
	}
	if !within(s.KmWithOrder, 0.85, 1.15) {
		t.Errorf("كم بالطلب = %.2f والمتوقَّع ~١", s.KmWithOrder)
	}
	// ١٫٥ إلى ب + ١٫٢٧ إلى د + ١٫٢٧ عودة ≈ ٤٫١
	if !within(s.KmNoOrder, 3.6, 4.7) {
		t.Errorf("كم بلا طلب = %.2f والمتوقَّع ~٤٫١", s.KmNoOrder)
	}
	if s.Patterns != 1 {
		t.Fatalf("أنماط «طلب من ورا ضهري» = %d والمتوقَّع ١", s.Patterns)
	}
	if s.StoreStops < 1 {
		t.Errorf("لا وقفة عند متجر بلا طلب")
	}
	if s.Gaps != 1 || s.GapMinutes < 21 {
		t.Errorf("الانقطاع: %d مرة %d دقيقة — والمتوقَّع ١ و٢٢", s.Gaps, s.GapMinutes)
	}
	if s.Mocked != 2 {
		t.Errorf("نقاط مزيَّفة = %d والمتوقَّع ٢", s.Mocked)
	}
	var sawPattern, sawGap, sawMocked bool
	for _, e := range got.Events {
		switch e.Kind {
		case "pattern":
			sawPattern = true
			if e.Place != storeB {
				t.Errorf("المتجر بالنمط = %q والمتوقَّع %q", e.Place, storeB)
			}
			if e.Distance < 800 || e.Distance > 1600 {
				t.Errorf("مسافة النمط = %d", e.Distance)
			}
			if !e.Returned {
				t.Errorf("النمط لم يلتقط العودة إلى المتجر")
			}
		case "gap":
			sawGap = true
			if !e.Flagged {
				t.Errorf("انقطاع وهو يمشي لم يُعلَّم")
			}
		case "mocked":
			sawMocked = true
			if e.Count != 2 {
				t.Errorf("عدد المزيَّف بالسطر = %d", e.Count)
			}
		}
	}
	if !sawPattern || !sawGap || !sawMocked {
		t.Errorf("أسطر ناقصة: نمط=%v انقطاع=%v مزيَّف=%v", sawPattern, sawGap, sawMocked)
	}
	// المسار ملوَّن بالأصناف الأربعة.
	classes := map[string]bool{}
	for _, p := range got.Path {
		classes[p.Class] = true
	}
	for _, c := range []string{"to_pickup", "with_order", "no_order", "stopped"} {
		if !classes[c] {
			t.Errorf("المسار بلا صنف %s", c)
		}
	}
}

// TestDriverMonitor_SameMovesWithAnOrderAreNotSuspicious **الشاهد السلبي**: نفس الحركة تماما
// ومعه طلب مسنَد وساري ⇒ لا نمط ولا وقفة مشبوهة عند المتجر، وتُحسب «بطلب».
func TestDriverMonitor_SameMovesWithAnOrderAreNotSuspicious(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	driver, day, _ := dmBuildDay(t, h, h.Factory(), true)
	got := dmFetch(t, h, admin.Token, driver, day)
	if got.Summary.Patterns != 0 || got.Summary.StoreStops != 0 {
		t.Fatalf("حركة بطلب سارٍ عُلِّمت مشبوهة: أنماط=%d وقفات=%d", got.Summary.Patterns, got.Summary.StoreStops)
	}
	if got.Summary.KmNoOrder > 2.0 {
		t.Errorf("كم بلا طلب = %.2f — أغلبها بطلب", got.Summary.KmNoOrder)
	}
}

// TestDriverMonitor_ListSortedWorstFirstAndAlert **القائمة**: الأخطر فوق، والعتبة تعلّم.
func TestDriverMonitor_ListSortedWorstFirstAndAlert(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	f := h.Factory()
	bad, day, _ := dmBuildDay(t, h, f, false)
	calm, _, _ := dmBuildDay(t, h, f, true)
	h.Setting("drivers.monitor_km_alert", "3")

	res := h.GET("/api/v1/admin/driver-monitor?day="+day, admin.Token)
	if res.Code != 200 {
		t.Fatalf("القائمة: %s", res)
	}
	var envL struct {
		Data struct {
			KmAlert float64 `json:"km_alert"`
			Drivers []struct {
				ID    string `json:"driver_id"`
				Alert bool   `json:"alert"`
			} `json:"drivers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &envL); err != nil {
		t.Fatal(err)
	}
	list := envL.Data
	if list.KmAlert != 3 {
		t.Errorf("العتبة = %v", list.KmAlert)
	}
	posBad, posCalm := -1, -1
	for i, d := range list.Drivers {
		if d.ID == bad {
			posBad = i
			if !d.Alert {
				t.Errorf("٤ كم بلا طلب فوق عتبة ٣ ولم يُعلَّم")
			}
		}
		if d.ID == calm {
			posCalm = i
		}
	}
	if posBad < 0 || posCalm < 0 || posBad > posCalm {
		t.Fatalf("الترتيب: المشبوه=%d والهادئ=%d — الأخطر أولا", posBad, posCalm)
	}
}

// TestDriverMonitor_RequiresCapability **الصلاحية**: بلا `drivers.monitor` ⇒ ٤٠٣، ومعها ⇒ ٢٠٠.
func TestDriverMonitor_RequiresCapability(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	for _, role := range []string{"customer", "driver", "operations", "finance", "customer_support"} {
		u := h.NewUser(role)
		if res := h.GET("/api/v1/admin/driver-monitor", u.Token); res.Code != 403 {
			t.Errorf("%s: القائمة = %d والمتوقَّع 403", role, res.Code)
		}
		if res := h.GET("/api/v1/admin/driver-monitor/"+drv.ID, u.Token); res.Code != 403 {
			t.Errorf("%s: اليوم = %d والمتوقَّع 403", role, res.Code)
		}
	}
	if res := h.GET("/api/v1/admin/driver-monitor", h.NewUser("admin").Token); res.Code != 200 {
		t.Errorf("admin: %s", res)
	}
	if res := h.GET("/api/v1/admin/driver-monitor?day=not-a-day", h.NewUser("admin").Token); res.Code != 400 {
		t.Errorf("يوم فاسد = %d والمتوقَّع 400", res.Code)
	}
}

// TestDriverMonitor_LocationHandlersKeepDownsampledTrail **الكتابة**: المعالجان (المفرد والدفعة)
// يخزّنان أثرا مخفَّفا — نقطة كل ≥٢٠ ث أو ≥٢٥ م، والمزيَّفة دائما.
func TestDriverMonitor_LocationHandlersKeepDownsampledTrail(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	t0 := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Second)
	pt := func(sec int, dlat float64, mocked bool) map[string]any {
		return map[string]any{"lat": 35.6000 + dlat, "lng": 38.6000,
			"at":        t0.Add(time.Duration(sec) * time.Second).Format(time.RFC3339),
			"speed_mps": 1.0, "accuracy_m": 5.0, "mocked": mocked}
	}
	batch := map[string]any{"points": []map[string]any{
		pt(0, 0, false),       // ✔ الأولى
		pt(5, 0, false),       // ✘ ٥ ث وصفر م
		pt(10, 0, false),      // ✘
		pt(20, 0, false),      // ✔ مرّ ٢٠ ث
		pt(25, 0, false),      // ✘
		pt(30, 0.0004, false), // ✔ تحرّك ~٤٤ م
		pt(32, 0.0004, false), // ✘
		pt(35, 0.0004, true),  // ✔ مزيَّفة دائما
	}}
	if res := h.POST("/api/v1/driver/location/batch", drv.Token, batch); res.Code != 200 {
		t.Fatalf("الدفعة: %s", res)
	}
	count := func() int {
		var n int
		if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*)::int FROM driver_trail WHERE driver_id = $1::uuid`, drv.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := count(); n != 4 {
		t.Fatalf("نقاط الأثر بعد الدفعة = %d والمتوقَّع ٤", n)
	}
	// إعادة الدفعة نفسها (انقطع الردّ) لا تكرّر.
	if res := h.POST("/api/v1/driver/location/batch", drv.Token, batch); res.Code != 200 {
		t.Fatalf("إعادة الدفعة: %s", res)
	}
	if n := count(); n != 4 {
		t.Fatalf("بعد إعادة الدفعة = %d — تكرّرت", n)
	}
	// النداء المفرد: بعد دقيقة ⇒ تُحفظ، وبعد ثانيتين بنفس المكان ⇒ لا.
	one := func(sec int) {
		if res := h.POST("/api/v1/driver/location", drv.Token, map[string]any{
			"lat": 35.6004, "lng": 38.6000, "speed_mps": 0.0, "accuracy_m": 5.0,
			"recorded_at": t0.Add(time.Duration(sec) * time.Second).Format(time.RFC3339),
		}); res.Code != 200 {
			t.Fatalf("النداء المفرد: %s", res)
		}
	}
	one(95)
	one(97)
	if n := count(); n != 5 {
		t.Fatalf("بعد المفرد = %d والمتوقَّع ٥", n)
	}
	// وأثر التوزيع القصير (`driver_track`) ما زال يُكتب كما كان.
	var tr int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*)::int FROM driver_track WHERE driver_id = $1::uuid`, drv.ID).Scan(&tr); err != nil || tr == 0 {
		t.Fatalf("driver_track توقّف: %d %v", tr, err)
	}
}

// TestDriverMonitor_RetentionPrunes **التقليم**: ما جاوز المدة يُحذف وما دونها يبقى.
func TestDriverMonitor_RetentionPrunes(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	for _, d := range []int{40, 31, 10, 0} {
		at := time.Now().Add(-time.Duration(d)*24*time.Hour - time.Hour)
		if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO driver_trail (driver_id, recorded_at, lat, lng)
			VALUES ($1::uuid, $2, 35.7, 38.7)`, drv.ID, at); err != nil {
			t.Fatal(err)
		}
	}
	deleted, err := drivermonitor.Prune(ctxBG(), h.Pool, 30)
	if err != nil || deleted < 2 {
		t.Fatalf("Prune: حُذف %d خطأ %v — والمتوقَّع ≥٢", deleted, err)
	}
	var left int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*)::int FROM driver_trail WHERE driver_id = $1::uuid`, drv.ID).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 2 {
		t.Fatalf("بقي %d والمتوقَّع ٢ (١٠ أيام واليوم)", left)
	}
}
