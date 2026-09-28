package server

// التوزيعُ بالقرب — **الأقربُ المؤهَّلُ أوّلاً، والعدلُ يفصل بين المتقاربين،
// وحلقةٌ تتوسّع، وشبكةُ أمانٍ إن لم يبقَ قريب.** (قرارُ المالك ٢٠٢٦-٠٩-٢٨.)
//
// **ولكلّ شرطٍ اختبارُه على حدة**: القربُ يبدو يعمل بينما يُسند إلى البعيد،
// **ولا يظهر في أيّ خطأ** — كلُّ إسنادٍ ينجح والطلبُ يصل متأخّراً وحسب.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// ── مواضعُ الرقّة للقرب — نقطةٌ للمتجر ومواضعُ سائقين على مسافاتٍ معلومة ──────
const (
	pmLat, pmLng = 35.9500, 39.0000 // المتجر (نقطة الالتقاط)
	pdLat, pdLng = 35.9600, 39.0100 // الزبون (لا يؤثّر في القرب)

	nearLat, nearLng   = 35.9503, 39.0003 // ~٤٠م من المتجر
	near2Lat, near2Lng = 35.9505, 39.0006 // ~٨٠م — نفسُ الشريحة للتعادل
	midLat, midLng     = 35.9560, 39.0060 // ~٨٦٠م — داخلَ الحلقة الأولى، شريحةٌ أبعد
	farLat, farLng     = 35.9700, 39.0300 // ~٣٫٥كم — خارجَ الأولى، داخلَ الأقصى
	outLat, outLng     = 36.0300, 39.1000 // ~١٢٫٦كم — خارجَ الأقصى
	zoutLat, zoutLng   = 35.9600, 39.0080 // ~١٫٣كم — داخلَ الحلقة، خارجَ منطقةِ ١كم
)

// armProximity يُفعّل التوزيعَ بالقرب بأرقامٍ اختباريّةٍ واضحة، **ويُعطّل نفسَ
// المسار** (نصفُ قطره صفرٌ) كي لا يتداخل مع ما يُختبَر هنا.
func armProximity(t *testing.T, f *driverFixture) {
	t.Helper()
	armRotation(t, f, 60)
	f.setSetting(t, "drivers.proximity_enabled", true)
	f.setSetting(t, "drivers.location_fresh_sec", 600)
	f.setSetting(t, "drivers.dispatch_radius_initial_m", 2000)
	f.setSetting(t, "drivers.dispatch_radius_step_m", 2000)
	f.setSetting(t, "drivers.dispatch_radius_max_m", 8000)
	f.setSetting(t, "drivers.proximity_bucket_m", 500)
	f.setSetting(t, "drivers.max_active_orders", 1)   // سقفٌ معلومٌ للاختبار
	f.setSetting(t, "drivers.same_route_radius_m", 0) // لا نفسَ مسارٍ في اختبارات الدور
	// **ولا يُسرَّب `proximity_enabled=false`** (يضبطه اختبارُ الإطفاء) إلى
	// اختباراتٍ لاحقةٍ في القاعدة المشتركة — يُعاد إلى الافتراض بعد كلّ اختبار.
	t.Cleanup(func() { f.setSetting(t, "drivers.proximity_enabled", true) })
}

func (f *driverFixture) offer(t *testing.T, orderID string, skip []string) {
	t.Helper()
	if err := f.srv.orders.OfferNext(context.Background(), orderID, skip); err != nil {
		t.Fatalf("تعذّر العرض: %v", err)
	}
}

func (f *driverFixture) offeredDriver(t *testing.T, orderID string) *string {
	t.Helper()
	var id *string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT offered_driver_id FROM orders WHERE id = $1`, orderID).Scan(&id); err != nil {
		t.Fatalf("تعذّرت قراءة العرض: %v", err)
	}
	return id
}

func (f *driverFixture) dispatchedAgo(t *testing.T, orderID string, secs int) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET dispatched_at = now() - make_interval(secs => $2) WHERE id = $1`,
		orderID, secs); err != nil {
		t.Fatalf("تعذّر ضبط زمن النزول: %v", err)
	}
}

func (f *driverFixture) locationAgo(t *testing.T, driverID string, secs int) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE users SET last_location_at = now() - make_interval(secs => $2) WHERE id = $1`,
		driverID, secs); err != nil {
		t.Fatalf("تعذّر تعتيقُ الموضع: %v", err)
	}
}

func (f *driverFixture) lastAssignedAgo(t *testing.T, driverID string, secs int) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE users SET last_assigned_at = now() - make_interval(secs => $2) WHERE id = $1`,
		driverID, secs); err != nil {
		t.Fatalf("تعذّر ضبط آخرِ إسناد: %v", err)
	}
}

