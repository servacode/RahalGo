package server

// ══════════════════════════════════════════════════════════════════════
// **«لم تكتمل المعالم الـ٥ الخاصة بك بعد» — وملخّصُ المالك كلَّ صبح** (قرارُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
//  ١ · **بالساعة** (`drivers.landmarks_reminder_hour`، افتراضُها ٦ المساء بدمشق): كلُّ سائقٍ
//      على الدوام، حسابُه فعّال، فتح دوامَه منذ نصف ساعةٍ فأكثر، **وعدُّه دون المطلوب.**
//  ٢ · **وعند إغلاق دوامه بنفسه** (`drivers.landmarks_remind_on_shift_end`) إن لم يكمل —
//      لا إغلاقُ الإدارة ولا الطارئ ولا الحادث ولا تركُ الطلب.
//  ٣ · **وملخّصُ أمس للمالك** ومن يملك `settings.general.manage` — بعد فتح اليوم بربع ساعة.
//
// **ومرّةً لكلّ سائقٍ ويومٍ وسبب** (`landmark_quota_reminders`)، **ومرّةً لليوم للملخّص**
// (`landmark_daily_digests`) — فإعادةُ تشغيل الخادم أو جولتان لا تكرّران شيئاً. **وعلى الجوّال
// تذكيرُ الإغلاق يحلّ محلَّ تذكير الساعة** في اليوم نفسِه: مفتاحُ الإشعار نوعُه و`entity_id`.

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

const (
	// landmarkRemindMinShift **من فتح دوامَه للتوّ لا يُزعَج.**
	landmarkRemindMinShift = 30 * time.Minute
	// landmarkDigestAfter **ملخّصُ أمس بعد فتح اليوم بربع ساعة.**
	landmarkDigestAfter = 15 * time.Minute
	// landmarkRemindBatch **سقفُ الجولة الواحدة** — والباقي في الجولة التالية.
	landmarkRemindBatch = 2000
)

// platformDayStart **لحظةُ بدء يوم المنصّة** — ساعةُ فتح الدوام بدمشق من `day` (YYYY-MM-DD).
func platformDayStart(day string, startHour int64) (time.Time, error) {
	loc := platform.Location()
	d, err := time.ParseInLocation("2006-01-02", day, loc)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(d.Year(), d.Month(), d.Day(), int(startHour), 0, 0, 0, loc), nil
}

// landmarkReminderAt **ساعةُ التذكير في يوم المنصّة ونهايةُ لحاقها** — **وساعةٌ قبل فتح الدوام**
// (٢ ليلاً والفتحُ ٩) **تقع في آخر اليوم نفسِه**: صباحَ الغد بالتقويم.
//
// **واللحاقُ إلى منتصف الليل** (مراجعة ٢٠٢٦-١١-٠٥): من فتح دوامَه السابعةَ مساءً يُذكَّر بعد
// نصف ساعة، **ومن فتح الثامنةَ صباحاً لا يُقال له «أرسلت ٠ من ٥ اليوم» عن أمسِه** ثمّ يعود العدُّ
// صفراً بعد ساعة؛ **ونشرٌ للخادم بعد منتصف الليل لا يُذكّر أحداً.** وساعةٌ بعد منتصف الليل
// لحاقُها إلى فتح الدوام — آخرِ يومها.
func landmarkReminderAt(day string, startHour, hour int64) (at, until time.Time, err error) {
	start, err := platformDayStart(day, startHour)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	at = time.Date(start.Year(), start.Month(), start.Day(), int(hour), 0, 0, 0, start.Location())
	if hour < startHour {
		at = at.AddDate(0, 0, 1)
	}
	until = time.Date(at.Year(), at.Month(), at.Day()+1, 0, 0, 0, 0, at.Location())
	if end := start.AddDate(0, 0, 1); end.Before(until) {
		until = end
	}
	return at, until, nil
}

