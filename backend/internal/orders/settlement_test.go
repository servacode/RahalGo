package orders_test

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// اختبار انحدار لأخطر خلل وُجد في المنصة (R-14): التسويات المالية كانت أحادية
// الاتجاه — التسليم يقيّد نقداً وعمولات، والاسترجاع لا يعكس منها شيئاً. فيبقى
// صندوق السائق يحمل نقد طلب مُلغى، وعمولة المندوب في رصيده، وعمولة المنصة في
// التقارير. هذا الملف يمنع عودة ذلك صامتاً.

type fixture struct {
	pool       *pgxpool.Pool
	svc        *orders.Service
	wallet     *wallet.Service
	cashbox    *cashbox.Service
	customer   string
	driver     string
	rep        string
	merchantID string
	orderID    string
}

// setup يبني أقلّ ما يلزم لطلب قابل للتسليم: متجر منسوب لمندوب، وزبون، وسائق،
// وطلب نقدي بمبلغ معلوم. الإدراج مباشر بالـSQL: نختبر التسويات لا مسار الإنشاء.
func setup(t *testing.T, status string, subtotal, deliveryFee int64, walletPaid int64) *fixture {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()

	f := &fixture{
		pool:     pool,
		wallet:   wallet.NewService(pool),
		cashbox:  cashbox.NewService(pool, settings.NewStore(pool)),
		customer: testdb.NewUser(t, pool, "customer"),
		driver:   testdb.NewUser(t, pool, "driver"),
		rep:      testdb.NewUser(t, pool, "sales"),
	}
	f.svc = orders.NewService(pool, nil, f.wallet, f.cashbox, nil,
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات في القاعدة: %v", err)
	}
	// نسبة المنصة 10% كي يكون الحساب المتوقع صريحاً
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, sales_rep_user_id, commission_percent)
		VALUES ('متجر اختبار التسويات', $1, $2, 10) RETURNING id`,
		categoryID, f.rep).Scan(&f.merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, f.merchantID)
	})

	total := subtotal + deliveryFee
	cashDue := total - walletPaid
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
		VALUES ($1, $2, $3, $4, 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', $5, $6, $7, $8, $9)
		RETURNING id`,
		f.customer, f.merchantID, f.driver, status, subtotal, deliveryFee, total, walletPaid, cashDue).
		Scan(&f.orderID); err != nil {
		t.Fatalf("تعذّر إنشاء طلب: %v", err)
	}
	return f
}

