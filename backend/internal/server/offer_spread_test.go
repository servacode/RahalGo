package server

// **طلبٌ لكلّ سائق — لا ثلاثةٌ لواحد.**
//
// # الحادثة
//
// كلُّ طلبٍ كان يختار «أطولَ انتظاراً» **مستقلاًّ عن الآخر، ولا يعلم أنّ عرضاً
// حيّاً عند ذاك السائق**. فثلاثةُ طلباتٍ تُحوَّل معاً وثلاثةُ سائقين في الدوام
// **تقع كلُّها على الأوّل**: يراها الثلاثةَ في شاشته، **والاثنان الآخران
// شاشتاهما فارغة.**
//
// ثمّ تنقضي مهلتُه على الثلاثة **فتنتقل كلُّها معاً إلى الثاني**، ثمّ إلى
// الثالث — **دورةٌ كاملةٌ تُهدر وثلاثةُ زبائنَ ينتظرون**، والطلبُ الذي كان
// يمكن أن ينطلق في الثانية الأولى ينطلق بعد دقيقتين.
//
// **ووقع أمام المالك** (٢٠٢٦-٠٨-٠٣): «المفروض الآن يوجد ٣ سائقين، الطلب
// الأوّل يذهب للأوّل والثاني للثاني والثالث للثالث».
//
// # ولماذا لم يُمسك
//
// **لأنّ كلَّ عرضٍ صحيحٌ وحدَه**: اختار أطولَ السائقين انتظاراً، وهو ما طُلب
// منه. **والخللُ لا يظهر إلّا حين يُنظر إلى الثلاثة معاً** — وهو ما لا يفعله
// اختبارٌ يفحص طلباً واحداً.

import (
	"context"
	"testing"
)

// TestOffers_SpreadAcrossDrivers **ثلاثةُ طلباتٍ ← ثلاثةُ سائقين.**
func TestOffers_SpreadAcrossDrivers(t *testing.T) {
	f := newDriverFixture(t, 3)
	ctx := context.Background()
	armRotation(t, f, 60) // مهلةٌ طويلة: **العروضُ حيّةٌ كلَّ الاختبار**
	for _, d := range f.drivers {
		f.onShift(t, d, true)
	}

	orders := []string{
		f.dispatchingOrder(t, 40_000, 8_000),
		f.dispatchingOrder(t, 50_000, 8_000),
		f.dispatchingOrder(t, 60_000, 8_000),
	}

	got := map[string]string{} // طلب ← سائق
	for _, id := range orders {
		if err := f.srv.orders.OfferNext(ctx, id, nil); err != nil {
			t.Fatalf("OfferNext: %v", err)
		}
		var offered *string
		if err := f.pool.QueryRow(ctx,
			`SELECT offered_driver_id::text FROM orders WHERE id = $1`, id).Scan(&offered); err != nil {
			t.Fatalf("قراءة العرض: %v", err)
		}
		if offered == nil {
			t.Fatalf("طلبٌ بلا عرضٍ وثلاثةُ سائقين في الدوام")
		}
		got[id] = *offered
	}

	// **ولا يتكرّر سائق** — وهو كلُّ المقصود.
	seen := map[string]int{}
	for _, d := range got {
		seen[d]++
	}
	if len(seen) != 3 {
		t.Fatalf("توزّعت الثلاثةُ على %d سائقٍ لا ٣ — **وقعت على واحدٍ والباقون شاشاتُهم فارغة**\n"+
			"التوزيع: %v", len(seen), seen)
	}
}

// TestOffer_WaitingPickedUpWhenDriverFrees **وما ينتظر لا يُهمَل.**
//
// **العرضُ الحيُّ واحدٌ لكلّ سائق**، فطلبٌ رابعٌ يأتي وكلُّهم مشغولون **لا يجد
// أحداً فيبقى بلا عرض** — ولا شيءَ يوقظه: الكانسُ كان ينظر إلى العروض
// المنقضية وحدَها، **وهذا لا عرضَ له أصلاً.**
//
// **وطلبٌ ينتظر بصمتٍ أسوأُ من طلبٍ يُرفض.**
func TestOffer_WaitingPickedUpWhenDriverFrees(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	armRotation(t, f, 60)
	driverID := f.drivers[0]
	f.onShift(t, driverID, true)

	// **العزل**: قاعدةُ الاختبار مشتركة، وفيها طلباتُ تجاربَ سابقة معلّقة في
	// `dispatching` — **وهي أقدمُ من طلبي فتسبقه إلى السائق**، فيسقط الاختبارُ
	// بسببٍ لا يخصّ ما يُختبَر. (وهو عزلُ السائقين نفسُه في `armRotation`.)
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET status = 'cancelled', closed_at = now()
		 WHERE status = 'dispatching'`); err != nil {
		t.Fatalf("تعذّر عزلُ الطلبات القديمة: %v", err)
	}

	first := f.dispatchingOrder(t, 40_000, 8_000)
	waiting := f.dispatchingOrder(t, 30_000, 8_000)

	if err := f.srv.orders.OfferNext(ctx, first, nil); err != nil {
		t.Fatalf("العرض الأوّل: %v", err)
	}
	// **الثاني لا يجد أحداً** — السائقُ الوحيدُ عنده عرضٌ حيّ.
	if err := f.srv.orders.OfferNext(ctx, waiting, nil); err != nil {
		t.Fatalf("العرض الثاني: %v", err)
	}
	var offered *string
	if err := f.pool.QueryRow(ctx,
		`SELECT offered_driver_id::text FROM orders WHERE id = $1`, waiting).Scan(&offered); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	if offered != nil {
		t.Fatal("عُرض ثانٍ على سائقٍ عنده عرضٌ حيّ — **قراران في شاشةٍ واحدة**")
	}

	// **يتحرّر السائقُ فيأخذه الكانس.**
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET offered_driver_id = NULL, offer_expires_at = NULL,
		 status = 'cancelled', closed_at = now() WHERE id = $1`, first); err != nil {
		t.Fatalf("تحرير الأوّل: %v", err)
	}
	f.srv.orders.SweepExpiredOffers(ctx)

	if err := f.pool.QueryRow(ctx,
		`SELECT offered_driver_id::text FROM orders WHERE id = $1`, waiting).Scan(&offered); err != nil {
		t.Fatalf("قراءة بعد الكنس: %v", err)
	}
	if offered == nil {
		t.Fatal("بقي المنتظِرُ بلا عرضٍ بعد أن تحرّر السائق — **وطلبٌ ينتظر بصمتٍ لا يوقظه أحد**")
	}
}
