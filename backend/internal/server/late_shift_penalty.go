package server

// ══════════════════════════════════════════════════════════════════════
// **عقوبةُ تأخير فتح الدوام** — قرار المالك ٢٠٢٦-١١-١٤
// ══════════════════════════════════════════════════════════════════════
//
// كلُّ سائقٍ فعّال (حسابٌ `active` غيرُ محذوفٍ بدور سائق) **لازم يفتح دوامه خلال دقائق السماح**
// (`drivers.late_shift_grace_min`، افتراضُه ٥) **بعد فتح المنصّة** — أوّلُ فترةٍ من جدول
// الدوام الأسبوعيّ بيوم المنصّة (`platform.day_start_hour`).
//
//	١ · **قبل الموعد بدقيقتين** — تذكيرٌ واحد «باقي دقيقتين، افتح دوامك» لمن لم يفتح.
//	٢ · **بعد الموعد** (أوّلُ جولةٍ والمنصّةُ مفتوحة) — من لم يكن بدوامه بين فتح المنصّة والموعد:
//	    يُخصم من محفظته `N_wallet × delivery.fee` (عقوبةٌ بالحوافز: الخزينةُ تقبض وما زاد على
//	    متاحه يصير ديناً) ومن عدّاد هدفه الشهريّ `N_target` توصيلات (`driver_target_adjustments`)،
//	    وإشعارٌ بالسبب والمقدار.
//
// **ولا عقوبة**: لمن فتح بالموعد ثمّ سكّر · لمن عليه «إجازة» لهذا اليوم · والمفتاحُ مطفأ ·
// والمنصّةُ مسكّرةٌ الآن أو بلا جدولٍ يسري (زرُّ «مفتوحين» أو «مسكّرين» أو جدولٌ لا يسري) ·
// ولحسابٍ أُنشئ بعد فتح المنصّة · **وإن شُغّل المفتاحُ بعد موعد اليوم** (يسري من الغد —
// لا عقوبةَ بأثرٍ رجعيّ).
//
// **ومرّةً لكلّ سائقٍ ويوم**: صفُّ العقوبة (`driver_late_shift_penalties`، فريدٌ بالسائق واليوم)
// يُكتب أوّلاً في معاملة الخصم نفسِها، فجولتان أو خادمان لا يكرّران.
//
// **والإلغاءُ من الإدارة** (`POST /admin/late-shift-penalties/{id}/cancel`): يُسقط باقي الدين،
// وتردّ الخزينةُ ما قبضت (`incentives.VoidPenaltyTx`)، ويُمحى خصمُ الهدف. **و«إجازة» تُضغط بعد
// العقوبة لا تلغيها** — الإلغاءُ فعلٌ صريحٌ وحدَه.

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/incentives"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

// مفاتيحُ الإعدادات — `settings/catalog.go`.
const (
	lateShiftOnKey     = "drivers.late_shift_penalty_on"
	lateShiftGraceKey  = "drivers.late_shift_grace_min"
	lateShiftWalletKey = "drivers.late_shift_wallet_deliveries"
	lateShiftTargetKey = "drivers.late_shift_target_deliveries"
)

// lateShiftRemindBefore **التذكيرُ قبل الموعد بدقيقتين** — «باقي دقيقتين».
const lateShiftRemindBefore = 2 * time.Minute

// lateShiftBatch سقفُ الجولة — والباقي بالجولة التالية.
const lateShiftBatch = 500

// lateShiftMoment **فتحُ المنصّة اليوم وموعدُ فتح الدوام** — وفارغٌ إن لم يكن لليوم فتح.
type lateShiftMoment struct {
	Day      string    // يومُ المنصّة «2026-11-14»
	Opening  time.Time // أوّلُ بدءِ فترةٍ بيوم المنصّة
	Deadline time.Time // الفتحُ + السماح
	Grace    int64
}

