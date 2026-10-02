package server

// **النبضةُ المفردةُ تحمل وقتَ التقاطها** (٢٠٢٦-١٠-٠٢) — كانت تُختَم بوقت
// وصولها، **فموضعٌ قديمٌ يُكتب «حديثاً الآن» ويبقى السائقُ مؤهَّلاً للقرب.**

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func (f *driverFixture) locationAge(t *testing.T, driverID string) time.Duration {
	t.Helper()
	var secs float64
	if err := f.pool.QueryRow(context.Background(), `
		SELECT EXTRACT(EPOCH FROM (now() - last_location_at)) FROM users WHERE id = $1`,
		driverID).Scan(&secs); err != nil {
		t.Fatalf("قراءةُ عمر الموضع: %v", err)
	}
	return time.Duration(secs * float64(time.Second))
}

func TestDriverLocation_StoresCaptureTime(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	ping := func(body string) {
		w := f.call(f.srv.handleDriverLocation, http.MethodPost, "/driver/location", "", d,
			[]string{"driver"}, body)
		if w.Code != http.StatusOK {
			t.Fatalf("النبضةُ ردّت %d: %s", w.Code, w.Body.String())
		}
	}

	captured := time.Now().Add(-4 * time.Minute).UTC().Format(time.RFC3339)
	ping(`{"lat":35.95,"lng":39.01,"recorded_at":"` + captured + `"}`)
	if age := f.locationAge(t, d); age < 3*time.Minute {
		t.Fatalf("عمرُ الموضع %v — **نقطةٌ التُقطت قبل أربع دقائق كُتبت حديثة**", age)
	}
	var trackAt time.Time
	if err := f.pool.QueryRow(context.Background(), `
		SELECT max(recorded_at) FROM driver_track WHERE driver_id = $1`, d).Scan(&trackAt); err != nil {
		t.Fatal(err)
	}
	if time.Since(trackAt) < 3*time.Minute {
		t.Errorf("الأثرُ مختومٌ بـ%v — لا بوقت الالتقاط", trackAt)
	}

	// **وبلا وقتٍ يُختَم بالوصول كما كان** — نسخةٌ لم تحدَّث.
	ping(`{"lat":35.95,"lng":39.01}`)
	if age := f.locationAge(t, d); age > 30*time.Second {
		t.Fatalf("بلا وقت: العمرُ %v", age)
	}

	// **ونقطةٌ أقدمُ لا تدهس الأحدث** — وصلتا بغير ترتيبهما.
	ping(`{"lat":35.90,"lng":39.00,"recorded_at":"` + captured + `"}`)
	if age := f.locationAge(t, d); age > 30*time.Second {
		t.Errorf("نقطةٌ قديمةٌ دهست الأحدث — العمرُ %v", age)
	}
}
