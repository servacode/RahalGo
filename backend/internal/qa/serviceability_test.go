package qa

import (
	"net/http"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حدُّ التوصيل — ونقطةُ التسليم وحدَها تقرّر** (`SRV`)
// ══════════════════════════════════════════════════════════════════════
//
// # العقد
//
// **وإحداثيّةُ العنوان المختارِ لهذا الطلب هي الحاكمة** — **لا موضعُ
// الجهاز، ولا مدينةُ الحساب، ولا مدينةُ المتجر، ولا نتيجةُ تصفّحٍ
// قديمة.**
//
// # وما يُقاس هنا
//
//	١ · داخلٌ بيّنٌ      ⇒ يُقبَل
//	٢ · خارجٌ بيّنٌ      ⇒ يُردّ
//	٣ · على الحدِّ       ⇒ حكمٌ واحدٌ لا يتبدّل
//	٤ · منطقةٌ مُطفأةٌ   ⇒ تُردّ
//	٥ · بلا إحداثيّة     ⇒ تُردّ
//	٦ · إحداثيّةٌ مشوَّهةٌ ⇒ تُردّ برسالتها لا برسالة التغطية
//	٧ · نداءٌ مباشرٌ خارجَ التغطية ⇒ يُردّ
//	٨ · وعميلٌ قديمٌ لا يلتفّ — الحكمُ يُعاد في المعاملة
//	٩ · وتبديلُ العنوان يُعيد الحكم
//	١٠ · والمردودُ لا يُنشئ صفّاً
//	١١ · والمخصَّصُ يتبع القاعدةَ نفسَها
//	١٢ · وبابُ الإطلاق مغلقٌ ⇒ ردُّ إطلاقٍ ولو كان داخلَ التغطية
//	١٣ · ومفتوحٌ وخارجَ التغطية ⇒ ردُّ تغطية — **والرمزان مفترقان**
//	١٤ · والتسعيرةُ تقرأ النقطةَ نفسَها
//	١٥ · والمردودُ لا يبلغ متجراً ولا سائقاً
//	١٦ · والصحيحُ داخلَ التغطية يبقى يعمل

// srvPoint نقطةُ الرقّة — مركزُ العمل.
const (
	srvLat = 35.9500
	srvLng = 39.0100
)

// srvOrderBody جسمُ طلبٍ عاديٍّ بنقطةٍ بعينها.
//
// **وعلى `orderBody` القائمة** — **ومُعامَلٌ يُكتب ثانيةً يفترق عن
// العقد يومَ يُبدَّل حقل.** (وقع: كُتب `menu_item_id` بـ`item_id`
// فرُدَّ الطلبُ بـ`invalid_items` **قبل أن يبلغ فحصَ التغطية**، فقُرئ
// قبولاً وهو رفضٌ لسببٍ آخر.)
func srvOrderBody(t *testing.T, hh *Harness, lat, lng float64) map[string]any {
	t.Helper()
	b := orderBody(hh.NewItem(1000), 1)
	b["lat"] = lat
	b["lng"] = lng
	return b
}

func srvCustomBody(lat, lng float64) map[string]any {
	return map[string]any{
		"request": "قياسُ التغطية", "address_text": "الرقة",
		"lat": lat, "lng": lng, "payment": "cash",
	}
}

// srvOpenAllLaunch **يفتح أبوابَ الإطلاق** — فلا يُخلَط ردٌّ بردّ.
func srvOpenAllLaunch(hh *Harness) {
	for _, k := range []string{
		"launch.customer_orders", "launch.customer_custom_orders",
		"launch.merchant_orders", "launch.customer_browse",
	} {
		hh.Setting(k, "true")
	}
}

// countOrders عددُ طلبات زبونٍ — **لِيُقاس أنّ المردودَ لا يُنشئ صفّاً.**
func countOrders(t *testing.T, hh *Harness, customerID string) int {
	t.Helper()
	var n int
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM orders WHERE customer_id = $1`, customerID).Scan(&n); err != nil {
		t.Fatalf("عدُّ الطلبات: %v", err)
	}
	return n
}

// TestSRV1_InsideAndOutsideAndBoundary **الثلاثةُ الأولى معاً.**
//
// **ودائرةٌ واحدةٌ بنصفِ قطرٍ معلوم** — **فالحكمُ يُقاس بالمتر لا
// بالظنّ.**
func TestSRV1_InsideAndOutsideAndBoundary(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	isolateZones(t, hh)
	makeRadiusZone(t, hh, srvLat, srvLng, 1000)

	u, tok := capUser(t, hh, "customer")

	// ── ١ · داخلٌ بيّن ───────────────────────────────────────────
	r := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, srvLat, srvLng))
	if r.Err() == "out_of_zone" {
		t.Errorf("**نقطةُ المركز رُدّت خارجَ التغطية** — والدائرةُ حولها.")
	}

	// ── ٢ · خارجٌ بيّن — دمشق، ٤٠٠ كم ───────────────────────────
	before := countOrders(t, hh, u.ID)
	out := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, 33.5138, 36.2765))
	if out.Err() != "out_of_zone" {
		t.Errorf("**نقطةٌ في دمشق قُبلت والدائرةُ في الرقّة**: %d / %s",
			out.Code, out.Err())
	}
	if after := countOrders(t, hh, u.ID); after != before {
		t.Errorf("**طلبٌ مردودٌ أنشأ صفّاً**: %d ⇒ %d", before, after)
	}

	// ── ٣ · على الحدّ — حكمٌ واحدٌ لا يتبدّل ─────────────────────
	//
	// **و`ST_DWithin` على `geography` تشمل الحدَّ** (`<=`) — **فالنقطةُ
	// على المحيط داخل.** **والمقيسُ ثباتُ الحكم لا اتّجاهُه.**
	//
	// **والهامشُ واسعٌ عمداً**: **درجةُ عرضٍ تساوي ١١٠٩٤٦ متراً عند
	// خطِّ ٣٦ لا ١١١٣٢٠** — **فهامشٌ ضيّقٌ يجعل «١٠٠١ متراً» ٩٩٨
	// متراً فعليّاً**، فيُقرأ العطبُ في المنتَج وهو في حسابي. (وقع
	// ٢٠٢٦-٠٩-١٣.)
	//
	// **وزبونٌ جديدٌ لكلّ نداء** — **وسقفُ الطلبات المفتوحة يُلوّث
	// القياسَ بعد ثالثِ طلبٍ ناجح**، فيصير الجوابُ
	// `too_many_open_orders` لا حكمَ تغطية.
	const degPerM = 1.0 / 110946.0
	for _, c := range []struct {
		m      float64
		inside bool
	}{{500, true}, {900, true}, {1500, false}, {3000, false}} {
		_, tk := capUser(t, hh, "customer")
		e := hh.POST("/api/v1/orders", tk,
			srvOrderBody(t, hh, srvLat+c.m*degPerM, srvLng)).Err()
		if c.inside && e == "out_of_zone" {
			t.Errorf("**%.0f متراً رُدّت وهي داخلَ ألف**", c.m)
		}
		if !c.inside && e != "out_of_zone" {
			t.Errorf("**%.0f متراً قُبلت وهي خارجَ ألف**: %q", c.m, e)
		}
	}

	// **وعلى الحدِّ نفسِه: ثلاثُ محاولاتٍ جوابُها واحد.**
	first := ""
	for i := 0; i < 3; i++ {
		_, tk := capUser(t, hh, "customer")
		e := hh.POST("/api/v1/orders", tk,
			srvOrderBody(t, hh, srvLat+1000*degPerM, srvLng)).Err()
		if i == 0 {
			first = e
		} else if e != first {
			t.Errorf("**حكمُ الحدِّ يتبدّل**: %q ثمّ %q", first, e)
		}
	}
	t.Logf("✓ حكمُ الحدِّ ثابتٌ ثلاثَ مرّات: %q", first)
}

// TestSRV2_DisabledZoneRejects **ومنطقةٌ مُطفأةٌ لا تُغطّي.**
//
// **وإطفاءُ منطقةٍ قرارُ مالكٍ** — **ولو بقيت تُغطّي لَما كان للإطفاء
// معنى.**
func TestSRV2_DisabledZoneRejects(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	isolateZones(t, hh)
	id := makeRadiusZone(t, hh, srvLat, srvLng, 1000)
	_, tok := capUser(t, hh, "customer")

	if r := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, srvLat, srvLng)); r.Err() == "out_of_zone" {
		t.Fatalf("رُدَّ وهو داخلَ منطقةٍ فعّالة")
	}
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE delivery_zones SET active = false WHERE id = $1::uuid`, id); err != nil {
		t.Fatalf("إطفاءُ المنطقة: %v", err)
	}
	// **ولا منطقةَ فعّالةً في النظام الآن** — **والجدولُ الفارغُ يُفتح
	// بقرار المالك ٢٠٢٦-٠٨-١٨**، فتُرسَم ثانيةٌ بعيدةٌ ليعمل الحدّ.
	makeRadiusZone(t, hh, 33.5138, 36.2765, 1000)
	if r := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, srvLat, srvLng)); r.Err() != "out_of_zone" {
		t.Errorf("**منطقةٌ مُطفأةٌ ما زالت تُغطّي**: %d / %s", r.Code, r.Err())
	}
}

