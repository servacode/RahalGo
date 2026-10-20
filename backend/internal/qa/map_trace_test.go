package qa

import (
	"testing"
	"time"
)

// mapTraceNear عددُ نقاط الخريطة حول موضعٍ — **فلا سائقَ في الجدول يُعدّ به.**
func mapTraceNear(t *testing.T, h *Harness, lat, lng, radiusM float64) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(h.T.Context(), `
		SELECT count(*)::int FROM map_trace_points
		WHERE ST_DWithin(point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3)`,
		lat, lng, radiusM).Scan(&n); err != nil {
		t.Fatalf("عدُّ نقاط الخريطة: %v", err)
	}
	return n
}

// TestMapTrace_KeepsOnlyWorthyAnonymousPoints **حركةُ السائقين تُحفظ لرسم الخريطة بلا هويّة**
// (قرارُ المالك ٢٠٢٦-١٠-٢٠: «لازم نستفيد من حركة السائقين لنرسم خريطة الرقة بدقة اكثر»):
// الدقيقُ الماشي غيرُ المزيّف البعيدُ عن سابقه وحدَه، **ودفعةٌ معادةٌ لا تتكرّر.**
func TestMapTrace_KeepsOnlyWorthyAnonymousPoints(t *testing.T) {
	h := New(t)
	// **بقعةٌ خارجَ المدينة لهذا الاختبار وحدَه** — تُنظَّف قبله فلا يُعدّ ما تركه تشغيلٌ سابق.
	const lat0, lng0 = 35.7000, 38.7000
	clear := func() {
		_, _ = h.Pool.Exec(h.T.Context(), `DELETE FROM map_trace_points
			WHERE ST_DWithin(point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, 3000)`, lat0, lng0)
	}
	clear()
	t.Cleanup(clear)

	// **ولا عمودَ يربط النقطةَ بسائقٍ** — الخصوصيّةُ في البنية لا في الوعد.
	var cols int
	if err := h.Pool.QueryRow(h.T.Context(), `SELECT count(*)::int FROM information_schema.columns
		WHERE table_name = 'map_trace_points' AND (column_name LIKE '%driver%' OR column_name LIKE '%user%'
		   OR column_name IN ('at', 'recorded_at', 'created_at'))`).Scan(&cols); err != nil {
		t.Fatal(err)
	}
	if cols != 0 {
		t.Fatalf("في الجدول %d عمودٌ يدلّ على السائق أو الساعة", cols)
	}

	drv := h.NewUser("driver")
	now := time.Now().UTC()
	at := func(sec int) string { return now.Add(time.Duration(sec-120) * time.Second).Format(time.RFC3339) }
	// كلُّ نقطةٍ ~١١٠ م بعد سابقتها وبينهما عشرون ثانية — **لا قفزةَ تُهمَل.**
	batch := map[string]any{"points": []map[string]any{
		{"lat": lat0 + 0.000, "lng": lng0, "at": at(0), "speed_mps": 6.0, "accuracy_m": 5.0},                  // ✔ دقيقٌ ماشٍ
		{"lat": lat0 + 0.001, "lng": lng0, "at": at(20), "speed_mps": 6.0, "accuracy_m": 30.0},                // ✘ غيرُ دقيق
		{"lat": lat0 + 0.002, "lng": lng0, "at": at(40), "speed_mps": 0.3, "accuracy_m": 5.0},                 // ✘ واقف
		{"lat": lat0 + 0.003, "lng": lng0, "at": at(60), "speed_mps": 6.0, "accuracy_m": 5.0, "mocked": true}, // ✘ مزيّف
		{"lat": lat0 + 0.004, "lng": lng0, "at": at(80), "speed_mps": 6.0, "accuracy_m": 8.0},                 // ✔
		{"lat": lat0 + 0.00402, "lng": lng0, "at": at(100), "speed_mps": 1.5, "accuracy_m": 8.0},              // ✘ ٢ م عن سابقتها
	}}
	if res := h.POST("/api/v1/driver/location/batch", drv.Token, batch); res.Code != 200 {
		t.Fatalf("الدفعة: %s", res)
	}
	if got := mapTraceNear(t, h, lat0+0.002, lng0, 1000); got != 2 {
		t.Fatalf("نقاطُ الخريطة = %d — والمتوقَّع ٢ (الأولى والخامسة)", got)
	}
	if got := mapTraceNear(t, h, lat0+0.001, lng0, 20); got != 0 {
		t.Fatalf("حُفظت نقطةٌ غيرُ دقيقة")
	}

	// **والدفعةُ نفسُها إن أُعيدت بعد انقطاع الردّ لا تتكرّر.**
	if res := h.POST("/api/v1/driver/location/batch", drv.Token, batch); res.Code != 200 {
		t.Fatalf("إعادةُ الدفعة: %s", res)
	}
	if got := mapTraceNear(t, h, lat0+0.002, lng0, 1000); got != 2 {
		t.Fatalf("بعد إعادة الدفعة = %d — تكرّرت النقاط", got)
	}

	// **والنداءُ المفرد يحفظ الصالحَ كذلك.**
	if res := h.POST("/api/v1/driver/location", drv.Token, map[string]any{
		"lat": lat0 + 0.0045, "lng": lng0, "speed_mps": 7.0, "accuracy_m": 6.0,
	}); res.Code != 200 {
		t.Fatalf("النداءُ المفرد: %s", res)
	}
	if got := mapTraceNear(t, h, lat0+0.0045, lng0, 5); got != 1 {
		t.Fatalf("النداءُ المفرد = %d — والمتوقَّع ١", got)
	}
}
