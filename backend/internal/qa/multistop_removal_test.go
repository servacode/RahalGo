package qa

// ══════════════════════════════════════════════════════════════════════
// **متجرٌ رفض قسمَه أو شاله المكتب — والطلبُ يكمل بغيره** (الدفعة P3)
// ══════════════════════════════════════════════════════════════════════
//
// **بنودُه تخرج ويُعاد الحساب** · **أجرُ السائق الأساسيُّ وحدَه** (قرارُ المالك ١ب) ·
// **والمحفظةُ يُردّ لها الفرق** · **والذنبُ على المتجر الرافض وحدَه** (٦أ) · **وتحويلُ قسمٍ لا يمسّ
// غيرَه** · **والمكتبُ يرى صناديقَ المتاجر، والزبونُ أقسامَه بلا متجر، والمتجرُ عددَها.**

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// msMoney **مالُ الطلب كما في القاعدة.**
type msMoney struct {
	Subtotal, DeliveryFee, SourcesFee, DriverFee, Total, WalletPaid, CashDue int64
	SharePct, ShareFixed                                                     int64
	MerchantID                                                               string
}

func moneyOf(t *testing.T, h *Harness, oid string) msMoney {
	t.Helper()
	var m msMoney
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT subtotal, delivery_fee, sources_fee, driver_fee, total, wallet_paid, cash_due,
		       snap_platform_delivery_percent, snap_platform_delivery_fixed, merchant_id::text
		FROM orders WHERE id = $1::uuid`, oid).
		Scan(&m.Subtotal, &m.DeliveryFee, &m.SourcesFee, &m.DriverFee, &m.Total, &m.WalletPaid, &m.CashDue,
			&m.SharePct, &m.ShareFixed, &m.MerchantID); err != nil {
		t.Fatalf("مالُ الطلب: %v", err)
	}
	return m
}

// refundsOf **مجموعُ ما رُدّ للزبون على هذا الطلب.**
func refundsOf(t *testing.T, h *Harness, oid string) int64 {
	t.Helper()
	var n int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions WHERE ref = $1 AND kind = 'refund'`, oid).
		Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// linesOf **عددُ بنود متجرٍ في الطلب.**
func linesOf(t *testing.T, h *Harness, oid, mid string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM order_items WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, mid).
		Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// saleOf **ثمنُ بنود متجرٍ في الطلب للزبون** — سعرُ البيع الملقوطُ × الكمّيّة.
