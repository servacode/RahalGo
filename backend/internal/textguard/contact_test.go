package textguard

import (
	"strings"
	"testing"
)

// **رقمُ الجوال ورابطُ الواتساب وكلمتُه تُخفى** (قرارُ المالك ٢٠٢٦-١١-٠٥) —
// بكلّ شكلٍ يُكتب به: غربيّاً وهنديّاً وفارسيّاً، مفصولاً ومخلوطاً وملتصقاً.
func TestMaskContacts_Masks(t *testing.T) {
	cases := map[string]string{
		"0991234567":          "***",
		"099 123 4567":        "***",
		"099-123-4567":        "***",
		"+963 991 234 567":    "***",
		"00963991234567":      "***",
		"٠٩٩١٢٣٤٥٦٧":          "***",
		"۰۹۹۱۲۳۴۵۶۷":          "***",
		"٠٩٩1234567":          "***",
		"0 9 9 1 2 3 4 5 6 7": "***",
		"رقمي0991234567":      "رقمي***",
		"1234567":             "***",
		"(099) 123-4567":      "(***",
		"099\u200c1234567":    "***",
		"0\uFE0F\u20E39\uFE0F\u20E39\uFE0F\u20E31\uFE0F\u20E32\uFE0F\u20E33\uFE0F\u20E34\uFE0F\u20E35\uFE0F\u20E36\uFE0F\u20E37\uFE0F\u20E3": "***",
		"wa.me/963991234567": "***",
		"https://api.whatsapp.com/send?phone=963991234567": "***",
		"https://wa.me/963991234567 هاد رابطي":             "*** هاد رابطي",
		"واتس":                "***",
		"واتساب":              "***",
		"الواتس":              "***",
		"عالواتس":             "***",
		"بالواتساب":           "***",
		"WhatsApp":            "***",
		"whats app":           "***",
		"واتس اب":             "***",
		"و ا ت س":             "***",
		"وتساب":               "***",
		"كلمني عالواتس بعدين": "كلمني *** بعدين",
		// **مسافةٌ وشَرطةٌ ومسافةٌ فاصلٌ واحد**، والمائلةُ والفاصلةُ تفصلان
		// رقماً سوريّاً كما تفصلان تاريخاً — **والشكلُ هو الحَكَم.**
		"099 - 123 - 4567": "***",
		"0991 - 234 - 567": "***",
		"099 – 123 – 4567": "***",
		"099/123/4567":     "***",
		"099,123,4567":     "***",
		"٠٩٩،١٢٣،٤٥٦٧":     "***",
		"991 234 567":      "***",
		"963 991 234 567":  "***",
		"09.12.34.56.78":   "***",
		"0991.23.45.67":    "***",
		// **وعددٌ قبله لا يحميه** — يُبحث عن الرقم داخل السلسلة.
		"السعر 25 099 123 4567": "السعر 25 ***",
		"0991234567 25":         "*** 25",
	}
	for in, want := range cases {
		got, hits := MaskContacts(in, "***")
		if got != want {
			t.Errorf("MaskContacts(%q) = %q، والمتوقّع %q", in, got, want)
		}
		if len(hits) == 0 {
			t.Errorf("MaskContacts(%q): لا شيءَ مسجّلٌ بما أُخفي", in)
		}
		// **والمُخفى لا يُخفى مرّةً ثانية** — فلا يبقى فيه ما يُمسك.
		if again, h2 := MaskContacts(got, "***"); again != got || len(h2) != 0 {
			t.Errorf("MaskContacts(%q) بعد الإخفاء بقي فيه شيء: %q %v", in, again, h2)
		}
	}
}

// **والعدديُّ العاديّ لا يُمسّ** — سعرٌ وكميّةٌ ورقمُ بناءٍ ووقتٌ وتاريخ.
func TestMaskContacts_LeavesOrdinaryNumbers(t *testing.T) {
	for _, s := range []string{
		"123456",
		"السعر 25000",
		"1,250,000",
		"١٬٢٥٠٬٠٠٠",
		"بناية 12 طابق 3",
		"الطلب #123456",
		"الساعة 10:30",
		"5/11/2026",
		"2 كيلو",
		"شارع 23 شباط",
		"واتسعت",
		"الغرفة واتسخت الكنبة",
		"السعر 25000 والبناية 12",
		"تمام، واصل بعد 5 دقائق",
		// **أسعارٌ ومدى أسعارٍ وتواريخُ وأوقاتٌ بفواصلها** — لا تشبه رقماً سوريّاً.
		"السعر بين 25000-30000",
		"5-11-2026",
		"05.11.2026",
		"05.11.2026 10:30",
		"2026/11/05",
		"من 10.30-11.45",
		"من 09.30-10.45",
		"1.250.000",
		"1 250 000",
		"5000 - 6000",
		"",
	} {
		got, hits := MaskContacts(s, "***")
		if got != s || len(hits) != 0 {
			t.Errorf("MaskContacts(%q) غيّر البريء: %q %v", s, got, hits)
		}
	}
}

