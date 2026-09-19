package qa

// ══════════════════════════════════════════════════════════════════════
//  CUST-DEF-003 — **متجرُ الطلب من أصنافه لا من العميل**
// ══════════════════════════════════════════════════════════════════════
//
// (بوّابةُ العوائق · `CUSTOMER-ACCEPTANCE-MASTER.md` §40.5.)
//
// # العيب
//
// **`POST /orders` كان يقبل `merchant_id` من العميل** ويجعله متجرَ الطلب:
// تُفحَص ساعاتُه، وتُبنى عليه عمولةُ المندوب وعكسُها وعدُّ التفعيل والإشعارُ
// والتوجيه (`orders.merchant_id` — انظر `transitions.go`/`notify.go`). **فمن
// صاغ الطلبَ بيده وسمّى متجراً غريباً عن أصنافه وجّه هذه كلَّها إلى متجرٍ لا
// علاقة له بالبضاعة** — حدُّ ثقةٍ مخروقٌ على حدٍّ ماليّ.
//
// # العقدُ بعد الإصلاح
//
// **متجرُ الطلب يُشتقّ من صفوف `menu_items` الحقيقيّة** (`SourcesOf`). المعرّفُ
// المُرسَلُ يجب أن يكون أحدَ مصادر الأصناف وإلّا رُفض `bad_merchant`، **فلا
// يصير متجرُ الطلب أبداً إلّا متجرَ أصنافه.** والطلبُ الوحيدُ (تطبيقُ الزبون)
// لا يرسل المعرّفَ أصلاً.

import (
	"testing"
)

// storedMerchant **متجرُ آخرِ طلبٍ لهذا الزبون — من القاعدة لا من الردّ**
// (الردُّ يُنقّي `merchant_id` عن الزبون).
func storedMerchant(t *testing.T, h *Harness, customerID string) string {
	t.Helper()
	var m string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT merchant_id::text FROM orders WHERE customer_id=$1 ORDER BY created_at DESC LIMIT 1`,
		customerID).Scan(&m); err != nil {
		t.Fatalf("متجرُ الطلب: %v", err)
	}
	return m
}

func orderCount(t *testing.T, h *Harness, customerID string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM orders WHERE customer_id=$1`, customerID).Scan(&n); err != nil {
		t.Fatalf("عدُّ الطلبات: %v", err)
	}
	return n
}

