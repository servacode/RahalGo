package qa

// ══════════════════════════════════════════════════════════════════════
// **«قُبل» ليست نهايةَ المحاولة** — `XG-44`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان يقع
//
// **`settleTransfer` كانت تعود عند `status != 'pending'`** — **وذاك
// أوّلُ الأفعال الثلاثة لا آخرُها.** **والمحاولةُ خيطٌ مستقلٌّ**
// (`go s.autoTransfer(context.WithoutCancel(...))`) **وترتيبُه**:
// قبولٌ ⇒ إبلاغٌ ⇒ وسمٌ ⇒ إنزال.
//
// **فيُقرأ منتصفُ عمليّةٍ ويُحكَم عليه**: `F5` تقرأ بين القبول
// والإبلاغ فترى **`نداءات=0`**، و`F7` تقرأ بين الإبلاغ والوسم فترى
// **نداءً واحداً بلا وسم.** **وتوقيعان لعلّةٍ واحدةٍ على عمقين.**
//
// **وقيس منفرداً**: عشرون تشغيلاً لكلٍّ ⇒ `F5` أربعُ سقطات · `F7`
// ثلاث. **فلا سابقةَ ولا تلوّث** — **ولا عطبَ في `R24`.**
//
// # وهذا الفحصُ يقيس عقدَ المُنتظِر نفسِه
//
// **بلا خيطٍ ولا سباق**: **تُصنَع الحالُ بيدٍ ثمّ يُقاس متى يعود
// المُنتظِر.** **فمن أعاد الشرطَ القديمَ رآه يسقط حتماً لا مصادفةً.**

import (
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

func TestXG44_SettleWaitsUntilTheTransferAttemptEnds(t *testing.T) {
	h := New(t)
	// **ولا تحويلَ تلقائيّ** — **الحالُ تُصنَع بيدٍ، فلا خيطَ يزاحم
	// القياس.**
	h.Setting("platform.orders_mode", `"platform"`)
	h.Setting("orders.auto_transfer", "false")
	m := h.Factory().Merchant()

	elapsed := func(oid string) (time.Duration, transferProbe) {
		t.Helper()
		start := time.Now()
		p := settleTransfer(t, h, oid)
		return time.Since(start), p
	}

	// ── ١ ── **قُبل ولا شيءَ بعد: المحاولةُ لم تنتهِ** ────────────
	//
	// **فينتظر إلى مهلته** — **ولا يعود بحكمٍ على منتصف.**
	mid := placeTransferOrder(t, h, m)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'accepted' WHERE id = $1::uuid`, mid); err != nil {
		t.Fatalf("صنعُ الحال: %v", err)
	}
	d, p := elapsed(mid)
	t.Logf("MID-FLIGHT: عاد بعد %s · الحالُ=%q · مُرسَلٌ=%v",
		d.Round(time.Millisecond), p.Status, p.SentAt != nil)
	if d < 2*time.Second {
		t.Errorf("**عاد بعد %s ومحاولتُه لم تنتهِ** — "+
			"**و«قُبل» أوّلُ الثلاثة لا آخرُها.** (`XG-44`)",
			d.Round(time.Millisecond))
	}

	// ── ٢ ── **ووُسم مُرسَلاً: انتهت** ───────────────────────────
	sent := placeTransferOrder(t, h, m)
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'accepted', sent_to_merchant_at = now()
		 WHERE id = $1::uuid`, sent); err != nil {
		t.Fatalf("صنعُ الحال: %v", err)
	}
	d, p = elapsed(sent)
	t.Logf("SENT: عاد بعد %s · مُرسَلٌ=%v", d.Round(time.Millisecond), p.SentAt != nil)
	if d > time.Second {
		t.Errorf("**انتظر %s وقد وُسم** — **والوسمُ نهاية**", d.Round(time.Millisecond))
	}
	if p.SentAt == nil {
		t.Error("**قُرئ بلا وسمٍ وقد وُسم**")
	}

	// ── ٣ ── **أو بقي أثرُ فشلٍ منسوبٌ إليه: انتهت أيضاً** ────────
	//
	// **وهذا هو طريقُ `F5`** — **الرسالةُ سقطت فبقي ما يُقرأ.**
	failed := placeTransferOrder(t, h, m)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'accepted' WHERE id = $1::uuid`, failed); err != nil {
		t.Fatalf("صنعُ الحال: %v", err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_events (order_id, actor_id, from_status, to_status, note)
		VALUES ($1::uuid, NULL, 'accepted', 'accepted', $2)`,
		failed, orders.TransferFailureNote+" — قياسُ العقد"); err != nil {
		t.Fatalf("أثرُ الفشل: %v", err)
	}
	d, _ = elapsed(failed)
	t.Logf("FAILURE TRACE: عاد بعد %s", d.Round(time.Millisecond))
	if d > time.Second {
		t.Errorf("**انتظر %s وأثرُ الفشل مكتوب** — **والأثرُ نهايةٌ أيضاً**",
			d.Round(time.Millisecond))
	}
}
