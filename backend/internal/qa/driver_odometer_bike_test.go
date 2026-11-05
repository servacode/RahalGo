package qa

// ══════════════════════════════════════════════════════════════════════
// **«بدّلت الدرّاجة / العدّاد»** — قرار المالك ٢٠٢٦-١١-٠٦
// ══════════════════════════════════════════════════════════════════════
//
// الرقمُ الأقلّ يُقبل **بالعلامة الصريحة وحدها** (`bike_change`)، ويُعلَّم، ويُبلَّغ من يراقب،
// ويُكتب سطرٌ بالسجلّ — **وتبديلٌ واحدٌ باليوم**، وبلا العلامة (التطبيقُ القديم) الرفضُ كما كان.
// **ولا مسافةَ تُحسب عبر التبديل** بالمراقبة.

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestOdometer_BikeChangeAcceptsLowerOncePerDay **التبديل**: أقلّ بلا علامة ⇒ ٤٠٩ ومعه
// «يقدر يبدّل» · بالعلامة ⇒ ٢٠١ معلَّمة وآخرُ رقمٍ قبلها محفوظ وسطرُ سجلّ وإشعارٌ لمن يراقب ·
// والقراءةُ التالية تُقاس بالعدّاد الجديد · وتبديلٌ ثانٍ باليوم ⇒ الرفضُ العاديّ بلا «يقدر يبدّل».
func TestOdometer_BikeChangeAcceptsLowerOncePerDay(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	h.Setting("drivers.odometer_max_jump_km", "150")
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	start := odoReading(t, h, drv, "start", "1250")
	if r := odoShift(t, h, drv, true, start, true); r.Code != 200 {
		t.Fatalf("فتح الدوام: %s", r)
	}

	// ── بلا علامة: الرفضُ كما كان، والتفصيلُ يقول إنّ التبديل متاح ──
	low := odoUpload(t, h, drv.Token, map[string]string{"phase": "end", "km": "400"}, true, driverApp("22"))
	if low.Code != 409 || low.Err() != "odometer_lower" {
		t.Fatalf("أقلّ بلا علامة: %s — المتوقَّع 409 odometer_lower", low)
	}
	if d := odoErrDetails(low); d["bike_change_left"] != "true" {
		t.Errorf("bike_change_left = %v والمتوقَّع true", d["bike_change_left"])
	}

	// ── بالعلامة: يُقبل ويُعلَّم ──
	ok := odoUpload(t, h, drv.Token, map[string]string{"phase": "end", "km": "400", "bike_change": "true"}, true, driverApp("22"))
	if ok.Code != http.StatusCreated || ok.JSON()["bike_change"] != true {
		t.Fatalf("تبديل الدرّاجة: %s — المتوقَّع 201 و bike_change", ok)
	}
	id, _ := ok.JSON()["id"].(string)
	var bike bool
	var prev float64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT bike_change, prev_km::float8 FROM driver_odometer_readings
		WHERE id = $1::uuid`, id).Scan(&bike, &prev); err != nil {
		t.Fatal(err)
	}
	if !bike || prev != 1250 {
		t.Errorf("الصفّ: bike_change=%v prev_km=%v — المتوقَّع true و 1250", bike, prev)
	}
	var audits int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log
		WHERE action = 'driver.odometer_bike_change' AND entity_id = $1`, drv.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 1 {
		t.Errorf("أسطر السجلّ = %d والمتوقَّع 1", audits)
	}
	var notes int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2 AND href LIKE '/dashboard/driver-monitor%'`,
		admin.ID, drv.ID).Scan(&notes); err != nil {
		t.Fatal(err)
	}
	if notes != 1 {
		t.Errorf("إشعارات المراقب = %d والمتوقَّع 1", notes)
	}

	// ── العدّادُ الجديد يحكم: 410 تمرّ لا «قفزة» ولا «أقلّ» ──
	if r := odoShift(t, h, drv, false, id, true); r.Code != 200 {
		t.Fatalf("سكّر الدوام بقراءة التبديل: %s", r)
	}
	next := odoUpload(t, h, drv.Token, map[string]string{"phase": "start", "km": "410"}, true, driverApp("22"))
	if next.Code != 201 || next.JSON()["check"] != "ok" {
		t.Fatalf("قراءة بعد التبديل: %s", next)
	}

	// ── تبديلٌ ثانٍ باليوم: الرفضُ العاديّ ──
	again := odoUpload(t, h, drv.Token, map[string]string{"phase": "start", "km": "5", "bike_change": "true"}, true, driverApp("22"))
	if again.Code != 409 || again.Err() != "odometer_lower" {
		t.Fatalf("تبديلٌ ثانٍ باليوم: %s — المتوقَّع 409 odometer_lower", again)
	}
	if d := odoErrDetails(again); d["bike_change_left"] != "false" {
		t.Errorf("bike_change_left = %v والمتوقَّع false", d["bike_change_left"])
	}
}

// TestDriverMonitor_BikeChangeNoBogusDistance **لا مسافةَ عبر التبديل**: فتحٌ بتبديل ⇒ لا
// «خارج الدوام» سالب (والدوامُ نفسه يُحسب على العدّاد الجديد) · سكّرٌ بتبديل ⇒ لا «حسب العدّاد» ·
// دوامٌ بلا سكّر يليه فتحٌ بتبديل ⇒ لا «تقريبي» · وكلّها بعلم «تبديل درّاجة» وعدٍّ بملخّص اليوم.
func TestDriverMonitor_BikeChangeNoBogusDistance(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	day, base := rdDayBase()
	h.Setting("drivers.odometer_off_alert_km", "1")
	bike := func(id string) {
		if _, err := h.Pool.Exec(ctxBG(), `UPDATE driver_odometer_readings SET bike_change = true WHERE id = $1::uuid`, id); err != nil {
			t.Fatal(err)
		}
	}
	shift := func(opened, closed time.Time, s, e string) string {
		var id string
		var end any
		if e != "" {
			end = e
		}
		if err := h.Pool.QueryRow(ctxBG(), `
			INSERT INTO driver_shifts (driver_id, opened_at, closed_at, close_reason, start_reading_id,
			                           end_reading_id, opened_build, opened_capable)
			VALUES ($1::uuid, $2, $3, 'driver', $4::uuid, $5::uuid, 22, true) RETURNING id::text`,
			drv.ID, opened, closed, s, end).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	// ١: 1250 → 1300 · ٢: فتحٌ بتبديل 400 → 450 · ٣: 460 → سكّرٌ بتبديل 20 ·
	// ٤: 30 بلا سكّر · ٥: فتحٌ بتبديل 7 → 9
	t1 := base
	a := shift(t1, t1.Add(time.Hour),
		odoInsertReading(t, h, drv.ID, "start", 1250, t1, nil),
		odoInsertReading(t, h, drv.ID, "end", 1300, t1.Add(time.Hour), nil))
	t2 := t1.Add(2 * time.Hour)
	s2 := odoInsertReading(t, h, drv.ID, "start", 400, t2, nil)
	bike(s2)
	b := shift(t2, t2.Add(time.Hour), s2, odoInsertReading(t, h, drv.ID, "end", 450, t2.Add(time.Hour), nil))
	t3 := t2.Add(2 * time.Hour)
	e3 := odoInsertReading(t, h, drv.ID, "end", 20, t3.Add(time.Hour), nil)
	bike(e3)
	c := shift(t3, t3.Add(time.Hour), odoInsertReading(t, h, drv.ID, "start", 460, t3, nil), e3)
	t4 := t3.Add(2 * time.Hour)
	d := shift(t4, t4.Add(time.Hour), odoInsertReading(t, h, drv.ID, "start", 30, t4, nil), "")
	t5 := t4.Add(2 * time.Hour)
	s5 := odoInsertReading(t, h, drv.ID, "start", 7, t5, nil)
	bike(s5)
	shift(t5, t5.Add(time.Hour), s5, odoInsertReading(t, h, drv.ID, "end", 9, t5.Add(time.Hour), nil))

	got := odoDayOf(t, h, admin.Token, drv.ID, day)
	by := map[string]int{}
	for i, s := range got.Shifts {
		by[s.ID] = i
	}
	sa, sb, sc, sd := got.Shifts[by[a]], got.Shifts[by[b]], got.Shifts[by[c]], got.Shifts[by[d]]
	if sa.OdometerKm == nil || *sa.OdometerKm != 50 || hasFlag(sa.Flags, "bike_change") {
		t.Errorf("الدوام ١: حسب العدّاد %v أعلام %v — المتوقَّع 50 بلا تبديل", sa.OdometerKm, sa.Flags)
	}
	if sb.OffBeforeKm != nil || hasFlag(sb.Flags, "off_duty_moved") || !hasFlag(sb.Flags, "bike_change") {
		t.Errorf("الدوام ٢ (فتحٌ بتبديل): خارج الدوام %v أعلام %v — المتوقَّع فارغ و bike_change", ptrStr(sb.OffBeforeKm), sb.Flags)
	}
	if sb.OdometerKm == nil || *sb.OdometerKm != 50 {
		t.Errorf("الدوام ٢: حسب العدّاد %v — المتوقَّع 50 على العدّاد الجديد", ptrStr(sb.OdometerKm))
	}
	if sc.OdometerKm != nil || sc.UnseenKm != nil || !hasFlag(sc.Flags, "bike_change") {
		t.Errorf("الدوام ٣ (سكّرٌ بتبديل): حسب العدّاد %v أعلام %v — المتوقَّع فارغ و bike_change", ptrStr(sc.OdometerKm), sc.Flags)
	}
	if sd.ApproxKm != nil {
		t.Errorf("الدوام ٤: التقريبي %v — المتوقَّع فارغ (الفتحُ التالي بتبديل)", ptrStr(sd.ApproxKm))
	}
	if got.Odometer.OnKm == nil || *got.Odometer.OnKm != 102 {
		t.Errorf("مجموع اليوم حسب العدّاد %v — المتوقَّع 102 (50+50+2)", ptrStr(got.Odometer.OnKm))
	}
	if got.Odometer.OffKm != nil && *got.Odometer.OffKm < 0 {
		t.Errorf("خارج الدوام باليوم سالب: %v", *got.Odometer.OffKm)
	}
	if got.Odometer.BikeChanges != 3 {
		t.Errorf("تبديلات اليوم = %d والمتوقَّع 3", got.Odometer.BikeChanges)
	}
}

func ptrStr(p *float64) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprint(*p)
}
