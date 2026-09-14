package qa

// ══════════════════════════════════════════════════════════════════════
// **العددُ غيرُ المقبول — أيبلغ صاحبَه باسمه؟** (`QI`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **وعقدٌ يُعرَض في الشاشة ولا يبلغه المحرّكُ أبداً عقدٌ ميّت** —
// **ونصٌّ مكتوبٌ لحالٍ لا تقع يُطمئن كاذباً.**
//
// **والقاعدةُ قائمةٌ في `priceItems`**: `qty < 1 || qty > 50`.
// **فالسؤالُ**: **أتصل إلى الشاشة مسمّاةً أم تُجمَع في رمزٍ أصمّ؟**

import (
	"testing"
)

// ═════════════════ QI-01 · QI-02 · QI-03 · QI-04 ═════════════════

// TestQI01_QI04_InvalidQuantityIsNamedThroughTheRealPath **بالطريق
// الحقيقيّ لا بصنع صفٍّ في الذاكرة.**
func TestQI01_QI04_InvalidQuantityIsNamedThroughTheRealPath(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ QI-01")
	it := hh.NewItem(1000)

	// **QI-01 · والعددُ المقبولُ لا يُنذَر عليه.**
	ok := changesOf(t, quoteWith(t, hh,
		[]map[string]any{{"menu_item_id": it.ID, "qty": 2}},
		z.Lat, z.Lng, map[string]any{"lines": map[string]any{it.ID: 1}}))
	if hasChange(ok, "quantity_invalid") != nil {
		t.Fatalf("**عددٌ مقبولٌ قيل إنّه غيرُ مقبول**: %v", ok)
	}

	// **QI-02 · وما جاوز الحدَّ يُقال.**
	cs := changesOf(t, quoteWith(t, hh,
		[]map[string]any{{"menu_item_id": it.ID, "qty": 51}},
		z.Lat, z.Lng, map[string]any{"lines": map[string]any{it.ID: 1000}}))
	ch := hasChange(cs, "quantity_invalid")
	if ch == nil {
		t.Fatalf("**عددٌ غيرُ مقبولٍ رُدّ برمزٍ أصمّ** — "+
			"**وعقدٌ يُعرَض ولا يبلغه المحرّكُ عقدٌ ميّت**: %v", cs)
	}

	// **QI-03 · ويُسمّى الصنفُ ليُعلَّم سطرُه.**
	if ch["menu_item_id"] != it.ID {
		t.Fatalf("**التبدّلُ لا يقول أيُّ صنف**: %v", ch)
	}
	if ch["name"] != it.Name {
		t.Fatalf("**الصنفُ بلا اسمٍ يُعلَّم به سطرُه**: %v", ch)
	}

	// **QI-04 · ويُقال العددُ الذي طلبه** — **لا عددٌ نقترحه عليه.**
	if int(ch["qty"].(float64)) != 51 {
		t.Fatalf("**العددُ المطلوبُ لم يُقَل**: %v", ch)
	}
}

// ═════════════════ QI-05 · QI-09 ═════════════════

// TestQI05_QI09_InvalidQuantityTravelsWithItsNeighbours **ولا يُخفي
// سطرٌ معطوبٌ ما بعده.**
func TestQI05_QI09_InvalidQuantityTravelsWithItsNeighbours(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ QI-05")
	big := hh.NewItem(1000)
	priced := hh.NewItem(2000)

	cs := changesOf(t, quoteWith(t, hh, []map[string]any{
		{"menu_item_id": big.ID, "qty": 99},
		{"menu_item_id": priced.ID, "qty": 1},
	}, z.Lat, z.Lng, map[string]any{
		// **وسعرٌ لا يساوي شيئاً** — **فيقع تبدّلُ سعرٍ قطعاً.**
		"lines": map[string]any{big.ID: 1, priced.ID: 1},
	}))

	qi := hasChange(cs, "quantity_invalid")
	if qi == nil || qi["menu_item_id"] != big.ID {
		t.Fatalf("**العددُ المعطوبُ لم يُسمَّ**: %v", cs)
	}
	// **QI-05 · والسطرُ السليمُ يبقى مرئيّاً بتبدّله.**
	pc := hasChange(cs, "product_price_changed")
	if pc == nil || pc["menu_item_id"] != priced.ID {
		t.Fatalf("**سطرٌ سليمٌ اختفى خلف المعطوب**: %v", cs)
	}
}

// ═════════════════ QI-06 · QI-10 ═════════════════

