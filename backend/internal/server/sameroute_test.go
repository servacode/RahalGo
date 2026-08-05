package server

// الإسنادُ بنفس المسار — **أربعةُ شروطٍ لا واحد.**
//
// # ولماذا اختبارٌ لكلّ شرطٍ على حدة
//
// **شرطٌ يسقط وحدَه لا يُرى**: تُسند الطلباتُ ويبدو النظامُ يعمل، **والخللُ
// أنّه يُسندها إلى من لا يجب** — سائقٌ في آخر المدينة، أو من جاوز المتجرَ
// فيرجع القهقرى.
//
// **ولا يظهر في أيّ خطأ**: كلُّ إسنادٍ ينجح، والطلبُ يصل متأخّراً وحسب.

import (
	"context"
	"testing"
)

// nearby يبني طلباً جديداً بمتجرٍ وزبونٍ في موضعٍ مُعطًى.
//
// **والمواضعُ تُمرَّر لا تُفترض**: جوهرُ ما يُختبَر هو المسافةُ نفسُها،
// **واختبارٌ يخفي المسافةَ في عُدّةٍ يفحص شيئاً لا يراه قارئُه.**
func (f *driverFixture) orderAt(t *testing.T, mLat, mLng, dLat, dLng float64) string {
	t.Helper()
	ctx := context.Background()
	var merchantID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, location)
		VALUES ('متجرُ مسارٍ', (SELECT category_id FROM merchants WHERE id = $1), 10,
		        ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography)
		RETURNING id`, f.merchantID, mLat, mLng).Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر إنشاءُ متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID)
	})

	var id string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
		VALUES ((SELECT customer_id FROM orders WHERE merchant_id = $1 LIMIT 1),
		        $1, 'dispatching', 'مسار',
		        ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
		        'cash', 20000, 10000, 30000, 0, 30000)
		RETURNING id`, merchantID, dLat, dLng).Scan(&id); err != nil {
		// **وأوّلُ طلبٍ لا زبونَ قبله** — فيُنشأ له واحد.
		customer := newCustomer(t, f)
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO orders (customer_id, merchant_id, status, address_text, dropoff,
				payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
			VALUES ($1, $2, 'dispatching', 'مسار',
			        ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography,
			        'cash', 20000, 10000, 30000, 0, 30000)
			RETURNING id`, customer, merchantID, dLat, dLng).Scan(&id); err != nil {
			t.Fatalf("تعذّر إنشاءُ طلب: %v", err)
		}
	}
	return id
}

func newCustomer(t *testing.T, f *driverFixture) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO users (phone, full_name)
		VALUES ('+9639' || lpad((nextval('test_phone_seq') % 100000000)::text, 8, '0'),
		        'زبونُ مسار')
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ زبون: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

// hold يضع طلباً في يد سائقٍ عند حالةٍ بعينها.
func (f *driverFixture) hold(t *testing.T, orderID, driverID, status string) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET driver_id = $2, status = $3 WHERE id = $1`,
		orderID, driverID, status); err != nil {
		t.Fatalf("تعذّر وضعُ الطلب في يده: %v", err)
	}
}

// standAt يضع السائقَ في موضعٍ الآن.
func (f *driverFixture) standAt(t *testing.T, driverID string, lat, lng float64) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), `
		UPDATE users SET last_location = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
		                 last_location_at = now()
		WHERE id = $1`, driverID, lat, lng); err != nil {
		t.Fatalf("تعذّر وضعُ موضعِه: %v", err)
	}
}

// ── مواضعُ الرقّة — أرقامٌ حقيقيةٌ تُقرأ ────────────────────────────────────
//
// **ونقاطٌ مخترعةٌ في المحيط تُنتج مسافاتٍ لا يفهمها من يقرأ الاختبار.**
const (
	mA1, mA2 = 35.9520, 39.0090 // متجرٌ أوّل
	mB1, mB2 = 35.9540, 39.0140 // متجرٌ ثانٍ — نحو ٥٠٠ م عنه
	far1     = 35.9900          // متجرٌ بعيدٌ — أكثرُ من أربعة كيلومترات
	dA1, dA2 = 35.9700, 39.0300 // زبونٌ أوّل
	dB1, dB2 = 35.9720, 39.0330 // زبونٌ ثانٍ — قريبٌ منه
	dFar1    = 35.9200          // زبونٌ في الجهة المقابلة
)

// armSameRoute يهيّئ سائقاً بطلبٍ في يده وموضعٍ قبل متجره.
func armSameRoute(t *testing.T, f *driverFixture) (driver, anchor string) {
	t.Helper()
	armRotation(t, f, 60)
	driver = f.drivers[0]
	f.onShift(t, driver, true)
	anchor = f.orderAt(t, mA1, mA2, dA1, dA2)
	f.hold(t, anchor, driver, "assigned")
	// **ويقف قبل متجره** — فهو لم يجاوزه بعد.
	f.standAt(t, driver, 35.9500, 39.0060)
	return driver, anchor
}

