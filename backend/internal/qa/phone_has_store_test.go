package qa

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **رقمٌ له متجرٌ لا يُفتح عليه متجرٌ آخر** (قرارُ المالك ٢٠٢٦-١٠-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// «لازم يعطي: الرقم مستخدم من قبل متجر آخر، لا يمكن فتح حساب بهذا الرقم» —
// من تطبيق المندوب، وعند الموافقة، ومن اللوحة (ولا فرعَ ثانياً بعد).
func TestPhoneHasStore_EveryDoorRefuses(t *testing.T) {
	h := New(t)
	rep, district, category := repLeadEnv(t, h)
	admin := h.NewUser("admin")
	phone := uniqPhone()

	// ١ · الطلبُ الأوّل يمرّ.
	first := h.POSTKey("/api/v1/rep/leads", rep.Token, uniq("k"), repLeadBody(district, category, phone))
	if first.Code >= 400 {
		t.Fatalf("الطلبُ الأوّل رُفض: %s", first)
	}
	// ٢ · وثانٍ بالرقم وهو ينتظر ⇒ يُرفض.
	if r := h.POSTKey("/api/v1/rep/leads", rep.Token, uniq("k"), repLeadBody(district, category, phone)); r.Err() != "phone_has_store" {
		t.Fatalf("**طلبٌ ثانٍ بالرقم نفسِه وهو ينتظر قُبل**: %s", r)
	}
	// ٣ · يُحوَّل الأوّلُ متجراً.
	id, _ := first.JSON()["id"].(string)
	if r := h.POST("/api/v1/admin/leads/"+id+"/status", admin.Token,
		map[string]any{"status": "converted"}); r.Code >= 400 {
		t.Fatalf("التحويل: %s", r)
	}
	// ٤ · وبعدها من المندوب ⇒ يُرفض.
	if r := h.POSTKey("/api/v1/rep/leads", rep.Token, uniq("k"), repLeadBody(district, category, phone)); r.Err() != "phone_has_store" {
		t.Fatalf("**فُتح طلبٌ برقم متجرٍ قائم**: %s", r)
	}
	// ٥ · ومن اللوحة ⇒ يُرفض — ولا فرعَ ثانياً.
	if r := h.POST("/api/v1/admin/merchants", admin.Token, map[string]any{
		"name": "فرعٌ ثانٍ", "category_id": category, "lat": 35.95, "lng": 39.01,
		"owner_phone": phone, "owner_name": "صاحب",
	}); r.Err() != "phone_has_store" {
		t.Fatalf("**اللوحةُ فتحت متجراً ثانياً بالرقم نفسِه**: %s", r)
	}
	// ٦ · وطلبٌ قديمٌ بالرقم (قبل القاعدة) ⇒ يُرفض عند التحويل.
	old := newLeadFor(t, h, rep.ID, category, phone)
	if r := h.POST("/api/v1/admin/leads/"+old+"/status", admin.Token,
		map[string]any{"status": "converted"}); r.Err() != "phone_has_store" {
		t.Fatalf("**تحويلُ طلبٍ قديمٍ برقم متجرٍ قائم مرّ**: %s", r)
	}
}