// TestQI06_QI10_NoClampAndFixingClears **ولا يُقلَّم عددٌ في الخفاء**،
// **ومن صحّحه زال الإنذار.**
//
// **و٥١ لا تصير ٥٠** — **ومن قُلِّم طلبُه صامتاً حصل على غير ما أراد.**
func TestQI06_QI10_NoClampAndFixingClears(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ QI-06")
	it := hh.NewItem(1000)

	bad := quoteWith(t, hh, []map[string]any{{"menu_item_id": it.ID, "qty": 51}},
		z.Lat, z.Lng, map[string]any{"lines": map[string]any{it.ID: 1000}})
	j := bad.JSON()

	// **QI-06 · ولا تُسعَّر سلّةٌ فيها ما لا يُقبَل** — **ومجموعٌ
	// محسوبٌ على خمسين يُقرأ قبولاً للواحدٍ وخمسين.**
	if j["blocked"] != true {
		t.Fatalf("**سلّةٌ فيها عددٌ غيرُ مقبولٍ سُعِّرت**: %v", j)
	}
	if sub, ok := j["subtotal"].(float64); ok && sub != 0 {
		t.Fatalf("**مجموعٌ عُرض لسلّةٍ لا تُقبَل** — **وقد يُقرأ قَبولاً**: %v", sub)
	}
	// **والعددُ المطلوبُ يبقى كما طلبه في الجواب.**
	if qi := hasChange(changesOf(t, bad), "quantity_invalid"); qi == nil ||
		int(qi["qty"].(float64)) != 51 {
		t.Fatalf("**العددُ قُلِّم أو ضاع**: %v", changesOf(t, bad))
	}

	// **QI-10 · ومن صحّحه زال الإنذار.**
	good := changesOf(t, quoteWith(t, hh,
		[]map[string]any{{"menu_item_id": it.ID, "qty": 50}},
		z.Lat, z.Lng, map[string]any{"lines": map[string]any{it.ID: 1000}}))
	if hasChange(good, "quantity_invalid") != nil {
		t.Fatalf("**صُحّح العددُ وبقي الإنذار**: %v", good)
	}
}

// ═════════════════ QI-07 — والمنعُ عند الإنشاء ═════════════════

// TestQI07_CreateStillRejectsInvalidQuantity **والشرحُ للشاشة والمنعُ
// في المحرّك.**
//
// **وعميلٌ معدَّلٌ لا يتخطّى القاعدة** — **والتشخيصُ راحةٌ لا إذن.**
func TestQI07_CreateStillRejectsInvalidQuantity(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ QI-07")
	it := hh.NewItem(1000)
	u := hh.Customer()

	r := hh.POST("/api/v1/orders", u.Token, map[string]any{
		"items":          []map[string]any{{"menu_item_id": it.ID, "qty": 51}},
		"address_text":   "الرقة — شارع الاختبار",
		"lat":            z.Lat,
		"lng":            z.Lng,
		"payment_method": "cash",
	})
	if r.Code < 400 {
		t.Fatalf("**طلبٌ أُنشئ بعددٍ غيرِ مقبول**: %d", r.Code)
	}
	if r.Err() != "bad_qty" {
		t.Fatalf("**رمزٌ غيرُ متوقَّع عند المنع**: %s", r.Err())
	}

	// **وصفرٌ وما دونه كذلك** — **ولا يُقبَل سطرٌ بلا عدد.**
	zero := hh.POST("/api/v1/orders", u.Token, map[string]any{
		"items":          []map[string]any{{"menu_item_id": it.ID, "qty": 0}},
		"address_text":   "الرقة — شارع الاختبار",
		"lat":            z.Lat,
		"lng":            z.Lng,
		"payment_method": "cash",
	})
	if zero.Code < 400 {
		t.Fatalf("**طلبٌ أُنشئ بعددٍ صفر**: %d", zero.Code)
	}
}

// ═════════════════ QI-08 — والعقدُ كما تقرؤه الشاشة ═════════════════

// TestQI08_ContractMatchesWhatTheScreenReads **والشاشةُ تقرأ الحقولَ
// بأسمائها** — **واسمٌ يتبدّل في المحرّك يُفرِغ الشاشةَ صامتا.**
//
// **وهذا يثبّت العقدَ الذي يقرؤه `CartChanges.kt`**: `type` و
// `menu_item_id` و`name` و`qty`.
func TestQI08_ContractMatchesWhatTheScreenReads(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ QI-08")
	it := hh.NewItem(1000)

	cs := changesOf(t, quoteWith(t, hh,
		[]map[string]any{{"menu_item_id": it.ID, "qty": 77}},
		z.Lat, z.Lng, map[string]any{"lines": map[string]any{it.ID: 1000}}))
	ch := hasChange(cs, "quantity_invalid")
	if ch == nil {
		t.Fatalf("**لا تشخيصَ أصلاً**: %v", cs)
	}
	for _, field := range []string{"type", "menu_item_id", "name", "qty"} {
		if _, ok := ch[field]; !ok {
			t.Fatalf("**حقلٌ تقرؤه الشاشةُ غائبٌ من العقد**: %s — %v", field, ch)
		}
	}
	// **والنوعُ نصُّه حرفاً كما في `CartChanges.QUANTITY_INVALID`.**
	if ch["type"] != "quantity_invalid" {
		t.Fatalf("**اسمُ النوع تبدّل** — **فتُفرَغ الشاشةُ صامتة**: %v", ch["type"])
	}
}