// lateShiftNow **أين نحن من موعد اليوم؟** — `nil` إن كانت العقوبةُ مطفأةً أو المنصّةُ مسكّرةً الآن
// أو بلا جدولٍ يسري أو لم تفتح اليومَ بعد.
func (s *Server) lateShiftNow(ctx context.Context, now time.Time) (*lateShiftMoment, error) {
	if s.platform == nil || !s.settings.GetBool(ctx, lateShiftOnKey) {
		return nil, nil
	}
	enforced, sch, err := s.platform.Gate(ctx, s.pg)
	if err != nil || !enforced || len(sch) == 0 || !sch.OpenAt(now) {
		return nil, err
	}
	dsh := s.dayStartHour(ctx)
	day := siteDay(now, dsh)
	d, err := time.ParseInLocation("2006-01-02", day, platform.Location())
	if err != nil {
		return nil, err
	}
	from := d.Add(time.Duration(dsh) * time.Hour)
	opening := sch.FirstStartIn(from, from.Add(24*time.Hour))
	if opening == nil || now.Before(*opening) {
		return nil, nil
	}
	grace := s.settings.GetInt(ctx, lateShiftGraceKey)
	if grace <= 0 {
		grace = 5
	}
	m := &lateShiftMoment{Day: day, Opening: *opening, Grace: grace,
		Deadline: opening.Add(time.Duration(grace) * time.Minute)}
	// **ولا أثرَ رجعيّ**: مفتاحٌ شُغّل بعد موعد اليوم يسري من الغد.
	var since time.Time
	if err := s.pg.QueryRow(ctx, `SELECT updated_at FROM app_settings WHERE key = $1`,
		lateShiftOnKey).Scan(&since); err == nil && since.After(m.Deadline) {
		return nil, nil
	}
	return m, nil
}

// lateShiftExpectedSQL **السائقُ المطلوبُ منه الفتح** — فعّالٌ بدور سائق، أُنشئ قبل فتح المنصّة،
// وليس عليه إجازةٌ لليوم. `$1` اليوم و`$2` لحظةُ الفتح.
const lateShiftExpectedSQL = `
	u.status = 'active' AND u.deleted_at IS NULL AND u.created_at < $2
	AND EXISTS (SELECT 1 FROM user_roles r WHERE r.user_id = u.id AND r.role_code = 'driver')
	AND NOT EXISTS (SELECT 1 FROM driver_leaves l WHERE l.driver_id = u.id AND l.day = $1::date)`

// RunLateShiftRound **جولةُ تأخير فتح الدوام** — تُنادى دوريّاً وفي الاختبار مباشرةً، وتردّ كم ذُكِّر
// وكم عوقب.
func (s *Server) RunLateShiftRound(ctx context.Context, now time.Time) (reminded, penalized int, err error) {
	m, err := s.lateShiftNow(ctx, now)
	if err != nil || m == nil {
		return 0, 0, err
	}
	if now.Before(m.Deadline) {
		if now.Before(m.Deadline.Add(-lateShiftRemindBefore)) {
			return 0, 0, nil
		}
		n, err := s.remindLateShift(ctx, m, now)
		return n, 0, err
	}
	n, err := s.penalizeLateShift(ctx, m)
	return 0, n, err
}

// remindLateShift **«باقي دقيقتين، افتح دوامك»** — لمن ليس بدوامه الآن، مرّةً باليوم (ختمُ
// `driver_shift_reminders` بنوع `open_soon`).
func (s *Server) remindLateShift(ctx context.Context, m *lateShiftMoment, now time.Time) (int, error) {
	rows, err := s.pg.Query(ctx, `
		INSERT INTO driver_shift_reminders (driver_id, day, kind, sent_at)
		SELECT u.id, $1::date, 'open_soon', $3
		  FROM users u
		 WHERE `+lateShiftExpectedSQL+`
		   AND NOT EXISTS (SELECT 1 FROM driver_shifts sh
		                    WHERE sh.driver_id = u.id AND sh.opened_at <= $3
		                      AND (sh.closed_at IS NULL OR sh.closed_at >= $2))
		   AND NOT EXISTS (SELECT 1 FROM driver_shift_reminders q
		                    WHERE q.driver_id = u.id AND q.day = $1::date AND q.kind = 'open_soon')
		 LIMIT `+strconv.Itoa(lateShiftBatch)+`
		ON CONFLICT DO NOTHING
		RETURNING driver_id::text`, m.Day, m.Opening, now)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	body := lateShiftFill(notifTitles.lateShiftRemindBody, m, 0, 0)
	for _, id := range ids {
		s.notify.Notify(ctx, notifications.Input{
			UserID: id, Kind: notifications.KindAccount,
			Title: notifTitles.lateShiftRemind, Body: body,
			Entity: "shift", EntityID: id,
			Apps: []string{notifications.AppDriver},
		})
	}
	return len(ids), nil
}

