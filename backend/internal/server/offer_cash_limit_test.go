package server

// **سقفُ النقد قاعدةٌ واحدة — والعرضُ والقبولُ يقرآنها معاً.**
//
// # الحادثة
//
// كانت مكتوبةً في موضعين بصيغتين:
//
//	العرض   (rotation.go)          held < limit
//	القبول  (driver_handlers.go)   held + cash_due > limit  →  يُردّ
//
// **فيُعرض الطلبُ على من لا يستطيع أخذَه**: سائقٌ حوزتُه صفرٌ مؤهَّلٌ للعرض،
// وطلبٌ نقدُه فوق السقف يُردّ عند الضغط. **ويدور العرضُ على الجميع بمهلته
// كاملةً** — ٤٥ ثانيةً لكلٍّ — ثمّ يسقط إلى «لا أحد».
//
// **والعملياتُ ترى «جارٍ إسناد سائق» وتنتظر من لن يأتي.**
//
// **ووقع أمام المالك** (٢٠٢٦-٠٨-٠٣): الطلب `#1002` نقدُه ٦٢٦٬٠٠٠ وسقفُ السائق
// ٥٠٠٬٠٠٠ — **عُرض على بشارٍ وحوزتُه صفر.**
//
// # ولماذا لم يُمسك
//
// **لأنّ الاثنين يعملان**: العرضُ يعرض، والقبولُ يردّ — **وكلٌّ منهما صحيحٌ
// وحدَه.** والخللُ في أنّهما لا يتّفقان، **وهو ما لا يراه اختبارٌ يفحص أحدَهما.**
//
// وهي عائلةُ `R-18` نفسُها: **قائمةٌ مكتوبةٌ مرّتين تفترق بلا صوت.**

import (
	"context"
	"testing"
)

// TestOffer_SkipsDriverWhoCannotAccept **لا يُعرض ما سيُردّ.**
func TestOffer_SkipsDriverWhoCannotAccept(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	driverID := f.drivers[0]
	// **armRotation لا `setSetting` وحدَه**: بلا `SetSettings` تبقى إعداداتُ
	// خدمة الطلبات فارغةً، **فيقرأ النمطُ «الأسرع» ويعود `OfferNext` بلا عرض**
	// — فينجح الاختبارُ لأنّ شيئاً لم يقع، لا لأنّ القاعدة صحيحة. **وعزلُ
	// السائقين الغرباء منه أيضاً.**
	armRotation(t, f, 10)
	f.onShift(t, driverID, true)
	f.setSetting(t, "drivers.cash_limit", 500_000)

	// **طلبٌ نقدُه فوق السقف وحدَه** — والسائقُ حوزتُه صفر، فهو «مؤهَّل»
	// بالقاعدة القديمة.
	orderID := f.dispatchingOrder(t, 600_000, 26_000)

	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("OfferNext: %v", err)
	}

	var offered *string
	if err := f.pool.QueryRow(ctx,
		`SELECT offered_driver_id::text FROM orders WHERE id = $1`, orderID).
		Scan(&offered); err != nil {
		t.Fatalf("قراءة العرض: %v", err)
	}
	if offered != nil {
		t.Fatalf("عُرض على سائقٍ لا يستطيع أخذَه — **والقبولُ سيردّه**، " +
			"ويدور العرضُ بمهلته كاملةً والعملياتُ تنتظر من لن يأتي")
	}

	// **وبرهانُ التناقض**: لو أُخذ لَرُدّ. فالعرضُ كان يَعِد بما يمنعه القبول.
	if code := f.accept(driverID, orderID).Code; code < 400 {
		t.Errorf("القبولُ مرّ %d — والسقفُ يجب أن يمنعه", code)
	}
}

// TestOffer_AllowsWhatFitsUnderLimit **وما يتّسع له يُعرض.**
//
// **حراسةٌ زائدةٌ أسوأُ من ناقصة**: شرطٌ يمنع ما يجوز يُجوّع السائقين ويُبقي
// الطلبات بلا حامل، **ولا يظهر إلّا في طابورٍ لا يتحرّك.**
func TestOffer_AllowsWhatFitsUnderLimit(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	driverID := f.drivers[0]
	// **armRotation لا `setSetting` وحدَه**: بلا `SetSettings` تبقى إعداداتُ
	// خدمة الطلبات فارغةً، **فيقرأ النمطُ «الأسرع» ويعود `OfferNext` بلا عرض**
	// — فينجح الاختبارُ لأنّ شيئاً لم يقع، لا لأنّ القاعدة صحيحة. **وعزلُ
	// السائقين الغرباء منه أيضاً.**
	armRotation(t, f, 10)
	f.onShift(t, driverID, true)
	f.setSetting(t, "drivers.cash_limit", 500_000)

	orderID := f.dispatchingOrder(t, 50_000, 8_000) // ٥٨٬٠٠٠ — دون السقف

	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("OfferNext: %v", err)
	}
	var offered *string
	if err := f.pool.QueryRow(ctx,
		`SELECT offered_driver_id::text FROM orders WHERE id = $1`, orderID).
		Scan(&offered); err != nil {
		t.Fatalf("قراءة العرض: %v", err)
	}
	if offered == nil || *offered != driverID {
		t.Fatal("لم يُعرض طلبٌ يتّسع له السقف — **والحراسةُ الزائدة تُجوّع الطابور**")
	}
}
