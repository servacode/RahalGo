package qa

// ══════════════════════════════════════════════════════════════════════
// **«لم تكتمل المعالم الـ٥ الخاصة بك بعد» — وملخّصُ المالك** (قرارُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// **والجولاتُ بوقتٍ ثابتٍ في ٢٠٣١** — يومٌ لا يختمه غيرُ هذا الاختبار، **فالعدُّ يُقرأ صفّاً
// صفّاً** ولا يمسّه ما خلّفه غيرُه.

import (
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/platform"
)

const lmQuotaTitle = "لم تكتمل المعالم الـ5 الخاصة بك بعد"

// lmAt **لحظةٌ بدمشق.**
func lmAt(y int, mo time.Month, d, hh, mm int) time.Time {
	return time.Date(y, mo, d, hh, mm, 0, 0, platform.Location())
}

// lmShift **على الدوام منذ** `since` — أو خارجَه.
func lmShift(t *testing.T, h *Harness, driverID string, on bool, since time.Time) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET on_shift = $2,
		shift_started_at = CASE WHEN $2 THEN $3::timestamptz END WHERE id = $1::uuid`, driverID, on, since); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `UPDATE users SET on_shift = false, shift_started_at = NULL WHERE id = $1::uuid`, driverID)
	})
}

// lmSeed **معالمُ بيومها وحالتها** — مباشرةً في القاعدة.
func lmSeed(t *testing.T, h *Harness, driverID, day, status string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO landmark_suggestions (name, kind, point, suggested_by, status, platform_day)
			VALUES ($1, 'shop', ST_SetSRID(ST_MakePoint(40.3, 34.4), 4326)::geography, $2::uuid, $3, $4::date)`,
			uniq("معلم تذكير "), driverID, status, day); err != nil {
			t.Fatal(err)
		}
	}
}

// lmReminders **تذكيراتُ المعالم في صندوق السائق** — وكلُّها للسائق وحدَه.
func lmReminders(t *testing.T, h *Harness, driverID string) []string {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `SELECT body, push_apps::text FROM notifications
		WHERE user_id = $1::uuid AND entity = 'landmarks' AND title = $2 ORDER BY created_at`, driverID, lmQuotaTitle)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var body, apps string
		if err := rows.Scan(&body, &apps); err != nil {
			t.Fatal(err)
		}
		if apps != "{driver}" {
			t.Fatalf("التذكيرُ لغير تطبيق السائق: %s", apps)
		}
		out = append(out, body)
	}
	return out
}