// TestSRV3_MissingAndMalformedPoint **ونقطةٌ ليست نقطةً تُردّ برسالتها.**
//
// **و«خارج نطاق التوصيل» جوابٌ خاطئٌ لمن لم يُحدَّد موضعُه** — **فيبحث
// عن العلّة في التغطية وهي في دبّوسه.**
func TestSRV3_MissingAndMalformedPoint(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	isolateZones(t, hh)
	makeRadiusZone(t, hh, srvLat, srvLng, 50000)
	u, tok := capUser(t, hh, "customer")
	before := countOrders(t, hh, u.ID)

	for _, c := range []struct {
		lat, lng float64
		why      string
	}{
		{0, 0, "صفرٌ صفرٌ — لم يُحدَّد موضعُه"},
		{91, 39.01, "خطُّ عرضٍ فوق التسعين"},
		{-91, 39.01, "خطُّ عرضٍ تحت ناقصِ تسعين"},
		{35.95, 181, "خطُّ طولٍ فوق المئة وثمانين"},
		{35.95, -181, "خطُّ طولٍ تحت ناقصِها"},
	} {
		b := srvOrderBody(t, hh, c.lat, c.lng)
		r := hh.POST("/api/v1/orders", tok, b)
		if r.Err() != "bad_point" {
			t.Errorf("**%s لم تُردَّ برسالتها**: %d / %s", c.why, r.Code, r.Err())
		}
		rc := hh.POST("/api/v1/orders/custom", tok, srvCustomBody(c.lat, c.lng))
		if rc.Err() != "bad_point" {
			t.Errorf("**%s في المخصَّص**: %d / %s", c.why, rc.Code, rc.Err())
		}
	}
	if after := countOrders(t, hh, u.ID); after != before {
		t.Errorf("**نقطةٌ مشوَّهةٌ أنشأت صفّاً**: %d ⇒ %d", before, after)
	}
}