// TestSameRoute_AssignsToDriverOnTheSameLine **طلبٌ في طريقه يقع في يده.**
func TestSameRoute_AssignsToDriverOnTheSameLine(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	driver, _ := armSameRoute(t, f)

	fresh := f.orderAt(t, mB1, mB2, dB1, dB2)
	if !f.srv.orders.TrySameRoute(ctx, fresh) {
		t.Fatal("لم يُسنَد وطلبٌ في طريقه — **رحلتان بدل رحلة**")
	}
	got, status := f.assignedTo(t, fresh)
	if got != driver || status != "assigned" {
		t.Fatalf("driver=%q status=%q — والمتوقّع أن يقع في يد صاحب المسار", got, status)
	}
	// **والسببُ في السجلّ** — العملياتُ تسأل «لماذا هذا؟»، **و«الخوارزميةُ
	// اختارته» ليس جواباً.**
	var note string
	if err := f.pool.QueryRow(ctx, `
		SELECT note FROM order_events
		WHERE order_id = $1 AND to_status = 'assigned'`, fresh).Scan(&note); err != nil {
		t.Fatalf("لا حدثَ إسناد: %v", err)
	}
	if note == "" {
		t.Fatal("حدثٌ بلا سبب — **ولا يُقرأ لماذا وقع الطلبُ في يده**")
	}
}

// TestSameRoute_FarMerchantIsNotSameRoute **ومتجرٌ بعيدٌ ليس في الطريق.**
func TestSameRoute_FarMerchantIsNotSameRoute(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	armSameRoute(t, f)

	fresh := f.orderAt(t, far1, mA2, dA1, dA2)
	if f.srv.orders.TrySameRoute(ctx, fresh) {
		t.Fatal("أُسند ومتجرُه على بعد كيلومترات — **وقفةٌ ثانيةٌ في آخر المدينة**")
	}
}

// TestSameRoute_OppositeCustomerIsNotSameRoute **وزبونان متعاكسان رحلتان.**
//
// **ومطعمان متلاصقان لا يكفيان**: الثاني يبرد طعامُه في الطريق إلى الأوّل.
func TestSameRoute_OppositeCustomerIsNotSameRoute(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	armSameRoute(t, f)

	fresh := f.orderAt(t, mB1, mB2, dFar1, dA2)
	if f.srv.orders.TrySameRoute(ctx, fresh) {
		t.Fatal("أُسند وزبونُه في الجهة المقابلة — **رحلتان لا رحلة**")
	}
}

// TestSameRoute_PassedMerchantIsNotSameRoute **ومن جاوز المتجرَ يرجع القهقرى.**
func TestSameRoute_PassedMerchantIsNotSameRoute(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	driver, _ := armSameRoute(t, f)

	// **وقف قربَ زبونه** — فالمتجرُ الثاني خلفه.
	f.standAt(t, driver, dA1, dA2)

	fresh := f.orderAt(t, mB1, mB2, dB1, dB2)
	if f.srv.orders.TrySameRoute(ctx, fresh) {
		t.Fatal("أُسند وقد جاوز المتجرَ — **رجوعٌ القهقرى أطولُ من رحلةٍ مستقلّة**")
	}
}

// TestSameRoute_StaleLocationIsNotTrusted **وموضعٌ شاخ لا يُقاس عليه.**
//
// من أطفأ التطبيقَ قبل ساعةٍ يبقى موضعُه مكتوباً، **فيُقرأ واقفاً قبل المتجر
// وهو في بيته.**
func TestSameRoute_StaleLocationIsNotTrusted(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	driver, _ := armSameRoute(t, f)

	if _, err := f.pool.Exec(ctx,
		`UPDATE users SET last_location_at = now() - interval '2 hours' WHERE id = $1`,
		driver); err != nil {
		t.Fatalf("تعذّر تعتيقُ الموضع: %v", err)
	}

	fresh := f.orderAt(t, mB1, mB2, dB1, dB2)
	if f.srv.orders.TrySameRoute(ctx, fresh) {
		t.Fatal("أُسند بموضعٍ عمرُه ساعتان — **والجهلُ ليس قرباً**")
	}
}

// TestSameRoute_AfterPickupIsNotACandidate **ومن استلم بضاعتَه لا يعود.**
//
// الطعامُ في صندوقه، **ووقفةٌ ثانيةٌ عند مطعمٍ تُبرّد ما حمله.**
func TestSameRoute_AfterPickupIsNotACandidate(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	driver, anchor := armSameRoute(t, f)
	f.hold(t, anchor, driver, "picked_up")

	fresh := f.orderAt(t, mB1, mB2, dB1, dB2)
	if f.srv.orders.TrySameRoute(ctx, fresh) {
		t.Fatal("أُسند وقد استلم بضاعتَه — **والطعامُ يبرد في صندوقه**")
	}
}