// RemindLandmarkQuota **جولةُ تذكير الساعة** — تُنادى دوريّاً وفي الاختبار مباشرةً، وتردّ كم ذُكِّر.
func (s *Server) RemindLandmarkQuota(ctx context.Context, now time.Time) (int, error) {
	quota := s.landmarkQuota(ctx)
	if quota <= 0 {
		return 0, nil
	}
	startHour := s.dayStartHour(ctx)
	day := siteDay(now, startHour)
	at, until, err := landmarkReminderAt(day, startHour, s.settings.GetInt(ctx, "drivers.landmarks_reminder_hour"))
	if err != nil {
		return 0, err
	}
	if now.Before(at) || !now.Before(until) {
		return 0, nil
	}
	rows, err := s.pg.Query(ctx, `
		SELECT u.id::text,
		       (SELECT count(*) FROM landmark_suggestions l
		         WHERE l.suggested_by = u.id AND l.platform_day = $1::date
		           AND l.status IN ('pending', 'approved'))::int
		  FROM users u
		  JOIN user_roles r ON r.user_id = u.id AND r.role_code = 'driver'
		 WHERE u.on_shift AND u.status = 'active' AND u.deleted_at IS NULL
		   AND COALESCE(u.shift_started_at, '-infinity') <= $2
		   AND NOT EXISTS (SELECT 1 FROM landmark_quota_reminders q
		                    WHERE q.driver_id = u.id AND q.day = $1::date AND q.reason = 'hour')
		 LIMIT `+strconv.Itoa(landmarkRemindBatch), day, now.Add(-landmarkRemindMinShift))
	if err != nil {
		return 0, err
	}
	type due struct {
		id   string
		sent int
	}
	var list []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.sent); err != nil {
			rows.Close()
			return 0, err
		}
		if d.sent < quota {
			list = append(list, d)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	sent := 0
	for _, d := range list {
		ok, err := s.remindLandmarks(ctx, d.id, day, "hour", d.sent, quota)
		if err != nil {
			return sent, err
		}
		if ok {
			sent++
		}
	}
	return sent, nil
}

// remindLandmarksOnShiftEnd **سكّر دوامَه بنفسه ولم يكمل** — مرّةً باليوم، والإعدادُ يطفئه.
//
// **ويُقاس ما قضاه من دوامه داخل يوم المنصّة الحاليّ** — من الأحدث بين فتح دوامه وبدء اليوم
// إلى الإغلاق، **ونصفُ ساعةٍ فأكثر يُذكَّر.** فمن سكّر التاسعةَ وخمسَ دقائقَ صباحاً بعد ليلته
// لا يُقال له «أرسلت ٠ من ٥ اليوم» واليومُ بدأ للتوّ، **ومن فتح الثامنةَ وسكّر الرابعةَ عصراً
// يُذكَّر** — كان شرطُ «فُتح قبل بدء اليوم» يُسقطه (مراجعة ٢٠٢٦-١١-٠٥).
func (s *Server) remindLandmarksOnShiftEnd(ctx context.Context, driverID string, startedAt *time.Time, now time.Time) {
	quota := s.landmarkQuota(ctx)
	if quota <= 0 || !s.settings.GetBool(ctx, "drivers.landmarks_remind_on_shift_end") {
		return
	}
	startHour := s.dayStartHour(ctx)
	day := siteDay(now, startHour)
	begin, err := platformDayStart(day, startHour)
	if err != nil {
		s.logger.Warn("تذكيرُ المعالم عند الإغلاق", "driver", driverID, "error", err)
		return
	}
	if startedAt != nil && startedAt.After(begin) {
		begin = *startedAt
	}
	if now.Sub(begin) < landmarkRemindMinShift {
		return
	}
	n, err := landmarkCountedOn(ctx, s.pg, driverID, day)
	if err != nil {
		s.logger.Warn("تذكيرُ المعالم عند الإغلاق", "driver", driverID, "error", err)
		return
	}
	if n >= quota {
		return
	}
	if _, err := s.remindLandmarks(ctx, driverID, day, "shift_end", n, quota); err != nil {
		s.logger.Warn("تذكيرُ المعالم عند الإغلاق", "driver", driverID, "error", err)
	}
}

// remindLandmarks **يختم ثمّ يُشعر** — والختمُ مفتاحٌ (سائق · يوم · سبب)، فلا إشعارَ ثانٍ.
func (s *Server) remindLandmarks(ctx context.Context, driverID, day, reason string, sent, quota int) (bool, error) {
	tag, err := s.pg.Exec(ctx, `
		INSERT INTO landmark_quota_reminders (driver_id, day, reason, sent_count)
		VALUES ($1::uuid, $2::date, $3, $4) ON CONFLICT DO NOTHING`, driverID, day, reason, sent)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	body := notifTitles.landmarkQuotaBody
	if reason == "shift_end" {
		body = notifTitles.landmarkQuotaShiftBody
	}
	kv := []string{"{sent}", strconv.Itoa(sent), "{quota}", strconv.Itoa(quota)}
	// **ونوعُ «الحساب» للسائق وحدَه** — كـ«موقعك ما عم يوصلنا»؛ والضغطُ يفتح الرئيسيّة.
	s.notify.Notify(ctx, notifications.Input{
		UserID: driverID, Kind: notifications.KindAccount,
		Title:  landmarkText(notifTitles.landmarkQuotaTitle, kv...),
		Body:   landmarkText(body, kv...),
		Entity: "landmarks", EntityID: day,
		Apps: []string{notifications.AppDriver},
	})
	return true, nil
}

