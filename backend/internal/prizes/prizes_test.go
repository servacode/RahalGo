package prizes

import (
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/offers"
)

// TestEntriesMath **كلُّ خمسٍ فرصة** — ٥←١، ١٠←٢، ١٥←٣، وما دونها صفر.
func TestEntriesMath(t *testing.T) {
	cases := []struct{ invites, per, entries, toNext int }{
		{0, 5, 0, 5}, {4, 5, 0, 1}, {5, 5, 1, 5}, {9, 5, 1, 1},
		{10, 5, 2, 5}, {15, 5, 3, 5}, {7, 3, 2, 2}, {3, 0, 3, 1},
	}
	for _, c := range cases {
		if got := EntriesFor(c.invites, c.per); got != c.entries {
			t.Errorf("EntriesFor(%d, %d) = %d — والمنتظَر %d", c.invites, c.per, got, c.entries)
		}
		if got := ToNext(c.invites, c.per); got != c.toNext {
			t.Errorf("ToNext(%d, %d) = %d — والمنتظَر %d", c.invites, c.per, got, c.toNext)
		}
	}
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

// TestPickWinnersWeightedOncePerPerson **سحبٌ مرجَّحٌ بلا إعادة.**
//
// بعشوائيّةٍ مثبَّتة: الأوّلُ ٣ فرص والثاني ١ والثالث ٢ — `rnd` يردّ صفراً دائماً فيُختار
// أوّلُ من بقي، **ولا يُختار أحدٌ مرّتين ولو طُلب أكثر من المشتركين.**
func TestPickWinnersWeightedOncePerPerson(t *testing.T) {
	pool := []Entrant{{UserID: "a", Entries: 3}, {UserID: "b", Entries: 1}, {UserID: "c", Entries: 2}, {UserID: "z", Entries: 0}}
	got := PickWinners(pool, 5, func(int64) int64 { return 0 })
	if len(got) != 3 {
		t.Fatalf("الرابحون %d — والمنتظَر 3 (لا يُختار ذو صفرٍ ولا أحدٌ مرّتين)", len(got))
	}
	seen := map[string]bool{}
	for _, w := range got {
		if seen[w.UserID] {
			t.Fatalf("**ربح %s مرّتين**", w.UserID)
		}
		seen[w.UserID] = true
	}
	// **والترجيح**: r=3 يقع على الثاني (فرصُ الأوّل ٠..٢).
	got = PickWinners(pool, 1, func(int64) int64 { return 3 })
	if got[0].UserID != "b" {
		t.Fatalf("r=3 اختار %s — والمنتظَر b", got[0].UserID)
	}
	got = PickWinners(pool, 1, func(int64) int64 { return 5 })
	if got[0].UserID != "c" {
		t.Fatalf("r=5 اختار %s — والمنتظَر c", got[0].UserID)
	}

	// **وبالعشوائيّة الحقيقيّة**: صاحبُ ٩٠ فرصةً يربح الأولى أكثرَ بكثير من صاحب ١٠.
	heavy := 0
	for i := 0; i < 2000; i++ {
		w := PickWinners([]Entrant{{UserID: "h", Entries: 90}, {UserID: "l", Entries: 10}}, 1, cryptoRand)
		if w[0].UserID == "h" {
			heavy++
		}
	}
	if heavy < 1600 || heavy > 1950 {
		t.Fatalf("صاحبُ ٩٠٪ ربح %d من 2000 — والترجيحُ معطوب", heavy)
	}
}

// TestMonthMath **الشهرُ بتوقيت دمشق** — السحبُ أوّلَ التالي ١٢ ظهراً، والانتهاءُ آخرُ ثانية.
func TestMonthMath(t *testing.T) {
	// ٢٣:٣٠ بتوقيت غرينتش آخرَ نوفمبر = ٠٢:٣٠ دمشق أوّلَ ديسمبر.
	at := time.Date(2026, 11, 30, 23, 30, 0, 0, time.UTC)
	if got := MonthKey(MonthStart(at)); got != "2026-12" {
		t.Fatalf("الشهر %s — والمنتظَر 2026-12 بتوقيت دمشق", got)
	}
	nov := time.Date(2026, 11, 1, 0, 0, 0, 0, offers.Damascus)
	if got := DrawAt(nov); !got.Equal(time.Date(2026, 12, 1, 12, 0, 0, 0, offers.Damascus)) {
		t.Fatalf("موعدُ السحب %v", got)
	}
	if got := ExpiryFor(time.Date(2026, 12, 1, 12, 0, 0, 0, offers.Damascus)); !got.Equal(time.Date(2026, 12, 31, 23, 59, 59, 0, offers.Damascus)) {
		t.Fatalf("الانتهاء %v", got)
	}
	launch := time.Date(2026, 12, 1, 0, 0, 0, 0, offers.Damascus)
	if _, _, ok := countWindow(nov, launch); ok {
		t.Fatal("**شهرٌ قبل الإطلاق يُعدّ**")
	}
	from, _, ok := countWindow(time.Date(2026, 12, 1, 0, 0, 0, 0, offers.Damascus), time.Date(2026, 12, 10, 0, 0, 0, 0, offers.Damascus))
	if !ok || from.Day() != 10 {
		t.Fatalf("نافذةُ شهر الإطلاق تبدأ %v — والمنتظَر يومُ الإطلاق", from)
	}
}

// TestDisplayName **الاسمُ الأوّلُ وحرفٌ ونقطة — ولا رقم.**
func TestDisplayName(t *testing.T) {
	cases := map[string]string{
		"أحمد محمد العلي": "أحمد ا.",
		"سارة خالد":       "سارة خ.",
		"علي":             "علي",
		"  ":              "زبون",
	}
	for in, want := range cases {
		if got := DisplayName(in); got != want {
			t.Errorf("DisplayName(%q) = %q — والمنتظَر %q", in, got, want)
		}
	}
}
