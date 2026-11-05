package qa

// **مفاتيحُ أجرة «طلب واتساب» وحصّتِه** (ملحقُ المالك ٢٠٢٦-١١-٠٥، هـ) — والهجرةُ تنسخ قيمَ «لدي توصيلة».

import (
	"os"
	"testing"
)

// TestWAORD022_OwnSettingsCopiedByMigration **هـ · مفاتيحُ «طلب واتساب» تُنسخ من «لدي توصيلة»** —
// فلا يتبدّل رقمٌ يومَ النشر. **والأجرةُ الثابتةُ لـ«لدي توصيلة» تصير أساساً بمسافةٍ مطفأة.**
//
// **وعلى الشيفرة القديمة لا هجرة** (`0507`).
func TestWAORD022_OwnSettingsCopiedByMigration(t *testing.T) {
	h := New(t)
	raw, err := os.ReadFile("../migrate/migrations/0507_whatsapp_orders_location_settings.sql")
	if err != nil {
		t.Fatalf("**هجرةُ الملحق غائبة**: %v", err)
	}
	keys := []string{"delivery.whatsapp_base_fee", "delivery.whatsapp_by_distance", "delivery.whatsapp_per_km",
		"delivery.whatsapp_max_fee", "delivery.share_whatsapp_mode", "delivery.share_whatsapp"}
	wipe := func() {
		if _, err := h.Pool.Exec(ctxBG(), `DELETE FROM app_settings WHERE key = ANY($1)`, keys); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(wipe)
	read := func() map[string]string {
		out := map[string]string{}
		rows, err := h.Pool.Query(ctxBG(), `SELECT key, value::text FROM app_settings WHERE key = ANY($1)`, keys)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var k, v string
			if err := rows.Scan(&k, &v); err != nil {
				t.Fatal(err)
			}
			out[k] = v
		}
		return out
	}

	h.Setting("delivery.merchant_fee", "0")
	h.Setting("delivery.merchant_base_fee", "700")
	h.Setting("delivery.merchant_by_distance", "true")
	h.Setting("delivery.merchant_per_km", "30")
	h.Setting("delivery.merchant_max_fee", "900")
	h.Setting("delivery.share_merchant_mode", `"fixed"`)
	h.Setting("delivery.share_merchant", "40")
	wipe()
	if _, err := h.Pool.Exec(ctxBG(), string(raw)); err != nil {
		t.Fatalf("الهجرةُ لا تُعاد: %v", err)
	}
	want := map[string]string{
		"delivery.whatsapp_base_fee": "700", "delivery.whatsapp_by_distance": "true",
		"delivery.whatsapp_per_km": "30", "delivery.whatsapp_max_fee": "900",
		"delivery.share_whatsapp_mode": `"fixed"`, "delivery.share_whatsapp": "40",
	}
	got := read()
	for k, v := range want {
		if got[k] != v {
			t.Errorf("**%s = %q — والمنتظَر %q** (قيمةُ «لدي توصيلة»)", k, got[k], v)
		}
	}

	// **وما ضُبط قبلها لا يُمسّ.**
	h.Setting("delivery.whatsapp_base_fee", "111")
	if _, err := h.Pool.Exec(ctxBG(), string(raw)); err != nil {
		t.Fatal(err)
	}
	if v := read()["delivery.whatsapp_base_fee"]; v != "111" {
		t.Errorf("**الهجرةُ كتبت فوق قيمةٍ مضبوطة**: %s", v)
	}

	// **والأجرةُ الثابتةُ لـ«لدي توصيلة» تصير أساساً بمسافةٍ مطفأة** — الرقمُ نفسُه.
	wipe()
	h.Setting("delivery.merchant_fee", "300")
	if _, err := h.Pool.Exec(ctxBG(), string(raw)); err != nil {
		t.Fatal(err)
	}
	got = read()
	if got["delivery.whatsapp_base_fee"] != "300" || got["delivery.whatsapp_by_distance"] != "false" {
		t.Errorf("**الأجرةُ الثابتة لم تُنسخ**: أساس %s · مسافة %s",
			got["delivery.whatsapp_base_fee"], got["delivery.whatsapp_by_distance"])
	}
}
