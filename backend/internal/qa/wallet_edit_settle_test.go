package qa

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **تعديلُ الطلب أو أجرتِه يسوّي محفظةَ الزبون** (سؤالُ المالك ٢٠٢٦-١١-٢٢: «اذا تعدل سعر او
// تعدل الطلب لازم تزبط العملية ويرجع الباقي للزبون او ينخصم منه»)
// ══════════════════════════════════════════════════════════════════════
//
// لكلّ نطاق (الكل · التوصيل · كاش): نقصُ البضاعة يُردّ، زيادتُها تُخصم، وزيادةُ الأجرة تُخصم —
// **ودائماً `wallet_paid + cash_due = total` وصافي المحفظة = `wallet_paid`.**

func wesCheck(t *testing.T, h *Harness, oid, step string, wantWallet func(o wsOrder) int64) {
	t.Helper()
	o := wsRead(t, h, oid)
	if o.walletPaid+o.cashDue != o.total {
		t.Errorf("%s: المحفظة %d + الكاش %d ≠ المجموع %d", step, o.walletPaid, o.cashDue, o.total)
	}
	if o.paid != o.walletPaid {
		t.Errorf("%s: صافي ما خرج من المحفظة %d ≠ wallet_paid %d", step, o.paid, o.walletPaid)
	}
	if w := wantWallet(o); o.walletPaid != w {
		t.Errorf("%s: wallet_paid %d — يُنتظر %d (%+v)", step, o.walletPaid, w, o)
	}
}

func TestWalletEditSettles(t *testing.T) {
	for _, sc := range []struct {
		scope, method string
		want          func(o wsOrder) int64
	}{
		{"full", "", func(o wsOrder) int64 { return o.total }},
		{"delivery", "", func(o wsOrder) int64 { return o.fee }},
		{"none", "cash", func(o wsOrder) int64 { return 0 }},
	} {
		t.Run(sc.scope, func(t *testing.T) {
			h := New(t)
			item, _ := m2Prep(t, h)
			admin := h.NewUser("admin")
			cust := h.Customer()
			oid := wsMake(t, h, cust, wsBody(item, 2, sc.method, sc.scope), 100_000)
			wesCheck(t, h, oid, "عند الإنشاء", sc.want)
			before := wsRead(t, h, oid).total

			var line string
			if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM order_items WHERE order_id = $1::uuid`, oid).Scan(&line); err != nil {
				t.Fatal(err)
			}
			edit := func(qty int) {
				if r := h.POST("/api/v1/admin/orders/"+oid+"/edit", admin.Token, map[string]any{
					"reason": "الزبون غيّر", "lines": []map[string]any{{"item_id": line, "qty": qty}},
				}); r.Code != 200 {
					t.Fatalf("التعديل إلى %d: %s", qty, r)
				}
			}
			edit(1)
			if o := wsRead(t, h, oid); o.total >= before {
				t.Fatalf("المجموع ما نزل بعد النقص: %d ← %d", before, o.total)
			}
			wesCheck(t, h, oid, "بعد نقص صنف", sc.want)
			edit(3)
			wesCheck(t, h, oid, "بعد زيادة صنف", sc.want)

			fee := wsRead(t, h, oid).fee
			if r := feeEditCall(h, admin.Token, oid, fee+5_000, false); r.Code != 200 {
				t.Fatalf("رفع الأجرة: %s", r)
			}
			wesCheck(t, h, oid, "بعد رفع الأجرة", sc.want)
			if r := feeEditCall(h, admin.Token, oid, fee, false); r.Code != 200 {
				t.Fatalf("تنزيل الأجرة: %s", r)
			}
			wesCheck(t, h, oid, "بعد تنزيل الأجرة", sc.want)
		})
	}
}
