package server

// ══════════════════════════════════════════════════════════════════════
// **مراقبة السائقين** (قرار المالك ٢٠٢٦-١١-٠٣) — الكتابة والأبواب
// ══════════════════════════════════════════════════════════════════════
//
// الكتابة: نفس معالجَي الموقع (المفرد والدفعة) بعد ما ينجحوا بحفظ الموضع.
// **وتعثّرها ما بيسقط حفظ الموضع** — الموضع هو المطلوب كل لحظة والأثر للمراجعة.
// الأبواب: `drivers.monitor` وحدها. التحليل في `internal/drivermonitor`.

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/errgroup"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/roadmatch"
)

var errMonitorDriver = httpx.NewError(http.StatusNotFound, "not_found", "errors.not_found")

// ══════════════════════════════════════════════════════════════════════
// **يومُ شغل المنصّة** — «كيلومترات اليوم» (قرار المالك ٢٠٢٦-١١-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// من فتح المنصّة (`platform.day_start_hour`) **لإغلاقها من جدول الدوام** — لا ٩←٩ ولا الدوام —
// وتبدأ من الصفر بالفتح التالي. **وما مشاه بعد الإغلاق** (لحد الفتح التالي) يُعرض وحده: «مشي بعد
// إغلاق المنصة» بكيلومتراته، **وأحمر بعتبة «مشى خارج الدوام»** (`drivers.odometer_off_alert_km`).
// **والمنصّةُ ٢٤ ساعة ⇒ لا إغلاق**، فاليومُ يومُ الحساب ٩←٩ كما كان.
//
// **وقراءاتُ العدّاد** (رقمُ الدرّاجة من الصور) لا تتغيّر — تبقى بأيّام الحساب وبطاقات الدوام.

// workDay نافذةُ اليوم: `From` الفتح · `End` الإغلاق (أو `To`) · `To` الفتحُ التالي.
type workDay struct {
	From, End, To time.Time
	CloseAt       *time.Time
}

// driverWorkDay **يومُ شغل المنصّة** لتاريخ `YYYY-MM-DD` (يومُ الحساب نفسُه بدايةً).
func (s *Server) driverWorkDay(ctx context.Context, day string) (workDay, error) {
	from, to, err := drivermonitor.AccountDayBounds(ctx, s.pg, day)
	if err != nil {
		return workDay{}, err
	}
	wd := workDay{From: from, End: to, To: to}
	if s.platform != nil {
		c, ok, err := s.platform.WorkDayClose(ctx, s.pg, from, to)
		if err != nil {
			return workDay{}, err
		}
		if ok {
			wd.End, wd.CloseAt = c, &c
		}
	}
	return wd, nil
}

// workDayOut **ما يُرسل للوحة** — والإغلاقُ وما بعده.
type workDayOut struct {
	From            time.Time  `json:"from"`
	CloseAt         *time.Time `json:"close_at"`
	To              time.Time  `json:"to"`
	AfterCloseKm    *float64   `json:"after_close_km"`
	AfterCloseAlert bool       `json:"after_close_alert"`
}

// afterClose **قيادةُ ما بعد إغلاق المنصّة** لحد الفتح التالي (أو الآن) — فارغٌ قبل الإغلاق أو
// والمنصّةُ ٢٤ ساعة.
func (s *Server) afterClose(ctx context.Context, driverID string, wd workDay, stores []drivermonitor.Store, now time.Time) (workDayOut, error) {
	out := workDayOut{From: wd.From, CloseAt: wd.CloseAt, To: wd.To}
	if wd.CloseAt == nil || !now.After(*wd.CloseAt) {
		return out, nil
	}
	res, _, err := drivermonitor.DriverDay(ctx, s.pg, driverID, *wd.CloseAt, wd.To, stores, now, s.roadMatchOn(ctx))
	if err != nil {
		return out, err
	}
	km := round1(res.Summary.DriveKmTotal)
	out.AfterCloseKm = &km
	off := float64(s.settings.GetInt(ctx, "drivers.odometer_off_alert_km"))
	out.AfterCloseAlert = km > 0 && (off <= 0 || km >= off)
	return out, nil
}

