package orders_test

// **نصيبُ المنصة بعد استرجاعِ طلبٍ نقديٍّ مُسلَّم — بالحساب لا بالانطباع.**
//
// # الحادثة
//
// استرجع المالكُ `#1003` في تجربةٍ حيّة (٢٠٢٦-٠٨-٠٣): طلبٌ نقديٌّ قدرُه
// ٢٢٬٠٠٠. **فخسرت الخزينةُ ٣٩٬٨٠٠** — أكثرَ من قيمة الطلب كلِّه، **وأجرُ
// السائق وحدَه سبعةُ آلاف.**
//
// وعلّتُه اثنتان اجتمعتا في معادلةٍ واحدة:
//
//	delta = (paid − refunded − toParties) − posted
//
//  ١ · **القبضُ يُنسى.** `paid` تضيف `cash_due` حين تكون الحالةُ `delivered`
//      وحدَها — **والحالةُ صارت `refunded` قبل أن تُقرأ**، والمالُ في صندوق
//      السائق فعلاً. فتُقيَّد الخسارةُ مرّتين: بأنّ المالَ لم يدخل، وبأنّه رُدّ.
//
//  ٢ · **العكسُ لا يُرى.** `reverseCommissions` كان يكتب عكسَ مستحقّ المتجر
//      بنوع `adjustment`، **والحسبةُ تجمع `merchant_earning` و`driver_earning`
//      و`commission` ولا ترى `adjustment`** — فيبقى في دفترها أنّها دفعت
//      للمتجر وقد استردّت.
//
// # ولماذا لم يُمسَك قبل اليوم
//
// **لأنّ الخزينةَ لم تُفحص في اختبارٍ قطّ.** كلُّ اختبارات التسوية تبني الخدمةَ
// بلا `settings`، **و`treasuryID` يردّ فراغاً فيخرج `creditTreasury` صامتاً**
// قبل أن يحسب شيئاً. فكان يُختبَر كلُّ طرفٍ إلّا الطرفَ الرابع.

import (
	"context"
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

// treasuryFixture ميدانٌ فيه خزينةٌ حقيقية — **وهو ما نقص الاختباراتِ كلَّها.**
type treasuryFixture struct {
	pool     *pgxpool.Pool
	svc      *orders.Service
	wallet   *wallet.Service
	customer string
	driver   string
	owner    string
	treasury string
	orderID  string
}

func newTreasuryFixture(t *testing.T, subtotal, deliveryFee int64) *treasuryFixture {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	store := settings.NewStore(pool)
	f := &treasuryFixture{
		pool:     pool,
		wallet:   wallet.NewService(pool),
		customer: testdb.NewUser(t, pool, "customer"),
		driver:   testdb.NewUser(t, pool, "driver"),
		owner:    testdb.NewUser(t, pool, "merchant"),
		treasury: testdb.NewUser(t, pool, "admin"),
	}
	f.svc = orders.NewService(pool, nil, f.wallet, cashbox.NewService(pool, store), nil, quiet)
	f.svc.SetSettings(store)

	if _, err := pool.Exec(ctx,
		`UPDATE wallets SET is_treasury = true WHERE user_id = $1`, f.treasury); err != nil {
		t.Fatalf("تعذّر وسمُ الخزينة: %v", err)
	}
	// **وتُطوى خزينةُ الاختبار طيّاً كاملاً.**
	//
	// `wallets_balance_check` يسمح بالرصيد السالب **للخزينة وحدَها**، فنزعُ
	// الصفة عن خزينةٍ خسرت يُسقط القيد. **وهو تنبيهٌ حقيقيٌّ لا عارضُ اختبار**:
	// تبديلُ حساب الخزينة ورصيدُ القديم غيرُ صفريّ يُعطّل أوّلَ تسويةٍ بعده.
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM wallet_transactions WHERE user_id = $1`, f.treasury)
		_, _ = pool.Exec(c, `DELETE FROM wallets WHERE user_id = $1`, f.treasury)
		_, _ = pool.Exec(c, `UPDATE wallets SET is_treasury = false WHERE user_id = $1`, f.treasury)
	})

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات في القاعدة: %v", err)
	}
	var merchantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, owner_user_id, commission_percent)
		VALUES ('متجر اختبار الخزينة', $1, $2, 10) RETURNING id`,
		categoryID, f.owner).Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID)
	})

	// **طلبٌ نقديٌّ عند باب الزبون** — يُسلَّم بالمحرّك لا يُزرع مُسلَّماً،
	// **وإلّا لم تُقيَّد أنصبتُه أصلاً فيُقاس استرجاعٌ لتسويةٍ لم تقع.**
	total := subtotal + deliveryFee
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
		VALUES ($1, $2, $3, 'at_dropoff', 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', $4, $5, $6, 0, $6)
		RETURNING id`,
		f.customer, merchantID, f.driver, subtotal, deliveryFee, total).Scan(&f.orderID); err != nil {
		t.Fatalf("تعذّر إنشاء طلب: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, qty, options)
		VALUES ($1, 'صنف اختبار', $2, $3, 1, '[]'::jsonb)`,
		f.orderID, subtotal, subtotal*9/10); err != nil {
		t.Fatalf("تعذّر إنشاء بند الطلب: %v", err)
	}
	return f
}

