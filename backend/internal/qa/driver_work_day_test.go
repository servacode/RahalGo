package qa

// ══════════════════════════════════════════════════════════════════════
// **«كيلومترات اليوم» تتبع يومَ المنصّة** — قرار المالك ٢٠٢٦-١١-٠٦
// ══════════════════════════════════════════════════════════════════════
//
// من فتح المنصّة (`platform.day_start_hour`) لإغلاقها من جدول الدوام — لا ٩←٩ ولا الدوام —
// وتبدأ من الصفر بالفتح التالي. وما مشاه بعد الإغلاق يُعرض وحده «مشي بعد إغلاق المنصة».
// **والمنصّةُ ٢٤ ساعة ⇒ يومُ الحساب ٩←٩ كما كان.**

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/platform"
)

// workDayHours **المنصّةُ من ٩ الصبح لـ١٠ الليل كلَّ يوم** والجدولُ يسري.
func workDayHours(t *testing.T, h *Harness) {
	t.Helper()
	var ws []platform.Window
	for d := 0; d < 7; d++ {
		ws = append(ws, platform.Window{Day: d, Start: 9 * 60, End: 22 * 60})
	}
	setHours(t, h, ws...)
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM platform_hours`) })
	h.Setting("hours.platform_enforced", "true")
	h.Setting("hours.platform_switch", `"schedule"`)
	h.Setting("platform.day_start_hour", "9")
}

// TestDriverMe_TodayDriveFollowsPlatformDay **عدّاد السائق**: ما انتهى بين الفتح والإغلاق يُحسب ·
// وما انتهى بعد الإغلاق لا · ويرجع صفراً بالفتح التالي · والمنصّةُ ٢٤ ساعة ⇒ ٩←٩ (يُحسب الكلّ).
func TestDriverMe_TodayDriveFollowsPlatformDay(t *testing.T) {
	h := New(t)
	workDayHours(t, h)
	f := h.Factory()
	drv := f.Driver()
	m := f.Merchant()
	cust := h.Customer()
	shift := odShift(t, h, drv.ID, lmAt(2031, 4, 7, 8, 0), nil)
	put := func(ended time.Time, toPickup, withOrder int) {
		var oid string
		if err := h.Pool.QueryRow(ctxBG(), `
			INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			                    payment_method, subtotal, delivery_fee, total)
			VALUES ($1::uuid, $2::uuid, $3::uuid, 'delivered', 'الرقة',
			        ST_SetSRID(ST_MakePoint(38.5110,35.5000),4326)::geography, 'cash', 10, 5, 15)
			RETURNING id::text`, cust.ID, m.ID, drv.ID).Scan(&oid); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM orders WHERE id = $1::uuid`, oid) })
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO order_drives (order_id, driver_id, shift_id, started_at, ended_at, end_kind, to_pickup_m, with_order_m)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $4, 'delivered', $5, $6)`,
			oid, drv.ID, shift, ended, toPickup, withOrder); err != nil {
			t.Fatal(err)
		}
	}
	put(lmAt(2031, 4, 7, 8, 30), 400, 0)     // قبل الفتح — أمس
	put(lmAt(2031, 4, 7, 10, 0), 1000, 2000) // بالدوام
	put(lmAt(2031, 4, 7, 22, 30), 500, 0)    // بعد إغلاق المنصّة

	at := func(now time.Time) int {
		t.Helper()
		m, _, err := h.API.TodayDrive(ctxBG(), drv.ID, now)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	if got := at(lmAt(2031, 4, 7, 23, 30)); got != 3000 {
		t.Fatalf("بعد الإغلاق: %d م — المتوقَّع ٣٠٠٠ (من الفتح للإغلاق، وما بعد العاشرة لا)", got)
	}
	if got := at(lmAt(2031, 4, 8, 9, 30)); got != 0 {
		t.Fatalf("بالفتح التالي: %d م — المتوقَّع صفر", got)
	}
	// **المنصّةُ ٢٤ ساعة** ⇒ يومُ الحساب ٩←٩: ما بعد العاشرة يُحسب
	h.Setting("hours.platform_switch", `"open"`)
	if got := at(lmAt(2031, 4, 7, 23, 30)); got != 3500 {
		t.Fatalf("المنصّة ٢٤ ساعة: %d م — المتوقَّع ٣٥٠٠ (٩←٩)", got)
	}
}

type workDayResp struct {
	Summary struct {
		DriveKmTotal float64 `json:"drive_km_total"`
	} `json:"summary"`
	WorkDay struct {
		CloseAt         *time.Time `json:"close_at"`
		AfterCloseKm    *float64   `json:"after_close_km"`
		AfterCloseAlert bool       `json:"after_close_alert"`
	} `json:"work_day"`
}

// TestDriverMonitor_DayFollowsPlatformDayAndFlagsAfterClose **صفحة المراقبة**: قيادةُ الدوام
// بأرقام اليوم، والقيادةُ بعد الإغلاق وحدها «مشي بعد إغلاق المنصة» بالأحمر (عتبةُ «مشى خارج
// الدوام») · وبالقائمة كمان · والمنصّةُ ٢٤ ساعة ⇒ الكلُّ باليوم ولا علم.
func TestDriverMonitor_DayFollowsPlatformDayAndFlagsAfterClose(t *testing.T) {
	h := New(t)
	workDayHours(t, h)
	h.Setting("drivers.odometer_off_alert_km", "1")
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	day, base := rdDayBase() // العاشرة صباحاً
	loc := platform.Location()
	start := rdPoint{at: base.Add(time.Hour), lat: 35.82, lng: 38.92, acc: rdF(6), spd: rdF(10)}
	leg1 := append([]rdPoint{start}, rdDrive(start, 5000, 10, 10*time.Second)...)
	night := time.Date(base.Year(), base.Month(), base.Day(), 23, 0, 0, 0, loc)
	p2 := rdPoint{at: night, lat: rdLast(leg1).lat, lng: 38.92, acc: rdF(6), spd: rdF(10)}
	leg2 := append([]rdPoint{p2}, rdDrive(p2, 3000, 10, 10*time.Second)...)
	rdPut(t, h, drv.ID, leg1)
	rdPut(t, h, drv.ID, leg2)

	fetch := func() workDayResp {
		t.Helper()
		res := h.GET(fmt.Sprintf("/api/v1/admin/driver-monitor/%s?day=%s", drv.ID, day), admin.Token)
		if res.Code != 200 {
			t.Fatalf("يوم السائق: %s", res)
		}
		var env struct {
			Data workDayResp `json:"data"`
		}
		if err := json.Unmarshal(res.Body, &env); err != nil {
			t.Fatal(err)
		}
		return env.Data
	}
	got := fetch()
	if math.Abs(got.Summary.DriveKmTotal-5) > 0.3 {
		t.Errorf("قيادة اليوم = %v كم — المتوقَّع ~٥ (لحد الإغلاق)", got.Summary.DriveKmTotal)
	}
	if got.WorkDay.CloseAt == nil || got.WorkDay.CloseAt.In(loc).Hour() != 22 {
		t.Errorf("الإغلاق = %v — المتوقَّع العاشرة ليلاً", got.WorkDay.CloseAt)
	}
	if got.WorkDay.AfterCloseKm == nil || math.Abs(*got.WorkDay.AfterCloseKm-3) > 0.3 || !got.WorkDay.AfterCloseAlert {
		t.Errorf("بعد الإغلاق = %v علم=%v — المتوقَّع ~٣ وأحمر", ptrStr(got.WorkDay.AfterCloseKm), got.WorkDay.AfterCloseAlert)
	}
	list := h.GET("/api/v1/admin/driver-monitor?day="+day, admin.Token)
	var lenv struct {
		Data struct {
			Drivers []struct {
				DriverID string `json:"driver_id"`
				WorkDay  struct {
					AfterCloseKm *float64 `json:"after_close_km"`
				} `json:"work_day"`
			} `json:"drivers"`
		} `json:"data"`
	}
	_ = json.Unmarshal(list.Body, &lenv)
	found := false
	for _, d := range lenv.Data.Drivers {
		if d.DriverID == drv.ID {
			found = true
			if d.WorkDay.AfterCloseKm == nil || math.Abs(*d.WorkDay.AfterCloseKm-3) > 0.3 {
				t.Errorf("القائمة: بعد الإغلاق = %v — المتوقَّع ~٣", ptrStr(d.WorkDay.AfterCloseKm))
			}
		}
	}
	if !found {
		t.Errorf("السائق مو بقائمة اليوم")
	}

	// **المنصّةُ ٢٤ ساعة** ⇒ ٩←٩ ولا «بعد الإغلاق»
	h.Setting("hours.platform_switch", `"open"`)
	got = fetch()
	if math.Abs(got.Summary.DriveKmTotal-8) > 0.5 || got.WorkDay.CloseAt != nil || got.WorkDay.AfterCloseKm != nil {
		t.Errorf("٢٤ ساعة: قيادة %v إغلاق %v بعد الإغلاق %v — المتوقَّع ~٨ بلا إغلاق", got.Summary.DriveKmTotal,
			got.WorkDay.CloseAt, ptrStr(got.WorkDay.AfterCloseKm))
	}
}
