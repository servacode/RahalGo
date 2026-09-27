package qa

// ══════════════════════════════════════════════════════════════════════
// **إثباتُ التسليم يفشل آمناً، وبابُ الاستثناء موثَّقٌ لا مجّانيّ**
// ══════════════════════════════════════════════════════════════════════
//
// المعرّفات: `PROOF-*` · قرارُ المالك (البند ٣، ٢٠٢٦-٠٩-٢٧)
//
// # المسألة
//
// **صورةٌ مطلوبةٌ وكاميرا لا تعمل** — هاتفٌ عطب، أو إذنٌ رُفض، أو ليلٌ لا يُرى
// فيه شيء. **قرارُ المالك: لا بابَ تخطٍّ مجّانيّ** — يفشل التسليمُ آمناً، **وبابُ
// الاستثناء يبقى موثَّقاً**: كلمةٌ تُكتب وتُقرأ يومَ النزاع، **ولا تُضعَّف
// القاعدةُ في صمت.**
//
// **والقاعدةُ قائمةٌ في المحرّك أصلاً** (`drivers.require_delivery_photo`
// افتراضُه صحيح، و`requireProofBeforeDelivery`، و`handleSkipDeliveryProof`
// بسببٍ إلزاميٍّ مُدقَّق). **وهذا الاختبارُ يُقفلها** فلا تُضعَّف بلا أن يسقط
// شيءٌ يُرى: يُثبت أنّ التسليمَ يُمنع بلا إثبات، وأنّ التخطّي بلا سببٍ يُرفض،
// وأنّ التخطّي بسببٍ يُقيَّد ثمّ يمرّ.
//
// **وما لم يُحسَم — قرارُ مالكٍ مطلوب**: أيبقى التخطّي بيدِ السائق بسببٍ مُدقَّق،
// أم يُرفع إلى إذنِ الأدمن؟ **لا يُغيَّر هنا** — بابُ إذنِ الأدمن قد يُوقف
// سائقاً في الشارع ينتظر ردّاً، والتخطّي المُدقَّق قابلٌ للقراءة والمساءلة.

import (
	"net/http"
	"testing"
	"time"
)

// TestPROOF_NoCameraFailsSafeWithAuditedException **الكاميرا لا تعمل: يفشل
// آمناً، والاستثناءُ موثَّقٌ لا مجّانيّ.**
func TestPROOF_NoCameraFailsSafeWithAuditedException(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("PROOF تعذّر إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	drv := h.driverOf(oid)
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to}); got.Code >= 400 {
			t.Fatalf("PROOF الانتقالُ إلى %s رُدّ: %s", to, got)
		}
	}

	// ── (١) يفشل آمناً: لا «سُلّم» بلا إثباتٍ ولا تخطٍّ ──
	bare := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"})
	if bare.Code != http.StatusConflict || bare.Err() != "delivery_proof_required" {
		t.Fatalf("PROOF-1 سُلّم — أو رُدّ بغير delivery_proof_required — بلا إثبات: %s", bare)
	}
	if got := h.statusOf(oid); got != "at_dropoff" {
		t.Fatalf("PROOF-1 الحالُ %q بعد منعِ التسليم — **تسلّم رغم المنع**", got)
	}

	// ── (٢) ليس باباً مجّانيّاً: تخطٍّ بلا سببٍ يُرفض ──
	//
	// **وهو ما يميّز الاستثناءَ المُدقَّق من البابِ المجّانيّ**: سببٌ يُكتب.
	empty := h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
		map[string]any{"reason": ""})
	if empty.Code < 400 {
		t.Fatalf("PROOF-2 تخطٍّ بسببٍ فارغٍ قُبل — **بابٌ مجّانيّ**: %s", empty)
	}
	none := h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
		map[string]any{})
	if none.Code < 400 {
		t.Fatalf("PROOF-2 تخطٍّ بلا حقلِ سببٍ قُبل — **بابٌ مجّانيّ**: %s", none)
	}

	// ── (٣) بابُ الاستثناء موثَّق: تخطٍّ بسببٍ يُقبل ويُقيَّد ──
	skip := h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
		map[string]any{"reason": "الكاميرا معطّلةٌ والضوءُ خافت"})
	if skip.Code >= 400 {
		t.Fatalf("PROOF-3 تعذّر التخطّي بسببٍ صريح: %s", skip)
	}

	// **والتخطّي يُقرأ يومَ النزاع** — يُقيَّد في السجلّ (كتابةٌ بالخلفيّة فيُنتظر).
	deadline := time.Now().Add(3 * time.Second)
	var audited int
	for time.Now().Before(deadline) {
		_ = h.Pool.QueryRow(h.T.Context(), `
			SELECT count(*) FROM audit_log
			WHERE action = 'driver.delivery_proof_skipped' AND entity_id = $1`,
			oid).Scan(&audited)
		if audited > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if audited == 0 {
		t.Errorf("PROOF-3 التخطّي لم يُقيَّد — **صامتٌ لا يُقرأ يومَ النزاع**")
	}

	// ── (٤) وبعد التخطّي الموثَّق يمرّ التسليم ──
	done := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"})
	if done.Code >= 400 {
		t.Fatalf("PROOF-4 التسليمُ رُدّ بعد التخطّي الموثَّق: %s", done)
	}
	if got := h.statusOf(oid); got != "delivered" {
		t.Fatalf("PROOF-4 الحالُ %q بعد التسليم — يُنتظر delivered", got)
	}
}
