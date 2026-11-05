package platform

// ══════════════════════════════════════════════════════════════════════
// **يومُ شغل المنصّة** — من فتحها لإغلاقها (قرار المالك ٢٠٢٦-١١-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// «كيلومترات اليوم» للسائق تتبع **يومَ المنصّة**: من ساعة الفتح (`platform.day_start_hour`)
// لساعة إغلاقها من جدول الدوام (مثلاً منتصف الليل) — لا يومَ الحساب ٩←٩ ولا الدوام. وتبدأ من
// الصفر بالفتح التالي. **وما مشاه بعد الإغلاق** يُعرض وحده («مشي بعد إغلاق المنصة»).
//
// **والمنصّةُ ٢٤ ساعة** (زرّ «مفتوحين»، أو جدولٌ لا يسري، أو يومٌ كلُّه مفتوح) **⇒ لا إغلاق**،
// فاليومُ يومُ الحساب ٩←٩ كما كان.

import (
	"context"
	"time"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// WorkDayClose **لحظةُ إغلاق المنصّة في يومٍ يبدأ `start` وينتهي `next`** (بداية اليوم التالي).
//
// **آخرُ إغلاقٍ قبل `next`**: فترتان باليوم (استراحةُ ظهر) لا تجعلان العصرَ «بعد الإغلاق» —
// يُتبع الجدولُ من فتحٍ لإغلاقٍ لفتحٍ حتى آخر إغلاق. ومنصّةٌ تفتح بعد بداية اليوم (١٠ واليومُ
// يبدأ ٩) يُقرأ أوّلُ فتحها. **و`false`** = لا إغلاق ضمن اليوم (٢٤ ساعة، أو يومٌ مسكّرٌ كلُّه)
// ⇒ اليومُ كلُّه.
func WorkDayClose(sch Schedule, enforced bool, start, next time.Time) (time.Time, bool) {
	if !enforced || len(sch) == 0 {
		return time.Time{}, false
	}
	t := start
	if !sch.OpenAt(t) {
		o := sch.NextOpenAt(t)
		if o == nil || !o.Before(next) {
			return time.Time{}, false
		}
		t = *o
	}
	for i := 0; i < 16; i++ {
		c := sch.NextCloseAt(t)
		if c == nil || !c.Before(next) {
			// **لا إغلاقَ قبل الفتح التالي** — مفتوحةٌ لآخر اليوم.
			return time.Time{}, false
		}
		o := sch.NextOpenAt(*c)
		if o == nil || !o.Before(next) {
			return *c, true
		}
		t = *o
	}
	return time.Time{}, false
}

// WorkDayClose **إغلاقُ المنصّة باليوم** كما يحكم الآن: الزرّ إن ضُغط («مفتوحين» ⇒ لا إغلاق ·
// «مسكّرين» ⇒ لا يومَ شغلٍ أصلاً فيُعدّ اليومُ كلُّه)، وإلّا الجدولُ إن سرى.
func (s *Service) WorkDayClose(ctx context.Context, q dbtx.Querier, start, next time.Time) (time.Time, bool, error) {
	sch, err := s.Schedule(ctx, q)
	if err != nil {
		return time.Time{}, false, err
	}
	enforced, gate := s.platformGate(ctx, sch)
	c, ok := WorkDayClose(gate, enforced, start, next)
	return c, ok, nil
}