func saleOf(t *testing.T, h *Harness, oid, mid string) int64 {
	t.Helper()
	var n int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(unit_price * qty), 0) FROM order_items WHERE order_id = $1::uuid AND merchant_id = $2::uuid`,
		oid, mid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// msFar **المتجران متباعدان** — فرسمُ المحطّة الثانية أجرةٌ كاملة (درجاتُ البُعد).
func (s *msShop) far(t *testing.T) {
	t.Helper()
	s.h.Setting("orders.extra_source_fee", "0")
	s.h.Setting("orders.source_proximity_m", "500")
	s.h.Setting("orders.source_far_m", "2000")
	s.h.Setting("orders.source_mid_percent", "50")
	s.h.Setting("orders.source_far_percent", "100")
	s.h.Setting("delivery.fee", "400")
	s.h.Setting("delivery.by_distance", "false")
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE merchants
		SET location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography WHERE id = $1::uuid`,
		s.mB.ID, raqqaLng, raqqaLat+0.03); err != nil {
		t.Fatal(err)
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE merchants
		SET location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography WHERE id = $1::uuid`,
		s.mA.ID, raqqaLng, raqqaLat); err != nil {
		t.Fatal(err)
	}
}

// walletOrder **طلبٌ من المتجرين يُدفع من المحفظة** — ورصيدُ الزبون يكفيه.
func (s *msShop) walletOrder(t *testing.T) string {
	t.Helper()
	s.f.Credit(s.cust.ID, 100_000, "topup")
	body := twoStoreBody(s.a, s.b)
	body["payment_method"] = "wallet"
	r := s.h.POSTKey("/api/v1/orders", s.cust.Token, uniq("ms2w"), body)
	if r.Code >= 400 {
		t.Fatalf("إنشاءُ طلب محفظة: %s", r)
	}
	id, _ := r.JSON()["id"].(string)
	return id
}

// partPost **فعلُ المكتب على قسم.**
func (s *msShop) partPost(tok, oid, mid, action string, body map[string]any) Res {
	return s.h.POST("/api/v1/admin/orders/"+oid+"/parts/"+mid+"/"+action, tok, body)
}

// TestMultiStop_PartRejectRemovesItemsAndExtraFee **متجرٌ رفض قسمَه ⇒ بنودُه تخرج ورسمُ المحطّة يسقط
// وأجرُ السائق الأساسيُّ وحدَه والمحفظةُ يُردّ لها الفرق** — `T9` (قرارُ المالك ١ب).
//
// **كان رفضُ القسم يُبقي بنودَه في الطلب** — فيبقى الطلبُ «بانتظار» عند المكتب، أو يدفع الزبونُ
// ثمنَ ما لن يصله.
func TestMultiStop_PartRejectRemovesItemsAndExtraFee(t *testing.T) {
	s := newMSShop(t, "merchants")
	s.far(t)
	oid := s.walletOrder(t)
	before := moneyOf(t, s.h, oid)
	sA, sB := saleOf(t, s.h, oid, s.mA.ID), saleOf(t, s.h, oid, s.mB.ID)
	if before.SourcesFee != 400 || before.DeliveryFee != 800 || before.Total != sA+sB+800 || before.WalletPaid != before.Total {
		t.Fatalf("قبل الرفض: %+v — يُنتظر رسمُ محطّة 400 وأجرة 800 والمجموعُ كلُّه من المحفظة", before)
	}

	s.mustAnswer(t, "رفضُ الثاني", s.tokB, oid, map[string]any{"to": "rejected", "note": "نفد الصنف"})

	after := moneyOf(t, s.h, oid)
	if n := linesOf(t, s.h, oid, s.mB.ID); n != 0 {
		t.Errorf("**بنودُ الرافض ما زالت في الطلب**: %d", n)
	}
	if after.Subtotal != sA || after.SourcesFee != 0 || after.DeliveryFee != 400 || after.Total != sA+400 {
		t.Errorf("بعد الرفض: %+v — يُنتظر بضاعة %d ورسمُ محطّة 0 وأجرة 400 ومجموع %d", after, sA, sA+400)
	}
	if want := orders.DriverFeeWithShare(400, after.SharePct, after.ShareFixed); after.DriverFee != want {
		t.Errorf("**أجرُ السائق %d — يُنتظر الأساسيّ %d** (رسمُ المحطّة يسقط عن الكلّ، ١ب)", after.DriverFee, want)
	}
	if after.WalletPaid != sA+400 || after.CashDue != 0 {
		t.Errorf("الدفع: محفظة %d نقد %d — يُنتظر %d و0", after.WalletPaid, after.CashDue, sA+400)
	}
	if got := refundsOf(t, s.h, oid); got != sB+400 {
		t.Errorf("**ما رُدّ للمحفظة %d — يُنتظر %d** (بضاعةُ الثاني + رسمُ المحطّة)", got, sB+400)
	}
	var snap string
	var removedTotal int64
	var fault string
	if err := s.h.Pool.QueryRow(ctxBG(), `
		SELECT status, removed_items::text, removed_total, removed_fault FROM order_parts
		WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, s.mB.ID).
		Scan(new(string), &snap, &removedTotal, &fault); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(snap, s.b.Name) || removedTotal != sB || fault != "merchant" {
		t.Errorf("لقطةُ القسم %s بمجموع %d وذنب %q — يُنتظر صنفُ الثاني و%d وذنبُ المتجر", snap, removedTotal, fault, sB)
	}
	if got := s.h.statusOf(oid); got != "pending" {
		t.Errorf("الطلبُ بعد رفض الثاني: %q — يُنتظر pending (الأوّلُ لم يجب)", got)
	}
	// **والزبونُ يُخبَر بأصنافه ومجموعه الجديد — بلا اسم متجر.**
	if !notifiedWith(t, s.h, s.cust.ID, oid, "قسم من طلبك ما توفّر", 2*time.Second) {
		t.Fatalf("**لم يُخبَر الزبونُ بالقسم الذي ما توفّر**")
	}
	var body string
	if err := s.h.Pool.QueryRow(ctxBG(), `SELECT body FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2 AND title = 'قسم من طلبك ما توفّر'`, s.cust.ID, oid).
		Scan(&body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, s.b.Name) || !strings.Contains(body, "صار طلبك") || strings.Contains(body, s.mB.Name) {
		t.Errorf("نصُّ خبر الزبون %q — يُنتظر الصنفُ والمجموعُ الجديد وبلا اسم المتجر", body)
	}

	// **والأوّلُ يقبل فيمضي الطلبُ بقسمه** — ثمّ يُسلَّم والدفترُ أخضر.
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	if got := s.h.statusOf(oid); got != "dispatching" {
		t.Fatalf("بعد قبول الأوّل: %q — يُنتظر dispatching", got)
	}
	deliverOrder(t, s.h, oid, s.h.driverOf(oid))
	assertFinancialTruth(t, s.h, oid)
}

// TestMultiStop_AllPartsRejectedRejectsOrder **رُفضت الأقسامُ كلُّها ⇒ الطلبُ مرفوض والمحفظةُ كاملة** — `T10`.
func TestMultiStop_AllPartsRejectedRejectsOrder(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.walletOrder(t)
	start := walletBalance(t, s.h, s.cust.ID) + moneyOf(t, s.h, oid).WalletPaid

	s.mustAnswer(t, "رفضُ الثاني", s.tokB, oid, map[string]any{"to": "rejected", "note": "نفد الصنف"})
	s.mustAnswer(t, "رفضُ الأوّل", s.tokA, oid, map[string]any{"to": "rejected", "note": "مغلق"})

	var status, reason string
	if err := s.h.Pool.QueryRow(ctxBG(), `SELECT status, COALESCE(cancel_reason, '') FROM orders WHERE id = $1::uuid`,
		oid).Scan(&status, &reason); err != nil {
		t.Fatal(err)
	}
	if status != "rejected" || reason != "ما توفّر شي من الطلب" {
		t.Fatalf("الطلبُ %q بسبب %q — يُنتظر rejected «ما توفّر شي من الطلب»", status, reason)
	}
	// **وآخرُ قسمٍ لا تُحذف بنودُه** — والطلبُ يُستردّ كلُّه في بابه، **ولقطتُه في سجلّ متجره.**
	if n := linesOf(t, s.h, oid, s.mA.ID); n != 1 {
		t.Errorf("بنودُ آخر قسمٍ %d — يُنتظر 1 (لا تُحذف)", n)
	}
	for _, m := range []*Merchant{s.mA, s.mB} {
		if p := partOf(t, s.h, oid, m.ID); p.Status != "rejected" {
			t.Errorf("قسمُ %s: %+v — يُنتظر rejected", m.Name, p)
		}
	}
	if got := walletBalance(t, s.h, s.cust.ID); got != start {
		t.Errorf("**الرصيدُ %d — يُنتظر %d كاملاً**", got, start)
	}
	if r := s.h.GET("/api/v1/merchant/stores/"+s.mA.ID+"/orders?closed_only=true", s.tokA); listed(r, oid) == nil {
		t.Errorf("**الطلبُ المرفوض ليس في سجلّ المتجر الأوّل**")
	}
	assertFinancialTruth(t, s.h, oid)
}

