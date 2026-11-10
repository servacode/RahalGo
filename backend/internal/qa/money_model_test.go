package qa

// ══════════════════════════════════════════════════════════════════════
// **نموذجُ المال على الطلب — يُختَم عند الإنشاء** (خطّةُ المال ٢٠٢٦-١١-١٠، الدفعة B1)
// ══════════════════════════════════════════════════════════════════════
//
// `orders.money_model`: ١ = القديم (البضاعةُ في الدفتر) · ٢ = الجديد (البضاعةُ خارجَه).
// **والمفتاحُ `finance.goods_outside` مطفأٌ افتراضاً** فكلُّ طلبٍ جديدٍ ١ كما كان. **ومشغَّلاً
// يُختَم كلُّ طلبٍ جديدٍ ٢ من أبوابه الأربعة** (عاديّ · خاصّ · «لدي توصيلة» · واتساب)، **والقديمُ
// يبقى ١.** ولا شيءَ يقرأ ٢ بعد — هذه الدفعةُ تضيف ولا تغيّر سلوكَ مال.

import (
	"net/http"
	"testing"
)

func moneyModelOf(t *testing.T, h *Harness, oid string) int {
	t.Helper()
	var m int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT money_model FROM orders WHERE id = $1::uuid`, oid).Scan(&m); err != nil {
		t.Fatalf("قراءةُ نموذج المال: %v", err)
	}
	return m
}

func TestMoneyModel_StampedAtCreation(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	w := newWAFx(t, h, "منطقةُ نموذج المال")

	standard := func(key string) string {
		item := h.NewItem(1000)
		made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq(key), orderBody(item, 1))
		if made.Code != http.StatusCreated {
			t.Fatalf("إنشاءُ الطلب العاديّ: %s", made)
		}
		id, _ := made.JSON()["id"].(string)
		return id
	}

	// ── ١ · المفتاحُ مطفأ (الافتراض) ⇒ ١ ─────────────────────────────
	old := standard("mm-old")
	if got := moneyModelOf(t, h, old); got != 1 {
		t.Fatalf("**طلبٌ والمفتاحُ مطفأ نموذجُه %d — يُنتظر ١**", got)
	}

	// ── ٢ · المفتاحُ مشغَّل ⇒ ٢ من كلّ باب ───────────────────────────
	h.Setting("finance.goods_outside", "true")
	ops := h.NewUser("operations")
	made := map[string]string{
		"عاديّ":      standard("mm-new"),
		"خاصّ":       newCustomOrder(t, h, h.Customer()),
		"لدي توصيلة": w.md.create(t, h, "recipient", uniq("mm-md")),
		"طلب واتساب": waCreated(t, waPost(t, h, ops.Token, w.body())),
	}
	for kind, id := range made {
		if got := moneyModelOf(t, h, id); got != 2 {
			t.Errorf("**%s والمفتاحُ مشغَّل نموذجُه %d — يُنتظر ٢**", kind, got)
		}
	}

	// ── ٣ · القديمُ لا يُمسّ ─────────────────────────────────────────
	if got := moneyModelOf(t, h, old); got != 1 {
		t.Errorf("**طلبٌ قديمٌ تبدّل نموذجُه إلى %d بتشغيل المفتاح**", got)
	}
}
