package orders_test

// مساعدا قياسِ المال في اختبارات نصيب المنصّة.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// cqArmTreasury **يُعِدُّ خزينةَ المنصّة** — بلا خزينةٍ لا تُقاس حصّتُها.
//
// **و`creditTreasury` تعود صامتةً إن لم توجد** (`treasuryOn` تردّ فراغاً):
// **فاختبارٌ بلا خزينةٍ يقرأ صفراً ويظنُّه «لم تأخذ»**، **وهو في الحقيقة
// «لا مكانَ تأخذ إليه».** والفرقُ بينهما كلُّ شيء.
func cqArmTreasury(t *testing.T) {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	// **وواحدةٌ لا اثنتان** — `treasuryOn` تأخذ الأولى، **وخزينتان تجعلان
	// القياسَ يعتمد على ترتيبِ صفوفٍ لا يضمنه أحد.**
	if _, err := pool.Exec(ctx, `UPDATE wallets SET is_treasury = false WHERE is_treasury`); err != nil {
		t.Fatalf("تعذّر تنظيفُ الخزائن: %v", err)
	}
	var uid string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (phone, full_name)
		VALUES ('+9639' || lpad((nextval('test_phone_seq') % 100000000)::text, 8, '0'),
		        'خزينةُ اختبار')
		RETURNING id::text`).Scan(&uid); err != nil {
		t.Fatalf("تعذّر إنشاءُ حاملِ الخزينة: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO wallets (user_id, balance, is_treasury) VALUES ($1, 0, true)
		ON CONFLICT (user_id) DO UPDATE SET is_treasury = true`, uid); err != nil {
		t.Fatalf("تعذّر إعدادُ الخزينة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`UPDATE wallets SET is_treasury = false WHERE user_id = $1`, uid)
	})
}

// cqTreasury رصيدُ خزينةِ المنصّة.
//
// **وتُقرأ بعلَمِها لا باسمٍ ولا بمعرّفٍ ثابت** (`is_treasury`) — **ومعرّفٌ
// مكتوبٌ في اختبارٍ يفترق عن القاعدة يومَ تُبذَر من جديد.**
//
// **ولا خزينةَ بعدُ تُقرأ صفراً** — وهي الحالُ قبل أوّل قيد.
func cqTreasury(t *testing.T) int64 {
	t.Helper()
	pool := testdb.Pool(t)
	var v int64
	if err := pool.QueryRow(context.Background(),
		`SELECT COALESCE(sum(balance), 0) FROM wallets WHERE is_treasury`).Scan(&v); err != nil {
		t.Fatalf("تعذّرت قراءةُ الخزينة: %v", err)
	}
	return v
}

// cqLedgerSum مجموعُ قيودِ الطلب — **ويجب أن يكون صفراً.**
//
// # لماذا المجموعُ لا الأطراف
//
// **ولا يكفي أن يبدو كلُّ طرفٍ صحيحاً**: ليرةٌ تُخصم من الزبون ولا تصل
// أحداً **لا تظهر في أيّ طرفٍ على حدة**، **وتظهر في المجموع وحدَه.**
//
// **وهو ما يجعل الدفترَ ثنائيَّ القيد بنيويّاً لا فحصيّاً** — (نصُّ المالك
// ٢٠٢٦-٠٩-٢٩: «المعادلة يجب أن تغلق: −17,000 + 16,500 + 500 = 0»).
func cqLedgerSum(t *testing.T, orderID string) int64 {
	t.Helper()
	pool := testdb.Pool(t)
	var v int64
	if err := pool.QueryRow(context.Background(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions WHERE ref = $1`,
		orderID).Scan(&v); err != nil {
		t.Fatalf("تعذّر جمعُ قيود الطلب: %v", err)
	}
	return v
}
