package server

// الإسنادُ المباشر — **الطلبُ يصير مهمّتَه بلا سؤال.**
//
// # ما يُثبت هنا
//
//	بالتساوي + إسناد  ←  يقع في مهامّه فوراً، ودورُه ينتقل
//	الأسرع + إسناد     ←  **لا شيء** — لا سائقَ مختاراً هناك أصلاً
//	أُسند ولم يتحرّك    ←  يُنزع منه ويذهب للتالي
//	أُسند ومضى         ←  **لا يُنزع** وهو في الطريق
//
// **والثانيةُ ليست تفصيلاً**: مفتاحٌ بقي مرفوعاً من وضعٍ سابقٍ لا يصنع إسناداً
// في وضعٍ لا يحتمله — **والشاشةُ تُخفي والمحرّكُ يقرأ.**

import (
	"context"
	"testing"
)

// armDirect يرفع الإسنادَ المباشر فوق ترتيبٍ مهيَّأ.
func (f *driverFixture) armDirect(t *testing.T, on bool) {
	t.Helper()
	f.setSetting(t, "drivers.direct_assign", on)
	t.Cleanup(func() { f.setSetting(t, "drivers.direct_assign", false) })
}

// assignedTo صاحبُ الطلبِ وحالتُه كما هما في القاعدة.
func (f *driverFixture) assignedTo(t *testing.T, orderID string) (driverID, status string) {
	t.Helper()
	var d *string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT driver_id::text, status FROM orders WHERE id = $1`, orderID).
		Scan(&d, &status); err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب: %v", err)
	}
	if d != nil {
		driverID = *d
	}
	return driverID, status
}

// TestDirectAssign_LandsInHisTasks **يقع في مهامّه بلا ضغطة.**
func TestDirectAssign_LandsInHisTasks(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	armRotation(t, f, 60)
	f.armDirect(t, true)
	f.onShift(t, f.drivers[0], true)

	orderID := f.dispatchingOrder(t, 48_000, 10_000)
	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("الإسناد فشل: %v", err)
	}

	driver, status := f.assignedTo(t, orderID)
	if driver != f.drivers[0] || status != "assigned" {
		t.Fatalf("driver=%q status=%q — **والمتوقّع أن يصير مهمّتَه فوراً**", driver, status)
	}
	// **ودورُه ينتقل لحظتَها** — فسائقٌ نائمٌ يعطّل طلباً واحداً لا كلَّ الطلبات.
	var moved bool
	if err := f.pool.QueryRow(ctx,
		`SELECT last_assigned_at IS NOT NULL FROM users WHERE id = $1`,
		f.drivers[0]).Scan(&moved); err != nil {
		t.Fatalf("تعذّرت قراءةُ الدور: %v", err)
	}
	if !moved {
		t.Fatal("لم ينتقل دورُه — **فالطلبُ التالي يقع عليه أيضاً وهو لم يردّ على الأوّل**")
	}

	// **ولحظةُ الإسناد تُكتب في سجلّ الطلب.**
	//
	// كان القيدُ يكتب الحالةَ بيده فلا حدثَ يُسجَّل، **فيُقرأ المسارُ
	// `dispatching → at_pickup`** ولا يُعرف متى وصل السائقَ الطلبُ ولا كيف —
	// أخذه بنفسه أم أُسند إليه. **وهو أوّلُ ما يُسأل عنه حين يتأخّر طلب.**
	var actor, note string
	if err := f.pool.QueryRow(ctx, `
		SELECT actor_id::text, note FROM order_events
		WHERE order_id = $1 AND to_status = 'assigned'`, orderID).Scan(&actor, &note); err != nil {
		t.Fatalf("لا حدثَ إسنادٍ في سجلّ الطلب: %v", err)
	}
	if actor != f.drivers[0] {
		t.Fatalf("فاعلُ الحدث %q لا السائق — **والطلبُ صار في يده وهو المسؤولُ عنه**", actor)
	}
	if note == "" {
		t.Fatal("حدثٌ بلا نصّ — **فلا يُفرَّق بين إسنادٍ وقع عليه وأخذٍ اختاره**")
	}

	// **ووقتُ قبول المتجر لا يُمسّ.**
	//
	// كان الإسنادُ يكتب `accepted_at = now()` — **وهو وقتُ قبول المتجر**،
	// فيُقرأ الطلبُ كأنّ المتجرَ قبله لحظةَ نزوله إلى الطابور. **وقياسُ سرعة
	// المتاجر يصير كذباً.**
	var touched bool
	if err := f.pool.QueryRow(ctx,
		`SELECT accepted_at IS NOT NULL FROM orders WHERE id = $1`, orderID).
		Scan(&touched); err != nil {
		t.Fatalf("تعذّرت قراءةُ وقت القبول: %v", err)
	}
	if touched {
		t.Fatal("كُتب وقتُ قبولٍ ولم يقبل متجرٌ شيئاً — **والإسنادُ ليس قبولاً**")
	}
}

// TestDirectAssign_IgnoredInQueueMode **ولا إسنادَ في «الأسرع».**
//
// لا سائقَ مختاراً هناك: الطلبُ ينزل للجميع ومن سبق أخذ. **فمفتاحٌ مرفوعٌ
// من وضعٍ سابقٍ لا يُسند إلى أحد** — ولو فعل لأخذ سائقٌ طلباً لم يره غيرُه
// في وضعٍ كلُّه سباق.
func TestDirectAssign_IgnoredInQueueMode(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	armRotation(t, f, 60)
	f.armDirect(t, true)
	f.onShift(t, f.drivers[0], true)
	// **ثمّ يُبدَّل الوضعُ والمفتاحُ مرفوع** — وهي الحالُ التي تُختبَر.
	f.setSetting(t, "drivers.assignment_mode", "queue")

	orderID := f.dispatchingOrder(t, 48_000, 10_000)
	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("النداء فشل: %v", err)
	}

	driver, status := f.assignedTo(t, orderID)
	if driver != "" || status != "dispatching" {
		t.Fatalf("driver=%q status=%q — **وفي «الأسرع» لا أحدَ يُسنَد إليه**", driver, status)
	}
}

// TestDirectAssign_SilentDriverLosesIt **من لم يتحرّك يُنزع منه.**
func TestDirectAssign_SilentDriverLosesIt(t *testing.T) {
	f := newDriverFixture(t, 2)
	ctx := context.Background()
	armRotation(t, f, 60)
	f.armDirect(t, true)
	for _, d := range f.drivers {
		f.onShift(t, d, true)
	}

	orderID := f.dispatchingOrder(t, 48_000, 10_000)
	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("الإسناد فشل: %v", err)
	}
	first, _ := f.assignedTo(t, orderID)

	// **إنضاجُ المهلة بالقاعدة لا بالانتظار.**
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET offer_expires_at = now() - interval '1 second' WHERE id = $1`,
		orderID); err != nil {
		t.Fatalf("تعذّر إنضاجُ المهلة: %v", err)
	}
	f.srv.orders.SweepExpiredOffers(ctx)

	second, status := f.assignedTo(t, orderID)
	if second == first {
		t.Fatalf("بقي عند الصامت (%s) — **والزبونُ يقف على من لا يعلم أنّ له مهمّة**", first)
	}
	if second == "" || status != "assigned" {
		t.Fatalf("driver=%q status=%q — **ويُنزع لِيُسنَد لا لِيُهمَل**", second, status)
	}
	// **ولا يعود إليه** — وإلّا دار الطلبُ على النائم أبداً.
	var passed []string
	if err := f.pool.QueryRow(ctx,
		`SELECT array(SELECT unnest(offer_passed)::text) FROM orders WHERE id = $1`,
		orderID).Scan(&passed); err != nil {
		t.Fatalf("تعذّرت قراءةُ من مرّ عليهم: %v", err)
	}
	found := false
	for _, p := range passed {
		if p == first {
			found = true
		}
	}
	if !found {
		t.Fatal("لم يُوسَم أنّ الدورَ مرّ على الصامت — **فيعود إليه: هو أطولُ انتظاراً**")
	}
}