// TestMultiStop_CustomerCancelCancelsAll **الزبونُ يُلغي ⇒ الطلبُ كلُّه ملغى والمحفظةُ كاملة** — `T11`.
func TestMultiStop_CustomerCancelCancelsAll(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.walletOrder(t)
	start := walletBalance(t, s.h, s.cust.ID) + moneyOf(t, s.h, oid).WalletPaid
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	if r := s.h.POST("/api/v1/orders/"+oid+"/cancel", s.cust.Token, map[string]any{}); r.Code >= 400 {
		t.Fatalf("إلغاءُ الزبون: %s", r)
	}
	if got := s.h.statusOf(oid); got != "cancelled" {
		t.Fatalf("الطلب %q — يُنتظر cancelled", got)
	}
	if got := walletBalance(t, s.h, s.cust.ID); got != start {
		t.Errorf("الرصيد %d — يُنتظر %d", got, start)
	}
	if n := linesOf(t, s.h, oid, s.mB.ID) + linesOf(t, s.h, oid, s.mA.ID); n != 2 {
		t.Errorf("**الإلغاءُ حذف بنوداً**: بقي %d — يُنتظر 2", n)
	}
	assertFinancialTruth(t, s.h, oid)
}

// TestMultiStop_RemovePartKeepsCashRemainder **شيلُ قسمٍ من طلب محفظةٍ عليه باقٍ نقديّ لا يضيّع الباقي** — `T26`.
//
// **وفيه شيلُ المكتب بعد التحويل** (قسمٌ ليس الأخير) **والسائقُ يُخبَر** ثمّ يُسلَّم الطلب.
func TestMultiStop_RemovePartKeepsCashRemainder(t *testing.T) {
	s := newMSShop(t, "merchants")
	body := twoStoreBody(s.a, s.b)
	body["payment_method"] = "wallet"
	// **الرصيدُ يساوي المجموعَ تماماً** — فزيادةُ التعديل تبقى نقداً.
	r := s.h.POST("/api/v1/public/quote", s.cust.Token, map[string]any{
		"items": body["items"], "lat": raqqaLat, "lng": raqqaLng,
	})
	total := int64(num(r.JSON()["total"]))
	if total <= 0 {
		t.Fatalf("التسعيرة: %s", r)
	}
	s.f.Credit(s.cust.ID, total, "topup")
	cr := s.h.POSTKey("/api/v1/orders", s.cust.Token, uniq("ms2c"), body)
	if cr.Code >= 400 {
		t.Fatalf("الإنشاء: %s", cr)
	}
	oid, _ := cr.JSON()["id"].(string)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})

	var lineA string
	if err := s.h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM order_items
		WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, s.mA.ID).Scan(&lineA); err != nil {
		t.Fatal(err)
	}
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/edit", s.admin.Token, map[string]any{
		"reason": "الزبون زاد", "lines": []map[string]any{{"item_id": lineA, "qty": 2}},
	}); r.Code >= 400 {
		t.Fatalf("تعديلُ الطلب: %s", r)
	}
	edited := moneyOf(t, s.h, oid)
	sA, sB := saleOf(t, s.h, oid, s.mA.ID)/2, saleOf(t, s.h, oid, s.mB.ID)
	if edited.CashDue != sA {
		t.Fatalf("بعد التعديل: %+v — يُنتظر باقٍ نقديّ %d", edited, sA)
	}
	drv := s.f.Driver(OnShift())
	t.Cleanup(func() {
		_, _ = s.h.Pool.Exec(ctxBG(), `UPDATE users SET on_shift = false WHERE id = $1::uuid`, drv.ID)
	})
	if r := s.h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); r.Code >= 400 {
		t.Fatalf("قبولُ السائق: %s", r)
	}

	if r := s.partPost(s.admin.Token, oid, s.mB.ID, "remove",
		map[string]any{"reason": "المتجر مسكّر", "fault": "platform"}); r.Code != http.StatusOK {
		t.Fatalf("شيلُ القسم الثاني: %s", r)
	}
	after := moneyOf(t, s.h, oid)
	if after.WalletPaid != edited.WalletPaid || after.CashDue != sA-sB || after.Total != edited.Total-sB {
		t.Errorf("**بعد الشيل: محفظة %d نقد %d مجموع %d — يُنتظر %d و%d و%d** (الباقي النقديّ لا يضيع)",
			after.WalletPaid, after.CashDue, after.Total, edited.WalletPaid, sA-sB, edited.Total-sB)
	}
	if got := refundsOf(t, s.h, oid); got != 0 {
		t.Errorf("استرجاعٌ %d — والمحفظةُ لم تدفع أكثرَ من المجموع الجديد", got)
	}
	if p := partOf(t, s.h, oid, s.mB.ID); p.Status != "removed" {
		t.Errorf("القسمُ الثاني %+v — يُنتظر removed (بعد قبول الطلب)", p)
	}
	if !notifiedWith(t, s.h, drv.ID, oid, "انشالت محطة من الطلب", 2*time.Second) {
		t.Errorf("**السائقُ لم يُخبَر بالمحطّة التي انشالت**")
	}
	if !notifiedWith(t, s.h, s.mB.Owner.ID, oid, "أُلغي قسمك من الطلب", 2*time.Second) {
		t.Errorf("**المتجرُ الثاني لم يُخبَر بأنّ المكتب شال قسمه**")
	}
	deliverOrder(t, s.h, oid, drv)
	assertFinancialTruth(t, s.h, oid)
}

