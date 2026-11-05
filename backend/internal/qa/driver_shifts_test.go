package qa

// ══════════════════════════════════════════════════════════════════════
// **سجلّ الدوام ومسافة كل طلب** — قرارات المالك ٢٠٢٦-١١-٠٥
// ══════════════════════════════════════════════════════════════════════
//
// كل فتح وسكّر للدوام صفّ بسببه (السائق · الخروج · الترك · الطارئ · الحادث · العمليات ·
// إيقاف الحساب)، وكانس المسافات يكتب مسافة كل طلب للمتجر وللزبون، والسائق يرى
// «مسافة طلباتك اليوم» وسجلّه بمسافة كل طلب، والمكتب يراها على ملف الطلب.

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

// shiftRows صفوف دوام السائق: (مفتوح؟، السبب) بالترتيب.
func shiftRows(t *testing.T, h *Harness, driver string) []struct {
	open   bool
	reason string
	build  *int
} {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT closed_at IS NULL, COALESCE(close_reason, ''), opened_build
		  FROM driver_shifts WHERE driver_id = $1::uuid ORDER BY opened_at, closed_at NULLS LAST`, driver)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []struct {
		open   bool
		reason string
		build  *int
	}
	for rows.Next() {
		var r struct {
			open   bool
			reason string
			build  *int
		}
		if err := rows.Scan(&r.open, &r.reason, &r.build); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

func onShiftOf(t *testing.T, h *Harness, driver string) bool {
	t.Helper()
	var on bool
	if err := h.Pool.QueryRow(ctxBG(), `SELECT on_shift FROM users WHERE id = $1::uuid`, driver).Scan(&on); err != nil {
		t.Fatal(err)
	}
	return on
}

// driverApp ترويستا تطبيق السائق بنسخةٍ معلومة.
func driverApp(build string) map[string]string {
	return map[string]string{"X-RahalGo-Client": "android-driver", "X-RahalGo-Version": build}
}

// TestShifts_ToggleOpensAndClosesRow **المفتاح**: فتحٌ ثم فتحٌ ثانٍ ثم سكّر ⇒ صفٌّ واحد سببه
// «السائق»، ونسخة التطبيق مكتوبة. والخروج من التطبيق سببه «الخروج».
func TestShifts_ToggleOpensAndClosesRow(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	drv := h.NewUser("driver")
	for i := 0; i < 2; i++ {
		if r := h.Call("POST", "/api/v1/driver/shift", drv.Token, map[string]any{"on": true}, driverApp("21")); r.Code != 200 {
			t.Fatalf("فتح الدوام %d: %s", i+1, r)
		}
	}
	got := shiftRows(t, h, drv.ID)
	if len(got) != 1 || !got[0].open || got[0].build == nil || *got[0].build != 21 {
		t.Fatalf("بعد فتحين: %+v — المتوقَّع صفٌّ مفتوح واحد بنسخة ٢١", got)
	}
	if r := h.POST("/api/v1/driver/shift", drv.Token, map[string]any{"on": false}); r.Code != 200 {
		t.Fatalf("سكّر الدوام: %s", r)
	}
	got = shiftRows(t, h, drv.ID)
	if len(got) != 1 || got[0].open || got[0].reason != "driver" {
		t.Fatalf("بعد السكّر: %+v — المتوقَّع صفٌّ مسكَّر سببه driver", got)
	}
	if onShiftOf(t, h, drv.ID) {
		t.Fatal("on_shift بقي صحيحاً بعد السكّر")
	}
	// الخروج من التطبيق
	if r := h.POST("/api/v1/driver/shift", drv.Token, map[string]any{"on": true}); r.Code != 200 {
		t.Fatal(r)
	}
	if r := h.POST("/api/v1/driver/shift", drv.Token, map[string]any{"on": false, "via": "logout"}); r.Code != 200 {
		t.Fatal(r)
	}
	got = shiftRows(t, h, drv.ID)
	if len(got) != 2 || got[1].reason != "logout" {
		t.Fatalf("الخروج: %+v — المتوقَّع صفٌّ ثانٍ سببه logout", got)
	}
}

// openShiftFor يفتح دوام السائق من بابه.
func openShiftFor(t *testing.T, h *Harness, drv *User) {
	t.Helper()
	if r := h.POST("/api/v1/driver/shift", drv.Token, map[string]any{"on": true}); r.Code != 200 {
		t.Fatalf("فتح الدوام: %s", r)
	}
}

func lastReason(t *testing.T, h *Harness, driver string) (bool, string) {
	t.Helper()
	got := shiftRows(t, h, driver)
	if len(got) == 0 {
		t.Fatalf("لا صفّ دوام للسائق %s", driver)
	}
	return got[len(got)-1].open, got[len(got)-1].reason
}

// TestShifts_SystemPathsCloseWithReason **كل مسار يسكّر الدوام بلا المفتاح يُكتب بسببه**:
// الترك · الترك بحادث · الطارئ · العمليات · إيقاف الحساب — **والترك الذي لم يقع يُرجع الدوام**.
func TestShifts_SystemPathsCloseWithReason(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")

	t.Run("release", func(t *testing.T) {
		oid, drv := h.assignedOrder(t)
		openShiftFor(t, h, drv)
		if r := h.POST("/api/v1/driver/orders/"+oid+"/release", drv.Token,
			map[string]any{"reason": "bike_broken", "note": "انقطع الجنزير"}); r.Code >= 400 {
			t.Fatalf("الترك: %s", r)
		}
		if open, reason := lastReason(t, h, drv.ID); open || reason != "release" {
			t.Fatalf("بعد الترك: مفتوح=%v سبب=%q والمتوقَّع release", open, reason)
		}
	})
	t.Run("release_accident", func(t *testing.T) {
		oid, drv := h.assignedOrder(t)
		openShiftFor(t, h, drv)
		if r := h.POST("/api/v1/driver/orders/"+oid+"/release", drv.Token,
			map[string]any{"reason": "accident", "note": "صدمة خفيفة"}); r.Code >= 400 {
			t.Fatalf("الترك بحادث: %s", r)
		}
		if open, reason := lastReason(t, h, drv.ID); open || reason != "accident" {
			t.Fatalf("بعد الترك بحادث: مفتوح=%v سبب=%q والمتوقَّع accident", open, reason)
		}
	})
	t.Run("emergency", func(t *testing.T) {
		oid, drv := h.assignedOrder(t)
		openShiftFor(t, h, drv)
		if r := h.POST("/api/v1/driver/orders/"+oid+"/emergency", drv.Token,
			map[string]any{"note": "عطل مفاجئ", "kind": "breakdown"}); r.Code >= 400 {
			t.Fatalf("الطارئ: %s", r)
		}
		if open, reason := lastReason(t, h, drv.ID); open || reason != "emergency" {
			t.Fatalf("بعد الطارئ: مفتوح=%v سبب=%q والمتوقَّع emergency", open, reason)
		}
	})
	t.Run("ops", func(t *testing.T) {
		drv := h.NewUser("driver")
		openShiftFor(t, h, drv)
		admin := h.NewUser("admin")
		if r := h.POST("/api/v1/admin/drivers/"+drv.ID+"/end-shift", admin.Token, map[string]any{"note": "انتهى وقته"}); r.Code != 200 {
			t.Fatalf("إنهاء العمليات: %s", r)
		}
		if open, reason := lastReason(t, h, drv.ID); open || reason != "ops" {
			t.Fatalf("بعد إنهاء العمليات: مفتوح=%v سبب=%q والمتوقَّع ops", open, reason)
		}
		// ومن ليس بدوامه: ٤٠٤ كما كان
		if r := h.POST("/api/v1/admin/drivers/"+drv.ID+"/end-shift", admin.Token, map[string]any{}); r.Code != 404 {
			t.Fatalf("إنهاءُ دوامٍ مسكَّر = %d والمتوقَّع 404", r.Code)
		}
	})
	t.Run("account", func(t *testing.T) {
		drv := h.NewUser("driver")
		openShiftFor(t, h, drv)
		admin := h.NewUser("admin")
		if r := h.PATCH("/api/v1/admin/users/"+drv.ID, admin.Token,
			map[string]any{"status": "blocked", "status_reason": "احتيال"}); r.Code != 200 {
			t.Fatalf("الحظر: %s", r)
		}
		if open, reason := lastReason(t, h, drv.ID); open || reason != "account" {
			t.Fatalf("بعد الحظر: مفتوح=%v سبب=%q والمتوقَّع account", open, reason)
		}
		if onShiftOf(t, h, drv.ID) {
			t.Fatal("المحظور بقي «بالدوام»")
		}
	})
	t.Run("failed_release_keeps_shift", func(t *testing.T) {
		oid, drv := h.assignedOrder(t)
		openShiftFor(t, h, drv)
		var started time.Time
		if err := h.Pool.QueryRow(ctxBG(), `SELECT shift_started_at FROM users WHERE id = $1::uuid`, drv.ID).Scan(&started); err != nil {
			t.Fatal(err)
		}
		// **الانتقالُ يسقط** — سطرُ السجلّ لا يُكتب.
		h.Arm("release-fails", "order_events", "INSERT", 1, "order_id", oid)
		if r := h.POST("/api/v1/driver/orders/"+oid+"/release", drv.Token,
			map[string]any{"reason": "bike_broken", "note": "انقطع الجنزير"}); r.Code < 400 {
			t.Fatalf("الترك نجح والنقطة مسلَّحة: %s", r)
		}
		if open, _ := lastReason(t, h, drv.ID); !open {
			t.Fatal("تركٌ لم يقع سكّر الدوام")
		}
		if n := len(shiftRows(t, h, drv.ID)); n != 1 {
			t.Fatalf("صفوف الدوام = %d والمتوقَّع ١ (الصفّ نفسه رجع مفتوحاً)", n)
		}
		var on bool
		var back time.Time
		if err := h.Pool.QueryRow(ctxBG(), `SELECT on_shift, shift_started_at FROM users WHERE id = $1::uuid`, drv.ID).Scan(&on, &back); err != nil {
			t.Fatal(err)
		}
		if !on || !back.Equal(started) {
			t.Fatalf("بعد الترك الفاشل: on=%v بداية=%v والمتوقَّع true و%v", on, back, started)
		}
	})
}

// odOrder طلبٌ مسلَّم لسائق بسجلّ إسنادٍ واستلامٍ وتسليم — ويرجع زبونه.
func odOrder(t *testing.T, h *Harness, driver, merchant string, assigned, picked, delivered time.Time) (string, *User) {
	t.Helper()
	cust := h.Customer()
	var oid string
	// **ولقطةُ الاقتصاد معه** — ملفُّ الطلب للمكتب يقرأ مالَه منها (`economics_snapshot.go`).
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
		                    payment_method, subtotal, delivery_fee, total,
		                    accepted_at, picked_up_at, delivered_at, closed_at,
		                    snap_merchant_commission_percent, snap_rep_commission_percent,
		                    snap_commission_source, snap_activation_orders)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'delivered', 'الرقة',
		        ST_SetSRID(ST_MakePoint(38.5110,35.5000),4326)::geography, 'cash', 10, 5, 15,
		        $4, $5, $6, $6, 0, 0, 'platform_commission', 0)
		RETURNING id::text`, cust.ID, merchant, driver, assigned, picked, delivered).Scan(&oid); err != nil {
		t.Fatalf("إنشاء الطلب: %v", err)
	}
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM orders WHERE id = $1::uuid`, oid) })
	rdEvent(t, h, oid, "dispatching", "assigned", driver, assigned)
	rdEvent(t, h, oid, "at_pickup", "picked_up", driver, picked)
	rdEvent(t, h, oid, "at_dropoff", "delivered", driver, delivered)
	return oid, cust
}

// odShift صفُّ دوام بحدّين (والفارغ مفتوح).
func odShift(t *testing.T, h *Harness, driver string, opened time.Time, closed *time.Time) string {
	t.Helper()
	var id string
	var reason any
	if closed != nil {
		reason = "driver"
	}
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO driver_shifts (driver_id, opened_at, closed_at, close_reason)
		VALUES ($1::uuid, $2, $3, $4) RETURNING id::text`, driver, opened, closed, reason).Scan(&id); err != nil {
		t.Fatalf("صفّ الدوام: %v", err)
	}
	return id
}