func (f *fixture) platformCommission(t *testing.T) int64 {
	t.Helper()
	var v int64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT platform_commission FROM orders WHERE id = $1`, f.orderID).Scan(&v); err != nil {
		t.Fatalf("تعذّرت قراءة عمولة المنصة: %v", err)
	}
	return v
}

func (f *fixture) balance(t *testing.T, user string) int64 {
	t.Helper()
	v, err := f.wallet.Balance(context.Background(), user)
	if err != nil {
		t.Fatalf("تعذّرت قراءة الرصيد: %v", err)
	}
	return v
}

func (f *fixture) held(t *testing.T) int64 {
	t.Helper()
	v, err := f.cashbox.Held(context.Background(), f.driver)
	if err != nil {
		t.Fatalf("تعذّرت قراءة الصندوق: %v", err)
	}
	return v
}

// التسليم يقيّد: نقد الطلب على صندوق السائق، وعمولة المنصة لقطةً، ونصيب المندوب.
func TestDelivery_CreditsCashAndCommissions(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}

	if got := f.held(t); got != 110_000 {
		t.Errorf("صندوق السائق = %d، والمتوقع 110000 (كامل المبلغ نقداً)", got)
	}
	if got := f.platformCommission(t); got != 10_000 {
		t.Errorf("عمولة المنصة = %d، والمتوقع 10000 (10%% من 100000)", got)
	}
	// نصيب المندوب = النسبة الديناميكية من عمولة المنصة (الافتراضي 10%)
	if got := f.balance(t, f.rep); got != 1_000 {
		t.Errorf("عمولة المندوب = %d، والمتوقع 1000", got)
	}
}

// الاسترجاع بعد التسليم يعكس كل ما قيّده — هذا هو الخلل الذي كان صامتاً.
func TestRefundAfterDelivery_ReversesEverything(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}
	if _, err := f.svc.Transition(ctx, f.driver, []string{"admin"}, f.orderID, "refunded", ""); err != nil {
		t.Fatalf("الاسترجاع فشل: %v", err)
	}

	// 1) الزبون يستعيد كامل ما دفعه — دفع نقداً فعلاً فيُردّ إلى محفظته
	if got := f.balance(t, f.customer); got != 110_000 {
		t.Errorf("لم يُردّ للزبون كامل المبلغ: %d، والمتوقع 110000", got)
	}
	// 2) عمولة المندوب تُعكس بقيد مقابل — الدفاتر لا تُعدَّل ولا تُحذف
	if got := f.balance(t, f.rep); got != 0 {
		t.Errorf("عمولة المندوب لم تُعكس: رصيده %d، والمتوقع 0", got)
	}
	st, err := f.wallet.StatementFor(ctx, f.rep, 10)
	if err != nil {
		t.Fatalf("تعذّر كشف محفظة المندوب: %v", err)
	}
	if len(st.Transactions) != 2 {
		t.Errorf("التصحيح لم يكن بقيد مقابل — عدد القيود %d، والمتوقع 2", len(st.Transactions))
	}
	// 3) عمولة المنصة تُصفَّر كي لا تتضخّم التقارير وفواتير المتاجر
	if got := f.platformCommission(t); got != 0 {
		t.Errorf("عمولة المنصة بقيت على طلب مُسترجَع: %d", got)
	}
	// 4) صندوق السائق لا يُعكس عمداً: النقد بحوزته ويدين به للمنصة
	if got := f.held(t); got != 110_000 {
		t.Errorf("صندوق السائق تغيّر بالاسترجاع: %d، والمتوقع 110000", got)
	}
}

// الإلغاء قبل التسليم: يُعاد المدفوع من المحفظة فقط — النقد لم يُحصَّل أصلاً.
func TestCancelBeforeDelivery_RefundsWalletOnly(t *testing.T) {
	// من "قيد التحضير": الإلغاء مسموح للعمليات، والنقد لم يُحصَّل بعد
	f := setup(t, "preparing", 50_000, 5_000, 20_000) // مختلط: 20 ألف محفظة والباقي نقداً
	ctx := context.Background()

	if _, err := f.svc.Transition(ctx, f.customer, []string{"ops"}, f.orderID, "cancelled", "اختبار"); err != nil {
		t.Fatalf("الإلغاء فشل: %v", err)
	}

	if got := f.balance(t, f.customer); got != 20_000 {
		t.Errorf("استرجاع الإلغاء = %d، والمتوقع 20000 (المدفوع من المحفظة فقط)", got)
	}
	if got := f.held(t); got != 0 {
		t.Errorf("صندوق السائق تأثّر بإلغاء قبل التسليم: %d", got)
	}
	if got := f.platformCommission(t); got != 0 {
		t.Errorf("عمولة قُيّدت لطلب لم يُسلَّم: %d", got)
	}
}

// المندوب لا يقبض عمولةً على شرائه هو.
//
// العمولة تكافئ **جلب الزبائن**، وشراءُ المندوب من متجره استهلاكٌ لا ترويج.
// وكان النظام يدفعها له: مالٌ يخرج بلا قيمة مقابلة، وأرقامٌ تقيس إنفاقه لا عمله.
// ويحرس الاختبار الاتجاهين معاً — فعكسُ عمولةٍ لم تُدفع خطأٌ مساوٍ في فداحته.
func TestDelivery_NoCommissionWhenRepIsTheBuyer(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()

	// نجعل المندوب نفسه زبونَ الطلب
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET customer_id = $2 WHERE id = $1`, f.orderID, f.rep); err != nil {
		t.Fatalf("تعذّر جعل المندوب زبوناً: %v", err)
	}

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}

	// نفحص **قيود العمولة بذاتها** لا الرصيد: الرصيد يتحرّك أيضاً باسترجاع ثمن
	// الطلب إليه كزبون، فقياسه يخلط أثرين ويخفي الخطأ الذي نبحث عنه.
	if n := f.commissionEntries(t); n != 0 {
		t.Fatalf("قُيّدت %d حركة عمولة للمندوب على شرائه هو، والمتوقع 0", n)
	}
	// عمولة المنصة تبقى كاملة: المتجر باع فعلاً ويدين بها
	if got := f.platformCommission(t); got != 10_000 {
		t.Fatalf("عمولة المنصة = %d، والمتوقع 10000 — الملغى نصيب المندوب لا العمولة", got)
	}

	// والاسترجاع لا يخصم منه شيئاً لم يقبضه
	if _, err := f.svc.Transition(ctx, f.driver, []string{"admin"}, f.orderID, "refunded", ""); err != nil {
		t.Fatalf("الاسترجاع فشل: %v", err)
	}
	if n := f.commissionEntries(t); n != 0 {
		t.Fatalf("عُكست عمولة لم تُدفع: %d حركة، والمتوقع 0", n)
	}
	if got := f.platformCommission(t); got != 0 {
		t.Fatalf("عمولة المنصة بعد الاسترجاع = %d، والمتوقع 0", got)
	}
}

