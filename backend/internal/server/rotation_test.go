package server

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// armRotation يحقن مخزنَ الإعدادات في محرّك الطلبات ويضبط النمط.
//
// **العُدّةُ تبني المحرّكَ بلا إعدادات عمداً** (انظر رأس driver_test.go):
// اختباراتُ التسويات لا تحتاج مخزناً لتفحص حساباً. **والترتيبُ يحتاجه** —
// وبلا هذا السطر يقرأ النمطَ «الأسرع» فلا يُعرض شيء، **ويمرّ الاختبار كاذباً
// لو لم يفحص أن العرض وقع.**
func armRotation(t *testing.T, f *driverFixture, timeoutSec int) {
	t.Helper()
	f.srv.orders.SetSettings(settings.NewStore(f.pool))
	// **العزل**: قاعدةُ الاختبار مشتركة، وسائقو تجاربَ سابقة يبقون فيها.
	// وترتيبُ الدور يختار من كلّ سائقٍ على الدوام — **فيُعرض على غريبٍ
	// ويسقط الاختبار بسببٍ لا يخصّ ما يُختبَر.**
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE users SET on_shift = false WHERE NOT (id = ANY($1::uuid[]))`,
		f.drivers); err != nil {
		t.Fatalf("تعذّر عزلُ السائقين: %v", err)
	}
	f.setSetting(t, "drivers.assignment_mode", "rotation")
	if timeoutSec > 0 {
		f.setSetting(t, "drivers.offer_timeout_sec", timeoutSec)
	}
	t.Cleanup(func() {
		f.setSetting(t, "drivers.assignment_mode", "queue")
	})
}

// TestRotation_TurnPassesAndNeverReturns الدورُ ينتقل ولا يعود إلى من مرّ عليه.
//
// **هذا هو العطبُ الذي يقتل الترتيب صامتاً.** الدورُ يُقاس بأطولِ انتظار، ومن
// مرّ عليه ولم يأخذ **يبقى أطولَ انتظاراً** — فيعود إليه العرضُ فوراً، ثم
// ينقضي، ثم يعود. **فيدور العرضُ على واحدٍ حتى يبرد الطعام**، ولا يظهر خطأٌ
// في أيّ سجلّ: كلُّ نداءٍ ينجح، والطلبُ لا يصل أحداً.
func TestRotation_TurnPassesAndNeverReturns(t *testing.T) {
	f := newDriverFixture(t, 3)
	ctx := context.Background()
	armRotation(t, f, 10)
	for _, d := range f.drivers {
		f.onShift(t, d, true)
	}

	orderID := f.dispatchingOrder(t, 48_000, 10_000)
	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("أوّل عرض فشل: %v", err)
	}

	seen := map[string]bool{}
	for round := 1; round <= 3; round++ {
		var offered *string
		if err := f.pool.QueryRow(ctx,
			`SELECT offered_driver_id::text FROM orders WHERE id = $1`, orderID).
			Scan(&offered); err != nil {
			t.Fatalf("تعذّرت قراءة العرض: %v", err)
		}
		if offered == nil {
			t.Fatalf("الجولة %d: لا عرضَ وثمّة سائقون مؤهّلون", round)
		}
		if seen[*offered] {
			t.Fatalf("الجولة %d: عاد الدورُ إلى من مرّ عليه — %s", round, *offered)
		}
		seen[*offered] = true

		// **إنضاجُ المهلة بالقاعدة لا بالانتظار**: اختبارٌ ينام ٤٥ ثانية
		// اختبارٌ لا يُشغَّل.
		if _, err := f.pool.Exec(ctx,
			`UPDATE orders SET offer_expires_at = now() - interval '1 second'
			 WHERE id = $1`, orderID); err != nil {
			t.Fatalf("تعذّر إنضاج المهلة: %v", err)
		}
		f.srv.orders.SweepExpiredOffers(ctx)
	}

	if len(seen) != 3 {
		t.Errorf("عُرض على %d سائقين والمتوقّع 3", len(seen))
	}
	// **وبعد الجميع يعود مشاعاً** — لا يبقى محجوزاً لمن لا يستطيع.
	var offered *string
	if err := f.pool.QueryRow(ctx,
		`SELECT offered_driver_id::text FROM orders WHERE id = $1`, orderID).
		Scan(&offered); err != nil {
		t.Fatalf("تعذّرت قراءة العرض: %v", err)
	}
	if offered != nil {
		t.Errorf("بقي محجوزاً بعد مرور الجميع: %s", *offered)
	}
}

// TestRotation_OffShiftNotOffered لا يُعرض على منصرفٍ عن الدوام.
//
// **عرضٌ على من لا يستطيع عرضٌ ضائع**: يمرّ وقتُه كاملاً ثم ينتقل الدور،
// والزبونُ ينتظر بلا سبب.
func TestRotation_OffShiftNotOffered(t *testing.T) {
	f := newDriverFixture(t, 2)
	ctx := context.Background()
	armRotation(t, f, 0)
	f.onShift(t, f.drivers[0], false)
	f.onShift(t, f.drivers[1], true)

	orderID := f.dispatchingOrder(t, 48_000, 10_000)
	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("العرض فشل: %v", err)
	}
	var offered *string
	if err := f.pool.QueryRow(ctx,
		`SELECT offered_driver_id::text FROM orders WHERE id = $1`, orderID).
		Scan(&offered); err != nil {
		t.Fatalf("تعذّرت قراءة العرض: %v", err)
	}
	if offered == nil || *offered != f.drivers[1] {
		got := "لا أحد"
		if offered != nil {
			got = *offered
		}
		t.Errorf("عُرض على %s والمتوقّع %s", got, f.drivers[1])
	}
}