// TestSRV4_CustomOrderFollowsCoverage **والمخصَّصُ يتبع القاعدةَ نفسَها.**
//
// **وكان بلا فحصٍ إطلاقاً** — صفرُ ذكرٍ لـ`ZoneAt` في ملفّه (قِيس
// ٢٠٢٦-٠٩-١٣) — **فيُقبَل طلبٌ إلى أيّ نقطةٍ في العالم.**
func TestSRV4_CustomOrderFollowsCoverage(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	isolateZones(t, hh)
	makeRadiusZone(t, hh, srvLat, srvLng, 1000)
	u, tok := capUser(t, hh, "customer")

	if r := hh.POST("/api/v1/orders/custom", tok, srvCustomBody(srvLat, srvLng)); r.Code >= 400 {
		t.Errorf("**مخصَّصٌ داخلَ التغطية رُدّ**: %d / %s", r.Code, r.Err())
	}
	before := countOrders(t, hh, u.ID)
	// ── ونقطةٌ في تركيا ─────────────────────────────────────────
	out := hh.POST("/api/v1/orders/custom", tok, srvCustomBody(41.0082, 28.9784))
	if out.Err() != "out_of_zone" {
		t.Errorf("**مخصَّصٌ إلى إستنبول قُبل**: %d / %s", out.Code, out.Err())
	}
	if after := countOrders(t, hh, u.ID); after != before {
		t.Errorf("**مخصَّصٌ مردودٌ أنشأ صفّاً**: %d ⇒ %d", before, after)
	}
}

