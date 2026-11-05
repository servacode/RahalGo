package server

// ══════════════════════════════════════════════════════════════════════
// **تدويرُ السائقين على الزبون نفسِه** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٤)
// ══════════════════════════════════════════════════════════════════════
//
// **سائقٌ وصّل لزبونٍ حدَّ الأسبوع (افتراضاً ٣) يأتي آخرَ الصفّ لطلبه التالي** —
// في كلّ طرق التوزيع: القربُ والعدلُ والإسنادُ المباشرُ والرفضُ ونفسُ المسار
// و«للجميع». **وتفضيلٌ لا منع**: إن لم يكن غيرُه أخذه، **فلا طلبَ يبقى بلا سائق.**

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// armSameCustomer يضبط حدَّ الأسبوع — **ويعيده إلى افتراضه بعد الاختبار** (القاعدةُ مشتركة).
func armSameCustomer(t *testing.T, f *driverFixture, capN int) {
	t.Helper()
	f.setSetting(t, "drivers.same_customer_weekly_max", capN)
	t.Cleanup(func() { f.setSetting(t, "drivers.same_customer_weekly_max", 3) })
}

func (f *driverFixture) setCustomer(t *testing.T, orderID, customer string) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET customer_id = $2 WHERE id = $1`, orderID, customer); err != nil {
		t.Fatalf("تعذّر ضبطُ الزبون: %v", err)
	}
}

// seedDelivered **n طلباتٍ مسلَّمةٍ من السائق للزبون** قبل `daysAgo` يوماً.
func (f *driverFixture) seedDelivered(t *testing.T, customer, driver string, n, daysAgo int) {
	t.Helper()
	for i := 0; i < n; i++ {
		id := f.dispatchingOrder(t, 10_000, 2_000)
		if _, err := f.pool.Exec(context.Background(), `
			UPDATE orders SET customer_id = $2, driver_id = $3, status = 'delivered',
			       driver_fee = delivery_fee,
			       accepted_at = now() - make_interval(days => $4) - interval '1 hour',
			       picked_up_at = now() - make_interval(days => $4) - interval '30 minutes',
			       delivered_at = now() - make_interval(days => $4),
			       closed_at = now() - make_interval(days => $4)
			WHERE id = $1`, id, customer, driver, daysAgo); err != nil {
			t.Fatalf("تعذّر بذرُ طلبٍ مسلَّم: %v", err)
		}
	}
}

// scProximity **سائقان: A على ٤٠م وB على ٨٦٠م من المتجر** — والقربُ يختار A.
func scProximity(t *testing.T, f *driverFixture) (a, b, ord, customer string) {
	t.Helper()
	armProximity(t, f)
	a, b = f.drivers[0], f.drivers[1]
	f.onShift(t, a, true)
	f.onShift(t, b, true)
	f.standAt(t, a, nearLat, nearLng)
	f.standAt(t, b, midLat, midLng)
	ord = f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	return a, b, ord, f.customerOf(t, ord)
}

// ProximityPrefersOther — **A الأقربُ وصّل للزبون ٣ مرّاتٍ هذا الأسبوع ⇒ العرضُ لـB.**
func TestSameCustomer_ProximityPrefersOther(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, b, ord, x := scProximity(t, f)
	armSameCustomer(t, f, 3)
	f.seedDelivered(t, x, a, 3, 1)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), b, "فوق الحدّ آخرُ الصفّ")
}

// FallsBackWhenAlone — **ولا غيرُه على الدوام ⇒ يأخذه هو.**
func TestSameCustomer_FallsBackWhenAlone(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, b, ord, x := scProximity(t, f)
	armSameCustomer(t, f, 3)
	f.onShift(t, b, false)
	f.seedDelivered(t, x, a, 3, 1)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), a, "لا بديل — لا يبقى بلا سائق")
}

// OldDeliveriesDontCount — **وما قبل سبعة أيّامٍ لا يُعدّ.**
func TestSameCustomer_OldDeliveriesDontCount(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, _, ord, x := scProximity(t, f)
	armSameCustomer(t, f, 3)
	f.seedDelivered(t, x, a, 3, 8)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), a, "تسليماتٌ قديمةٌ لا تُعدّ")
}

// BelowCap — **وتسليمتان تحت الحدّ.**
func TestSameCustomer_BelowCap(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, _, ord, x := scProximity(t, f)
	armSameCustomer(t, f, 3)
	f.seedDelivered(t, x, a, 2, 1)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), a, "تحت الحدّ")
}

// CapZeroOff — **وصفرُ الإعداد يطفئ القاعدة.**
func TestSameCustomer_CapZeroOff(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, _, ord, x := scProximity(t, f)
	armSameCustomer(t, f, 0)
	f.seedDelivered(t, x, a, 5, 1)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), a, "الحدّ صفر — مطفأ")
}

// DeclineGoesBackToSame — **B رفض ⇒ يعود إلى A** (لا يبقى بلا سائق).
func TestSameCustomer_DeclineGoesBackToSame(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, b, ord, x := scProximity(t, f)
	armSameCustomer(t, f, 3)
	f.seedDelivered(t, x, a, 3, 1)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), b, "أوّلُ عرضٍ لغير صاحب الزبون")
	if err := f.srv.orders.DeclineOffer(context.Background(), ord, b); err != nil {
		t.Fatalf("الرفض: %v", err)
	}
	want(t, f.offeredDriver(t, ord), a, "بعد رفض B")
}

// LegacyFairnessPrefersOther — **والعدلُ الصِّرف (القربُ مطفأ) كذلك.**
func TestSameCustomer_LegacyFairnessPrefersOther(t *testing.T) {
	f := newDriverFixture(t, 2)
	armRotation(t, f, 60)
	armSameCustomer(t, f, 3)
	f.setSetting(t, "drivers.same_route_radius_m", 0)
	a, b := f.drivers[0], f.drivers[1]
	f.onShift(t, a, true)
	f.onShift(t, b, true)
	// **A أطولُ انتظاراً** — فالعدلُ يختاره لولا الحدّ.
	f.lastAssignedAgo(t, a, 3600)
	f.lastAssignedAgo(t, b, 60)
	ord := f.dispatchingOrder(t, 10_000, 2_000)
	f.seedDelivered(t, f.customerOf(t, ord), a, 3, 1)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), b, "العدلُ الصِّرف يفضّل غيرَ صاحب الزبون")
}

// DirectAssignPrefersOther — **والإسنادُ المباشرُ يقع في يد B.**
func TestSameCustomer_DirectAssignPrefersOther(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, b, ord, x := scProximity(t, f)
	armSameCustomer(t, f, 3)
	f.setSetting(t, "drivers.direct_assign", true)
	t.Cleanup(func() { f.setSetting(t, "drivers.direct_assign", false) })
	f.seedDelivered(t, x, a, 3, 1)
	f.offer(t, ord, nil)
	got, status := f.assignedTo(t, ord)
	if got != b || status != "assigned" {
		t.Fatalf("driver=%q status=%q — والمتوقّع إسنادُه إلى B", got, status)
	}
}

// ── نفسُ المسار ─────────────────────────────────────────────────────────

// SameRouteSkipsWhenAlternative — **صاحبُ المسار فوق الحدّ وB فارغٌ ⇒ لا نفسَ مسار، والعرضُ لـB.**
func TestSameCustomer_SameRouteSkipsWhenAlternative(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, _ := armSameRoute(t, f)
	armSameCustomer(t, f, 3)
	b := f.drivers[1]
	f.onShift(t, b, true)
	fresh := f.orderAt(t, mB1, mB2, dB1, dB2)
	f.seedDelivered(t, f.customerOf(t, fresh), a, 3, 1)
	if f.srv.orders.TrySameRoute(context.Background(), fresh) {
		t.Fatal("**أُسند لصاحب المسار وهو فوق حدّ الزبون وثمّة بديل**")
	}
	f.offer(t, fresh, nil)
	want(t, f.offeredDriver(t, fresh), b, "العرضُ للبديل")
}

// SameRouteKeepsWhenAlone — **ولا بديلَ ⇒ نفسُ المسار كما كان.**
func TestSameCustomer_SameRouteKeepsWhenAlone(t *testing.T) {
	f := newDriverFixture(t, 1)
	a, _ := armSameRoute(t, f)
	armSameCustomer(t, f, 3)
	fresh := f.orderAt(t, mB1, mB2, dB1, dB2)
	f.seedDelivered(t, f.customerOf(t, fresh), a, 3, 1)
	if !f.srv.orders.TrySameRoute(context.Background(), fresh) {
		t.Fatal("**لا بديلَ ولم يُسنَد بنفس المسار** — الطلبُ ينتظر بلا سبب")
	}
	if got, _ := f.assignedTo(t, fresh); got != a {
		t.Fatalf("أُسند إلى %q والمتوقّع صاحبُ المسار", got)
	}
}

// SameRouteSameDoorKeeps — **وطلبُه الذي بيده للزبون نفسِه ⇒ مشوارٌ واحد.**
func TestSameCustomer_SameRouteSameDoorKeeps(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, anchor := armSameRoute(t, f)
	armSameCustomer(t, f, 3)
	f.onShift(t, f.drivers[1], true)
	fresh := f.orderAt(t, mB1, mB2, dB1, dB2)
	x := f.customerOf(t, fresh)
	f.setCustomer(t, anchor, x)
	f.seedDelivered(t, x, a, 3, 1)
	if !f.srv.orders.TrySameRoute(context.Background(), fresh) {
		t.Fatal("**البابُ نفسُه ولم يُسنَد بنفس المسار** — زيارتان بدل زيارة")
	}
	if got, _ := f.assignedTo(t, fresh); got != a {
		t.Fatalf("أُسند إلى %q والمتوقّع صاحبُ المسار", got)
	}
}

// ── «للجميع» ────────────────────────────────────────────────────────────

func scQueue(t *testing.T, f *driverFixture) (a, b, ord, customer string) {
	t.Helper()
	a, b, ord, customer = scProximity(t, f)
	f.setSetting(t, "drivers.assignment_mode", "queue")
	armSameCustomer(t, f, 3)
	return a, b, ord, customer
}

// QueueHiddenDuringGrace — **A فوق الحدّ لا يرى الطلبَ أوّلَ المهلة وB يراه، وقبولُه يُردّ.**
func TestSameCustomer_QueueHiddenDuringGrace(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, b, ord, x := scQueue(t, f)
	f.seedDelivered(t, x, a, 3, 1)
	if hasID(f.queueIDs(t, a), ord) {
		t.Fatal("**A فوق الحدّ رأى الطلبَ في المهلة وB يراه**")
	}
	if !hasID(f.queueIDs(t, b), ord) {
		t.Fatal("B لم يرَ الطلب")
	}
	w := f.accept(a, ord)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "order_taken") {
		t.Fatalf("**قبولُ A مرّ في المهلة**: %d %s", w.Code, w.Body.String())
	}
}

// QueueVisibleWhenAlone — **ولا غيرُه ⇒ يراه.**
func TestSameCustomer_QueueVisibleWhenAlone(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, b, ord, x := scQueue(t, f)
	f.onShift(t, b, false)
	f.seedDelivered(t, x, a, 3, 1)
	if !hasID(f.queueIDs(t, a), ord) {
		t.Fatal("**لا سائقَ غيرُه والطلبُ محجوبٌ عنه** — يبقى بلا سائق")
	}
}

// QueueVisibleAfterGrace — **وبعد المهلة يراه.**
func TestSameCustomer_QueueVisibleAfterGrace(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, _, ord, x := scQueue(t, f)
	f.seedDelivered(t, x, a, 3, 1)
	f.dispatchedAgo(t, ord, 61) // المهلة ٦٠ (armRotation)
	if !hasID(f.queueIDs(t, a), ord) {
		t.Fatal("**انقضت المهلةُ والطلبُ ما زال محجوباً**")
	}
}

// QueueMerchantDeliveryUnaffected — **و«لدي توصيلة» بلا زبونٍ لا حدَّ عليها.**
func TestSameCustomer_QueueMerchantDeliveryUnaffected(t *testing.T) {
	f := newDriverFixture(t, 2)
	a, b, ord, x := scQueue(t, f)
	f.seedDelivered(t, x, a, 3, 1)
	if _, err := f.pool.Exec(context.Background(), `
		UPDATE orders SET kind = 'merchant_delivery', customer_id = NULL,
		       recipient_name = 'مستلم', recipient_phone = '0991234567', fee_payer = 'merchant'
		WHERE id = $1`, ord); err != nil {
		t.Fatalf("تحويلُه إلى توصيلة: %v", err)
	}
	if !hasID(f.queueIDs(t, a), ord) || !hasID(f.queueIDs(t, b), ord) {
		t.Fatal("**توصيلةُ متجرٍ بلا زبونٍ حُجبت**")
	}
}
