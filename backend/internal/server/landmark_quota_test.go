package server

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// lmKeepSetting **يعيد الإعدادَ بعد الاختبار كما كان** — `setSetting` لا يعيده.
func lmKeepSetting(t *testing.T, f *driverFixture, key string) {
	t.Helper()
	ctx := context.Background()
	var old *string
	_ = f.pool.QueryRow(ctx, `SELECT value::text FROM app_settings WHERE key = $1`, key).Scan(&old)
	t.Cleanup(func() {
		if old == nil {
			_, _ = f.pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = $1`, key)
			return
		}
		_, _ = f.pool.Exec(context.Background(), `UPDATE app_settings SET value = $2::jsonb WHERE key = $1`, key, *old)
	})
}

// TestLandmark_CountUsesPlatformDay **العدُّ بيوم المنصّة لا بيوم التقويم** (قرارُ المالك
// ٢٠٢٦-١١-٠٥، يومُ الحساب ٢٠٢٦-١٠-٢٦): معلمُ ٨:٣٠ صباحاً على اليوم السابق، ومعلمُ ٩:٣٠ على
// يومه — **وساعةُ فتح الدوام من الإعداد**: بالثامنة يصير الأوّلُ على يومه.
func TestLandmark_CountUsesPlatformDay(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	drv := testdb.NewUser(t, f.pool, "driver")
	lmKeepSetting(t, f, "platform.day_start_hour")
	f.setSetting(t, "platform.day_start_hour", 9)
	sfx := strconv.FormatInt(time.Now().UnixNano()%1_000_000_000, 10)
	loc := damascusLoc()
	early := time.Date(2031, 3, 10, 8, 30, 0, 0, loc)
	late := time.Date(2031, 3, 10, 9, 30, 0, 0, loc)
	const lat, lng = 34.5010, 40.1010

	a, err := f.srv.submitLandmark(ctx, drv, landmarkSubmit{Name: "جامع البكور " + sfx, Kind: "mosque", Lat: lat, Lng: lng}, early)
	if err != nil {
		t.Fatal(err)
	}
	if a.Day != "2031-03-09" || a.Today != 1 || a.Dup != nil {
		t.Fatalf("**معلمُ ٨:٣٠ ليس على اليوم السابق**: %+v", a)
	}
	b, err := f.srv.submitLandmark(ctx, drv, landmarkSubmit{Name: "جامع الظهيرة " + sfx, Kind: "mosque", Lat: lat, Lng: lng}, late)
	if err != nil {
		t.Fatal(err)
	}
	if b.Day != "2031-03-10" || b.Today != 1 {
		t.Fatalf("**معلمُ ٩:٣٠ حُسب مع اليوم السابق**: %+v", b)
	}
	var stamped string
	if err := f.pool.QueryRow(ctx, `SELECT platform_day::text FROM landmark_suggestions WHERE id = $1`, a.ID).Scan(&stamped); err != nil {
		t.Fatal(err)
	}
	if stamped != "2031-03-09" {
		t.Fatalf("اليومُ المختوم %q", stamped)
	}

	// **والساعةُ إعدادٌ لا رقمٌ مكتوب** — بالثامنة صار ٨:٣٠ على يومه.
	f.setSetting(t, "platform.day_start_hour", 8)
	c, err := f.srv.submitLandmark(ctx, drv, landmarkSubmit{Name: "جامع الضحى " + sfx, Kind: "mosque", Lat: lat, Lng: lng}, early)
	if err != nil {
		t.Fatal(err)
	}
	if c.Day != "2031-03-10" || c.Today != 2 {
		t.Fatalf("ساعةُ فتح الدوام لم تُقرأ من الإعداد: %+v", c)
	}
}
