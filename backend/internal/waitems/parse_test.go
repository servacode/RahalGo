package waitems

import "testing"

func TestParseCaption(t *testing.T) {
	cases := []struct {
		in        string
		name      string
		price     float64
		has       bool
		ambiguous bool
		cur       string
	}{
		{"جبنة بيضاء 25000", "جبنة بيضاء", 25000, true, false, ""},
		{"25000 جبنة بيضاء", "جبنة بيضاء", 25000, true, false, ""},
		{"جبنة بيضاء ٢٥٠٠٠", "جبنة بيضاء", 25000, true, false, ""},
		{"٢٥ ألف جبنة", "جبنة", 25000, true, false, ""},
		{"جبنة 25 الف", "جبنة", 25000, true, false, ""},
		{"جبنة ٢٥ الاف", "جبنة", 25000, true, false, ""},
		{"جبنة 25,000", "جبنة", 25000, true, false, ""},
		{"جبنة ٢٥٬٠٠٠", "جبنة", 25000, true, false, ""},
		{"جبنة 25.000", "جبنة", 25000, true, false, ""},
		{"جبنة 25", "جبنة", 25, true, false, ""},
		{"جبنة ٢٥", "جبنة", 25, true, false, ""},
		{"2.5 ألف جبنة", "جبنة", 2500, true, false, ""},
		{"جبنة 25k", "جبنة", 25000, true, false, ""},
		{"قميص 15$", "قميص", 15, true, false, "USD"},
		{"قميص $15", "قميص", 15, true, false, "USD"},
		{"قميص 15 $", "قميص", 15, true, false, "USD"},
		{"قميص 15 دولار", "قميص", 15, true, false, "USD"},
		{"قميص ١٥ دولار", "قميص", 15, true, false, "USD"},
		{"قميص 2.5$", "قميص", 2.5, true, false, "USD"},
		{"قميص 2.500$", "قميص", 2.5, true, false, "USD"},
		{"قميص ٢٫٥ دولار", "قميص", 2.5, true, false, "USD"},
		{"قميص 15000 ليرة", "قميص", 15000, true, false, "SYP"},
		{"قميص 15000 ل.س", "قميص", 15000, true, false, "SYP"},
		{"سعر 5000 عصير برتقال", "عصير برتقال", 5000, true, false, ""},
		{"عصير برتقال بسعر 5000", "عصير برتقال", 5000, true, false, ""},
		{"عصير برتقال ب5000", "عصير برتقال", 5000, true, false, ""},
		{"عصير برتقال بـ 5000", "عصير برتقال", 5000, true, false, ""},
		{"عصير 250 مل 5000", "عصير 250 مل", 5000, true, false, ""},
		{"عصير 250مل 5000", "عصير 250مل", 5000, true, false, ""},
		{"بيبسي 2 لتر 8000", "بيبسي 2 لتر", 8000, true, false, ""},
		{"شوكولا 3 قطع 1500", "شوكولا 3 قطع", 1500, true, false, ""},
		{"جبنة", "جبنة", 0, false, false, ""},
		{"جبنة بيضاء نابلسية", "جبنة بيضاء نابلسية", 0, false, false, ""},
		{"بيبسي 2 لتر", "بيبسي 2 لتر", 0, false, false, ""},
		{"25000", "", 25000, true, false, ""},
		{"٢٥ ألف", "", 25000, true, false, ""},
		{"5$", "", 5, true, false, "USD"},
		{"", "", 0, false, false, ""},
		{"   ", "", 0, false, false, ""},
		{"بيبسي 330 5000", "بيبسي", 0, false, true, ""},
		{"قميص 40 15$", "قميص 40", 15, true, false, "USD"},
		{"جبنة - 25000", "جبنة", 25000, true, false, ""},
		{"جبنة: 25000", "جبنة", 25000, true, false, ""},
		{"جبنة\n25000", "جبنة", 25000, true, false, ""},
		{"جبنة 0", "جبنة", 0, false, false, ""},
		{"كولا ٥ الاف ل.س", "كولا", 5000, true, false, "SYP"},
		{"آيفون 15 ب 500$", "آيفون 15", 500, true, false, "USD"},
	}
	for _, c := range cases {
		got := ParseCaption(c.in)
		if got.Name != c.name || got.HasPrice != c.has || got.Ambiguous != c.ambiguous || got.Currency != c.cur ||
			(c.has && got.Price != c.price) {
			t.Errorf("ParseCaption(%q) = %+v ؛ المتوقع name=%q price=%v has=%v amb=%v cur=%q",
				c.in, got, c.name, c.price, c.has, c.ambiguous, c.cur)
		}
	}
}

func TestToASCIIDigits(t *testing.T) {
	if got := ToASCIIDigits("٠١٢٣٤٥٦٧٨٩ ۰۱۲"); got != "0123456789 012" {
		t.Fatalf("got %q", got)
	}
}

func TestAnswerWords(t *testing.T) {
	for _, s := range []string{"نعم", "ايه", "أيوه", "اكيد", "yes", " نعم! ", "ايوا"} {
		if !isYes(s) {
			t.Errorf("%q يجب أن تكون نعم", s)
		}
	}
	for _, s := range []string{"لا", "لأ", "no", "غلط"} {
		if !isNo(s) {
			t.Errorf("%q يجب أن تكون لا", s)
		}
	}
	if n, ok := pickNumber("٢"); !ok || n != 2 {
		t.Errorf("pickNumber ٢ = %d %v", n, ok)
	}
	if _, ok := pickNumber("اتنين"); ok {
		t.Errorf("نصّ ليس رقماً")
	}
}
