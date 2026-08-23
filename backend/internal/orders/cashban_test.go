package orders_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestCashBlocked_AfterCustomerFault **من رفض الاستلامَ يدفع مقدَّماً.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٢٣: «الزبونُ الذي يرفض الاستلام مرّةً واحدةً لا
//
//	تصبح طلباتُه إلّا عن طريق المحفظة حصراً… تُقفل لديه لمدّةٍ يحدّدها
//	الأدمن».)
//
// # ما يحرسه
//
// **الطلبُ المرفوضُ خسارةٌ كاملةٌ على المنصّة** — قِيس على طلبٍ حقيقيّ
// (٢٠٢٦-٠٨-٢٣): ٢٣٤ دُفعت للمتجر لحظةَ استلام السائق، و٥٠ تعويضاً له،
// **والزبونُ لم يدفع شيئاً والطعامُ بيدنا.**
//
// **والمقابلُ موجودٌ للمتجر منذ زمن** (`merchants.cancel_ban_*`) —
// **وبابٌ مبنيٌّ لطرفٍ ومفتوحٌ للآخر ليس عدلاً بل سهو.**
//
// # ويُقاس من بابه العامّ لا من داخله
//
// **`cashBlocked` غيرُ مصدَّرة** — والاختبارُ في حزمةٍ خارجيّة. **وهذا
// أصحُّ قياساً**: يقيس ما يقع للزبون فعلاً حين يطلب، **لا ما تردّه
// دالّةٌ في الداخل.**
func TestCashBlocked_AfterCustomerFault(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)

	// **والعتبةُ واحدةٌ والمدّةُ ثلاثون** — كما قرّر المالك.
	if _, err := pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES
			('customers.cash_ban_failures', '1'::jsonb),
			('customers.cash_ban_days', '30'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("تعذّر ضبط المفاتيح: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`UPDATE app_settings SET value = '0'::jsonb
			  WHERE key = 'customers.cash_ban_failures'`)
	})

	itemID, cleanup := arena(t, pool, f)
	defer cleanup()

	// ── قبل الإخفاق: النقدُ مفتوح ──────────────────────────────────────
	//
	// **ويُقاس أوّلاً** — **ولو لم يُقس لَما عرفنا أنّ المنعَ لاحقاً سببُه
	// الإخفاقُ لا شيءٌ آخرُ في التهيئة.**
	if _, err := placeCash(ctx, f, itemID); err != nil {
		t.Fatalf("مُنع النقدُ عن زبونٍ لم يُخفق قطّ: %v", err)
	}

	// ── يفشل التسليمُ بذنبه ────────────────────────────────────────────
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "failed",
		"customer_refused"); err != nil {
		t.Fatalf("تعذّر الفشل: %v", err)
	}
	// **والذنبُ يُثبَّت صراحةً** — الاختبارُ يقيس الحارسَ لا كتابةَ الذنب.
	if _, err := pool.Exec(ctx,
		`UPDATE orders SET fault = 'customer' WHERE id = $1`, f.orderID); err != nil {
		t.Fatalf("تعذّر تثبيتُ الذنب: %v", err)
	}

	// ── بعده: النقدُ يُردّ برمزٍ يُقرأ ─────────────────────────────────
	_, err := placeCash(ctx, f, itemID)
	if !errors.Is(err, orders.ErrCashBlocked) {
		t.Fatalf("قُبل طلبٌ نقديٌّ بعد إخفاقٍ بذنب الزبون (%v) — "+
			"والطلبُ التالي يُرفض كأخيه فتخسر المنصّةُ ثانيةً", err)
	}
}

// TestCashBlocked_MerchantFaultDoesNotCount **ذنبُ المتجر لا يُعاقَب به الزبون.**
//
// **ولو حُسب لَعوقب على ما لا يملك من أمره شيئاً** — متجرٌ اعتذر متأخّراً
// أو طعامٌ لم يجهز، **والزبونُ ينتظر في بيته.**
func TestCashBlocked_MerchantFaultDoesNotCount(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)

	if _, err := pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES
			('customers.cash_ban_failures', '1'::jsonb),
			('customers.cash_ban_days', '30'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("تعذّر ضبط المفاتيح: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`UPDATE app_settings SET value = '0'::jsonb
			  WHERE key = 'customers.cash_ban_failures'`)
	})

	itemID, cleanup := arena(t, pool, f)
	defer cleanup()

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "failed",
		"merchant_closed"); err != nil {
		t.Fatalf("تعذّر الفشل: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE orders SET fault = 'merchant' WHERE id = $1`, f.orderID); err != nil {
		t.Fatalf("تعذّر ضبطُ الذنب: %v", err)
	}

	if _, err := placeCash(ctx, f, itemID); err != nil {
		t.Fatalf("مُنع النقدُ عن زبونٍ ذنبُه على المتجر (%v) — "+
			"وعقوبةٌ على ما لا يملكه تفقدك زبوناً بلا سبب", err)
	}
}

// arena **صنفٌ ومنطقةُ توصيلٍ تغطّي نقطةَ الاختبار.**
//
// **وبلا منطقةٍ يسقط الطلبُ بـ`out_of_zone` قبل أن يبلغ فحصَ الدفع** —
// **فيمرّ الاختبارُ لسببٍ خاطئ ولا يحرس شيئاً.**
func arena(t *testing.T, pool *pgxpool.Pool, f *fixture) (string, func()) {
	t.Helper()
	ctx := context.Background()

	var sectionID, itemID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name, sort_order) VALUES ($1, 'قسمُ نقد', 1)
		RETURNING id`, f.merchantID).Scan(&sectionID); err != nil {
		t.Fatalf("تعذّر القسم: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id, name,
		                        merchant_price, price, available, approved)
		VALUES ($1, $2, (SELECT id FROM platform_sections ORDER BY sort_order LIMIT 1),
		        'صنفُ نقد', 20000, 20000, true, true)
		RETURNING id`, f.merchantID, sectionID).Scan(&itemID); err != nil {
		t.Fatalf("تعذّر الصنف: %v", err)
	}

	var zoneID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO delivery_zones (name, delivery_fee, min_order, active, center, radius_m)
		VALUES ('منطقةُ نقد', 5000, 0, true,
		        ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography, 50000)
		RETURNING id`).Scan(&zoneID); err != nil {
		t.Fatalf("تعذّرت المنطقة: %v", err)
	}

	return itemID, func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM delivery_zones WHERE id = $1`, zoneID)
		_, _ = pool.Exec(c, `DELETE FROM menu_items WHERE id = $1`, itemID)
		_, _ = pool.Exec(c, `DELETE FROM menu_sections WHERE id = $1`, sectionID)
	}
}

// placeCash **يُنشئ طلباً نقديّاً** — الفعلُ الذي يُقاس.
func placeCash(ctx context.Context, f *fixture, itemID string) (*orders.Order, error) {
	return f.svc.Create(ctx, f.customer, []string{"customer"}, orders.CreateInput{
		CustomerID:    f.customer,
		MerchantID:    f.merchantID,
		Items:         []orders.ItemInput{{MenuItemID: itemID, Qty: 1}},
		AddressText:   "عنوانُ اختبار",
		Lat:           35.9528,
		Lng:           39.0079,
		PaymentMethod: "cash",
	}, "127.0.0.1")
}
