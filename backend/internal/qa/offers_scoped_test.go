package qa

// ══════════════════════════════════════════════════════════════════════
// **عروضُ المتجر — من يملك الصنفَ يملك عرضَه** (`OF`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// # ولا محرّكَ ثانٍ
//
// **والجدولُ نفسُه وشرطُ السريان نفسُه** — **والدالّةُ التي تُنقص السعرَ
// في بناء الطلب هي التي تُنقصه في التسعيرة.**
//
// **فلا يُقاس هنا «هل يُخزَّن الصفّ؟»** — **بل: هل يتبدّل ما يدفعه
// الزبونُ فعلاً؟** **وحارسٌ يقرأ صفّاً ولا يقرأ سعراً زينة** (درسُ
// الدفعة السادسة).
//
// # ووقتُ الخادم وحدَه
//
// **ولا يُرسَل وقتٌ من الجهاز أصلاً** — **والحالُ تُشتقّ من `now()` في
// القاعدة ومن `time.Now()` في العمليّة.** **فتقديمُ ساعةِ الهاتف لا
// يُحيي عرضاً انتهى ولا يُقدّم مجدولاً.**

import (
	"net/http"
	"testing"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **أدواتُ الفحص**
// ══════════════════════════════════════════════════════════════════════

// offerFx متجرٌ وصنفٌ ومندوبٌ — **حالٌ كاملةٌ يُبنى عليها.**
type offerFx struct {
	M    *Merchant
	Item *Item
	Rep  *Rep
	// Tok رمزُ صاحب المتجر، و RepTok رمزُ مندوبه.
	Tok    string
	RepTok string
}

// newOfferFx **متجرٌ بصاحبٍ ومندوبٍ وصنفٍ بسعرٍ معلوم.**
//
// **والسعرُ يُثبَّت** (`NewItemPriced`) — **فالخصمُ يُقاس على رقمٍ يُعرَف
// لا على هامشٍ يتبدّل بإعداد.**
//
// **والمصنعُ يُمرَّر ولا يُبنى هنا** — **وبذرتُه اسمُ السيناريو**
// (`NewNamespace(t.Name())`): **فمصنعان في فحصٍ واحدٍ يُنتجان الأرقامَ
// نفسَها**، **فيسقط الثاني على `users_phone_key`.** (قِيس.)
func newOfferFx(t *testing.T, h *Harness, f *Factory, sale int64) offerFx {
	t.Helper()
	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID))
	it := h.NewItemPriced(m, sale, sale)
	return offerFx{
		M: m, Item: it, Rep: rep,
		Tok:    h.TokenFor(m.Owner.ID, "merchant"),
		RepTok: h.TokenFor(rep.ID, "sales"),
	}
}

// makeOffer ينشئ عرضاً من باب المتجر.
func makeOffer(t *testing.T, h *Harness, fx offerFx, tok string,
	percent int, starts, ends *time.Time) Res {
	t.Helper()
	body := map[string]any{
		"title": "عرضُ فحص", "menu_item_id": fx.Item.ID, "discount_percent": percent,
	}
	if starts != nil {
		body["starts_at"] = starts.Format(time.RFC3339)
	}
	if ends != nil {
		body["ends_at"] = ends.Format(time.RFC3339)
	}
	return h.POST("/api/v1/merchant/stores/"+fx.M.ID+"/offers", tok, body)
}

// offerStatus حالُ العرض كما يقولها الخادم.
func offerStatus(t *testing.T, r Res) string {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("**العرضُ رُدّ**: %d / %s", r.Code, r.Err())
	}
	st, _ := r.JSON()["status"].(string)
	return st
}

// unitPriceOf سعرُ الوحدة كما تقوله التسعيرةُ — **ما يدفعه الزبون.**
func unitPriceOf(t *testing.T, h *Harness, z zoneFx, itemID string) int64 {
	t.Helper()
	r := quoteWith(t, h, []map[string]any{{"menu_item_id": itemID, "qty": 1}}, z.Lat, z.Lng, nil)
	if r.Code != http.StatusOK {
		t.Fatalf("**التسعيرةُ رُدّت**: %d / %s", r.Code, r.Err())
	}
	sub, ok := r.JSON()["subtotal"].(float64)
	if !ok {
		t.Fatalf("**لا مجموعَ في التسعيرة**: %v", r.JSON())
	}
	return int64(sub)
}