// commissionEntries عدد قيود العمولة وعكسها المرتبطة بهذا الطلب لهذا المندوب.
func (f *fixture) commissionEntries(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM wallet_transactions
		WHERE user_id = $1 AND ref = $2::text AND kind IN ('commission', 'adjustment')`,
		f.rep, f.orderID).Scan(&n); err != nil {
		t.Fatalf("تعذّرت قراءة قيود العمولة: %v", err)
	}
	return n
}

// طلبٌ فشل دفعه من المحفظة يجب ألّا يترك أثراً.
//
// كان الخصم يقع بعد الـCommit، فيُلغى الطلب بقيد تعويضي عند فشله — فيبقى في
// سجل الزبون وعدّاد المتجر ومقياس «الملغي» عند المندوب طلبٌ **لم يوجد تجارياً
// قط**. وهو نفس خلل R-02 من باب آخر: مالٌ خارج المعاملة يحتاج تعويضاً بدل أن
// يتراجع معها.
func TestCreate_WalletChargeFailure_LeavesNoOrder(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	f := setup(t, "pending", 1, 0, 0) // نستعمل التهيئة للمتجر والصنف فقط

	var itemID string
	var sectionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name, sort_order) VALUES ($1, 'قسم', 1)
		RETURNING id`, f.merchantID).Scan(&sectionID); err != nil {
		t.Fatalf("تعذّر إنشاء قسم: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, name, price, available)
		VALUES ($1, $2, 'صنف', 20000, true) RETURNING id`,
		f.merchantID, sectionID).Scan(&itemID); err != nil {
		t.Fatalf("تعذّر إنشاء صنف: %v", err)
	}

	// منطقة تسليم تغطّي نقطة الاختبار — بدونها يفشل الطلب بـout_of_zone قبل أن
	// يبلغ الدفع أصلاً، فيمرّ الاختبار لسببٍ خاطئ ولا يحرس شيئاً.
	var zoneID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO delivery_zones (name, delivery_fee, min_order, active, center, radius_m)
		VALUES ('منطقة اختبار', 5000, 0, true,
		        ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography, 50000)
		RETURNING id`).Scan(&zoneID); err != nil {
		t.Fatalf("تعذّر إنشاء منطقة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM delivery_zones WHERE id = $1`, zoneID)
	})

	before := countOrders(t, pool, f.customer)

	// رصيده صفر — الدفع من المحفظة يجب أن يفشل
	_, err := f.svc.Create(ctx, f.customer, []string{"customer"}, orders.CreateInput{
		CustomerID:    f.customer,
		MerchantID:    f.merchantID,
		Items:         []orders.ItemInput{{MenuItemID: itemID, Qty: 1}},
		AddressText:   "عنوان اختبار",
		Lat:           35.9528,
		Lng:           39.0079,
		PaymentMethod: "wallet",
	}, "127.0.0.1")
	if err == nil {
		t.Fatal("نجح الطلب رغم أن الرصيد صفر")
	}

	if after := countOrders(t, pool, f.customer); after != before {
		t.Fatalf("بقي أثر لطلب لم يُدفع: عدد الطلبات %d والمتوقع %d", after, before)
	}
}

