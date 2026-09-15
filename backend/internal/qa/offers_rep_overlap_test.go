package qa

// ══════════════════════════════════════════════════════════════════════
// **المندوبُ والتداخلُ والسلّةُ البائتة** (`RP-OF` · `OF-OV` · `OF-17`)
// ══════════════════════════════════════════════════════════════════════
//
// # وسلطانُ المندوب قائمٌ لا مُخترَع
//
// **و`merchants.sales_rep_user_id` علاقةٌ في القاعدة منذ ما قبل هذه
// الدفعة** — **بها يكتب المندوبُ في قائمة عميله** (`repClient`).
// **فعروضُه كقائمته**: **ولا تُخترَع له قدرةٌ عامّة.**
//
// # والتداخلُ محسومٌ في القاعدة لا في غُو
//
// **`offers_one_live_per_item`** — **فهرسٌ فريدٌ على الصنف حيث
// `active`** (الهجرة ٠٠٧٤): **وخصمان على صنفٍ واحدٍ سؤالٌ بلا جواب**:
// **أيُّهما يُطبَّق؟ الأكبرُ أم الأحدث؟** **فمُنع في القاعدة.**
//
// **ولا تُخترَع قاعدةُ تراكمٍ هنا** — **والمنعُ قائمٌ فيُقاس.**

import (
	"net/http"
	"testing"
	"time"
)

// repOffer ينشئ عرضاً من باب المندوب.
func repOffer(t *testing.T, h *Harness, fx offerFx, tok, merchantID string,
	percent int, ends *time.Time) Res {
	t.Helper()
	body := map[string]any{
		"title": "عرضُ المندوب", "menu_item_id": fx.Item.ID, "discount_percent": percent,
	}
	if ends != nil {
		body["ends_at"] = ends.Format(time.RFC3339)
	}
	return h.POST("/api/v1/rep/stores/"+merchantID+"/offers", tok, body)
}

// ═════════════════ RP-OF-01 · RP-OF-05 ═════════════════

// TestRPOF01_RPOF05_RepManagesAssignedMerchant **ومندوبُ المتجر ينزّل
// عرضاً لعميله** — **ويراه صاحبُ المتجر في بابه هو.**
func TestRPOF01_RPOF05_RepManagesAssignedMerchant(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ RP-OF-01")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)

	r := repOffer(t, hh, fx, fx.RepTok, fx.M.ID, 20, &end)
	if r.Code != http.StatusOK {
		t.Fatalf("**مُنع المندوبُ من عرضٍ لعميله**: %d / %s", r.Code, r.Err())
	}
	id := offerIDOf(t, r)

	// **والسعرُ نزل فعلاً** — **بالمحرّك نفسِه.**
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 800 {
		t.Fatalf("**عرضُ المندوب لم يُنقص السعر**: %d", got)
	}

	// **RP-OF-05 · ويراه صاحبُ المتجر** — **نظامٌ واحدٌ لا نظامان.**
	lst := hh.GET("/api/v1/merchant/stores/"+fx.M.ID+"/offers", fx.Tok)
	if lst.Code != http.StatusOK {
		t.Fatalf("**قائمةُ صاحب المتجر رُدّت**: %d", lst.Code)
	}
	rows, _ := lst.JSON()["offers"].([]any)
	if len(rows) != 1 {
		t.Fatalf("**عرضُ مندوبه لا يُرى في بابه**: %v", rows)
	}
	first, _ := rows[0].(map[string]any)
	if first["id"] != id {
		t.Fatalf("**عرضٌ آخر**: %v", first)
	}

	// **ويُنزله صاحبُ المتجر** — **فالعرضُ عرضُ متجره لا عرضُ من أنشأه.**
	stop := "/api/v1/merchant/stores/" + fx.M.ID + "/offers/" + id + "/stop"
	if got := offerStatus(t, hh.POST(stop, fx.Tok, map[string]any{})); got != "stopped" {
		t.Fatalf("**لم يملك صاحبُ المتجر إنزالَ عرضٍ في متجره**: %q", got)
	}
}

