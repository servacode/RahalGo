package qa

import (
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **التحويلُ التلقائيّ يقبل ولا يُبلّغ** — `R21` · `EV-09`
// ══════════════════════════════════════════════════════════════════════
//
// # ما قِيس في الفحص القديم
//
// **`TestEV_R21AutoTransferAwareness` يضبط `orders.auto_transfer_amount`**
// — **ومفتاحُ التشغيل `orders.auto_transfer` منطقيٌّ بلا عتبة**:
// «**وذهبت العتبتان**» (`auto_transfer.go:42`).
//
// **فلم يقع تحويلٌ قطُّ في ذلك الفحص** — والطلبُ يبقى `pending`
// فيُقرأ `PARTIAL` ويُنهي نفسَه. **ولذلك بقي `EV-09` «يُقاس في
// التشغيل».**
//
// # والعقدُ (`EV-09`)
//
//	الجمهور     صاحبُ المتجر
//	القنوات     في التطبيق · وبثٌّ حيّ
//	الديمومة    DURABLE REQUIRED
//
// **فالمنصّةُ تقبل نيابةً عنه** — **ومن لم يعلم لم يطبخ**، والزبونُ
// يقرأ «قيد التحضير» ولا أحدَ خلف الباب.

// autoTransferOrder يُشغّل التحويلَ التلقائيَّ ويُنشئ طلباً.
func autoTransferOrder(t *testing.T, hh *Harness, m *Merchant) (oid string, before int) {
	t.Helper()
	// **والمفتاحُ الصحيح** — منطقيٌّ لا عتبة.
	hh.Setting("orders.auto_transfer", "true")
	item := hh.NewItemFor(m, 1500)

	before = evNotifCount(t, hh, m.Owner.ID)
	made := hh.POSTKey("/api/v1/orders", hh.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ = made.JSON()["id"].(string)
	return oid, before
}

// waitStatus ينتظر أن يبلغ الطلبُ حالاً غيرَ `pending`.
//
// **والخيطُ لا يُنتظَر** (`XOB-7`) — فيُنتظَر أثرُه في القاعدة.
func waitStatus(t *testing.T, hh *Harness, oid string, d time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(d)
	var status string
	for {
		if err := hh.Pool.QueryRow(ctxBG(),
			`SELECT status FROM orders WHERE id = $1::uuid`, oid).Scan(&status); err != nil {
			t.Fatalf("حالُ الطلب: %v", err)
		}
		if status != "pending" || time.Now().After(deadline) {
			return status
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T1 · وضعُ المتاجر: تُقبَل نيابةً عنه — أيعلم؟**
// ══════════════════════════════════════════════════════════════════════
//
// **وفي هذا الوضع لا رسالةَ واتساب**: «البوّابةُ هي القناة».
// **فالسؤالُ**: **أيصل صاحبَ المتجر خبرٌ دائمٌ أنّ طلباً قُبل باسمه؟**
func TestR21_T1_SelfManageAutoTransferInformsMerchant(t *testing.T) {
	hh := New(t)
	hh.Setting("platform.orders_mode", `"merchants"`)
	m := hh.Factory().Merchant()

	cap := hh.Listen("merchant:" + m.ID)
	oid, before := autoTransferOrder(t, hh, m)
	status := waitStatus(t, hh, oid, 5*time.Second)
	msgs := cap.Drain(2 * time.Second)
	waitNotif(hh, m.Owner.ID, before+1, 3*time.Second)
	after := evNotifCount(t, hh, m.Owner.ID)

	t.Logf("T1: الحالُ=%q · بثٌّ للمتجر=%d · إشعاراتٌ %d ⇒ %d",
		status, len(msgs), before, after)

	if status == "pending" {
		t.Fatalf("**لم يقع تحويلٌ تلقائيّ** — **والتركيبةُ خطأ**، فلا يُقاس شيء")
	}

	// **والعقدُ يشترط الديمومةَ لا البثَّ وحدَه.**
	//
	// **وبثٌّ يصل الشاشاتِ المفتوحةَ وحدَها** — **ومن أغلق لوحتَه
	// ليلاً لم يعلم أنّ طلباً قُبل باسمه.**
	if after <= before {
		t.Errorf("**قُبل الطلبُ باسم المتجر ولا خبرَ دائمٌ يبلغه** "+
			"(إشعاراتٌ %d ⇒ %d · بثٌّ=%d) — **ومن لم يعلم لم يطبخ، "+
			"والزبونُ يقرأ «قيد التحضير» ولا أحدَ خلف الباب.** "+
			"(`R21` · `EV-09`)", before, after, len(msgs))
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T2 · وضعُ المنصّة: البوتُ غيرُ جاهزٍ ⇒ لا يُقبَل أصلاً**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا العقدُ القائمُ في الشيفرة**: **القناةُ تُفحص قبل القبول لا
// بعده** — **فلا يبقى الطلبُ «مقبولاً» بلا أن يعلم به المتجر.**
func TestR21_T2_PlatformModeChecksChannelBeforeAccepting(t *testing.T) {
	hh := New(t)
	hh.Setting("platform.orders_mode", `"platform"`)
	m := hh.Factory().Merchant()

	oid, _ := autoTransferOrder(t, hh, m)
	status := waitStatus(t, hh, oid, 3*time.Second)
	t.Logf("T2: البوتُ غيرُ جاهزٍ في الفحص · الحالُ=%q", status)

	if status != "pending" {
		t.Errorf("**قُبل الطلبُ والقناةُ غيرُ جاهزة** (%q) — "+
			"**فيبقى «مقبولاً» ولا يعلم به المتجر.** (`R21`)", status)
	}
}
