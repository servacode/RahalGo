package qa

// ══════════════════════════════════════════════════════════════════════
// **لـ«لدي توصيلة» مفاتيحُ أجرتها هي** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت تقرأ مفاتيحَ الطلب العاديّ** — فمن رفع أجرةَ العاديّ أو أشعل مسافتَه رفع أجرةَ
// توصيلة المتجر معه. **ويُقاس هنا بعرض السعر والإنشاء معاً** (دالّةٌ واحدة).

import (
	"fmt"
	"math"
	"net/http"
	"testing"
)

func TestMDFeeKeys_OwnSettingsNotStandard(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ مفاتيح التوصيلة")
	quote := func() map[string]any {
		r := h.GET(fmt.Sprintf("%s?lat=%f&lng=%f", m.url("/delivery-quote"), m.dropLat, m.dropLng), m.fx.Tok)
		if r.Code != http.StatusOK {
			t.Fatalf("عرضُ السعر: %d / %s", r.Code, r.Err())
		}
		return r.JSON()
	}

	// ── ١ · مفاتيحُ العاديّ تتبدّل كلُّها — وأجرةُ التوصيلة لا تتحرّك ──
	h.Setting("delivery.fee", "9000")
	h.Setting("delivery.by_distance", "true")
	h.Setting("delivery.per_km", "777")
	h.Setting("delivery.max_fee", "0")
	if j := quote(); j["fee"] != float64(500) {
		t.Fatalf("**أجرةُ التوصيلة %v بعد تبديل مفاتيح العاديّ** — والمنتظَر أساسُها 500", j["fee"])
	}

	// ── ٢ · ومفاتيحُها هي تحكمها ──
	h.Setting("delivery.merchant_base_fee", "650")
	h.Setting("delivery.merchant_by_distance", "true")
	h.Setting("delivery.merchant_per_km", "40")
	h.Setting("delivery.merchant_max_fee", "0")
	j := quote()
	d, _ := j["distance_m"].(float64)
	want := 650 + 40*int64(math.Ceil(d/1000))
	if j["fee"] != float64(want) || j["fee_basis"] != "distance" {
		t.Fatalf("الأجرةُ %v (%v) والمنتظَر %d — أساسُها ٦٥٠ وكيلومترُها ٤٠ على %.0fم",
			j["fee"], j["fee_basis"], want, d)
	}
	// **والسقفُ سقفُها**.
	h.Setting("delivery.merchant_max_fee", "660")
	if j := quote(); j["fee"] != float64(660) {
		t.Fatalf("السقفُ: %v والمنتظَر 660", j["fee"])
	}

	// ── ٣ · والإنشاءُ بالرقم نفسِه ──
	id := m.create(t, h, "recipient", uniq("mdkeys"))
	var fee int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT delivery_fee FROM orders WHERE id = $1::uuid`, id).Scan(&fee)
	if fee != 660 {
		t.Fatalf("أجرةُ الإنشاء %d — والمنتظَر 660 كعرض السعر", fee)
	}

	// ── ٤ · والثابتةُ تسبقها كلَّها كما كانت ──
	h.Setting("delivery.merchant_fee", "700")
	if j := quote(); j["fee"] != float64(700) || j["fee_basis"] != "flat" {
		t.Fatalf("الأجرةُ الثابتة: %v (%v) والمنتظَر 700 flat", j["fee"], j["fee_basis"])
	}
}