func closeMerchant(t *testing.T, h *Harness, merchantID string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE merchants SET emergency_closed=true WHERE id=$1::uuid`, merchantID); err != nil {
		t.Fatalf("إغلاقُ المتجر: %v", err)
	}
}

func bodyWithMerchant(it *Item, qty int, merchantID string) map[string]any {
	b := orderBody(it, qty)
	b["merchant_id"] = merchantID
	return b
}

// ── ١ · معرّفٌ غريبٌ يُرفض ولا يُنشئ طلباً ────────────────────────────
func TestCDEF003_ForeignMerchantIsRejected(t *testing.T) {
	h := New(t)
	u := h.Customer()
	real := h.NewItem(1000)    // أصنافٌ من المتجر الحقيقيّ A
	foreign := h.NewItem(1000) // متجرٌ آخرُ B لا صنفَ له في السلّة

	cases := []struct {
		id, merchant string
	}{
		{"foreign-active-store", foreign.MerchantID},
		{"random-uuid", "00000000-0000-0000-0000-000000000000"},
		{"garbage", "not-a-uuid"},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			before := orderCount(t, h, u.ID)
			res := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), bodyWithMerchant(real, 1, c.merchant))
			if res.Code < 400 {
				t.Fatalf("**قُبل معرّفُ متجرٍ غريب** — %s", res)
			}
			if res.Err() != "bad_merchant" {
				t.Errorf("رمزٌ غيرُ متوقّع: %s (المنتظَر bad_merchant)", res.Err())
			}
			if after := orderCount(t, h, u.ID); after != before {
				t.Errorf("**أُنشئ طلبٌ رغم الرفض**: %d ⇐ %d", before, after)
			}
		})
	}
}

// ── ٢ · الطلبُ المشروعُ يخزّن متجرَ أصنافه ────────────────────────────
func TestCDEF003_LegitimateOrderStoresItemMerchant(t *testing.T) {
	h := New(t)
	real := h.NewItem(1000)

	t.Run("merchant_id omitted", func(t *testing.T) {
		u := h.Customer()
		res := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), orderBody(real, 1))
		if res.Code >= 400 {
			t.Fatalf("**رُفض طلبٌ مشروعٌ بلا معرّف** — %s", res)
		}
		if got := storedMerchant(t, h, u.ID); got != real.MerchantID {
			t.Errorf("**متجرُ الطلب ليس متجرَ الأصناف**: %s ≠ %s", got, real.MerchantID)
		}
	})

	t.Run("correct merchant_id supplied", func(t *testing.T) {
		u := h.Customer()
		res := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), bodyWithMerchant(real, 1, real.MerchantID))
		if res.Code >= 400 {
			t.Fatalf("**رُفض معرّفٌ صحيحٌ مطابقٌ للأصناف** — %s", res)
		}
		if got := storedMerchant(t, h, u.ID); got != real.MerchantID {
			t.Errorf("**متجرُ الطلب تبدّل**: %s ≠ %s", got, real.MerchantID)
		}
	})
}

// ── ٣ · فحصُ الدوام على المتجر الحقيقيّ لا المُرسَل ───────────────────
func TestCDEF003_OpenHoursUsesRealStoreNotSupplied(t *testing.T) {
	h := New(t)

	t.Run("real CLOSED + foreign OPEN supplied -> not orderable", func(t *testing.T) {
		u := h.Customer()
		real := h.NewItem(1000)
		foreign := h.NewItem(1000) // مفتوحٌ (لا دوام)
		closeMerchant(t, h, real.MerchantID)
		before := orderCount(t, h, u.ID)
		// **المتجرُ المفتوحُ الغريبُ لا يجعل الطلبَ ممكناً** — يُرفض قبل الدوام.
		res := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), bodyWithMerchant(real, 1, foreign.MerchantID))
		if res.Code < 400 || res.Err() != "bad_merchant" {
			t.Errorf("**متجرٌ مفتوحٌ غريبٌ جعل طلباً من متجرٍ مغلقٍ ممكناً** — %s", res)
		}
		if orderCount(t, h, u.ID) != before {
			t.Error("**أُنشئ طلبٌ لمتجرٍ مغلق**")
		}
	})

	t.Run("real CLOSED + no merchant -> merchant_closed (real store governs)", func(t *testing.T) {
		u := h.Customer()
		real := h.NewItem(1000)
		closeMerchant(t, h, real.MerchantID)
		res := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), orderBody(real, 1))
		if res.Code < 400 || res.Err() != "merchant_closed" {
			t.Errorf("**الدوامُ لم يُفحَص على المتجر الحقيقيّ** — %s (المنتظَر merchant_closed)", res)
		}
	})

	t.Run("real OPEN + foreign CLOSED supplied -> foreign cannot govern", func(t *testing.T) {
		u := h.Customer()
		real := h.NewItem(1000) // مفتوحٌ
		foreign := h.NewItem(1000)
		closeMerchant(t, h, foreign.MerchantID) // الغريبُ مغلقٌ — يجب ألّا يؤثّر
		res := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), bodyWithMerchant(real, 1, foreign.MerchantID))
		// **الغريبُ لا يُسمّى أصلاً** — يُرفض bad_merchant، لا يُقرأ دوامُه.
		if res.Code < 400 || res.Err() != "bad_merchant" {
			t.Errorf("**متجرٌ غريبٌ مغلقٌ دخل قرارَ الطلب** — %s", res)
		}
	})

	t.Run("real OPEN + no merchant -> ok", func(t *testing.T) {
		u := h.Customer()
		real := h.NewItem(1000)
		res := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), orderBody(real, 1))
		if res.Code >= 400 {
			t.Fatalf("**رُفض طلبٌ مشروعٌ لمتجرٍ مفتوح** — %s", res)
		}
		if got := storedMerchant(t, h, u.ID); got != real.MerchantID {
			t.Errorf("متجرُ الطلب ليس متجرَ الأصناف: %s ≠ %s", got, real.MerchantID)
		}
	})
}

// ── ٤ · معرّفٌ غريبٌ لا يوجّه العمولةَ ولا التفعيل ─────────────────────
//
// **العمولةُ وعكسُها وعدُّ التفعيل تُقرأ من `orders.merchant_id`** (متجرُ
// الطلب المخزَّن). ولمّا صار هذا مضموناً أن يكون متجرَ الأصناف، **لا يستطيع
// متجرٌ غريبٌ أن يصير مستحقَّ العمولة أبداً**: الطلبُ لا يُنشأ أصلاً بمعرّفٍ
// غريب، وإن أُنشئ مشروعاً فمتجرُه متجرُ أصنافه.
func TestCDEF003_ForeignMerchantCannotBecomeCommissionStore(t *testing.T) {
	h := New(t)
	u := h.Customer()
	real := h.NewItem(1000)
	foreignRepStore := h.NewItem(1000) // متجرٌ آخرُ قد يكون له مندوبُه

	// محاولةُ توجيه العمولة إلى المتجر الغريب ترفضها البوّابة.
	res := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), bodyWithMerchant(real, 1, foreignRepStore.MerchantID))
	if res.Code < 400 || res.Err() != "bad_merchant" {
		t.Fatalf("**قُبل توجيهُ الطلب إلى متجرٍ غريب** — %s", res)
	}
	// وطلبٌ مشروعٌ متجرُه متجرُ أصنافه — فأيُّ عمولةٍ لاحقةٍ تُقرأ منه هو.
	ok := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), orderBody(real, 1))
	if ok.Code >= 400 {
		t.Fatalf("طلبٌ مشروعٌ رُفض: %s", ok)
	}
	if got := storedMerchant(t, h, u.ID); got != real.MerchantID {
		t.Errorf("**أساسُ العمولة ليس متجرَ الأصناف**: %s ≠ %s", got, real.MerchantID)
	}
	if got := storedMerchant(t, h, u.ID); got == foreignRepStore.MerchantID {
		t.Error("**متجرُ الطلب صار المتجرَ الغريب**")
	}
}
