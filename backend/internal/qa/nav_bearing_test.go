package qa

// ══════════════════════════════════════════════════════════════════════
// **الطبقةُ الثالثة عشرة — عقدُ اتّجاه السائق**
// ══════════════════════════════════════════════════════════════════════
//
// المعرّفات: `NAVB-*` · الوسم: `@api @release`
//
// (المرحلة ١ من الملاحة، بأمر المالك ٢٠٢٦-٠٨-٢٠: «أثبت فقط، بدون
//  تغيير جديد، أنّ… **القراءة القديمة بدون Bearing تبقى مقبولة** ولا
//  يوجد Breaking Change للنسخ القديمة».)
//
// # وما يُقاس
//
// **أنّ حقلاً أُضيف لا يكسر من لا يعرفه** — وهو أخطرُ ما في إضافةِ
// حقل: **نسخةُ السائق المنشورةُ اليوم لا ترسله**، ومن جعله إلزاميّاً
// أوقف كلَّ سائقٍ لم يحدّث.

import (
	"encoding/json"
	"testing"
	"time"
)

// bearingOf **آخرُ اتّجاهٍ حُفظ لهذا السائق** — وفارغٌ يعني `NULL`.
func bearingOf(t *testing.T, h *Harness, driverID string) *float64 {
	t.Helper()
	var v *float64
	err := h.Pool.QueryRow(h.T.Context(), `
		SELECT bearing_deg FROM driver_track
		 WHERE driver_id = $1::uuid
		 ORDER BY recorded_at DESC LIMIT 1`, driverID).Scan(&v)
	if err != nil {
		t.Fatalf("NAVB: تعذّرت قراءةُ الأثر: %v", err)
	}
	return v
}

// TestNAVB_001_BearingIsStored **والاتّجاهُ يصل ويُحفظ.**
func TestNAVB_001_BearingIsStored(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	res := h.POST("/api/v1/driver/location", drv.Token, map[string]any{
		"lat": 35.9506, "lng": 39.0094,
		"speed_mps": 11.2, "accuracy_m": 6.0, "bearing_deg": 137.5,
	})
	if res.Code != 200 {
		t.Fatalf("NAVB-001 الإرسالُ رُدّ: %s", res)
	}
	got := bearingOf(t, h, drv.ID)
	if got == nil {
		t.Fatal("NAVB-001 **الاتّجاهُ أُرسل ولم يُحفظ**")
	}
	if *got != 137.5 {
		t.Errorf("NAVB-001 حُفظ %v لا 137.5", *got)
	}
}

// TestNAVB_010_OldClientStillWorks **ونسخةٌ لا ترسل اتّجاهاً تعمل.**
//
// **وهذا هو التوافقُ الخلفيُّ بعينه**: تطبيقُ السائق المنشورُ اليوم
// **لا يعرف الحقلَ أصلاً.**
func TestNAVB_010_OldClientStillWorks(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	// **جسمٌ كما ترسله النسخةُ القديمةُ حرفاً بحرف** — بلا `bearing_deg`.
	res := h.POST("/api/v1/driver/location", drv.Token, map[string]any{
		"lat": 35.9506, "lng": 39.0094,
		"speed_mps": 9.0, "accuracy_m": 8.0,
	})
	if res.Code != 200 {
		t.Fatalf("NAVB-010 **النسخةُ القديمةُ رُدّت**: %s", res)
	}
	if got := bearingOf(t, h, drv.ID); got != nil {
		t.Errorf("NAVB-010 اتّجاهٌ اخترعه الخادمُ: %v", *got)
	}
}

// TestNAVB_011_BadBearingDoesNotBreakWrite **واتّجاهٌ خارجَ الدائرة لا
// يُسقط كتابةَ الأثر.**
//
// **وبعضُ المستقبِلات ترسل ٣٦٠ أو سالباً** — **وقيدُ القاعدة يرفضه
// فتسقط الكتابةُ كلُّها**، فيضيع الموضعُ لأنّ الاتّجاهَ شاذّ.
func TestNAVB_011_BadBearingDoesNotBreakWrite(t *testing.T) {
	h := New(t)
	for _, bad := range []float64{360, -1, 999} {
		drv := h.NewUser("driver")
		res := h.POST("/api/v1/driver/location", drv.Token, map[string]any{
			"lat": 35.9506, "lng": 39.0094, "bearing_deg": bad,
		})
		if res.Code != 200 {
			t.Fatalf("NAVB-011 اتّجاهٌ %v ردَّ النداء: %s", bad, res)
		}
		var n int
		if err := h.Pool.QueryRow(h.T.Context(),
			`SELECT count(*)::int FROM driver_track WHERE driver_id = $1::uuid`,
			drv.ID).Scan(&n); err != nil {
			t.Fatalf("NAVB-011 تعذّر العدّ: %v", err)
		}
		if n == 0 {
			t.Errorf("NAVB-011 **اتّجاهٌ %v أسقط كتابةَ الموضع كلَّها**", bad)
		}
		if got := bearingOf(t, h, drv.ID); got != nil {
			t.Errorf("NAVB-011 حُفظ اتّجاهٌ شاذّ: %v", *got)
		}
	}
}

