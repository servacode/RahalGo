package server

// دورةُ العرض تُغلق وتُفتح — **فإن دار الطابورُ ولم يأخذه أحد عاد إلى الجميع.**
//
// # الوعدُ المكتوبُ الذي لم يُنفَّذ
//
// `transitions.go` يقول: **«ولا يُحرم منه أبداً: الاستثناءُ لهذه الجولة وحدَها،
// فإن دار الطابورُ ولم يأخذه أحد عاد إليه مع الجميع.»**
// و`OfferNext` يقول: **«لا أحدَ أهلٌ — فليَرَه الجميع.»**
//
// **والطابورُ كان يدور ولا يعود، والجميعُ لا يرونه**: `offer_passed` يُضاف إليه
// في ثلاثة مواضعَ **ولا يُصفَّر في موضعٍ واحدٍ من المشروع كلِّه.**
//
// **فبعتادِ سائقٍ واحدٍ مؤهَّلٍ يموت الطلبُ عند أوّل مهلةٍ تنقضي**: لا يُعرض
// عليه (المحرّكُ يستثني من مرّ) **ولا يراه** (قائمتا التطبيق تستثنيانه كذلك:
// `driver_handlers.go` — `AND NOT ($1 = ANY(o.offer_passed))`).
//
// **وقِيس حيّاً على التجهيز ٢٠٢٦-٠٩-٢٩**: طلبٌ خاصٌّ انقضت مهلتُه فصار يُوسَم
// في كلّ نبضةِ راصدٍ (٣٠ث) — **٩ ثمّ ١٠ ثمّ ١١ مدخلاً، كلُّها المعرّفُ نفسُه
// مكرَّراً**، وحالتُه `dispatching` وصاحبُ عرضِه ثابتٌ. **حلقةٌ لا تنتهي.**

import (
	"context"
	"testing"
)

func (f *driverFixture) expireOffer(t *testing.T, orderID string) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET offer_expires_at = now() - make_interval(secs => 1)
		 WHERE id = $1`, orderID); err != nil {
		t.Fatalf("تعذّر إنهاءُ المهلة: %v", err)
	}
}

// offerState صاحبُ العرض · أحيَّةٌ مهلتُه · كم مرّةً وُسم المرور.
func (f *driverFixture) offerState(t *testing.T, orderID string) (string, bool, int) {
	t.Helper()
	var drv *string
	var live bool
	var passed int
	if err := f.pool.QueryRow(context.Background(), `
		SELECT offered_driver_id,
		       COALESCE(offer_expires_at > now(), false),
		       COALESCE(array_length(offer_passed, 1), 0)
		FROM orders WHERE id = $1`, orderID).Scan(&drv, &live, &passed); err != nil {
		t.Fatalf("تعذّرت قراءةُ حال العرض: %v", err)
	}
	if drv == nil {
		return "", live, passed
	}
	return *drv, live, passed
}

// ── أ · العدلُ الصِّرف: الجولةُ تُصفَّر فيعود العرضُ إلى وحيدِ المؤهَّلين ─────
func TestOfferRound_RestartsWhenEveryonePassed(t *testing.T) {
	f := newDriverFixture(t, 1)
	armRotation(t, f, 60)
	d := f.drivers[0]
	f.onShift(t, d, true)

	ord := f.dispatchingOrder(t, 20_000, 10_000)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), d, "أوّلُ عرض")

	// **تنقضي المهلةُ ولم يردّ** — والكانسُ يوسم مرورَه ثمّ يبحث عن غيره.
	f.expireOffer(t, ord)
	f.srv.orders.SweepExpiredOffers(context.Background())

	drv, live, passed := f.offerState(t, ord)
	if drv != d || !live {
		t.Fatalf("بعد انقضاء المهلة: صاحبُ العرض %q ومهلتُه حيّةٌ=%v — "+
			"**والمتوقّع أن تدور الجولةُ فيعود إليه بمهلةٍ جديدة**", drv, live)
	}
	// **ولا يُوسَم المعرّفُ مرّتين** — الجولةُ الجديدةُ تبدأ نظيفة.
	if passed > 1 {
		t.Fatalf("وُسم المرورُ %d مرّةً — **والوسمُ لا يتكرّر لمعرّفٍ واحد**", passed)
	}
}

// ── ب · ولا ينمو الوسمُ بلا حدٍّ مع كلّ نبضة ─────────────────────────────────
//
// **قِيس حيّاً**: ٣٠ ثانيةً بين نبضتين، **ومدخلٌ جديدٌ في كلّ نبضة** — والطلبُ
// ساكنٌ لا يتقدّم. **وصفٌّ ينمو بلا حدٍّ أثرُه، وموتُ الطلب هو العلّة.**
func TestOfferRound_PassMarkDoesNotGrowUnbounded(t *testing.T) {
	f := newDriverFixture(t, 1)
	armRotation(t, f, 60)
	d := f.drivers[0]
	f.onShift(t, d, true)

	ord := f.dispatchingOrder(t, 20_000, 10_000)
	f.offer(t, ord, nil)

	for i := 0; i < 3; i++ {
		f.expireOffer(t, ord)
		f.srv.orders.SweepExpiredOffers(context.Background())
	}

	drv, _, passed := f.offerState(t, ord)
	if passed > 1 {
		t.Fatalf("بعد ثلاثِ نبضاتٍ: وُسم المرورُ %d مرّةً — **والمتوقّع ≤ ١**", passed)
	}
	if drv != d {
		t.Fatalf("بعد ثلاثِ نبضاتٍ: صاحبُ العرض %q — **والطلبُ لا يموت**", drv)
	}
}

// ── ج · والخاصُّ كذلك: بالقرب مُفعَّلاً وبلا نقطةِ التقاط ────────────────────
//
// **وهي الحالةُ المقيسةُ حيّاً بعينها** — طلبٌ خاصٌّ، والقربُ مُفعَّلٌ، وسائقٌ
// واحدٌ حديثُ الموضع.
func TestOfferRound_CustomOrderSurvivesExpiry(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng)

	ord := f.customOrderAt(t, pdLat, pdLng, "dispatching")
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), d, "أوّلُ عرضٍ لطلبٍ خاصّ")

	f.expireOffer(t, ord)
	f.srv.orders.SweepExpiredOffers(context.Background())

	drv, live, passed := f.offerState(t, ord)
	if drv != d || !live {
		t.Fatalf("الخاصُّ بعد انقضاء المهلة: صاحبُ العرض %q وحيّةٌ=%v "+
			"(وُسم %d) — **والطلبُ لا يموت بانقضاء مهلة**", drv, live, passed)
	}
}
