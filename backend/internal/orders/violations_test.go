package orders

import "testing"

// TestAuthorizingRole من أنهى الطلب — الدورُ الذي خوّله لا الدورُ الأوّل.
//
// **هذا هو الحقلُ الذي يقرّر من يُحظَر.** و`cancelled` يصل إليها الزبونُ
// والمتجرُ والعملياتُ والأدمن — **فلو حُسبت كلُّها على المتجر لحُظر بريءٌ
// بإلغاءِ زبونٍ غيّر رأيه.**
func TestAuthorizingRole(t *testing.T) {
	cases := []struct {
		name     string
		from, to string
		roles    []string
		want     string
	}{
		{"المتجر يلغي بعد قبوله", StAccepted, StCancelled,
			[]string{"merchant"}, "merchant"},
		{"الزبون يلغي في نافذته", StAccepted, StCancelled,
			[]string{"customer"}, "customer"},
		{"العمليات تلغي من الطابور", StDispatching, StCancelled,
			[]string{"ops"}, "ops"},
		{"السائق يُفشل عند باب المطعم", StAtPickup, StFailed,
			[]string{"driver"}, "driver"},
		{"دورٌ لا يخوّل يُعطي فراغاً", StPickedUp, StCancelled,
			[]string{"customer"}, ""},
		// **الدورُ الذي خوّل لا الأوّل في القائمة**: الزبونُ لا يملك الإلغاء
		// من الطابور، والعملياتُ تملكه — فيُنسب لمن ملك.
		{"يُنسب لمن ملك لا لمن سبق", StDispatching, StCancelled,
			[]string{"customer", "ops"}, "ops"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := authorizingRole(KindStandard, c.from, c.to, c.roles); got != c.want {
				t.Errorf("النتيجة %q والمتوقّع %q", got, c.want)
			}
		})
	}
}
