package comms

import "testing"

// TestOffenseCatchesDisguises **الشتيمةُ تُكتب بعشرة أشكال.**
func TestOffenseCatchesDisguises(t *testing.T) {
	cases := []string{
		"يا كلب",
		"يا كــلـب", // تطويل
		"يا كلللب",  // تكرار
		"يا كَلْب",  // تشكيل
		"انت حمار!", // علامةُ ترقيم
		"you are STUPID",
	}
	for _, in := range cases {
		if Offense(in) == "" {
			t.Errorf("مرّت شتيمةٌ متنكّرة: %q", in)
		}
	}
}

// TestOffenseSparesInnocents **ووسمٌ كاذبٌ يُتعب من يراجع** حتّى يكفّ عن
// المراجعة — فيصير الحارسُ سببَ ضياع ما يحرسه.
func TestOffenseSparesInnocents(t *testing.T) {
	cases := []string{
		"كسرت الزجاج",       // «كس» داخلَ كلمة
		"الطلب عند الكلبشة", // «كلب» بادئةً لا كلمة
		"وصلت للحي الغربي",
		"معك ٣٠٠ ليرة",
		"حماره جاهز", // اسمُ محلّ — «حماره» ليست «حمار»
	}
	for _, in := range cases {
		if w := Offense(in); w != "" {
			t.Errorf("وُسم بريء: %q بسبب %q", in, w)
		}
	}
}

// TestSanitizeStripsDeception **محرفٌ لا يُرى يغيّر ما يُرى.**
func TestSanitizeStripsDeception(t *testing.T) {
	// قلبُ الاتّجاه بين حرفين — يُعرض النصُّ معكوسا.
	got := sanitize("سلام" + string(rune(0x202E)) + "عليكم")
	if got != "سلامعليكم" {
		t.Fatalf("لم يُسقط قلبُ الاتّجاه: %q", got)
	}
	// عرضٌ صفريٌّ يقطّع الكلمة فتمرّ من الحارس.
	hidden := "ك" + string(rune(0x200B)) + "لب"
	if Offense(sanitize(hidden)) == "" {
		t.Fatalf("مرّت شتيمةٌ مقطّعةٌ بمحرفٍ صفريّ: %q", hidden)
	}
	// **والسطرُ الجديد يبقى** — وحدَه من محارف التحكّم.
	if sanitize("سطر\nثانٍ") != "سطر\nثانٍ" {
		t.Fatal("أُسقط السطرُ الجديد")
	}
	// **وعشرون سطراً فارغاً ليست رسالة.**
	if sanitize("أ\n\n\n\n\nب") != "أ\n\nب" {
		t.Fatalf("لم تُطوَ الأسطرُ الفارغة: %q", sanitize("أ\n\n\n\n\nب"))
	}
}
