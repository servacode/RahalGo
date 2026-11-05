package qa

// ══════════════════════════════════════════════════════════════════════
// **نسبةُ المتجر تتثبّت لحظةَ الاستلام** — خلل PD-1 (٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// **كان التسليمُ يسقط** إن بدّل المندوبُ أو الأدمنُ نسبةَ عمولة المتجر بعد أن استلم السائقُ
// منه: نداءُ التسوية الاحتياطيُّ عند التسليم يُعيد حسابَ حقّ المتجر بالنسبة الحيّة، ويجد الصفَّ
// المقيَّد عند الاستلام بمبلغٍ آخر — `XS-1` ⇒ ٥٠٠، **والسائقُ لا يستطيع «سلّمت»** حتّى تُعاد
// النسبة.
//
// **وصار**: النسبةُ تُكتب في صفّ التسوية (`merchant_settlements.commission_percent`) وتُقرأ منه.
// الطلبُ المستلَمُ يبقى على نسبة استلامه، **والجديدةُ تسري على كلّ طلبٍ لم يُستلم بعد.**

import (
	"net/http"
	"testing"
)

// pd1Steps **يقود الطلبَ بانتقالات السائق الحقيقيّة** — ويسقط عند أوّل ردّ.
func pd1Steps(t *testing.T, h *Harness, oid string, drv *User, steps ...string) {
	t.Helper()
	for _, to := range steps {
		if to == "delivered" {
			h.authorizeProofExempt(oid)
		}
		if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to}); got.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s رُدّ: %s", to, got)
		}
	}
}

// pd1Row **صفُّ تسوية المتجر في الطلب** — مبلغُه وطريقتُه ونسبتُه نصّاً (فارغةٌ إن لم تُكتب).
func pd1Row(t *testing.T, h *Harness, oid, merchantID string) (amount int64, method, pct string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT amount + adjusted_amount, method, COALESCE(commission_percent::text, '')
		  FROM merchant_settlements WHERE order_id = $1::uuid AND merchant_id = $2::uuid`,
		oid, merchantID).Scan(&amount, &method, &pct); err != nil {
		t.Fatalf("صفُّ التسوية: %v", err)
	}
	return amount, method, pct
}

// TestCommissionChangeAfterPickupStillDelivers **T0a — استلامٌ بعشرة، ثمّ تبديلُ النسبة إلى
// خمسة، ثمّ التسليم ⇒ يُسلَّم، وحقُّ المتجر وعمولةُ المنصّة بعشرة.** بالطريقين: تبديلُ المندوب
// لمتجرٍ «محفظة» وتبديلُ الأدمن لمتجرٍ «نقداً». **وطلبٌ بعد التبديل يُسوّى بالخمسة.**
func TestCommissionChangeAfterPickupStillDelivers(t *testing.T) {
	for _, tc := range []struct {
		name, by, method string
		earningKind      string
	}{
		{"rep_patch_wallet_store", "rep", "wallet", "merchant_earning"},
		{"admin_patch_cash_store", "admin", "cash", "merchant_cash_accrued"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := New(t)
			treasury(t, h)
			h.Setting("merchants.commission_percent", "10")
			f := h.Factory()
			rep := h.NewUser("sales")
			admin := h.NewUser("admin")
			m := f.Merchant(OwnedByRep(rep.ID), Commission(10))
			if tc.method == "cash" {
				coSetCash(t, h, m.ID)
			}
			item := h.NewItemFor(m, 10_000)
			base := financialBaseline(t, h)

			cust := h.Customer()
			made := h.POSTKey("/api/v1/orders", cust.Token, uniq("pd1"), orderBody(item, 1))
			if made.Code >= 400 {
				t.Fatalf("إنشاءُ الطلب: %s", made)
			}
			oid, _ := made.JSON()["id"].(string)
			drv := h.driverOf(oid)
			pd1Steps(t, h, oid, drv, "at_pickup", "picked_up")

			// **عند الاستلام**: ١٠٬٠٠٠ − ١٠٪ = ٩٬٠٠٠ — **والنسبةُ مكتوبةٌ في الصفّ.**
			amount, method, pct := pd1Row(t, h, oid, m.ID)
			if amount != 9_000 || method != tc.method {
				t.Fatalf("صفُّ الاستلام: %d %s — يُنتظر 9000 %s", amount, method, tc.method)
			}
			if pct != "10.00" {
				t.Errorf("نسبةُ صفّ الاستلام %q — يُنتظر 10.00", pct)
			}

			// **ثمّ تُبدَّل النسبةُ والبضاعةُ في يد السائق.**
			var r Res
			if tc.by == "rep" {
				r = h.PATCH("/api/v1/rep/stores/"+m.ID+"/commission", rep.Token,
					map[string]any{"percent": 5})
			} else {
				r = h.PATCH("/api/v1/admin/merchants/"+m.ID, admin.Token,
					map[string]any{"commission_percent": 5})
			}
			if r.Code != http.StatusOK {
				t.Fatalf("تبديلُ النسبة (%s): %s", tc.by, r)
			}
			if got := merchantCommissionText(t, h, m.ID); got != "5.00" {
				t.Fatalf("نسبةُ المتجر بعد التبديل %q — يُنتظر 5.00", got)
			}

			// **والتسليمُ يمرّ** — كان يُردّ ٥٠٠ (XS-1: 9000 ≠ 9500).
			pd1Steps(t, h, oid, drv, "on_the_way", "at_dropoff", "delivered")
			if st := h.statusOf(oid); st != "delivered" {
				t.Fatalf("الحالُ %q — يُنتظر delivered", st)
			}
			amount, _, pct = pd1Row(t, h, oid, m.ID)
			if amount != 9_000 || pct != "10.00" {
				t.Errorf("صفُّ التسوية بعد التسليم: %d بنسبة %q — يُنتظر 9000 بنسبة 10.00", amount, pct)
			}
			if _, _, _, _, commission := coOrderMoney(t, h, oid); commission != 1_000 {
				t.Errorf("عمولةُ المنصّة %d — يُنتظر 1000 (نسبةُ الاستلام)", commission)
			}
			if got := ledgerByRef(t, h, oid, tc.earningKind); got != 9_000 {
				t.Errorf("قيدُ حقّ المتجر (%s) %d — يُنتظر 9000", tc.earningKind, got)
			}

			// **والنسبةُ الجديدة تسري على طلبٍ لم يُستلم بعد.**
			made2 := h.POSTKey("/api/v1/orders", cust.Token, uniq("pd1b"), orderBody(item, 1))
			if made2.Code >= 400 {
				t.Fatalf("إنشاءُ الطلب الثاني: %s", made2)
			}
			oid2, _ := made2.JSON()["id"].(string)
			deliverOrder(t, h, oid2, h.driverOf(oid2))
			if amount, _, pct := pd1Row(t, h, oid2, m.ID); amount != 9_500 || pct != "5.00" {
				t.Errorf("الطلبُ بعد التبديل: %d بنسبة %q — يُنتظر 9500 بنسبة 5.00", amount, pct)
			}
			if _, _, _, _, commission := coOrderMoney(t, h, oid2); commission != 500 {
				t.Errorf("عمولةُ الطلب بعد التبديل %d — يُنتظر 500", commission)
			}

			assertNewViolations(t, h, base)
		})
	}
}
