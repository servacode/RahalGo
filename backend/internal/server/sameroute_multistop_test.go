package server

import (
	"context"
	"testing"
)

// TestSameRoute_MultiStopNeverJoins **الطلبُ من عدّة متاجر لا يُضاف إلى طريقٍ ولا يُضاف إليه** (قرارُ المالك
// ٢٠٢٦-١١-٠٦، ٧أ) — **لا مرساةً ولا مرشّحاً**: وقفاتُه كثيرةٌ أصلاً، والثاني يبرد في الطريق.
func TestSameRoute_MultiStopNeverJoins(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	_, anchor := armSameRoute(t, f)
	fresh := f.orderAt(t, mB1, mB2, dB1, dB2)
	if f.srv.orders.SameRouteDriver(ctx, fresh) == nil {
		t.Fatal("لا مرشّحَ والشروطُ قائمة — الاختبارُ لا يقيس شيئاً")
	}
	setStops := func(id string, n int) {
		t.Helper()
		if _, err := f.pool.Exec(ctx, `UPDATE orders SET stops = $2 WHERE id = $1`, id, n); err != nil {
			t.Fatal(err)
		}
	}
	setStops(fresh, 2)
	if f.srv.orders.SameRouteDriver(ctx, fresh) != nil {
		t.Error("**طلبٌ من متجرين أُضيف إلى طريق سائق**")
	}
	setStops(fresh, 1)
	setStops(anchor, 2)
	if f.srv.orders.SameRouteDriver(ctx, fresh) != nil {
		t.Error("**أُضيف طلبٌ إلى طريق سائقٍ يحمل طلباً من متجرين**")
	}
	setStops(anchor, 1)
}
