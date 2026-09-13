package qa

import (
	"net/http"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **لا تغطيةَ صالحةً ⇒ لا قبولَ طلب** (`CFC`، ٢٠٢٦-٠٩-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// # نقضُ المالك
//
// **وكان الجدولُ الفارغُ يُقرأ بابًا مفتوحاً** (قرارُ ٢٠٢٦-٠٨-١٨):
// «لم تُرسم خريطةٌ بعد» حالُ إعدادٍ لا قرارُ سياسة.
//
// **وهو غيرُ مقبولٍ لإطلاقٍ عامّ** (٢٠٢٦-٠٩-١٣): **إعدادُ تغطيةٍ غائبٌ
// أو مُطفأٌ أو معطوبٌ لا يفتح العالمَ لقبولِ الطلبات.**
//
// # والحالاتُ الأربعُ مفترقةٌ في الرمز
//
//	launch_closed          بابُ الإطلاق مغلق
//	coverage_unavailable   لا إعدادَ تغطيةٍ صالحاً — **حالُ إعداد**
//	out_of_zone            تغطيةٌ صالحةٌ والنقطةُ خارجَها — **حكمٌ جغرافيّ**
//	bad_point              إحداثيّةٌ ليست إحداثيّة
//
// **ومن جمعها أخبر زبوناً في قلب المدينة أنّه خارجَ التغطية** — والعلّةُ
// في اللوحة لا في موضعه.
//
// # وحادثُ اللوحة هو ما يحرسه البندُ الأخير
//
// **ومن أطفأ المناطقَ كلَّها ليُصلح واحدةً لا يجوز أن يفتح العالم.**

// wipeZones **يمحو المناطقَ كلَّها ويُرجعها** — لحالِ «لا جدولَ».
func wipeZones(t *testing.T, hh *Harness) {
	t.Helper()
	type row struct{ id string }
	rows, err := hh.Pool.Query(ctxBG(), `SELECT id::text FROM delivery_zones`)
	if err != nil {
		t.Fatalf("قراءةُ المناطق: %v", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	// **ويُنسَخ الصفُّ كلُّه قبل محوه** — فيُرجَع كما كان.
	if _, err := hh.Pool.Exec(ctxBG(), `
		CREATE TEMP TABLE IF NOT EXISTS cfc_backup AS SELECT * FROM delivery_zones WHERE false`); err != nil {
		t.Fatalf("جدولُ النسخ: %v", err)
	}
	if _, err := hh.Pool.Exec(ctxBG(),
		`INSERT INTO cfc_backup SELECT * FROM delivery_zones`); err != nil {
		t.Fatalf("النسخ: %v", err)
	}
	if _, err := hh.Pool.Exec(ctxBG(), `DELETE FROM delivery_zones`); err != nil {
		t.Fatalf("المحو: %v", err)
	}
	t.Cleanup(func() {
		_, _ = hh.Pool.Exec(ctxBG(), `INSERT INTO delivery_zones SELECT * FROM cfc_backup
			ON CONFLICT (id) DO NOTHING`)
		_, _ = hh.Pool.Exec(ctxBG(), `DROP TABLE IF EXISTS cfc_backup`)
	})
	_ = ids
}

// disableAllZones **يُطفئ الفعّالةَ كلَّها ويُرجعها** — حادثُ اللوحة.
func disableAllZones(t *testing.T, hh *Harness) {
	t.Helper()
	isolateZones(t, hh)
}

// cfcProbe نداءُ إنشاءٍ عاديٍّ ومخصَّصٍ بنقطةٍ — ويعيد رمزَيهما.
func cfcProbe(t *testing.T, hh *Harness, tok string, lat, lng float64) (string, string) {
	t.Helper()
	n := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, lat, lng))
	c := hh.POST("/api/v1/orders/custom", tok, srvCustomBody(lat, lng))
	return n.Err(), c.Err()
}

// TestCFC1_NoZonesAtAllIsClosed **جدولٌ فارغٌ يُغلق ولا يفتح.**
//
// **وهذا نقضُ السلوك القديم بعينه.**
func TestCFC1_NoZonesAtAllIsClosed(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	u, tok := capUser(t, hh, "customer")
	wipeZones(t, hh)

	before := countOrders(t, hh, u.ID)
	n, c := cfcProbe(t, hh, tok, srvLat, srvLng)
	if n != "coverage_unavailable" {
		t.Errorf("**العاديُّ مرّ وجدولُ التغطية فارغ**: %q — "+
			"**ولا تغطيةَ صالحةً ليس «العالمُ كلُّه مُغطّى».**", n)
	}
	if c != "coverage_unavailable" {
		t.Errorf("**المخصَّصُ مرّ وجدولُ التغطية فارغ**: %q", c)
	}
	if after := countOrders(t, hh, u.ID); after != before {
		t.Errorf("**صفٌّ أُنشئ بلا تغطية**: %d ⇒ %d", before, after)
	}
}

// TestCFC2_AllDisabledIsClosed **وإطفاءُ الكلِّ لا يفتح العالم.**
//
// **وهو حادثُ اللوحة**: **من أطفأ المناطقَ ليُصلح واحدةً.**
func TestCFC2_AllDisabledIsClosed(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	u, tok := capUser(t, hh, "customer")
	disableAllZones(t, hh)

	before := countOrders(t, hh, u.ID)
	n, c := cfcProbe(t, hh, tok, srvLat, srvLng)
	if n != "coverage_unavailable" || c != "coverage_unavailable" {
		t.Errorf("**إطفاءُ المناطق كلِّها فتح العالم**: عاديٌّ=%q · مخصَّصٌ=%q", n, c)
	}
	if after := countOrders(t, hh, u.ID); after != before {
		t.Errorf("**صفٌّ أُنشئ بعد إطفاء الكلّ**: %d ⇒ %d", before, after)
	}
}

// TestCFC3_MalformedZonesAreNotCoverage **وصفٌّ فعّالٌ ليس تغطيةً.**
//
// **ودائرةٌ بلا مركزٍ أو بنصفِ قطرٍ صفرٍ لا تُغطّي شيئاً**، **ومضلَّعٌ
// بهندسةٍ فارغةٍ كذلك** — **و«فعّالٌ» ليس «صالحٌ للاستعمال».**
func TestCFC3_MalformedZonesAreNotCoverage(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	u, tok := capUser(t, hh, "customer")
	wipeZones(t, hh)

	for _, c := range []struct {
		sql string
		why string
	}{
		{`INSERT INTO delivery_zones (name, center, radius_m, delivery_fee, min_order, active, shape)
		  VALUES ('CFC بلا مركز', NULL, 5000, 0, 0, true, 'radius')`,
			"دائرةٌ بلا مركز"},
		{`INSERT INTO delivery_zones (name, center, radius_m, delivery_fee, min_order, active, shape)
		  VALUES ('CFC نصفُ قطرٍ صفر',
		          ST_SetSRID(ST_MakePoint(39.01,35.95),4326)::geography, 0, 0, 0, true, 'radius')`,
			"نصفُ قطرٍ صفر"},
		{`INSERT INTO delivery_zones (name, center, radius_m, delivery_fee, min_order, active, shape)
		  VALUES ('CFC نصفُ قطرٍ فارغ',
		          ST_SetSRID(ST_MakePoint(39.01,35.95),4326)::geography, NULL, 0, 0, true, 'radius')`,
			"نصفُ قطرٍ فارغ"},
		{`INSERT INTO delivery_zones (name, center, radius_m, delivery_fee, min_order, active, shape, area)
		  VALUES ('CFC مضلَّعٌ فارغ',
		          ST_SetSRID(ST_MakePoint(39.01,35.95),4326)::geography, 0, 0, 0, true, 'polygon', NULL)`,
			"مضلَّعٌ بلا هندسة"},
		{`INSERT INTO delivery_zones (name, center, radius_m, delivery_fee, min_order, active, shape)
		  VALUES ('CFC شكلٌ مجهول',
		          ST_SetSRID(ST_MakePoint(39.01,35.95),4326)::geography, 5000, 0, 0, true, 'blob')`,
			"شكلٌ مجهول"},
		// ══════════════════════════════════════════════════════════
		// **وهذه وحدَها هي التي يسمح بها المخطَّط** (قِيس ٢٠٢٦-٠٩-١٣)
		// ══════════════════════════════════════════════════════════
		//
		// **وقيودُ القاعدة تمنع أكثرَ العطب**:
		//
		//	center NOT NULL · radius_m NOT NULL
		//	radius_m BETWEEN 100 AND 50000
		//	shape IN ('radius','polygon')
		//	shape <> 'polygon' OR area IS NOT NULL
		//
		// **فالقاعدةُ خطُّ الدفاع الأوّل** — **والحالاتُ فوقُ تُرفض قبل
		// أن تُكتب، ويُسجّلها الاختبارُ ويتجاوزها.**
		//
		// **والباقي الممكنُ الوحيد**: **مضلَّعٌ فعّالٌ بهندسةٍ حاضرةٍ
		// مساحتُها صفر** — **و`ST_Area > 0` هو ما يردّه.** (قِيس على
		// التجهيز: فعّالةٌ=١ · مساحتُها=٠ ⇒ ٥٠٣.)
		{`INSERT INTO delivery_zones (name, center, radius_m, delivery_fee, min_order, active, shape, area)
		  VALUES ('CFC مضلَّعٌ صفريّ',
		          ST_SetSRID(ST_MakePoint(39.01,35.95),4326)::geography, 100, 0, 0, true, 'polygon',
		          ST_SetSRID(ST_GeomFromText(
		            'POLYGON((39.01 35.95, 39.01 35.95, 39.01 35.95, 39.01 35.95))'),4326)::geography)`,
			"مضلَّعٌ صفريُّ المساحة"},
	} {
		if _, err := hh.Pool.Exec(ctxBG(), `DELETE FROM delivery_zones`); err != nil {
			t.Fatalf("تفريغ: %v", err)
		}
		if _, err := hh.Pool.Exec(ctxBG(), c.sql); err != nil {
			t.Logf("تعذّر زرعُ «%s» (قيدٌ في القاعدة): %v — ويُتجاوَز", c.why, err)
			continue
		}
		before := countOrders(t, hh, u.ID)
		n, cu := cfcProbe(t, hh, tok, srvLat, srvLng)
		if n != "coverage_unavailable" || cu != "coverage_unavailable" {
			t.Errorf("**%s قُرئت تغطيةً**: عاديٌّ=%q · مخصَّصٌ=%q — "+
				"**و«فعّالٌ» ليس «صالحٌ للاستعمال».**", c.why, n, cu)
		}
		if after := countOrders(t, hh, u.ID); after != before {
			t.Errorf("**صفٌّ أُنشئ بتغطيةٍ معطوبةٍ (%s)**: %d ⇒ %d", c.why, before, after)
		}
	}
}

// TestCFC4_FourStatesAreDistinct **والأربعةُ مفترقةٌ في الرمز.**
func TestCFC4_FourStatesAreDistinct(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "customer")
	seen := map[string]string{}

	// ── أ · بابُ الإطلاق مغلقٌ — وتغطيةٌ صالحةٌ قائمة ────────────
	onlyZone(t, hh, srvLat, srvLng, 1000)
	srvOpenAllLaunch(hh)
	hh.Setting("launch.customer_orders", "false")
	if e, _ := cfcProbe(t, hh, tok, srvLat, srvLng); e != "launch_closed" {
		t.Errorf("**بابٌ مغلقٌ ردّ %q** — والمنتظَرُ launch_closed", e)
	} else {
		seen["launch"] = e
	}

	// ── ب · مفتوحٌ وتغطيةٌ صالحةٌ وخارجَها ──────────────────────
	srvOpenAllLaunch(hh)
	if e, _ := cfcProbe(t, hh, tok, 33.5138, 36.2765); e != "out_of_zone" {
		t.Errorf("**خارجَ تغطيةٍ صالحةٍ ردّ %q** — والمنتظَرُ out_of_zone", e)
	} else {
		seen["outside"] = e
	}

	// ── ج · إحداثيّةٌ مشوَّهة ────────────────────────────────────
	if e, _ := cfcProbe(t, hh, tok, 0, 0); e != "bad_point" {
		t.Errorf("**نقطةٌ مشوَّهةٌ ردّت %q** — والمنتظَرُ bad_point", e)
	} else {
		seen["point"] = e
	}

	// ── د · ولا إعدادَ تغطية ────────────────────────────────────
	disableAllZones(t, hh)
	if e, _ := cfcProbe(t, hh, tok, srvLat, srvLng); e != "coverage_unavailable" {
		t.Errorf("**بلا تغطيةٍ ردّ %q** — والمنتظَرُ coverage_unavailable", e)
	} else {
		seen["config"] = e
	}

	if len(seen) == 4 {
		uniq := map[string]bool{}
		for _, v := range seen {
			uniq[v] = true
		}
		if len(uniq) != 4 {
			t.Errorf("**رمزان تطابقا**: %v", seen)
		}
		t.Logf("✓ الأربعةُ مفترقة: %q · %q · %q · %q",
			seen["launch"], seen["config"], seen["outside"], seen["point"])
	}
}

// TestCFC5_UsableCoverageStillWorks **والصالحُ يبقى يعمل.**
//
// **ومنعٌ يمنع الصحيحَ أسوأُ من لا منع** — **وتغطيةُ الإنتاج صالحةٌ
// (أربعُ دوائرَ)، فلا يُغلَق عندها شيء.**
func TestCFC5_UsableCoverageStillWorks(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	onlyZone(t, hh, srvLat, srvLng, 50000)
	u, tok := capUser(t, hh, "customer")
	before := countOrders(t, hh, u.ID)

	r := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, srvLat, srvLng))
	if r.Code != http.StatusCreated && r.Code != http.StatusOK {
		t.Fatalf("**طلبٌ صحيحٌ بتغطيةٍ صالحةٍ رُدّ**: %d / %s", r.Code, r.Err())
	}
	if n := countOrders(t, hh, u.ID); n != before+1 {
		t.Errorf("**الصحيحُ لم يُنشئ صفّاً**: %d ⇒ %d", before, n)
	}
}

