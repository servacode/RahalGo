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
	// merchantCost سعرُ الشراء المزروع — **الأساسُ الذي تُحسب عليه العمولة
	// ومستحقُّ المتجر**، والفرقُ بينه وبين `subtotal` هو الهامش.
	merchantCost int64
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
	// **ومخزنُ الإعدادات يُحقَن كما يُحقَن في الإقلاع** (`server.go`).
	//
	// كانت العُدّةُ تبنيه بلا مخزن **وتمرّ الاختبارات** — لأنّ قواعدَ المال
	// كانت تُقرأ بـSQL خامٍّ يتجاوزه. **فكانت تفحص مساراً لا وجودَ له في
	// الإنتاج.** ولمّا صار المصدرُ واحداً ظهر الفرق.
	f.svc.SetSettings(settings.NewStore(pool))

	// العتبة = 1 تعني «بلا عتبة»: هذه الاختبارات تفحص التسوية لا التفعيل، وطلبٌ
	// واحد يجب أن يُنتج عمولة فيها. واختبارات التفعيل ترفعها صراحةً.
	if _, err := pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ('sales.activation_orders','1'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("تعذّر ضبط عتبة التفعيل: %v", err)
	}

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

	// **بندٌ واحدٌ بسعرين — وهو ما تقرؤه التسوية.**
	//
	// بعد نموذج السعرين صار كلُّ حسابٍ يقرأ `order_items`: **مستحقُّ المتجر من
	// `merchant_price`، وعمولةُ المندوب من الفرق.** وطلبٌ بلا بنود يجعل
	// الهامشَ صفراً — **فيمرّ الاختبار على نموذجٍ لا وجودَ له.**
	//
	// والهامشُ عُشر سعر البيع: `100000` بيعاً و`90000` شراءً. **ورقمٌ مستدير
	// يجعل الخطأ يُرى بالعين** حين يقع.
	f.merchantCost = subtotal * 9 / 10
	if _, err := pool.Exec(ctx, `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, qty, options)
		VALUES ($1, 'صنف اختبار', $2, $3, 1, '[]'::jsonb)`,
		f.orderID, subtotal, f.merchantCost); err != nil {
		t.Fatalf("تعذّر إنشاء بند الطلب: %v", err)
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
	// **على سعر الشراء لا سعر البيع** — بيعةُ المتجر هي ما يُحاسَب عليه.
	if got := f.platformCommission(t); got != 9_000 {
		t.Errorf("عمولة المنصة = %d، والمتوقع 9000 (10%% من 90000)", got)
	}
	// **نصيبُه من الهامش لا من العمولة** — ١٠٪ من ١٠٬٠٠٠ = ١٬٠٠٠.
	//
	// **والـ٢٪ مالٌ مرصودٌ لخسارة**: لو أخذ منها لربح على متجرٍ هامشُه صفر
	// **والمنصةُ تدفع له من جيبها.**
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
	// 2) **وعمولةُ المندوب تبقى له** — قرارُ المالك (٢٠٢٦-٠٨-٠٣):
	//
	// **«المنصة سوف تتحمل المسؤولية، ويكون النزاع بين المنصة والمتجر —
	// والمندوب لا علاقة له بذلك، ولا السائق، ولا الزبون.»**
	//
	// وكانت تُعكس بقرارٍ من المنفِّذ لا من المالك، **وهو قياسٌ يناقض نفسَه**:
	// أجرُ السائق يبقى بالحجّة المقابلة، **فصار طرفان أدّيا دورَيهما
	// يُعامَلان بقاعدتين.**
	if got := f.balance(t, f.rep); got != 1_000 {
		t.Errorf("عمولة المندوب = %d، والمتوقّع 1000 — **تبقى له**", got)
	}
	st, err := f.wallet.StatementFor(ctx, f.rep, 10)
	if err != nil {
		t.Fatalf("تعذّر كشف محفظة المندوب: %v", err)
	}
	if len(st.Transactions) != 1 {
		t.Errorf("قُيّد للمندوب ما لم يكن ينبغي — عدد القيود %d، والمتوقّع 1", len(st.Transactions))
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
	if got := f.platformCommission(t); got != 9_000 {
		t.Fatalf("عمولة المنصة = %d، والمتوقع 9000 — الملغى نصيب المندوب لا العمولة", got)
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
		INSERT INTO menu_items (merchant_id, section_id, name, merchant_price, price, available)
		VALUES ($1, $2, 'صنف', 20000, 20000, true) RETURNING id`,
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

// TestCreate_RequiresWhatsAppVerified لا طلب قبل توثيق واتساب.
//
// الرقمُ الوهميّ يعني سائقاً يقف أمام بابٍ لا أحد فيه، وطلباً نقدياً لا يُقبض،
// ومتجراً حضّر بضاعةً لا تُستلَم. **والخسارة تقع على ثلاثة أطراف لا على من كتب
// الرقم** — ولذلك الحارسُ في المحرّك لا في الواجهة: إخفاءُ زرٍّ لا يوقف من
// ينادي النقطة مباشرةً.
func TestCreate_RequiresWhatsAppVerified(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	f := setup(t, "pending", 1, 0, 0)

	// الحارس يقرأ إعداداً — وبلا حقنِ المخزن لا يعمل أصلاً (وهو ما يجعل بقية
	// اختبارات التسويات تمضي بلا توثيق).
	f.svc.SetSettings(settings.NewStore(pool))
	if _, err := pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ('customers.require_whatsapp','true'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("تعذّر ضبط الإعداد: %v", err)
	}

	var sectionID, itemID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name, sort_order) VALUES ($1, 'قسم', 1)
		RETURNING id`, f.merchantID).Scan(&sectionID); err != nil {
		t.Fatalf("قسم: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, name, merchant_price, price, available)
		VALUES ($1, $2, 'صنف', 20000, 20000, true) RETURNING id`,
		f.merchantID, sectionID).Scan(&itemID); err != nil {
		t.Fatalf("صنف: %v", err)
	}
	var zoneID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO delivery_zones (name, delivery_fee, min_order, active, center, radius_m)
		VALUES ('منطقة اختبار التوثيق', 5000, 0, true,
		        ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography, 50000)
		RETURNING id`).Scan(&zoneID); err != nil {
		t.Fatalf("منطقة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM delivery_zones WHERE id = $1`, zoneID)
	})

	// رصيدٌ يكفي — كي يكون التوثيق هو المانع الوحيد، لا نقصُ المال.
	if _, err := f.wallet.Apply(ctx, f.customer, 100_000, "topup", "", "رصيد اختبار", nil); err != nil {
		t.Fatalf("تعذّر شحن المحفظة: %v", err)
	}

	in := orders.CreateInput{
		CustomerID:    f.customer,
		MerchantID:    f.merchantID,
		Items:         []orders.ItemInput{{MenuItemID: itemID, Qty: 1}},
		AddressText:   "عنوان اختبار",
		Lat:           35.9528,
		Lng:           39.0079,
		PaymentMethod: "wallet",
	}

	if _, err := f.svc.Create(ctx, f.customer, []string{"customer"}, in, "127.0.0.1"); err == nil {
		t.Fatal("قُبل طلبٌ من حسابٍ غير موثَّق")
	}

	// ويمرّ بعد التوثيق — كي لا يمرّ الاختبار لأن كل شيءٍ مرفوض
	if _, err := pool.Exec(ctx,
		`UPDATE users SET whatsapp_verified_at = now() WHERE id = $1`, f.customer); err != nil {
		t.Fatalf("تعذّر التوثيق: %v", err)
	}
	if _, err := f.svc.Create(ctx, f.customer, []string{"customer"}, in, "127.0.0.1"); err != nil {
		t.Fatalf("رُفض طلبٌ من حسابٍ موثَّق: %v", err)
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
	// **الأساسُ سعرُ الشراء**: ٩٠٬٠٠٠ ناقصَ عمولةِ ١٠٪ = ٨١٬٠٠٠.
	// **ولا يأخذ هامشَنا معه** — هو لم يبعه بذلك السعر ولا يراه.
	if got := f.balance(t, ownerID); got != 81_000 {
		t.Fatalf("مستحقّ المتجر = %d، والمتوقع 81000 (سعرُ الشراء − عمولة)", got)
	}

	if _, err := f.svc.Transition(ctx, f.driver, []string{"admin"}, f.orderID, "refunded", ""); err != nil {
		t.Fatalf("الاسترجاع فشل: %v", err)
	}
	if got := f.balance(t, ownerID); got != 0 {
		t.Fatalf("بقي مستحقّ عن طلب مُسترجَع: %d والمتوقع 0", got)
	}
}

// أجر السائق يُقيَّد بالتسليم، **خارج** مبلغ الدَّين عليه.
//
// صندوقه يسجّل ما يدين به للمنصة، ومحفظته تسجّل ما تدين به له. خلطُهما (أن
// يسلّم المبلغ ناقصاً أجره) يجعل تسوية الصندوق غير قابلة للمطابقة: لا يعود
// مجموع ما حصّله يساوي مجموع ما سلّمه.
func TestDelivery_PaysDriverShare(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()

	// النمط الافتراضي: نسبة من رسم التوصيل (70% من 10,000 = 7,000)
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ('drivers.share_mode','"percent"'::jsonb),
		                                             ('drivers.share_value','70'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("تعذّر ضبط الإعدادات: %v", err)
	}

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}

	if got := f.balance(t, f.driver); got != 7_000 {
		t.Fatalf("أجر السائق = %d، والمتوقع 7000 (70%% من رسم التوصيل)", got)
	}
	// والدَّين كامل غير منقوص من الأجر
	if got := f.held(t); got != 110_000 {
		t.Fatalf("صندوق السائق = %d، والمتوقع 110000 كاملاً — الأجر خارج الدَّين", got)
	}
}

// النمط المقطوع: مبلغ ثابت لكل طلب مهما كان رسم التوصيل.
func TestDelivery_DriverShare_FixedMode(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()

	if _, err := f.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ('drivers.share_mode','"fixed"'::jsonb),
		                                             ('drivers.share_value','5000'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("تعذّر ضبط الإعدادات: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `
			UPDATE app_settings SET value = '"percent"'::jsonb WHERE key='drivers.share_mode';
			UPDATE app_settings SET value = '70'::jsonb WHERE key='drivers.share_value'`)
	})

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}
	if got := f.balance(t, f.driver); got != 5_000 {
		t.Fatalf("أجر السائق المقطوع = %d، والمتوقع 5000", got)
	}
}

// عتبة التفعيل: لا عمولة عن عميلٍ لم يُثبت أنه يعمل.
//
// كانت العمولة تُقيَّد من الطلب الأول، فمن يسجّل خمسين متجراً ينتج كلٌّ منها طلباً
// واحداً يقبض عن الخمسين — الكمّ يُكافأ كما يُكافأ الإنتاج.
func TestCommission_HeldUntilMerchantActivates(t *testing.T) {
	ctx := context.Background()
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)

	// **بعد** setup: هي تضبط العتبة إلى 1، فضبطُها قبلها يُمحى
	if _, err := f.pool.Exec(ctx, `
		UPDATE app_settings SET value = '3'::jsonb WHERE key = 'sales.activation_orders'`); err != nil {
		t.Fatalf("تعذّر ضبط العتبة: %v", err)
	}

	// الطلب الأول: دون العتبة (3) فلا عمولة
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم الأول فشل: %v", err)
	}
	if got := f.balance(t, f.rep); got != 0 {
		t.Fatalf("قُيّدت عمولة قبل بلوغ العتبة: %d", got)
	}

	// طلبان آخران من زبون عادي يبلغان بهما العتبة
	for i := 0; i < 2; i++ {
		id := f.extraDeliveredOrder(t, f.customer)
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, id, "delivered", ""); err != nil {
			t.Fatalf("تسليم إضافي فشل: %v", err)
		}
	}
	// الثالث بلغ العتبة فقُيّدت عنه عمولة
	if got := f.balance(t, f.rep); got <= 0 {
		t.Fatalf("لم تُقيَّد عمولة بعد بلوغ العتبة: %d", got)
	}
}

// طلبات المندوب نفسه لا تُحتسب في العتبة — وإلا فعّل عميله بيده.
func TestActivation_RepOwnOrdersDoNotCount(t *testing.T) {
	ctx := context.Background()
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)

	// **بعد** setup: هي تضبط العتبة إلى 1، فضبطُها قبلها يُمحى
	if _, err := f.pool.Exec(ctx, `
		UPDATE app_settings SET value = '3'::jsonb WHERE key = 'sales.activation_orders'`); err != nil {
		t.Fatalf("تعذّر ضبط العتبة: %v", err)
	}

	// ثلاثة طلبات اشتراها المندوب نفسه — لا تُفعّل ولا تُعطي عمولة
	for i := 0; i < 3; i++ {
		id := f.extraDeliveredOrder(t, f.rep)
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, id, "delivered", ""); err != nil {
			t.Fatalf("تسليم فشل: %v", err)
		}
	}
	if got := f.balance(t, f.rep); got != 0 {
		t.Fatalf("فعّل المندوب عميله بمشترياته: الرصيد %d والمتوقع 0", got)
	}
}

// extraDeliveredOrder ينشئ طلباً جاهزاً للتسليم على المتجر نفسه لزبونٍ معيّن.
func (f *fixture) extraDeliveredOrder(t *testing.T, customerID string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
		VALUES ($1, $2, $3, 'at_dropoff', 'عنوان', 
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 100000, 10000, 110000, 0, 110000)
		RETURNING id`, customerID, f.merchantID, f.driver).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاء طلب إضافي: %v", err)
	}
	// **وبندٌ بسعرين كالطلب الأصليّ** — وإلّا كان هامشُه صفراً فلا عمولة
	// للمندوب، **فيبدو الاختبارُ ساقطاً والسببُ زراعتُه لا الشيفرة.**
	if _, err := f.pool.Exec(context.Background(), `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, qty, options)
		VALUES ($1, 'صنف اختبار', 100000, 90000, 1, '[]'::jsonb)`, id); err != nil {
		t.Fatalf("تعذّر إنشاء بند الطلب الإضافي: %v", err)
	}
	return id
}