// deliverThenRefund يمشي الطلبَ في مساره الحقيقيّ: **تسليمٌ يُقيّد الأنصبة،
// ثمّ استرجاعٌ يعكسها.** ولا معنى لقياس عكسٍ لم يسبقه قيد.
func (f *treasuryFixture) deliverThenRefund(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
		f.orderID, "delivered", ""); err != nil {
		t.Fatalf("تعذّر التسليم: %v", err)
	}
	if _, err := f.svc.Transition(ctx, f.treasury, []string{"admin"},
		f.orderID, "refunded", "اختبارُ الاسترجاع"); err != nil {
		t.Fatalf("تعذّر الاسترجاع: %v", err)
	}
}

func (f *treasuryFixture) sumKind(t *testing.T, kind string) int64 {
	t.Helper()
	var v int64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = $2`, f.orderID, kind).Scan(&v); err != nil {
		t.Fatalf("تعذّرت قراءة قيود %s: %v", kind, err)
	}
	return v
}

// TestRefund_TreasuryLosesOnlyWhatItPaid **الخزينةُ تخسر ما دفعته لا أضعافه.**
//
// طلبٌ نقديٌّ قدرُه ٢٢٬٠٠٠ سُلّم ثمّ استُرجع:
//
//	+22,000  نقدٌ قبضه السائقُ وهو في صندوقه (مالُ المنصة)
//	−22,000  رُدّ إلى محفظة الزبون
//	−     0  المتجر — عُكس مستحقُّه
//	− أجرُ السائق — رحلةٌ أدّاها فلا تُسترَدّ منه
//
// **فنصيبُها = −أجرُ السائق.** لا أكثر.
func TestRefund_TreasuryLosesOnlyWhatItPaid(t *testing.T) {
	f := newTreasuryFixture(t, 15000, 7000)
	f.deliverThenRefund(t)

	refunded := f.sumKind(t, "refund")
	merchantNet := f.sumKind(t, "merchant_earning")
	driverPaid := f.sumKind(t, "driver_earning")
	treasury := f.sumKind(t, "platform_profit")

	if refunded != 22000 {
		t.Fatalf("رُدّ للزبون %d لا 22,000 — والزبونُ لا يخسر", refunded)
	}
	// **صافي مستحقّ المتجر صفرٌ** — قُيّد ثمّ عُكس في حسابه هو.
	if merchantNet != 0 {
		t.Fatalf("بقي للمتجر %d عن طلبٍ مُسترجَع — والعكسُ يجب أن يقع في حسابه\n"+
			"ولو كُتب `adjustment` لَبقي المبلغُ في هذا الجمع ولم تره الخزينة", merchantNet)
	}

	want := -driverPaid
	if treasury != want {
		t.Fatalf("نصيبُ الخزينة %d والصوابُ %d — فارقٌ قدرُه %d\n"+
			"  المقبوضُ نقداً:      22,000 (في صندوق السائق — لا يُمحى بالاسترجاع)\n"+
			"  المردودُ للزبون:    %d\n"+
			"  صافي المتجر:        %d\n"+
			"  أجرُ السائق:        %d\n"+
			"**والخسارةُ تُقيَّد مرّةً لا مرّتين.**",
			treasury, want, treasury-want, refunded, merchantNet, driverPaid)
	}
	if driverPaid <= 0 {
		t.Fatalf("أجرُ السائق %d — ورحلةٌ أُدّيت تُدفع", driverPaid)
	}
}

// TestRefund_DriverKeepsHisFee **الناقلُ يُدفع له عن رحلةٍ أدّاها.**
//
// والاسترجاعُ كلفةُ منصّةٍ لا تُسترَدّ ممّن قاد — **وهو المتعارَف عليه في
// منصّات التوصيل.** ولو عُكس أجرُه لَصار السائقُ يتحمّل خلافاً بين زبونٍ ومتجر
// **لا يدَ له فيه**، ولا يستطيع أن يعرف قبل الرحلة أيُسترجَع الطلبُ أم لا.
func TestRefund_DriverKeepsHisFee(t *testing.T) {
	f := newTreasuryFixture(t, 15000, 7000)
	f.deliverThenRefund(t)

	bal, err := f.wallet.Balance(context.Background(), f.driver)
	if err != nil {
		t.Fatalf("تعذّرت قراءة رصيد السائق: %v", err)
	}
	if bal <= 0 {
		t.Fatalf("رصيدُ السائق %d بعد استرجاعِ طلبٍ سلّمه — **قاد وسلّم فعلاً**", bal)
	}
}