// TestCFC6_UnavailableLeavesNoTrace **والمردودُ لا يترك أثراً.**
func TestCFC6_UnavailableLeavesNoTrace(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	u, tok := capUser(t, hh, "customer")
	disableAllZones(t, hh)

	base := financialBaseline(t, hh)
	beforeOrders := countOrders(t, hh, u.ID)
	var beforeEvents int
	_ = hh.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM outbox`).Scan(&beforeEvents)

	if n, c := cfcProbe(t, hh, tok, srvLat, srvLng); n != "coverage_unavailable" ||
		c != "coverage_unavailable" {
		t.Fatalf("لم يُردّا: %q · %q", n, c)
	}
	if n := countOrders(t, hh, u.ID); n != beforeOrders {
		t.Errorf("**صفوفٌ أُنشئت**: %d ⇒ %d", beforeOrders, n)
	}
	assertNewViolations(t, hh, base)
	var afterEvents int
	_ = hh.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM outbox`).Scan(&afterEvents)
	if afterEvents != beforeEvents {
		t.Errorf("**أحداثٌ خرجت بلا تغطية**: %d ⇒ %d — "+
			"**ولا متجرَ ولا سائقَ يُبلَّغ.**", beforeEvents, afterEvents)
	}
}

// TestCFC7_QuoteSaysConfigNotGeography **والتسعيرةُ تقول الحالَ لا الجغرافيا.**
func TestCFC7_QuoteSaysConfigNotGeography(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	disableAllZones(t, hh)
	it := hh.NewItem(1000)

	q := hh.POST("/api/v1/public/quote", "", map[string]any{
		"items": []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
		"lat":   srvLat, "lng": srvLng,
	})
	if q.Code != http.StatusOK {
		t.Fatalf("**التسعيرةُ سقطت** — وسلّةٌ تنهار سلّةٌ لا تُستعمل: %d / %s",
			q.Code, q.Err())
	}
	b := string(q.Body)
	if !strings.Contains(b, "coverage_unavailable") {
		t.Errorf("**التسعيرةُ لا تقول الحال**: %s", b)
	}
	if strings.Contains(b, `"out_of_zone":true`) {
		t.Error("**التسعيرةُ تقول «خارجَ التغطية» والعلّةُ في اللوحة** — " +
			"**ويُبدّل الزبونُ عنوانَه بلا جدوى.**")
	}
	if strings.Contains(b, `"serviceable":true`) {
		t.Error("**التسعيرةُ تقول «يُوصَّل» ولا تغطيةَ صالحة.**")
	}
}