// trailPoints يحوّل نقاط الدفعة إلى نقاط الأثر.
func trailPoints(pts []trackPoint) []drivermonitor.Point {
	out := make([]drivermonitor.Point, 0, len(pts))
	for _, p := range pts {
		out = append(out, drivermonitor.Point{At: p.At, Lat: p.Lat, Lng: p.Lng,
			AccuracyM: p.AccuracyM, SpeedMps: p.SpeedMps, Mocked: p.Mocked != nil && *p.Mocked})
	}
	return out
}

// recordTrail يكتب الأثر اليومي ويقلّم القديم مرة بالساعة. لا يُسقط شيئا.
//
// **ونقاط متأخرة** (دفعة الطابور بعد انقطاع) أقدم من مؤشّر المطابق ⇒ القطع من عندها بتنعاد
// (`roadmatch.MarkDirty` — تحديث واحد للدفعة). النقطة الحيّة أحدث من ٣٠ ثانية ما بتحتاجه:
// المطابق ما بيقرأ آخر ٣٠ ثانية أصلاً.
func (s *Server) recordTrail(r *http.Request, driverID string, pts []drivermonitor.Point) {
	ctx := r.Context()
	n, err := drivermonitor.Record(ctx, s.pg, driverID, pts)
	if err != nil {
		s.logger.Warn("مراقبة السائقين: تعذّر كتابةُ الأثر", "driver", driverID, "error", err)
		return
	}
	now := time.Now()
	if n > 0 {
		oldest := pts[0].At
		for _, p := range pts {
			if p.At.Before(oldest) {
				oldest = p.At
			}
		}
		if oldest.Before(now.Add(-roadmatch.DefaultOptions().LiveLag)) {
			if err := roadmatch.MarkDirty(ctx, s.pg, driverID, oldest); err != nil {
				s.logger.Warn("مطابقة الطريق: تعذّر تعليمُ النقاط المتأخرة", "driver", driverID, "error", err)
			}
		}
	}
	if drivermonitor.PruneDue(now) {
		days := int(s.settings.GetInt(ctx, "drivers.trail_retention_days"))
		if _, err := drivermonitor.Prune(ctx, s.pg, days); err != nil {
			s.logger.Warn("مراقبة السائقين: تعذّر التقليم", "error", err)
		}
		// قطع الطريق بتعيش قد الأثر بالضبط (قرار المالك: طريق الطلب بينمحى مع الأثر، والأرقام بتضل).
		if _, err := roadmatch.Prune(ctx, s.pg, days); err != nil {
			s.logger.Warn("مطابقة الطريق: تعذّر التقليم", "error", err)
		}
	}
}

// monitorDay يقرأ ?day= (افتراضيه يومُ الحساب الآن) ويردّ معه يومَ الحساب الحاليّ.
//
// **ويومُ المراقبة يومُ الحساب** (قرار ٢٠٢٦-١١-٠٥): من ساعة فتح الدوام لمثلها،
// كعدّاد السائق والنقد — لا من نصف الليل.
func (s *Server) monitorDay(r *http.Request) (day, today string, ok bool) {
	today, err := drivermonitor.AccountDay(r.Context(), s.pg, time.Now())
	if err != nil {
		today = drivermonitor.Today(time.Now())
	}
	day = r.URL.Query().Get("day")
	if day == "" {
		return today, today, true
	}
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return "", today, false
	}
	return day, today, true
}

