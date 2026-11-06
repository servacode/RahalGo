package qa

// **كشفُ الشوارع محدودُ القراءة** — نافذةُ أيّامٍ وسقفُ نقاط (طلبُ المالك ٢٠٢٦-١١-٠٦: «هلّق»).
//
// التقليمُ (٩٠ يوماً) يقع مع كتابة نقطةٍ جديدة وحدَها، فإن غاب السائقون بقي القديمُ في الجدول —
// **والكشفُ كان يقرأ الجدولَ كلَّه بلا حدّ**. هنا: شارعٌ قديمٌ خارجَ النافذة لا يُقترح، وشارعٌ حديثٌ يُقترح.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/maplearn"
)

// farRoads **كلُّ نقطةٍ بعيدةٌ عن الطرق** — بديلُ المحرّك في الاختبار.
type farRoads struct{}

func (farRoads) NearestM(context.Context, float64, float64) (float64, error) { return 50, nil }

// mlStreetDays يدخل شارعاً مستقيماً (~٣٠٠ م شرقاً) في الأيّام المعطاة (أيّامٌ قبل اليوم بتوقيت دمشق).
func mlStreetDays(t *testing.T, h *Harness, lat, lng0 float64, daysAgo ...int) {
	t.Helper()
	for _, d := range daysAgo {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO map_trace_points (point, accuracy_m, speed_mps, day)
			SELECT ST_SetSRID(ST_MakePoint($1 + i * 0.00009, $2 + (i % 2) * 0.00002), 4326)::geography, 5, 4,
			       (now() AT TIME ZONE 'Asia/Damascus')::date - $3::int
			  FROM generate_series(0, 35) AS i`, lng0, lat, d); err != nil {
			t.Fatalf("إدخال نقاط: %v", err)
		}
	}
}

func TestMapLearning_DetectionReadsWindowOnly(t *testing.T) {
	h := New(t)
	for _, q := range []string{`DELETE FROM map_trace_points`, `DELETE FROM map_street_candidates`} {
		if _, err := h.Pool.Exec(ctxBG(), q); err != nil {
			t.Fatal(err)
		}
	}
	// شارعٌ قديمٌ (١٠٠–١٠٢ يوم) لم يُقلَّم لأنّ أحداً لم يكتب بعده، وشارعٌ حديثٌ (١–٢ يوم) بعيدٌ عنه.
	mlStreetDays(t, h, 35.950, 39.000, 100, 101, 102)
	mlStreetDays(t, h, 35.960, 39.010, 1, 2)

	res, err := maplearn.Run(ctxBG(), h.Pool, farRoads{}, maplearn.DefaultOptions())
	if err != nil {
		t.Fatalf("الجولة: %v", err)
	}
	if res.Candidates != 1 {
		t.Fatalf("المتوقَّع شارعٌ واحدٌ حديث، والناتج %d (نقاطٌ بعيدةٌ مقروءة %d)", res.Candidates, res.FarPoints)
	}
	if res.FarPoints != 72 {
		t.Errorf("المقروءُ من البعيد = %d والمتوقَّع ٧٢ (يومان × ٣٦) — القديمُ خارجَ النافذة لا يُقرأ", res.FarPoints)
	}
	var lat float64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT ST_Y(ST_StartPoint(line::geometry)) FROM map_street_candidates WHERE status = 'pending'`).Scan(&lat); err != nil {
		t.Fatal(err)
	}
	if lat < 35.955 {
		t.Errorf("المقترحُ هو الشارعُ القديم (عرض %.4f)", lat)
	}

	// **وسقفُ النقاط يحدّ القراءة** — الأحدثُ أوّلاً.
	opt := maplearn.DefaultOptions()
	opt.MaxFarPoints = 36
	res, err = maplearn.Run(ctxBG(), h.Pool, farRoads{}, opt)
	if err != nil {
		t.Fatalf("الجولة بسقف: %v", err)
	}
	if res.FarPoints != 36 {
		t.Errorf("بسقف ٣٦ قُرئ %d", res.FarPoints)
	}
}
