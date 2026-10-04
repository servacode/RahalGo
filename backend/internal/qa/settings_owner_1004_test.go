package qa

// ══════════════════════════════════════════════════════════════════════
// **قسمُ الإعدادات — قراراتُ المالك ٢٠٢٦-١٠-٠٤** عبر الباب الحقيقيّ
// ══════════════════════════════════════════════════════════════════════

import (
	"net/http"
	"testing"
)

func settingStamp(t *testing.T, h *Harness, key string) string {
	t.Helper()
	var at string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT updated_at::text || '|' || value::text FROM app_settings WHERE key = $1`, key).Scan(&at); err != nil {
		t.Fatalf("قراءةُ الإعداد: %v", err)
	}
	return at
}

// **لا حفظَ بلا تغيير** (البند ١٤) — كان يُكتب في السجلّ «١٠ ← ١٠» وتُطلب
// كلمةُ السرّ لكلّ ضغطةِ سهم.
func TestSETTINGS_UnchangedValueNotSaved(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	const key = "orders.delivery_estimate_min"
	h.Setting(key, "17")
	before := settingStamp(t, h, key)
	res := h.Call("PUT", "/api/v1/admin/settings/"+key, admin.Token, map[string]any{"value": 17}, nil)
	if res.Code != http.StatusOK || res.JSON()["updated"] != false {
		t.Fatalf("القيمةُ نفسُها ⇒ %s — والمنتظَرُ «لم يتغيّر»", res)
	}
	if after := settingStamp(t, h, key); after != before {
		t.Fatalf("**كُتب الإعدادُ بلا تغيير** — %s ⇒ %s", before, after)
	}
	res = h.Call("PUT", "/api/v1/admin/settings/"+key, admin.Token, map[string]any{"value": 18}, nil)
	if res.Code != http.StatusOK || res.JSON()["updated"] != true {
		t.Fatalf("قيمةٌ جديدة ⇒ %s", res)
	}
	if after := settingStamp(t, h, key); after == before {
		t.Fatal("التغييرُ لم يُحفظ")
	}
}

// **المفاتيحُ المرتبطةُ تُفحص في الخادم** (البند ١٦) — بسببٍ يُسمّي الطرفَ الآخر.
func TestSETTINGS_RelatedKeysRejectedWithReason(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	h.Setting("delivery.custom_fee_min", "1000")
	h.Setting("delivery.custom_fee_max", "100000")
	res := h.Call("PUT", "/api/v1/admin/settings/delivery.custom_fee_min", admin.Token,
		map[string]any{"value": 200000}, nil)
	if res.Code != http.StatusBadRequest || res.Err() != "setting_conflict" {
		t.Fatalf("الأدنى فوق الأعلى ⇒ %s — والمنتظَرُ setting_conflict", res)
	}
	h.Setting("orders.accept_timeout_min", "5")
	res = h.Call("PUT", "/api/v1/admin/settings/orders.auto_accept_min", admin.Token,
		map[string]any{"value": 3}, nil)
	if res.Code != http.StatusBadRequest || res.Err() != "setting_conflict" {
		t.Fatalf("القبولُ التلقائيُّ قبل التنبيه ⇒ %s", res)
	}
}

// **قالبُ الرمز لا يُحفظ بلا `{code}`** — وإلّا وقف الدخولُ بالرمز للجميع.
func TestSETTINGS_OTPTemplateNeedsCode(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	h.Setting("whatsapp.otp_template", `"رمز التحقق: {code}"`)
	res := h.Call("PUT", "/api/v1/admin/settings/whatsapp.otp_template", admin.Token,
		map[string]any{"value": "رمز التحقق"}, nil)
	if res.Code != http.StatusBadRequest || res.Err() != "setting_placeholder_missing" {
		t.Fatalf("قالبٌ بلا رمز ⇒ %s", res)
	}
	res = h.Call("PUT", "/api/v1/admin/settings/accounts.welcome_template", admin.Token,
		map[string]any{"value": "أهلاً {password}"}, nil)
	if res.Code != http.StatusBadRequest || res.Err() != "setting_placeholder_missing" {
		t.Fatalf("ترحيبٌ بلا رابط ⇒ %s", res)
	}
}

// **الشارةُ من قائمة الخطورة** — ومعها موضوعُ العمود الجانبيّ.
func TestSETTINGS_ListCarriesRiskAndTopic(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	res := h.GET("/api/v1/admin/settings", admin.Token)
	if res.Code != http.StatusOK {
		t.Fatalf("القائمة: %s", res)
	}
	rows, _ := res.JSON()["settings"].([]any)
	found := map[string]map[string]any{}
	for _, r := range rows {
		m, _ := r.(map[string]any)
		k, _ := m["key"].(string)
		found[k] = m
	}
	p := found["delivery.platform_percent"]
	if p == nil || p["risk"] != "money" || p["sensitive"] != true || p["topic"] != "money" {
		t.Fatalf("حصّةُ المنصّة ⇒ %v", p)
	}
	if s := found["security.session_days"]; s == nil || s["risk"] != "security" || s["sensitive"] != true {
		t.Fatalf("مدّةُ الجلسة ⇒ %v", s)
	}
	// **والقسمُ الفرعيُّ القديمُ باقٍ** — حقلان باسمٍ واحدٍ كانا يُسقطانه.
	if s := found["platform.address"]; s == nil || s["section"] != "page.contact" {
		t.Fatalf("عنوانُ المكتب فقد قسمَه: %v", s)
	}
}

// **حصّةُ المنصّة تُقتطع من أجر السائق في الطلب العاديّ وتُلقَط** (البند ١).
func TestSETTINGS_PlatformShareAppliesToStandardOrder(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("delivery.fee", "10000")
	h.Setting("delivery.platform_percent", "10")
	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemFor(m, 5000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("pp"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	var fee, drv, snap int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT delivery_fee, driver_fee, snap_platform_delivery_percent
		  FROM orders WHERE id = $1::uuid`, oid).Scan(&fee, &drv, &snap); err != nil {
		t.Fatal(err)
	}
	if snap != 10 || drv != fee-fee*10/100 {
		t.Fatalf("أجرةٌ %d وأجرُ السائق %d واللقطة %d — والمنتظَرُ ١٠٪ للمنصّة", fee, drv, snap)
	}
	// **ورفعُها بعد الإنشاء لا يمسّ الطلب.**
	h.Setting("delivery.platform_percent", "30")
	var after int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT driver_fee FROM orders WHERE id = $1::uuid`, oid).Scan(&after)
	if after != drv {
		t.Fatalf("تبدّل أجرُ طلبٍ قائم: %d ⇒ %d", drv, after)
	}
}

// **أجرةُ «لدي توصيلة» من `delivery.fee` لا من عمود المنطقة** (البند ٢).
func TestSETTINGS_MerchantDeliveryFeeFromGeneralFee(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ الإعدادات ١٠٠٤")
	fundWallet(t, h, m.admin, m.fx.M.Owner.ID, 5000)
	// **ولوحةُ المناطق كانت تكتب العمودَ صفراً** — فيُصفَّر عمداً.
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE delivery_zones SET delivery_fee = 0`); err != nil {
		t.Fatal(err)
	}
	id := m.create(t, h, "merchant", uniq("mdfee"))
	var fee, drv int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT delivery_fee, driver_fee FROM orders WHERE id = $1::uuid`, id).Scan(&fee, &drv); err != nil {
		t.Fatal(err)
	}
	if fee != 500 || drv != 450 {
		t.Fatalf("أجرةٌ %d وأجرُ السائق %d — والمنتظَرُ ٥٠٠ و٤٥٠ (من الأجرة العامّة)", fee, drv)
	}
}

// **المثالُ الحيُّ يحسبه المحرّك** (البند ١٧) — بالدوالّ التي تجري على الطلب.
func TestSETTINGS_MoneyExampleFromEngine(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	h.Setting("delivery.fee", "10000")
	h.Setting("pricing.margin_fixed", "1000")
	h.Setting("merchants.commission_percent", "10")
	h.Setting("sales.commission_percent", "10")
	h.Setting("delivery.platform_percent", "10")
	res := h.GET("/api/v1/admin/settings/money-example?amount=50000", admin.Token)
	if res.Code != http.StatusOK {
		t.Fatalf("المثال: %s", res)
	}
	j := res.JSON()
	want := map[string]float64{
		"customer_pays": 61000, "merchant_gets": 45000, "driver_gets": 9000,
		"rep_gets": 600, "platform_gets": 6400,
	}
	for k, v := range want {
		if got, _ := j[k].(float64); got != v {
			t.Errorf("%s = %v والمنتظَرُ %v", k, j[k], v)
		}
	}
}
