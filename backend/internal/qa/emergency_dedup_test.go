package qa

// ══════════════════════════════════════════════════════════════════════
// **بلاغُ الطوارئ لا يتكرّر لطلبٍ مفتوح** (`DRV-DEF-001`)
// ══════════════════════════════════════════════════════════════════════
//
// **كان الزرُّ يُدرج صفّاً بلا حارس** — فإعادةُ الضغطِ بعد فشلِ شبكةٍ (والنداءُ
// قد وصل وضاع ردُّه) تُنشئ بلاغاً ثانياً لطارئٍ واحد. **والآن حارسٌ ذرّيّ**:
// فهرسٌ فريدٌ جزئيٌّ `(order_id) WHERE status='open'` + `ON CONFLICT`، فتُرتَدُّ
// الإعادةُ إلى البلاغِ القائمِ (`duplicate=true`) لا تُنشئ ثانياً.
//
// **والتسجيلُ مرّةً واحدةً، والإخطارُ مرّةً على الأقلّ** — وهو العقدُ الصحيحُ
// لإنذارِ سلامة. **وحمايتان تمنعان التكرار**: هذا الحارسُ، وتصفيةُ `driver_id`
// عند التحرير (فإعادةٌ بعد نجاحٍ تامٍّ يردّها فحصُ الملكيّة).

import (
	"net/http"
	"testing"
)

// TestDRVDEF001_EmergencyDedupedPerOpenOrder **بلاغٌ مفتوحٌ واحدٌ لكلّ طلب.**
func TestDRVDEF001_EmergencyDedupedPerOpenOrder(t *testing.T) {
	h := New(t)
	z := zoneForDemand(t, h, "منطقةُ DRV-DEF-001")
	cust := h.Customer()
	drv := h.NewUser("driver")
	oid := placeOrder(t, h, cust, z, h.NewItem(1000))
	assignDriver(t, h, oid, drv.ID)

	// ── الضغطةُ الأولى: بلاغٌ جديد ────────────────────────────────
	r1 := h.POST("/api/v1/driver/orders/"+oid+"/emergency", drv.Token,
		map[string]any{"note": "الأولى"})
	if r1.Code != http.StatusCreated {
		t.Fatalf("**البلاغُ الأوّل رُدّ**: %d / %s", r1.Code, r1.Err())
	}
	if dup, _ := r1.JSON()["duplicate"].(bool); dup {
		t.Fatalf("**البلاغُ الأوّل عُدَّ إعادةً**: %v", r1.JSON())
	}
	id1, _ := r1.JSON()["emergency_id"].(string)
	if id1 == "" {
		t.Fatalf("**لا معرّفَ بلاغٍ في الردّ الأوّل**: %v", r1.JSON())
	}

	// **التحريرُ صفّى `driver_id`** — نُعيده لنختبر الحارسَ عينَه (كحالِ إعادةٍ
	// وقعت قبل اكتمال التحرير، فبقيت الملكيّة).
	assignDriver(t, h, oid, drv.ID)

	// ── الضغطةُ الثانية (إعادة): تُرتَدُّ إلى القائم ─────────────────
	r2 := h.POST("/api/v1/driver/orders/"+oid+"/emergency", drv.Token,
		map[string]any{"note": "إعادة"})
	if r2.Code != http.StatusCreated {
		t.Fatalf("**الإعادةُ رُدّت**: %d / %s", r2.Code, r2.Err())
	}
	if dup, _ := r2.JSON()["duplicate"].(bool); !dup {
		t.Fatalf("**الإعادةُ لم تُعرَف إعادةً** — يجب `duplicate=true`: %v", r2.JSON())
	}
	if id2, _ := r2.JSON()["emergency_id"].(string); id2 != id1 {
		t.Fatalf("**الإعادةُ أنشأت بلاغاً بمعرّفٍ آخر**: %q ≠ %q", id2, id1)
	}

	// ── العدُّ الحاسم: بلاغٌ مفتوحٌ واحدٌ لا اثنان ───────────────────
	var open int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM driver_emergencies WHERE order_id = $1::uuid AND status = 'open'`,
		oid).Scan(&open); err != nil {
		t.Fatalf("عدُّ البلاغات: %v", err)
	}
	if open != 1 {
		t.Fatalf("**بلاغاتٌ مفتوحةٌ متعدّدةٌ لطلبٍ واحد**: %d (المنتظَر 1) — "+
			"**فتُربَك العملياتُ بحادثةٍ تُقرأ مرّتين**", open)
	}
}