// TestSRV5_LaunchAndCoverageAreDistinct **والردّان مفترقان.**
//
// **ولا يُدمَجان**: **«لم نفتح بعد» غيرُ «لا نُوصّل إلى هنا»** — **ومن
// خلطهما أخبر زبوناً داخلَ التغطية أنّه خارجَها.**
func TestSRV5_LaunchAndCoverageAreDistinct(t *testing.T) {
	hh := New(t)
	isolateZones(t, hh)
	makeRadiusZone(t, hh, srvLat, srvLng, 1000)
	_, tok := capUser(t, hh, "customer")

	// ── ١٢ · بابٌ مغلقٌ وداخلَ التغطية ⇒ ردُّ إطلاق ─────────────
	srvOpenAllLaunch(hh)
	hh.Setting("launch.customer_orders", "false")
	r := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, srvLat, srvLng))
	if r.Err() != "launch_closed" {
		t.Errorf("**بابٌ مغلقٌ وداخلَ التغطية لم يردّ ردَّ الإطلاق**: %d / %s",
			r.Code, r.Err())
	}

	// ── ١٣ · بابٌ مفتوحٌ وخارجَ التغطية ⇒ ردُّ تغطية ────────────
	srvOpenAllLaunch(hh)
	r2 := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, 33.5138, 36.2765))
	if r2.Err() != "out_of_zone" {
		t.Errorf("**بابٌ مفتوحٌ وخارجَ التغطية لم يردّ ردَّ التغطية**: %d / %s",
			r2.Code, r2.Err())
	}
	if r.Err() == r2.Err() {
		t.Error("**الردّان متطابقان** — ولا يُفرَّق بين «ليس الآن» و«ليس هنا».")
	}
	t.Logf("✓ مفترقان: إطلاقٌ=%q · تغطيةٌ=%q", r.Err(), r2.Err())
}

// TestSRV6_RejectedOrderLeavesNoTrace **والمردودُ لا يترك أثراً.**
//
// **ولا صفَّ طلبٍ · ولا قيدَ مالٍ · ولا حجزَ محفظةٍ · ولا بثَّ حدثٍ ·
// ولا إشعارَ متجرٍ أو سائق.**
func TestSRV6_RejectedOrderLeavesNoTrace(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	isolateZones(t, hh)
	makeRadiusZone(t, hh, srvLat, srvLng, 1000)
	u, tok := capUser(t, hh, "customer")

	base := financialBaseline(t, hh)
	beforeOrders := countOrders(t, hh, u.ID)

	var beforeEvents int
	_ = hh.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM outbox`).Scan(&beforeEvents)

	// ── خارجَ التغطية ───────────────────────────────────────────
	if r := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, 33.5138, 36.2765)); r.Err() != "out_of_zone" {
		t.Fatalf("لم يُردّ: %d / %s", r.Code, r.Err())
	}
	if r := hh.POST("/api/v1/orders/custom", tok, srvCustomBody(33.5138, 36.2765)); r.Err() != "out_of_zone" {
		t.Fatalf("المخصَّصُ لم يُردّ: %d / %s", r.Code, r.Err())
	}

	if n := countOrders(t, hh, u.ID); n != beforeOrders {
		t.Errorf("**صفوفُ طلباتٍ أُنشئت لمردود**: %d ⇒ %d", beforeOrders, n)
	}
	// **ولا خرقَ ماليٌّ جديد** — والمقارنةُ بخطِّ الأساس لا بصفر.
	assertNewViolations(t, hh, base)

	var afterEvents int
	_ = hh.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM outbox`).Scan(&afterEvents)
	if afterEvents != beforeEvents {
		t.Errorf("**أحداثٌ خرجت لطلبٍ مردود**: %d ⇒ %d — "+
			"**ولا متجرَ ولا سائقَ يُبلَّغ بما لم يُقبَل.**", beforeEvents, afterEvents)
	}
}

