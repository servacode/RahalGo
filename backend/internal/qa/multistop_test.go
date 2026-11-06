package qa

// ══════════════════════════════════════════════════════════════════════
// **طلبٌ واحدٌ بعدّة محطّات** — قرارُ المالك ٢٠٢٦-١١-٠٥ («١ موافق»)
// ══════════════════════════════════════════════════════════════════════
//
// **الدفعةُ الأولى (P1)**: بوّابةُ نسخة السائق · صفوفُ الأقسام عند الإنشاء ·
// كلُّ المصادر مفتوحة. **ولا شيءَ يقرأ الأقسامَ بعد** — القبولُ والاستلامُ
// لكلّ محطّةٍ في الدفعات التالية.

import (
	"net/http"
	"testing"
	"time"
)

// multiStopBuild **رقمُ نسخة السائق «التي تعرف المحطّات» في الاختبار** —
// أعلى من كلّ نسخةٍ منشورة، **والمِسنَدُ لا يرسل ترويسةَ النسخة** فلا يُردّ
// نداءُ سائقٍ في الاختبار بـ٤٢٦.
const multiStopBuild = "9000"

// OpenMultiStop **يفتح بوّابةَ الطلب من عدّة متاجر** — نسخةُ المحطّات معلومةٌ
// (`orders.multistop_driver_build`) **ومفروضةٌ على كلّ السائقين**
// (`app.min_version.driver`). **ويُعاد الاثنان بعد الاختبار.**
func (h *Harness) OpenMultiStop() {
	h.T.Helper()
	h.Setting("orders.multistop_driver_build", multiStopBuild)
	h.Setting("app.min_version.driver", multiStopBuild)
}

// twoStoreBody **طلبٌ صحيحٌ بصنفٍ من كلّ متجر** — بترتيب السلّة.
func twoStoreBody(items ...*Item) map[string]any {
	body := orderBody(items[0], 1)
	lines := []map[string]any{}
	for _, it := range items {
		lines = append(lines, map[string]any{"menu_item_id": it.ID, "qty": 1})
	}
	body["items"] = lines
	return body
}

// quoteSources **سقفُ المصادر كما تقوله التسعيرةُ للسلّة.**
func quoteSources(t *testing.T, h *Harness, tok string, items ...*Item) (max float64, tooMany bool) {
	t.Helper()
	lines := []map[string]any{}
	for _, it := range items {
		lines = append(lines, map[string]any{"menu_item_id": it.ID, "qty": 1})
	}
	r := h.POST("/api/v1/public/quote", tok, map[string]any{
		"items": lines, "lat": raqqaLat, "lng": raqqaLng,
	})
	if r.Code != http.StatusOK {
		t.Fatalf("التسعيرة: %s", r)
	}
	j := r.JSON()
	max, _ = j["max_sources"].(float64)
	tooMany, _ = j["too_many_sources"].(bool)
	return max, tooMany
}

