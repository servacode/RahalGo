package qa

// **منعُ التكرار: بصمةُ الجسم** — `CAF-02` · `13-029` (٢٠٢٦-٠٩-٢١).
//
// # لماذا فوق ما سبق
//
// **`IDEM-*` تحرس «مفتاحٌ واحدٌ = طلبٌ واحد»** حين يتطابق الجسم. **لكنّ
// مفتاحاً واحداً لجسمين مختلفين كان يمرّ**: الأوّلُ يُنشئ، **والثاني —
// سلّةٌ عُدّلت والمفتاحُ نفسُه — كان يُرَدّ عليه ردُّ الأوّل** (`writeReplay`)
// كأنّ تعديلَه وقع، **وهو لم يقع.** أو، لو دُوِّر الجسمُ دون المفتاح،
// نُفِّذ ما لا يُقصَد على ردِّ ما يُقصَد.
//
// # ما تحرسه هذه الحزمة
//
//   - **مفتاحٌ + جسمٌ مختلف** ⇒ `409 idempotency_key_reused`، **صفرُ تنفيذٍ ثانٍ.**
//   - **مفتاحٌ + جسمٌ أُعيد ترتيبُه** ⇒ بصمةٌ واحدة ⇒ إعادةٌ لا رفض.
//   - **صفٌّ بلا بصمة (تركةٌ)** ⇒ يُعامَل كما كان ⇒ إعادةٌ لا رفض.
//   - **والخاصُّ كالعاديّ** — بابان لفعلٍ واحد.
//
// **ونفيُ الحارس**: بلا فرع الرفض في `acquireClaim` يُعاد ردُّ الأوّل على
// الجسم المختلف — فيسقط `wantReused` (يصير الرمزُ إعادةً لا `409`).

import (
	"testing"
)

// reusedCode رمزُ خطأِ الردّ — **أو فارغٌ إن لم يكن خطأً.**
func reusedCode(r Res) string {
	env, _ := r.JSON()["error"].(map[string]any)
	code, _ := env["code"].(string)
	return code
}

// TestCAF02_SameKeyDifferentBodyRejected **مفتاحٌ لجسمين ⇒ رفضٌ لا إعادة.**
func TestCAF02_SameKeyDifferentBodyRejected(t *testing.T) {
	h := New(t)
	u := h.Customer()
	item := h.NewItem(1500)
	key := uniq("caf02")

	before := h.CountOrders(u.ID)
	first := h.POSTKey("/api/v1/orders", u.Token, key, orderBody(item, 1))
	if first.Code >= 400 {
		t.Fatalf("CAF-02 تعذّر النداءُ الأوّل: %s", first)
	}

	// **المفتاحُ نفسُه، الكمّيّةُ تبدّلت** — جسمٌ آخر.
	second := h.POSTKey("/api/v1/orders", u.Token, key, orderBody(item, 3))
	if second.Code != 409 {
		t.Fatalf("CAF-02 **جسمٌ مختلفٌ بالمفتاح نفسِه لم يُرَدّ** — رمز=%d (يُنتظر 409): %s",
			second.Code, second)
	}
	if c := reusedCode(second); c != "idempotency_key_reused" {
		t.Errorf("CAF-02 رمزُ الرفض %q — يُنتظر idempotency_key_reused", c)
	}
	if got := h.CountOrders(u.ID) - before; got != 1 {
		t.Errorf("CAF-02 **نُفِّذ طلبٌ ثانٍ** — أُنشئ %d (يُنتظر 1)", got)
	}
}

// TestCAF02_ReorderedItemsReplayNotReused **إعادةُ ترتيب السلّة ليست طلباً آخر.**
func TestCAF02_ReorderedItemsReplayNotReused(t *testing.T) {
	h := New(t)
	u := h.Customer()
	m := h.Factory().Merchant()
	a := h.NewItemFor(m, 1000)
	b := h.NewItemFor(m, 1500)
	key := uniq("caf02")

	body := func(items ...map[string]any) map[string]any {
		return map[string]any{
			"items":          items,
			"address_text":   "الرقة — شارع الاختبار",
			"lat":            35.9506,
			"lng":            39.0094,
			"payment_method": "cash",
		}
	}
	itemA := map[string]any{"menu_item_id": a.ID, "qty": 1}
	itemB := map[string]any{"menu_item_id": b.ID, "qty": 2}

	before := h.CountOrders(u.ID)
	first := h.POSTKey("/api/v1/orders", u.Token, key, body(itemA, itemB))
	if first.Code >= 400 {
		t.Fatalf("CAF-02 تعذّر النداءُ الأوّل (صنفان): %s", first)
	}

	// **الأصنافُ نفسُها، ترتيبُها معكوس** — بصمةٌ واحدة.
	second := h.POSTKey("/api/v1/orders", u.Token, key, body(itemB, itemA))
	if second.Code == 409 {
		t.Fatalf("CAF-02 **إعادةُ ترتيبِ السلّة عُدّت جسماً آخر** — رُفضت %s", second)
	}
	if second.Code >= 400 {
		t.Fatalf("CAF-02 الإعادةُ المُرتّبةُ رُدّت: %s", second)
	}
	if first.JSON()["id"] != second.JSON()["id"] {
		t.Errorf("CAF-02 المُرتّبةُ ردّت طلباً آخر: %v ≠ %v",
			first.JSON()["id"], second.JSON()["id"])
	}
	if got := h.CountOrders(u.ID) - before; got != 1 {
		t.Errorf("CAF-02 إعادةُ الترتيب أنشأت %d طلبا — يُنتظر 1", got)
	}
}

