package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/maplearn"
	"github.com/servacode/rahalgo/backend/internal/roadmatch"
)

// Q-R10 — **نداء تتبّع واحد بالمرّة** للخادم كله: المطابق وتعلّم الطرق (عمّاله الأربعة) ع محرّك مزيَّف
// بيعدّ النداءات المتزامنة — ما بيتداخلوا أبداً.
func TestTraceGate_OneCallInFlight(t *testing.T) {
	var inFlight, peak, total int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&inFlight, 1)
		for {
			p := atomic.LoadInt32(&peak)
			if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
				break
			}
		}
		atomic.AddInt32(&total, 1)
		time.Sleep(15 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		_ = json.NewEncoder(w).Encode(map[string]any{"shape": "", "edges": []any{}, "matched_points": []any{}})
	}))
	defer srv.Close()

	pts := []drivermonitor.Point{{Lat: 35.4, Lng: 38.4, At: time.Now()}, {Lat: 35.4, Lng: 38.401, At: time.Now().Add(4 * time.Second)}}
	road := roadmatch.NewClient(srv.URL)
	learn := gateRoads(maplearn.NewMatcher(srv.URL))
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for k := 0; k < 3; k++ {
				if _, err := road.Match(ctx, pts); err != nil {
					t.Errorf("المطابق: %v", err)
				}
			}
		}()
		go func() {
			defer wg.Done()
			for k := 0; k < 3; k++ {
				if _, err := learn.Match(ctx, pts); err != nil {
					t.Errorf("تعلّم الطرق: %v", err)
				}
			}
		}()
	}
	wg.Wait()
	if total != 24 {
		t.Fatalf("النداءات: %d", total)
	}
	if peak != 1 {
		t.Errorf("نداءات متزامنة بأقصاها %d — لازم ١", peak)
	}
	if gateRoads(nil) != nil {
		t.Error("بلا Valhalla تعلّم الطرق لازم يضل nil")
	}
}

// جولة تعلّم الطرق اليوميّة الساعة ٣ الفجر بتوقيت دمشق (مو بعد ساعة من كل إقلاع).
func TestNextDailyAt_ThreeAmDamascus(t *testing.T) {
	loc := drivermonitor.Loc()
	cases := []struct{ now, want time.Time }{
		{time.Date(2026, 11, 5, 14, 0, 0, 0, loc), time.Date(2026, 11, 6, 3, 0, 0, 0, loc)},
		{time.Date(2026, 11, 5, 2, 59, 0, 0, loc), time.Date(2026, 11, 5, 3, 0, 0, 0, loc)},
		{time.Date(2026, 11, 5, 3, 0, 0, 0, loc), time.Date(2026, 11, 6, 3, 0, 0, 0, loc)},
	}
	for _, c := range cases {
		if got := nextDailyAt(c.now.UTC(), 3); !got.Equal(c.want) {
			t.Errorf("%v ⇒ %v (المتوقَّع %v)", c.now, got, c.want)
		}
	}
}