// TestMultiStop_RuntimeGate **سقفُ المصادر فوق الواحد ينتظر نسخةَ السائق** — `T36`.
//
// **وفي وقت التشغيل لا عند الحفظ**: **إنزالُ أدنى نسخةٍ للسائق يُغلق السلّةَ
// في اللحظة**، والتسعيرةُ والإنشاءُ يقولان الشيءَ نفسَه.
//
// **وكان السقفُ يسري وحدَه** — فسلّةٌ من متجرين تُقبل وسائقٌ بتطبيقٍ لا يعرف
// إلّا محطّةً واحدةً يأخذها.
func TestMultiStop_RuntimeGate(t *testing.T) {
	h := New(t)
	treasury(t, h)
	u := h.Customer()
	a := h.NewItem(1000)
	b := h.NewItem(1000)
	h.Setting("orders.max_sources", "2")
	h.Setting("orders.multistop_driver_build", "22")
	h.Setting("app.min_version.driver", "21")

	expect := func(step string, wantMax float64, wantTooMany bool) {
		t.Helper()
		max, tooMany := quoteSources(t, h, u.Token, a, b)
		if max != wantMax || tooMany != wantTooMany {
			t.Errorf("%s: التسعيرةُ تقول السقفَ %v وتجاوزَه %v — يُنتظر %v و%v",
				step, max, tooMany, wantMax, wantTooMany)
		}
	}

	// ١ · أدنى نسخةٍ للسائق دون نسخة المحطّات ⇒ متجرٌ واحد.
	expect("أدنى نسخةٍ ٢١ ونسخةُ المحطّات ٢٢", 1, true)
	r := h.POSTKey("/api/v1/orders", u.Token, uniq("ms-gate"), twoStoreBody(a, b))
	if r.Code != http.StatusConflict || r.Err() != "too_many_sources" {
		t.Fatalf("**طلبٌ من متجرين والبوّابةُ مغلقة**: %s — يُنتظر 409 too_many_sources", r)
	}

	// ٢ · فُرضت النسخة ⇒ السقفُ المضبوط.
	h.Setting("app.min_version.driver", "22")
	expect("أدنى نسخةٍ ٢٢", 2, false)
	if r := h.POSTKey("/api/v1/orders", u.Token, uniq("ms-gate"), twoStoreBody(a, b)); r.Code >= 400 {
		t.Fatalf("**البوّابةُ مفتوحةٌ والطلبُ من متجرين رُدّ**: %s", r)
	}

	// ٣ · رجوعٌ يُنزل أدنى نسخة ⇒ متجرٌ واحدٌ في اللحظة.
	h.Setting("app.min_version.driver", "21")
	expect("رجوعٌ إلى ٢١", 1, true)

	// ٤ · نسخةُ المحطّات صفر (الافتراض) ⇒ مغلقةٌ مهما علت أدنى نسخة.
	h.Setting("orders.multistop_driver_build", "0")
	h.Setting("app.min_version.driver", "99")
	expect("نسخةُ المحطّات صفر", 1, true)

	// ٥ · ورقمُ الملفّ المرفوع مع الفرض التلقائيّ يفتحها كالحدّ اليدويّ —
	// **وهو ما تقرؤه بوّابةُ التحديث نفسُها** (`effectiveMinVersion`).
	h.Setting("orders.multistop_driver_build", "22")
	h.Setting("app.min_version.driver", "0")
	h.Setting("release.driver.apk", `"driver-22.apk"`)
	h.Setting("release.driver.version_code", "22")
	expect("ملفٌّ مرفوعٌ برقم ٢٢ والفرضُ تلقائيّ", 2, false)
	h.Setting("release.driver.auto_force", "false")
	expect("ملفٌّ مرفوعٌ والفرضُ مطفأ", 1, true)

	// ٦ · والسقفُ واحدٌ يبقى واحداً ولو فُتحت البوّابة.
	h.Setting("release.driver.auto_force", "true")
	h.Setting("orders.max_sources", "1")
	expect("السقفُ واحد", 1, true)
}