func want(t *testing.T, got *string, expect, label string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s: لا عرضَ والمتوقّع %s", label, expect[:8])
	}
	if *got != expect {
		t.Fatalf("%s: عُرض على %s والمتوقّع %s", label, (*got)[:8], expect[:8])
	}
}

// A — قريبٌ وبعيدٌ مؤهّلان → القريبُ يُفضَّل.
func TestProximity_NearPreferredOverFar(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	near, mid := f.drivers[0], f.drivers[1]
	f.onShift(t, near, true)
	f.onShift(t, mid, true)
	f.standAt(t, near, nearLat, nearLng)
	f.standAt(t, mid, midLat, midLng)
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), near, "الأقرب يُفضَّل")
}

// B — القريبُ مشغولٌ (بلغ سقفَ الطلبات) → يُستبعَد ويُختار التالي المؤهّل.
func TestProximity_NearBusyExcluded(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	near, mid := f.drivers[0], f.drivers[1]
	f.onShift(t, near, true)
	f.onShift(t, mid, true)
	f.standAt(t, near, nearLat, nearLng)
	f.standAt(t, mid, midLat, midLng)
	// **طلبٌ في يد القريب** بمتجرٍ بعيدٍ (فلا نفسَ مسارٍ ولا تداخل) — يبلغ سقفَه.
	busy := f.orderAt(t, outLat, outLng, pdLat, pdLng)
	f.hold(t, busy, near, "assigned")
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), mid, "المشغولُ القريبُ يُستبعَد")
}

// C — القريبُ خارجَ الدوام → يُستبعَد.
func TestProximity_NearOffShiftExcluded(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	near, mid := f.drivers[0], f.drivers[1]
	f.onShift(t, near, false)
	f.onShift(t, mid, true)
	f.standAt(t, near, nearLat, nearLng)
	f.standAt(t, mid, midLat, midLng)
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), mid, "القريبُ خارجَ الدوام يُستبعَد")
}

// D — القريبُ موقوفٌ (غيرُ نشط) → يُستبعَد.
func TestProximity_NearInactiveExcluded(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	near, mid := f.drivers[0], f.drivers[1]
	f.onShift(t, near, true)
	f.onShift(t, mid, true)
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE users SET status = 'suspended' WHERE id = $1`, near); err != nil {
		t.Fatalf("تعذّر إيقافُ القريب: %v", err)
	}
	f.standAt(t, near, nearLat, nearLng)
	f.standAt(t, mid, midLat, midLng)
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), mid, "القريبُ الموقوفُ يُستبعَد")
}

// E — موضعٌ شائخٌ لا يتفوّق على قريبٍ حديث.
func TestProximity_StaleDoesNotOutrankFresh(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	nearStale, midFresh := f.drivers[0], f.drivers[1]
	f.onShift(t, nearStale, true)
	f.onShift(t, midFresh, true)
	f.standAt(t, nearStale, nearLat, nearLng)
	f.standAt(t, midFresh, midLat, midLng)
	// **القريبُ موضعُه شائخٌ (أطولُ من الحداثة ٦٠٠ث)** — لا يُقاس عليه.
	f.locationAgo(t, nearStale, 1200)
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), midFresh, "الشائخُ لا يتفوّق على الحديث")
}

// F — بوّابةُ المنطقة (اختياريّة): من خارجِ منطقةِ الطلب يُستبعَد حين تُشعَل،
// ويُقبل حين تُطفأ.
func TestProximity_ZoneGateWhenEnabled(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	d := f.drivers[0]
	f.onShift(t, d, true)
	// **داخلَ الحلقة (٢٠٠٠) لكن خارجَ منطقةٍ نصفُ قطرها ١٠٠٠.**
	f.standAt(t, d, zoutLat, zoutLng)
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)

	var zoneID string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO delivery_zones (name, center, radius_m, shape, delivery_fee, min_order, active)
		VALUES ('منطقةُ اختبار', ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, 1000,
		        'radius', 1000, 0, true)
		RETURNING id`, pmLat, pmLng).Scan(&zoneID); err != nil {
		t.Fatalf("تعذّر إنشاءُ منطقة: %v", err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM delivery_zones WHERE id = $1`, zoneID) })
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET zone_id = $2 WHERE id = $1`, ord, zoneID); err != nil {
		t.Fatalf("تعذّر ربطُ المنطقة: %v", err)
	}

	// **البوّابةُ مُشعَلة** → السائقُ خارجَ المنطقة يُستبعَد.
	f.setSetting(t, "drivers.zone_gate_enabled", true)
	f.offer(t, ord, nil)
	if o := f.offeredDriver(t, ord); o != nil {
		t.Fatalf("البوّابةُ مشعلة: عُرض على من خارجِ المنطقة %s", (*o)[:8])
	}
	// **البوّابةُ مُطفأة** → القربُ وحدَه، فيُقبل داخلَ الحلقة.
	f.setSetting(t, "drivers.zone_gate_enabled", false)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), d, "البوّابةُ مطفأة: القربُ وحدَه")
}

