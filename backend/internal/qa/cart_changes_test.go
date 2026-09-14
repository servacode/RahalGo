package qa

// ══════════════════════════════════════════════════════════════════════
// **ما تبدّل يُقال باسمه — لا «حدث خطأ»** (`CC`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **والمحرّكُ يعرف أيَّ صنفٍ نفد وأيَّ سعرٍ تبدّل** — **فلا يُردّ رمزٌ
// واحدٌ لا يقول أيَّها.** **ومن مُحيت سلّتُه لأنّ صنفاً نفد خسر عشرةَ
// اختيارات.**

import (
	"net/http"
	"testing"
)

// quoteWith تسعيرةٌ تحمل ما كان معروضاً على الشاشة.
func quoteWith(t *testing.T, h *Harness, items []map[string]any,
	lat, lng float64, expected map[string]any) Res {
	t.Helper()
	body := map[string]any{"items": items, "lat": lat, "lng": lng}
	if expected != nil {
		body["expected"] = expected
	}
	return h.POST("/api/v1/public/quote", "", body)
}

// changesOf يقرأ التبدّلات من ردّ التسعيرة.
func changesOf(t *testing.T, r Res) []map[string]any {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("**التسعيرةُ رُدّت**: %d / %s", r.Code, r.Err())
	}
	raw, _ := r.JSON()["changes"].([]any)
	out := []map[string]any{}
	for _, c := range raw {
		if m, ok := c.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// hasChange يبحث عن تبدّلٍ بنوعه.
func hasChange(cs []map[string]any, kind string) map[string]any {
	for _, c := range cs {
		if c["type"] == kind {
			return c
		}
	}
	return nil
}

// ═════════════════ CC-04 · CC-05 — الصنفُ يُسمّى ═════════════════

// TestCC04_CC05_RemovedAndDisabledItemsAreNamed **ولا يُمحى سطرٌ بصمت.**
func TestCC04_CC05_RemovedAndDisabledItemsAreNamed(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ CC-04")
	good := hh.NewItem(1000)
	gone := hh.NewItem(1000)
	off := hh.NewItem(1000)

	// **صنفٌ يُمحى** — **ومعرّفُه باقٍ في سلّةٍ فُتحت قبل محوه.**
	if _, err := hh.Pool.Exec(ctxBG(),
		`DELETE FROM menu_items WHERE id = $1`, gone.ID); err != nil {
		t.Fatalf("محوُ الصنف: %v", err)
	}
	// **وصنفٌ يُوقَف.**
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE menu_items SET available = false WHERE id = $1`, off.ID); err != nil {
		t.Fatalf("إيقافُ الصنف: %v", err)
	}

	items := []map[string]any{
		{"menu_item_id": good.ID, "qty": 1},
		{"menu_item_id": gone.ID, "qty": 1},
		{"menu_item_id": off.ID, "qty": 2},
	}
	cs := changesOf(t, quoteWith(t, hh, items, z.Lat, z.Lng, map[string]any{
		"lines": map[string]any{good.ID: 1000, gone.ID: 1000, off.ID: 1000},
	}))

	// **CC-04 · المحذوفُ يُسمّى بمعرّفه.**
	rm := hasChange(cs, "product_removed")
	if rm == nil {
		t.Fatalf("**صنفٌ محذوفٌ لم يُقَل** — **فتُمحى السلّةُ أو يُقال «حدث خطأ»**: %v", cs)
	}
	if rm["menu_item_id"] != gone.ID {
		t.Fatalf("**التبدّلُ لا يقول أيُّ صنف**: %v", rm)
	}

	// **CC-05 · والموقوفُ باسمه وعدده.**
	un := hasChange(cs, "product_unavailable")
	if un == nil {
		t.Fatalf("**صنفٌ موقوفٌ لم يُقَل**: %v", cs)
	}
	if un["menu_item_id"] != off.ID {
		t.Fatalf("**التبدّلُ لا يقول أيُّ صنف**: %v", un)
	}
	if un["name"] != off.Name {
		t.Fatalf("**الصنفُ بلا اسمٍ يُعلَّم به سطرُه**: %v", un)
	}
	if int(un["qty"].(float64)) != 2 {
		t.Fatalf("**العددُ لم يُقَل**: %v", un)
	}

	// **ولا يُخترَع تبدّلٌ لما لم يتبدّل** — **والسليمُ يصمت.**
	for _, c := range cs {
		if c["menu_item_id"] == good.ID {
			t.Fatalf("**صنفٌ سليمٌ قيل إنّه تبدّل**: %v", c)
		}
	}
}

// ═════════════════ CC-07 · CC-08 — السعرُ صعوداً ونزولاً ═════════════

// TestCC07_CC08_PriceChangeIsSurfacedBothWays **ولا يُستبدَل رقمٌ بصمت.**
//
// **ومن قرأ سعراً ودُفع غيرُه ظنّ أنّه خُدع** — **والنزولُ يُقال كما
// يُقال الصعود.**
func TestCC07_CC08_PriceChangeIsSurfacedBothWays(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ CC-07")
	it := hh.NewItem(1000)
	items := []map[string]any{{"menu_item_id": it.ID, "qty": 1}}

	// **ما يقوله المحرّكُ الآن** — **لا ما نظنّه.**
	plain := quoteWith(t, hh, items, z.Lat, z.Lng, nil)
	if len(changesOf(t, plain)) != 0 {
		t.Fatalf("**تبدّلاتٌ بلا مُتوقَّع**")
	}
	now := int64(plain.JSON()["subtotal"].(float64))

	// **CC-07 · صعودٌ** — **ما عُرض أقلُّ ممّا يقوله المحرّك.**
	up := hasChange(changesOf(t, quoteWith(t, hh, items, z.Lat, z.Lng, map[string]any{
		"lines": map[string]any{it.ID: now - 500},
	})), "product_price_changed")
	if up == nil {
		t.Fatalf("**ارتفاعُ سعرٍ لم يُقَل** — **فيُدفَع غيرُ ما رُئي**")
	}
	if int64(up["old_value"].(float64)) != now-500 || int64(up["new_value"].(float64)) != now {
		t.Fatalf("**الرقمان لم يُقالا معاً**: %v", up)
	}
	if up["name"] != it.Name {
		t.Fatalf("**السطرُ لا يُعرَف باسمه**: %v", up)
	}

	// **CC-08 · ونزولٌ** — **ولا يُبتلَع الفرقُ صامتاً.**
	if hasChange(changesOf(t, quoteWith(t, hh, items, z.Lat, z.Lng, map[string]any{
		"lines": map[string]any{it.ID: now + 500},
	})), "product_price_changed") == nil {
		t.Fatalf("**انخفاضُ سعرٍ لم يُقَل**")
	}

	// **CC-20 · والمطابقُ يصمت** — **وإنذارٌ بلا تبدّلٍ يُعلَّم عليه.**
	same := changesOf(t, quoteWith(t, hh, items, z.Lat, z.Lng, map[string]any{
		"lines": map[string]any{it.ID: now},
	}))
	if hasChange(same, "product_price_changed") != nil {
		t.Fatalf("**سعرٌ لم يتبدّل قيل إنّه تبدّل**: %v", same)
	}
}

// ═════════════════ CC-10 — أجورُ التوصيل ═════════════════

// TestCC10_DeliveryFeeChangeIsSurfaced **وتبدّلُ الأجور يُقال قبل الإتمام.**
func TestCC10_DeliveryFeeChangeIsSurfaced(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ CC-10")
	it := hh.NewItem(1000)
	items := []map[string]any{{"menu_item_id": it.ID, "qty": 1}}

	fee := int64(quoteWith(t, hh, items, z.Lat, z.Lng, nil).
		JSON()["delivery_fee"].(float64))

	ch := hasChange(changesOf(t, quoteWith(t, hh, items, z.Lat, z.Lng, map[string]any{
		"lines":        map[string]any{},
		"delivery_fee": fee + 1000,
	})), "delivery_fee_changed")
	if ch == nil {
		t.Fatalf("**تبدّلُ الأجور لم يُقَل**")
	}
	if int64(ch["new_value"].(float64)) != fee {
		t.Fatalf("**الأجرُ الحاليُّ لم يُقَل**: %v", ch)
	}

	// **والمطابقُ يصمت.**
	same := changesOf(t, quoteWith(t, hh, items, z.Lat, z.Lng, map[string]any{
		"lines":        map[string]any{},
		"delivery_fee": fee,
	}))
	if hasChange(same, "delivery_fee_changed") != nil {
		t.Fatalf("**أجرٌ لم يتبدّل قيل إنّه تبدّل**: %v", same)
	}
}

// ═════════════════ CA-06 — تبدّلان يُقالان معاً ═════════════════

// TestCA06_AllChangesAreSurfacedNotOnlyFirst **ولا يُخفي الأوّلُ ما بعده.**
//
// **ومن أُخبر بواحدٍ فأصلحه ثمّ أُخبر بثانٍ يقرأ المنصّةَ تتلاعب به.**
func TestCA06_AllChangesAreSurfacedNotOnlyFirst(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ CA-06")
	off := hh.NewItem(1000)
	priced := hh.NewItem(2000)

	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE menu_items SET available = false WHERE id = $1`, off.ID); err != nil {
		t.Fatalf("إيقافُ الصنف: %v", err)
	}

	items := []map[string]any{
		{"menu_item_id": off.ID, "qty": 1},
		{"menu_item_id": priced.ID, "qty": 1},
	}
	cs := changesOf(t, quoteWith(t, hh, items, z.Lat, z.Lng, map[string]any{
		// **سعرٌ لا يساوي شيئاً** — **فيقع تبدّلُ سعرٍ قطعاً.**
		"lines": map[string]any{off.ID: 1, priced.ID: 1},
	}))

	// **ولا أجرَ يُقارَن في سلّةٍ لا تُسعَّر** — **ورقمٌ يُقال عن
	// سلّةٍ لم تُحسَب رقمٌ مخترَع.** **والأجرُ يُقاس في `CC-10`.**
	for _, want := range []string{"product_unavailable", "product_price_changed"} {
		if hasChange(cs, want) == nil {
			t.Fatalf("**تبدّلٌ لم يُقَل مع إخوته**: %s — %v", want, cs)
		}
	}
}

