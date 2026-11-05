package server

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestLandmarkReminder_ShiftEndCountsTimeInsideToday **تذكيرُ الإغلاق يقيس ما قضاه من دوامه في
// يوم المنصّة الحاليّ** (مراجعة ٢٠٢٦-١١-٠٥): كان كلُّ دوامٍ فُتح قبل التاسعة يُسقَط، **فمن فتح
// الثامنةَ صباحاً وسكّر الرابعةَ عصراً ومعه ١ من ٥ لم يصله شيءٌ ذلك اليوم.** ومن سكّر التاسعةَ
// وخمسَ دقائقَ بعد ليلته لا يُذكَّر — خمسُ دقائقَ من يومه.
func TestLandmarkReminder_ShiftEndCountsTimeInsideToday(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	f.srv.notify = notifications.New(f.pool, f.srv.hub, quiet)
	for _, k := range []string{"platform.day_start_hour", "drivers.landmarks_daily_quota", "drivers.landmarks_remind_on_shift_end"} {
		lmKeepSetting(t, f, k)
	}
	f.setSetting(t, "platform.day_start_hour", 9)
	f.setSetting(t, "drivers.landmarks_daily_quota", 5)
	f.setSetting(t, "drivers.landmarks_remind_on_shift_end", true)
	loc := damascusLoc()
	at := func(d, hh, mm int) time.Time { return time.Date(2031, 9, d, hh, mm, 0, 0, loc) }
	reminders := func(driverID, day string) int {
		t.Helper()
		var n int
		if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications
			WHERE user_id = $1::uuid AND entity = 'landmarks' AND entity_id = $2`, driverID, day).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// ١ · فتح الثامنةَ صباحاً (قبل بدء اليوم) وسكّر الرابعةَ عصراً ومعه ١ من ٥ ⇒ تذكيرٌ واحد.
	day := testdb.NewUser(t, f.pool, "driver")
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO landmark_suggestions (name, kind, point, suggested_by, status, platform_day)
		VALUES ('معلم الإغلاق', 'shop', ST_SetSRID(ST_MakePoint(40.3, 34.4), 4326)::geography, $1::uuid, 'pending', '2031-09-08')`,
		day); err != nil {
		t.Fatal(err)
	}
	opened := at(8, 8, 0)
	f.srv.remindLandmarksOnShiftEnd(ctx, day, &opened, at(8, 16, 0))
	if n := reminders(day, "2031-09-08"); n != 1 {
		t.Fatalf("**دوامٌ من ٨ صباحاً لـ٤ عصراً ومعه ١ من ٥ ولم يصله تذكير**: %d", n)
	}

	// ٢ · دوامُ ليلٍ من الثامنة مساءً للتاسعة وخمس دقائق صباحاً ⇒ لا شيء عن اليوم الجديد.
	night := testdb.NewUser(t, f.pool, "driver")
	started := at(8, 20, 0)
	f.srv.remindLandmarksOnShiftEnd(ctx, night, &started, at(9, 9, 5))
	if n := reminders(night, "2031-09-09"); n != 0 {
		t.Fatalf("**ذُكِّر من سكّر التاسعةَ وخمسَ دقائقَ بعد ليلته**: %d", n)
	}

	// ٣ · فتح وسكّر بعد عشر دقائق في وسط اليوم ⇒ لا شيء.
	brief := testdb.NewUser(t, f.pool, "driver")
	from := at(10, 12, 0)
	f.srv.remindLandmarksOnShiftEnd(ctx, brief, &from, at(10, 12, 10))
	if n := reminders(brief, "2031-09-10"); n != 0 {
		t.Fatalf("ذُكِّر بعد عشر دقائق من فتح دوامه: %d", n)
	}
}