// forceWindow **يُزحزَح زمنُ العرض في القاعدة** — **ولا يُنتظَر ساعةُ حائط.**
//
// **والصفُّ يُكتب مباشرةً لأنّ البابَ يمنع ماضياً** — **وهو ما يُراد
// إثباتُه**: **أنّ التسعيرَ يقرأ الزمنَ لا الرايةَ.**
func forceWindow(t *testing.T, h *Harness, offerID string, startsIn, endsIn time.Duration) {
	t.Helper()
	now := time.Now()
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE offers SET starts_at = $2, ends_at = $3
		WHERE id = $1::uuid`, offerID, now.Add(startsIn), now.Add(endsIn)); err != nil {
		t.Fatalf("تحريكُ مدّة العرض: %v", err)
	}
}

func offerIDOf(t *testing.T, r Res) string {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("**العرضُ رُدّ**: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	if id == "" {
		t.Fatalf("**عرضٌ بلا معرّف**: %v", r.JSON())
	}
	return id
}

// ═════════════════ OF-01 · OF-02 · OF-03 · OF-19 ═════════════════

// TestOF01_MerchantCreatesOwnOffer **وصاحبُ المتجر ينزّل عرضَه بنفسه.**
func TestOF01_MerchantCreatesOwnOffer(t *testing.T) {
	hh := New(t)
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)

	r := makeOffer(t, hh, fx, fx.Tok, 20, nil, &end)
	if r.Code != http.StatusOK {
		t.Fatalf("**مُنع صاحبُ المتجر من عرضٍ على صنفه**: %d / %s", r.Code, r.Err())
	}
	if got := offerStatus(t, r); got != "active" {
		t.Fatalf("**عرضٌ يبدأ الآن ولم يُقَل إنّه سارٍ**: %s", got)
	}
	// **ويتحمّله المتجرُ حتماً** — **ولا ينفق من هامشِ غيره.**
	if by, _ := r.JSON()["borne_by"].(string); by != "merchant" {
		t.Fatalf("**خصمٌ ينشئه المتجرُ ويتحمّله غيرُه**: %q", by)
	}
}

// TestOF02_OF19_CannotCreateForAnotherMerchant **ولا يُنزَّل عرضٌ على
// قائمة غيره** — **لا بمعرّف المتجر ولا بمعرّف الصنف.**
func TestOF02_OF19_CannotCreateForAnotherMerchant(t *testing.T) {
	hh := New(t)
	f := hh.Factory()
	mine := newOfferFx(t, hh, f, 1000)
	theirs := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)

	// **OF-19 · بمعرّفِ متجرٍ ليس له** — **ولو زُوّر في المسار.**
	body := map[string]any{
		"title": "سطو", "menu_item_id": theirs.Item.ID, "discount_percent": 50,
		"ends_at": end.Format(time.RFC3339),
	}
	r := hh.POST("/api/v1/merchant/stores/"+theirs.M.ID+"/offers", mine.Tok, body)
	if r.Code != http.StatusForbidden {
		t.Fatalf("**نُزّل عرضٌ على متجرٍ ليس لصاحب الحساب**: %d", r.Code)
	}

	// **OF-02 · وبمسارِ متجره هو وصنفِ غيره** — **وهو الأخطر**:
	// **البابُ مأذونٌ فيه والحمولةُ تشير إلى مكانٍ آخر.**
	r = hh.POST("/api/v1/merchant/stores/"+mine.M.ID+"/offers", mine.Tok, body)
	if r.Code != http.StatusForbidden {
		t.Fatalf("**خصمٌ نُزّل على صنفِ متجرٍ آخرَ من بابٍ مأذونٍ فيه**: %d", r.Code)
	}
	// **ولا أثرَ في القاعدة.**
	var n int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM offers WHERE menu_item_id = $1::uuid`, theirs.Item.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("**كُتب عرضٌ رغم المنع**: %d", n)
	}
}

