package itemmatch

import (
	"reflect"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"أرز مصري", []string{"رز", "مصري"}},
		{"رز مصري", []string{"رز", "مصري"}},
		{"الأرز المصري", []string{"رز", "مصري"}},
		{"أَرُزّ", []string{"رز"}},           // التشكيل
		{"شـــاورما", []string{"شاورما"}},    // التطويل
		{"شاورمة", []string{"شاورمه"}},       // ة ← ه
		{"حلوى", []string{"حلوي"}},           // ى ← ي
		{"إسباجيتي", []string{"سباجيتي"}},    // إ ← ا ثمّ الألفُ الأولى
		{"آيس كريم", []string{"يس", "كريم"}}, // آ ← ا
		{"رز-مصري (كيس)", []string{"رز", "مصري", "كيس"}},
		{"  رز    مصري  ", []string{"رز", "مصري"}},
		{"بيبسي ٣٣٠ مل", []string{"بيبسي", "330", "مل"}},
		{"Pepsi Can", []string{"pepsi", "can"}},
		{"ال", []string{"ال"}}, // لا تُمحى
		{"", []string{}},
		{"...", []string{}},
	}
	for _, c := range cases {
		if got := Normalize(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Normalize(%q) = %q — والمتوقّع %q", c.in, got, c.want)
		}
	}
}

func TestScore(t *testing.T) {
	cases := []struct {
		want, have string
		match      bool
		exact      bool // الدرجةُ ١ تماماً
	}{
		// **أمثلةُ المالك**
		{"رز مصري", "أرز مصري", true, true},
		{"رز مصري", "رز مصري حبة طويلة", true, false},
		{"أرز", "رز", true, true},
		{"رز", "بطاطا", false, false},
		// **فروقُ الكتابة**
		{"الأرز المصري", "رز مصري", true, true},
		{"شاورما دجاج", "شاورمة دجاج", true, true},
		{"شاورما دجاج", "شاورمه الدجاج", true, true},
		{"عصير برتقال", "عَصير البُرتقال", true, true},
		{"حلاوة", "حلاوه", true, true},
		{"بيبسي علبة كبيرة", "بيبسي كبيرة", true, false},
		// **وما لا يُطابق** — نصفُ الاسم ليس الاسم
		{"شاورما دجاج", "شاورما لحم", false, false},
		{"رز مصري", "رز بسمتي", false, false},
		{"بيتزا خضار", "بطاطا مقلية", false, false},
		{"لحم", "شحم", false, false}, // كلمتان قصيرتان بينهما حرف
		{"رز مصري حبة طويلة", "رز", false, false},
		{"", "رز", false, false},
		{"رز", "", false, false},
	}
	for _, c := range cases {
		sc := Score(c.want, c.have)
		if got := sc >= Threshold; got != c.match {
			t.Errorf("Score(%q, %q) = %.3f — والمتوقّع مطابقة=%v", c.want, c.have, sc, c.match)
		}
		if c.exact && sc != 1 {
			t.Errorf("Score(%q, %q) = %.3f — والمتوقّع ١", c.want, c.have, sc)
		}
		if !c.exact && sc == 1 {
			t.Errorf("Score(%q, %q) = ١ — ولا تطابقَ تامّاً هنا", c.want, c.have)
		}
	}
}

// **التطابقُ التامُّ يسبق الكلماتِ الزائدة** — «رز مصري» يختار «أرز مصري» قبل «رز مصري حبة طويلة».
func TestBestPrefersExactOverExtraWords(t *testing.T) {
	cands := []Candidate{
		{ID: "a", Name: "رز مصري حبة طويلة", Price: 9000},
		{ID: "b", Name: "أرز مصري", Price: 15000},
		{ID: "c", Name: "بطاطا", Price: 9000},
	}
	i, sc := Best("رز مصري", 9000, cands)
	if i != 1 || sc != 1 {
		t.Fatalf("Best = %d (%.2f) — والمتوقّع «أرز مصري»", i, sc)
	}
}

// **والتعادلُ يُحسم بالسعر الأقرب.**
func TestBestTieBreaksByClosestPrice(t *testing.T) {
	cands := []Candidate{
		{ID: "a", Name: "أرز مصري", Price: 20000},
		{ID: "b", Name: "الرز المصري", Price: 10500},
		{ID: "c", Name: "رز مصري", Price: 7000},
	}
	i, _ := Best("رز مصري", 10000, cands)
	if i != 1 {
		t.Fatalf("Best = %d — والمتوقّعُ الأقربُ سعراً (١)", i)
	}
}

func TestBestNoneAboveThreshold(t *testing.T) {
	cands := []Candidate{
		{ID: "a", Name: "بطاطا", Price: 1},
		{ID: "b", Name: "شاورما لحم", Price: 1},
	}
	if i, _ := Best("شاورما دجاج", 1, cands); i != -1 {
		t.Fatalf("Best = %d — ولا مقابلَ يبلغ الحدّ", i)
	}
	if i, _ := Best("رز", 1, nil); i != -1 {
		t.Fatalf("Best على قائمةٍ فارغة = %d", i)
	}
}
