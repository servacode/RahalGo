package server

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/routing"
)

// ══════════════════════════════════════════════════════════════════════
// **المسافاتُ على الطريق لا بالهواء** — المالك ٢٠٢٦-١٠-٢٥: «المفروض تحسب حسب الطريق الحقيقي»
// ══════════════════════════════════════════════════════════════════════
//
// كانت بطاقةُ الطلب تقول مسافاتٍ بخطٍّ مستقيم (`ST_Distance`). **وتُستبدَل هنا بطول الطريق** من
// محرّك المسارات نفسِه — من المتجر للزبون، ومن السائق للمتجر، ومن السائق للزبون.
//
// **وذاكرةٌ قصيرةٌ تحمي المحرّك**: النقاطُ الثابتةُ (متجرٌ وزبون) تُحسب مرّةً لعشر دقائق، **وموضعُ
// السائق يُقرَّب إلى ~١٠٠ متر** فلا يُعاد السؤالُ إلّا إن تحرّك فعلاً. **وما تعذّر يبقى بالخطّ
// المستقيم** — رقمٌ تقريبيٌّ خيرٌ من فراغ، ولا تنتظر القائمةُ محرّكاً بطيئاً (مهلةُ ثانيتين).

type roadCacheEntry struct {
	m   float64
	exp time.Time
}

var (
	roadCacheMu sync.Mutex
	roadCache   = map[string]roadCacheEntry{}
)

func roadKey(from, to [2]float64, prec int) string {
	f := "%." + fmt.Sprint(prec) + "f"
	return fmt.Sprintf(f+","+f+">"+f+","+f, from[0], from[1], to[0], to[1])
}

// roadMeters **طولُ الطريق بالمتر** — أو `-1` إن تعذّر.
func (s *Server) roadMeters(ctx context.Context, from, to [2]float64, prec int, ttl time.Duration) float64 {
	key := roadKey(from, to, prec)
	roadCacheMu.Lock()
	if e, ok := roadCache[key]; ok && time.Now().Before(e.exp) {
		roadCacheMu.Unlock()
		return e.m
	}
	roadCacheMu.Unlock()
	rt, err := s.route.Route(ctx,
		routing.Point{Lat: from[0], Lng: from[1]}, routing.Point{Lat: to[0], Lng: to[1]})
	if err != nil || rt == nil || rt.DistanceM <= 0 || math.IsNaN(rt.DistanceM) {
		return -1
	}
	roadCacheMu.Lock()
	if len(roadCache) > 5000 {
		roadCache = map[string]roadCacheEntry{}
	}
	roadCache[key] = roadCacheEntry{m: rt.DistanceM, exp: time.Now().Add(ttl)}
	roadCacheMu.Unlock()
	return rt.DistanceM
}

// fillRoadDistances **يستبدل المسافاتِ المستقيمةَ بطول الطريق** لطلباتٍ جارية — بتوازٍ محدود.
func (s *Server) fillRoadDistances(ctx context.Context, list []orders.Order) {
	if s.route == nil || !s.route.Enabled() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	run := func(fn func()) {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			fn()
		}()
	}
	for i := range list {
		o := &list[i]
		if o.ClosedAt != nil {
			continue
		}
		drop := [2]float64{o.Lat, o.Lng}
		if drop[0] == 0 && drop[1] == 0 {
			continue
		}
		if o.PickupPt != nil && o.LegM >= 0 {
			pk := *o.PickupPt
			run(func() {
				if m := s.roadMeters(ctx, pk, drop, 5, 10*time.Minute); m > 0 {
					o.LegM = m
				}
			})
		}
		if o.DriverPt != nil {
			dr := *o.DriverPt
			if o.PickupPt != nil && o.DriverToPickupM >= 0 {
				pk := *o.PickupPt
				run(func() {
					if m := s.roadMeters(ctx, dr, pk, 3, time.Minute); m > 0 {
						o.DriverToPickupM = m
					}
				})
			}
			if o.DriverToDropM >= 0 {
				run(func() {
					if m := s.roadMeters(ctx, dr, drop, 3, time.Minute); m > 0 {
						o.DriverToDropM = m
					}
				})
			}
		}
	}
	wg.Wait()
}