// TestLandmarkReminder_HourOncePerDay **بالساعة مرّةً باليوم** — لمن على الدوام منذ نصف ساعةٍ
// فأكثر ولم يكمل؛ **لا لمن أكمل، ولا لمن خارجَ الدوام، ولا لمن فتح للتوّ، ولا والعدُّ مطفأ.**
func TestLandmarkReminder_HourOncePerDay(t *testing.T) {
	h := New(t)
	h.Setting("platform.day_start_hour", "9")
	h.Setting("drivers.landmarks_reminder_hour", "18")
	h.Setting("drivers.landmarks_daily_quota", "5")
	const day = "2031-04-07"
	at := lmAt(2031, 4, 7, 18, 5)
	two, five, off, fresh := h.NewUser("driver"), h.NewUser("driver"), h.NewUser("driver"), h.NewUser("driver")
	lmShift(t, h, two.ID, true, at.Add(-2*time.Hour))
	lmShift(t, h, five.ID, true, at.Add(-2*time.Hour))
	lmShift(t, h, off.ID, false, at)
	lmShift(t, h, fresh.ID, true, at.Add(-10*time.Minute))
	lmSeed(t, h, two.ID, day, "pending", 1)
	lmSeed(t, h, two.ID, day, "approved", 1)
	lmSeed(t, h, two.ID, day, "duplicate", 2) // **المكرَّرُ لا يُعدّ**
	lmSeed(t, h, two.ID, day, "rejected", 1)  // **ولا المرفوض**
	lmSeed(t, h, five.ID, day, "pending", 5)

	// **قبل الساعة لا شيء.**
	if _, err := h.API.RemindLandmarkQuota(ctxBG(), lmAt(2031, 4, 7, 17, 55)); err != nil {
		t.Fatal(err)
	}
	if got := lmReminders(t, h, two.ID); len(got) != 0 {
		t.Fatalf("ذُكِّر قبل الساعة: %v", got)
	}
	if _, err := h.API.RemindLandmarkQuota(ctxBG(), at); err != nil {
		t.Fatal(err)
	}
	got := lmReminders(t, h, two.ID)
	if len(got) != 1 || !strings.Contains(got[0], "أرسلت 2 من 5") {
		t.Fatalf("**سائقٌ معه ٢ من ٥ لم يُذكَّر مرّةً واحدة**: %v", got)
	}
	// **وجولةٌ ثانيةٌ لا تكرّر.**
	if _, err := h.API.RemindLandmarkQuota(ctxBG(), at.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if n := len(lmReminders(t, h, two.ID)); n != 1 {
		t.Fatalf("تكرّر التذكيرُ في اليوم نفسِه: %d", n)
	}
	for who, u := range map[string]*User{"أكمل الخمسة": five, "خارجَ الدوام": off, "فتح للتوّ": fresh} {
		if n := len(lmReminders(t, h, u.ID)); n != 0 {
			t.Fatalf("**ذُكِّر من %s**: %d", who, n)
		}
	}
	// **ومن فتح للتوّ يُذكَّر حين يتمّ نصفَ ساعته.**
	if _, err := h.API.RemindLandmarkQuota(ctxBG(), at.Add(25*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if n := len(lmReminders(t, h, fresh.ID)); n != 1 {
		t.Fatalf("من أتمّ نصفَ ساعةٍ على الدوام لم يُذكَّر: %d", n)
	}

	// **والعدُّ المطفأُ لا يُذكِّر** — في اليوم التالي.
	h.Setting("drivers.landmarks_daily_quota", "0")
	if _, err := h.API.RemindLandmarkQuota(ctxBG(), lmAt(2031, 4, 8, 18, 5)); err != nil {
		t.Fatal(err)
	}
	if n := len(lmReminders(t, h, two.ID)); n != 1 {
		t.Fatalf("ذُكِّر والعدُّ مطفأ: %d", n)
	}
}

// TestLandmarkReminder_ShiftEnd **سكّر دوامَه بنفسه ولم يكمل** — تذكيرٌ واحدٌ باليوم، **والإعدادُ
// يطفئه.**
func TestLandmarkReminder_ShiftEnd(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	h.Setting("drivers.landmarks_daily_quota", "5")
	h.Setting("drivers.landmarks_remind_on_shift_end", "true")
	drv := h.NewUser("driver")
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `UPDATE users SET on_shift = false WHERE id = $1::uuid`, drv.ID)
	})
	shift := func(u *User, on bool) {
		t.Helper()
		if r := h.POST("/api/v1/driver/shift", u.Token, map[string]any{"on": on}); r.Code != 200 {
			t.Fatalf("الدوام %v: %s", on, r)
		}
	}
	shift(drv, true)
	if r := lmPost(h, drv.Token, uniq("جامع الإغلاق "), "mosque", 34.4510, 40.3510); r.Code != 200 {
		t.Fatalf("المعلم: %s", r)
	}
	shift(drv, false)
	got := lmReminders(t, h, drv.ID)
	if len(got) != 1 || !strings.Contains(got[0], "سكّرت دوامك") || !strings.Contains(got[0], "أرسلت 1 من 5") {
		t.Fatalf("**سكّر دوامَه ومعه ١ من ٥ ولم يُذكَّر**: %v", got)
	}
	// **فتح وسكّر ثانيةً في اليوم نفسِه ⇒ لا شيء.**
	shift(drv, true)
	shift(drv, false)
	if n := len(lmReminders(t, h, drv.ID)); n != 1 {
		t.Fatalf("تكرّر تذكيرُ الإغلاق: %d", n)
	}
	// **وإغلاقٌ وهو خارجَ الدوام ليس إغلاقاً.**
	idle := h.NewUser("driver")
	shift(idle, false)
	if n := len(lmReminders(t, h, idle.ID)); n != 0 {
		t.Fatalf("ذُكِّر من لم يكن على الدوام: %d", n)
	}
	// **والإعدادُ المطفأُ لا يُذكِّر.**
	h.Setting("drivers.landmarks_remind_on_shift_end", "false")
	other := h.NewUser("driver")
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `UPDATE users SET on_shift = false WHERE id = $1::uuid`, other.ID)
	})
	shift(other, true)
	shift(other, false)
	if n := len(lmReminders(t, h, other.ID)); n != 0 {
		t.Fatalf("ذُكِّر والإعدادُ مطفأ: %d", n)
	}
}