// TestOF03_CannotReadAnotherMerchantOffers **ولا يُعدّ عروضُ غيره.**
func TestOF03_CannotReadAnotherMerchantOffers(t *testing.T) {
	hh := New(t)
	f := hh.Factory()
	mine := newOfferFx(t, hh, f, 1000)
	theirs := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	if r := makeOffer(t, hh, theirs, theirs.Tok, 30, nil, &end); r.Code != http.StatusOK {
		t.Fatalf("تجهيزُ عرضِ الآخر: %d / %s", r.Code, r.Err())
	}

	r := hh.GET("/api/v1/merchant/stores/"+theirs.M.ID+"/offers", mine.Tok)
	if r.Code != http.StatusForbidden {
		t.Fatalf("**قُرئت عروضُ متجرٍ آخر**: %d", r.Code)
	}
	// **وقائمتُه هو لا تحمل عروضَ غيره.**
	r = hh.GET("/api/v1/merchant/stores/"+mine.M.ID+"/offers", mine.Tok)
	if r.Code != http.StatusOK {
		t.Fatalf("**مُنع من قراءة عروضه**: %d", r.Code)
	}
	rows, _ := r.JSON()["offers"].([]any)
	if len(rows) != 0 {
		t.Fatalf("**عروضُ غيره في قائمته**: %v", rows)
	}
}

// ═════════════════ OF-04 · OF-05 · OF-07 · OF-08 ═════════════════

// TestOF04_OF05_OF07_OF08_StatusIsDerivedFromServerTime **والحالُ تُشتقّ
// من وقت الخادم.**
func TestOF04_OF05_OF07_OF08_StatusIsDerivedFromServerTime(t *testing.T) {
	hh := New(t)
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)

	// **OF-04 · مجدولٌ** — بدايتُه غدا.
	start := time.Now().Add(24 * time.Hour)
	end := start.Add(24 * time.Hour)
	r := makeOffer(t, hh, fx, fx.Tok, 10, &start, &end)
	if got := offerStatus(t, r); got != "scheduled" {
		t.Fatalf("**عرضٌ يبدأ غداً قيل عنه** %q", got)
	}
	id := offerIDOf(t, r)

	// **OF-05 · فإذا حلّ وقتُه صار ساريا** — **والصفُّ كما هو.**
	forceWindow(t, hh, id, -time.Hour, time.Hour)
	r = hh.GET("/api/v1/merchant/stores/"+fx.M.ID+"/offers", fx.Tok)
	if got := statusOfFirst(t, r); got != "active" {
		t.Fatalf("**حلّ وقتُه وقيل عنه** %q", got)
	}

	// **OF-07 · OF-08 · ومضت نهايتُه فانتهى وحدَه** — **بلا مُجدوِلٍ
	// ولا نداءٍ من أحد.**
	forceWindow(t, hh, id, -2*time.Hour, -time.Hour)
	r = hh.GET("/api/v1/merchant/stores/"+fx.M.ID+"/offers", fx.Tok)
	if got := statusOfFirst(t, r); got != "expired" {
		t.Fatalf("**مضت نهايتُه وقيل عنه** %q", got)
	}
}

// statusOfFirst حالُ أوّل عرضٍ في القائمة.
func statusOfFirst(t *testing.T, r Res) string {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("**القائمةُ رُدّت**: %d / %s", r.Code, r.Err())
	}
	rows, _ := r.JSON()["offers"].([]any)
	if len(rows) == 0 {
		t.Fatalf("**لا عرضَ في القائمة**")
	}
	first, _ := rows[0].(map[string]any)
	st, _ := first["status"].(string)
	return st
}

// ═════════════════ OF-06 · OF-12 · OF-13 · OF-16 ═════════════════

// TestOF06_OF12_OF13_OF16_StopEndsPricingAndIsIdempotent **والمُنزَل لا
// يُسعَّر به** — **والإنزالُ مرّتين حالٌ واحدة.**
func TestOF06_OF12_OF13_OF16_StopEndsPricingAndIsIdempotent(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ OF-06")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	id := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 20, nil, &end))

	// **والسعرُ نزل فعلاً** — **ولا يُصدَّق الصفُّ وحدَه.**
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 800 {
		t.Fatalf("**عرضٌ سارٍ ولم يُنقص السعر**: %d", got)
	}

	stop := "/api/v1/merchant/stores/" + fx.M.ID + "/offers/" + id + "/stop"
	r := hh.POST(stop, fx.Tok, map[string]any{})
	if got := offerStatus(t, r); got != "stopped" {
		t.Fatalf("**أُنزل وقيل عنه** %q", got)
	}

	// **OF-12 · ولا يُسعَّر به بعد إنزاله.**
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 1000 {
		t.Fatalf("**عرضٌ مُنزَلٌ ما زال يُنقص السعر**: %d", got)
	}

	// **OF-16 · وإنزالٌ ثانٍ لا أثرَ له** — **ومن ضغط مرّتين لأنّ
	// الشبكةَ تأخّرت لا يُعاقَب.**
	r2 := hh.POST(stop, fx.Tok, map[string]any{})
	if r2.Code != http.StatusOK || offerStatus(t, r2) != "stopped" {
		t.Fatalf("**الإنزالُ الثاني لم يكن آمناً**: %d / %s", r2.Code, r2.Err())
	}
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 1000 {
		t.Fatalf("**السعرُ تبدّل بإنزالٍ مكرّر**: %d", got)
	}
}

