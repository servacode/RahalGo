package support_test

// **والمتجرُ يشتكي — على سائقه وحدَه وعلى طلبه وحدَه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٦.)
//
// # وما يُحرَس
//
// **الشرطُ على المالك لا على المتجر**: من ملك متجرين **لا يُبلّغ عن طلبِ
// ثالثٍ ليس له** — وبابٌ يقبل معرّفَ أيّ طلبٍ يفتح ملفَّ غيره.
//
// **ولا بلاغَ على سائقٍ لم يُسنَد** — **وشكوى بلا مشكوٍّ عليه تقف عند
// المكتب بلا طرفٍ يُسأل.**
//
// **ولا عن طلبٍ لم ينتهِ** — يُغلق أوّلاً ثمّ يُحكى عنه.
//
// **والجهةُ السائقُ لا الزبون**: `against_user_id` هو من يُنذَر، **ومن
// وضع الزبونَ فيه أنذر بريئا.**

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestMerchantReport_OnHisOwnOrderAgainstTheDriver(t *testing.T) {
	pool := testdb.Pool(t)
	svc := support.NewService(pool, nil, nil)
	ctx := context.Background()
	owner := testdb.NewUser(t, pool, "merchant")
	stranger := testdb.NewUser(t, pool, "merchant")
	customer := testdb.NewUser(t, pool, "customer")
	driver := testdb.NewUser(t, pool, "driver")

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	var merchantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, owner_user_id, commission_percent, location)
		VALUES ('متجرُ البلاغ', $1, $2, 10,
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography)
		RETURNING id::text`, categoryID, owner).Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر المتجر: %v", err)
	}
	mkOrder := func(withDriver bool, closed bool) string {
		var drv *string
		if withDriver {
			drv = &driver
		}
		var id string
		q := `
			INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text,
				dropoff, payment_method, subtotal, delivery_fee, total, cash_due, closed_at)
			VALUES ($1, $2, $3, 'delivered', 'عنوان اختبار',
				ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
				'cash', 1000, 0, 1000, 1000, CASE WHEN $4 THEN now() END)
			RETURNING id::text`
		if err := pool.QueryRow(ctx, q, customer, merchantID, drv, closed).Scan(&id); err != nil {
			t.Fatalf("تعذّر الطلب: %v", err)
		}
		return id
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM tickets WHERE created_by = $1`, owner)
		_, _ = pool.Exec(c, `DELETE FROM orders WHERE merchant_id = $1`, merchantID)
		_, _ = pool.Exec(c, `DELETE FROM merchants WHERE id = $1`, merchantID)
	})

	good := mkOrder(true, true)

	// ── ما يُردّ ──────────────────────────────────────────────────────
	if _, err := svc.MerchantReport(ctx, owner, good, "لفظٌ ليس في القائمة", ""); err == nil {
		t.Fatal("قُبل سببٌ ليس في القائمة — **والحكمُ في المحرّك لا في الشاشة**")
	}
	if _, err := svc.MerchantReport(ctx, stranger, good, "driver_late_pickup", ""); err == nil {
		t.Fatal("أبلغ غريبٌ عن طلبِ متجرٍ ليس له — **فيقرأ ملفَّ غيره من بابِ بلاغ**")
	}
	if _, err := svc.MerchantReport(ctx, owner, mkOrder(false, true), "driver_conduct", ""); err == nil {
		t.Fatal("قُبل بلاغٌ على طلبٍ بلا سائق — **وشكوى بلا مشكوٍّ عليه تقف بلا طرف**")
	}
	if _, err := svc.MerchantReport(ctx, owner, mkOrder(true, false), "driver_conduct", ""); !errors.Is(err, support.ErrOrderNotClosed) {
		t.Fatalf("قُبل بلاغٌ على طلبٍ جارٍ: %v — **يُغلق أوّلاً ثمّ يُحكى عنه**", err)
	}

	// ── وما يُقبل ─────────────────────────────────────────────────────
	tk, err := svc.MerchantReport(ctx, owner, good, "driver_late_pickup", "الطلبُ جاهزٌ منذ ساعة")
	if err != nil {
		t.Fatalf("رُدّ بلاغٌ صحيح: %v", err)
	}
	if tk.Reason != "driver_late_pickup" {
		t.Fatalf("السببُ %q", tk.Reason)
	}
	// **وصاحبُ التذكرة زبونُ الطلب لا المتجر** — العمودُ يقول «تذكرةُ أيّ
	// طلبٍ هذه» لا «من كتبها».
	if tk.CustomerID != customer {
		t.Fatalf("صاحبُ التذكرة %s لا الزبون", tk.CustomerID)
	}
	if tk.OpenedByCustomer {
		t.Fatal("عُدَّ بلاغُ المتجر شكوى الزبون — **والحكمان متناقضان**")
	}
	// **والجهةُ السائق** — ومن وضع الزبونَ فيها أنذر بريئا.
	var against *string
	if err := pool.QueryRow(ctx,
		`SELECT against_user_id::text FROM tickets WHERE id = $1`, tk.ID).Scan(&against); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if against == nil || *against != driver {
		t.Fatalf("الجهةُ %v لا السائق — **فيُنذَر من لم يُشتكَ عليه**", against)
	}

	// **ولا بلاغان على طلبٍ واحدٍ وأوّلُهما مفتوح** — **وإلّا أُغرق المكتبُ
	// بشكوًى واحدةٍ مكرّرة.**
	if _, err := svc.MerchantReport(ctx, owner, good, "driver_conduct", ""); err == nil {
		t.Fatal("فُتح بلاغٌ ثانٍ على الطلب نفسِه والأوّلُ مفتوح")
	}
}
