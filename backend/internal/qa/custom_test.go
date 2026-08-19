package qa

// **الطبقةُ الثامنة — الطلبُ الخاصُّ والفشلُ والإرجاع.**
//
// المعرّفات: `CUST-*` · `FAIL-*` · الوسم: `@api @critical @release`
//
// # ولماذا الطلبُ الخاصُّ حزمةٌ وحدَه
//
// **لا متجرَ فيه ولا سعرَ عند الإنشاء** — الزبونُ يكتب ما يريد،
// **والسائقُ يشتريه ويتّفقان على المبلغ بعدها.** فالمسارُ غيرُ مسار
// الطلب العاديّ في كلّ خطوة.
//
// **وفيه وقع BUG-002**: بابُه كان بلا حمايةٍ من التكرار **بينما أخوه
// محميّ.**
//
// # وحقلُه `request` لا `description`
//
// **وهذا ما جعل `IDEM-005` تُتخطّى** حتّى ٢٠٢٦-٠٨-١٩: كان الاختبارُ
// يرسل اسماً مخترَعاً فيُردّ، **فيُقرأ «تعذّر التجهيز» لا «الاسمُ
// خطأ».** وهي عائلةُ العقد نفسُها — انظر `CONTRACT-*`.

import (
	"testing"
)

func customBody(text string) map[string]any {
	return map[string]any{
		"request":        text,
		"address_text":   "الرقة — شارع الاختبار",
		"lat":            35.9506,
		"lng":            39.0094,
		"payment_method": "cash",
	}
}

// TestCUST_001_Create **الطلبُ الخاصُّ يُنشأ.**
func TestCUST_001_Create(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	got := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("k"),
		customBody("كيلو بندورة وخبز من أيّ محلّ"))
	if got.Code >= 400 {
		t.Fatalf("CUST-001 تعذّر إنشاءُ طلبٍ خاصّ: %s", got)
	}
	oid, _ := got.JSON()["id"].(string)
	if oid == "" {
		t.Fatalf("CUST-001 الردُّ بلا معرّف: %s", got)
	}
	var kind, req string
	if err := h.Pool.QueryRow(t.Context(),
		`SELECT kind, custom_request FROM orders WHERE id = $1::uuid`, oid).
		Scan(&kind, &req); err != nil {
		t.Fatalf("CUST-001 تعذّرت القراءة: %v", err)
	}
	if kind != "custom" {
		t.Errorf("CUST-001 النوعُ %q — يُنتظر custom", kind)
	}
	// **ونصُّ الطلب يُحفظ كما كُتب** — **ونصٌّ يضيع يجعل السائقَ يشتري
	// ما لم يُطلب.**
	if req == "" {
		t.Error("CUST-001 **نصُّ الطلب ضاع** — والسائقُ لا يعرف ما يشتري")
	}
}

// TestCUST_002_EmptyRequestRejected **وطلبٌ خاصٌّ بلا نصٍّ يُردّ.**
//
// **وطلبٌ فارغٌ يصل سائقاً يقف حائراً** — ولا أحدَ يعرف ما يُشترى.
func TestCUST_002_EmptyRequestRejected(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	for _, txt := range []string{"", "   "} {
		got := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("k"), customBody(txt))
		if got.Code < 400 {
			t.Errorf("CUST-002 طلبٌ خاصٌّ بنصٍّ %q قُبل: %s", txt, got)
		}
	}
}

// TestCUST_003_Idempotent **ومفتاحٌ واحدٌ = طلبٌ خاصٌّ واحد** —
// (BUG-002، وهذه هي `IDEM-005` بعد أن صحّ حقلُها).
func TestCUST_003_Idempotent(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	key := uniq("idem")
	body := customBody("طلبٌ خاصٌّ للاختبار الآليّ")

	before := h.CountOrders(cust.ID)
	a := h.POSTKey("/api/v1/orders/custom", cust.Token, key, body)
	if a.Code >= 400 {
		t.Fatalf("CUST-003 النداءُ الأوّلُ رُدّ: %s", a)
	}
	b := h.POSTKey("/api/v1/orders/custom", cust.Token, key, body)
	if b.Code >= 400 {
		t.Fatalf("CUST-003 الإعادةُ بالمفتاح نفسِه رُدَّت: %s", b)
	}
	if got := h.CountOrders(cust.ID) - before; got != 1 {
		t.Errorf("CUST-003 **الطلبُ الخاصُّ تكرّر**: %d — يُنتظر 1", got)
	}
	if a.JSON()["id"] != b.JSON()["id"] {
		t.Errorf("CUST-003 الردّان لطلبين: %v ≠ %v", a.JSON()["id"], b.JSON()["id"])
	}
}