// TestMultiStop_RemovalRoleWindows **من يشيل ومتى** — `T35` (D10 · C11).
//
//	المتجرُ يشيل قسمَه حتّى «قيد التحضير»، وبعد النزول للسائقين للمكتب (`use_part`)
//	والعمليّاتُ لا تشيل آخرَ قسمٍ بعد التحويل (إلغاءٌ للطلب) — والأدمنُ يشيله فيُلغى الطلب
//	وقسمٌ استُلم لا يُشال (`part_collected`)
func TestMultiStop_RemovalRoleWindows(t *testing.T) {
	t.Run("المتجر", func(t *testing.T) {
		s := newMSShop(t, "merchants")
		s.h.Setting("orders.auto_dispatch", "false")
		oid := s.order(t)
		s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
		s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
		if got := s.h.statusOf(oid); got != "preparing" {
			t.Fatalf("بعد القبولين: %q — يُنتظر preparing", got)
		}
		// **قيد التحضير: المتجرُ يشيل قسمَه بنفسه** — والطلبُ يكمل بالأوّل.
		s.mustAnswer(t, "شيلُ الثاني قسمَه", s.tokB, oid, map[string]any{"to": "cancelled", "note": "انقطعت الكهرباء"})
		if p := partOf(t, s.h, oid, s.mB.ID); p.Status != "removed" {
			t.Errorf("القسمُ الثاني %+v — يُنتظر removed", p)
		}
		if got := s.h.statusOf(oid); got != "preparing" {
			t.Errorf("الطلبُ بعد شيل الثاني: %q — يُنتظر preparing (يكمل بالأوّل)", got)
		}
		// **وبعد النزول للسائقين المتجرُ لا يشيل** — للمكتب.
		if r := s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
			map[string]any{"to": "dispatching"}); r.Code >= 400 {
			t.Fatalf("النزول: %s", r)
		}
		if r := s.answer(s.tokA, oid, map[string]any{"to": "cancelled", "note": "تأخّرنا"}); r.Code != http.StatusConflict || r.Err() != "use_part" {
			t.Errorf("شيلُ المتجر بعد النزول: %s — يُنتظر 409 use_part", r)
		}
	})
	t.Run("المكتب", func(t *testing.T) {
		s := newMSShop(t, "merchants")
		oid := s.order(t)
		s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
		s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
		if got := s.h.statusOf(oid); got != "dispatching" {
			t.Fatalf("بعد القبولين: %q — يُنتظر dispatching", got)
		}
		ops := s.h.NewUser("operations")
		if r := s.partPost(ops.Token, oid, s.mB.ID, "remove", map[string]any{"reason": "مسكّر", "fault": "platform"}); r.Code != http.StatusOK {
			t.Fatalf("العمليّاتُ تشيل قسماً ليس الأخير: %s", r)
		}
		// **آخرُ قسمٍ بعد التحويل إلغاءٌ للطلب — للأدمن وحدَه.**
		if r := s.partPost(ops.Token, oid, s.mA.ID, "remove", map[string]any{"reason": "مسكّر", "fault": "platform"}); r.Code < 400 {
			t.Errorf("**العمليّاتُ شالت آخرَ قسمٍ بعد التحويل**: %s", r)
		}
		if p := partOf(t, s.h, oid, s.mA.ID); p.Status != "accepted" {
			t.Errorf("**رُفض الشيلُ وكُتب القسم**: %+v", p)
		}
		if got := s.h.statusOf(oid); got != "dispatching" {
			t.Errorf("الطلبُ بعد شيلٍ مرفوض: %q", got)
		}
		// **وفعلٌ بلا ذنبٍ معروف يُردّ.**
		if r := s.partPost(s.admin.Token, oid, s.mA.ID, "remove", map[string]any{"reason": "مسكّر", "fault": "x"}); r.Code != http.StatusBadRequest {
			t.Errorf("ذنبٌ مجهول: %s — يُنتظر 400", r)
		}
		if r := s.partPost(s.admin.Token, oid, s.mA.ID, "remove", map[string]any{"reason": "مسكّر", "fault": "platform"}); r.Code != http.StatusOK {
			t.Fatalf("الأدمنُ يشيل آخرَ قسم: %s", r)
		}
		if got := s.h.statusOf(oid); got != "cancelled" {
			t.Errorf("بعد شيل آخر قسم: %q — يُنتظر cancelled", got)
		}
		assertFinancialTruth(t, s.h, oid)
	})
	t.Run("المستلَم", func(t *testing.T) {
		s := newMSShop(t, "merchants")
		oid := s.order(t)
		s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
		s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
		// **محطّةٌ استُلمت** — حالُها كما يكتبها الاستلامُ لكلّ محطّة (الدفعة التالية).
		if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE order_parts SET status = 'picked_up', picked_up_at = now()
			WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, s.mA.ID); err != nil {
			t.Fatal(err)
		}
		if r := s.partPost(s.admin.Token, oid, s.mA.ID, "remove", map[string]any{"reason": "x", "fault": "platform"}); r.Code != http.StatusConflict || r.Err() != "part_collected" {
			t.Errorf("شيلُ قسمٍ مستلَم: %s — يُنتظر 409 part_collected", r)
		}
		var lineA string
		if err := s.h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM order_items
			WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, s.mA.ID).Scan(&lineA); err != nil {
			t.Fatal(err)
		}
		if r := s.h.POST("/api/v1/admin/orders/"+oid+"/items/"+lineA+"/unavailable", s.admin.Token,
			map[string]any{}); r.Code != http.StatusConflict || r.Err() != "part_collected" {
			t.Errorf("صنفٌ غيرُ متوفّر لمتجرٍ مستلَم: %s — يُنتظر 409 part_collected", r)
		}
	})
}