// TestNAVB_020_BatchAcceptsPlainPoints **ودفعةُ ما جُمع بلا شبكةٍ تعمل.**
//
// **وكانت لا تحمل اتّجاهاً حتّى صحّحه المالك** (٢٠٢٦-٠٨-٢٠): «أيّ
// نقاط تُجمع أثناء انقطاع الشبكة تفقد الاتجاه نهائيًا». **وحقلٌ
// يُضاف في مكانٍ ويُنسى في آخرَ يجعل نصفَ الأثر بلا اتّجاه** — وهو
// ما وقع.
//
// **وهذه تحرس أنّ الإضافةَ لم تكسر ما كان** — انظر `NAVB-030`
// فما بعدها للاتّجاه نفسِه.
func TestNAVB_020_BatchAcceptsPlainPoints(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	// **ووقتٌ حيٌّ لا ثابتٌ مكتوب** — الدفعةُ ترفض ما شاخ (`batchMaxAge`)،
	// **واختبارٌ بوقتٍ ثابتٍ يمرّ اليومَ ويسقط غدا.**
	now := time.Now().UTC()
	res := h.POST("/api/v1/driver/location/batch", drv.Token, map[string]any{
		"points": []map[string]any{
			{"lat": 35.9506, "lng": 39.0094, "at": now.Add(-40 * time.Second).Format(time.RFC3339),
				"speed_mps": 8.0, "accuracy_m": 7.0},
			{"lat": 35.9510, "lng": 39.0098, "at": now.Add(-20 * time.Second).Format(time.RFC3339),
				"speed_mps": 9.0, "accuracy_m": 6.0},
		},
	})
	if res.Code != 200 {
		t.Fatalf("NAVB-020 **الدفعةُ انكسرت**: %s", res)
	}
	if n, _ := res.JSON()["accepted"].(float64); n != 2 {
		t.Errorf("NAVB-020 قُبلت %v من نقطتين", res.JSON()["accepted"])
	}
}

// ══════════════════════════════════════════════════════════════════════
// **الاتّجاهُ في الدفعة — أُكمل ٢٠٢٦-٠٨-٢٠**
// ══════════════════════════════════════════════════════════════════════
//
// (تصحيحُ المالك: «أيّ نقاط تُجمع أثناء انقطاع الشبكة تفقد الاتجاه
//  نهائيًا».)
//
// **والانقطاعُ في الشارع كثير** — فأطولُ المسارات وأغناها بالمنعطفات
// هي التي كانت تصل بلا اتّجاه.

// batchOf **يبني دفعةً بأوقاتٍ حيّة** — الدفعةُ ترفض ما شاخ.
func batchOf(points ...map[string]any) map[string]any {
	now := time.Now().UTC()
	out := make([]map[string]any, 0, len(points))
	for i, p := range points {
		q := map[string]any{}
		for k, v := range p {
			q[k] = v
		}
		q["at"] = now.Add(time.Duration(-(len(points)-i)*20) * time.Second).Format(time.RFC3339)
		out = append(out, q)
	}
	return map[string]any{"points": out}
}

// trackRows **ما حُفظ لهذا السائق** — موضعٌ واتّجاه.
func trackRows(t *testing.T, h *Harness, driverID string) []*float64 {
	t.Helper()
	rows, err := h.Pool.Query(h.T.Context(), `
		SELECT bearing_deg FROM driver_track
		 WHERE driver_id = $1::uuid ORDER BY recorded_at`, driverID)
	if err != nil {
		t.Fatalf("NAVB: تعذّرت قراءةُ الأثر: %v", err)
	}
	defer rows.Close()
	var out []*float64
	for rows.Next() {
		var v *float64
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("NAVB: تعذّرت القراءة: %v", err)
		}
		out = append(out, v)
	}
	return out
}