// TestMultiStop_MaxSourcesSettingWaitsForDriverBuild **ولا يُحفظ سقفٌ لا يقع أثرُه.**
//
// **كان يُحفظ فلا يتبدّل شيءٌ في السلّة** — **فيرفع المالكُ السقفَ ويبقى
// «متجرٌ واحد» ولا يعرف لماذا.** **فالردُّ يقول السبب** (`multistop_driver_not_forced`)،
// **والإنزالُ إلى الواحد يُحفظ دائماً.**
func TestMultiStop_MaxSourcesSettingWaitsForDriverBuild(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	h.Setting("orders.max_sources", "1")
	h.Setting("orders.multistop_driver_build", "22")
	h.Setting("app.min_version.driver", "21")

	put := func(v int) Res {
		t.Helper()
		return h.Call("PUT", "/api/v1/admin/settings/orders.max_sources",
			admin.Token, map[string]any{"value": v}, nil)
	}
	stored := func() string {
		t.Helper()
		var v string
		if err := h.Pool.QueryRow(ctxBG(),
			`SELECT value::text FROM app_settings WHERE key = 'orders.max_sources'`).Scan(&v); err != nil {
			t.Fatalf("قراءةُ السقف: %v", err)
		}
		return v
	}

	if r := put(2); r.Code != http.StatusConflict || r.Err() != "multistop_driver_not_forced" {
		t.Fatalf("**سقفٌ فوق الواحد حُفظ والنسخةُ لم تُفرض**: %s — يُنتظر 409 multistop_driver_not_forced", r)
	}
	if got := stored(); got != "1" {
		t.Fatalf("**الرفضُ كتب**: السقفُ المحفوظ %s — يُنتظر 1", got)
	}

	// **وبعد الفرض يُحفظ.**
	h.Setting("app.min_version.driver", "22")
	if r := put(2); r.Code >= 400 {
		t.Fatalf("**النسخةُ مفروضةٌ والسقفُ رُدّ**: %s", r)
	}
	if got := stored(); got != "2" {
		t.Fatalf("السقفُ المحفوظ %s — يُنتظر 2", got)
	}

	// **والإنزالُ إلى الواحد لا يُمنع أبداً** — ولو أُغلقت البوّابة.
	h.Setting("app.min_version.driver", "21")
	if r := put(1); r.Code >= 400 {
		t.Fatalf("**إنزالُ السقف إلى الواحد رُدّ**: %s", r)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **صفوفُ الأقسام عند الإنشاء** — `T1`
// ══════════════════════════════════════════════════════════════════════

// partRow **صفُّ قسمٍ كما في القاعدة.**
type partRow struct {
	MerchantID string
	Seq        int
	Status     string
}

// partsOf **عددُ محطّات الطلب وأقسامُه بترتيبها.**
func partsOf(t *testing.T, h *Harness, orderID string) (stops int, parts []partRow) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT stops FROM orders WHERE id = $1::uuid`, orderID).Scan(&stops); err != nil {
		t.Fatalf("قراءةُ عدد المحطّات: %v", err)
	}
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT merchant_id::text, seq, status FROM order_parts
		WHERE order_id = $1::uuid ORDER BY seq`, orderID)
	if err != nil {
		t.Fatalf("قراءةُ الأقسام: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p partRow
		if err := rows.Scan(&p.MerchantID, &p.Seq, &p.Status); err != nil {
			t.Fatalf("قراءةُ قسم: %v", err)
		}
		parts = append(parts, p)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("قراءةُ الأقسام: %v", err)
	}
	return stops, parts
}

// TestMultiStop_PartsCreatedOnlyForTwoStores **صفٌّ لكلّ متجرٍ في الطلب من متجرين فأكثر
// وحدَه** — `T1`.
//
// **والطلبُ من متجرٍ واحدٍ بلا صفٍّ و`stops = 1`** — **ومسارُه اليومَ لا يتبدّل.**
// **و`seq = 1` هو متجرُ الطلب** (`orders.merchant_id`): أوّلُ ما في السلّة، **أو
// ما أُرسل من مصادرها** — ثمّ البقيّةُ بترتيب السلّة.
func TestMultiStop_PartsCreatedOnlyForTwoStores(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.OpenMultiStop()
	h.Setting("orders.max_sources", "3")
	h.Setting("orders.max_open_per_customer", "10")
	u := h.Customer()
	a := h.NewItem(1000)
	b := h.NewItem(700)
	c := h.NewItem(500)

	create := func(label string, body map[string]any) string {
		t.Helper()
		r := h.POSTKey("/api/v1/orders", u.Token, uniq("ms-parts"), body)
		if r.Code >= 400 {
			t.Fatalf("%s: الإنشاءُ رُدّ: %s", label, r)
		}
		id, _ := r.JSON()["id"].(string)
		if id == "" {
			t.Fatalf("%s: لا معرّفَ في الردّ: %s", label, r)
		}
		return id
	}
	want := func(label, orderID string, stops int, merchants ...string) {
		t.Helper()
		gotStops, parts := partsOf(t, h, orderID)
		if gotStops != stops {
			t.Errorf("%s: stops = %d — يُنتظر %d", label, gotStops, stops)
		}
		if len(parts) != len(merchants) {
			t.Fatalf("%s: %d قسماً — يُنتظر %d: %+v", label, len(parts), len(merchants), parts)
		}
		for i, p := range parts {
			if p.MerchantID != merchants[i] || p.Seq != i+1 || p.Status != "pending" {
				t.Errorf("%s: القسمُ %d = %+v — يُنتظر متجر %s بالترتيب %d «pending»",
					label, i+1, p, merchants[i], i+1)
			}
		}
		if len(merchants) == 0 {
			return
		}
		var primary string
		if err := h.Pool.QueryRow(ctxBG(),
			`SELECT merchant_id::text FROM orders WHERE id = $1::uuid`, orderID).Scan(&primary); err != nil {
			t.Fatalf("%s: متجرُ الطلب: %v", label, err)
		}
		if primary != merchants[0] {
			t.Errorf("%s: **القسمُ الأوّلُ غيرُ متجر الطلب** — %s و%s", label, merchants[0], primary)
		}
	}

	// ١ · متجرٌ واحد ⇒ لا صفّ.
	want("متجرٌ واحد", create("متجرٌ واحد", orderBody(a, 2)), 1)

	// ٢ · صنفان من متجرٍ واحد ⇒ لا صفّ — **المصدرُ متجرٌ لا صنف.**
	a2 := h.NewItemFor(&Merchant{ID: a.MerchantID}, 300)
	want("صنفان من متجرٍ واحد", create("صنفان من متجرٍ واحد", twoStoreBody(a, a2)), 1)

	// ٣ · متجران — ب أوّلاً في السلّة ⇒ ب ثمّ أ.
	want("متجران", create("متجران", twoStoreBody(b, a)), 2, b.MerchantID, a.MerchantID)

	// ٤ · ثلاثةُ متاجر بترتيب السلّة.
	want("ثلاثة", create("ثلاثة", twoStoreBody(c, a, b)), 3,
		c.MerchantID, a.MerchantID, b.MerchantID)

	// ٥ · متجرٌ مُرسَلٌ من مصادر السلّة ⇒ هو الأوّل، والبقيّةُ بترتيبها.
	sent := twoStoreBody(b, c, a)
	sent["merchant_id"] = a.MerchantID
	want("متجرٌ مُرسَل", create("متجرٌ مُرسَل", sent), 3,
		a.MerchantID, b.MerchantID, c.MerchantID)
}

// TestMultiStop_EverySourceMustBeOpen **كلُّ متاجر الطلب مفتوحة — لا الأوّلُ وحدَه** — `G-13`.
//
// **كان الإنشاءُ يفحص متجرَ الطلب وحدَه** — **فنداءٌ يُصاغ باليد (أو سلّةٌ فُتحت
// قبل الإغلاق) يُدخل متجراً ثانياً مغلقاً**، فيصل الطلبُ ولا أحدَ يحضّر قسمَه.
// **والسلّةُ كانت تعرف** (`closedSource`) **والإنشاءُ لا يسأل.**
func TestMultiStop_EverySourceMustBeOpen(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.OpenMultiStop()
	h.Setting("orders.max_sources", "2")
	h.Setting("orders.max_open_per_customer", "10")
	u := h.Customer()
	a := h.NewItem(1000)
	b := h.NewItem(700)

	refused := func(label string) {
		t.Helper()
		before := h.CountOrders(u.ID)
		r := h.POSTKey("/api/v1/orders", u.Token, uniq("ms-open"), twoStoreBody(a, b))
		if r.Code != http.StatusConflict || r.Err() != "merchant_closed" {
			t.Errorf("%s: **طلبٌ بمتجرٍ ثانٍ مغلقٍ قُبل**: %s — يُنتظر 409 merchant_closed", label, r)
		}
		if got := h.CountOrders(u.ID) - before; got != 0 {
			t.Errorf("%s: **أُنشئ %d طلباً** والرفضُ لا يكتب شيئاً", label, got)
		}
	}

	// ١ · إغلاقٌ طارئٌ للمتجر الثاني.
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE merchants SET emergency_closed = true WHERE id = $1::uuid`, b.MerchantID); err != nil {
		t.Fatal(err)
	}
	refused("إغلاقٌ طارئ")

	// ٢ · دوامُ المتجر الثاني مغلقٌ اليوم — **بالنصّ نفسِه الذي يحكم الأوّل** (`OpenNowSQL`).
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE merchants SET emergency_closed = false WHERE id = $1::uuid;`, b.MerchantID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO merchant_hours (merchant_id, day_of_week, open_time, close_time, closed)
		VALUES ($1::uuid, EXTRACT(dow FROM (now() AT TIME ZONE 'Asia/Damascus'))::int,
		        '00:00', '00:00', true)`, b.MerchantID); err != nil {
		t.Fatal(err)
	}
	refused("دوامٌ مغلق")

	// ٣ · وفُتح ⇒ يُقبل.
	if _, err := h.Pool.Exec(ctxBG(),
		`DELETE FROM merchant_hours WHERE merchant_id = $1::uuid`, b.MerchantID); err != nil {
		t.Fatal(err)
	}
	if r := h.POSTKey("/api/v1/orders", u.Token, uniq("ms-open"), twoStoreBody(a, b)); r.Code >= 400 {
		t.Fatalf("**المتجران مفتوحان والطلبُ رُدّ**: %s", r)
	}
}

// TestMultiStop_StoreWithAPartIsNotDeleted **متجرٌ له قسمٌ في طلبٍ لا يُحذف** —
// **يُردّ ٤٠٩ «له طلبات» لا ٥٠٠.**
//
// **وقسمُه يبقى ولو خرجت بنودُه كلُّها من الطلب** (صنفٌ غيرُ متوفّر) — **فهو تاريخُ
// المتجر.** **وبلا هذا الفحص يصطدم الحذفُ بمفتاح القسم الأجنبيّ فيردّ ٥٠٠**،
// **وكان قبل الأقسام يُحذف وطلبُه يذكره.**
func TestMultiStop_StoreWithAPartIsNotDeleted(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.OpenMultiStop()
	h.Setting("orders.max_sources", "2")
	admin := h.NewUser("admin")
	u := h.Customer()
	a := h.NewItem(1000)
	b := h.NewItem(700)

	made := h.POSTKey("/api/v1/orders", u.Token, uniq("ms-del"), twoStoreBody(a, b))
	if made.Code >= 400 {
		t.Fatalf("الإنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	// **بندُ المتجر الثاني يخرج** — صنفٌ غيرُ متوفّرٍ بيد المكتب.
	var line string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT id::text FROM order_items WHERE order_id = $1::uuid AND merchant_id = $2::uuid`,
		oid, b.MerchantID).Scan(&line); err != nil {
		t.Fatalf("بندُ المتجر الثاني: %v", err)
	}
	if r := h.POST("/api/v1/admin/orders/"+oid+"/items/"+line+"/unavailable", admin.Token,
		map[string]any{}); r.Code >= 400 {
		t.Fatalf("صنفٌ غيرُ متوفّر: %s", r)
	}

	r := h.DEL("/api/v1/admin/merchants/"+b.MerchantID, admin.Token)
	if r.Code != http.StatusConflict || r.Err() != "merchant_has_orders" {
		t.Fatalf("**حذفُ متجرٍ له قسمٌ في طلب**: %s — يُنتظر 409 merchant_has_orders", r)
	}
}

