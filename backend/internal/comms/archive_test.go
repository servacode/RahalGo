package comms_test

// **سجلُّ المحادثات — ومن سُحب منه الطلبُ يبقى له ما قال.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «يجب أن يكون هناك دردشاتي السابقة… مشان إثبات».)
//
// # ولماذا يُحرَس بالقاعدة لا بالحكم وحدَه
//
// **العطبُ الذي وقع كان في الاستعلام نفسِه** — لا في شرطٍ يُختبر بلا قاعدة:
// الصفُّ كان يُقاس بمن **يحمل** الطلبَ الآن، **والحديثُ يحمل من قاله.**
//
// **فسائقٌ كتب ثمّ أُعيد الطلبُ إلى الطابور تختفي كلماتُه من سجلّه هو**
// ويراها الزبونُ وحدَه. **وسجلٌّ يراه طرفٌ ولا يراه الآخرُ ليس إثباتاً** —
// إنّما حجّةٌ في يدٍ واحدة. **ولا خطأ ولا سطرٌ في سجلّ**: قائمةٌ أقصرُ بصفّ.
//
// (وقع فعلاً — كشفه المشيُ الحيُّ على الطلب #1126.)

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// spokenThenPulled **طلبٌ تحدّث فيه سائقٌ ثمّ سُحب منه.**
//
// **وهو حالُ العطب بعينه**: الرسالةُ باقيةٌ و`driver_id` فارغ.
func spokenThenPulled(t *testing.T) (*comms.Service, *pgxpool.Pool, string, string, string) {
	t.Helper()
	pool := testdb.Pool(t)
	customer := testdb.NewUser(t, pool, "customer")
	driver := testdb.NewUser(t, pool, "driver")
	ctx := context.Background()

	var merchant string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id)
		VALUES ('متجرُ اختبار',
			(SELECT id FROM categories ORDER BY created_at LIMIT 1))
		RETURNING id`).Scan(&merchant); err != nil {
		t.Fatalf("تعذّر إنشاءُ متجر: %v", err)
	}

	var order string
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
		VALUES ($1, $2, $3, 'at_pickup', 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 10000, 0, 10000, 0, 10000)
		RETURNING id`, customer, merchant, driver).Scan(&order); err != nil {
		t.Fatalf("تعذّر إنشاءُ طلب: %v", err)
	}

	// **قال السائقُ كلمتَه** — ثمّ سُحب منه الطلبُ وأُعيد إلى الطابور.
	if _, err := pool.Exec(ctx, `
		INSERT INTO order_messages (order_id, sender_id, sender_role, body)
		VALUES ($1, $2, 'driver', 'أنا تحت البناية')`, order, driver); err != nil {
		t.Fatalf("تعذّرت كتابةُ رسالة: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE orders SET driver_id = NULL, status = 'cancelled' WHERE id = $1`,
		order); err != nil {
		t.Fatalf("تعذّر سحبُ الطلب: %v", err)
	}

	return comms.New(pool), pool, order, customer, driver
}

// TestArchive_TheOneWhoSpokeKeepsHisWords **الطرفان يريانه أو ليس إثباتاً.**
func TestArchive_TheOneWhoSpokeKeepsHisWords(t *testing.T) {
	svc, _, order, customer, driver := spokenThenPulled(t)
	ctx := context.Background()

	for _, who := range []struct {
		name string
		id   string
	}{{"الزبون", customer}, {"السائق", driver}} {
		list, err := svc.Threads(ctx, who.id)
		if err != nil {
			t.Fatalf("تعذّر قراءةُ سجلّ %s: %v", who.name, err)
		}
		var found *comms.Thread
		for i := range list {
			if list[i].OrderID == order {
				found = &list[i]
			}
		}
		if found == nil {
			t.Fatalf("الحديثُ غائبٌ عن سجلّ %s — **وحجّةٌ في يدٍ واحدةٍ ليست إثباتاً**", who.name)
		}
		// **واسمُ الطرف الآخر يُقرأ** — **وسطرٌ بلا قائلٍ لا يُحتجّ به.**
		if found.Peer == "" {
			t.Fatalf("سجلُّ %s بلا اسمِ الطرف الآخر — **سطرٌ بلا قائل**", who.name)
		}
		// **ومغلقةٌ** — الطلبُ انتهى.
		if found.Open {
			t.Fatalf("محادثةُ %s مفتوحةٌ بعد انتهاء الطلب", who.name)
		}
	}
}

// TestArchive_TheOneWhoSpokeCanStillReadIt **وصفٌّ لا يُفتح ليس سجلّاً.**
//
// **والقائمةُ نصفُ الطريق**: من رأى الحديثَ في سجلّه ثمّ فُتح على فراغٍ لم
// يربح شيئاً. **و`Permit` كانت تردّ «لستَ طرفاً» لمن كتب بيده.**
func TestArchive_TheOneWhoSpokeCanStillReadIt(t *testing.T) {
	svc, _, order, customer, driver := spokenThenPulled(t)
	ctx := context.Background()

	for _, who := range []struct {
		name string
		id   string
	}{{"الزبون", customer}, {"السائق", driver}} {
		p, err := svc.Permit(ctx, order, who.id)
		if err != nil {
			t.Fatalf("%s لا يبلغ حديثَه: %v", who.name, err)
		}
		msgs, err := svc.List(ctx, p)
		if err != nil {
			t.Fatalf("تعذّرت قراءةُ حديث %s: %v", who.name, err)
		}
		if len(msgs) != 1 {
			t.Fatalf("%s يرى %d رسالةً والمكتوبُ واحدة", who.name, len(msgs))
		}

		// **ولا يكتب** — القناةُ انتهت بانتهاء الطلب، **والسجلُّ يُقرأ ولا
		// يُزاد عليه بعد الخلاف.**
		if _, err := svc.Send(ctx, p, "كلمةٌ بعد الإغلاق"); err == nil {
			t.Fatalf("%s كتب في حديثٍ منتهٍ — **ودليلٌ يُزاد عليه بعد الخلاف ليس دليلاً**", who.name)
		}
	}
}