// TestNAVB_030_OldBatchStillAccepted **ودفعةٌ قديمةٌ بلا اتّجاهٍ تُقبل.**
//
// **وطابورُ من كان بلا شبكةٍ كُتب بالنسخة القديمة** — ومن رفضه أضاع
// مسارَ ساعةٍ كاملة.
func TestNAVB_030_OldBatchStillAccepted(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	res := h.POST("/api/v1/driver/location/batch", drv.Token, batchOf(
		map[string]any{"lat": 35.9506, "lng": 39.0094, "speed_mps": 8.0, "accuracy_m": 7.0},
		map[string]any{"lat": 35.9510, "lng": 39.0098, "speed_mps": 9.0, "accuracy_m": 6.0},
	))
	if res.Code != 200 {
		t.Fatalf("NAVB-030 **الدفعةُ القديمةُ رُدّت**: %s", res)
	}
	if n, _ := res.JSON()["accepted"].(float64); n != 2 {
		t.Fatalf("NAVB-030 قُبلت %v من نقطتين", res.JSON()["accepted"])
	}
	for i, b := range trackRows(t, h, drv.ID) {
		if b != nil {
			t.Errorf("NAVB-030 النقطةُ %d اخترع لها اتّجاه: %v", i, *b)
		}
	}
}

// TestNAVB_031_BatchBearingStored **والاتّجاهُ الصالحُ يُخزَّن كما أُرسل.**
func TestNAVB_031_BatchBearingStored(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	res := h.POST("/api/v1/driver/location/batch", drv.Token, batchOf(
		map[string]any{"lat": 35.9506, "lng": 39.0094, "bearing_deg": 0.0},
		map[string]any{"lat": 35.9510, "lng": 39.0098, "bearing_deg": 271.25},
		map[string]any{"lat": 35.9514, "lng": 39.0102, "bearing_deg": 359.9},
	))
	if res.Code != 200 {
		t.Fatalf("NAVB-031 الدفعةُ رُدّت: %s", res)
	}
	got := trackRows(t, h, drv.ID)
	want := []float64{0, 271.25, 359.9}
	if len(got) != len(want) {
		t.Fatalf("NAVB-031 حُفظت %d نقاطٍ لا %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] == nil {
			t.Errorf("NAVB-031 **النقطةُ %d فقدت اتّجاهَها**", i)
			continue
		}
		if *got[i] != w {
			t.Errorf("NAVB-031 النقطةُ %d حُفظت %v لا %v", i, *got[i], w)
		}
	}
}

// TestNAVB_032_BadBatchBearingIgnored **واتّجاهٌ شاذٌّ يُهمَل ولا يُسقط
// الموضع.**
//
// **وقيدُ القاعدة يرفض ما خرج عن الدائرة، ورفضُه يُسقط الدفعةَ
// كلَّها** — فتضيع عشرون نقطةً لأنّ واحدةً منها شاذّة.
func TestNAVB_032_BadBatchBearingIgnored(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	res := h.POST("/api/v1/driver/location/batch", drv.Token, batchOf(
		map[string]any{"lat": 35.9506, "lng": 39.0094, "bearing_deg": 360.0},
		map[string]any{"lat": 35.9510, "lng": 39.0098, "bearing_deg": -5.0},
		map[string]any{"lat": 35.9514, "lng": 39.0102, "bearing_deg": 4000.0},
	))
	if res.Code != 200 {
		t.Fatalf("NAVB-032 **اتّجاهٌ شاذٌّ ردَّ الدفعة**: %s", res)
	}
	got := trackRows(t, h, drv.ID)
	if len(got) != 3 {
		t.Fatalf("NAVB-032 **ضاعت مواضعُ بسبب اتّجاهٍ شاذّ**: حُفظت %d من ٣", len(got))
	}
	for i, b := range got {
		if b != nil {
			t.Errorf("NAVB-032 حُفظ اتّجاهٌ شاذٌّ في %d: %v", i, *b)
		}
	}
}