// TestXG46_MultiStopCreateNeedsOneConnection **والطلبُ من متجرين يكفيه وصلةٌ واحدة** — `XG-46`.
//
// **وبوّابةُ نسخة السائق تُسأل داخلَ معاملة الإنشاء** (`maxSources`)، **وصفوفُ
// الأقسام تُكتب فيها** — **فقراءةٌ واحدةٌ من المَسبَح هناك تُجمّد الطلبَ حين
// يمتلئ.** **ومَسبَحٌ سقفُه وصلةٌ واحدةٌ يقطع الشكّ** (`oneConnHarness`).
func TestXG46_MultiStopCreateNeedsOneConnection(t *testing.T) {
	h := oneConnHarness(t, 1)
	treasury(t, h)
	h.OpenMultiStop()
	h.Setting("orders.max_sources", "2")
	cust := h.Customer()
	a := h.NewItem(1000)
	b := h.NewItem(700)

	done := make(chan Res, 1)
	go func() {
		done <- h.POSTKey("/api/v1/orders", cust.Token, uniq("xg46ms"), twoStoreBody(a, b))
	}()
	select {
	case r := <-done:
		if r.Code >= 400 {
			t.Fatalf("**الطلبُ من متجرين رُدّ على وصلةٍ واحدة**: %s", r)
		}
		oid, _ := r.JSON()["id"].(string)
		if stops, parts := partsOf(t, h, oid); stops != 2 || len(parts) != 2 {
			t.Fatalf("stops = %d وأقسامٌ %d — يُنتظر 2 و2", stops, len(parts))
		}
	case <-time.After(20 * time.Second):
		st := h.Pool.Stat()
		t.Fatalf("**جمد إنشاءُ الطلب من متجرين على وصلةٍ واحدة** — قراءةٌ من المَسبَح "+
			"داخلَ المعاملة (محجوزٌ=%d خاملٌ=%d) (`XG-46`)", st.AcquiredConns(), st.IdleConns())
	}
}
