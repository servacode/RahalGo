package maplearn

import (
	"strings"
	"testing"
)

func TestNormalizeUnifiesArabicForms(t *testing.T) {
	a := Normalize("  صَيدليّة   الشِّفاء  ")
	b := Normalize("صيدليه الشفاء")
	if a != "صيدليه الشفاء" || b != a {
		t.Fatalf("التوحيد: %q و %q", a, b)
	}
	if got := Normalize("أإآ ة ى ؤ ئ ٣٤"); got != "ااا ه ي و ي 34" {
		t.Fatalf("الأشكال: %q", got)
	}
}

func TestExtractNames(t *testing.T) {
	cases := []struct {
		text string
		want []string // أسماءٌ معروضة
	}{
		{"جانب صيدلية الشفاء", []string{"صيدلية الشفاء"}},
		{"مقابل فرن الأمل، الطابق الثاني", []string{"فرن الأمل"}},
		{"بجانب مدرسة الشهيد أحمد الطابق 2", []string{"مدرسة الشهيد أحمد"}},
		{"حي الرميلة قرب الجامع الكبير - بيت أبو أحمد", []string{"الجامع الكبير"}},
		{"خلف سوبرماركت النور وعند مخبز السلام", []string{"سوبرماركت النور", "مخبز السلام"}},
		{"قرب البيت الأزرق", nil}, // ليس نوعَ معلم
		{"جانب صيدلية", nil},      // لا اسمَ بعد النوع
		{"شارع 5 بيت 3", nil},     // لا عبارة موضع
		{"قرب جامع، الشفاء", nil}, // الفاصلة تقطع الاسم
		{"جانب صيدلية الشفاء ورقم الهاتف 0933", []string{"صيدلية الشفاء"}}, // «ورقم» لا تدخل، فالاسم كلمتان
	}
	for _, c := range cases {
		hits := ExtractNames(c.text)
		var got []string
		for _, h := range hits {
			got = append(got, h.Display)
		}
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("%q: %v والمتوقَّع %v", c.text, got, c.want)
		}
	}
	h := ExtractNames("جانب صيدلية الشفاء")[0]
	if h.Kind != "pharmacy" || h.Key != "صيدليه شفاء" {
		t.Errorf("النوع والمفتاح: %+v", h)
	}
	// الصيغتان تعطيان المفتاح نفسه.
	if a, b := ExtractNames("جانب صيدلية الشفاء")[0].Key, ExtractNames("بجنب صيدليّه شفاء")[0].Key; a != b {
		t.Errorf("المفتاحان %q و%q", a, b)
	}
}

func TestNameMatches(t *testing.T) {
	key := NameKey("صيدلية الشفاء")
	for _, n := range []string{"صيدلية الشفاء", "صيدليه الشفاء الجديدة", "الصيدلية الشفاء", "مركز صيدلية الشفاء"} {
		if !NameMatches(n, key) {
			t.Errorf("%q يجب أن يطابق %q", n, key)
		}
	}
	for _, n := range []string{"صيدلية النور", "فرن الشفاء", ""} {
		if NameMatches(n, key) {
			t.Errorf("%q لا يجب أن يطابق %q", n, key)
		}
	}
}

func TestClusterNames(t *testing.T) {
	ev := []AddrEvidence{
		{"جانب صيدلية الشفاء", "c1", 35.3000, 38.3000},
		{"بجانب صيدليه الشفاء", "c2", 35.3004, 38.3004}, // ~٦٠ م
		{"قرب صيدلية الشفاء", "c1", 35.3001, 38.3001},   // الزبون نفسُه — لا يزيد العدّ
		{"قرب فرن الأمل", "c3", 35.3100, 38.3100},       // زبونٌ واحد — لا يكفي
		{"جانب صيدلية الشفاء", "c4", 35.4000, 38.4000},  // بعيدٌ جدّاً: نقطة أخرى بزبون واحد
		{"عند مخبز السلام 0933123456", "c5", 35.3200, 38.3200},
		{"مقابل مخبز السلام", "c6", 35.3201, 38.3201},
	}
	got := ClusterNames(ev, DefaultPlaceParams())
	if len(got) != 2 {
		t.Fatalf("المتوقَّع مجموعتان، والناتج %d: %+v", len(got), got)
	}
	var ph, bk *NameCluster
	for i := range got {
		switch got[i].Kind {
		case "pharmacy":
			ph = &got[i]
		case "shop":
			bk = &got[i]
		}
	}
	if ph == nil || ph.Customers != 2 || ph.Evidence != 3 || ph.Name != "صيدلية الشفاء" {
		t.Errorf("الصيدلية: %+v", ph)
	}
	if bk == nil || bk.Customers != 2 || bk.Name != "مخبز السلام" {
		t.Errorf("المخبز: %+v", bk)
	}
	for _, s := range bk.Samples {
		if strings.Contains(s, "0933123456") {
			t.Errorf("رقمٌ طويل في الدليل: %q", s)
		}
	}
}

func TestClusterStops(t *testing.T) {
	var obs []StopObs
	// ثلاثةُ سائقين في يومٍ واحد عند نقطة (مكان أ).
	for _, d := range []string{"a", "b", "c"} {
		obs = append(obs, StopObs{35.5000, 38.5000, d, "2026-10-30"})
	}
	// سائقٌ واحدٌ في ثلاثة أيّام (مكان ب).
	for _, day := range []string{"2026-10-28", "2026-10-29", "2026-10-30"} {
		obs = append(obs, StopObs{35.5100, 38.5100, "a", day})
	}
	// سائقان في يومين (مكان ج): لا يكفي.
	obs = append(obs, StopObs{35.5200, 38.5200, "a", "2026-10-29"}, StopObs{35.5200, 38.5200, "b", "2026-10-30"})
	got := ClusterStops(obs, DefaultPlaceParams())
	if len(got) != 2 {
		t.Fatalf("المتوقَّع مكانان، والناتج %+v", got)
	}
	for _, c := range got {
		if c.Visits != 3 {
			t.Errorf("الزيارات: %+v", c)
		}
	}
}

func TestSanitizeSample(t *testing.T) {
	if got := SanitizeSample("بيت 12 جانب الجامع 0933123456 طابق ٢"); strings.Contains(got, "0933") || !strings.Contains(got, "12") {
		t.Errorf("المسح: %q", got)
	}
	if got := SanitizeSample("اتصل +963933123456 الآن"); strings.Contains(got, "963") {
		t.Errorf("الدولي: %q", got)
	}
}