// TestSRV7_AddressChangeIsRevalidated **وتبديلُ العنوان يُعيد الحكم.**
//
// **وسلّةٌ بُنيت على عنوانٍ داخلَ التغطية ثمّ بُدّل العنوان** — **لا
// تُسلَّم إلى الإحداثيّة القديمة.**
func TestSRV7_AddressChangeIsRevalidated(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	isolateZones(t, hh)
	makeRadiusZone(t, hh, srvLat, srvLng, 1000)
	u, tok := capUser(t, hh, "customer")

	// **والطلبُ الأوّلُ بنقطةٍ داخل** — فالسلّةُ صالحةٌ حينها.
	if r := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, srvLat, srvLng)); r.Err() == "out_of_zone" {
		t.Fatal("الأوّلُ رُدّ وهو داخل")
	}
	before := countOrders(t, hh, u.ID)

	// **وبالسلّة نفسِها إلى نقطةٍ خارج** — **يُردّ ولا يُسلَّم إلى
	// القديمة.**
	out := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, 33.5138, 36.2765))
	if out.Err() != "out_of_zone" {
		t.Errorf("**العنوانُ الجديدُ خارجَ التغطية ومرّ**: %d / %s", out.Code, out.Err())
	}
	if n := countOrders(t, hh, u.ID); n != before {
		t.Errorf("**طلبٌ أُنشئ بعد تبديلٍ إلى خارج**: %d ⇒ %d", before, n)
	}
}

// TestSRV8_PolygonZoneIsHonoured **والمضلَّعُ يُقرأ كما تُقرأ الدائرة.**
//
// **وقِيس ٢٠٢٦-٠٩-١٣: `catalog.ZoneForPoint` كانت تفحص الدائرةَ
// وحدَها** — **فلو نُوديت لَقالت «خارجَ التغطية» لنقطةٍ داخلَ مضلَّعٍ
// مرسوم.** (حُذفت.)
func TestSRV8_PolygonZoneIsHonoured(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	isolateZones(t, hh)
	// **مربَّعٌ نصفُ ضلعه ٠٫٠٢ درجة** — نحوَ ٢٫٢ كم.
	makePolygonZone(t, hh, srvLat, srvLng, 0.02)
	_, tok := capUser(t, hh, "customer")

	if r := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, srvLat, srvLng)); r.Err() == "out_of_zone" {
		t.Errorf("**نقطةٌ في قلب المضلَّع رُدّت** — والمضلَّعُ لا يُقرأ.")
	}
	// **وخارجَه بيّناً** — على بعدِ ضلعين.
	if r := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, srvLat+0.05, srvLng)); r.Err() != "out_of_zone" {
		t.Errorf("**نقطةٌ خارجَ المضلَّع قُبلت**: %s", r.Err())
	}
}