// handleDriverMonitorList GET /driver-monitor — سائقو اليوم، الأخطر أولا.
func (s *Server) handleDriverMonitorList(w http.ResponseWriter, r *http.Request) {
	day, today, ok := s.monitorDay(r)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	alert := float64(s.settings.GetInt(r.Context(), "drivers.monitor_km_alert"))
	wd, err := s.driverWorkDay(r.Context(), day)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	now := time.Now()
	rows, err := drivermonitor.Overview(r.Context(), s.pg, day, wd.From, wd.End, alert, now, s.roadMatchOn(r.Context()))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **وملخّصُ العدّاد لكلّ سائق** (قرار المالك ٢٠٢٦-١١-٠٥) — العدّاد مقابل التطبيق، ومشيٌ
	// خارج الدوام، وسكّرٌ بلا صورة. **من حساب الكانس** لا تحليلٍ جديد لكلّ سائق كلّ ٣٠ ثانية.
	type rowOut struct {
		drivermonitor.Row
		Odometer odoDay     `json:"odometer"`
		WorkDay  workDayOut `json:"work_day"`
	}
	stores, err := drivermonitor.LoadStores(r.Context(), s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	out := make([]rowOut, 0, len(rows))
	if from, to, err := drivermonitor.AccountDayBounds(r.Context(), s.pg, day); err == nil {
		// **وبالتوازي كقائمة اليوم نفسِها** (`drivermonitor.OverviewWorkers`) — كلُّ عاملٍ صفُّه وحدَه.
		out = out[:len(rows)]
		g, gctx := errgroup.WithContext(r.Context())
		g.SetLimit(drivermonitor.OverviewWorkers)
		for i, row := range rows {
			g.Go(func() error {
				_, odo, err := s.monitorShifts(gctx, row.DriverID, from, to, now, false)
				if err != nil {
					return err
				}
				after, err := s.afterClose(gctx, row.DriverID, wd, stores, now)
				if err != nil {
					return err
				}
				out[i] = rowOut{Row: row, Odometer: odo, WorkDay: after}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"day":            day,
		"today":          today,
		"day_start_hour": s.settings.GetInt(r.Context(), "platform.day_start_hour"),
		"km_alert":       alert,
		"retention_days": s.settings.GetInt(r.Context(), "drivers.trail_retention_days"),
		"drivers":        out,
	})
}

// handleDriverMonitorDay GET /driver-monitor/{id} — يوم سائق: مسار وقصة وأرقام.
func (s *Server) handleDriverMonitorDay(w http.ResponseWriter, r *http.Request) {
	day, today, ok := s.monitorDay(r)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	id := chi.URLParam(r, "id")
	var name, phone string
	if err := s.pg.QueryRow(r.Context(), `
		SELECT full_name, phone FROM users u
		 WHERE u.id::text = $1 AND EXISTS (
		   SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id AND ur.role_code = 'driver')`, id).
		Scan(&name, &phone); err != nil {
		s.respondErr(w, errMonitorDriver)
		return
	}
	stores, err := drivermonitor.LoadStores(r.Context(), s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	wd, err := s.driverWorkDay(r.Context(), day)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, wins, err := drivermonitor.DriverDay(r.Context(), s.pg, id, wd.From, wd.End, stores, time.Now(),
		s.roadMatchOn(r.Context()))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	after, err := s.afterClose(r.Context(), id, wd, stores, time.Now())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	alert := float64(s.settings.GetInt(r.Context(), "drivers.monitor_km_alert"))
	// **ودوامات اليوم** — متى فُتحت وسُكّرت ومين سكّرها وأرقامها (قرار المالك ٢٠٢٦-١١-٠٥).
	dayFrom, dayTo, err := drivermonitor.AccountDayBounds(r.Context(), s.pg, day)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	dayShifts, dayOdo, err := s.monitorShifts(r.Context(), id, dayFrom, dayTo, time.Now(), true)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **وكل طلبٍ بمسافته** (قرار المالك ٢٠٢٦-١١-٠٥) — من القيادة الفعلية: للمتجر وللزبون.
	type order struct {
		ID         string     `json:"id"`
		Number     int64      `json:"number"`
		Store      string     `json:"store"`
		AssignedAt *time.Time `json:"assigned_at"`
		PickedUpAt *time.Time `json:"picked_up_at"`
		EndAt      *time.Time `json:"end_at"`
		EndKind    string     `json:"end_kind"`
		ToPickupM  int        `json:"to_pickup_m"`
		WithOrderM int        `json:"with_order_m"`
		Shared     bool       `json:"shared"`
		GapMin     int        `json:"gap_min"`
	}
	orders := make([]order, 0, len(wins))
	seen := map[string]bool{}
	for _, o := range wins {
		row := order{ID: o.OrderID, Number: o.Number, Store: o.Store, AssignedAt: o.AssignedAt,
			PickedUpAt: o.PickedUpAt, EndAt: o.EndAt, EndKind: o.EndKind}
		// **وطلبٌ له ولايتان يُعرض بمسافته كلّها مرّةً واحدة.**
		if t := res.Tenures[o.OrderID]; t != nil && !seen[o.OrderID] {
			row.ToPickupM, row.WithOrderM = int(math.Round(t.ToPickupM)), int(math.Round(t.WithOrderM))
			row.Shared, row.GapMin = t.Shared, t.GapMin
		}
		seen[o.OrderID] = true
		orders = append(orders, row)
	}
	// **متاجر حول المسار بأسمائها** — تُرسم على الخريطة باسمها (طلب المالك ٢٠٢٦-١١-٠٣).
	type near struct {
		ID   string  `json:"id"`
		Name string  `json:"name"`
		Lat  float64 `json:"lat"`
		Lng  float64 `json:"lng"`
	}
	nearStores := []near{}
	const pad = 0.004 // ~٤٤٠ م
	minLat, maxLat, minLng, maxLng := 90.0, -90.0, 180.0, -180.0
	have := false
	for _, part := range res.Path {
		for _, c := range part.Coords {
			have = true
			minLat, maxLat = math.Min(minLat, c[0]), math.Max(maxLat, c[0])
			minLng, maxLng = math.Min(minLng, c[1]), math.Max(maxLng, c[1])
		}
	}
	if have {
		for _, st := range stores {
			if st.Lat >= minLat-pad && st.Lat <= maxLat+pad && st.Lng >= minLng-pad && st.Lng <= maxLng+pad {
				nearStores = append(nearStores, near{st.ID, st.Name, st.Lat, st.Lng})
			}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"day":      day,
		"today":    today,
		"stores":   nearStores,
		"driver":   map[string]any{"id": id, "name": name, "phone": phone},
		"km_alert": alert,
		"alert":    res.Alert(alert),
		"summary":  res.Summary,
		"path":     res.Path,
		"events":   res.Events,
		"orders":   orders,
		"shifts":   dayShifts,
		"odometer": dayOdo,
		"work_day": after,
	})
}

// recentDefault **من وين بيبلّش طريقه الأخير**: أقدم طلب مفتوح معه (من إسناده)، وإلا آخر ٣٠ دقيقة.
const recentDefault = 30 * time.Minute

// handleDriverMonitorRecent GET /driver-monitor/{id}/recent?since=RFC3339 — **طريقه الحقيقي ورا الماركر**
// بخريطة العمليات (R3، طلب المالك ٢٠٢٦-١١-٠٥). `drivers.monitor` وحدها (المالك والمدير).
//
// `path` من `drivermonitor.Measure` (نفس دالة المسافة): ع الطريق وين انطابق، والجزء بعد آخر مطابقة
// (لحد ٥ دقايق) خط من نقاطه المنظّفة (`pending`) ⇒ الخط دايماً بيوصل للماركر.
func (s *Server) handleDriverMonitorRecent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	var ok bool
	if err := s.pg.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id::text = $1 AND ur.role_code = 'driver')`, id).
		Scan(&ok); err != nil || !ok {
		s.respondErr(w, errMonitorDriver)
		return
	}
	now := time.Now()
	since := now.Add(-recentDefault)
	if q := r.URL.Query().Get("since"); q != "" {
		t, err := time.Parse(time.RFC3339, q)
		if err != nil || t.After(now) || now.Sub(t) > 24*time.Hour {
			s.respondErr(w, errValidation)
			return
		}
		since = t
	} else {
		wins, err := drivermonitor.LoadTenures(ctx, s.pg, id, now.Add(-12*time.Hour), now, now)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		open := false
		for _, wd := range wins {
			if wd.EndAt == nil && wd.AssignedAt != nil && (!open || wd.AssignedAt.Before(since)) {
				since, open = *wd.AssignedAt, true
			}
		}
	}
	wins, err := drivermonitor.LoadTenures(ctx, s.pg, id, since, now, now)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := drivermonitor.Measure(ctx, s.pg, id, since, now.Add(time.Second), wins, nil, now, s.roadMatchOn(ctx))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"since":      since,
		"path":       res.Path,
		"km":         res.Summary.KmTotal,
		"pending_km": res.Summary.PendingKm,
	})
}
