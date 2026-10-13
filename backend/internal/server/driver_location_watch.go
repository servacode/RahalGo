package server

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/push"
)

// ══════════════════════════════════════════════════════════════════════
// **سائقٌ في الدوام انقطع موقعُه — يُوقَظ تطبيقُه ولا يُغلَق دوامُه**
// ══════════════════════════════════════════════════════════════════════
//
// (قِيس على الإنتاج ٢٠٢٦-١٠-١٣: الطلبُ #1004 يبحث عن سائق، والسائقُ الوحيدُ
//  في الدوام على ثمانية أمتار منه — **وآخرُ موضعٍ له قبل أربع ساعات** فلا
//  يُعرض عليه شيء، ولا أحدَ يعلم.)
//
// **وقرارُ المالك**: «ما يصير يتسكّر الدوام تلقائي بدون ما يسكّره السائق —
// بس لازم يظلّ الموقع شغّال ويصله الإشعارات بدون ما يضطرّ يفتح التطبيق.»
//
// ١ · بعد **ثلاث دقائق** صمت: **خبرٌ صامتٌ عاجل** (`location_wake`) يُعيد
//     تطبيقُه به خدمةَ الموقع بلا إشعارٍ ولا فتح — والعاجلُ وحدَه يُسمح له
//     في أندرويد بإقلاع خدمةٍ أماميّةٍ من الخلفيّة. ويُعاد كلَّ ثلاث دقائق.
// ٢ · بعد **عشر دقائق**: الإيقاظُ لم يكفِ — فعلّةٌ لا يحلّها التطبيقُ وحدَه
//     (إذنٌ سُحب، موقعٌ مطفأ، جوّالٌ مطفأ). **إشعارٌ ظاهرٌ للسائق** يقول ما
//     يفعل، **وتنبيهٌ للمكتب** — مرّةً لكلّ انقطاع، ويُذكَّر السائقُ كلَّ ربع ساعة.

const (
	locWakeAfter    = 3 * time.Minute
	locWakeEvery    = 3 * time.Minute
	locNoticeAfter  = 10 * time.Minute
	locNoticeEvery  = 15 * time.Minute
	locWakeMinApp   = "0.4.6" // أوّلُ نسخةٍ تفهم الإيقاظَ الصامت
	locWakeKind     = "location_wake"
	locWatchBatchSz = 2000
)

type locWatchState struct {
	wakeAt, noticeAt time.Time
	opsTold          bool
}

type locWatch struct {
	mu sync.Mutex
	by map[string]*locWatchState
}

// staleShiftDriver سائقٌ في الدوام وصمتُه.
type staleShiftDriver struct {
	ID, Name string
	Silent   time.Duration
}

// staleShiftDrivers **من في الدوام وموقعُه أقدمُ من الحدّ** — **الأحدثُ انقطاعاً
// أوّلاً**: هو الأقربُ إلى العودة بإيقاظ، ومن انقطع منذ ساعاتٍ لا يُقدَّم عليه.
//
// **والشرطُ في الاستعلام لا بعده**: كانت تُقرأ أوّلُ خمسمئةٍ في الدوام بلا
// ترتيب ثمّ تُصفّى — **فمع كثرة السائقين يفوت الصامتُ الذي يُبحث عنه.**
func (s *Server) staleShiftDrivers(ctx context.Context) ([]staleShiftDriver, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT u.id::text, COALESCE(NULLIF(u.full_name, ''), u.phone::text),
		       EXTRACT(EPOCH FROM now() - COALESCE(u.last_location_at, u.shift_started_at, now()))::bigint AS silent
		FROM users u
		JOIN user_roles r ON r.user_id = u.id AND r.role_code = 'driver'
		WHERE u.on_shift AND u.status = 'active' AND u.deleted_at IS NULL
		  AND COALESCE(u.last_location_at, u.shift_started_at, now()) < now() - $1::interval
		ORDER BY silent ASC
		LIMIT `+strconv.Itoa(locWatchBatchSz), locWakeAfter.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []staleShiftDriver
	for rows.Next() {
		var d staleShiftDriver
		var sec int64
		if rows.Scan(&d.ID, &d.Name, &sec) != nil {
			continue
		}
		d.Silent = time.Duration(sec) * time.Second
		out = append(out, d)
	}
	return out, rows.Err()
}

// WatchDriverLocations **نبضةٌ واحدة** — تُنادى دوريّاً، وفي الاختبار مباشرةً.
func (s *Server) WatchDriverLocations(ctx context.Context, now time.Time) {
	stale, err := s.staleShiftDrivers(ctx)
	if err != nil {
		s.logger.Warn("location watch", "err", err)
		return
	}
	w := &s.locWatch
	w.mu.Lock()
	if w.by == nil {
		w.by = map[string]*locWatchState{}
	}
	staleSet := map[string]bool{}
	for _, d := range stale {
		staleSet[d.ID] = true
	}
	// **ومن عاد موقعُه أو خرج من الدوام يُنسى** — فالانقطاعُ التالي يُعَدّ جديداً.
	for id := range w.by {
		if !staleSet[id] {
			delete(w.by, id)
		}
	}
	type act struct {
		d            staleShiftDriver
		wake, notice bool
		tellOps      bool
	}
	var acts []act
	for _, d := range stale {
		st := w.by[d.ID]
		if st == nil {
			st = &locWatchState{}
			w.by[d.ID] = st
		}
		a := act{d: d}
		if now.Sub(st.wakeAt) >= locWakeEvery {
			a.wake, st.wakeAt = true, now
		}
		if d.Silent >= locNoticeAfter && now.Sub(st.noticeAt) >= locNoticeEvery {
			a.notice, st.noticeAt = true, now
			if !st.opsTold {
				a.tellOps, st.opsTold = true, true
			}
		}
		acts = append(acts, a)
	}
	w.mu.Unlock()

	for _, a := range acts {
		if a.wake && s.push != nil {
			s.push.SendToUser(ctx, a.d.ID, push.Message{
				Data:          map[string]string{"kind": locWakeKind},
				Urgent:        true,
				Apps:          []string{notifications.AppDriver},
				TTL:           2 * time.Minute,
				Collapse:      locWakeKind,
				MinAppVersion: locWakeMinApp,
			})
		}
		if a.notice {
			s.notify.Notify(ctx, notifications.Input{
				UserID: a.d.ID, Kind: notifications.KindAccount,
				Title: notifTitles.driverLocationLost,
				Body:  notifTitles.driverLocationLostBody,
				Apps:  []string{notifications.AppDriver},
			})
		}
		if a.tellOps {
			s.notify.NotifyOps(ctx, notifications.Input{
				Kind:   notifications.KindAccount,
				Title:  notifTitles.opsDriverLocationLost,
				Body:   a.d.Name + " — " + strconv.Itoa(int(a.d.Silent.Minutes())) + " " + minutesWord,
				Entity: "driver", EntityID: a.d.ID, Href: "/dashboard/drivers",
			})
		}
	}
}

// RunDriverLocationWatch **كلَّ دقيقة.**
func (s *Server) RunDriverLocationWatch(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			s.WatchDriverLocations(ctx, t)
		}
	}
}
