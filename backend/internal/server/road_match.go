package server

// ══════════════════════════════════════════════════════════════════════
// **مطابق مسار السائق ع الطرق** (طلب المالك ٢٠٢٦-١١-٠٥) — الدورة والبوّابة
// ══════════════════════════════════════════════════════════════════════
//
// «والطريق يحسب حسب طريق السائق مو ع كيفنا». الشغل كله بـ`internal/roadmatch`؛ هون:
//
//   - `RunRoadMatcher` دورة كل دقيقة — **ولا شي** والإعداد `drivers.road_match_enabled` مطفي (الافتراض).
//   - `gatedRoadsMatcher` تعلّم الطرق بيمرق من نفس البوّابة ⇒ نداء تتبّع واحد بالمرّة للخادم كله
//     (Valhalla عنا بخيط واحد).
//   - حالة المطابق سطر بصفحة صحّة التشغيل — **مو جزء من الحكم**.

import (
	"context"
	"sync"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/maplearn"
	"github.com/servacode/rahalgo/backend/internal/roadmatch"
)

var roadWorkerMu sync.Mutex

// roadWorker المطابق الوحيد للخادم (بينبني أول مرّة بينطلب).
func (s *Server) roadWorker() *roadmatch.Worker {
	roadWorkerMu.Lock()
	defer roadWorkerMu.Unlock()
	if s.roadW == nil {
		var m roadmatch.Matcher
		var tiles func(context.Context) time.Time
		if c := roadmatch.NewClient(s.cfg.ValhallaURL); c != nil {
			m, tiles = c, c.TilesTime
		}
		w := roadmatch.NewWorker(s.pg, m, s.logger)
		w.Tiles = tiles
		w.Busy = maplearn.RoadsBusy
		s.roadW = w
	}
	return s.roadW
}

// roadMatchOn **المطابقة شغّالة؟** (`drivers.road_match_enabled`، افتراضه لأ).
func (s *Server) roadMatchOn(ctx context.Context) bool {
	return s.settings.GetBool(ctx, "drivers.road_match_enabled")
}

// RunRoadMatcher **دورة المطابق** — كل `interval`.
func (s *Server) RunRoadMatcher(ctx context.Context, interval time.Duration) {
	w := s.roadWorker()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tctx, cancel := context.WithTimeout(ctx, interval*3)
			res, err := w.Tick(tctx, time.Now(), s.roadMatchOn(tctx))
			cancel()
			if err != nil && ctx.Err() == nil {
				s.logger.Warn("مطابقة الطريق: تعثّرت الدورة", "error", err)
				continue
			}
			if res.Calls > 0 {
				s.logger.Debug("مطابقة الطريق: دورة", "calls", res.Calls, "rows", res.Rows,
					"no_match", res.NoMatch, "backfill", res.Backfill)
			}
		}
	}
}

// gatedRoadsMatcher **مطابق تعلّم الطرق عبر بوّابة التتبّع** — عمّاله الأربعة بيصيروا نداء بالمرّة.
type gatedRoadsMatcher struct{ inner maplearn.Matcher }

func (g gatedRoadsMatcher) Match(ctx context.Context, pts []drivermonitor.Point) (maplearn.MatchResult, error) {
	release, err := roadmatch.Acquire(ctx)
	if err != nil {
		return maplearn.MatchResult{}, err
	}
	defer release()
	return g.inner.Match(ctx, pts)
}

// gateRoads يلفّ مطابق تعلّم الطرق بالبوّابة (nil بيضل nil).
func gateRoads(m maplearn.Matcher) maplearn.Matcher {
	if m == nil {
		return nil
	}
	return gatedRoadsMatcher{inner: m}
}

// roadMatchStatus حالة المطابق لصفحة صحّة التشغيل.
func (s *Server) roadMatchStatus(ctx context.Context) roadmatch.Status {
	now := time.Now()
	w := s.roadWorker()
	st := w.Status(now)
	st.Enabled = s.roadMatchOn(ctx)
	if !st.Enabled {
		st.State = "off"
	} else if st.State == "" || st.State == "off" {
		st.State = "ok"
		if w.Matcher == nil {
			st.State = "no_engine"
		}
	}
	if d, err := roadmatch.Backlog(ctx, s.pg); err == nil {
		st.BacklogDays = d
	}
	return st
}

// nextDailyAt **أول وقت بعد `now`** على الساعة `hour` بتوقيت دمشق.
func nextDailyAt(now time.Time, hour int) time.Time {
	loc := drivermonitor.Loc()
	n := now.In(loc)
	at := time.Date(n.Year(), n.Month(), n.Day(), hour, 0, 0, 0, loc)
	if !at.After(n) {
		at = at.AddDate(0, 0, 1)
	}
	return at
}