// TestCFC8_DestinationCoherence **ونقطةٌ واحدةٌ للفحصِ والتسعيرِ والتخزين.**
//
// **وما لا يُقبَل**: **الفحصُ على إحداثيّةٍ والمخزَّنُ إحداثيّةٌ أخرى.**
//
// **والعقدُ حرُّ الإحداثيّة لا بمعرّفِ عنوانٍ محفوظ** — `lat`/`lng`
// و`address_text` — **فيُقاس أنّ الزوجَ الواصلَ هو الزوجُ المخزَّن.**
func TestCFC8_DestinationCoherence(t *testing.T) {
	hh := New(t)
	srvOpenAllLaunch(hh)
	onlyZone(t, hh, srvLat, srvLng, 50000)
	u, tok := capUser(t, hh, "customer")

	// **ونقطةٌ مميَّزةٌ لا تشبه المركز** — فلا يُقرأ التطابقُ مصادفةً.
	const lat, lng = 35.9731, 39.0417
	r := hh.POST("/api/v1/orders", tok, srvOrderBody(t, hh, lat, lng))
	if r.Code >= 400 {
		t.Fatalf("لم يُقبَل: %d / %s", r.Code, r.Err())
	}

	var gotLat, gotLng float64
	var zoneID *string
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT ST_Y(dropoff::geometry), ST_X(dropoff::geometry), zone_id::text
		  FROM orders WHERE customer_id = $1 ORDER BY created_at DESC LIMIT 1`,
		u.ID).Scan(&gotLat, &gotLng, &zoneID); err != nil {
		t.Fatalf("قراءةُ المقصد المخزَّن: %v", err)
	}
	// **وفرقُ التمثيل العائم يُحتمَل، وفرقُ النقطة لا** — مترٌ واحدٌ
	// نحوَ ٩ من مئةِ ألفٍ من الدرجة.
	const eps = 1e-6
	if diff := gotLat - lat; diff > eps || diff < -eps {
		t.Errorf("**خطُّ العرضِ المخزَّنُ غيرُ المفحوص**: %.7f ≠ %.7f", gotLat, lat)
	}
	if diff := gotLng - lng; diff > eps || diff < -eps {
		t.Errorf("**خطُّ الطولِ المخزَّنُ غيرُ المفحوص**: %.7f ≠ %.7f", gotLng, lng)
	}
	if zoneID == nil || *zoneID == "" {
		t.Error("**الطلبُ بلا منطقةٍ وقد قُبل بتغطيةٍ صالحة** — " +
			"**والمنطقةُ التي أذنت به تُكتب معه.**")
	}
	t.Logf("✓ المقصدُ واحدٌ: فحصٌ وتسعيرٌ وتخزينٌ على %.4f,%.4f", gotLat, gotLng)
}