// penalizeLateShift **العقوبةُ لمن لم يكن بدوامه بين فتح المنصّة والموعد** — كلٌّ بمعاملته.
func (s *Server) penalizeLateShift(ctx context.Context, m *lateShiftMoment) (int, error) {
	walletN := max(s.settings.GetInt(ctx, lateShiftWalletKey), 0)
	targetN := max(s.settings.GetInt(ctx, lateShiftTargetKey), 0)
	fee := max(s.settings.GetInt(ctx, "delivery.fee"), 0)
	amount := walletN * fee
	if amount == 0 && targetN == 0 {
		// **لا خصمَ مضبوطاً** — لا شيءَ يُكتب ولا يُرسل.
		return 0, nil
	}
	if amount == 0 {
		walletN = 0
	}
	actor := s.systemActorID(ctx)
	if actor == "" {
		s.logger.Warn("عقوبةُ تأخير الدوام: لا حسابَ نظام — لا عقوبة")
		return 0, nil
	}
	rows, err := s.pg.Query(ctx, `
		SELECT u.id::text FROM users u
		 WHERE `+lateShiftExpectedSQL+`
		   AND NOT EXISTS (SELECT 1 FROM driver_late_shift_penalties p
		                    WHERE p.driver_id = u.id AND p.day = $1::date)
		   AND NOT EXISTS (SELECT 1 FROM driver_shifts sh
		                    WHERE sh.driver_id = u.id AND sh.opened_at < $3
		                      AND (sh.closed_at IS NULL OR sh.closed_at >= $2))
		 ORDER BY u.id
		 LIMIT `+strconv.Itoa(lateShiftBatch), m.Day, m.Opening, m.Deadline)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	reason := "تأخير فتح الدوام — يوم " + m.Day + " (فتحت المنصة " +
		m.Opening.In(platform.Location()).Format("15:04") + ")"
	done := 0
	for _, id := range ids {
		ok, err := s.penalizeOne(ctx, actor, id, m, walletN, targetN, fee, amount, reason)
		if err != nil {
			s.logger.Warn("عقوبةُ تأخير الدوام تعذّرت", "driver", id, "error", err)
			continue
		}
		if !ok {
			continue
		}
		done++
		body := lateShiftFill(notifTitles.lateShiftPenaltyBody, m, 0, 0)
		if amount > 0 {
			body += lateShiftFill(notifTitles.lateShiftWalletPart, m, amount, 0)
		}
		if targetN > 0 {
			body += lateShiftFill(notifTitles.lateShiftTargetPart, m, 0, targetN)
		}
		s.notify.Notify(ctx, notifications.Input{
			UserID: id, Kind: notifications.KindWallet,
			Title: notifTitles.lateShiftPenalty, Body: body,
			Entity: "wallet", Href: "/portal/wallet",
			Apps: []string{notifications.AppDriver},
		})
	}
	if done > 0 {
		s.touch("driver", "ops")
	}
	return done, nil
}

// penalizeOne **الصفُّ أوّلاً ثمّ الخصم — بمعاملةٍ واحدة.** `false` إن عوقب لهذا اليوم سابقاً.
func (s *Server) penalizeOne(ctx context.Context, actor, driverID string, m *lateShiftMoment,
	walletN, targetN, fee, amount int64, reason string) (bool, error) {
	ok := false
	err := s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
		var pid string
		err := q.QueryRow(ctx, `
			INSERT INTO driver_late_shift_penalties
			       (driver_id, day, opening_at, grace_min, wallet_deliveries, target_deliveries,
			        unit_fee, amount)
			VALUES ($1::uuid, $2::date, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (driver_id, day) DO NOTHING
			RETURNING id::text`,
			driverID, m.Day, m.Opening, m.Grace, walletN, targetN, fee, amount).Scan(&pid)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if amount > 0 {
			e, err := s.incentives.GrantRoleTx(ctx, q, actor, driverID, incentives.KindPenalty,
				amount, reason, "driver")
			if err != nil {
				return err
			}
			if _, err := q.Exec(ctx, `
				UPDATE driver_late_shift_penalties SET incentive_id = $2::uuid WHERE id = $1::uuid`,
				pid, e.ID); err != nil {
				return err
			}
		}
		if targetN > 0 {
			if _, err := q.Exec(ctx, `
				INSERT INTO driver_target_adjustments (driver_id, month, day, deliveries, reason, penalty_id)
				VALUES ($1::uuid, $2, $3::date, $4, $5, $6::uuid)`,
				driverID, m.Day[:7], m.Day, targetN, reason, pid); err != nil {
				return err
			}
		}
		ok = true
		return nil
	})
	return ok, err
}

