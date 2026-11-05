package qa

// ══════════════════════════════════════════════════════════════════════
// **«المنصة أغلقت، يجب عليك إغلاق دوامك»** — قرار المالك ٢٠٢٦-١١-٠٦
// ══════════════════════════════════════════════════════════════════════
//
// إشعارٌ واحدٌ لكلّ سائقٍ بدوامه لما تسكّر المنصّة، وتذكيرٌ واحدٌ باليوم التالي إن بقي دوامُه
// نفسُه مفتوحاً — ولا تسكيرٌ آليّ، ولا شيءَ والمنصّةُ مفتوحةٌ ٢٤ ساعة.

import (
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/platform"
	"github.com/servacode/rahalgo/backend/internal/shifts"
)

// shiftRemindTitles عناوينُ إشعارات الدوام المفتوح للسائق، بترتيبها.
func shiftRemindTitles(t *testing.T, h *Harness, driverID string) []string {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `SELECT title FROM notifications
		WHERE user_id = $1::uuid AND entity = 'shift' ORDER BY created_at`, driverID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

// shiftRemindHours **المنصّةُ من ٩ الصبح لـ١١ الليل كلَّ يوم** والجدولُ يسري — ويُمحى بعد الفحص.
func shiftRemindHours(t *testing.T, h *Harness) {
	t.Helper()
	var ws []platform.Window
	for d := 0; d < 7; d++ {
		ws = append(ws, platform.Window{Day: d, Start: 9 * 60, End: 23 * 60})
	}
	setHours(t, h, ws...)
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM platform_hours`) })
	h.Setting("hours.platform_enforced", "true")
	h.Setting("hours.platform_switch", `"schedule"`)
	h.Setting("platform.day_start_hour", "9")
}

// shiftRemindOpen **سائقٌ بدوامه** — من باب الدوام الواحد.
func shiftRemindOpen(t *testing.T, h *Harness, driverID string) {
	t.Helper()
	if _, err := shifts.Open(ctxBG(), h.Pool, driverID, 0, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = shifts.Close(ctxBG(), h.Pool, driverID, shifts.ReasonQA) })
}

func shiftRemindRound(t *testing.T, h *Harness, at time.Time) {
	t.Helper()
	if _, _, err := h.API.RemindOpenShifts(ctxBG(), at); err != nil {
		t.Fatal(err)
	}
}

// TestShiftCloseReminder_OncePerClosingAndNextDay **إغلاقُ المنصّة**: المفتوحُ لا يُرسَل له شيء ·
// بعد الإغلاق إشعارٌ واحد ولو دارت الجولةُ ثلاثاً (وبعد منتصف الليل) · ولا شيءَ لمن خارج الدوام ·
// ولا تسكيرَ آليّ · وباليوم التالي لما تفتح تذكيرٌ واحد لمن بقي دوامُه مفتوحاً — لا لمن سكّره بالليل.
func TestShiftCloseReminder_OncePerClosingAndNextDay(t *testing.T) {
	h := New(t)
	shiftRemindHours(t, h)
	on := h.NewUser("driver")
	off := h.NewUser("driver")
	closer := h.NewUser("driver")
	shiftRemindOpen(t, h, on.ID)
	shiftRemindOpen(t, h, closer.ID)

	shiftRemindRound(t, h, lmAt(2031, 4, 7, 22, 0)) // مفتوحة
	if got := shiftRemindTitles(t, h, on.ID); len(got) != 0 {
		t.Fatalf("والمنصّةُ مفتوحة: %v — المتوقَّع لا شيء", got)
	}
	for _, at := range []time.Time{lmAt(2031, 4, 7, 23, 1), lmAt(2031, 4, 7, 23, 30), lmAt(2031, 4, 8, 1, 0)} {
		shiftRemindRound(t, h, at)
	}
	got := shiftRemindTitles(t, h, on.ID)
	if len(got) != 1 || got[0] != "المنصة أغلقت، يجب عليك إغلاق دوامك" {
		t.Fatalf("بعد الإغلاق: %q — المتوقَّع إشعارٌ واحد «المنصة أغلقت، يجب عليك إغلاق دوامك»", got)
	}
	if got := shiftRemindTitles(t, h, off.ID); len(got) != 0 {
		t.Errorf("لمن خارج الدوام: %v — المتوقَّع لا شيء", got)
	}
	if !onShiftOf(t, h, on.ID) {
		t.Fatal("انسكّر الدوامُ آليّاً — المتوقَّع يضلّ مفتوحاً")
	}
	// سكّر دوامَه بالليل بعد الإشعار ⇒ لا تذكيرَ الصبح
	if _, err := shifts.Close(ctxBG(), h.Pool, closer.ID, shifts.ReasonDriver); err != nil {
		t.Fatal(err)
	}

	// الصبح: الثامنة مسكّرة، والعاشرة فتحت ويومُ المنصّة تغيّر
	shiftRemindRound(t, h, lmAt(2031, 4, 8, 8, 30))
	if got := shiftRemindTitles(t, h, on.ID); len(got) != 1 {
		t.Fatalf("قبل الفتح: %v — المتوقَّع لسّا إشعارٌ واحد", got)
	}
	shiftRemindRound(t, h, lmAt(2031, 4, 8, 10, 0))
	shiftRemindRound(t, h, lmAt(2031, 4, 8, 10, 30))
	got = shiftRemindTitles(t, h, on.ID)
	if len(got) != 2 || got[1] != "دوامك من أمس لسّا مفتوح" {
		t.Fatalf("باليوم التالي: %q — المتوقَّع تذكيرٌ واحد «دوامك من أمس لسّا مفتوح»", got)
	}
	if got := shiftRemindTitles(t, h, closer.ID); len(got) != 1 {
		t.Errorf("من سكّر بالليل: %v — المتوقَّع إشعارُ الإغلاق وحدَه", got)
	}
	if !onShiftOf(t, h, on.ID) {
		t.Fatal("انسكّر الدوامُ آليّاً الصبح")
	}
	// والليلةُ التالية إغلاقٌ جديد ⇒ إشعارٌ جديدٌ واحد
	shiftRemindRound(t, h, lmAt(2031, 4, 8, 23, 10))
	shiftRemindRound(t, h, lmAt(2031, 4, 8, 23, 20))
	if got := shiftRemindTitles(t, h, on.ID); len(got) != 3 {
		t.Errorf("الإغلاقُ التالي: %v — المتوقَّع ثلاثة بالمجموع", got)
	}
}

// TestShiftCloseReminder_NothingWhenAlwaysOpen **المنصّةُ ٢٤ ساعة** — زرُّ «مفتوحين»، أو جدولٌ
// لا يسري ⇒ لا شيء بالليل ولا الصبح. **وزرُّ «مسكّرين»** إغلاقٌ يُرسَل به.
func TestShiftCloseReminder_NothingWhenAlwaysOpen(t *testing.T) {
	h := New(t)
	shiftRemindHours(t, h)
	drv := h.NewUser("driver")
	shiftRemindOpen(t, h, drv.ID)

	h.Setting("hours.platform_switch", `"open"`)
	shiftRemindRound(t, h, lmAt(2031, 5, 7, 23, 30))
	shiftRemindRound(t, h, lmAt(2031, 5, 8, 10, 0))
	if got := shiftRemindTitles(t, h, drv.ID); len(got) != 0 {
		t.Fatalf("زرّ «مفتوحين»: %v — المتوقَّع لا شيء", got)
	}
	h.Setting("hours.platform_switch", `"schedule"`)
	h.Setting("hours.platform_enforced", "false")
	shiftRemindRound(t, h, lmAt(2031, 5, 9, 23, 30))
	if got := shiftRemindTitles(t, h, drv.ID); len(got) != 0 {
		t.Fatalf("جدولٌ لا يسري: %v — المتوقَّع لا شيء", got)
	}
	h.Setting("hours.platform_switch", `"closed"`)
	shiftRemindRound(t, h, lmAt(2031, 5, 10, 14, 0))
	if got := shiftRemindTitles(t, h, drv.ID); len(got) != 1 {
		t.Fatalf("زرّ «مسكّرين»: %v — المتوقَّع إشعارٌ واحد", got)
	}
}
