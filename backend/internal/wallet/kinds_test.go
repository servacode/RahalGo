package wallet_test

// **كلُّ نوعٍ يكتبه المحرّكُ تقبله القاعدة.**
//
// (كشفه قياسٌ على خادم المالك ٢٠٢٦-٠٨-١٦: **`reward` و`penalty` يُكتبان في
//  الشيفرة ويردّهما قيدُ `kind`** — فكلُّ مكافأةِ دعوةٍ أو هدفٍ تُردّ.)
//
// # لماذا حارسٌ لا انتباه
//
// **نوعٌ جديدٌ يُكتب في Go بسطر، وقبولُه في القاعدة يحتاج هجرة** — وبينهما
// لا بناءٌ يشتكي ولا تحقّقُ أنواعٍ يمنع. **والخطأُ يظهر وقتَ التشغيل عند
// أوّل نداء.**
//
// **ولم يظهر شهرين**: لم يُسلَّم طلبٌ لمدعوٍّ برمز، ولم تُمنح مكافأةُ هدف.
// **وما لا يُجرَّب يُقرأ سليماً وهو معطوب.**
//
// **وأخطرُ من رسالة خطأ**: قيدُ الدعوة داخلَ معاملةِ تسليم الطلب —
// **فإرجاعُها يعني أن يفشل تسليمُ طلبٍ لأنّ صاحبَه جاء بدعوة.**

import (
	"context"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// walletKinds **ما تكتبه الشيفرةُ فعلاً** — يُقرأ من مصادره لا من ذاكرتي.
//
// **ومن أضاف نوعاً غداً يضيفه هنا** — والفحصُ يقول له إن نسي الهجرة.
var walletKinds = []string{
	"topup", "order_payment", "refund", "compensation", "commission",
	"merchant_earning", "driver_earning", "payout", "adjustment",
	"platform_profit", "platform_expense", "operating_expense",
	// **وهذان هما اللذان كانا مرفوضين** (٢٠٢٦-٠٨-١٦).
	"reward", "penalty",
}

func TestWalletKinds_AllAcceptedByTheDatabase(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	user := testdb.NewUser(t, pool, "customer")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM wallet_transactions WHERE user_id = $1`, user)
	})

	var rejected []string
	for _, kind := range walletKinds {
		// **ويُجرَّب بالكتابة لا بقراءة نصّ القيد** — **ونصٌّ يُقرأ ويُقارَن
		// يمرّ بفارقِ مسافةٍ أو ترتيب**، والكتابةُ تحكم كما يحكم التشغيل.
		if _, err := pool.Exec(ctx, `
			INSERT INTO wallet_transactions (user_id, amount, kind)
			VALUES ($1, 1, $2)`, user, kind); err != nil {
			if strings.Contains(err.Error(), "wallet_transactions_kind_check") {
				rejected = append(rejected, kind)
				continue
			}
			t.Fatalf("تعذّر القيدُ %q لسببٍ آخر: %v", kind, err)
		}
	}
	if len(rejected) > 0 {
		t.Fatalf("أنواعٌ تكتبها الشيفرةُ وتردّها القاعدة: %s\n"+
			"   **ولا بناءٌ يشتكي ولا تحقّقُ أنواعٍ يمنع** — والخطأُ يظهر عند "+
			"أوّل نداءٍ حيّ. أضف هجرةً توسّع `wallet_transactions_kind_check`.",
			strings.Join(rejected, "، "))
	}
}