// lateShiftFill **يملأ ما بين القوسين** — `{time}` `{grace}` `{amount}` `{n}`.
func lateShiftFill(text string, m *lateShiftMoment, amount, n int64) string {
	return strings.NewReplacer(
		"{time}", m.Opening.In(platform.Location()).Format("15:04"),
		"{grace}", strconv.FormatInt(m.Grace, 10),
		"{amount}", obligations.Money(amount),
		"{n}", strconv.FormatInt(n, 10),
		"{day}", m.Day,
	).Replace(text)
}

// RunLateShiftPenalty **كلَّ دقيقة** — والصفُّ الفريدُ والختمُ يمنعان التكرار.
func (s *Server) RunLateShiftPenalty(ctx context.Context, interval time.Duration) {
	round := func(now time.Time) {
		if _, _, err := s.RunLateShiftRound(ctx, now); err != nil {
			s.logger.Warn("عقوبةُ تأخير فتح الدوام", "error", err)
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

// ══════════════════════════════════════════════════════════════════════
// **الإجازة والعقوبات على صفحة السائق** — للوحة
// ══════════════════════════════════════════════════════════════════════

// lateShiftPenaltyRow **عقوبةٌ كما تُعرض.**
type lateShiftPenaltyRow struct {
	ID               string     `json:"id"`
	Day              string     `json:"day"`
	OpeningAt        time.Time  `json:"opening_at"`
	GraceMin         int        `json:"grace_min"`
	WalletDeliveries int        `json:"wallet_deliveries"`
	TargetDeliveries int        `json:"target_deliveries"`
	Amount           int64      `json:"amount"`
	CreatedAt        time.Time  `json:"created_at"`
	CancelledAt      *time.Time `json:"cancelled_at"`
	CancelReason     string     `json:"cancel_reason"`
}

// errLateShiftNotDriver **ليس سائقاً** — لا إجازةَ لغير سائق.
var errLateShiftNotDriver = httpx.ErrNotFound

// lateShiftDriver **يتحقّق أنّ المعرّفَ سائق.**
func (s *Server) lateShiftDriver(ctx context.Context, id string) error {
	if !isUUID(id) {
		return errLateShiftNotDriver
	}
	var ok bool
	if err := s.pg.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1::uuid AND role_code = 'driver')`,
		id).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return errLateShiftNotDriver
	}
	return nil
}

// handleAdminDriverLateShift `GET /admin/drivers/{id}/late-shift` — **أعليه إجازةٌ اليوم، وآخرُ عقوباته.**
func (s *Server) handleAdminDriverLateShift(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()
	if err := s.lateShiftDriver(ctx, id); err != nil {
		s.respondErr(w, err)
		return
	}
	day := siteDay(time.Now(), s.dayStartHour(ctx))
	var onLeave bool
	if err := s.pg.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM driver_leaves WHERE driver_id = $1::uuid AND day = $2::date)`,
		id, day).Scan(&onLeave); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(ctx, `
		SELECT id::text, day::text, opening_at, grace_min, wallet_deliveries, target_deliveries,
		       amount, created_at, cancelled_at, cancel_reason
		  FROM driver_late_shift_penalties
		 WHERE driver_id = $1::uuid
		 ORDER BY day DESC
		 LIMIT 30`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	list := []lateShiftPenaltyRow{}
	for rows.Next() {
		var x lateShiftPenaltyRow
		if err := rows.Scan(&x.ID, &x.Day, &x.OpeningAt, &x.GraceMin, &x.WalletDeliveries,
			&x.TargetDeliveries, &x.Amount, &x.CreatedAt, &x.CancelledAt, &x.CancelReason); err != nil {
			s.respondErr(w, err)
			return
		}
		list = append(list, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"day":       day,
		"on_leave":  onLeave,
		"enabled":   s.settings.GetBool(ctx, lateShiftOnKey),
		"penalties": list,
	})
}

// handleAdminDriverLeave `POST|DELETE /admin/drivers/{id}/leave` — **إجازةُ اليوم (يومُ المنصّة) أو
// التراجعُ عنها.** ولا تلغي عقوبةً وقعت — الإلغاءُ فعلُه وحدَه.
func (s *Server) handleAdminDriverLeave(on bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		ctx := r.Context()
		if err := s.lateShiftDriver(ctx, id); err != nil {
			s.respondErr(w, err)
			return
		}
		day := siteDay(time.Now(), s.dayStartHour(ctx))
		err := s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
			if on {
				if _, err := q.Exec(ctx, `
					INSERT INTO driver_leaves (driver_id, day, created_by)
					VALUES ($1::uuid, $2::date, NULLIF($3, '')::uuid)
					ON CONFLICT (driver_id, day) DO NOTHING`, id, day, userIDFrom(r)); err != nil {
					return err
				}
				return s.auditTx(ctx, q, r, "ops.driver_leave", "user", id, map[string]any{"day": day})
			}
			if _, err := q.Exec(ctx, `
				DELETE FROM driver_leaves WHERE driver_id = $1::uuid AND day = $2::date`, id, day); err != nil {
				return err
			}
			return s.auditTx(ctx, q, r, "ops.driver_leave_removed", "user", id, map[string]any{"day": day})
		})
		if err != nil {
			s.respondErr(w, err)
			return
		}
		s.touch("driver", "ops")
		httpx.JSON(w, http.StatusOK, map[string]any{"day": day, "on_leave": on})
	}
}