// TestMultiStop_ItemUnavailableLastLineRemovesPart **آخرُ صنفٍ لمتجرٍ غيرُ متوفّر ⇒ قسمُه يُشال** (§6.2).
func TestMultiStop_ItemUnavailableLastLineRemovesPart(t *testing.T) {
	s := newMSShop(t, "platform")
	oid := s.order(t)
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
		map[string]any{"to": "accepted"}); r.Code >= 400 {
		t.Fatalf("قبولُ المكتب: %s", r)
	}
	var lineB string
	if err := s.h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM order_items
		WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, s.mB.ID).Scan(&lineB); err != nil {
		t.Fatal(err)
	}
	r := s.h.POST("/api/v1/admin/orders/"+oid+"/items/"+lineB+"/unavailable", s.admin.Token, map[string]any{})
	if r.Code != http.StatusOK {
		t.Fatalf("صنفٌ غيرُ متوفّر: %s", r)
	}
	if res, _ := r.JSON()["result"].(map[string]any); res["source_gone"] != true || num(res["new_total"]) == 0 {
		t.Errorf("النتيجة %v — يُنتظر خروجُ المتجر ومجموعٌ جديد", res)
	}
	var status, fault string
	if err := s.h.Pool.QueryRow(ctxBG(), `SELECT status, removed_fault FROM order_parts
		WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, s.mB.ID).Scan(&status, &fault); err != nil {
		t.Fatal(err)
	}
	if status != "removed" || fault != "platform" {
		t.Errorf("قسمُ الثاني %q بذنب %q — يُنتظر removed على المنصّة", status, fault)
	}
	// **وإبلاغُ الأوّل وحدَه يُنزل الطلب** — لا قسمَ آخرَ ينتظر تذكرته.
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/whatsapp", s.admin.Token,
		map[string]any{"channel": "manual", "merchant_id": s.mA.ID}); r.Code >= 400 {
		t.Fatalf("الإرسال: %s", r)
	}
	if got := s.h.statusOf(oid); got != "dispatching" {
		t.Errorf("بعد إبلاغ الأوّل: %q — يُنتظر dispatching", got)
	}
	assertFinancialTruth(t, s.h, oid)
}

// TestMultiStop_PartTransferMovesOnlyThatPart **تحويلُ قسمٍ يمسّه وحدَه** — `T20` (G-15 · M7 · C16 · D13).
func TestMultiStop_PartTransferMovesOnlyThatPart(t *testing.T) {
	s := newMSShop(t, "platform")
	oid := s.order(t)
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
		map[string]any{"to": "accepted"}); r.Code >= 400 {
		t.Fatalf("قبولُ المكتب: %s", r)
	}
	for _, m := range []*Merchant{s.mA, s.mB} {
		if r := s.h.POST("/api/v1/admin/orders/"+oid+"/whatsapp", s.admin.Token,
			map[string]any{"channel": "manual", "merchant_id": m.ID}); r.Code >= 400 {
			t.Fatalf("الإرسال: %s", r)
		}
	}
	if got := s.h.statusOf(oid); got != "dispatching" {
		t.Fatalf("بعد الإرسالين: %q", got)
	}
	// **متجرٌ ثالثٌ نقديٌّ عنده صنفُ الثاني باسمه.**
	mC := s.f.Merchant(Commission(5))
	c := s.h.NewItemPriced(mC, 450, 700)
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE menu_items SET name = $2 WHERE id = $1::uuid`, c.ID, s.b.Name); err != nil {
		t.Fatal(err)
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE merchants SET settlement_method = 'cash' WHERE id = $1::uuid`, mC.ID); err != nil {
		t.Fatal(err)
	}

	// **المرشّحون لقسم الثاني**: بنودُه وحدَها، ولا متجرَ في الطلب أصلاً.
	cand := s.h.GET("/api/v1/admin/orders/"+oid+"/transfer-candidates?part_merchant_id="+s.mB.ID, s.admin.Token)
	if cand.Code != http.StatusOK {
		t.Fatalf("المرشّحون: %s", cand)
	}
	if items, _ := cand.JSON()["items"].([]any); len(items) != 1 {
		t.Errorf("بنودُ المرشّحين %d — يُنتظر بندُ الثاني وحدَه", len(items))
	}
	raw := string(cand.Body)
	if strings.Contains(raw, `"merchant_id":"`+s.mA.ID+`"`) {
		t.Errorf("**المتجرُ الأوّلُ مرشَّحٌ وهو في الطلب**")
	}
	if !strings.Contains(raw, mC.ID) {
		t.Errorf("المتجرُ الثالثُ ليس في المرشّحين")
	}

	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/transfer", s.admin.Token,
		map[string]any{"merchant_id": mC.ID, "note": "x"}); r.Code != http.StatusConflict || r.Err() != "use_part_transfer" {
		t.Errorf("تحويلُ الطلب كلِّه: %s — يُنتظر 409 use_part_transfer", r)
	}
	if r := s.partPost(s.admin.Token, oid, s.mB.ID, "transfer", map[string]any{"to_merchant_id": s.mA.ID}); r.Code != http.StatusConflict || r.Err() != "transfer_target_in_order" {
		t.Errorf("تحويلٌ لمتجرٍ في الطلب: %s — يُنتظر 409 transfer_target_in_order", r)
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE orders SET offer_expires_at = now() WHERE id = $1::uuid`, oid); err != nil {
		t.Fatal(err)
	}
	r := s.partPost(s.admin.Token, oid, s.mB.ID, "transfer", map[string]any{"to_merchant_id": mC.ID})
	if r.Code != http.StatusOK {
		t.Fatalf("تحويلُ القسم: %s", r)
	}
	if got := s.h.statusOf(oid); got != "dispatching" {
		t.Errorf("**التحويلُ أرجع حالَ الطلب**: %q", got)
	}
	if p := partOf(t, s.h, oid, s.mB.ID); p.Status != "removed" || p.Reason != "حُوّل لمتجر آخر" {
		t.Errorf("القسمُ القديم %+v — يُنتظر removed «حُوّل لمتجر آخر»", p)
	}
	if p := partOf(t, s.h, oid, mC.ID); p.Status != "accepted" || p.Sent {
		t.Errorf("القسمُ الجديد %+v — يُنتظر accepted لم يُبلَّغ", p)
	}
	var seq int
	var method, menuID string
	var price int64
	if err := s.h.Pool.QueryRow(ctxBG(), `
		SELECT p.seq, oi.merchant_settlement_method, oi.menu_item_id::text, oi.merchant_price
		FROM order_parts p JOIN order_items oi ON oi.order_id = p.order_id AND oi.merchant_id = p.merchant_id
		WHERE p.order_id = $1::uuid AND p.merchant_id = $2::uuid`, oid, mC.ID).Scan(&seq, &method, &menuID, &price); err != nil {
		t.Fatalf("بندُ المتجر الجديد: %v", err)
	}
	if seq != 3 || method != "cash" || menuID != c.ID || price != 450 {
		t.Errorf("القسمُ الجديد seq %d طريقة %q صنف %s سعر %d — يُنتظر 3 ونقد وصنفُه و450", seq, method, menuID, price)
	}
	if n := linesOf(t, s.h, oid, s.mA.ID); n != 1 {
		t.Errorf("**بنودُ الأوّل تبدّلت**: %d", n)
	}
	var sent, offer bool
	if err := s.h.Pool.QueryRow(ctxBG(), `SELECT sent_to_merchant_at IS NOT NULL, offer_expires_at IS NOT NULL
		FROM orders WHERE id = $1::uuid`, oid).Scan(&sent, &offer); err != nil {
		t.Fatal(err)
	}
	if sent || offer {
		t.Errorf("بعد التحويل: «أُبلغ» %v ومهلةُ إسناد %v — يُنتظر الاثنان فارغين", sent, offer)
	}
	// **والزبونُ لا يرى أنّ متجراً تبدّل** — قسمان مرقّمان ١ و٢ بلا «ما توفّر».
	cv := s.h.GET("/api/v1/my/orders/"+oid, s.cust.Token).JSON()
	secs, _ := cv["sections"].([]any)
	if len(secs) != 2 {
		t.Fatalf("أقسامُ الزبون %v — يُنتظر ٢", cv["sections"])
	}
	for i, x := range secs {
		sec, _ := x.(map[string]any)
		if int(num(sec["seq"])) != i+1 || sec["status"] == "unavailable" {
			t.Errorf("قسمُ الزبون %d: %v — يُنتظر رقمٌ متتالٍ بلا «ما توفّر»", i, sec)
		}
	}
	// **ومحطّةٌ مستلَمة لا تُحوَّل.**
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE order_parts SET status = 'picked_up'
		WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, s.mA.ID); err != nil {
		t.Fatal(err)
	}
	mD := s.f.Merchant()
	if r := s.partPost(s.admin.Token, oid, s.mA.ID, "transfer", map[string]any{"to_merchant_id": mD.ID}); r.Code != http.StatusConflict || r.Err() != "part_collected" {
		t.Errorf("تحويلُ قسمٍ مستلَم: %s — يُنتظر 409 part_collected", r)
	}
}