// sweepUntil يكنس حتى يُحسب الدوام (قد يكون في الجولة دواماتٌ لغيره).
func sweepUntil(t *testing.T, h *Harness, shift string, now time.Time) {
	t.Helper()
	for i := 0; i < 6; i++ {
		h.API.SweepDrives(context.Background(), now)
		var at *time.Time
		if err := h.Pool.QueryRow(ctxBG(), `SELECT computed_at FROM driver_shifts WHERE id = $1::uuid`, shift).Scan(&at); err != nil {
			t.Fatal(err)
		}
		if at != nil && !at.Before(now.Add(-time.Second)) {
			return
		}
	}
	t.Fatalf("الكانس ما حسب الدوام %s", shift)
}

func driveRow(t *testing.T, h *Harness, order, driver string) (toPickup, withOrder int, final bool, ended *time.Time) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT to_pickup_m, with_order_m, final, ended_at FROM order_drives
		 WHERE order_id = $1::uuid AND driver_id = $2::uuid`, order, driver).Scan(&toPickup, &withOrder, &final, &ended); err != nil {
		t.Fatalf("صفّ مسافة الطلب: %v", err)
	}
	return
}

// TestOrderDrive_StoredAndShown **مسافة الطلب تُكتب وتُعرض**: الكانس يكتب ١٠٠٠ م للمتجر
// و١٥٠٠ للزبون ⇒ «مسافة طلباتك اليوم» ٢٥٠٠ · سطر السجلّ · ملف الطلب للمكتب — **ولا يراها الزبون**.
func TestOrderDrive_StoredAndShown(t *testing.T) {
	h := New(t)
	f := h.Factory()
	drv := f.Driver()
	m := f.Merchant()
	admin := h.NewUser("admin")

	// **الطلب ينتهي قبل دقيقة** — فيقع في يوم الحساب أيّاً كانت الساعة.
	now := time.Now()
	delivered := now.Add(-time.Minute).Truncate(time.Second)
	picked := delivered.Add(-300 * time.Second)
	atStore := picked.Add(-3 * time.Minute)
	assigned := atStore.Add(-200 * time.Second)
	start := rdPoint{at: assigned, lat: 35.78, lng: 38.88, acc: rdF(6), spd: rdF(5)}
	trail := append([]rdPoint{start}, rdDrive(start, 1000, 5, 10*time.Second)...)
	for i := 1; i <= 3; i++ {
		p := rdLast(trail)
		trail = append(trail, rdPoint{at: p.at.Add(time.Minute), lat: p.lat, lng: p.lng, acc: rdF(6), spd: rdF(0)})
	}
	trail = append(trail, rdDrive(rdLast(trail), 1500, 5, 10*time.Second)...)
	rdPut(t, h, drv.ID, trail)
	oid, cust := odOrder(t, h, drv.ID, m.ID, assigned, picked, delivered)
	shift := odShift(t, h, drv.ID, assigned.Add(-2*time.Minute), nil)

	sweepUntil(t, h, shift, now)
	tp, wo, final, ended := driveRow(t, h, oid, drv.ID)
	if math.Abs(float64(tp)-1000) > 20 || math.Abs(float64(wo)-1500) > 20 || ended == nil || final {
		t.Fatalf("صفّ المسافة: للمتجر=%d للزبون=%d نهائي=%v انتهى=%v — المتوقَّع ~١٠٠٠/~١٥٠٠ غير نهائي", tp, wo, final, ended)
	}
	var shiftOrders, shiftDrive int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT orders_m, drive_m FROM driver_shifts WHERE id = $1::uuid`, shift).Scan(&shiftOrders, &shiftDrive); err != nil {
		t.Fatal(err)
	}
	if math.Abs(float64(shiftOrders-(tp+wo))) > 2 || shiftDrive < shiftOrders {
		t.Fatalf("أرقام الدوام: بطلب=%d قيادة=%d ومجموع الطلب %d", shiftOrders, shiftDrive, tp+wo)
	}

	// ── السائق: «مسافة طلباتك اليوم» ──
	me := h.GET("/api/v1/driver/me", drv.Token)
	if me.Code != 200 {
		t.Fatal(me)
	}
	if got, _ := me.JSON()["today_drive_m"].(float64); math.Abs(got-float64(tp+wo)) > 0.5 {
		t.Fatalf("today_drive_m = %v والمتوقَّع %d", got, tp+wo)
	}
	// ── سجلّه ──
	hist := h.GET("/api/v1/driver/orders/history", drv.Token)
	var found bool
	rows, _ := hist.JSON()["orders"].([]any)
	for _, x := range rows {
		row, _ := x.(map[string]any)
		if row["id"] == oid {
			found = true
			if d, _ := row["drive_m"].(float64); int(d) != tp+wo {
				t.Errorf("drive_m بالسجلّ = %v والمتوقَّع %d", row["drive_m"], tp+wo)
			}
		}
	}
	if !found {
		t.Fatalf("الطلب ما ظهر بسجلّ السائق: %s", hist)
	}
	// ── ملف الطلب للمكتب ──
	file := h.GET("/api/v1/admin/orders/"+oid, admin.Token)
	var env struct {
		Data struct {
			Drive []struct {
				DriverID   string `json:"driver_id"`
				ToPickupM  int    `json:"to_pickup_m"`
				WithOrderM int    `json:"with_order_m"`
			} `json:"drive"`
		} `json:"data"`
	}
	if err := json.Unmarshal(file.Body, &env); err != nil || file.Code != 200 {
		t.Fatalf("ملف الطلب: %s", file)
	}
	if len(env.Data.Drive) != 1 || env.Data.Drive[0].DriverID != drv.ID || env.Data.Drive[0].ToPickupM != tp {
		t.Fatalf("ملف الطلب drive = %+v", env.Data.Drive)
	}
	list := h.GET("/api/v1/admin/orders?closed=1&driver_id="+drv.ID, admin.Token)
	if !strings.Contains(string(list.Body), `"drive":[`) {
		t.Errorf("بطاقة السجلّ بلا مسافة: %s", list)
	}
	// ── والزبون لا يراها ──
	mine := h.GET("/api/v1/my/orders/"+oid, cust.Token)
	if mine.Code != 200 {
		t.Fatalf("طلب الزبون: %s", mine)
	}
	if strings.Contains(string(mine.Body), `"drive"`) || strings.Contains(string(mine.Body), "to_pickup_m") {
		t.Fatalf("مسافة السائق وصلت الزبون: %s", mine.Body)
	}
}