// errLateShiftCancelled **أُلغيت قبلُ أو لا وجودَ لها.**
var errLateShiftCancelled = httpx.NewError(http.StatusConflict, "late_shift_penalty_cancelled", "errors.conflict")

// handleCancelLateShiftPenalty `POST /admin/late-shift-penalties/{id}/cancel` — **إلغاءُ العقوبة**:
// يُسقط باقي دينها وتردّ الخزينةُ ما قبضت ويُمحى خصمُ الهدف — بمعاملةٍ واحدة.
func (s *Server) handleCancelLateShiftPenalty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, _ := decode[struct {
		Reason string `json:"reason"`
	}](r)
	reason := ""
	if req != nil {
		reason = clip(strings.TrimSpace(req.Reason), 300)
	}
	ctx := r.Context()
	var driverID, day string
	var refund int64
	err := s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
		var err error
		driverID, day, refund, err = s.CancelLateShiftPenaltyTx(ctx, q, id, userIDFrom(r), reason)
		if err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "finance.late_shift_penalty_cancelled", "user", driverID,
			map[string]any{"penalty": id, "day": day, "refund": refund, "reason": reason})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: driverID, Kind: notifications.KindWallet,
		Title:  notifTitles.lateShiftCancelled,
		Body:   strings.ReplaceAll(notifTitles.lateShiftCancelledBody, "{day}", day),
		Entity: "wallet", Href: "/portal/wallet",
		Apps: []string{notifications.AppDriver},
	})
	s.touch("driver", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"cancelled": true, "refund": refund})
}

// CancelLateShiftPenaltyTx **يلغي عقوبةَ تأخيرٍ في المعاملة المُمرَّرة** — ويُرجع السائقَ واليومَ وما رُدّ.
func (s *Server) CancelLateShiftPenaltyTx(ctx context.Context, q dbtx.Querier,
	penaltyID, actorID, reason string) (driverID, day string, refund int64, err error) {
	var incID *string
	err = q.QueryRow(ctx, `
		SELECT driver_id::text, day::text, incentive_id::text FROM driver_late_shift_penalties
		 WHERE id = $1::uuid AND cancelled_at IS NULL
		 FOR UPDATE`, penaltyID).Scan(&driverID, &day, &incID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", 0, errLateShiftCancelled
	}
	if err != nil {
		return "", "", 0, err
	}
	if incID != nil && *incID != "" {
		note := "إلغاء عقوبة تأخير الدوام — يوم " + day
		if refund, err = s.incentives.VoidPenaltyTx(ctx, q, actorID, *incID, note); err != nil {
			return "", "", 0, err
		}
	}
	if _, err = q.Exec(ctx, `DELETE FROM driver_target_adjustments WHERE penalty_id = $1::uuid`,
		penaltyID); err != nil {
		return "", "", 0, err
	}
	if _, err = q.Exec(ctx, `
		UPDATE driver_late_shift_penalties
		   SET cancelled_at = now(), cancelled_by = NULLIF($2, '')::uuid, cancel_reason = $3
		 WHERE id = $1::uuid`, penaltyID, actorID, reason); err != nil {
		return "", "", 0, err
	}
	return driverID, day, refund, nil
}
