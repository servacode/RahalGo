package comms_test

// **أحاديثُ الحسابِ تُقرأ من ملفّه — والموسومُ يتقدّم.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٥: «نسينا سجلَّ الدردشات… مهمّةٌ في حال حدوث أيّ
//  مشكلةٍ أو نزاع».)
//
// # وما يُحرَس
//
// **الطرفان اثنان والمعرّفُ واحد** — ومن خلط `customer_id` بـ`driver_id`
// **أرى السائقَ أحاديثَ لم يكن فيها.**
//
// **والطرفُ الآخرُ يُشتقّ من موقعه**: من كان زبوناً رأى سائقَه ومن كان
// سائقاً رأى زبونَه. **وعكسُه يُري الإنسانَ نفسَه في خانة «مع مَن».**
//
// **والموسومُ يتقدّم** — «قال لي كذا» كانت كلمةً ضدّ كلمة، **وترتيبٌ
// بالزمن يدفن السطرَ الموسومَ تحت «أنا تحت البناية».**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestUserThreads_BothSidesAndFlaggedFirst(t *testing.T) {
	pool := testdb.Pool(t)
	svc := comms.New(pool)
	ctx := context.Background()
	customer := testdb.NewUser(t, pool, "customer")
	driver := testdb.NewUser(t, pool, "driver")

	mkOrder := func() string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO orders (kind, customer_id, driver_id, status, address_text, dropoff,
				payment_method, custom_request, subtotal, delivery_fee, total, cash_due)
			VALUES ('custom', $1, $2, 'delivered', 'عنوان اختبار',
				ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
				'cash', 'طلبٌ للفحص', 0, 0, 0, 0)
			RETURNING id::text`, customer, driver).Scan(&id); err != nil {
			t.Fatalf("تعذّر إنشاءُ طلب: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
		})
		return id
	}
	say := func(orderID, sender, role, body string, flagged bool) {
		if _, err := pool.Exec(ctx, `
			INSERT INTO order_messages (order_id, sender_id, sender_role, body, flagged)
			VALUES ($1, $2, $3, $4, $5)`, orderID, sender, role, body, flagged); err != nil {
			t.Fatalf("تعذّرت الرسالة %q: %v", body, err)
		}
	}

	// **حديثان: الأقدمُ فيه موسوم، والأحدثُ نظيف.**
	//
	// **فلو رُتّبا بالزمن لَتقدّم النظيفُ** — وهو نقيضُ ما يُفتح الملفُّ لأجله.
	dirty := mkOrder()
	say(dirty, customer, "customer", "سطرٌ فيه لفظٌ لا يُقال", true)
	say(dirty, driver, "driver", "حسناً", false)
	clean := mkOrder()
	say(clean, customer, "customer", "أنا تحت البناية", false)

	// **وطلبٌ بلا حديثٍ لا يُعدّ** — قائمةٌ فيها ما لا يُقرأ تُقلَّب بلا فائدة.
	mkOrder()

	// ── من باب الزبون ─────────────────────────────────────────────────
	th, total, err := svc.UserThreads(ctx, customer, 10, 0)
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if total != 2 || len(th) != 2 {
		t.Fatalf("%d حديثاً و%d مجموعاً — **وطلبٌ بلا حديثٍ لا يُعدّ**", len(th), total)
	}
	// **والموسومُ أوّلاً.**
	if th[0].OrderID != dirty {
		t.Fatal("النظيفُ تقدّم الموسومَ — **فيُدفن ما يُبحث عنه تحت «أنا تحت البناية»**")
	}
	if th[0].Flagged != 1 || th[0].Count != 2 {
		t.Fatalf("العدُّ %d والموسومُ %d — **والرأسُ يُقرأ بلمحةٍ فيُعرف أيُفتح**",
			th[0].Count, th[0].Flagged)
	}
	// **وآخرُ ما قيل** — لا أوّلُه.
	if th[0].Last != "حسناً" || th[0].LastRole != "driver" {
		t.Fatalf("آخرُ سطرٍ %q من %q", th[0].Last, th[0].LastRole)
	}
	// **والطرفُ الآخرُ سائقُه لا هو.**
	if th[0].PeerRole != "driver" {
		t.Fatalf("الطرفُ %q — **ومن رأى نفسَه في خانة «مع مَن» لم يعرف مع مَن**",
			th[0].PeerRole)
	}

	// ── ومن باب السائق ────────────────────────────────────────────────
	//
	// **وهو لم يكتب في الحديث النظيف حرفاً** — **وله فيه سطرُ الآخر**،
	// وهو ما يُحتجّ به عليه أو له.
	dth, dtotal, err := svc.UserThreads(ctx, driver, 10, 0)
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ السائق: %v", err)
	}
	if dtotal != 2 {
		t.Fatalf("عند السائق %d — **والمعرّفُ يُقارن بطرفَي الطلب لا بمن كتب**", dtotal)
	}
	if dth[0].PeerRole != "customer" {
		t.Fatalf("الطرفُ عند السائق %q لا customer", dth[0].PeerRole)
	}
}
