package server

// **قفزةُ موقعٍ مستحيلةٌ تُهمَل** (قرارُ المالك ٢٠٢٦-١٠-٠٣ مساءً) — قِيس في تجربة القبول:
// الرقّة ثمّ دمشق بعد ثانية **فكُتب السائقُ في دمشق.**

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func (f *driverFixture) lastPoint(t *testing.T, driverID string) (lat, lng float64) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(), `
		SELECT ST_Y(last_location::geometry), ST_X(last_location::geometry) FROM users WHERE id = $1`,
		driverID).Scan(&lat, &lng); err != nil {
		t.Fatalf("قراءةُ الموضع: %v", err)
	}
	return lat, lng
}

func TestDriverLocation_ImpossibleJumpIgnored(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	ping := func(lat, lng string, at time.Time) string {
		t.Helper()
		w := f.call(f.srv.handleDriverLocation, http.MethodPost, "/driver/location", "", d, []string{"driver"},
			`{"lat":`+lat+`,"lng":`+lng+`,"recorded_at":"`+at.UTC().Format(time.RFC3339)+`"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("النبضةُ ردّت %d: %s — **والقفزةُ تُهمَل بلا خطأ**", w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	now := time.Now()
	ping("35.9506", "39.0094", now.Add(-2*time.Second))         // الرقّة
	body := ping("33.5138", "36.2765", now.Add(-1*time.Second)) // دمشق بعد ثانية
	if lat, _ := f.lastPoint(t, d); lat < 35 {
		t.Fatalf("كُتب السائقُ في دمشق (%v) بعد ثانيةٍ من الرقّة", lat)
	}
	if !strings.Contains(body, `"saved":false`) {
		t.Errorf("ردُّ القفزة %s — والمتوقّعُ saved:false", body)
	}
	// **وحركةٌ معقولةٌ تُكتب** — ٢٠٠ م في ٢٠ ثانية.
	ping("35.9524", "39.0094", now.Add(18*time.Second))
	if lat, _ := f.lastPoint(t, d); lat < 35.952 {
		t.Fatalf("حركةٌ معقولةٌ أُهملت: %v", lat)
	}

	// **والدفعةُ كذلك** — نقطةٌ في دمشق بين نقطتين في الرقّة.
	base := now.Add(30 * time.Second)
	w := f.call(f.srv.handleDriverLocationBatch, http.MethodPost, "/driver/location/batch", "", d, []string{"driver"},
		`{"points":[`+
			`{"lat":35.9530,"lng":39.0094,"at":"`+base.UTC().Format(time.RFC3339)+`"},`+
			`{"lat":33.5138,"lng":36.2765,"at":"`+base.Add(2*time.Second).UTC().Format(time.RFC3339)+`"}`+
			`]}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"accepted":1`) {
		t.Fatalf("الدفعةُ ردّت %d: %s — والمتوقّعُ قبولَ نقطةٍ واحدة", w.Code, w.Body.String())
	}
	if lat, _ := f.lastPoint(t, d); lat < 35 {
		t.Fatalf("الدفعةُ كتبت السائقَ في دمشق (%v)", lat)
	}
}
