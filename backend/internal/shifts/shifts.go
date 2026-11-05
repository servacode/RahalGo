// Package shifts **دوامُ السائق — بابٌ واحد لفتحه وسكّره** (قرار المالك ٢٠٢٦-١١-٠٥).
//
// كان الدوام عمودين في `users` (`on_shift` · `shift_started_at`) يكتبهما سبعةُ مواضع،
// **بلا سجلّ**: ما في يعرف مين سكّر الدوام ولا ليش. وصار **كل فتح وسكّر صفّاً** في
// `driver_shifts` — والعمودان يُكتبان هون وحده، مع الصفّ في الجملة نفسها.
//
// **وحارسٌ يمنع غيره** (`TestShifts_NoDirectOnShiftWrites`): أيّ `UPDATE users SET … on_shift =`
// خارج هالحزمة يُسقط البناء — فما في مسار ثامن بيسكّر الدوام بلا ما ينكتب.
//
// والحزمة صغيرة عمداً (pgx وdbtx وحدهما) فتناديها `identity` كما يناديها الخادم.
package shifts

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// Reason مين سكّر الدوام.
type Reason string

const (
	// ReasonDriver السائق من المفتاح.
	ReasonDriver Reason = "driver"
	// ReasonDriverSkipped السائق بلا صورة عدّاد (وضع «يطلب ويسمح بالتخطّي»).
	ReasonDriverSkipped Reason = "driver_skipped"
	// ReasonLogout بالخروج من التطبيق.
	ReasonLogout Reason = "logout"
	// ReasonRelease لما ترك طلباً قبل الاستلام.
	ReasonRelease Reason = "release"
	// ReasonEmergency بزرّ الطارئ.
	ReasonEmergency Reason = "emergency"
	// ReasonAccident بعد حادث (ويُقفل حتى «السائق بخير»).
	ReasonAccident Reason = "accident"
	// ReasonOps سكّرته العمليات.
	ReasonOps Reason = "ops"
	// ReasonAccount انوقف الحساب أو انحذف.
	ReasonAccount Reason = "account"
	// ReasonQA شاهد التجربة على التجهيز.
	ReasonQA Reason = "qa"
	// ReasonStale **صفٌّ بقي مفتوحاً والدوامُ مسكَّرٌ بغير هذا الباب** (تنظيفُ التجهيز
	// `scripts/clean-slate.sql`، أو المحرّكُ القديم بنافذة النشر) — يُسكَّر عند الفتح التالي.
	ReasonStale Reason = "stale"
)

// precedence **أيُّ سببٍ يغلب إن سكّر مساران الدوام بنداءٍ واحد** — الأخير أغلب
// (تركُ طلبٍ بسبب حادث: «انسكّر بعد حادث» لا «لما ترك طلب»).
const precedence = `ARRAY['driver','driver_skipped','logout','qa','account','ops','release','emergency','accident']`

// Closed ما سكّره `Close` — ليُرجَع إن لم يقع ما سُكّر لأجله (`Reopen`).
type Closed struct {
	ShiftID   string
	StartedAt *time.Time
	WasOn     bool
}

// Open **يفتح الدوام** — العمودان والصفّ بجملة واحدة. ونداءٌ ثانٍ والدوام مفتوح = الصفّ نفسه.
//
// `build` نسخة التطبيق من ترويسته (صفر = غير معروف)، و`capable` **أعلن التطبيقُ أنّه يعرف
// ورقةَ العدّاد** (`odometer_capable`) — ومنه يُعلَّم «تطبيق قديم» بالمراقبة.
//
// **وصفٌّ بقي مفتوحاً والدوامُ مسكَّر** (`on_shift = false` — تنظيفُ التجهيز، أو المحرّكُ القديم
// بنافذة النشر) **يُسكَّر أوّلاً بسبب `stale`** ثمّ يُفتح صفٌّ جديد — لا يكمل «الدوامُ الجديد»
// من قبل أيّام، ولا تضيع قراءةُ فتحه على صفٍّ قديمٍ له قراءتُه.
func Open(ctx context.Context, q dbtx.Querier, driverID string, build int, capable bool) (string, error) {
	var b any
	if build > 0 {
		b = build
	}
	if _, err := q.Exec(ctx, `
		UPDATE driver_shifts SET closed_at = now(), close_reason = $2
		 WHERE driver_id = $1::uuid AND closed_at IS NULL
		   AND NOT EXISTS (SELECT 1 FROM users WHERE id = $1::uuid AND on_shift)`,
		driverID, string(ReasonStale)); err != nil {
		return "", err
	}
	var id string
	err := q.QueryRow(ctx, `
		WITH u AS (
			UPDATE users SET on_shift = true, shift_started_at = now(), updated_at = now()
			 WHERE id = $1::uuid
			RETURNING id
		), ins AS (
			INSERT INTO driver_shifts (driver_id, opened_build, opened_capable)
			SELECT id, $2::int, $3::boolean FROM u
			ON CONFLICT (driver_id) WHERE closed_at IS NULL DO NOTHING
			RETURNING id
		)
		SELECT id::text FROM ins
		UNION ALL
		SELECT s.id::text FROM driver_shifts s WHERE s.driver_id = $1::uuid AND s.closed_at IS NULL
		LIMIT 1`, driverID, b, capable).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		// **ضغطتان معاً**: صفُّ الأخرى كُتب بعد لقطة هذه الجملة — يُقرأ الآن.
		err = q.QueryRow(ctx, `SELECT id::text FROM driver_shifts
			WHERE driver_id = $1::uuid AND closed_at IS NULL`, driverID).Scan(&id)
	}
	return id, err
}

