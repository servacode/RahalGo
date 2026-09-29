package server

// الرفضُ وإعادةُ العرض — **المرحلة السابعة، البند الثالث.**
//
// # التوتّرُ الذي أدخلَه إصلاحُ الجولة
//
// **`GAP-DISP-08` أُصلح بتصفير الجولة** حين تدور ولا يأخذها أحد — **وهو
// وعدُ الشيفرة نفسِها**: «ولا يُحرم منه أبداً… فإن دار الطابورُ ولم يأخذه
// أحد عاد إليه مع الجميع».
//
// **والسؤالُ الذي يفتحه**: أيعود الطلبُ إلى **من رفضه صراحةً**؟
//
//	**داخلَ الجولة**  ⇐ **لا** — وثمّة غيرُه، فالعرضُ عليه إهدارُ وقتٍ
//	                      وإغضابُ سائقٍ قال «لا» لتوّه.
//	**بعد دورانها**   ⇐ **نعم** — **وإلّا مات الطلب**، وهو ما أُصلح.
//
// **والفرقُ بينهما كلُّ شيء**: الأوّلُ «دورةٌ لا تتقدّم» (يُعرض على واحدٍ
// حتّى يبرد الطعام)، **والثاني طلبٌ يختفي فلا يصل أحداً.**

import (
	"context"
	"testing"
)

// ── أ · رفضٌ وثمّة غيرُه ⇒ ينتقل ولا يعود ─────────────────────────────
func TestDecline_MovesToNextAndDoesNotReturnWithinRound(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	a, b := f.drivers[0], f.drivers[1]
	f.onShift(t, a, true)
	f.onShift(t, b, true)
	// **كلاهما داخلَ الحلقة وحديثُ الموضع** — فالفرقُ في الرفض لا في الأهليّة.
	f.standAt(t, a, nearLat, nearLng)
	f.standAt(t, b, near2Lat, near2Lng)
	// **وصاحبُ الدور يُحسم بالعدل** — الأقدمُ إسناداً أوّلاً، فلا يتأرجح
	// الاختيارُ بين تشغيلٍ وآخر.
	f.lastAssignedAgo(t, a, 9999)
	f.lastAssignedAgo(t, b, 10)

	ord := f.dispatchingOrder(t, 20_000, 10_000)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), a, "أوّلُ عرضٍ لصاحب الدور")

	if err := f.srv.orders.DeclineOffer(context.Background(), ord, a); err != nil {
		t.Fatalf("الرفض: %v", err)
	}

	// **ينتقل إلى الثاني** — ولا يبقى عند الرافض.
	got := f.offeredDriver(t, ord)
	if got == nil {
		t.Fatalf("رُفض فبقي بلا عرضٍ وثمّة مؤهَّلٌ ثانٍ — **والطلبُ لا يقف على رافض**")
	}
	if *got == a {
		t.Fatalf("عاد العرضُ إلى من رفضه فوراً وثمّة غيرُه — "+
			"**دورةٌ لا تتقدّم**: يُعرض على واحدٍ حتّى يبرد الطعام (driver=%s)", (*got)[:8])
	}
	want(t, got, b, "ينتقل إلى التالي بعد الرفض")
}

// ── ب · ورفضٌ ولا أحدَ غيرُه ⇒ الطلبُ لا يموت ──────────────────────────
//
// **وهو الفرقُ عن السلوك القديم**: كان يُوسَم مارّاً فيختفي من قائمته
// ومن المحرّك معاً — **فيموت الطلبُ بلا أن يعلم به أحد** (`GAP-DISP-08`).
//
// **والصوابُ أن تدور الجولةُ وتُفتح جديدة** — **وغرفةُ العمليّات تُنبَّه**
// (`GAP-WATCH-01` أُصلحت)، فالطلبُ مرئيٌّ لا صامت.
func TestDecline_SoleDriverOrderSurvives(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng)

	ord := f.dispatchingOrder(t, 20_000, 10_000)
	f.offer(t, ord, nil)
	want(t, f.offeredDriver(t, ord), d, "أوّلُ عرض")

	if err := f.srv.orders.DeclineOffer(context.Background(), ord, d); err != nil {
		t.Fatalf("الرفض: %v", err)
	}
	// **والكنسةُ تُدير الجولة** — كما تفعل نبضةُ الراصد كلَّ ثلاثين ثانية.
	f.srv.orders.SweepExpiredOffers(context.Background())

	drv, live, passed := f.offerState(t, ord)
	if drv == "" || !live {
		t.Fatalf("الطلبُ مات بعد رفضِ وحيدِ المؤهَّلين — صاحبُ العرض %q حيٌّ=%v — "+
			"**ولا يُعرض ولا يُرى** (GAP-DISP-08)", drv, live)
	}
	if passed > 1 {
		t.Fatalf("وُسم المرورُ %d مرّةً — **والجولةُ الجديدةُ تبدأ نظيفة**", passed)
	}
}

// ── ج · والرفضُ لا يُسند ولا يُغيّر الحالة ─────────────────────────────
//
// **ورفضٌ يترك الطلبَ `assigned` أسوأُ من رفضٍ لا يقع**: يظنّ المكتبُ أنّ
// له سائقاً، **والسائقُ يظنّ أنّه تخلّص منه.**
func TestDecline_LeavesOrderDispatching(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng)

	ord := f.dispatchingOrder(t, 20_000, 10_000)
	f.offer(t, ord, nil)
	if err := f.srv.orders.DeclineOffer(context.Background(), ord, d); err != nil {
		t.Fatalf("الرفض: %v", err)
	}

	var status string
	var driverID *string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT status, driver_id::text FROM orders WHERE id = $1`, ord).
		Scan(&status, &driverID); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if status != "dispatching" {
		t.Fatalf("الحالةُ بعد الرفض %q — **والمنتظَر dispatching**", status)
	}
	if driverID != nil {
		t.Fatalf("أُسند سائقٌ برفضٍ: %s — **ورفضٌ يُسند عكسُ نفسِه**", (*driverID)[:8])
	}
}