// TestMultiStop_RejectedPartHistory **قسمٌ رُفض في سجلّ متجره: ما رفضه، بمالٍ صفر** — `T39` (C6).
func TestMultiStop_RejectedPartHistory(t *testing.T) {
	s := newMSShop(t, "merchants")
	oid := s.order(t)
	s.mustAnswer(t, "رفضُ الثاني", s.tokB, oid, map[string]any{"to": "rejected", "note": "نفد الصنف"})
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})

	check := func(label string, v map[string]any) {
		t.Helper()
		if v == nil {
			t.Fatalf("%s: لا حمولة", label)
		}
		raw, _ := json.Marshal(v)
		items, _ := v["items"].([]any)
		if len(items) != 1 || !strings.Contains(string(raw), s.b.Name) || strings.Contains(string(raw), s.a.Name) {
			t.Errorf("%s: البنود %s — يُنتظر صنفُ الثاني وحدَه من لقطته", label, raw)
		}
		if v["status"] != "rejected" || num(v["subtotal"]) != 0 || num(v["merchant_net"]) != 0 || num(v["platform_commission"]) != 0 {
			t.Errorf("%s: الحال %v المجموع %v الصافي %v — يُنتظر rejected بمالٍ صفر (لا مجموعُ الطلب)",
				label, v["status"], v["subtotal"], v["merchant_net"])
		}
		if num(v["shared_stops"]) != 2 {
			t.Errorf("%s: shared_stops %v — يُنتظر 2", label, v["shared_stops"])
		}
	}
	check("سجلُّ الثاني", listed(s.h.GET("/api/v1/merchant/stores/"+s.mB.ID+"/orders?closed_only=true", s.tokB), oid))
	d := s.h.GET("/api/v1/merchant/orders/"+oid, s.tokB)
	if d.Code != http.StatusOK {
		t.Fatalf("تفصيلُ الثاني: %s", d)
	}
	check("تفصيلُ الثاني", d.JSON())
	// **والأوّلُ يرى عددَ المتاجر لا أسماءها.**
	a := s.h.GET("/api/v1/merchant/orders/"+oid, s.tokA).JSON()
	if num(a["shared_stops"]) != 2 || strings.Contains(string(s.h.GET("/api/v1/merchant/orders/"+oid, s.tokA).Body), s.mB.Name) {
		t.Errorf("حمولةُ الأوّل: shared_stops %v — يُنتظر 2 بلا اسم الثاني", a["shared_stops"])
	}
}

