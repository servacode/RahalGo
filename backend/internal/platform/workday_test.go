package platform

import (
	"testing"
	"time"
)

// TestWorkDayClose **إغلاقُ يوم الشغل**: آخرُ إغلاقٍ قبل الفتح التالي (استراحةُ الظهر لا تقصّ
// العصر) · الإغلاقُ بمنتصف الليل · ولا إغلاقَ والجدولُ لا يسري أو اليومُ كلُّه مفتوح.
func TestWorkDayClose(t *testing.T) {
	loc := Location()
	start := time.Date(2031, 4, 7, 9, 0, 0, 0, loc)
	next := start.AddDate(0, 0, 1)
	all := func(ws ...Window) Schedule {
		var out Schedule
		for d := 0; d < 7; d++ {
			for _, w := range ws {
				w.Day = d
				out = append(out, w)
			}
		}
		return out
	}
	cases := []struct {
		name     string
		sch      Schedule
		enforced bool
		want     time.Time
		ok       bool
	}{
		{"٩←٢٢", all(Window{Start: 9 * 60, End: 22 * 60}), true, time.Date(2031, 4, 7, 22, 0, 0, 0, loc), true},
		{"استراحة ظهر", all(Window{Start: 9 * 60, End: 13 * 60}, Window{Start: 15 * 60, End: 23 * 60}), true,
			time.Date(2031, 4, 7, 23, 0, 0, 0, loc), true},
		{"لمنتصف الليل", all(Window{Start: 9 * 60, End: 0}), true, time.Date(2031, 4, 8, 0, 0, 0, 0, loc), true},
		{"يفتح ١٠", all(Window{Start: 10 * 60, End: 20 * 60}), true, time.Date(2031, 4, 7, 20, 0, 0, 0, loc), true},
		{"لا يسري", all(Window{Start: 9 * 60, End: 22 * 60}), false, time.Time{}, false},
		{"فارغ", nil, true, time.Time{}, false},
	}
	for _, c := range cases {
		got, ok := WorkDayClose(c.sch, c.enforced, start, next)
		if ok != c.ok || !got.Equal(c.want) {
			t.Errorf("%s: %v %v — المتوقَّع %v %v", c.name, got, ok, c.want, c.ok)
		}
	}
}
