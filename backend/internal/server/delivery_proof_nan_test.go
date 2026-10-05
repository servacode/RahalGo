package server

// **إحداثيّةٌ ليست رقماً لا تتجاوز حارسَ المسافة** (فحصُ الهجوم ٢٠٢٦-١٠-٠٥)
//
// `strconv.ParseFloat` يقبل `NaN` و`Inf` نصّاً، **والمقارنةُ مع `NaN` كاذبةٌ
// دائماً** — فكانت `dist > maxM` لا تصدق، **وتُقبل صورةُ التسليم من أيّ مكان**
// برفع `lat=NaN`. والطريقُ وحارسُ «سلّمتُ البضاعة» يرفضانها منذ مدّة؛ والإثباتُ
// وحدَه نسيها.

import (
	"context"
	"math"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/media"
)

func TestDeliveryProof_NaNOrInfLocationIsRejected(t *testing.T) {
	f := newDriverFixture(t, 1)
	m, err := media.NewService(f.pool, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f.srv.media = m
	d := f.drivers[0]
	ctx := context.Background()
	id := f.problemOrderAt(t, "at_dropoff", d)
	for _, c := range []struct {
		name     string
		lat, lng float64
	}{
		{"nan", math.NaN(), math.NaN()},
		{"inf", math.Inf(1), math.Inf(-1)},
		{"out_of_range", 135, 39},
	} {
		w := f.proofUpload(t, d, id, c.lat, c.lng, false)
		if w.Code == 201 {
			t.Fatalf("%s: صورةٌ بإحداثيّةٍ ليست موضعاً قُبلت: %s", c.name, w.Body.String())
		}
		if w.Code >= 500 {
			t.Fatalf("%s: ردّ %d بدل رفضٍ صريح: %s", c.name, w.Code, w.Body.String())
		}
		var has bool
		_ = f.pool.QueryRow(ctx, `SELECT pod_media_id IS NOT NULL FROM orders WHERE id = $1`, id).Scan(&has)
		if has {
			t.Fatalf("%s: صورةٌ بلا موضعٍ صالحٍ حُفظت إثباتاً", c.name)
		}
	}
}