// landmarkDriverDay **سائقٌ في يوم** — ما أرسله وحالُه وتذكيرُه.
type landmarkDriverDay struct {
	DriverID  string `json:"driver_id"`
	Name      string `json:"name"`
	Sent      int    `json:"sent"`
	New       int    `json:"new"`
	Duplicate int    `json:"duplicate"`
	Pending   int    `json:"pending"`
	Approved  int    `json:"approved"`
	Rejected  int    `json:"rejected"`
	// Worked **اشتغل ذلك اليوم** — له أثرٌ في `driver_trail` (يُكتب على الدوام وحدَه).
	Worked bool `json:"worked"`
	// Completed **بلغ المطلوب** — والعدُّ المطفأُ لا يُكمَل.
	Completed          bool       `json:"completed"`
	RemindedHourAt     *time.Time `json:"reminded_hour_at"`
	RemindedShiftEndAt *time.Time `json:"reminded_shift_end_at"`
}

// landmarkDayTotals **مربّعاتُ اليوم.**
type landmarkDayTotals struct {
	Sent      int `json:"sent"`
	New       int `json:"new"`
	Duplicate int `json:"duplicate"`
	Pending   int `json:"pending"`
	Approved  int `json:"approved"`
	Rejected  int `json:"rejected"`
	// Drivers **من في الجدول** — أرسل أو اشتغل.
	Drivers int `json:"drivers"`
	// Senders **من أرسل جديداً.**
	Senders    int `json:"senders"`
	Completed  int `json:"completed"`
	Incomplete int `json:"incomplete"`
}

// landmarkDaily **يومُ المعالم سائقاً سائقاً** — ومن اشتغل ولم يرسل شيئاً بأصفاره.
//
// **ومن لم يكمل أوّلاً** — هو من يُسأل عنه، ثمّ الأكثرُ جديداً.
func (s *Server) landmarkDaily(ctx context.Context, day string, startHour int64, quota int) ([]landmarkDriverDay, landmarkDayTotals, error) {
	var tot landmarkDayTotals
	start, err := platformDayStart(day, startHour)
	if err != nil {
		return nil, tot, err
	}
	rows, err := s.pg.Query(ctx, `
		WITH s AS (
			SELECT l.suggested_by AS driver_id,
			       count(*)::int AS sent,
			       count(*) FILTER (WHERE l.status IN ('pending', 'approved'))::int AS fresh,
			       count(*) FILTER (WHERE l.status = 'duplicate')::int AS dup,
			       count(*) FILTER (WHERE l.status = 'pending')::int AS pending,
			       count(*) FILTER (WHERE l.status = 'approved')::int AS approved,
			       count(*) FILTER (WHERE l.status = 'rejected')::int AS rejected
			  FROM landmark_suggestions l
			 WHERE l.platform_day = $1::date AND l.suggested_by IS NOT NULL
			 GROUP BY l.suggested_by
		), w AS (
			SELECT DISTINCT t.driver_id FROM driver_trail t
			 WHERE t.recorded_at >= $2 AND t.recorded_at < $3
		), ids AS (
			SELECT driver_id FROM s UNION SELECT driver_id FROM w
		)
		SELECT u.id::text, COALESCE(u.full_name, ''),
		       COALESCE(s.sent, 0), COALESCE(s.fresh, 0), COALESCE(s.dup, 0),
		       COALESCE(s.pending, 0), COALESCE(s.approved, 0), COALESCE(s.rejected, 0),
		       w.driver_id IS NOT NULL,
		       (SELECT q.sent_at FROM landmark_quota_reminders q
		         WHERE q.driver_id = u.id AND q.day = $1::date AND q.reason = 'hour'),
		       (SELECT q.sent_at FROM landmark_quota_reminders q
		         WHERE q.driver_id = u.id AND q.day = $1::date AND q.reason = 'shift_end')
		  FROM ids
		  JOIN users u ON u.id = ids.driver_id
		  LEFT JOIN s ON s.driver_id = ids.driver_id
		  LEFT JOIN w ON w.driver_id = ids.driver_id`, day, start, start.AddDate(0, 0, 1))
	if err != nil {
		return nil, tot, err
	}
	defer rows.Close()
	out := []landmarkDriverDay{}
	for rows.Next() {
		var d landmarkDriverDay
		if err := rows.Scan(&d.DriverID, &d.Name, &d.Sent, &d.New, &d.Duplicate, &d.Pending,
			&d.Approved, &d.Rejected, &d.Worked, &d.RemindedHourAt, &d.RemindedShiftEndAt); err != nil {
			return nil, tot, err
		}
		d.Completed = quota > 0 && d.New >= quota
		tot.Sent += d.Sent
		tot.New += d.New
		tot.Duplicate += d.Duplicate
		tot.Pending += d.Pending
		tot.Approved += d.Approved
		tot.Rejected += d.Rejected
		tot.Drivers++
		if d.New > 0 {
			tot.Senders++
		}
		if d.Completed {
			tot.Completed++
		} else if quota > 0 {
			tot.Incomplete++
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, tot, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Completed != out[j].Completed {
			return !out[i].Completed
		}
		if out[i].New != out[j].New {
			return out[i].New > out[j].New
		}
		return out[i].Name < out[j].Name
	})
	return out, tot, nil
}