// TestCUST_010_NoPriceBeforeAgreement **ولا مبلغَ قبل الاتّفاق.**
//
// **والطلبُ الخاصُّ يُنشأ بلا سعر** — والسائقُ يشتري ثمّ يُدخل ما دفع.
// **وإجماليٌّ صفرٌ يُعرض «مجّانا»** إن لم تفرّق الشاشةُ بين «صفر»
// و«لم يُتّفق بعد».
func TestCUST_010_NoPriceBeforeAgreement(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	got := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("k"),
		customBody("أدويةٌ من الصيدليّة"))
	if got.Code >= 400 {
		t.Fatalf("CUST-010 تعذّر التجهيز: %s", got)
	}
	oid, _ := got.JSON()["id"].(string)

	var goods, fee *int64
	if err := h.Pool.QueryRow(t.Context(),
		`SELECT custom_goods_amount, custom_fee FROM orders WHERE id = $1::uuid`, oid).
		Scan(&goods, &fee); err != nil {
		t.Fatalf("CUST-010 تعذّرت القراءة: %v", err)
	}
	// **و`NULL` لا صفر** — **وصفرٌ يُقرأ سعراً متّفقاً عليه.**
	if goods != nil {
		t.Errorf("CUST-010 مبلغُ البضاعة %d قبل الاتّفاق — يُنتظر NULL", *goods)
	}
	if fee != nil {
		t.Errorf("CUST-010 الأجرةُ %d قبل الاتّفاق — يُنتظر NULL", *fee)
	}
}

// TestCUST_020_ForeignCannotRead **وطلبُ غيرِه الخاصُّ لا يُقرأ.**
//
// **والخاصُّ فيه نصٌّ يكتبه صاحبُه** — دواءٌ أو حاجةٌ شخصيّة، **وقراءتُه
// تسريبٌ لا مجرّدَ خللِ تخويل.**
func TestCUST_020_ForeignCannotRead(t *testing.T) {
	h := New(t)
	victim, attacker := h.Customer(), h.Customer()
	made := h.POSTKey("/api/v1/orders/custom", victim.Token, uniq("k"),
		customBody("دواءٌ من الصيدليّة — خاصّ"))
	if made.Code >= 400 {
		t.Fatalf("CUST-020 تعذّر التجهيز: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	got := h.GET("/api/v1/my/orders/"+oid, attacker.Token)
	if got.Code < 400 {
		t.Errorf("CUST-020 **قُرئ طلبٌ خاصٌّ لغيرِه**: %s", got)
	}
}

// --- الفشلُ والإرجاع -------------------------------------------------

// TestFAIL_001_DriverFailsDelivery **والسائقُ يُعلن فشلَ التسليم.**
//
// **والفشلُ واقعٌ يوميّ**: لا أحدَ في البيت، أو رفض الزبونُ الاستلام.
// **وطلبٌ لا نهايةَ له يبقى في الطابور إلى الأبد.**
func TestFAIL_001_DriverFailsDelivery(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("FAIL-001 تعذّر التجهيز: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to}); got.Code >= 400 {
			t.Fatalf("FAIL-001 الانتقالُ إلى %s رُدّ: %s", to, got)
		}
	}

	// ══════════════════════════════════════════════════════════════════
	// **والسببُ رمزٌ من قائمةٍ لا نصٌّ حرّ**
	// ══════════════════════════════════════════════════════════════════
	//
	// **ونصٌّ حرٌّ لا يُعدّ ولا يُقاس** — فلا يُعرف كم مرّةً لم يفتح
	// أحدٌ البابَ ولا على من تُحسب الأجرة. (قاعدةٌ قائمةٌ في المحرّك،
	// كشفها هذا الاختبارُ بـ`bad_fail_reason`.)
	//
	// **ويُقرأ الرمزُ من القائمة الحيّة لا يُكتب هنا** — **ورمزٌ مكتوبٌ
	// بيدٍ يشيخ يومَ تتبدّل القائمة**، فيسقط الاختبارُ على تغييرٍ سليم.
	code := firstFailCode(t, h, drv, "at_dropoff")

	got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "failed", "reason": code, "note": "لا أحدَ في العنوان"})
	if got.Code >= 400 {
		t.Fatalf("FAIL-001 إعلانُ الفشل رُدّ: %s", got)
	}
	if st := h.statusOf(oid); st != "failed" {
		t.Errorf("FAIL-001 الحالُ %q — يُنتظر failed", st)
	}
}

