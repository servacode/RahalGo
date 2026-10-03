package textguard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// **الحيلُ المعروفةُ لا تمرّ.**
func TestFilter_CatchesEvasions(t *testing.T) {
	f := defaultFilter
	for _, s := range []string{
		"كلب", "يا كلب", "ياكلب", "الكلب", "كلبك",
		"كــلـب", "كَلْب", "كلللللب", "ك ل ب", "ك.ل.ب", "ك-ل-ب",
		"حمار", "حمااار", "7مار", "انت حمار", "أنت حمارة",
		"شرموطة", "شرموطه", "شرمــوطة",
		"kalb", "KALB", "kaaalb", "7mar", "3rs", "sharmouta",
		"fuck", "f u c k", "fuuuck", "FUCKING", "sh1t", "shits",
		"خراء", "منيك", "يلعن", "زبالة",
		"ك\u200bل\u200bب",
	} {
		if f.Find(Clean(s, false)) == "" {
			t.Errorf("مرّ %q", s)
		}
	}
}

// **والبريءُ لا يُمسك** — أسماءُ أماكنَ وأكلاتٍ وكلماتٌ تشبه.
func TestFilter_NoFalsePositives(t *testing.T) {
	f := defaultFilter
	for _, s := range []string{
		"كباب", "شاورما", "حي الثكنة", "دوار النعيم", "شارع تل أبيض", "حي المشلب",
		"كسرت الزجاج", "كسكس", "زبون", "زبيب", "زبدة", "خراب", "إيران", "حيوانات أليفة",
		"متجر حيوانات", "كلبش", "دوار الساعة", "حي الدرعية", "مكسرات", "طيزانة",
		"نيكل", "الزقاق", "بناء 5 ط 3", "شارع 23 شباط", "أمام جامع النور",
		"عرصات", "سمك", "ملوخية", "Dickens street", "classic", "assistant", "shiitake",
		"بيتزا مارغريتا", "فلافل", "كبة", "عصير برتقال",
	} {
		if w := f.Find(Clean(s, false)); w != "" {
			t.Errorf("أُمسك البريءُ %q بـ%q", s, w)
		}
	}
}

func TestClean(t *testing.T) {
	cases := map[string]string{
		"مرحبا\u202eعالم":                 "مرحباعالم",
		"  كثير   من    الفراغ  ":         "كثير من الفراغ",
		"<script>alert(1)</script>شارع":   "alert(1)شارع",
		"<img src=x onerror=alert(1)>بيت": "بيت",
		"هههههههههههههههههههههههههه":      "هههههههههه",
		"!!!!!!!!!!!!!!!!!!!!!!!!":        "!!!!!!!!!!",
		"سطر\nسطر":                        "سطر سطر",
		"\ufeffاسم\u200f":                 "اسم",
		"a\x00b\x07c":                     "abc",
	}
	for in, want := range cases {
		if got := Clean(in, false); got != want {
			t.Errorf("Clean(%q) = %q، والمتوقّع %q", in, got, want)
		}
	}
	if got := Clean("أ\n\n\n\n\nب", true); got != "أ\n\nب" {
		t.Errorf("الأسطرُ الفارغة لم تُطوَ: %q", got)
	}
}

func code(err error) string {
	var e *httpx.AppError
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// **الاسمُ والعنوانُ والملاحظاتُ ترفض — والحديثُ والشكوى تُخفي.**
func TestApply_RejectVsMask(t *testing.T) {
	var g *Guard // **وحارسٌ فارغٌ يعمل بالقائمة المدمجة.**
	ctx := context.Background()

	for _, p := range []Policy{Name, Address, Notes} {
		v := "يا ك ل ب"
		if _, err := g.Apply(ctx, Field{Name: "f", Value: &v, Max: 100, Policy: p}); code(err) != "text_offensive" {
			t.Errorf("السياسة %v: الشتيمةُ لم تُرفض (%v)", p, err)
		}
	}
	for _, p := range []Policy{Chat, Complaint} {
		v := "وصلت متأخر يا حمار 😡"
		masked, err := g.Apply(ctx, Field{Name: "f", Value: &v, Max: 100, Policy: p})
		if err != nil {
			t.Fatalf("السياسة %v: رُفض الحديث: %v", p, err)
		}
		if v != "وصلت متأخر يا *** 😡" || len(masked) != 1 || masked[0] != "حمار" {
			t.Errorf("السياسة %v: %q %v", p, v, masked)
		}
	}
	// **الإيموجي: ممنوعةٌ في الاسم والعنوان، مقبولةٌ في الملاحظات.**
	for _, p := range []Policy{Name, Address} {
		v := "أحمد 🌹"
		if _, err := g.Apply(ctx, Field{Name: "f", Value: &v, Max: 100, Policy: p}); code(err) != "text_bad_chars" {
			t.Errorf("السياسة %v: الإيموجي لم تُرفض", p)
		}
	}
	v := "بدون بصل 🙏"
	if _, err := g.Apply(ctx, Field{Name: "notes", Value: &v, Max: 100, Policy: Notes}); err != nil {
		t.Errorf("الإيموجي رُفضت في الملاحظات: %v", err)
	}
	// **والطولُ قبل طيّ التكرار** — مئةُ حرفٍ مكرَّرٍ لا تصير عشرةً فتُقبل.
	rep := strings.Repeat("ش", 11)
	if _, err := g.Apply(ctx, Field{Name: "f", Value: &rep, Max: 10, Policy: Name}); code(err) != "text_too_long" {
		t.Errorf("المكرَّرُ الطويلُ قُبل بعد طيّه: %v", err)
	}
	long := strings.Repeat("عب", 6)
	if _, err := g.Apply(ctx, Field{Name: "f", Value: &long, Max: 10, Policy: Notes}); code(err) != "text_too_long" {
		t.Errorf("الطويلُ لم يُرفض: %v", err)
	}
	// **والقيمةُ تُنظَّف في موضعها.**
	name := "  <b>متجر</b>   الخير\u200f "
	if _, err := g.Apply(ctx, Field{Name: "name", Value: &name, Max: 100, Policy: Name}); err != nil || name != "متجر الخير" {
		t.Errorf("لم يُنظَّف: %q %v", name, err)
	}
}

// **والقائمةُ من الإعداد** — لفظٌ يُضاف من اللوحة يُمسك، ومحذوفٌ لا يُمسك.
func TestGuard_ListFromSetting(t *testing.T) {
	g := New(func(context.Context) string { return "بطيخ\nkalb" })
	v := "يا بطيخ"
	if _, err := g.Apply(context.Background(), Field{Name: "f", Value: &v, Max: 100, Policy: Notes}); code(err) != "text_offensive" {
		t.Errorf("لفظُ الإعداد لم يُمسك")
	}
	v = "يا حمار"
	if _, err := g.Apply(context.Background(), Field{Name: "f", Value: &v, Max: 100, Policy: Notes}); err != nil {
		t.Errorf("لفظٌ ليس في الإعداد أُمسك: %v", err)
	}
	// **وفارغُ الإعداد يعني المدمجة.**
	g = New(func(context.Context) string { return "  " })
	v = "يا حمار"
	if _, err := g.Apply(context.Background(), Field{Name: "f", Value: &v, Max: 100, Policy: Notes}); code(err) != "text_offensive" {
		t.Errorf("الإعدادُ الفارغ لم يرجع إلى المدمجة")
	}
}
