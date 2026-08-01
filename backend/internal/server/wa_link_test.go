package server

import (
	"net/url"
	"strings"
	"testing"
)

// TestWALink الرابطُ يفتح المحادثةَ الصحيحة بالنصّ الصحيح.
//
// **الرقمُ سوريٌّ والنصُّ عربيٌّ متعدّدُ الأسطر** — وكلاهما يكسر رابطاً بُني
// بالتسلسل. و`wa.me` تريده دولياً بلا `+` ولا صفرٍ بادئ: صفرٌ باقٍ يفتح
// محادثةً لرقمٍ لا وجود له، **ويظنّ الموظّفُ أنه حوّل الطلب وقد فتح صفحةَ
// خطأ**.
func TestWALink(t *testing.T) {
	const text = "طلب جديد #1002\n——————————————\n• سلطة خضراء ×2"

	got := waLink("0223344556", text)
	const wantPrefix = "https://wa.me/963223344556?text="
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("الرابط = %q، والمتوقّع أن يبدأ بـ%q", got, wantPrefix)
	}

	// **النصُّ يعود كما دخل** — لا سطرٌ ضاع ولا حرفٌ تشوّه.
	decoded, err := url.QueryUnescape(strings.TrimPrefix(got, wantPrefix))
	if err != nil {
		t.Fatalf("فكُّ الترميز تعذّر: %v", err)
	}
	if decoded != text {
		t.Errorf("النصُّ بعد الترميز اختلف:\nوصل: %q\nأُريد: %q", decoded, text)
	}

	// **رقمٌ لا يُفهم يعطي رابطاً فارغاً لا رابطاً مكسوراً.** والواجهة تُخفي
	// الزرَّ عند الفراغ — فلا يُعرض زرٌّ يفتح صفحةَ خطأ.
	for _, bad := range []string{"", "123", "لا رقم"} {
		if got := waLink(bad, text); got != "" {
			t.Errorf("waLink(%q) = %q، والمتوقّع فراغ", bad, got)
		}
	}
}
