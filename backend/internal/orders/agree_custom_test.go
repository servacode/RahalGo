package orders_test

// **توثيقُ سعر الطلب الخاصّ يقع فعلاً — لا يسقط بخمسمئة.**
//
// (شكوى المالك ٢٠٢٦-٠٨-١٨: «عند توثيق السعر بالطلب الخاصّ يعطي: تعذّر
//
//	الاتصال، حاول بعد قليل».)
//
// # ولماذا لم يُكتشف قبل اليوم
//
// **لم يكن على `AgreeCustom` اختبارٌ واحد.** والاختباراتُ الأخرى تمرّ
// خضراءَ لأنّها لا تبلغ هذا الاستعلام — **وأخضرُ لا يمسّ الشيءَ لا يقول
// عنه شيئا.**
//
// # وما يُقاس
//
// **الاستعلامُ يُنفَّذ على قاعدةٍ حقيقيّة** — والعطبُ كان في تحليل SQL
// نفسِه (`unknown + unknown`)، **فلا يظهر إلّا حين تُسأل بوستغرس.**
//
// **ويُقاس ما كُتب لا أنّه لم يسقط**: `total` يجب أن يساوي المجموع،
// **ودالّةٌ تكتب صفراً ولا تسقط تمرّ في اختبارٍ يفحص الخطأ وحدَه.**

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

func TestAgreeCustom_WritesTotals(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL غير مضبوط")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("تعذّر الاتّصال: %v", err)
	}
	t.Cleanup(db.Close)

	// ── حسابان وطلبٌ خاصٌّ مُسنَد ────────────────────────────────────
	var customer, driver, orderID string
	mk := func(phone, name string) string {
		var id string
		if err := db.QueryRow(ctx, `
			INSERT INTO users (phone, full_name, status)
			VALUES ($1, $2, 'active') RETURNING id::text`, phone, name).Scan(&id); err != nil {
			t.Skipf("تعذّر إنشاءُ حساب: %v", err)
		}
		return id
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, `DELETE FROM orders WHERE id::text = $1`, orderID)
		_, _ = db.Exec(ctx, `DELETE FROM users WHERE id::text = ANY($1)`,
			[]string{customer, driver})
	})
	_, _ = db.Exec(ctx, `DELETE FROM users WHERE phone IN ('+963900777001','+963900777002')`)

	customer = mk("+963900777001", "زبونُ الفحص")
	driver = mk("+963900777002", "سائقُ الفحص")

	// **والطلبُ خاصٌّ بلا متجر** — وهو ما يميّزه.
	if err := db.QueryRow(ctx, `
		INSERT INTO orders (customer_id, driver_id, kind, status, address_text,
		                    dropoff, payment_method, subtotal, delivery_fee, total)
		VALUES ($1, $2, 'custom', 'assigned', 'الرقة',
		        ST_SetSRID(ST_MakePoint(39.0094, 35.9506),4326)::geography, 'cash',
		        0, 0, 0)
		RETURNING id::text`, customer, driver).Scan(&orderID); err != nil {
		t.Skipf("تعذّر إنشاءُ الطلب: %v", err)
	}

	svc := orders.NewService(db, nil, nil, nil, nil, nil)

	const goods, fee = 6_000, 1_500
	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("سقط التوثيق: %v — **وهذا ما يراه السائقُ «تعذّر الاتصال»**", err)
	}

	var gotGoods, gotFee, gotSub, gotDelivery, gotTotal int64
	if err := db.QueryRow(ctx, `
		SELECT custom_goods_amount, custom_fee, subtotal, delivery_fee, total
		FROM orders WHERE id::text = $1`, orderID).
		Scan(&gotGoods, &gotFee, &gotSub, &gotDelivery, &gotTotal); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}

	if gotGoods != goods || gotFee != fee {
		t.Errorf("وُثّق %d و%d والمنتظَر %d و%d", gotGoods, gotFee, goods, fee)
	}
	// **والإجماليُّ مجموعُهما** — (قرارُ المالك ٢٠٢٦-٠٨-١٣: «يجب أن
	// يُكتب الإجماليُّ وأجرةُ التوصيل بالطلب أيضاً»).
	if gotSub != goods || gotDelivery != fee || gotTotal != goods+fee {
		t.Errorf("الأعمدة: subtotal=%d delivery=%d total=%d — والمنتظَر %d و%d و%d",
			gotSub, gotDelivery, gotTotal, goods, fee, goods+fee)
	}

	// **ولا يُمسّ ما على السائق** — ما قبضه في الخاصّ مالُه هو.
	var cashDue int64
	_ = db.QueryRow(ctx, `SELECT cash_due FROM orders WHERE id::text = $1`, orderID).
		Scan(&cashDue)
	if cashDue != 0 {
		t.Errorf("cash_due = %d — **ودَينٌ لا وجودَ له في ذمّة السائق**", cashDue)
	}
}
