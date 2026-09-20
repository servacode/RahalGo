package qa

// ══════════════════════════════════════════════════════════════════════
// **المحرّكُ هو الحاكمُ الأخير للدفع بالمحفظة** (`CUST-12-023` / `CUST-WAL`)
// ══════════════════════════════════════════════════════════════════════
//
// **واجهةُ السلّةِ تُعطّل الخيارَ حين ينقص الرصيدُ** (`CustWalletUxTest`)،
// **لكن لا يُوثَق بالعميلِ وحدَه** — **فلو تسلّل طلبُ محفظةٍ برصيدٍ ناقصٍ
// إلى المحرّك** وجب أن يُردَّ **بلا طلبٍ يُنشأ ولا قرشٍ يُخصَم.**
//
// **والوحدةُ تختبر رفضَ الخصمِ الزائد** (`TestApply_RejectsOverdraft`)،
// **لكن لا اختبارَ لمسارِ الطلبِ كاملاً**: طلبٌ بالمحفظةِ ورصيدٌ < الإجماليّ
// ⇒ `409 insufficient_balance`، **ولا صفَّ طلبٍ، والرصيدُ كما كان** (ذرّيّة).

import (
	"net/http"
	"testing"
)

// TestCUST12023_WalletOrderRejectedWhenInsufficient **رصيدٌ ناقصٌ ⇒ رفضٌ
// ذرّيّ.**
func TestCUST12023_WalletOrderRejectedWhenInsufficient(t *testing.T) {
	h := New(t)
	f := h.Factory()

	cust := h.Customer()
	// **رصيدٌ موجودٌ لكنّه ناقص** — كحالِ من عنده ١٥ وطلبُه أضعافُها.
	const funded int64 = 5_000
	f.Credit(cust.ID, funded, "topup")

	// **صنفٌ أغلى من الرصيد** — فالإجماليُّ (صنف + توصيل) > الرصيدِ يقيناً.
	item := h.NewItem(20_000)
	body := orderBody(item, 1)
	body["payment_method"] = "wallet"

	res := h.POSTKey("/api/v1/orders", cust.Token, uniq("wal-insuf"), body)

	// ١ · يُردُّ بـ `409 insufficient_balance`.
	if res.Code != http.StatusConflict || res.Err() != "insufficient_balance" {
		t.Fatalf("**طلبُ محفظةٍ برصيدٍ ناقصٍ لم يُردَّ صحيحاً**: %d / %q — %s",
			res.Code, res.Err(), res)
	}

	// ٢ · ولا صفَّ طلبٍ نُشئ (لا شبحَ طلبٍ في القاعدة).
	var orders int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID).Scan(&orders); err != nil {
		t.Fatalf("عدُّ الطلبات: %v", err)
	}
	if orders != 0 {
		t.Fatalf("**رُفض الدفعُ وبقي طلبٌ**: صفوف=%d — **فشبحُ طلبٍ يُسنَد ولا يُدفَع**", orders)
	}

	// ٣ · والرصيدُ كما كان — لا خصمَ (تراجعت المعاملةُ كاملةً).
	var bal int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT balance FROM wallets WHERE user_id = $1::uuid`, cust.ID).Scan(&bal); err != nil {
		t.Fatalf("قراءةُ الرصيد: %v", err)
	}
	if bal != funded {
		t.Fatalf("**خُصم من رصيدٍ ناقصٍ رغمَ الرفض**: %d (المنتظَر %d)", bal, funded)
	}
}