// TestLandmarkReminder_HourBeforeDayStart **ساعةُ التذكير قبل فتح الدوام** (٢ ليلاً والفتحُ ٩)
// **تقع في آخر اليوم نفسِه** — ويُختم على يوم المنصّة لا على يوم التقويم.
func TestLandmarkReminder_HourBeforeDayStart(t *testing.T) {
	h := New(t)
	h.Setting("platform.day_start_hour", "9")
	h.Setting("drivers.landmarks_reminder_hour", "2")
	h.Setting("drivers.landmarks_daily_quota", "5")
	drv := h.NewUser("driver")
	lmShift(t, h, drv.ID, true, lmAt(2031, 5, 10, 20, 0))
	lmSeed(t, h, drv.ID, "2031-05-10", "pending", 1)
	if _, err := h.API.RemindLandmarkQuota(ctxBG(), lmAt(2031, 5, 10, 23, 0)); err != nil {
		t.Fatal(err)
	}
	if n := len(lmReminders(t, h, drv.ID)); n != 0 {
		t.Fatalf("ذُكِّر قبل الثانية ليلاً: %d", n)
	}
	if _, err := h.API.RemindLandmarkQuota(ctxBG(), lmAt(2031, 5, 11, 2, 10)); err != nil {
		t.Fatal(err)
	}
	got := lmReminders(t, h, drv.ID)
	if len(got) != 1 || !strings.Contains(got[0], "أرسلت 1 من 5") {
		t.Fatalf("**تذكيرُ الثانية ليلاً لم يصل على يوم المنصّة نفسِه**: %v", got)
	}
	var day string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT day::text FROM landmark_quota_reminders
		WHERE driver_id = $1::uuid AND reason = 'hour'`, drv.ID).Scan(&day); err != nil {
		t.Fatal(err)
	}
	if day != "2031-05-10" {
		t.Fatalf("خُتم على %s والمنتظَر يومُ المنصّة 2031-05-10", day)
	}
}

// TestLandmarkOwnerDigest_OncePerDay **ملخّصُ أمس للمالك مرّةً** — بعد فتح اليوم بربع ساعة،
// **وبأرقامٍ صحيحة**: الجديدُ ومن أرسله، والمكرَّر، ومن لم يكمل (ومنهم من اشتغل ولم يرسل).
func TestLandmarkOwnerDigest_OncePerDay(t *testing.T) {
	h := New(t)
	h.Setting("platform.day_start_hour", "9")
	h.Setting("drivers.landmarks_daily_quota", "5")
	owner := h.NewUser("owner_super_admin")
	const yesterday = "2031-06-14"
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM landmark_daily_digests WHERE day = $1::date`, yesterday)
	})
	a, b, c := h.NewUser("driver"), h.NewUser("driver"), h.NewUser("driver")
	lmSeed(t, h, a.ID, yesterday, "pending", 3)
	lmSeed(t, h, a.ID, yesterday, "duplicate", 1)
	lmSeed(t, h, c.ID, yesterday, "approved", 5)
	// **اشتغل ولم يرسل** — أثرُه على الدوام.
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO driver_trail (driver_id, recorded_at, lat, lng)
		VALUES ($1::uuid, $2, 34.4, 40.3)`, b.ID, lmAt(2031, 6, 14, 12, 0)); err != nil {
		t.Fatal(err)
	}
	digests := func() []string {
		t.Helper()
		rows, err := h.Pool.Query(ctxBG(), `SELECT body FROM notifications
			WHERE user_id = $1::uuid AND title = 'معالم السائقين — أمس' AND entity_id = $2`, owner.ID, yesterday)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var b string
			_ = rows.Scan(&b)
			out = append(out, b)
		}
		return out
	}
	if sent, err := h.API.DigestLandmarks(ctxBG(), lmAt(2031, 6, 15, 9, 10)); err != nil || sent {
		t.Fatalf("أُرسل قبل ربع الساعة: %v %v", sent, err)
	}
	if sent, err := h.API.DigestLandmarks(ctxBG(), lmAt(2031, 6, 15, 9, 20)); err != nil || !sent {
		t.Fatalf("الملخّص لم يُرسَل: %v %v", sent, err)
	}
	got := digests()
	if len(got) != 1 {
		t.Fatalf("الملخّصُ وصل %d مرّة", len(got))
	}
	for _, want := range []string{"جديد 8 من 2 سائق", "مكرّر 1", "بانتظار موافقتك", "ما كمّلوا الـ5: 2 سائق"} {
		if !strings.Contains(got[0], want) {
			t.Fatalf("الملخّصُ بلا «%s»: %q", want, got[0])
		}
	}
	if sent, _ := h.API.DigestLandmarks(ctxBG(), lmAt(2031, 6, 15, 10, 0)); sent || len(digests()) != 1 {
		t.Fatal("**تكرّر الملخّص في اليوم نفسِه**")
	}
}
