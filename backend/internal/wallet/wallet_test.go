package wallet_test

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// المحفظة دفتر قيود: كل حركة سطر دائم، والرصيد محصلة مصانة — ولا رصيد سالب أبداً.
// هذه الاختبارات تحرس أخطر ما في المنصة: مال الناس.

func TestApply_CreditThenDebit(t *testing.T) {
	pool := testdb.Pool(t)
	svc := wallet.NewService(pool)
	ctx := context.Background()
	user := testdb.NewUser(t, pool, "customer")

	balance, err := svc.Apply(ctx, user, 50_000, "topup", "", "شحن", nil)
	if err != nil {
		t.Fatalf("الشحن فشل: %v", err)
	}
	if balance != 50_000 {
		t.Fatalf("الرصيد بعد الشحن = %d، والمتوقع 50000", balance)
	}

	balance, err = svc.Apply(ctx, user, -20_000, "order_payment", "ref-1", "دفع طلب", nil)
	if err != nil {
		t.Fatalf("الخصم فشل: %v", err)
	}
	if balance != 30_000 {
		t.Fatalf("الرصيد بعد الخصم = %d، والمتوقع 30000", balance)
	}

	// القيود تُحفظ كما هي — الدفتر لا يُختصر
	st, err := svc.StatementFor(ctx, user, 10)
	if err != nil {
		t.Fatalf("تعذّر كشف الحساب: %v", err)
	}
	if len(st.Transactions) != 2 {
		t.Fatalf("عدد القيود = %d، والمتوقع 2", len(st.Transactions))
	}
	if st.Balance != 30_000 {
		t.Fatalf("رصيد الكشف = %d، والمتوقع 30000", st.Balance)
	}
}

func TestApply_RejectsOverdraft(t *testing.T) {
	pool := testdb.Pool(t)
	svc := wallet.NewService(pool)
	ctx := context.Background()
	user := testdb.NewUser(t, pool, "customer")

	if _, err := svc.Apply(ctx, user, 10_000, "topup", "", "", nil); err != nil {
		t.Fatalf("الشحن فشل: %v", err)
	}

	// خصم يتجاوز الرصيد يجب أن يُرفض — لا رصيد سالب في المنصة
	_, err := svc.Apply(ctx, user, -10_001, "payout", "", "", nil)
	if !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("الخصم الزائد لم يُرفض بـinsufficient_balance بل: %v", err)
	}

	// ولا يترك أثراً: لا الرصيد تغيّر ولا قيد كُتب
	balance, err := svc.Balance(ctx, user)
	if err != nil {
		t.Fatalf("تعذّرت قراءة الرصيد: %v", err)
	}
	if balance != 10_000 {
		t.Fatalf("الرصيد تأثّر بخصم مرفوض: %d", balance)
	}
	st, _ := svc.StatementFor(ctx, user, 10)
	if len(st.Transactions) != 1 {
		t.Fatalf("قيد يتيم لخصم مرفوض — عدد القيود %d", len(st.Transactions))
	}
}

func TestApply_RejectsZero(t *testing.T) {
	pool := testdb.Pool(t)
	svc := wallet.NewService(pool)
	user := testdb.NewUser(t, pool, "customer")

	if _, err := svc.Apply(context.Background(), user, 0, "adjustment", "", "", nil); !errors.Is(err, wallet.ErrInvalidAmount) {
		t.Fatalf("حركة بصفر لم تُرفض: %v", err)
	}
}

// ApplyTx داخل معاملة المستدعي: فشل المعاملة يلغي الحركة المالية معها.
// هذا جوهر إصلاح R-02 — كانت التسويات تنجح وحدها ويبقى المال معلّقاً.
func TestApplyTx_RollsBackWithCallerTransaction(t *testing.T) {
	pool := testdb.Pool(t)
	svc := wallet.NewService(pool)
	ctx := context.Background()
	user := testdb.NewUser(t, pool, "sales")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("تعذّر بدء المعاملة: %v", err)
	}
	if _, err := svc.ApplyTx(ctx, tx, user, 7_000, "commission", "order-x", "عمولة", nil); err != nil {
		t.Fatalf("ApplyTx فشل: %v", err)
	}
	// المستدعي تراجع — يجب ألّا يبقى للحركة أثر
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("التراجع فشل: %v", err)
	}

	balance, err := svc.Balance(ctx, user)
	if err != nil {
		t.Fatalf("تعذّرت قراءة الرصيد: %v", err)
	}
	if balance != 0 {
		t.Fatalf("حركة نجت من تراجع معاملة المستدعي — الرصيد %d", balance)
	}
}

// كشف الحساب مستند مالي: من يراجعه يجمع أسطره، فإن لم تُغلق المعادلة
// (افتتاحي + حركات = ختامي) فقد الثقة بالمنصة كلها. وهذا يجب أن يصحّ **حتى
// حين يُقصّ الكشف** عند السقف — وهي الحالة التي يسهل أن تنكسر بصمت.
func TestStatement_BalancesEvenWhenTruncated(t *testing.T) {
	pool := testdb.Pool(t)
	svc := wallet.NewService(pool)
	ctx := context.Background()
	user := testdb.NewUser(t, pool, "customer")

	for i := 0; i < 5; i++ {
		if _, err := svc.Apply(ctx, user, 10_000, "topup", "", "شحن", nil); err != nil {
			t.Fatalf("الشحن %d فشل: %v", i, err)
		}
	}

	// كشف كامل
	full, err := svc.Statement(ctx, user, wallet.StatementRange{Limit: 10})
	if err != nil {
		t.Fatalf("تعذّر الكشف الكامل: %v", err)
	}
	if full.Truncated {
		t.Fatal("الكشف الكامل أُعلن مقصوصاً وهو ليس كذلك")
	}
	if got := sumTx(full.Transactions); full.Opening+got != full.Closing {
		t.Fatalf("الكشف الكامل لا يتوازن: %d + %d ≠ %d", full.Opening, got, full.Closing)
	}

	// كشف مقصوص: ثلاثة أسطر من خمسة
	cut, err := svc.Statement(ctx, user, wallet.StatementRange{Limit: 3})
	if err != nil {
		t.Fatalf("تعذّر الكشف المقصوص: %v", err)
	}
	if len(cut.Transactions) != 3 {
		t.Fatalf("عدد الأسطر = %d، والمتوقع 3", len(cut.Transactions))
	}
	if !cut.Truncated {
		t.Fatal("الكشف قُصّ ولم يُعلن ذلك — كشفٌ ناقص يصمت عن نقصه أسوأ من لا كشف")
	}
	// الافتتاحي يُحسب من المعروض لا من المدى، فيبقى الجمع صحيحاً بين يدَي المراجع
	if got := sumTx(cut.Transactions); cut.Opening+got != cut.Closing {
		t.Fatalf("الكشف المقصوص لا يتوازن: %d + %d ≠ %d", cut.Opening, got, cut.Closing)
	}
	if cut.Opening != 20_000 {
		t.Fatalf("الافتتاحي المقصوص = %d، والمتوقع 20000 (سطران لم يُعرضا)", cut.Opening)
	}
}

func sumTx(txs []wallet.Transaction) int64 {
	var s int64
	for _, t := range txs {
		s += t.Amount
	}
	return s
}