// ═════════════════ RP-OF-02 · RP-OF-03 ═════════════════

// TestRPOF02_RPOF03_RepCannotEscapeAssignment **ولا يبلغ متجراً ليس
// عميلَه** — **لا بمساره ولا بمعرّفِ صنفه.**
func TestRPOF02_RPOF03_RepCannotEscapeAssignment(t *testing.T) {
	hh := New(t)
	f := hh.Factory()
	mine := newOfferFx(t, hh, f, 1000)
	theirs := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)

	// **RP-OF-02 · متجرٌ لمندوبٍ آخر.**
	r := repOffer(t, hh, theirs, mine.RepTok, theirs.M.ID, 50, &end)
	if r.Code != http.StatusForbidden {
		t.Fatalf("**بلغ المندوبُ متجراً ليس عميلَه**: %d", r.Code)
	}

	// **RP-OF-03 · وبمسار عميله وصنفِ غيره** — **وهو التفافٌ لا يُكشف
	// بحارس المسار وحدَه.**
	body := map[string]any{
		"title": "التفاف", "menu_item_id": theirs.Item.ID, "discount_percent": 50,
		"ends_at": end.Format(time.RFC3339),
	}
	r = hh.POST("/api/v1/rep/stores/"+mine.M.ID+"/offers", mine.RepTok, body)
	if r.Code != http.StatusForbidden {
		t.Fatalf("**خصمٌ نُزّل على صنفِ متجرٍ ليس عميلَه**: %d", r.Code)
	}

	// **ولا يُنزِل عرضَ غيره بمعرّفه** — **والمعرّفاتُ تُقرأ من ردٍّ سابق.**
	live := offerIDOf(t, repOffer(t, hh, theirs, theirs.RepTok, theirs.M.ID, 10, &end))
	stop := "/api/v1/rep/stores/" + mine.M.ID + "/offers/" + live + "/stop"
	if r = hh.POST(stop, mine.RepTok, map[string]any{}); r.Code != http.StatusForbidden {
		t.Fatalf("**أنزل المندوبُ عرضَ متجرٍ ليس عميلَه**: %d", r.Code)
	}
	var active bool
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT active FROM offers WHERE id = $1::uuid`, live).Scan(&active)
	if !active {
		t.Fatalf("**أُنزل العرضُ رغم المنع**")
	}
}

// ═════════════════ RP-OF-04 ═════════════════

// TestRPOF04_RepActionIsAuditedAsRep **ومن فعلها يُقيَّد بدوره.**
//
// **و«أنزل المندوبُ عرضاً» و«أنزله صاحبُ المتجر» سؤالان مختلفان
// يُسألان بعد شهر** — **وسجلٌّ لا يفرّق بينهما لا يُجيب.**
func TestRPOF04_RepActionIsAuditedAsRep(t *testing.T) {
	hh := New(t)
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	id := offerIDOf(t, repOffer(t, hh, fx, fx.RepTok, fx.M.ID, 15, &end))

	// **والقيدُ يُكتب في خيطٍ جانبيّ** — **فيُنتظَر قليلاً ولا يُفترَض.**
	var actor, by string
	for i := 0; i < 50; i++ {
		err := hh.Pool.QueryRow(ctxBG(), `
			SELECT actor_user_id::text, COALESCE(details->>'by', '')
			FROM audit_log WHERE entity = 'offer' AND entity_id = $1
			  AND action = 'catalog.offer_create'
			ORDER BY created_at DESC LIMIT 1`, id).Scan(&actor, &by)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if actor != fx.Rep.ID {
		t.Fatalf("**الفاعلُ في السجلّ ليس المندوب**: %q", actor)
	}
	if by != "rep" {
		t.Fatalf("**لم يُقيَّد دورُ الفاعل**: %q", by)
	}
}

// ═════════════════ OF-OV-01 … OF-OV-04 ═════════════════

// TestOFOV01_OFOV02_NoStacking **ولا خصمان على صنفٍ واحد.**
//
// **والمنعُ في القاعدة** — **لا في شرطٍ في غُو يُنسى في بابٍ ثانٍ.**
func TestOFOV01_OFOV02_NoStacking(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ OF-OV-01")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)

	if r := makeOffer(t, hh, fx, fx.Tok, 20, nil, &end); r.Code != http.StatusOK {
		t.Fatalf("الأوّل: %d / %s", r.Code, r.Err())
	}
	// **والثاني يُردّ بصراحة** — **ولا يُقبَل صامتاً فيتراكم.**
	second := makeOffer(t, hh, fx, fx.Tok, 30, nil, &end)
	if second.Code != http.StatusConflict {
		t.Fatalf("**قُبل خصمٌ ثانٍ على الصنف نفسِه**: %d / %s", second.Code, second.Err())
	}
	// **والسعرُ خصمٌ واحدٌ لا خصمان** — **٨٠٠ لا ٥٦٠.**
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 800 {
		t.Fatalf("**تراكم خصمان**: %d", got)
	}
}

// TestOFOV03_QuoteAndOrderResolveIdentically **والتسعيرةُ والطلبُ
// يقرآن الخصمَ نفسَه.**
//
// **ومصدرٌ واحدٌ** (`LiveDiscount`) — **ولو افترقا لرأى سعراً ودُفع
// غيرُه.**
func TestOFOV03_QuoteAndOrderResolveIdentically(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ OF-OV-03")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	if r := makeOffer(t, hh, fx, fx.Tok, 25, nil, &end); r.Code != http.StatusOK {
		t.Fatalf("العرض: %d / %s", r.Code, r.Err())
	}

	quoted := unitPriceOf(t, hh, z, fx.Item.ID)
	u := hh.Customer()
	charged := subtotalOf(t, hh, placeOrder(t, hh, u, z, fx.Item))
	if quoted != charged {
		t.Fatalf("**رأى %d ودُفع %d**", quoted, charged)
	}
	if quoted != 750 {
		t.Fatalf("**الخصمُ لم يُطبَّق**: %d", quoted)
	}
}

// TestOFOV04_StoppingOneRestoresPrice **ومن أنزل عرضاً عاد السعرُ إلى
// ما كان** — **ولا يبقى أثرٌ من خصمٍ مُنزَل.**
func TestOFOV04_StoppingOneRestoresPrice(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ OF-OV-04")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	id := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 20, nil, &end))

	stop := "/api/v1/merchant/stores/" + fx.M.ID + "/offers/" + id + "/stop"
	if r := hh.POST(stop, fx.Tok, map[string]any{}); r.Code != http.StatusOK {
		t.Fatalf("الإنزال: %d / %s", r.Code, r.Err())
	}
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 1000 {
		t.Fatalf("**بقي أثرُ خصمٍ مُنزَل**: %d", got)
	}

	// **والموضعُ يُخلى بعد الإنزال** — **فيُنشأ عرضٌ جديدٌ بلا تعارض.**
	if r := makeOffer(t, hh, fx, fx.Tok, 35, nil, &end); r.Code != http.StatusOK {
		t.Fatalf("**مُنع عرضٌ جديدٌ بعد إنزال سابقه**: %d / %s", r.Code, r.Err())
	}
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 650 {
		t.Fatalf("**العرضُ الجديدُ لم يُطبَّق**: %d", got)
	}
}

// TestOFOV05_ExpiredHoldsNoSlot **والمنتهي لا يحجز موضعَ صنفه.**
//
// **والفهرسُ يمنع عرضين `active`** — **ومنتهٍ لم يُنزَل يشغل الموضعَ
// وهو لا يُسعَّر به**: **فيقول لصاحبه «هذا الصنفُ مخصومٌ» وليس كذلك.**
//
// **فيُطوى المنتهي في الإنشاء نفسِه** — **ولا يُحذف** (تقريرُ ما مضى).
func TestOFOV05_ExpiredHoldsNoSlot(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ OF-OV-05")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	old := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 20, nil, &end))
	forceWindow(t, hh, old, -3*time.Hour, -time.Hour)

	r := makeOffer(t, hh, fx, fx.Tok, 40, nil, &end)
	if r.Code != http.StatusOK {
		t.Fatalf("**حجز عرضٌ منتهٍ موضعَ الصنف**: %d / %s", r.Code, r.Err())
	}
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 600 {
		t.Fatalf("**العرضُ الجديدُ لم يُطبَّق**: %d", got)
	}
	// **والقديمُ باقٍ للتقرير** — **مطويّاً لا محذوفا.**
	var n int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM offers WHERE id = $1::uuid`, old).Scan(&n)
	if n != 1 {
		t.Fatalf("**حُذف العرضُ القديم** — **ولا يُقرأ في تقرير**")
	}
}