func countOrders(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, customerID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM orders WHERE customer_id = $1`, customerID).Scan(&n); err != nil {
		t.Fatalf("تعذّر عدّ الطلبات: %v", err)
	}
	return n
}

// المندوب لا يقيّم متجراً هو مندوبه.
//
// التقييم شهادةُ زبونٍ مستقلّ، وشهادةُ من ينتفع بنجاح المتجر ليست شهادة. وهو
// نظير منعِ عمولته على شرائه: لا مكافأة على ما ليس ترويجاً.
func TestRate_RepCannotRateOwnClient(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	f := setup(t, "at_dropoff", 50_000, 5_000, 0)

	if _, err := pool.Exec(ctx,
		`UPDATE orders SET customer_id = $2 WHERE id = $1`, f.orderID, f.rep); err != nil {
		t.Fatalf("تعذّر جعل المندوب زبوناً: %v", err)
	}
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}

	err := f.svc.RateOrder(ctx, f.rep, []string{"sales"}, f.orderID, 5, nil, "ممتاز")
	if !errors.Is(err, orders.ErrRateOwnClient) {
		t.Fatalf("قيّم المندوب متجره: الخطأ %v والمتوقع rate_own_client", err)
	}

	// وزبونٌ عادي على المتجر نفسه يقيّم بلا مانع
	if _, err := pool.Exec(ctx,
		`UPDATE orders SET customer_id = $2 WHERE id = $1`, f.orderID, f.customer); err != nil {
		t.Fatalf("تعذّرت إعادة الزبون: %v", err)
	}
	if err := f.svc.RateOrder(ctx, f.customer, []string{"customer"}, f.orderID, 4, nil, ""); err != nil {
		t.Fatalf("مُنع زبون عادي من التقييم: %v", err)
	}
}

// مستحقّ المتجر يُقيَّد بالتسليم ويُعكس بالاسترجاع.
//
// المتجر كان الطرف الوحيد بلا دفتر: المال يقع (الزبون يدفع، والسائق يسوّي
// للمنصة) والمنصة تمسك مال المتجر بلا قيدٍ يقول كم عليها. وأساس المستحقّ
// **subtotal** لا `total`: رسم التوصيل أجرُ خدمة المنصة لا نصيبَ المتجر.
func TestDelivery_CreditsMerchantEarning(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()

	var ownerID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO users (phone, full_name) VALUES ('+96399' || floor(random()*10000000)::text, 'مالك')
		RETURNING id`).Scan(&ownerID); err != nil {
		t.Fatalf("تعذّر إنشاء مالك: %v", err)
	}
	if _, err := f.pool.Exec(ctx,
		`UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, f.merchantID, ownerID); err != nil {
		t.Fatalf("تعذّر ربط المالك: %v", err)
	}

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}

	// 100,000 بضاعة − 10,000 عمولة (10%) = 90,000 — ورسم التوصيل 10,000 للمنصة
	if got := f.balance(t, ownerID); got != 90_000 {
		t.Fatalf("مستحقّ المتجر = %d، والمتوقع 90000 (subtotal − عمولة، بلا رسم التوصيل)", got)
	}

	if _, err := f.svc.Transition(ctx, f.driver, []string{"admin"}, f.orderID, "refunded", ""); err != nil {
		t.Fatalf("الاسترجاع فشل: %v", err)
	}
	if got := f.balance(t, ownerID); got != 0 {
		t.Fatalf("بقي مستحقّ عن طلب مُسترجَع: %d والمتوقع 0", got)
	}
}