// ═════════════════ CC-21 · CC-22 — ولا حدَّ أدنى خفيّ ═════════════════

// TestCC21_CC22_NoHiddenMinimumOrder **والحدُّ الأدنى مُلغىً بقرار
// المالك ٢٠٢٦-٠٨-٠١.**
//
// **وحدٌّ خفيٌّ أسوأُ من حدٍّ عالٍ** — **ومن رُدّ بما لم يره لا يفهم
// لماذا.** **فيُقاس أنّ طلباً زهيداً يمرّ.**
func TestCC21_CC22_NoHiddenMinimumOrder(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ CC-21")

	// **وحدٌّ عالٍ يُكتب في المنطقة** — **ولا يُقرأ.**
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE delivery_zones SET min_order = 500000 WHERE id = $1::uuid`, z.ID); err != nil {
		t.Fatalf("كتابةُ الحدّ: %v", err)
	}

	it := hh.NewItem(100)
	u := hh.Customer()
	r := hh.POST("/api/v1/orders", u.Token, map[string]any{
		"items":          []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
		"address_text":   "الرقة — شارع الاختبار",
		"lat":            z.Lat,
		"lng":            z.Lng,
		"payment_method": "cash",
	})
	if r.Code >= 400 {
		t.Fatalf("**طلبٌ زهيدٌ رُدّ بحدٍّ لم يره صاحبُه**: %d / %s — "+
			"**والحدُّ مُلغىً بقرار المالك**", r.Code, r.Err())
	}

	// **CC-22 · ولا يُعلَن حدٌّ في التسعيرة** — **ورقمٌ يُعرَض ولا
	// يُفرَض وعدٌ بالعكس.**
	j := quoteWith(t, hh, []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
		z.Lat, z.Lng, nil).JSON()
	if _, ok := j["min_order"]; ok {
		t.Fatalf("**حدٌّ أدنى معروضٌ وهو غيرُ مفروض**: %v", j["min_order"])
	}
}