// Close **يسكّر الدوام بسببه** — العمودان والصفّ المفتوح بجملة واحدة.
//
// **ودوامٌ مفتوح بلا صفّ** (قبل الميزة، أو فكسچر) يُكتب له صفٌّ مسكَّر من `shift_started_at`.
// **وما كان مسكّراً قبل ثوانٍ** بمسارٍ آخر في النداء نفسه يأخذ السببَ الأغلب.
//
// **وصفُّ السائق يُقفل أوّلاً** (`FOR UPDATE`): سكّران بلحظةٍ واحدة (العمليّاتُ والسائق، أو
// الحظرُ والمفتاح) — الثاني ينتظر الأوّل ثمّ يقرأ حالَه بعده (`on_shift = false`)، **فلا
// يكتب صفّاً «قديماً» مكرّراً** من لقطةٍ قرأها قبل أن يُسكَّر الدوام.
func Close(ctx context.Context, q dbtx.Querier, driverID string, reason Reason) (Closed, error) {
	var out Closed
	var id *string
	err := q.QueryRow(ctx, `
		WITH prev AS (
			SELECT on_shift, shift_started_at FROM users WHERE id = $1::uuid FOR UPDATE
		), u AS (
			-- والقفلُ قبل الكتابة: EXISTS (prev) يُقرأ قبل أن يُكتب الصفّ — ولو كُتب أوّلاً
			-- لم يرجع prev (صفٌّ عدّله الأمرُ نفسُه يُتخطّى) فضاع «كان بالدوام» على Reopen.
			UPDATE users SET on_shift = false, shift_started_at = NULL, updated_at = now()
			 WHERE id = $1::uuid AND (on_shift OR shift_started_at IS NOT NULL)
			   AND EXISTS (SELECT 1 FROM prev)
			RETURNING id
		), c AS (
			UPDATE driver_shifts SET closed_at = now(), close_reason = $2
			 WHERE driver_id = $1::uuid AND closed_at IS NULL
			RETURNING id
		), legacy AS (
			INSERT INTO driver_shifts (driver_id, opened_at, closed_at, close_reason)
			SELECT $1::uuid, COALESCE(p.shift_started_at, now()), now(), $2
			  FROM prev p
			 WHERE p.on_shift AND NOT EXISTS (SELECT 1 FROM c)
			RETURNING id
		)
		SELECT (SELECT id::text FROM c UNION ALL SELECT id::text FROM legacy LIMIT 1),
		       COALESCE((SELECT on_shift FROM prev), false),
		       (SELECT shift_started_at FROM prev)`, driverID, string(reason)).Scan(&id, &out.WasOn, &out.StartedAt)
	if err != nil {
		return out, err
	}
	if id != nil {
		out.ShiftID = *id
		return out, nil
	}
	// **ولا دوامَ مفتوح** — سكّره مسارٌ آخر قبل لحظة؟ السببُ الأغلب يبقى.
	_, err = q.Exec(ctx, `
		UPDATE driver_shifts SET close_reason = $2
		 WHERE id = (SELECT id FROM driver_shifts
		              WHERE driver_id = $1::uuid AND closed_at > now() - interval '1 minute'
		              ORDER BY closed_at DESC LIMIT 1)
		   AND array_position(`+precedence+`, close_reason) < array_position(`+precedence+`, $2::text)`,
		driverID, string(reason))
	return out, err
}

// Reopen **يُرجع ما سكّره `Close`** — لتركٍ لم يقع (فشل الانتقال): الدوام يعود كما كان
// ببدايته نفسها، والصفّ نفسه يُفتح من جديد.
func Reopen(ctx context.Context, q dbtx.Querier, driverID string, c Closed) error {
	if c.ShiftID != "" {
		if _, err := q.Exec(ctx, `
			UPDATE driver_shifts SET closed_at = NULL, close_reason = NULL
			 WHERE id = $1::uuid
			   AND NOT EXISTS (SELECT 1 FROM driver_shifts o
			                    WHERE o.driver_id = $2::uuid AND o.closed_at IS NULL)`, c.ShiftID, driverID); err != nil {
			return err
		}
	}
	if !c.WasOn {
		return nil
	}
	_, err := q.Exec(ctx, `
		UPDATE users SET on_shift = true, shift_started_at = COALESCE($2, shift_started_at, now()), updated_at = now()
		 WHERE id = $1::uuid`, driverID, c.StartedAt)
	return err
}

// Current **الدوام المفتوح الآن** — معرّفه وبدايته، أو فارغ.
func Current(ctx context.Context, q dbtx.Querier, driverID string) (string, time.Time, error) {
	var id string
	var at time.Time
	err := q.QueryRow(ctx, `
		SELECT id::text, opened_at FROM driver_shifts
		 WHERE driver_id = $1::uuid AND closed_at IS NULL`, driverID).Scan(&id, &at)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, nil
	}
	return id, at, err
}