// DigestLandmarks **ملخّصُ أمس للمالك** — مرّةً لليوم بعد فتح اليوم بربع ساعة. ويردّ `true` إن أُرسل.
//
// **ويومٌ بلا خبرٍ يُختم ولا يُرسَل** — لا معلمَ أُرسل ولا سائقَ اشتغل.
func (s *Server) DigestLandmarks(ctx context.Context, now time.Time) (bool, error) {
	startHour := s.dayStartHour(ctx)
	today := siteDay(now, startHour)
	start, err := platformDayStart(today, startHour)
	if err != nil {
		return false, err
	}
	if now.Before(start.Add(landmarkDigestAfter)) {
		return false, nil
	}
	yesterday := start.AddDate(0, 0, -1).Format("2006-01-02")
	var done bool
	if err := s.pg.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM landmark_daily_digests WHERE day = $1::date)`,
		yesterday).Scan(&done); err != nil || done {
		return false, err
	}
	quota := s.landmarkQuota(ctx)
	_, tot, err := s.landmarkDaily(ctx, yesterday, startHour, quota)
	if err != nil {
		return false, err
	}
	var queue int
	if err := s.pg.QueryRow(ctx, `SELECT count(*) FROM landmark_suggestions WHERE status = 'pending'`).Scan(&queue); err != nil {
		return false, err
	}
	tag, err := s.pg.Exec(ctx, `INSERT INTO landmark_daily_digests (day) VALUES ($1::date) ON CONFLICT DO NOTHING`, yesterday)
	if err != nil || tag.RowsAffected() == 0 {
		return false, err
	}
	if tot.Sent == 0 && tot.Drivers == 0 {
		return false, nil
	}
	body := landmarkText(notifTitles.landmarkDigestBody,
		"{new}", strconv.Itoa(tot.New), "{drivers}", strconv.Itoa(tot.Senders),
		"{duplicate}", strconv.Itoa(tot.Duplicate), "{pending}", strconv.Itoa(queue))
	if quota > 0 {
		body += landmarkText(notifTitles.landmarkDigestIncomplete,
			"{quota}", strconv.Itoa(quota), "{incomplete}", strconv.Itoa(tot.Incomplete))
	}
	s.notify.NotifyCapsRoles(ctx, []string{string(authz.SettingsGeneralManage)},
		[]string{authz.RoleOwnerSuperAdmin}, notifications.Input{
			Kind:   notifications.KindAccount,
			Title:  notifTitles.landmarkDigestTitle,
			Body:   body,
			Entity: "landmarks", EntityID: yesterday,
			Href: "/dashboard/map-learning?section=landmarks&day=" + yesterday,
		})
	return true, nil
}

// RunLandmarkQuota **كلَّ خمس دقائق**: تذكيرُ الساعة وملخّصُ المالك — والختمُ يمنع التكرار.
func (s *Server) RunLandmarkQuota(ctx context.Context, interval time.Duration) {
	round := func(now time.Time) {
		if _, err := s.RemindLandmarkQuota(ctx, now); err != nil {
			s.logger.Warn("تذكيرُ المعالم", "error", err)
		}
		if _, err := s.DigestLandmarks(ctx, now); err != nil {
			s.logger.Warn("ملخّصُ المعالم", "error", err)
		}
	}
	round(time.Now())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			round(t)
		}
	}
}
