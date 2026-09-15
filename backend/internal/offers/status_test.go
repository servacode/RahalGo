package offers

// ══════════════════════════════════════════════════════════════════════
// **الحالُ المشتقّة — قرارٌ يُقاس بلا قاعدة** (`OF`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **وحارسٌ يبني صفَّ بياناتٍ بيده ويقرؤه لا يقيس قراراً** — **درسُ
// الدفعة السادسة.** **فالقرارُ هنا دالّةٌ خالصة، والزمنُ يُمرَّر.**

import (
	"testing"
	"time"
)

func at(h int) *time.Time {
	t := time.Now().Add(time.Duration(h) * time.Hour)
	return &t
}

func TestStatusAt_DerivesFromServerTime(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name   string
		active bool
		starts *time.Time
		ends   *time.Time
		want   string
	}{
		{"بلا حدٍّ وفعّالٌ ⇒ سارٍ", true, nil, nil, StatusActive},
		{"بدايتُه غداً ⇒ مجدول", true, at(24), at(48), StatusScheduled},
		{"في مدّته ⇒ سارٍ", true, at(-1), at(1), StatusActive},
		{"مضت نهايتُه ⇒ منتهٍ", true, at(-3), at(-1), StatusExpired},
		{"أُنزل قبل نهايته ⇒ مُنزَل", false, at(-1), at(1), StatusStopped},
		{"أُنزل ولم يبدأ ⇒ مُنزَل", false, at(24), at(48), StatusStopped},
		// **والانتهاءُ يسبق الإنزال** — **ومن أنزل عرضاً بعد انتهائه
		// لم يُنزله: انتهى وحدَه.**
		{"أُنزل بعد انتهائه ⇒ منتهٍ", false, at(-3), at(-1), StatusExpired},
		{"بلا نهايةٍ ومُنزَلٌ ⇒ مُنزَل", false, nil, nil, StatusStopped},
	}
	for _, c := range cases {
		if got := StatusAt(c.active, c.starts, c.ends, now); got != c.want {
			t.Fatalf("%s: قيل %q والمنتظَر %q", c.name, got, c.want)
		}
	}
}

// TestLiveMatchesStatus **وما يُسعَّر به هو ما يُقال عنه «سارٍ»** —
// **ولا حالان: واحدةٌ تُعرض وأخرى تُطبَّق.**
func TestLiveMatchesStatus(t *testing.T) {
	now := time.Now()
	for _, active := range []bool{true, false} {
		for _, s := range []*time.Time{nil, at(-1), at(24)} {
			for _, e := range []*time.Time{nil, at(-1), at(24)} {
				live := LiveAt(active, s, e, now)
				want := StatusAt(active, s, e, now) == StatusActive
				if live != want {
					t.Fatalf("افترقت الحالُ عن السريان: active=%v", active)
				}
			}
		}
	}
}

// TestAfterDiscount_NeverBelowZero **ولا سعرَ سالب.**
func TestAfterDiscount_NeverBelowZero(t *testing.T) {
	if got := AfterDiscount(1000, 90); got != 100 {
		t.Fatalf("خصمُ تسعين: %d", got)
	}
	if got := AfterDiscount(1000, 0); got != 1000 {
		t.Fatalf("بلا خصم: %d", got)
	}
	if got := AfterDiscount(0, 50); got != 0 {
		t.Fatalf("سعرٌ صفر: %d", got)
	}
}