// TestCAF02_LegacyNullFingerprintReplays **صفٌّ بلا بصمة يُعامَل كما كان.**
//
// **صفوفُ ما قبل الهجرة بلا بصمة** — **لا حكمَ عليها**: تُعاد كما كانت
// (ردُّ الأوّل)، **ولا تُرفَض** ولو اختلف الجسم. يُحاكى بمحو بصمةِ صفٍّ
// مثبَّتٍ ثمّ نداءٍ بجسمٍ مختلف.
func TestCAF02_LegacyNullFingerprintReplays(t *testing.T) {
	h := New(t)
	u := h.Customer()
	item := h.NewItem(1500)
	key := uniq("caf02")

	before := h.CountOrders(u.ID)
	first := h.POSTKey("/api/v1/orders", u.Token, key, orderBody(item, 1))
	if first.Code >= 400 {
		t.Fatalf("CAF-02 تعذّر النداءُ الأوّل: %s", first)
	}

	// **تُمحى البصمةُ** — كأنّ الصفَّ من قبل الهجرة.
	if _, err := h.Pool.Exec(t.Context(),
		`UPDATE idempotency_keys SET request_fingerprint = NULL
		  WHERE user_id = $1::uuid AND key = $2`, u.ID, key); err != nil {
		t.Fatalf("CAF-02 تعذّر محوُ البصمة: %v", err)
	}

	// **جسمٌ مختلفٌ على صفٍّ بلا بصمة** ⇒ إعادةٌ لا رفض.
	second := h.POSTKey("/api/v1/orders", u.Token, key, orderBody(item, 9))
	if second.Code == 409 {
		t.Fatalf("CAF-02 **صفٌّ بلا بصمةٍ رُفض** — يجب أن يُعامَل كتركةٍ ويُعاد: %s", second)
	}
	if second.Code >= 400 {
		t.Fatalf("CAF-02 صفُّ التركةِ رُدّ: %s", second)
	}
	if first.JSON()["id"] != second.JSON()["id"] {
		t.Errorf("CAF-02 التركةُ لم تُعِد الطلبَ الأوّل: %v ≠ %v",
			first.JSON()["id"], second.JSON()["id"])
	}
	if got := h.CountOrders(u.ID) - before; got != 1 {
		t.Errorf("CAF-02 التركةُ أنشأت %d طلبا — يُنتظر 1", got)
	}
}

// TestCAF02_CustomSameKeyDifferentRequestRejected **والخاصُّ كالعاديّ.**
func TestCAF02_CustomSameKeyDifferentRequestRejected(t *testing.T) {
	h := New(t)
	u := h.Customer()
	key := uniq("caf02c")

	before := h.CountOrders(u.ID)
	first := h.POSTKey("/api/v1/orders/custom", u.Token, key,
		customBody("كيلو بندورة وخبز"))
	if first.Code >= 400 {
		t.Fatalf("CAF-02 تعذّر الطلبُ الخاصُّ الأوّل: %s", first)
	}

	// **الوصفُ تبدّل والمفتاحُ نفسُه** — جسمٌ آخر.
	second := h.POSTKey("/api/v1/orders/custom", u.Token, key,
		customBody("كيلو لحم وأرز — طلبٌ مختلفٌ تماما"))
	if second.Code != 409 {
		t.Fatalf("CAF-02 **وصفٌ مختلفٌ بالمفتاح نفسِه لم يُرَدّ** — رمز=%d (يُنتظر 409): %s",
			second.Code, second)
	}
	if c := reusedCode(second); c != "idempotency_key_reused" {
		t.Errorf("CAF-02 رمزُ رفض الخاصّ %q — يُنتظر idempotency_key_reused", c)
	}
	if got := h.CountOrders(u.ID) - before; got != 1 {
		t.Errorf("CAF-02 **نُفِّذ طلبٌ خاصٌّ ثانٍ** — أُنشئ %d (يُنتظر 1)", got)
	}
}

// TestCAF02_CustomSameRequestReplays **والخاصُّ يُعيد على التطابق لا يرفض.**
func TestCAF02_CustomSameRequestReplays(t *testing.T) {
	h := New(t)
	u := h.Customer()
	key := uniq("caf02c")
	body := customBody("كيلو بندورة وخبز")

	before := h.CountOrders(u.ID)
	first := h.POSTKey("/api/v1/orders/custom", u.Token, key, body)
	if first.Code >= 400 {
		t.Fatalf("CAF-02 تعذّر الطلبُ الخاصُّ الأوّل: %s", first)
	}
	second := h.POSTKey("/api/v1/orders/custom", u.Token, key, body)
	if second.Code >= 400 {
		t.Fatalf("CAF-02 إعادةُ الخاصّ بالجسم نفسِه رُدّت: %s", second)
	}
	if first.JSON()["id"] != second.JSON()["id"] {
		t.Errorf("CAF-02 إعادةُ الخاصّ ردّت طلباً آخر: %v ≠ %v",
			first.JSON()["id"], second.JSON()["id"])
	}
	if got := h.CountOrders(u.ID) - before; got != 1 {
		t.Errorf("CAF-02 إعادةُ الخاصّ أنشأت %d طلبا — يُنتظر 1", got)
	}
}