// TestFAIL_002_ReasonIsRecorded **وسببُ الفشل يُكتب.**
//
// **وفشلٌ بلا سببٍ يجعل الحكمَ بين زبونٍ وسائقٍ كلمةً ضدّ كلمة** — ولا
// يُعرف على من تُحسب الأجرة.
func TestFAIL_002_ReasonIsRecorded(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to})
	}
	code := firstFailCode(t, h, drv, "at_dropoff")
	if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "failed", "reason": code,
			"note": "رفض الزبونُ الاستلام"}); got.Code >= 400 {
		t.Fatalf("FAIL-002 إعلانُ الفشل رُدّ: %s", got)
	}
	var stored string
	if err := h.Pool.QueryRow(t.Context(),
		`SELECT coalesce(fail_reason, '') FROM orders WHERE id = $1::uuid`, oid).
		Scan(&stored); err != nil {
		t.Fatalf("FAIL-002 تعذّرت القراءة: %v", err)
	}
	if stored == "" {
		t.Error("FAIL-002 **سقط السبب** — والحكمُ بين اثنين بلا شاهد")
	}
}

// TestFAIL_010_NoMoveAfterFailed **والفاشلُ لا يُحرَّك.**
func TestFAIL_010_NoMoveAfterFailed(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	if _, err := h.Pool.Exec(t.Context(),
		`UPDATE orders SET status = 'failed' WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تعذّرت زراعةُ الحال: %v", err)
	}
	for _, to := range []string{"delivered", "on_the_way", "pending"} {
		got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to})
		if got.Code < 400 {
			t.Errorf("FAIL-010 حُرّك طلبٌ فاشلٌ إلى %s: %s", to, got)
		}
	}
	if st := h.statusOf(oid); st != "failed" {
		t.Errorf("FAIL-010 تغيّرت حالُ الفاشل إلى %q", st)
	}
}

// firstFailCode **أوّلُ رمزِ فشلٍ تعرفه المنصّة** — يُقرأ من بابِه.
func firstFailCode(t *testing.T, h *Harness, drv *User, at string) string {
	t.Helper()
	// **والأسبابُ مرشَّحةٌ بالحال** (`?at=`) — **ونداءٌ بلا مُرشِّحٍ يردّ
	// قائمةً فارغةً لا خطأ**، فيُقرأ «لا أسبابَ» وهي موجودة.
	res := h.GET("/api/v1/driver/fail-reasons?at="+at, drv.Token)
	if res.Code >= 400 {
		t.Fatalf("FAIL: أسبابُ الفشل لا تُقرأ: %s", res)
	}
	list, _ := res.JSON()["reasons"].([]any)
	if len(list) == 0 {
		t.Skip("FAIL: لا أسبابَ مضبوطةٌ في هذه القاعدة — يُتخطّى")
	}
	first, _ := list[0].(map[string]any)
	code, _ := first["code"].(string)
	if code == "" {
		t.Fatalf("FAIL: سببٌ بلا رمز: %v", first)
	}
	return code
}
