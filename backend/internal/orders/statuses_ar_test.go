package orders

import "testing"

// TestEveryStatusHasArabicName **لا حالةَ تُكتب بالإنكليزيّة في محفظةِ زبون.**
//
// (شهده المالك ٢٠٢٦-٠٨-٠٧: «استرجاع طلب (cancelled) — فيه كلمةٌ إنكليزيّةٌ
// غيرُ مفهومة».)
//
// **و`statusAr` تردّ الحالةَ كما هي حين لا تعرفها** — وذلك صحيحٌ في اللحظة
// (نصٌّ غريبٌ أهونُ من نصٍّ ناقص) **وخطأٌ على المدى**: من أضاف حالةً غداً
// لا يرى شيئاً ينكسر، **ويكتشفها زبونٌ في محفظته.**
//
// فالحارسُ يمرّ على الثوابت كلِّها ويطالب كلَّ واحدةٍ باسمٍ يخالفها.
func TestEveryStatusHasArabicName(t *testing.T) {
	all := []string{
		StPending, StAccepted, StPreparing, StDispatching, StAssigned,
		StAtPickup, StPickedUp, StOnTheWay, StAtDropoff, StDelivered,
		StRejected, StCancelled, StFailed, StRefunded,
	}
	for _, st := range all {
		name := statusAr(st)
		if name == st {
			t.Errorf("الحالة %q بلا اسمٍ عربيّ — أضِفها إلى statusAr", st)
		}
		// **ولا حرفَ لاتينيٍّ في الاسم** — اسمٌ مثل "cancelled ملغى" يمرّ
		// الفحصَ الأوّل ويبقى غيرَ مفهوم.
		for _, r := range name {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				t.Errorf("الحالة %q اسمُها %q فيه حرفٌ لاتينيّ", st, name)
				break
			}
		}
	}
}