// ═════════════════ OF-10 · OF-11 ═════════════════

// TestOF10_OF11_ExpiredOfferCannotPrice **والمنتهي لا يُسعَّر به** —
// **لا في تسعيرةٍ ولا في طلبٍ يُبنى.**
func TestOF10_OF11_ExpiredOfferCannotPrice(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ OF-10")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	id := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 25, nil, &end))
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 750 {
		t.Fatalf("**تجهيزٌ فاسد**: %d", got)
	}

	// **ومضت نهايتُه** — **والرايةُ `active` كما هي.**
	forceWindow(t, hh, id, -3*time.Hour, -time.Hour)

	// **OF-10 · التسعيرة.**
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 1000 {
		t.Fatalf("**عرضٌ منتهٍ ما زال يُنقص سعرَ التسعيرة**: %d", got)
	}

	// **OF-11 · وبناءُ الطلب** — **وهو المسارُ الذي يُقيَّد في الدفتر.**
	//
	// **ويُقاس مجموعُ الأصناف لا المجموعُ الكلّيّ** — **ورسمُ التوصيل
	// شأنٌ آخرُ لا يخصّ الخصم.**
	u := hh.Customer()
	oid := placeOrder(t, hh, u, z, fx.Item)
	if got := subtotalOf(t, hh, oid); got != 1000 {
		t.Fatalf("**طلبٌ بُني بخصمٍ منتهٍ**: %d", got)
	}
}

// ═════════════════ OF-14 · OF-15 ═════════════════

// TestOF14_OF15_HistoryIsImmutable **ولقطةُ الطلب لا تتبدّل بعده.**
//
// **ومن أنزل عرضَه بعد أن بيع به لا يُغيّر ما قُيّد** — **والتسويةُ
// تقرأ اللقطةَ لا الجدول.**
func TestOF14_OF15_HistoryIsImmutable(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ OF-14")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	id := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 20, nil, &end))

	u := hh.Customer()
	oid := placeOrder(t, hh, u, z, fx.Item)
	const want = int64(800)
	if got := subtotalOf(t, hh, oid); got != want {
		t.Fatalf("**الطلبُ لم يُبنَ بالخصم**: %d (والمنتظَر %d)", got, want)
	}

	// **OF-14 · يُنزَل العرض.**
	stop := "/api/v1/merchant/stores/" + fx.M.ID + "/offers/" + id + "/stop"
	if r := hh.POST(stop, fx.Tok, map[string]any{}); r.Code != http.StatusOK {
		t.Fatalf("الإنزال: %d / %s", r.Code, r.Err())
	}
	if got := subtotalOf(t, hh, oid); got != want {
		t.Fatalf("**مجموعُ طلبٍ مضى تبدّل بإنزال عرض**: %d ← %d", want, got)
	}

	// **OF-15 · ثمّ تمضي نهايتُه.**
	forceWindow(t, hh, id, -3*time.Hour, -time.Hour)
	if got := subtotalOf(t, hh, oid); got != want {
		t.Fatalf("**مجموعُ طلبٍ مضى تبدّل بانتهاء عرض**: %d ← %d", want, got)
	}
}

