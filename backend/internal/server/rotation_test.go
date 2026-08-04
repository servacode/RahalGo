package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

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
	isolateStaleOrders(t, f.pool)
	f.setSetting(t, "drivers.assignment_mode", "rotation")
	if timeoutSec > 0 {
		f.setSetting(t, "drivers.offer_timeout_sec", timeoutSec)
	}
	t.Cleanup(func() {
		f.setSetting(t, "drivers.assignment_mode", "queue")
	})
}

// isolateStaleOrders يُغلق ركامَ التجارب السابقة — **قبل أن يُنشأ طلبُ الاختبار.**
//
// # لماذا يلزم
//
// محرّكُ الترتيب يقرأ **كلَّ** طلبٍ مفتوحٍ في القاعدة لا طلبَ هذا الاختبار:
// `offerWaiting` يعرض المنتظِر، و`reclaim` يستردّ الإسنادَ الصامت. وطلباتُ
// تجاربَ سابقةٍ تبقى مفتوحةً إلى الأبد، **فتُعرض على سائقي الاختبار فيصيرون
// مشغولين** — ويسقط الاختبارُ بـ«لا عرضَ وثمّة سائقون مؤهّلون»، **صادقاً في
// وصفه كاذباً في سببه.** ولا يسقط إلّا بعد أن يتراكم ما يكفي، فيبدو تقلّباً
// عشوائياً.
//
// # وكلُّ مفتوحٍ لا `dispatching` وحدَها
//
// كان الشرطُ عليها، **ثمّ جاء الإسنادُ المباشر فصار الركامُ `assigned` أيضاً.**
// **ومحرّكٌ يقرأ كلَّ الطلبات يلزمه عزلٌ يشمل كلَّ الطلبات.**
//
// # والحالةُ تُبدَّل لا `closed_at` وحدَه
//
// المحرّكُ يقرأ `status` **ولا يقرأ `closed_at`** — فإغلاقٌ بلا تبديلِ حالةٍ
// يترك الطلبَ يُعرض ويأخذ سائقاً. (وهي حالٌ لا تقع في الإنتاج: كلُّ إغلاقٍ
// يبدّل الحالةَ معه.)
//
// # وهو آمنٌ لأنّ الحزمَ لا تتوازى
//
// **نمطُ التوزيع إعدادٌ في القاعدة لا في الذاكرة.** فحين ترفعه هذه الحزمةُ إلى
// «بالتساوي» **يرتفع لكلّ حزمةٍ تعمل في اللحظة نفسِها** — واختبارٌ في حزمة
// الطلبات ينقل طلباً إلى `dispatching` فيُنادى `OfferNext` من حيث لا يتوقّع
// أحد، **ثمّ يُلغيه هذا العزل.** (وقع فعلاً ٢٠٢٦-٠٨-٠٥: «الحالة cancelled
// والمتوقّع dispatching» — ولا شيءَ في تلك الحزمة يفسّره.)
//
// **وقيدُ عمرٍ لا يحلّها**: ركامُ هذه الحزمة عمرُه ثوانٍ أيضاً، فالحدُّ الذي
// يحمي غيرَنا يترك ركامَنا.
//
// **فالحلُّ في `-p 1`** (انظر `Makefile`): حزمةٌ واحدةٌ في المرّة. **ومَوردٌ
// عامٌّ مشتركٌ لا يُقسَّم بحيلةٍ في استعلام.**
func isolateStaleOrders(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`UPDATE orders SET status = 'cancelled', closed_at = now(),
		                   offered_driver_id = NULL, offer_expires_at = NULL,
		                   driver_id = NULL
		 WHERE closed_at IS NULL`); err != nil {
		t.Fatalf("تعذّر عزلُ الطلبات الغابرة: %v", err)
	}
}

// driverEligibility حالُ كلِّ سائقٍ بمقاييس `OfferNext` الخمسة.
//
// **وشرطٌ يسقط بلا صوت لا يُشخَّص من رسالةٍ تقول «لا عرض».** والعلّةُ في
// قاعدةٍ مشتركةٍ تتراكم عادةً خارجَ ما يُختبَر — **فيُقرأ أيُّها منع بدل أن
// يُخمَّن.**
func (f *driverFixture) driverEligibility(t *testing.T, orderID string) string {
	t.Helper()
	rows, err := f.pool.Query(context.Background(), `
		SELECT u.id::text, u.on_shift, u.status,
		       COALESCE((SELECT b.held FROM driver_cash_boxes b WHERE b.driver_id = u.id), 0),
		       (SELECT count(*) FROM orders o WHERE o.driver_id = u.id AND o.closed_at IS NULL),
		       (SELECT count(*) FROM orders o2 WHERE o2.offered_driver_id = u.id
		          AND o2.id <> $2 AND o2.status = 'dispatching'
		          AND o2.driver_id IS NULL AND o2.offer_expires_at > now())
		FROM users u WHERE u.id = ANY($1::uuid[])`, f.drivers, orderID)
	if err != nil {
		return "تعذّرت قراءةُ حال السائقين: " + err.Error()
	}
	defer rows.Close()

	out := "\tالسائق | دوام | حال | نقدٌ بيده | طلباتٌ مفتوحة | عروضٌ حيّةٌ أخرى\n"
	for rows.Next() {
		var id, status string
		var onShift bool
		var held int64
		var open, offers int
		if err := rows.Scan(&id, &onShift, &status, &held, &open, &offers); err != nil {
			return "تعذّرت قراءةُ صفّ: " + err.Error()
		}
		out += fmt.Sprintf("\t%s | %v | %s | %d | %d | %d\n",
			id[:8], onShift, status, held, open, offers)
	}
	var passed int
	_ = f.pool.QueryRow(context.Background(),
		`SELECT coalesce(array_length(offer_passed, 1), 0) FROM orders WHERE id = $1`,
		orderID).Scan(&passed)
	out += fmt.Sprintf("\tمرّ عليهم الدورُ في هذا الطلب: %d", passed)
	return out
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
			// **ولا يُقال «لا عرض» ويُسكت**: شروطُ الأهلية خمسة، وواحدٌ منها
			// يسقط بلا صوت. فيُطبع حالُ كلِّ سائقٍ ليُقرأ أيُّها منع.
			t.Fatalf("الجولة %d: لا عرضَ وثمّة سائقون مؤهّلون\n%s",
				round, f.driverEligibility(t, orderID))
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