// TestMultiStop_StoreFaultWarningOnThatStore **رفضُ المتجر قسمَه إنذارٌ عليه وحدَه** — `T43` (٦أ).
//
// **وإنذارُ متجرٍ ثانٍ على الطلب نفسِه يبقى** (فهرسُ ٠٥١٧) — **وكان الفهرسُ واحداً لكلّ طلب.**
func TestMultiStop_StoreFaultWarningOnThatStore(t *testing.T) {
	s := newMSShop(t, "merchants")
	baseB, err := s.h.Orders.MerchantViolations(ctxBG(), s.h.Pool, s.mB.ID)
	if err != nil {
		t.Fatal(err)
	}
	oid := s.order(t)
	s.mustAnswer(t, "رفضُ الثاني", s.tokB, oid, map[string]any{"to": "rejected", "note": "نفد الصنف"})
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})

	warned := func(mid string) int {
		t.Helper()
		var n int
		if err := s.h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM warnings
			WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, mid).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	deadline := time.Now().Add(2 * time.Second)
	for warned(s.mB.ID) == 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if warned(s.mB.ID) != 1 || warned(s.mA.ID) != 0 {
		t.Fatalf("الإنذارات: الثاني %d الأوّل %d — يُنتظر 1 و0", warned(s.mB.ID), warned(s.mA.ID))
	}
	if n, err := s.h.Orders.MerchantViolations(ctxBG(), s.h.Pool, s.mB.ID); err != nil || n != baseB+1 {
		t.Errorf("**مخالفاتُ الثاني %d — يُنتظر %d** (رفضُ قسمه يُعدّ كرفض طلب): %v", n, baseB+1, err)
	}
	// **وإنذارُ المتجر الأوّل على الطلب نفسِه يُكتب ولا يبتلعه إنذارُ الثاني** — والتكرارُ لا.
	insert := func(mid string) error {
		_, err := s.h.Pool.Exec(ctxBG(), `
			INSERT INTO warnings (user_id, merchant_id, role_code, reason, order_id)
			SELECT owner_user_id, id, 'merchant', 'merchant_refused', $2::uuid FROM merchants WHERE id = $1::uuid
			ON CONFLICT (order_id, merchant_id) WHERE order_id IS NOT NULL AND merchant_id IS NOT NULL DO NOTHING`,
			mid, oid)
		return err
	}
	if err := insert(s.mA.ID); err != nil {
		t.Fatalf("إنذارُ الأوّل: %v", err)
	}
	if err := insert(s.mB.ID); err != nil {
		t.Fatalf("إنذارٌ مكرّرٌ للثاني: %v", err)
	}
	if warned(s.mA.ID) != 1 || warned(s.mB.ID) != 1 {
		t.Errorf("بعد الإدخال: الأوّل %d الثاني %d — يُنتظر 1 و1", warned(s.mA.ID), warned(s.mB.ID))
	}
	// **وشيلُ المكتب بذنب المنصّة لا يُنذر.**
	oid2 := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid2, map[string]any{"to": "accepted"})
	if r := s.partPost(s.admin.Token, oid2, s.mB.ID, "reject", map[string]any{"reason": "طلب الزبون"}); r.Code != http.StatusOK {
		t.Fatalf("رفضُ المكتب للقسم: %s", r)
	}
	var n int
	_ = s.h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM warnings WHERE order_id = $1::uuid`, oid2).Scan(&n)
	if n != 0 {
		t.Errorf("**رفضُ المكتب أنذر متجراً**: %d", n)
	}
	if p := partOf(t, s.h, oid2, s.mB.ID); p.Status != "rejected" || p.Role != "admin" {
		t.Errorf("قسمُ الثاني بعد رفض المكتب %+v — يُنتظر rejected بيد الأدمن", p)
	}
}

// TestMultiStop_OpsPartsPayload **صندوقٌ لكلّ متجرٍ في حمولة المكتب** — العقدُ §2 (القائمةُ والتفصيلُ والبثّ).
func TestMultiStop_OpsPartsPayload(t *testing.T) {
	s := newMSShop(t, "platform")
	capOps := s.h.Listen("ops")
	oid := s.order(t)
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
		map[string]any{"to": "accepted"}); r.Code >= 400 {
		t.Fatalf("قبولُ المكتب: %s", r)
	}
	sA, sB := saleOf(t, s.h, oid, s.mA.ID), saleOf(t, s.h, oid, s.mB.ID)
	if r := s.partPost(s.admin.Token, oid, s.mB.ID, "remove", map[string]any{"reason": "مسكّر", "fault": "merchant"}); r.Code != http.StatusOK {
		t.Fatalf("الشيل: %s", r)
	}
	keys := []string{"merchant_id", "seq", "name", "phone", "address", "lat", "lng", "status", "answered_role",
		"answered_at", "reason", "removed_fault", "prep_minutes", "ready_at", "sent_at", "sent_channel",
		"arrived_at", "picked_up_at", "picked_up_by_name", "collected", "items", "subtotal", "cost",
		"settlement_method", "pay_amount", "rep_name", "rep_commission"}
	check := func(label string, v map[string]any) {
		t.Helper()
		parts, _ := v["parts"].([]any)
		if len(parts) != 2 {
			t.Fatalf("%s: parts %v — يُنتظر ٢", label, v["parts"])
		}
		for _, x := range parts {
			p, _ := x.(map[string]any)
			for _, k := range keys {
				if _, ok := p[k]; !ok {
					t.Errorf("%s: الحقلُ %q ناقصٌ في القسم", label, k)
				}
			}
		}
		pa, _ := parts[0].(map[string]any)
		pb, _ := parts[1].(map[string]any)
		if pa["merchant_id"] != s.mA.ID || pa["status"] != "accepted" || num(pa["subtotal"]) != sA || num(pa["cost"]) != 1000 {
			t.Errorf("%s: قسمُ الأوّل %v", label, pa)
		}
		if pa["name"] != s.mA.Name || pa["phone"] != s.mA.Owner.Phone || pa["settlement_method"] != "wallet" {
			t.Errorf("%s: هويّةُ الأوّل %v %v %v", label, pa["name"], pa["phone"], pa["settlement_method"])
		}
		items, _ := pb["items"].([]any)
		if pb["status"] != "removed" || pb["removed_fault"] != "merchant" || pb["reason"] != "مسكّر" ||
			len(items) != 1 || num(pb["subtotal"]) != sB || num(pb["cost"]) != 500 {
			t.Errorf("%s: قسمُ الثاني %v — يُنتظر removed بلقطة بنده وماله", label, pb)
		}
		if it, _ := items[0].(map[string]any); it["name"] != s.b.Name || num(it["merchant_price"]) != 500 {
			t.Errorf("%s: بندُ لقطة الثاني %v", label, it)
		}
	}
	d := s.h.GET("/api/v1/admin/orders/"+oid, s.admin.Token)
	if d.Code != http.StatusOK {
		t.Fatalf("تفصيلُ المكتب: %s", d)
	}
	check("التفصيل", d.JSON())
	l := s.h.GET("/api/v1/admin/orders?per_page=100&q="+oid, s.admin.Token)
	if v := listed(l, oid); v != nil {
		check("القائمة", v)
	} else if v := listed(s.h.GET("/api/v1/admin/orders?per_page=100", s.admin.Token), oid); v != nil {
		check("القائمة", v)
	} else {
		t.Errorf("الطلبُ ليس في قائمة المكتب")
	}
	got := false
	for _, ev := range capOps.Drain(1500 * time.Millisecond) {
		if o, ok := ev["order"].(map[string]any); ok && o["id"] == oid {
			if parts, _ := o["parts"].([]any); len(parts) == 2 {
				got = true
			}
		}
	}
	if !got {
		t.Errorf("**بثُّ المكتب بلا صناديق المتاجر**")
	}
	// **وطلبُ المتجر الواحد بلا parts** — حمولتُه كما كانت.
	single := s.h.POSTKey("/api/v1/orders", s.cust.Token, uniq("ms1"), orderBody(s.a, 1))
	sid, _ := single.JSON()["id"].(string)
	if v := s.h.GET("/api/v1/admin/orders/"+sid, s.admin.Token).JSON(); v["parts"] != nil {
		t.Errorf("**طلبُ متجرٍ واحدٍ يحمل parts**")
	}
	if v := s.h.GET("/api/v1/my/orders/"+sid, s.cust.Token).JSON(); v["sections"] != nil {
		t.Errorf("**طلبُ متجرٍ واحدٍ يحمل sections**")
	}
	if v := s.h.GET("/api/v1/merchant/orders/"+sid, s.tokA).JSON(); v["shared_stops"] != nil {
		t.Errorf("**طلبُ متجرٍ واحدٍ يحمل shared_stops**")
	}
}

// TestMultiStop_CustomerSectionsWithoutStores **أقسامُ الزبون بلا اسم متجرٍ ولا معرّفه** — العقدُ §3 (`P-1`).
func TestMultiStop_CustomerSectionsWithoutStores(t *testing.T) {
	s := newMSShop(t, "merchants")
	capC := s.h.Listen("customer:" + s.cust.ID)
	oid := s.order(t)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	s.mustAnswer(t, "رفضُ الثاني", s.tokB, oid, map[string]any{"to": "rejected", "note": "نفد الصنف"})

	r := s.h.GET("/api/v1/my/orders/"+oid, s.cust.Token)
	if r.Code != http.StatusOK {
		t.Fatalf("طلبُ الزبون: %s", r)
	}
	for _, leak := range []string{s.mA.Name, s.mB.Name, s.mA.ID, s.mB.ID} {
		if strings.Contains(string(r.Body), leak) {
			t.Errorf("**حمولةُ الزبون تحمل هويّةَ متجر** %q", leak)
		}
	}
	secs, _ := r.JSON()["sections"].([]any)
	if len(secs) != 2 {
		t.Fatalf("الأقسام %v — يُنتظر ٢", r.JSON()["sections"])
	}
	want := []struct {
		status, item string
	}{{"accepted", s.a.Name}, {"unavailable", s.b.Name}}
	for i, x := range secs {
		sec, _ := x.(map[string]any)
		for k := range sec {
			if k != "seq" && k != "status" && k != "items" {
				t.Errorf("قسمُ الزبون فيه حقلٌ غيرُ متّفق %q", k)
			}
		}
		items, _ := sec["items"].([]any)
		it, _ := items[0].(map[string]any)
		if int(num(sec["seq"])) != i+1 || sec["status"] != want[i].status || it["name"] != want[i].item || num(it["qty"]) != 1 {
			t.Errorf("القسم %d: %v — يُنتظر %s بـ%s", i+1, sec, want[i].status, want[i].item)
		}
	}
	for _, ev := range capC.Drain(1500 * time.Millisecond) {
		raw, _ := json.Marshal(ev)
		for _, leak := range []string{s.mA.Name, s.mB.Name, s.mA.ID, s.mB.ID} {
			if strings.Contains(string(raw), leak) {
				t.Errorf("**بثُّ الزبون يحمل هويّةَ متجر** %q", leak)
			}
		}
	}
}