// **وما أُخفي يبقى بنصّه الأصليّ للإدارة** — دليلاً لا يراه الطرفُ الآخر.
func TestMaskContacts_KeepsOriginalForAdmin(t *testing.T) {
	got, hits := MaskContacts("رقمي 0991234567 كلمني واتس", "***")
	if got != "رقمي *** كلمني ***" {
		t.Fatalf("الجملةُ بعد الإخفاء: %q", got)
	}
	if len(hits) != 2 {
		t.Fatalf("المُخفى: %v", hits)
	}
	if hits[0].Kind != "phone" || hits[0].Text != "0991234567" {
		t.Errorf("الأوّل: %+v", hits[0])
	}
	if hits[1].Kind != "word" || hits[1].Text != "واتس" {
		t.Errorf("الثاني: %+v", hits[1])
	}
	sum := ContactSummary(hits)
	if sum != "رقم: 0991234567 · كلمة: واتس" {
		t.Errorf("الملخّص: %q", sum)
	}
	if !strings.Contains(ContactSummary([]ContactHit{{Kind: "link", Text: "wa.me/963991234567"}}), "رابط: wa.me/963991234567") {
		t.Errorf("ملخّصُ الرابط لا يحمل الرابط")
	}
	if ContactSummary(nil) != "" {
		t.Errorf("ملخّصُ الفراغ ليس فراغاً")
	}
}

// **تلغرام وفيسبوك وماسنجر وانستغرام والبريد تُخفى كالواتساب** (قرارُ المالك ٢٠٢٦-١١-٠٦) —
// روابطُها وأسماؤها، والمعرّفُ «@…» بعد كلمتها، والبريدُ الإلكترونيّ.
func TestMaskContacts_SocialAndEmail(t *testing.T) {
	cases := map[string]string{
		"t.me/ahmad99":                     "***",
		"https://t.me/ahmad99 راسلني":      "*** راسلني",
		"telegram.me/ahmad":                "***",
		"telegram.dog/ahmad":               "***",
		"تلغرام":                           "***",
		"عالتلغرام":                        "***",
		"تليغرام":                          "***",
		"تيليجرام":                         "***",
		"Telegram":                         "***",
		"facebook.com/ahmad.ali":           "***",
		"https://www.facebook.com/profile": "***",
		"fb.com/ahmad":                     "***",
		"fb.me/ahmad":                      "***",
		"m.me/ahmad":                       "***",
		"messenger.com/t/ahmad":            "***",
		"فيسبوك":                           "***",
		"عالفيسبوك":                        "***",
		"فيس بوك":                          "***",
		"ماسنجر":                           "***",
		"عالماسنجر":                        "***",
		"Messenger":                        "***",
		"Facebook":                         "***",
		"instagram.com/ahmad":              "***",
		"instagr.am/ahmad":                 "***",
		"انستغرام":                         "***",
		"عالانستغرام":                      "***",
		"Instagram":                        "***",
		"ahmad.ali@gmail.com":              "***",
		"ايميلي ahmad_99@yahoo.com":        "ايميلي ***",
		"ضيفني تلغرام @ahmad_99":           "ضيفني *** ***",
		"انستغرام: @ahmad.ali":             "***: ***",
		"كلمني عالفيسبوك اسمي احمد": "كلمني *** اسمي احمد",
	}
	for in, want := range cases {
		got, hits := MaskContacts(in, "***")
		if got != want {
			t.Errorf("MaskContacts(%q) = %q، والمتوقّع %q", in, got, want)
		}
		if len(hits) == 0 {
			t.Errorf("MaskContacts(%q): لا شيءَ مسجّلٌ بما أُخفي", in)
		}
		if again, h2 := MaskContacts(got, "***"); again != got || len(h2) != 0 {
			t.Errorf("MaskContacts(%q) بعد الإخفاء بقي فيه شيء: %q %v", in, again, h2)
		}
	}
}

// **والعاديُّ لا يُمسّ** — «@» بلا كلمة تلغرام/انستغرام قبلها، وكلماتٌ تشبه ولا تكون.
func TestMaskContacts_SocialLeavesOrdinary(t *testing.T) {
	for _, s := range []string{
		"@احمد وصلت",
		"شكراً @ahmad",
		"الفيس تعبان",
		"وجهه فيس",
		"المسنن مكسور",
		"السعر 25000 والبناية 12",
		"عند المطعم m.",
		"حي الجامع، بناية 3",
		"tme",
		"install the app",
	} {
		got, hits := MaskContacts(s, "***")
		if got != s || len(hits) != 0 {
			t.Errorf("MaskContacts(%q) غيّر البريء: %q %v", s, got, hits)
		}
	}
	// **والأصلُ للإدارة** بنوعه: بريدٌ ورابط.
	_, hits := MaskContacts("ahmad@gmail.com t.me/ahmad", "***")
	if len(hits) != 2 || hits[0].Kind != "email" || hits[1].Kind != "link" {
		t.Fatalf("المُخفى: %+v", hits)
	}
	if sum := ContactSummary(hits); !strings.Contains(sum, "بريد: ahmad@gmail.com") || !strings.Contains(sum, "رابط: t.me/ahmad") {
		t.Errorf("الملخّص: %q", sum)
	}
}
