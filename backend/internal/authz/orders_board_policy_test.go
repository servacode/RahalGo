package authz

import "testing"

// TestOrdersBoardPolicy **قراراتُ المالك ٢٠٢٦-١٠-٠٤ — قسمُ «الطلبات» في الجدول.**
//
//	حسمُ البضاعة للعمليّات · وتعويضُ المتجر للماليّة بتأكيد   البندان ١٢ و١٣
//	إعادةُ حساب التسوية للمالك والأدمن بتأكيد                البند ١٥
//	الطوارئُ بقدرتها — تبلغها العمليّاتُ                     البند ٧
//	مرشّحو الإسناد وعدّاداتُ اللوحة                          البندان ٣ و١٠
func TestOrdersBoardPolicy(t *testing.T) {
	for _, c := range []struct {
		method, pattern string
		want            Capability
	}{
		{"POST", "/orders/{}/goods", OrdersIntervene},
		{"POST", "/orders/{}/goods/compensation", FinanceManage},
		{"POST", "/orders/{}/recompute", FinanceRecompute},
		{"GET", "/emergencies", EmergenciesManage},
		{"POST", "/emergencies/{}/resolve", EmergenciesManage},
		{"GET", "/emergencies/banner", EmergenciesManage},
		{"POST", "/emergencies/{}/ack", EmergenciesManage},
		{"GET", "/orders/{}/assign-candidates", OrdersIntervene},
		{"GET", "/orders/board", OrdersRead},
	} {
		got, ok := LookupAdmin(c.method, c.pattern)
		if !ok || got != c.want {
			t.Errorf("%s %s ⇒ %q (%v) والمتوقّع %q", c.method, c.pattern, got, ok, c.want)
		}
	}
	for _, c := range []struct{ method, pattern string }{
		{"POST", "/orders/{}/goods/compensation"},
		{"POST", "/orders/{}/recompute"},
	} {
		if _, ok := LookupSensitive(c.method, c.pattern); !ok {
			t.Errorf("**%s %s بلا تأكيد كلمة السرّ** — مالٌ يتحرّك بضغطة", c.method, c.pattern)
		}
	}
	if s, _ := LookupSensitive("POST", "/orders/{}/goods/compensation"); len(s.Params) == 0 || s.Params[0] != "amount" {
		t.Errorf("بصمةُ التعويض لا تحمل المبلغ — «أكّدتُ خمسين» تصير إذناً بخمسِ مئة")
	}
	for _, c := range []Capability{EmergenciesManage, FinanceRecompute} {
		if !Known(c) {
			t.Errorf("القدرةُ %q ليست في المعجم", c)
		}
	}
}