// TestNAVB_033_MixedBatch **وخليطٌ بعضُه يحمل اتّجاهاً وبعضُه لا.**
//
// **وهي الحالُ الواقعيّة**: الجهازُ لا يقول اتّجاهاً عند الوقوف،
// **فطابورُ رحلةٍ فيها إشاراتُ مرورٍ خليطٌ بطبعه.**
func TestNAVB_033_MixedBatch(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	res := h.POST("/api/v1/driver/location/batch", drv.Token, batchOf(
		map[string]any{"lat": 35.9506, "lng": 39.0094, "bearing_deg": 45.0},
		map[string]any{"lat": 35.9510, "lng": 39.0098},
		map[string]any{"lat": 35.9514, "lng": 39.0102, "bearing_deg": 200.0},
		map[string]any{"lat": 35.9518, "lng": 39.0106, "bearing_deg": 500.0},
	))
	if res.Code != 200 {
		t.Fatalf("NAVB-033 الدفعةُ رُدّت: %s", res)
	}
	if n, _ := res.JSON()["accepted"].(float64); n != 4 {
		t.Fatalf("NAVB-033 قُبلت %v من أربع", res.JSON()["accepted"])
	}
	got := trackRows(t, h, drv.ID)
	if len(got) != 4 {
		t.Fatalf("NAVB-033 حُفظت %d من أربع", len(got))
	}
	if got[0] == nil || *got[0] != 45 {
		t.Errorf("NAVB-033 الأولى: %v", got[0])
	}
	if got[1] != nil {
		t.Errorf("NAVB-033 الثانيةُ بلا اتّجاهٍ واخترع لها: %v", *got[1])
	}
	if got[2] == nil || *got[2] != 200 {
		t.Errorf("NAVB-033 الثالثة: %v", got[2])
	}
	if got[3] != nil {
		t.Errorf("NAVB-033 الرابعةُ شاذّةٌ وحُفظت: %v", *got[3])
	}
}

// TestNAVB_034_ContractHasNoNewRequiredField **ولا حقلَ إلزاميٍّ جديد.**
//
// **وهذا هو التوافقُ الخلفيُّ في أنقى صوره**: أقلُّ جسمٍ ممكنٍ يُقبل —
// موضعٌ ووقتٌ لا غير.
func TestNAVB_034_ContractHasNoNewRequiredField(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	res := h.POST("/api/v1/driver/location/batch", drv.Token, batchOf(
		map[string]any{"lat": 35.9506, "lng": 39.0094},
	))
	if res.Code != 200 {
		t.Fatalf("NAVB-034 **أقلُّ جسمٍ ممكنٍ رُدّ**: %s", res)
	}
	if n, _ := res.JSON()["accepted"].(float64); n != 1 {
		t.Errorf("NAVB-034 قُبلت %v من واحدة", res.JSON()["accepted"])
	}
}

