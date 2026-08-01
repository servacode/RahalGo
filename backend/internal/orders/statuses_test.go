package orders

import "testing"

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to string
		roles    []string
		want     bool
	}{
		{StPending, StAccepted, []string{"merchant"}, true},
		{StPending, StAccepted, []string{"ops"}, true},
		{StPending, StAccepted, []string{"customer"}, false},
		{StPending, StCancelled, []string{"customer"}, true},
		{StPending, StDelivered, []string{"admin"}, false}, // قفز غير معرف حتى للأدمن
		{StAccepted, StPreparing, []string{"merchant"}, true},
		{StPreparing, StDispatching, []string{"ops"}, true},
		{StPreparing, StDispatching, []string{"merchant"}, false},
		// **مقبول ← بانتظار سائق**: طريقُ وضع «المنصة تدير»، حيث لا يضغط
		// المتجرُ «بدء التحضير» لأنه خارج النظام أصلاً.
		{StAccepted, StDispatching, []string{"ops"}, true},
		// **وللعمليات وحدها.** لو ملكه المتجر لاستدعى سائقاً متى شاء —
		// فيقف عند بابه ينتظر طعاماً لم يبدأ فيه، وأجرتُه تجري على المنصة.
		{StAccepted, StDispatching, []string{"merchant"}, false},
		{StAccepted, StDispatching, []string{"driver"}, false},
		{StDispatching, StAssigned, []string{"driver"}, true},
		{StAtDropoff, StDelivered, []string{"driver"}, true},
		{StAtDropoff, StFailed, []string{"driver"}, true},
		{StDelivered, StRefunded, []string{"admin"}, true},
		{StDelivered, StRefunded, []string{"ops"}, false},
		{StDelivered, StPending, []string{"admin"}, false},
		{StCancelled, StAccepted, []string{"admin"}, false}, // لا عودة من نهائية
	}
	for _, c := range cases {
		if got := canTransition(c.from, c.to, c.roles); got != c.want {
			t.Errorf("canTransition(%s→%s, %v) = %v, want %v", c.from, c.to, c.roles, got, c.want)
		}
	}
}

// TestAutoDispatchNeedsSettings الإنزالُ التلقائيّ بلا إعداداتٍ لا يقع.
//
// **محرّكُ التسويات يُبنى في الاختبارات بلا مخزن إعدادات** — فلو قرأ
// `autoDispatch` من مخزنٍ فارغ لانهار كلُّ اختبارِ حساب. والأهمُّ: **بلا مخزنٍ
// لا نعرف رغبةَ المالك، والصمتُ لا يُقرأ موافقة.**
func TestAutoDispatchNeedsSettings(t *testing.T) {
	s := &Service{}
	for _, to := range []string{StPreparing, StAccepted, StDispatching} {
		if s.autoDispatch(t.Context(), to) {
			t.Errorf("autoDispatch(%s) بلا إعدادات = true، والمتوقّع false", to)
		}
	}
	if s.AutoDispatchEnabled(t.Context()) {
		t.Error("AutoDispatchEnabled بلا إعدادات = true، والمتوقّع false")
	}
}

func TestTerminal(t *testing.T) {
	for _, st := range []string{StDelivered, StRejected, StCancelled, StFailed, StRefunded} {
		if !terminal(st) {
			t.Errorf("terminal(%s) should be true", st)
		}
	}
	for _, st := range []string{StPending, StPreparing, StOnTheWay} {
		if terminal(st) {
			t.Errorf("terminal(%s) should be false", st)
		}
	}
}