// subtotalOf مجموعُ أصنافِ طلبٍ كما قُيّد — **اللقطةُ لا الجدول.**
func subtotalOf(t *testing.T, h *Harness, orderID string) int64 {
	t.Helper()
	var sub int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT subtotal FROM orders WHERE id = $1::uuid`, orderID).Scan(&sub); err != nil {
		t.Fatalf("قراءةُ المجموع: %v", err)
	}
	return sub
}

// placeOrder **طلبٌ يُبنى بالمسار الحقيقيّ** — **لا بكتابةٍ في القاعدة.**
func placeOrder(t *testing.T, h *Harness, u *User, z zoneFx, it *Item) string {
	t.Helper()
	made := h.POST("/api/v1/orders", u.Token, zoneBody(it, z.Lat, z.Lng))
	if made.Code != http.StatusOK && made.Code != http.StatusCreated {
		t.Fatalf("**الطلبُ رُدّ**: %d / %s", made.Code, made.Err())
	}
	id, _ := made.JSON()["id"].(string)
	if id == "" {
		t.Fatalf("**طلبٌ بلا معرّف**: %v", made.JSON())
	}
	return id
}

// ═════════════════ OF-09 ═════════════════

// TestOF09_BadWindowRejected **ونهايةٌ قبل بدايةٍ تُردّ.**
func TestOF09_BadWindowRejected(t *testing.T) {
	hh := New(t)
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)

	start := time.Now().Add(3 * time.Hour)
	end := start.Add(-1 * time.Hour)
	if r := makeOffer(t, hh, fx, fx.Tok, 10, &start, &end); r.Code != http.StatusBadRequest {
		t.Fatalf("**قُبل عرضٌ ينتهي قبل أن يبدأ**: %d", r.Code)
	}
	// **ونهايةٌ مضت** — **وُلد منتهيا.**
	past := time.Now().Add(-1 * time.Hour)
	if r := makeOffer(t, hh, fx, fx.Tok, 10, nil, &past); r.Code != http.StatusBadRequest {
		t.Fatalf("**قُبل عرضٌ نهايتُه في الماضي**: %d", r.Code)
	}
	// **والمساواةُ كذلك** — **مدّةٌ صفرٌ لا تقع.**
	same := time.Now().Add(2 * time.Hour)
	if r := makeOffer(t, hh, fx, fx.Tok, 10, &same, &same); r.Code != http.StatusBadRequest {
		t.Fatalf("**قُبلت مدّةٌ صفر**: %d", r.Code)
	}
}

// ═════════════════ OF-20 ═════════════════

// TestOF20_DirectApiBypassFails **ولا بابَ من غير بابه.**
func TestOF20_DirectApiBypassFails(t *testing.T) {
	hh := New(t)
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	body := map[string]any{
		"title": "تجاوز", "menu_item_id": fx.Item.ID, "discount_percent": 50,
		"ends_at": end.Format(time.RFC3339), "borne_by": "platform",
	}

	// **ولا بابَ للإدارة بيد المتجر** — **ولم تُوسَّع قدرتُه.**
	if r := hh.POST("/api/v1/admin/offers", fx.Tok, body); r.Code == http.StatusOK {
		t.Fatalf("**بلغ بابَ الإدارة بحساب متجر**")
	}
	// **وبلا رمزٍ أصلاً.**
	if r := hh.POST("/api/v1/merchant/stores/"+fx.M.ID+"/offers", "", body); r.Code == http.StatusOK {
		t.Fatalf("**نُزّل عرضٌ بلا هُويّة**")
	}
	// **وبحساب زبونٍ موثَّق** — **الدورُ يُفحص قبل الملكيّة.**
	cust := hh.Customer()
	if r := hh.POST("/api/v1/merchant/stores/"+fx.M.ID+"/offers", cust.Token, body); r.Code == http.StatusOK {
		t.Fatalf("**نزّل زبونٌ عرضاً**")
	}
}

// ═════════════════ من يتحمّل لا يُختار ═════════════════

// TestOF18_MerchantCannotSpendPlatformMargin **ولا ينفق من هامشِ غيره.**
//
// **وحقلُ `borne_by` في الحمولة لا يُقرأ من بابه** — **ولو أرسله.**
func TestOFB_MerchantCannotSpendPlatformMargin(t *testing.T) {
	hh := New(t)
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	r := hh.POST("/api/v1/merchant/stores/"+fx.M.ID+"/offers", fx.Tok, map[string]any{
		"title": "على حساب المنصّة", "menu_item_id": fx.Item.ID,
		"discount_percent": 30, "ends_at": end.Format(time.RFC3339),
		"borne_by": "platform",
	})
	if r.Code != http.StatusOK {
		t.Fatalf("العرض: %d / %s", r.Code, r.Err())
	}
	if by, _ := r.JSON()["borne_by"].(string); by != "merchant" {
		t.Fatalf("**قرّر المتجرُ أن تتحمّله المنصّة**: %q", by)
	}
	var stored string
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT borne_by FROM offers WHERE id = $1::uuid`, offerIDOf(t, r)).Scan(&stored)
	if stored != "merchant" {
		t.Fatalf("**القاعدةُ تقول غيرَ الردّ**: %q", stored)
	}
}
