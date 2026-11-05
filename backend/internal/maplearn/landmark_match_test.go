package maplearn

import "testing"

// TestLandmarkNameMatch **«موجود من قبل» بالاسم والعائلة** (قرارُ المالك ٢٠٢٦-١١-٠٥) — جدولُ
// الأمثلة في الخطّة كاملاً، ومعه ما يحمي السائقَ من «مكرّرٍ» ليس مكرّراً.
func TestLandmarkNameMatch(t *testing.T) {
	cases := []struct {
		name       string
		a          string
		famA       string // نوعُ السائق (يمرّ بـDriverFamily)
		b          string
		famB       string // عائلةُ الموجود كما يحسبها الخادم
		driverSide bool   // الموجودُ اقتراحُ سائقٍ: عائلتُه تمرّ بـDriverFamily أيضاً
		want       bool
	}{
		// ── أمثلةُ الخطّة ──
		{"جامع = مسجد", "جامع النور", "mosque", "مسجد النور", "mosque", false, true},
		{"صيدلية غيرُ مشفى", "صيدلية الشفاء", "pharmacy", "مشفى الشفاء", "hospital", false, false},
		{"«غير ذلك» يأخذ نوعَ كلمته", "دوار النعيم", "other", "دوار النعيم", "roundabout", false, true},
		{"ماركت = سوبرماركت", "ماركت الأمل", "shop", "سوبرماركت الأمل", "shop", false, true},
		{"فرنٌ عند جامع", "فرن أبو علي", "shop", "جامع النور", "mosque", false, false},

		// ── التوحيد ──
		{"«ال» على كلمة النوع", "الجامع النور", "mosque", "جامع النور", "mosque", false, true},
		{"التشكيل والهمزة", "صيدلية الشفاء", "pharmacy", "صيدليّة الشفا", "pharmacy", false, true},
		{"الأرقام الهنديّة", "مدرسة ١٥", "school", "مدرسة 15", "school", false, true},
		{"اللاتينيّة بلا حالة", "Cafe Roma", "shop", "cafe roma", "shop", false, true},
		{"سوبر ماركت بكلمتين", "سوبر ماركت الأمل", "shop", "ماركت الأمل", "shop", false, true},
		{"مركزٌ قبل الصيدليّة", "مركز صيدلية الشفاء", "pharmacy", "صيدلية الشفاء", "pharmacy", false, true},

		// ── داخلَ الآخر · حرفٌ واحد ──
		{"اسمٌ داخل الآخر بكلمات", "صيدلية الشفاء", "pharmacy", "صيدلية الشفاء الجديدة", "pharmacy", false, true},
		{"حرفٌ في اسمٍ طويل", "صيدلية الياسمين", "pharmacy", "صيدلية الياسمن", "pharmacy", false, true},
		{"حرفٌ في اسمٍ قصير ليس مكرّراً", "صيدلية النور", "pharmacy", "صيدلية النوري", "pharmacy", false, false},
		{"حرفٌ بلا عائلةٍ واحدة ليس مكرّراً", "الياسمين", "other", "الياسمن", "shop", false, false},

		// ── ما يحمي السائق ──
		{"كلمتا نوعٍ لا تترادفان", "فرن الأمل", "shop", "سوبرماركت الأمل", "shop", false, false},
		{"«محل» عامٌّ يوافق", "محل الأمل", "shop", "سوبرماركت الأمل", "shop", false, true},
		{"«غير ذلك» بالاسم كاملاً", "ملعب البلدية", "other", "ملعب البلدية", "other", false, true},
		{"«غير ذلك» لا يكفيه نصفُ الاسم", "البلدية", "other", "دوار البلدية", "roundabout", false, false},
		{"دوّارٌ غيرُ ملعب", "دوار البلدية", "roundabout", "ملعب البلدية", "roundabout", false, false},
		{"اسمان مختلفان", "جامع النور", "mosque", "جامع الفاروق", "mosque", false, false},
		{"الكلمةُ الثانية اسمٌ لا نوع", "جامع الحديقة 7", "mosque", "جامع الحديقة الرابع 7", "mosque", false, false},
		{"رقمٌ وحدَه لا يُحتوى", "مدرسة 15", "school", "مدرسة الشهيد 15", "school", false, false},

		// ── اقتراحُ سائقٍ آخر: عائلتُه من نوعه وكلمته ──
		{"اقتراحٌ بنوع «غير ذلك» وكلمةِ دوّار", "دوار النعيم", "roundabout", "دوار النعيم", "other", true, true},
	}
	for _, c := range cases {
		ka, kb := LandmarkKeyOf(c.a), LandmarkKeyOf(c.b)
		famB := c.famB
		if c.driverSide {
			famB = DriverFamily(c.famB, kb)
		}
		got := SameLandmark(ka, DriverFamily(c.famA, ka), kb, famB)
		if got != c.want {
			t.Errorf("%s: «%s»(%s) و«%s»(%s) ⇒ %v والمنتظَر %v — %+v · %+v",
				c.name, c.a, c.famA, c.b, c.famB, got, c.want, ka, kb)
		}
		// **والحكمُ متناظر** — من سبق ومن لحق سواء.
		if back := SameLandmark(kb, famB, ka, DriverFamily(c.famA, ka)); back != got {
			t.Errorf("%s: الحكمُ غيرُ متناظر (%v ثمّ %v)", c.name, got, back)
		}
	}

	// **والصنفُ الذي لا يُرسم لا يُقارَن إلّا بالاسم كاملاً** — «النور» ليس «حي النور».
	if SameLandmark(LandmarkKeyOf("النور"), "other", LandmarkKeyOf("حي النور"), "") {
		t.Error("قورن بصنفٍ لا يُرسم احتواءً")
	}
	// **وبالاسم كاملاً بحرفه يُطابَق** (مراجعة ٢٠٢٦-١١-٠٥) — مقبرةٌ أو مكتبٌ على الخريطة بلا صنف.
	for _, c := range []struct{ a, famA, b string }{
		{"مقبرة تل البيعة", "other", "مقبرة تل البيعة"},
		{"مقبرة تل البيعة", "mosque", "مقبرة تل البيعة"},
		{"مكتب الأمانة", "shop", "مكتب الأمانة"},
	} {
		ka := LandmarkKeyOf(c.a)
		if !SameLandmark(ka, DriverFamily(c.famA, ka), LandmarkKeyOf(c.b), "") {
			t.Errorf("«%s»(%s) و«%s» بلا صنف — الاسمُ نفسُه بحرفه ولم يُطابَق", c.a, c.famA, c.b)
		}
	}
	if SameLandmark(LandmarkKeyOf("تل البيعة"), "other", LandmarkKeyOf("مقبرة تل البيعة"), "") {
		t.Error("صنفٌ لا يُرسم طوبق بالاحتواء")
	}
	if SameLandmark(LandmarkKeyOf("مكتب الامانه"), "shop", LandmarkKeyOf("مكتب الامانا"), "") {
		t.Error("صنفٌ لا يُرسم طوبق بحرف")
	}

	// **وما أرسله هو نفسُه بالاسم لا بنوع القائمة** (مراجعة ٢٠٢٦-١١-٠٥).
	own := []struct {
		name    string
		a, famA string
		b, famB string
		want    bool
	}{
		{"الاسمُ نفسُه بنوعٍ آخر", "جامع النور", "mosque", "جامع النور", "shop", true},
		{"مترادفٌ بنوعٍ آخر", "جامع النور", "mosque", "مسجد النور", "shop", true},
		{"بلا كلمة نوع", "جامع النور", "mosque", "النور", "shop", true},
		{"العائلةُ نفسُها", "ماركت الأمل", "shop", "محل الأمل", "shop", true},
		{"محلٌّ باسم الجامع بجانبه", "جامع النور", "mosque", "محل النور", "shop", false},
		{"صيدليّةٌ باسم الجامع", "جامع النور", "mosque", "صيدلية النور", "pharmacy", false},
		{"اسمان مختلفان", "جامع النور", "mosque", "جامع الفاروق", "shop", false},
	}
	for _, c := range own {
		ka, kb := LandmarkKeyOf(c.a), LandmarkKeyOf(c.b)
		if got := SameOwnLandmark(ka, DriverFamily(c.famA, ka), kb, DriverFamily(c.famB, kb)); got != c.want {
			t.Errorf("%s: «%s»(%s) و«%s»(%s) ⇒ %v والمنتظَر %v", c.name, c.a, c.famA, c.b, c.famB, got, c.want)
		}
		if back := SameOwnLandmark(kb, DriverFamily(c.famB, kb), ka, DriverFamily(c.famA, ka)); back != c.want {
			t.Errorf("%s: الحكمُ غيرُ متناظر", c.name)
		}
	}

	// **والمفتاحُ نفسُه**: كلمةُ النوع تُرفع، ويبقى منها أخصُّها.
	k := LandmarkKeyOf("  مركز صيدليّة الشِّفاء - فرع ٢ ")
	if k.Core != "شفا فرع 2" || k.Kind != "pharmacy" || k.Group != "pharmacy" {
		t.Errorf("المفتاح: %+v", k)
	}
	// **وكلمةُ النوع الأولى وحدَها تُرفع** — «جامع الحديقة» اسمُه «الحديقة».
	if g := LandmarkKeyOf("جامع الحديقة"); g.Core != "حديقه" || g.Kind != "mosque" {
		t.Errorf("كلمةُ نوعٍ ثانيةٌ رُفعت: %+v", g)
	}
	if one := LandmarkKeyOf("صيدلية"); one.Core != "صيدليه" || one.Kind != "pharmacy" {
		t.Errorf("كلمةُ النوع وحدَها اسمُها: %+v", one)
	}
	if got := DriverFamily("other", LandmarkKeyOf("ملعب البلدية")); got != "other" {
		t.Errorf("«ملعب» نوعُه «غير ذلك» ⇒ %q", got)
	}
	if got := DriverFamily("mosque", LandmarkKeyOf("صيدلية النور")); got != "mosque" {
		t.Errorf("اختيارُ السائق يغلب في غير «غير ذلك» ⇒ %q", got)
	}
}
