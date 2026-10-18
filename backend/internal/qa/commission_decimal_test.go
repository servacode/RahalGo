package qa

import (
	"encoding/json"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **عمولةُ المنصّة بمنزلتين عشريّتين** (قرارُ المالك ٢٠٢٦-١٠-١٨: «١٫٥٪»)
// ══════════════════════════════════════════════════════════════════════
//
// **وقبل الإصلاح**: الإعدادُ `1.5` كان يُقرأ بـ`GetInt` فيُبتر إلى ١٪، واللقطةُ
// تفكّه في `int64` فتفشل فتُلقط العشرةُ الافتراضيّة — **فيُقتطع من المتجر ١٠٪
// وهو موعودٌ بـ١٫٥٪.** وتجاوزُ المتجر `2.25` كان يُردّ عند الفكّ (`int64`).
//
// **والحسبةُ بأجزاء المئة وبالبتر نفسِه**: النسبُ الصحيحةُ ليرةً بليرة كما كانت.

// decimalOrder طلبٌ بصنفٍ واحدٍ بكلفةٍ معلومة بلا هامش، يُسلَّم بانتقالاته الحقيقيّة.
func decimalOrder(t *testing.T, h *Harness, m *Merchant, cost int64) string {
	t.Helper()
	item := h.NewItemFor(m, cost)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("pct2"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	deliverOrder(t, h, oid, h.driverOf(oid))
	return oid
}

// decimalLedger عمولةُ المنصّة على الطلب وقيدُ مستحقّ المتجر ونسبةُ اللقطة نصّاً.
func decimalLedger(t *testing.T, h *Harness, oid string) (platform, earning int64, snap string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT o.platform_commission,
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = o.id::text AND kind = 'merchant_earning'), 0),
		       COALESCE(o.snap_merchant_commission_percent::text, '')
		  FROM orders o WHERE o.id = $1::uuid`, oid).Scan(&platform, &earning, &snap); err != nil {
		t.Fatalf("قراءةُ الدفتر: %v", err)
	}
	return platform, earning, snap
}

// rawData **ما في غلاف `{"data": …}` خاماً** — ليُرى نوعُ الرقم كما يصل التطبيق.
func rawData(r Res) json.RawMessage {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(r.Body, &env) == nil && len(env.Data) > 0 {
		return env.Data
	}
	return r.Body
}

// merchantOrderJSON طلبٌ من سجلّ المتجر كما يصل التطبيق — **خاماً**، ليُرى نوعُ الرقم.
func merchantOrderJSON(t *testing.T, h *Harness, m *Merchant, oid string) map[string]json.RawMessage {
	t.Helper()
	res := h.GET("/api/v1/merchant/stores/"+m.ID+"/orders?status=delivered", m.Owner.Token)
	if res.Code >= 400 {
		t.Fatalf("سجلُّ المتجر: %s", res)
	}
	var body struct {
		Orders []map[string]json.RawMessage `json:"orders"`
	}
	if err := json.Unmarshal(rawData(res), &body); err != nil {
		t.Fatalf("فكُّ السجلّ: %v", err)
	}
	for _, o := range body.Orders {
		var id string
		_ = json.Unmarshal(o["id"], &id)
		if id == oid {
			return o
		}
	}
	t.Fatalf("الطلب %s ليس في سجلّ المتجر: %s", oid, res)
	return nil
}

// TestCommission2_GlobalOnePointFive **١٫٥٪ عامّةً — اللقطةُ والدفترُ والسجلّ.**
//
//	١٢٬٣٤٥ × ١٥٠ ÷ ١٠٬٠٠٠ = ١٨٥٫١٧٥ ⇒ ١٨٥ (بترٌ كما كان) · والمتجرُ ١٢٬١٦٠
func TestCommission2_GlobalOnePointFive(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("merchants.commission_percent", "1.5")
	m := h.Factory().Merchant()

	oid := decimalOrder(t, h, m, 12_345)
	platform, earning, snap := decimalLedger(t, h, oid)
	t.Logf("لقطة=%s · عمولةُ المنصّة=%d · مستحقُّ المتجر=%d", snap, platform, earning)
	if snap != "1.50" {
		t.Errorf("اللقطة %q — والإعدادُ 1.5", snap)
	}
	if platform != 185 {
		t.Errorf("عمولةُ المنصّة %d — والمنتظَر 185 (١٫٥٪ من ١٢٣٤٥ بالبتر)", platform)
	}
	if earning != 12_160 {
		t.Errorf("مستحقُّ المتجر %d — والمنتظَر 12160", earning)
	}

	// **والتطبيقُ القديمُ يقرأ `commission_percent` عدداً صحيحاً** (`Int` في كوتلن):
	// ١٫٥ تُقرَّب نصفيّاً إلى ٢، **والدقيقةُ في حقلٍ جديد.**
	o := merchantOrderJSON(t, h, m, oid)
	if got := string(o["commission_percent"]); got != "2" {
		t.Errorf("commission_percent = %s — والعميلُ القديمُ يحتاج عدداً صحيحاً (2)", got)
	}
	if got := string(o["commission_percent_exact"]); got != "1.5" {
		t.Errorf("commission_percent_exact = %s — والمنتظَر 1.5", got)
	}
	if got := string(o["platform_commission"]); got != "185" {
		t.Errorf("platform_commission في السجلّ = %s — والدفترُ 185", got)
	}
}

// TestCommission2_MerchantOverride **تجاوزُ المتجر ٢٫٢٥٪ من اللوحة.**
//
//	١٠٬٠٠٠ × ٢٢٥ ÷ ١٠٬٠٠٠ = ٢٢٥ · والمتجرُ ٩٬٧٧٥ — **والعامّةُ ١٠٪ لا تمسّه.**
func TestCommission2_MerchantOverride(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("merchants.commission_percent", "10")
	m := h.Factory().Merchant()
	admin := h.NewUser("admin")

	res := h.PATCH("/api/v1/admin/merchants/"+m.ID, admin.Token,
		map[string]any{"commission_percent": 2.25})
	if res.Code >= 400 {
		t.Fatalf("اللوحةُ ردّت ٢٫٢٥: %s", res)
	}
	got := h.GET("/api/v1/admin/merchants/"+m.ID, admin.Token)
	var inner map[string]json.RawMessage
	_ = json.Unmarshal(rawData(got), &inner)
	if v := string(inner["commission_percent"]); v != "2" {
		t.Errorf("commission_percent = %s — والمنتظَر 2 مقرَّباً", v)
	}
	if v := string(inner["commission_percent_exact"]); v != "2.25" {
		t.Errorf("commission_percent_exact = %s — والمنتظَر 2.25", v)
	}

	// **وثلاثُ منازلَ تُردّ ولا تُقرَّب** — ومئةٌ وواحدٌ خارجَ المدى.
	for _, bad := range []any{2.255, 101, -5} {
		if r := h.PATCH("/api/v1/admin/merchants/"+m.ID, admin.Token,
			map[string]any{"commission_percent": bad}); r.Code != 400 {
			t.Errorf("نسبة %v قُبلت (%d) — والمنتظَر ٤٠٠", bad, r.Code)
		}
	}

	oid := decimalOrder(t, h, m, 10_000)
	platform, earning, _ := decimalLedger(t, h, oid)
	t.Logf("عمولةُ المنصّة=%d · مستحقُّ المتجر=%d", platform, earning)
	if platform != 225 || earning != 9_775 {
		t.Errorf("٢٫٢٥٪ من ١٠٠٠٠: عمولة %d ومستحقّ %d — والمنتظَر 225 و9775", platform, earning)
	}
}

// TestCommission2_IntegerUnchanged **النسبةُ الصحيحةُ كما كانت ليرةً بليرة.**
//
//	١٢٬٣٤٥ × ١٠ ÷ ١٠٠ = ١٢٣٤٫٥ ⇒ ١٢٣٤ · والمتجرُ ١١٬١١١
func TestCommission2_IntegerUnchanged(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("merchants.commission_percent", "10")
	m := h.Factory().Merchant()

	oid := decimalOrder(t, h, m, 12_345)
	platform, earning, snap := decimalLedger(t, h, oid)
	t.Logf("لقطة=%s · عمولةُ المنصّة=%d · مستحقُّ المتجر=%d", snap, platform, earning)
	if platform != 1_234 || earning != 11_111 {
		t.Errorf("١٠٪ من ١٢٣٤٥: عمولة %d ومستحقّ %d — والمنتظَر 1234 و11111", platform, earning)
	}
	o := merchantOrderJSON(t, h, m, oid)
	if got := string(o["commission_percent"]); got != "10" {
		t.Errorf("commission_percent = %s — والمنتظَر 10", got)
	}
}