// ═════════════════ OF-17 ═════════════════

// TestOF17_StaleCartCatchesOfferLoss **والسلّةُ البائتةُ تُكشف بعقد
// الدفعة الخامسة.**
//
// **ومن فتح سلّتَه والخصمُ قائمٌ ثمّ أُنزل قبل أن يرسل** — **يُقال له
// إنّ السعرَ تبدّل، ولا يُرسَل طلبٌ بسعرٍ رآه ولم يعد قائما.**
func TestOF17_StaleCartCatchesOfferLoss(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ OF-17")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	id := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 20, nil, &end))

	// **وما رآه على الشاشة**: ٨٠٠.
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 800 {
		t.Fatalf("تجهيزٌ فاسد: %d", got)
	}

	// **ثمّ يُنزَل العرضُ وهو في الدفع.**
	stop := "/api/v1/merchant/stores/" + fx.M.ID + "/offers/" + id + "/stop"
	if r := hh.POST(stop, fx.Tok, map[string]any{}); r.Code != http.StatusOK {
		t.Fatalf("الإنزال: %d / %s", r.Code, r.Err())
	}

	cs := changesOf(t, quoteWith(t, hh,
		[]map[string]any{{"menu_item_id": fx.Item.ID, "qty": 1}}, z.Lat, z.Lng,
		map[string]any{"lines": map[string]any{fx.Item.ID: 800}}))
	ch := hasChange(cs, "product_price_changed")
	if ch == nil {
		t.Fatalf("**سقط الخصمُ ولم يُقَل** — **فيُرسَل طلبٌ بسعرٍ رآه ولم يعد قائما**: %v", cs)
	}
	if ch["old_value"] != float64(800) || ch["new_value"] != float64(1000) {
		t.Fatalf("**التبدّلُ لا يقول الرقمين**: %v", ch)
	}
}

