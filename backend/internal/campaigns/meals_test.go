package campaigns

import (
	"testing"
	"time"
)

// **إشعاراتُ الوجبات** (قرارُ المالك ٢٠٢٦-١٠-٠٥): تُرسَل في ساعتها بتوقيت دمشق،
// **ولا تُلحَق بعد ساعةٍ من موعدها، وتدور نصوصُها يوماً بعد يوم.**
func TestMealDueWindow(t *testing.T) {
	m := Meal{Key: "lunch", At: "13:30"}
	at := func(h, min int) time.Time { // ساعةُ دمشق ← العالميّ
		return time.Date(2026, 10, 6, h, min, 0, 0, damascus).UTC()
	}
	for _, tc := range []struct {
		h, m int
		due  bool
	}{
		{13, 29, false}, {13, 30, true}, {14, 29, true}, {14, 30, false}, {20, 0, false},
	} {
		day, due := MealDue(m, at(tc.h, tc.m))
		if due != tc.due {
			t.Fatalf("%02d:%02d ⇒ %v والمنتظَرُ %v", tc.h, tc.m, due, tc.due)
		}
		if due && day != "2026-10-06" {
			t.Fatalf("اليومُ %q لا 2026-10-06", day)
		}
	}
	if _, due := MealDue(Meal{At: ""}, at(13, 30)); due {
		t.Fatal("موعدٌ فارغٌ وأُرسلت الوجبة")
	}
	if _, due := MealDue(Meal{At: "25:00"}, at(1, 0)); due {
		t.Fatal("موعدٌ لا يُفهم وأُرسلت الوجبة")
	}
}

func TestMealTextRotatesAndSplits(t *testing.T) {
	texts := "أوّل | نصٌّ أوّل\n\nثانٍ\n"
	t1, b1, ok1 := MealText(texts, "2026-10-06")
	t2, _, ok2 := MealText(texts, "2026-10-07")
	if !ok1 || !ok2 || t1 == t2 {
		t.Fatalf("لم تَدُر النصوص: %q / %q", t1, t2)
	}
	if t1 == "أوّل" && b1 != "نصٌّ أوّل" {
		t.Fatalf("«العنوان | النصّ» لم ينقسم: %q %q", t1, b1)
	}
	if _, _, ok := MealText("  \n ", "2026-10-06"); ok {
		t.Fatal("نصوصٌ فارغةٌ وأُرسل شيء")
	}
}