// TestNAVB_035_BatchStillMovesDriver **وسلوكُ الدفعة لم يتبدّل.**
//
// **وحقلٌ يُضاف قد يكسر ما جاوره**: أحدثُ نقطةٍ تكتب الموضعَ الحاليّ،
// **ومن أخطأ في ترتيب الوسائط كتب خطَّ الطول مكان العرض** فذهب
// السائقُ إلى المحيط.
func TestNAVB_035_BatchStillMovesDriver(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	res := h.POST("/api/v1/driver/location/batch", drv.Token, batchOf(
		map[string]any{"lat": 35.9506, "lng": 39.0094, "bearing_deg": 10.0},
		map[string]any{"lat": 35.9600, "lng": 39.0200, "bearing_deg": 20.0},
	))
	if res.Code != 200 {
		t.Fatalf("NAVB-035 الدفعةُ رُدّت: %s", res)
	}
	var lat, lng float64
	if err := h.Pool.QueryRow(h.T.Context(), `
		SELECT ST_Y(last_location::geometry), ST_X(last_location::geometry)
		  FROM users WHERE id = $1::uuid`, drv.ID).Scan(&lat, &lng); err != nil {
		t.Fatalf("NAVB-035 تعذّرت قراءةُ الموضع: %v", err)
	}
	if int(lat*1000) != 35960 || int(lng*1000) != 39020 {
		t.Errorf("NAVB-035 **الموضعُ الأخيرُ خطأ**: %f,%f", lat, lng)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **السلسلةُ كاملةً — من نصِّ التطبيق إلى عمود القاعدة**
// ══════════════════════════════════════════════════════════════════════
//
// (المرحلة ١، أمرُ المالك ٢٠٢٦-٠٨-٢٠: «أريد اختبار تكامل واحد على
//  الأقلّ يثبت السلسلة: Android-side TrackPoint → serialized request
//  → backend → stored bearing_deg».)
//
// # ولماذا نصٌّ خامٌّ لا خريطةُ مفاتيح
//
// **الجسمُ هنا مكتوبٌ كما يكتبه `kotlinx.serialization` حرفاً بحرف** —
// أسماءُ الحقول وترتيبُها. **وخريطةٌ تُبنى في Go تختبر Go لا تختبر
// العقد**: من بدّل `@SerialName` في كوتلن **لا يسقط شيء**، ويصل
// الحقلُ باسمٍ لا يعرفه الخادم.

// TestNAVB_040_AndroidBodyReachesColumn **نصُّ التطبيق يصل العمود.**
func TestNAVB_040_AndroidBodyReachesColumn(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")

	// **وهذا ما تنتجه `DriverApi.sendLocation` حرفاً بحرف.**
	raw := `{"lat":35.9506,"lng":39.0094,"speed_mps":11.2,"accuracy_m":6.0,"bearing_deg":137.5}`
	res := h.Call("POST", "/api/v1/driver/location", drv.Token, json.RawMessage(raw), nil)
	if res.Code != 200 {
		t.Fatalf("NAVB-040 نصُّ التطبيق رُدّ: %s", res)
	}
	got := bearingOf(t, h, drv.ID)
	if got == nil || *got != 137.5 {
		t.Fatalf("NAVB-040 **السلسلةُ انقطعت**: العمودُ %v", got)
	}
}

// TestNAVB_041_AndroidQueueBodyReachesColumn **وطابورُ التطبيق كذلك.**
//
// **وهو ما يُقرأ من `points.jsonl` ويُرسَل دفعةً** — بخليطه الواقعيّ:
// نقطةٌ بالاتّجاه، وأخرى بلاه (وقوفٌ عند إشارة)، وثالثةٌ بشاذّ.
func TestNAVB_041_AndroidQueueBodyReachesColumn(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	now := time.Now().UTC()
	f := func(d time.Duration) string { return now.Add(d).Format(time.RFC3339) }

	raw := `{"points":[` +
		`{"lat":35.9506,"lng":39.0094,"at":"` + f(-60*time.Second) + `","speed_mps":11.2,"accuracy_m":6.0,"bearing_deg":45.0},` +
		`{"lat":35.9510,"lng":39.0098,"at":"` + f(-40*time.Second) + `","speed_mps":0.1,"accuracy_m":7.0},` +
		`{"lat":35.9514,"lng":39.0102,"at":"` + f(-20*time.Second) + `","speed_mps":9.5,"accuracy_m":5.0,"bearing_deg":400.0}` +
		`]}`
	res := h.Call("POST", "/api/v1/driver/location/batch", drv.Token, json.RawMessage(raw), nil)
	if res.Code != 200 {
		t.Fatalf("NAVB-041 طابورُ التطبيق رُدّ: %s", res)
	}
	if n, _ := res.JSON()["accepted"].(float64); n != 3 {
		t.Fatalf("NAVB-041 قُبلت %v من ثلاث", res.JSON()["accepted"])
	}
	rows := trackRows(t, h, drv.ID)
	if len(rows) != 3 {
		t.Fatalf("NAVB-041 حُفظت %d من ثلاث", len(rows))
	}
	if rows[0] == nil || *rows[0] != 45 {
		t.Errorf("NAVB-041 **الاتّجاهُ ضاع في الطابور**: %v", rows[0])
	}
	if rows[1] != nil {
		t.Errorf("NAVB-041 اخترع اتّجاهاً لنقطةِ وقوف: %v", *rows[1])
	}
	if rows[2] != nil {
		t.Errorf("NAVB-041 حُفظ شاذٌّ: %v", *rows[2])
	}
}

// TestNAVB_042_LegacyQueueLineStillWorks **وسطرٌ من نسخةٍ قديمةٍ يُرفع.**
//
// **وطابورُ سائقٍ انقطعت شبكتُه ساعةً كُتب بالنسخة القديمة** — ومن
// رفضه أضاع مسارَه كلَّه.
func TestNAVB_042_LegacyQueueLineStillWorks(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	now := time.Now().UTC().Add(-30 * time.Second).Format(time.RFC3339)
	// **بلا `bearing_deg` أصلاً** — كما يكتبها التطبيقُ المنشورُ اليوم.
	raw := `{"points":[{"lat":35.9506,"lng":39.0094,"at":"` + now +
		`","speed_mps":9.0,"accuracy_m":8.0}]}`
	res := h.Call("POST", "/api/v1/driver/location/batch", drv.Token, json.RawMessage(raw), nil)
	if res.Code != 200 {
		t.Fatalf("NAVB-042 **سطرٌ قديمٌ رُدّ**: %s", res)
	}
	if n, _ := res.JSON()["accepted"].(float64); n != 1 {
		t.Errorf("NAVB-042 قُبلت %v من واحدة", res.JSON()["accepted"])
	}
	if b := bearingOf(t, h, drv.ID); b != nil {
		t.Errorf("NAVB-042 اخترع اتّجاهاً: %v", *b)
	}
}