// ═════════════════ OF-18 ═════════════════

// TestOF18_BrowseMatchesQuote **وما يُعرض في التصفّح هو ما تقوله
// التسعيرة.**
//
// **وسعرٌ مشطوبٌ في شاشةٍ وسعرٌ آخرُ في السلّة خدعةٌ لا خطأ** —
// **والسعران يُحسبان في الخادم بالقاعدة نفسِها** (`AfterDiscount`).
func TestOF18_BrowseMatchesQuote(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ OF-18")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	if r := makeOffer(t, hh, fx, fx.Tok, 20, nil, &end); r.Code != http.StatusOK {
		t.Fatalf("العرض: %d / %s", r.Code, r.Err())
	}

	br := hh.GET("/api/v1/public/offers", "")
	if br.Code != http.StatusOK {
		t.Fatalf("**التصفّحُ رُدّ**: %d", br.Code)
	}
	rows, _ := br.JSON()["offers"].([]any)
	var shown float64
	var found bool
	for _, raw := range rows {
		o, _ := raw.(map[string]any)
		if o["menu_item_id"] == fx.Item.ID {
			shown, _ = o["price_after"].(float64)
			found = true
		}
	}
	if !found {
		t.Fatalf("**عرضٌ سارٍ لا يظهر في التصفّح**")
	}
	if got := unitPriceOf(t, hh, z, fx.Item.ID); int64(shown) != got {
		t.Fatalf("**عُرض %d وسُعّر %d**", int64(shown), got)
	}
}

