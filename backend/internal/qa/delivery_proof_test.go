package qa

// ══════════════════════════════════════════════════════════════════════
// **تخطّي إثبات التسليم إذنُ عملياتٍ مُخوَّلٌ لا فعلُ سائق**
// ══════════════════════════════════════════════════════════════════════
//
// المعرّفات: `PROOF-*` · قرارُ المالك النهائيّ (٢٠٢٦-٠٩-٢٧)
//
// # القرار
//
// **لا يتخطّى السائقُ الصورةَ الإلزاميّةَ بكلمةٍ يكتبها بنفسه.** للحالة الحقيقيّة
// (كاميرا معطّلةٌ أو غيرُ متاحة): مسارُ السائق العاديُّ **يفشل آمناً**،
// والاستثناءُ يحتاج **إذنَ أدمن/عمليّاتٍ مُخوَّل**، السببُ إلزاميّ، ويُدقَّق
// (الفاعلُ + السائقُ + الطلبُ + السببُ + الوقت) دائماً، وإعادةُ النداء آمنة،
// **والسائقُ العاديُّ لا يأذن لنفسه.**
//
// **وهذا الاختبارُ يُقفل كلَّ بندٍ منها** فلا يُضعَّف صمتاً.

import (
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// authorizeProofExempt **بديلُ العُدّة لإذنِ الاستثناء من العمليّات**: يعلّم
// الطلبَ قابلاً للتسليم بلا صورة (`pod_skip_by`). **والسائقُ لم يعد يتخطّى
// بنفسه** (قرارُ المالك ٢٠٢٦-٠٩-٢٧)؛ فالاختباراتُ التي تحتاج فقط الوصولَ إلى
// `delivered` تستعمل هذا بدل بابِ التخطّي المحذوف — والمسارُ الحقيقيُّ للاستثناء
// هو `POST /admin/orders/{id}/proof-exception` (يفحصه `PROOF-*`).
func (h *Harness) authorizeProofExempt(orderID string) {
	h.T.Helper()
	if _, err := h.Pool.Exec(h.T.Context(), `
		UPDATE orders SET pod_skip_by = driver_id, pod_skip_reason = 'qa-fixture', pod_skip_at = now()
		WHERE id = $1::uuid AND driver_id IS NOT NULL`, orderID); err != nil {
		h.T.Fatalf("qa: authorizeProofExempt: %v", err)
	}
}

// TestPROOF_NoCameraNeedsOpsAuthorizedException **الكاميرا لا تعمل: يفشل آمناً،
// والاستثناءُ إذنُ عملياتٍ مُخوَّلٌ مُدقَّقٌ لا كلمةُ سائق.**
func TestPROOF_NoCameraNeedsOpsAuthorizedException(t *testing.T) {
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

	// ── (١) يفشل آمناً: لا «سُلّم» بلا إثباتٍ ولا إذنِ استثناء ──
	bare := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"})
	if bare.Code != http.StatusConflict || bare.Err() != "delivery_proof_required" {
		t.Fatalf("PROOF-1 سُلّم — أو رُدّ بغير delivery_proof_required — بلا إثبات: %s", bare)
	}

	// ── (٢) لا يأذن السائقُ لنفسه: لا مسارَ تخطٍّ للسائق أصلاً ──
	gone := h.POST("/api/v1/driver/orders/"+oid+"/proof/skip", drv.Token,
		map[string]any{"reason": "الكاميرا معطّلة"})
	if gone.Code != http.StatusNotFound {
		t.Fatalf("PROOF-2 مسارُ تخطّي السائق ما زال حيّاً (%s) — **السائقُ يأذن لنفسه**", gone)
	}

	// ── (٣) وحتّى على باب العمليّات: السائقُ بلا قدرةٍ يُمنع ──
	drvTry := h.POST("/api/v1/admin/orders/"+oid+"/proof-exception", drv.Token,
		map[string]any{"reason": "محاولةُ سائق"})
	if drvTry.Code != http.StatusForbidden && drvTry.Code != http.StatusUnauthorized {
		t.Fatalf("PROOF-3 سائقٌ أذن لنفسه عبر باب العمليّات (%s) — **لا فصلَ سلطة**", drvTry)
	}
	// **ولم يُفتَح الباب**: لا يزال محجوباً.
	if still := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); still.Code != http.StatusConflict {
		t.Fatalf("PROOF-3 تسلّم بعد محاولة السائق (%s) — **تخطٍّ التفافيّ**", still)
	}

	// ── عملياتٌ مُخوَّلة (قدرةُ OrdersIntervene) ──
	capRole(t, h, "qa_ops_proof", authz.OrdersIntervene)
	ops, opsTok := capUser(t, h, "qa_ops_proof")

	// ── (٤) السببُ إلزاميّ ──
	noReason := h.POST("/api/v1/admin/orders/"+oid+"/proof-exception", opsTok,
		map[string]any{"reason": ""})
	if noReason.Code < 400 {
		t.Fatalf("PROOF-4 إذنٌ بلا سببٍ قُبل: %s", noReason)
	}

	// ── (٥) إذنٌ بسببٍ من مُخوَّلٍ: يمرّ · يُسجَّل مَن أذن · يُدقَّق مرّةً ──
	auth1 := h.POST("/api/v1/admin/orders/"+oid+"/proof-exception", opsTok,
		map[string]any{"reason": "الكاميرا معطّلةٌ والضوءُ خافت"})
	if auth1.Code >= 400 {
		t.Fatalf("PROOF-5 تعذّر إذنُ الاستثناء من مُخوَّل: %s", auth1)
	}
	var skipBy *string
	if err := h.Pool.QueryRow(h.T.Context(),
		`SELECT pod_skip_by::text FROM orders WHERE id = $1`, oid).Scan(&skipBy); err != nil {
		t.Fatalf("PROOF-5 قراءةُ pod_skip_by: %v", err)
	}
	if skipBy == nil || *skipBy != ops.ID {
		t.Fatalf("PROOF-5 لم يُسجَّل مَن أذن (skipBy=%v · ops=%s)", skipBy, ops.ID)
	}
	if n := auditCount(t, h, "ops.delivery_proof_exception", oid); n != 1 {
		t.Fatalf("PROOF-5 تدقيقُ الإذن = %d، والمتوقّع 1 (الفاعل+السائق+الطلب+السبب+الوقت)", n)
	}

	// ── (٦) إعادةُ النداء آمنة: لا تدقيقَ ثانٍ ──
	auth2 := h.POST("/api/v1/admin/orders/"+oid+"/proof-exception", opsTok,
		map[string]any{"reason": "نداءٌ ثانٍ"})
	if auth2.Code >= 400 {
		t.Fatalf("PROOF-6 إعادةُ النداء رُدّت: %s", auth2)
	}
	if n := auditCount(t, h, "ops.delivery_proof_exception", oid); n != 1 {
		t.Fatalf("PROOF-6 تكرّر التدقيقُ (%d) — **النداءُ ليس مأموناً**", n)
	}

	// ── (٧) وبعد الإذن الموثَّق يمرّ التسليم ──
	done := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"})
	if done.Code >= 400 {
		t.Fatalf("PROOF-7 التسليمُ رُدّ بعد الإذن: %s", done)
	}
	if got := h.statusOf(oid); got != "delivered" {
		t.Fatalf("PROOF-7 الحالُ %q بعد التسليم — يُنتظر delivered", got)
	}
}