// TestOrderDrive_LatePointsThenFrozen **نقاطٌ متأخّرة تدخل، ثم يثبت الرقم**: دفعةٌ متأخّرة
// بعد السكّر بعشرين دقيقة تغيّر المسافة؛ وبعد ساعتين وعشر دقائق يثبت ولا يتغيّر.
func TestOrderDrive_LatePointsThenFrozen(t *testing.T) {
	h := New(t)
	f := h.Factory()
	drv := f.Driver()
	m := f.Merchant()
	base := time.Now().Add(-6 * time.Hour).Truncate(time.Second)
	start := rdPoint{at: base, lat: 35.80, lng: 38.90, acc: rdF(6), spd: rdF(5)}
	leg1 := append([]rdPoint{start}, rdDrive(start, 1000, 5, 10*time.Second)...) // للمتجر
	picked := rdLast(leg1).at
	leg2 := rdDrive(rdLast(leg1), 1500, 5, 10*time.Second) // للزبون
	delivered := rdLast(leg2).at
	rdPut(t, h, drv.ID, leg1)
	rdPut(t, h, drv.ID, leg2[:10]) // **وصل نصفُ الطريق للزبون فقط**
	oid, _ := odOrder(t, h, drv.ID, m.ID, base, picked, delivered)
	closed := delivered.Add(5 * time.Minute)
	shift := odShift(t, h, drv.ID, base.Add(-time.Minute), &closed)

	sweepUntil(t, h, shift, closed.Add(time.Minute))
	_, wo1, final1, _ := driveRow(t, h, oid, drv.ID)
	if final1 || math.Abs(float64(wo1)-500) > 20 {
		t.Fatalf("أوّل حساب: للزبون=%d نهائي=%v والمتوقَّع ~٥٠٠ غير نهائي", wo1, final1)
	}
	// **دفعةُ ما جُمع بلا شبكة** تصل متأخّرة
	rdPut(t, h, drv.ID, leg2[10:])
	sweepUntil(t, h, shift, closed.Add(20*time.Minute))
	_, wo2, final2, _ := driveRow(t, h, oid, drv.ID)
	if final2 || math.Abs(float64(wo2)-1500) > 20 {
		t.Fatalf("بعد النقاط المتأخّرة: للزبون=%d نهائي=%v والمتوقَّع ~١٥٠٠ غير نهائي", wo2, final2)
	}
	// **ويثبت** بعد ساعتين وعشر دقائق
	sweepUntil(t, h, shift, closed.Add(2*time.Hour+11*time.Minute))
	_, wo3, final3, _ := driveRow(t, h, oid, drv.ID)
	if !final3 || wo3 != wo2 {
		t.Fatalf("بعد ساعتين وعشر: للزبون=%d نهائي=%v والمتوقَّع %d نهائي", wo3, final3, wo2)
	}
	// **وبعد الثبات لا يتغيّر شيء** ولو وصلت نقاط
	rdPut(t, h, drv.ID, rdDrive(rdLast(leg2), 2000, 5, 10*time.Second))
	h.API.SweepDrives(context.Background(), closed.Add(3*time.Hour))
	if _, wo4, _, _ := driveRow(t, h, oid, drv.ID); wo4 != wo3 {
		t.Fatalf("تغيّر الرقم بعد الثبات: %d → %d", wo3, wo4)
	}
}

