package prizes

import (
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/offers"
)

func dam(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, offers.Damascus)
}

// TestAmountsByRank **جائزةُ كلّ مرتبة — وما بعد الخامسة كالخامسة.**
func TestAmountsByRank(t *testing.T) {
	base := []int64{1500, 1000, 750, 500, 400}
	got := amountsFor(base, 7)
	want := []int64{1500, 1000, 750, 500, 400, 400, 400}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("amountsFor = %v — والمنتظَر %v", got, want)
		}
	}
	if got := amountsFor(base, 2); len(got) != 2 || got[1] != 1000 {
		t.Fatalf("amountsFor(2) = %v", got)
	}
}

// TestCutoffAndAnnounce **الحسمُ يومَ min(29, آخرُ يومٍ − 1) ١٢:٠٠ والإعلانُ آخرَ يومٍ ١٢:٠٠.**
//
// فبراير ٢٨ يوماً ⇒ حسمٌ ٢٧ وإعلانٌ ٢٨ (قرارُ المالك)؛ والكبيس ٢٨/٢٩؛ وما سواهما ٢٩.
func TestCutoffAndAnnounce(t *testing.T) {
	cases := []struct {
		month          time.Time
		cutoff, annDay int
	}{
		{dam(2027, 2, 1, 0, 0), 27, 28},
		{dam(2028, 2, 1, 0, 0), 28, 29},
		{dam(2026, 12, 1, 0, 0), 29, 31},
		{dam(2027, 4, 1, 0, 0), 29, 30},
		{dam(2027, 1, 1, 0, 0), 29, 31},
	}
	for _, c := range cases {
		cut, ann := CutoffAt(c.month), AnnounceAt(c.month)
		if !cut.Equal(dam(c.month.Year(), c.month.Month(), c.cutoff, 12, 0)) {
			t.Errorf("%s: الحسم %v — والمنتظَر يوم %d ١٢:٠٠", MonthKey(c.month), cut, c.cutoff)
		}
		if !ann.Equal(dam(c.month.Year(), c.month.Month(), c.annDay, 12, 0)) {
			t.Errorf("%s: الإعلان %v — والمنتظَر يوم %d ١٢:٠٠", MonthKey(c.month), ann, c.annDay)
		}
	}
}

// TestCompetitionMonth **ما بعد الحسم للشهر التالي** — والحسمُ بتوقيت دمشق.
func TestCompetitionMonth(t *testing.T) {
	cases := []struct {
		at   time.Time
		want string
	}{
		{dam(2026, 12, 29, 11, 59), "2026-12"},
		{dam(2026, 12, 29, 12, 0), "2027-01"},
		{dam(2026, 12, 31, 23, 0), "2027-01"},
		{dam(2027, 2, 27, 11, 59), "2027-02"},
		{dam(2027, 2, 27, 12, 0), "2027-03"},
		// ٢٣:٣٠ غرينتش آخرَ نوفمبر = ٠٢:٣٠ دمشق أوّلَ ديسمبر.
		{time.Date(2026, 11, 30, 23, 30, 0, 0, time.UTC), "2026-12"},
	}
	for _, c := range cases {
		if got := MonthKey(CompetitionMonth(c.at)); got != c.want {
			t.Errorf("CompetitionMonth(%v) = %s — والمنتظَر %s", c.at, got, c.want)
		}
	}
}

// TestCountWindow **نافذةُ الشهر من حسم الماضي إلى حسمه** — ويومُ الإطلاق يقصّها من أوّلها.
func TestCountWindow(t *testing.T) {
	launch := dam(2026, 12, 1, 0, 0)
	if _, _, ok := countWindow(dam(2026, 11, 1, 0, 0), launch); ok {
		t.Fatal("**شهرٌ قبل الإطلاق يُعدّ**")
	}
	from, to, ok := countWindow(dam(2026, 12, 1, 0, 0), launch)
	if !ok || !from.Equal(launch) || !to.Equal(dam(2026, 12, 29, 12, 0)) {
		t.Fatalf("نافذةُ شهر الإطلاق %v..%v — والمنتظَر ١/١٢ ٠٠:٠٠ .. ٢٩/١٢ ١٢:٠٠", from, to)
	}
	from, to, _ = countWindow(dam(2027, 3, 1, 0, 0), launch)
	if !from.Equal(dam(2027, 2, 27, 12, 0)) || !to.Equal(dam(2027, 3, 29, 12, 0)) {
		t.Fatalf("نافذةُ آذار %v..%v — والمنتظَر من حسم شباط ٢٧ إلى ٢٩ آذار", from, to)
	}
}

