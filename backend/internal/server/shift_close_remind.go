package server

// ══════════════════════════════════════════════════════════════════════
// **«المنصة أغلقت، يجب عليك إغلاق دوامك»** — قرار المالك ٢٠٢٦-١١-٠٦
// ══════════════════════════════════════════════════════════════════════
//
// دوامٌ بقي مفتوحاً طول الليل **يضلّ دواماً واحداً** — لا يُقسم الصبح ولا يُسكَّر آليّاً.
//
//	١ · **لما تسكّر المنصّة** (جدولُ الدوام إن سرى، أو زرّ «مسكّرين» — `platform.HoursClosedAt`)
//	    كلُّ سائقٍ لسّا بدوامه يوصله **إشعارٌ واحد**: «المنصة أغلقت، يجب عليك إغلاق دوامك».
//	٢ · **وإذا ما سكّره** — لما تفتح المنصّة **باليوم التالي** (يومُ المنصّة تغيّر) ودوامُه
//	    نفسُه لسّا مفتوح: **تذكيرٌ واحد** إنّ دوامه من أمس لسّا مفتوح.
//
// **ولا تسكيرٌ آليّ ولا إلحاح ولا تصعيدٌ للعمليّات.** **ومرّةً لكلّ سائقٍ وإغلاق**: الختمُ
// (`driver_shift_reminders`: سائق · يوم المنصّة · نوع) يُكتب قبل الإشعار، فجولتان أو إعادةُ
// تشغيل الخادم لا تكرّران — كختم تذكير المعالم (`landmark_remind.go`).
//
// **و«مفتوحين» بالزرّ أو جدولٌ لا يسري (٢٤ ساعة) ⇒ لا إغلاقَ أصلاً فلا شيءَ يُرسل.** والإيقافُ
// المؤقّتُ ليس إغلاقاً للدوام فلا يُرسل به شيء.
//
// **و«اليوم التالي» يومُ المنصّة** (`platform.day_start_hour`) لا التقويم: منصّةٌ تفتح الثامنة
// ويومُها يبدأ التاسعة يُذكَّر سائقُها التاسعة.

import (
	"context"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// shiftRemindLookback **تذكيرُ اليوم التالي لإغلاقاتٍ قريبة وحدها** — دوامٌ مفتوحٌ من أسبوع لا
// يُذكَّر عن كلّ ليلةٍ فاتت يومَ يُنشر الخادم.
const shiftRemindLookback = 2

// shiftRemindBatch سقفُ الجولة — والباقي بالجولة التالية.
const shiftRemindBatch = 2000

// RemindOpenShifts **جولةُ تذكير الدوام المفتوح** — تُنادى دوريّاً وفي الاختبار مباشرةً، وتردّ
// كم وصله «المنصة أغلقت» وكم وصله «دوامك من أمس لسّا مفتوح».
func (s *Server) RemindOpenShifts(ctx context.Context, now time.Time) (closed, nextDay int, err error) {
	if s.platform == nil {
		return 0, 0, nil
	}
	shut, err := s.platform.HoursClosedAt(ctx, s.pg, now)
	if err != nil {
		return 0, 0, err
	}
	day := siteDay(now, s.dayStartHour(ctx))
	if shut {
		ids, err := s.stampShiftReminders(ctx, `
			INSERT INTO driver_shift_reminders (driver_id, day, kind, shift_id, sent_at)
			SELECT u.id, $1::date, 'platform_closed', sh.id, $2
			  FROM users u
			  JOIN driver_shifts sh ON sh.driver_id = u.id AND sh.closed_at IS NULL
			 WHERE u.on_shift AND u.status = 'active' AND u.deleted_at IS NULL
			   AND EXISTS (SELECT 1 FROM user_roles r WHERE r.user_id = u.id AND r.role_code = 'driver')
			   AND NOT EXISTS (SELECT 1 FROM driver_shift_reminders q
			                    WHERE q.driver_id = u.id AND q.day = $1::date AND q.kind = 'platform_closed')
			 LIMIT `+strconv.Itoa(shiftRemindBatch)+`
			ON CONFLICT DO NOTHING
			RETURNING driver_id::text, shift_id::text`, day, now)
		if err != nil {
			return 0, 0, err
		}
		for _, x := range ids {
			s.notifyShiftReminder(ctx, x[0], x[1], notifTitles.shiftPlatformClosed, notifTitles.shiftPlatformClosedBody)
		}
		return len(ids), 0, nil
	}
	ids, err := s.stampShiftReminders(ctx, `
		INSERT INTO driver_shift_reminders (driver_id, day, kind, shift_id, sent_at)
		SELECT r.driver_id, r.day, 'next_day', r.shift_id, $2
		  FROM driver_shift_reminders r
		  JOIN driver_shifts sh ON sh.id = r.shift_id AND sh.closed_at IS NULL
		  JOIN users u ON u.id = r.driver_id
		 WHERE r.kind = 'platform_closed'
		   AND r.day < $1::date AND r.day >= $1::date - `+strconv.Itoa(shiftRemindLookback)+`
		   AND u.on_shift AND u.status = 'active' AND u.deleted_at IS NULL
		   AND NOT EXISTS (SELECT 1 FROM driver_shift_reminders q
		                    WHERE q.driver_id = r.driver_id AND q.day = r.day AND q.kind = 'next_day')
		 LIMIT `+strconv.Itoa(shiftRemindBatch)+`
		ON CONFLICT DO NOTHING
		RETURNING driver_id::text, shift_id::text`, day, now)
	if err != nil {
		return 0, 0, err
	}
	for _, x := range ids {
		s.notifyShiftReminder(ctx, x[0], x[1], notifTitles.shiftStillOpen, notifTitles.shiftStillOpenBody)
	}
	return 0, len(ids), nil
}

// stampShiftReminders **يختم أوّلاً** — وما رجع من الختم وحدَه يُشعَر.
func (s *Server) stampShiftReminders(ctx context.Context, sql string, args ...any) ([][2]string, error) {
	rows, err := s.pg.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][2]string
	for rows.Next() {
		var d, sh string
		if err := rows.Scan(&d, &sh); err != nil {
			return nil, err
		}
		out = append(out, [2]string{d, sh})
	}
	return out, rows.Err()
}

// notifyShiftReminder **للسائق وحدَه** — والمفتاحُ الدوامُ نفسُه، فتذكيرُ الصبح يحلّ على الجوّال
// محلَّ إشعار الليل.
func (s *Server) notifyShiftReminder(ctx context.Context, driverID, shiftID, title, body string) {
	s.notify.Notify(ctx, notifications.Input{
		UserID: driverID, Kind: notifications.KindAccount,
		Title: title, Body: body,
		Entity: "shift", EntityID: shiftID,
		Apps: []string{notifications.AppDriver},
	})
}

// RunShiftCloseReminder **كلَّ دقيقة** — والختمُ يمنع التكرار.
func (s *Server) RunShiftCloseReminder(ctx context.Context, interval time.Duration) {
	round := func(now time.Time) {
		if _, _, err := s.RemindOpenShifts(ctx, now); err != nil {
			s.logger.Warn("تذكيرُ الدوام المفتوح", "error", err)
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