// G — لا مرشّحَ في الحلقة الأولى → التوسّعُ يقع مع انتظار الطلب.
func TestProximity_RadiusExpands(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	far := f.drivers[0]
	f.onShift(t, far, true)
	f.standAt(t, far, farLat, farLng) // ~٣٫٥كم — خارجَ الأولى (٢٠٠٠)
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)

	// **الآن**: لا أحدَ داخلَ الحلقة الأولى → لا عرض (يُنتظر التوسّع).
	f.offer(t, ord, nil)
	if o := f.offeredDriver(t, ord); o != nil {
		t.Fatalf("قبل التوسّع: عُرض على البعيد %s", (*o)[:8])
	}
	// **بعد انتظارٍ يكفي لتوسّعِ الحلقة** (١٢٠ث ÷ ٦٠ = خطوتان → ٦٠٠٠م).
	f.dispatchedAgo(t, ord, 120)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), far, "التوسّعُ يبلغ البعيد")
}

// H — متقاربان في الشريحة نفسِها → العدلُ يفصل (أطولُ انتظاراً منذ آخرِ إسناد).
func TestProximity_FairnessTieBreak(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	a, b := f.drivers[0], f.drivers[1]
	f.onShift(t, a, true)
	f.onShift(t, b, true)
	// **كلاهما داخلَ شريحةِ ٥٠٠م** (~٤٠م و~٨٠م).
	f.standAt(t, a, nearLat, nearLng)
	f.standAt(t, b, near2Lat, near2Lng)
	// **`a` أُسند إليه للتوّ، و`b` انتظر طويلاً** → العدلُ يختار `b`.
	f.lastAssignedAgo(t, a, 10)
	f.lastAssignedAgo(t, b, 3600)
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), b, "العدلُ يفصل بين المتقاربين")
}