// TestSRV9_PricingReadsTheSamePoint **والتسعيرةُ تقرأ النقطةَ نفسَها.**
//
// **ولو حُسبت بنقطتين لَقالت السلّةُ رقماً ويُحاسَب الزبونُ بغيره** —
// **وهو أسوأُ ما يقع في شاشة دفع.**
//
// **و`DeliveryAt` تنادي `ZoneAt` نفسَها** — فالمصدرُ واحدٌ بنيويّاً،
// **والمقيسُ أنّ بابَ التسعيرة يردّ ما يردّه بابُ الإنشاء.**
func TestSRV9_PricingReadsTheSamePoint(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	isolateZones(t, hh)
	makeRadiusZone(t, hh, srvLat, srvLng, 1000)

	for _, c := range []struct {
		lat, lng float64
		want     string
		why      string
	}{
		{srvLat, srvLng, "", "المركز — يُوصَّل"},
		{33.5138, 36.2765, "out_of_zone", "دمشق — خارجَ التغطية"},
		{0, 0, "bad_point", "صفرٌ صفرٌ — لم يُحدَّد الموضع"},
	} {
		// **وبعقدِ البابِ نفسِه** — `items` و`lat` و`lng`. **وجسمٌ
		// ناقصٌ يُردّ `validation` قبل أن يبلغ التغطية**، فيُقرأ
		// اختلافاً وهو نقصُ مُعامَل. (وقع ٢٠٢٦-٠٩-١٣.)
		it := hh.NewItem(1000)
		q := hh.POST("/api/v1/public/quote", "", map[string]any{
			"items": []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
			"lat":   c.lat, "lng": c.lng,
		})
		if q.Code != http.StatusOK {
			t.Errorf("**التسعيرةُ سقطت على %s** — **وسلّةٌ تنهار لأنّ "+
				"الدبّوسَ لم يُوضع سلّةٌ لا تُستعمل**: %d / %s",
				c.why, q.Code, q.Err())
			continue
		}
		body := string(q.Body)
		// ── وهي تقول حكمَها صراحةً ────────────────────────────────
		wantServiceable := c.want == ""
		gotServiceable := strings.Contains(body, `"serviceable":true`)
		if gotServiceable != wantServiceable {
			t.Errorf("**التسعيرةُ تقول serviceable=%v على %s** — والمنتظَرُ %v · %s",
				gotServiceable, c.why, wantServiceable, body)
		}
		if c.want != "" && !strings.Contains(body, c.want) {
			t.Errorf("**التسعيرةُ لا تقول السبب** على %s — المنتظَرُ %q", c.why, c.want)
		}

		// ── والإنشاءُ يوافقها ────────────────────────────────────
		_, tk := capUser(t, hh, "customer")
		o := hh.POST("/api/v1/orders", tk, srvOrderBody(t, hh, c.lat, c.lng))
		created := o.Code == http.StatusCreated || o.Code == http.StatusOK
		if created != wantServiceable {
			t.Errorf("**التسعيرةُ والإنشاءُ يختلفان على %s**: تسعيرةٌ=%v · إنشاءٌ=%q",
				c.why, wantServiceable, o.Err())
		}
		if c.want != "" && o.Err() != c.want {
			t.Errorf("**الإنشاءُ ردّ %q على %s والمنتظَرُ %q**", o.Err(), c.why, c.want)
		}
	}
}

// TestSRV10_ValidOrderStillWorks **والصحيحُ يبقى يعمل.**
//
// **ومنعٌ يمنع الصحيحَ أسوأُ من لا منع** — فيُقاس القبولُ لا الردُّ وحدَه.
func TestSRV10_ValidOrderStillWorks(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	isolateZones(t, hh)
	makeRadiusZone(t, hh, srvLat, srvLng, 50000)
	u, tok := capUser(t, hh, "customer")
	before := countOrders(t, hh, u.ID)

	r := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, srvLat, srvLng))
	if r.Code != http.StatusCreated && r.Code != http.StatusOK {
		t.Fatalf("**طلبٌ صحيحٌ داخلَ التغطية رُدّ**: %d / %s · %s",
			r.Code, r.Err(), string(r.Body))
	}
	if n := countOrders(t, hh, u.ID); n != before+1 {
		t.Errorf("**الطلبُ الصحيحُ لم يُنشئ صفّاً**: %d ⇒ %d", before, n)
	}
	rc := hh.POST("/api/v1/orders/custom", tok, srvCustomBody(srvLat, srvLng))
	if rc.Code >= 400 {
		t.Errorf("**مخصَّصٌ صحيحٌ رُدّ**: %d / %s", rc.Code, rc.Err())
	}
}