// TestDriverMe_TodayDriveFollowsAccountDay **يومُ العدّاد يومُ الحساب**: طلبٌ انتهى قبل ساعة
// فتح الدوام اليوم يُحسب على أمس، وما انتهى بعدها (أو ما زال معه) على اليوم.
func TestDriverMe_TodayDriveFollowsAccountDay(t *testing.T) {
	h := New(t)
	f := h.Factory()
	drv := f.Driver()
	m := f.Merchant()
	var dayStart time.Time
	if err := h.Pool.QueryRow(ctxBG(), `SELECT account_day_start(account_day(now()))`).Scan(&dayStart); err != nil {
		t.Fatal(err)
	}
	put := func(ended *time.Time, toPickup, withOrder int) {
		cust := h.Customer()
		var oid string
		if err := h.Pool.QueryRow(ctxBG(), `
			INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			                    payment_method, subtotal, delivery_fee, total)
			VALUES ($1::uuid, $2::uuid, $3::uuid, 'assigned', 'الرقة',
			        ST_SetSRID(ST_MakePoint(38.5110,35.5000),4326)::geography, 'cash', 10, 5, 15)
			RETURNING id::text`, cust.ID, m.ID, drv.ID).Scan(&oid); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM orders WHERE id = $1::uuid`, oid) })
		started := dayStart.Add(-3 * time.Hour)
		kind := "delivered"
		if ended == nil {
			kind = "open"
		}
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO order_drives (order_id, driver_id, started_at, ended_at, end_kind, to_pickup_m, with_order_m)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)`, oid, drv.ID, started, ended, kind, toPickup, withOrder); err != nil {
			t.Fatal(err)
		}
	}
	yesterday := dayStart.Add(-30 * time.Minute)
	today := dayStart.Add(time.Minute)
	put(&yesterday, 700, 800) // أمس — لا يُحسب
	put(&today, 1000, 2000)   // اليوم
	put(nil, 300, 0)          // لسّا معه — «لحد هلق»
	me := h.GET("/api/v1/driver/me", drv.Token)
	if me.Code != 200 {
		t.Fatal(me)
	}
	if got, _ := me.JSON()["today_drive_m"].(float64); got != 3300 {
		t.Fatalf("today_drive_m = %v والمتوقَّع ٣٣٠٠ (اليوم ٣٠٠٠ + الجاري ٣٠٠، وأمس لا)", got)
	}
	if n, _ := me.JSON()["today_drive_orders"].(float64); n != 2 {
		t.Fatalf("today_drive_orders = %v والمتوقَّع ٢", n)
	}
}