// I — انقضاءُ المهلة → يمرّ الدورُ ويُعرض على المرشّح التالي.
func TestProximity_TimeoutReoffersNext(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	near, mid := f.drivers[0], f.drivers[1]
	f.onShift(t, near, true)
	f.onShift(t, mid, true)
	f.standAt(t, near, nearLat, nearLng)
	f.standAt(t, mid, midLat, midLng)
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), near, "أوّلُ عرضٍ للأقرب")

	// **انقضاءُ المهلة ثمّ الكنس** → يمرّ الدورُ عن القريب ويُعرض على التالي.
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET offer_expires_at = now() - interval '1 second' WHERE id = $1`, ord); err != nil {
		t.Fatalf("تعذّر إنقاءُ المهلة: %v", err)
	}
	f.srv.orders.SweepExpiredOffers(context.Background())
	want(t, f.offeredDriver(t, ord), mid, "بعد الانقضاء يُعرض على التالي")
}

// J — وضعُ «للجميع» (البثّ): يُبثّ للمؤهَّلين القريبين فقط، لا لكلّ المدينة.
func TestProximity_QueueBroadcastIsBounded(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	f.setSetting(t, "drivers.assignment_mode", "queue")
	near, out := f.drivers[0], f.drivers[1]
	f.onShift(t, near, true)
	f.onShift(t, out, true)
	f.standAt(t, near, nearLat, nearLng) // داخلَ الحلقة
	f.standAt(t, out, outLat, outLng)    // ~١٢٫٦كم — خارجَ الأقصى
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)

	if !hasID(f.queueIDs(t, near), ord) {
		t.Fatalf("القريبُ لا يرى طلباً داخلَ حلقته")
	}
	if hasID(f.queueIDs(t, out), ord) {
		t.Fatalf("البعيدُ خارجَ الأقصى يرى الطلبَ — بثٌّ لكلّ المدينة")
	}
}

// K — نفسُ المسار يبقى مقيّداً بالإعداد المركزيّ ولا يلتفّ على الأهليّة.
func TestProximity_SameRouteStillGated(t *testing.T) {
	f := newDriverFixture(t, 1)
	armRotation(t, f, 60)
	f.setSetting(t, "drivers.proximity_enabled", true)
	f.setSetting(t, "drivers.location_fresh_sec", 600)
	driver := f.drivers[0]
	f.onShift(t, driver, true)
	anchor := f.orderAt(t, mA1, mA2, dA1, dA2)
	f.hold(t, anchor, driver, "assigned")
	f.standAt(t, driver, 35.9500, 39.0060) // قبل متجره

	// **`SameRouteDriver` قراءةٌ لا تُسنِد** — فلا يتلوّث المرساةُ بين الحالتين.
	ord := f.orderAt(t, mB1, mB2, dB1, dB2)

	// **بمدىً كافٍ (٨٠٠م)** — طلبٌ في طريقه يجد مرشّحاً.
	f.setSetting(t, "drivers.same_route_radius_m", 800)
	if f.srv.orders.SameRouteDriver(context.Background(), ord) == nil {
		t.Fatalf("نفسُ المسار: لا مرشّحَ رغم توفّر الشروط عند ٨٠٠م")
	}

	// **وبمدىً ضيّقٍ (١٠٠م) أقلَّ من المسافة بين المتجرين** — لا مرشّح.
	f.setSetting(t, "drivers.same_route_radius_m", 100)
	if f.srv.orders.SameRouteDriver(context.Background(), ord) != nil {
		t.Fatalf("نفسُ المسار: وُجد مرشّحٌ رغم ضيقِ المدى المركزيّ")
	}
}

// L1 — لا قريبٌ حديث ونصفُ القطر بلغ أقصاه ويوجد شائخٌ فقط → **لا عرض** (لا يُهبَط
// إلى الشائخ، يُنتظر تحديثُ الموضع).
func TestProximity_StaleOnlyAtMaxGetsNoOffer(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng) // قريبٌ بالمكان…
	f.locationAgo(t, d, 1200)         // …لكنّ موضعَه شائخ
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	f.dispatchedAgo(t, ord, 600) // انتظارٌ يبلغ أقصى نصف القطر
	f.offer(t, ord, nil)
	if o := f.offeredDriver(t, ord); o != nil {
		t.Fatalf("عُرض على شائخِ الموضع عند الأقصى %s — والمتوقّع لا عرض", (*o)[:8])
	}
}

// L2 — بعد أن يصير موضعُ ذاك السائق حديثاً → يصير مؤهَّلاً.
func TestProximity_StaleBecomesFreshThenEligible(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng)
	f.locationAgo(t, d, 1200)
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	f.dispatchedAgo(t, ord, 600)
	f.offer(t, ord, nil)
	if o := f.offeredDriver(t, ord); o != nil {
		t.Fatalf("لا ينبغي عرضٌ والموضعُ شائخ: %s", (*o)[:8])
	}
	// **يتحدّث الموضع** → يصير مؤهَّلاً.
	f.standAt(t, d, nearLat, nearLng) // last_location_at = now()
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), d, "بعد تحديث الموضع يصير مؤهَّلاً")
}

// L3 — إطفاءُ القرب صراحةً → العدلُ الأعمى عن الموضع يعمل عمداً (يقبل الشائخ).
func TestProximity_DisabledFallsBackToLegacy(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	f.setSetting(t, "drivers.proximity_enabled", false) // قرارُ المشغّل الصريح
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng)
	f.locationAgo(t, d, 1200) // شائخ
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), d, "القربُ مطفأً: العدلُ يقبل الشائخ عمداً")
}

// L4 — الطابور: السائقُ الشائخُ الموضع لا يرى الطلب؛ وحين يتحدّث موضعُه يراه.
func TestProximity_QueueHidesStaleCaller(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	f.setSetting(t, "drivers.assignment_mode", "queue")
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng)
	f.locationAgo(t, d, 1200) // شائخ
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	if hasID(f.queueIDs(t, d), ord) {
		t.Fatalf("الطابور: شائخُ الموضع رأى الطلب")
	}
	f.standAt(t, d, nearLat, nearLng) // يتحدّث
	if !hasID(f.queueIDs(t, d), ord) {
		t.Fatalf("الطابور: حديثُ الموضع لم يرَ الطلب داخلَ حلقته")
	}
}

// ── أدواتُ الطابور ───────────────────────────────────────────────────────
func (f *driverFixture) queueIDs(t *testing.T, driverID string) []string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/driver/queue", nil)
	rc := chi.NewRouteContext()
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, driverID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})
	w := httptest.NewRecorder()
	f.srv.handleDriverQueue(w, req.WithContext(ctx))
	if w.Code != http.StatusOK {
		t.Fatalf("الطابور ردَّ %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("ردُّ الطابور غيرُ صالح: %s", w.Body.String())
	}
	ids := make([]string, 0, len(body.Data))
	for _, o := range body.Data {
		ids = append(ids, o.ID)
	}
	return ids
}

func hasID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}