// TestCurrentMonthBeforeLaunch **قبل الإطلاق يُقال حسمُ شهر الإطلاق.**
func TestCurrentMonthBeforeLaunch(t *testing.T) {
	launch := dam(2026, 12, 1, 0, 0)
	if got := MonthKey(CurrentMonth(dam(2026, 11, 25, 10, 0), launch)); got != "2026-12" {
		t.Fatalf("قبل الإطلاق: %s — والمنتظَر 2026-12", got)
	}
	if got := MonthKey(CurrentMonth(dam(2027, 1, 30, 10, 0), launch)); got != "2027-02" {
		t.Fatalf("بعد حسم يناير: %s — والمنتظَر 2027-02", got)
	}
}

// TestAwardExpiry **الجائزةُ تنتهي آخرَ الشهر الذي يلي الإعلان** — لا بعد ساعات.
func TestAwardExpiry(t *testing.T) {
	if got := AwardExpiry(dam(2026, 12, 31, 12, 0)); !got.Equal(time.Date(2027, 1, 31, 23, 59, 59, 0, offers.Damascus)) {
		t.Fatalf("الانتهاء %v", got)
	}
	if got := AwardExpiry(dam(2027, 1, 31, 12, 0)); !got.Equal(time.Date(2027, 2, 28, 23, 59, 59, 0, offers.Damascus)) {
		t.Fatalf("الانتهاء %v", got)
	}
}

// TestRankTieThresholdExclusion **الترتيبُ بالنقاط، والتعادلُ للأبكر، والحدُّ، واستثناءُ رابح الماضي.**
func TestRankTieThresholdExclusion(t *testing.T) {
	t0 := dam(2026, 12, 10, 10, 0)
	in := []Standing{
		{UserID: "late", Points: 90, ReachedAt: t0.Add(2 * time.Hour)},
		{UserID: "early", Points: 90, ReachedAt: t0},
		{UserID: "top", Points: 120, ReachedAt: t0.Add(5 * time.Hour)},
		{UserID: "prev", Points: 200, ReachedAt: t0},
		{UserID: "low", Points: 79, ReachedAt: t0},
		{UserID: "edge", Points: 80, ReachedAt: t0.Add(time.Hour)},
	}
	got := RankStandings(in, map[string]bool{"prev": true}, 80)
	order := []string{}
	for _, s := range got {
		order = append(order, s.UserID)
	}
	wantOrder := []string{"prev", "top", "early", "late", "edge", "low"}
	for i := range wantOrder {
		if order[i] != wantOrder[i] {
			t.Fatalf("الترتيب %v — والمنتظَر %v", order, wantOrder)
		}
	}
	byID := map[string]Standing{}
	for _, s := range got {
		byID[s.UserID] = s
	}
	if p := byID["prev"]; !p.Excluded || p.Rank != 0 || p.Qualified {
		t.Fatalf("**رابحُ الماضي يُرتَّب**: %+v", p)
	}
	if byID["top"].Rank != 1 || byID["early"].Rank != 2 || byID["late"].Rank != 3 {
		t.Fatalf("المراتب: top %d early %d late %d", byID["top"].Rank, byID["early"].Rank, byID["late"].Rank)
	}
	if !byID["edge"].Qualified || byID["low"].Qualified {
		t.Fatal("**الحدّ**: ٨٠ يتأهّل و٧٩ لا")
	}
	w := PickWinners(got, 5)
	if len(w) != 4 || w[0].UserID != "top" || w[1].UserID != "early" || w[3].UserID != "edge" {
		t.Fatalf("الرابحون %v — والمنتظَر top early late edge (لا prev ولا low)", w)
	}
	if w := PickWinners(got, 2); len(w) != 2 || w[1].UserID != "early" {
		t.Fatalf("رابحان: %v", w)
	}
}
