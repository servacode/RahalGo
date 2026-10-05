package qa

import (
	"testing"
)

// TestFIN_FI06d_RejectedWalletOrderIsNotAViolation **طلبُ محفظةٍ رُفض قبل
// الاستلام ليس خرقاً.**
//
// (كشفه فحصُ المتصفّح للمال ٢٠٢٦-١٠-٠٥: `moneycheck` أعلن خرقاً على طلبٍ
// رفضته العمليّات — خصمٌ ثمّ استرجاعٌ بالمقدار نفسِه، **وهو الصواب**.)
//
// الخصمُ عند الإنشاء والاسترجاعُ عند الرفض قيدان صحيحان، **والفحصُ كان
// يجمعهما فيقرأ صفراً ويقارنه بسالب المدفوع.** فكلُّ طلبِ محفظةٍ أُلغي أو
// رُفض قبل الاستلام كان يُسقط `moneycheck` بلا ذنب.
func TestFIN_FI06d_RejectedWalletOrderIsNotAViolation(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h, "FI-06.d")

	cust := h.Customer()
	f := h.Factory()
	f.Credit(cust.ID, 100_000, "topup")
	item := h.NewItem(1000)

	body := orderBody(item, 2)
	body["payment_method"] = "wallet"
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), body)
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "rejected", "note": "المطبخ مغلق"}); got.Code >= 400 {
		t.Fatalf("الرفضُ رُدّ: %s", got)
	}

	var refunded int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'refund'`, oid).Scan(&refunded)
	if refunded <= 0 {
		t.Fatalf("لا استرجاعَ بعد الرفض — السيناريو لم يقع")
	}
	assertNewViolations(t, h, base, "FI-06.d")
}