// ═════════════════ التزاحم (البند ١٦) ═════════════════

// TestOFCC_StopBetweenQuoteAndOrderWins **ويُنزَل العرضُ بين تسعيرةٍ
// وطلبٍ** — **فالحَكَمُ آخرُ قراءةٍ من القاعدة لا ما رآه.**
//
// # ولمَ تداخلٌ مرتَّبٌ لا خيوطٌ متسابقة
//
// **وقِيس في هذه الدفعة**: **تسعيرتان متزامنتان تمرّان، وثلاثٌ تعلق
// حتّى مهلةِ الخادم** — **ومجمّعُ الفحص على أربعةِ اتّصالاتٍ افتراضاً
// والتسعيرةُ تمسك اثنين.** **وهو حدُّ أداةِ الفحص لا حدُّ المنتَج**:
// **الإنتاجُ على عشرين** (`database/postgres.go`).
//
// **وفحصٌ يعلق ثلاثين ثانيةً ثمّ يُسقط فحصاً بعده ليس دليلاً** —
// **وقد أسقط `TestOF18` فعلاً.** **فيُرتَّب التداخلُ ويُقاس ما يُراد
// قياسُه**: **أنّ ما بعد الإنزال لا يحمل خصماً.**
func TestOFCC_StopBetweenQuoteAndOrderWins(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ OF-CC")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	id := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 20, nil, &end))

	// **وما رآه في سلّته.**
	if got := unitPriceOf(t, hh, z, fx.Item.ID); got != 800 {
		t.Fatalf("تجهيزٌ فاسد: %d", got)
	}
	// **ثمّ يُنزَل وهو يضغط «أرسل».**
	stop := "/api/v1/merchant/stores/" + fx.M.ID + "/offers/" + id + "/stop"
	if r := hh.POST(stop, fx.Tok, map[string]any{}); r.Code != http.StatusOK {
		t.Fatalf("الإنزال: %d / %s", r.Code, r.Err())
	}
	u := hh.Customer()
	if got := subtotalOf(t, hh, placeOrder(t, hh, u, z, fx.Item)); got != 1000 {
		t.Fatalf("**طلبٌ بُني بخصمٍ أُنزل قبله**: %d", got)
	}
	// **ولا أثرَ ماليٌّ مزدوج** — **صفٌّ واحدٌ بسعرٍ واحد.**
	var lines int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM order_items WHERE menu_item_id = $1::uuid`,
		fx.Item.ID).Scan(&lines)
	if lines != 1 {
		t.Fatalf("**سطورٌ غيرُ متوقَّعة**: %d", lines)
	}
}

// TestOFCC_OrderAtExpiryIsNotDiscounted **وطلبٌ يُبنى بعد انتهائه لا
// يحمل خصمَه** — **ولا أثرَ ماليٌّ مزدوج.**
func TestOFCC_OrderAtExpiryIsNotDiscounted(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ OF-CC-2")
	f := hh.Factory()
	fx := newOfferFx(t, hh, f, 1000)
	end := time.Now().Add(2 * time.Hour)
	id := offerIDOf(t, makeOffer(t, hh, fx, fx.Tok, 20, nil, &end))

	u := hh.Customer()
	first := subtotalOf(t, hh, placeOrder(t, hh, u, z, fx.Item))
	if first != 800 {
		t.Fatalf("تجهيزٌ فاسد: %d", first)
	}
	// **ثمّ تمضي نهايتُه بين طلبين.**
	forceWindow(t, hh, id, -3*time.Hour, -time.Nanosecond)
	second := subtotalOf(t, hh, placeOrder(t, hh, u, z, fx.Item))
	if second != 1000 {
		t.Fatalf("**طلبٌ بعد الانتهاء حمل الخصم**: %d", second)
	}
	// **والأوّلُ كما قُيّد** — **ولا يُعاد حسابُه.**
	if first != 800 {
		t.Fatalf("**تبدّل ما قُيّد**: %d", first)
	}
}
