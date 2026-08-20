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

// TestNAVB_020_BatchUnchanged **ودفعةُ ما جُمع بلا شبكةٍ تعمل كما كانت.**
//
// **ولا تحمل اتّجاهاً** — قِيس ٢٠٢٦-٠٨-٢٠: عقدُ الدفعة لم يُمسّ في
// المرحلة ١، **وحقلٌ يُضاف في مكانٍ ويُنسى في آخرَ يجعل نصفَ الأثر
// بلا اتّجاه.** مسجَّلٌ في التقرير.
func TestNAVB_020_BatchUnchanged(t *testing.T) {
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