// TestDirectAssign_MovingDriverKeepsIt **ومن مضى لا يُنزع منه.**
//
// **نزعُ طلبٍ من سائقٍ صار في الطريق إليه أسوأُ من تركه نائماً**: بضاعةٌ
// تُطلب من متجرٍ مرّتين، وسائقٌ يصل فيجد الطلبَ ليس له.
func TestDirectAssign_MovingDriverKeepsIt(t *testing.T) {
	f := newDriverFixture(t, 2)
	ctx := context.Background()
	armRotation(t, f, 60)
	f.armDirect(t, true)
	for _, d := range f.drivers {
		f.onShift(t, d, true)
	}

	orderID := f.dispatchingOrder(t, 48_000, 10_000)
	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("الإسناد فشل: %v", err)
	}
	first, _ := f.assignedTo(t, orderID)

	// **تحرّك** — خطوةٌ نحو المتجر تُخرجه من `assigned`.
	if _, err := f.srv.orders.Transition(ctx, first, []string{"driver"},
		orderID, "at_pickup", ""); err != nil {
		t.Fatalf("تعذّر التحرّك: %v", err)
	}
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET offer_expires_at = now() - interval '1 second' WHERE id = $1`,
		orderID); err != nil {
		t.Fatalf("تعذّر إنضاجُ المهلة: %v", err)
	}
	f.srv.orders.SweepExpiredOffers(ctx)

	still, status := f.assignedTo(t, orderID)
	if still != first || status != "at_pickup" {
		t.Fatalf("driver=%q status=%q — **ومن مضى إلى المتجر لا يُنزع منه**", still, status)
	}
}
