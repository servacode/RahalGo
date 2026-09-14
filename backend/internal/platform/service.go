package platform

// ══════════════════════════════════════════════════════════════════════
// **البابُ بين القرار والقاعدة** (`PH`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **والقرارُ في `state.go` خالصٌ لا يعرف قاعدةً** — **وهذه تقرأ له ما
// يحتاج وتكتب ما يضبطه المالك.**
//
// **ولا تُحسب هنا حالٌ ولا يُقارَن وقتٌ** — **وحسبةٌ في موضعين تفترق.**

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// EnforcedKey **رايةُ سريان الجدول** — **وافتراضُها «لا يسري».**
//
// **ولو كان الجدولُ يسري بمجرّد وجود صفوفٍ لَسرى نصفُ جدولٍ يُكتب على
// دفعات** — **ومن أدخل الأحدَ وحدَه أغلق ستّةَ أيّامٍ بلا أن يقصد.**
const EnforcedKey = "hours.platform_enforced"

// ErrClosureRange **مدّةُ إيقافٍ لا تصلح** — موعدُ عودةٍ مضى.
//
// **وإيقافٌ ينتهي قبل أن يبدأ لا يُفعِّل شيئاً** — **فيظنّ المالكُ أنّه
// أوقف الخدمةَ وهي تعمل.**
var ErrClosureRange = httpx.NewError(http.StatusBadRequest,
	"invalid_closure_range", "errors.invalid_closure_range")

// ErrBadSchedule **جدولٌ لا يُقبَل** — يومٌ خارجَ المدى أو فترةٌ فارغةٌ
// أو تداخلٌ.
var ErrBadSchedule = httpx.NewError(http.StatusBadRequest,
	"invalid_hours", "errors.invalid_hours")

// Settings **ما تحتاجه هذه الوحدةُ من الإعدادات** — **ولا أكثر.**
//
// **وواجهةٌ ضيّقةٌ لا خزّانُ الإعدادات كلُّه**: **وحدةٌ تأخذ كلَّ شيءٍ
// تستطيع أن تفعل كلَّ شيء.**
type Settings interface {
	GetBool(ctx context.Context, key string) bool
}

// Service **دوامُ المنصّة وإيقافُها — قراءةً وضبطاً.**
type Service struct {
	db       *pgxpool.Pool
	settings Settings
	// now **لحظةُ الخادم** — **دالّةٌ لا نداءٌ مباشر**: **فتُزاح في
	// الفحص إلى حدّ الدقيقة بلا انتظارِ ساعةٍ حقيقيّة.**
	now func() time.Time
}

// New يبني الخدمة.
func New(db *pgxpool.Pool, st Settings) *Service {
	return &Service{db: db, settings: st, now: time.Now}
}

// SetClock **يُزيح ساعةَ الخدمة — للفحص وحدَه.**
//
// **ولا يناديها منتَج** — **والمنتَجُ يقرأ ساعةَ الخادم.**
func (s *Service) SetClock(f func() time.Time) { s.now = f }

// Now لحظةُ الخادم كما تراها هذه الخدمة.
func (s *Service) Now() time.Time { return s.now() }

// Schedule **يقرأ جدولَ الأسبوع مرتَّباً.**
func (s *Service) Schedule(ctx context.Context, q dbtx.Querier) (Schedule, error) {
	rows, err := q.Query(ctx, `
		SELECT day_of_week,
		       EXTRACT(hour FROM starts_at)::int * 60 + EXTRACT(minute FROM starts_at)::int,
		       EXTRACT(hour FROM ends_at)::int   * 60 + EXTRACT(minute FROM ends_at)::int
		FROM platform_hours
		ORDER BY day_of_week, starts_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := Schedule{}
	for rows.Next() {
		var w Window
		if err := rows.Scan(&w.Day, &w.Start, &w.End); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// Closure **يقرأ حالَ الإيقاف المؤقّت.**
func (s *Service) Closure(ctx context.Context, q dbtx.Querier) (Closure, error) {
	var c Closure
	err := q.QueryRow(ctx, `
		SELECT active, message, ends_at FROM service_closure WHERE id`).
		Scan(&c.Active, &c.Message, &c.EndsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// **ولا صفَّ يعني لا إيقاف** — **وقاعدةٌ لم تُهاجَر بعدُ لا
		// تُوقف الخدمة.**
		return Closure{}, nil
	}
	return c, err
}

// State **حالُ الاستقبال الآن — مقروءةً من القاعدة ومحسوبةً بالقرار.**
//
// **وهي المصدرُ الواحد**: **يقرؤها بابُ الطلب العاديّ، وبابُ المخصَّص،
// والردُّ العامُّ الذي تقرؤه الشاشة.** **وثلاثُ حسباتٍ تفترق يوماً،
// فيقول التطبيقُ «مفتوح» ويردّ المحرّكُ «مغلق».**
func (s *Service) State(ctx context.Context, q dbtx.Querier) (State, error) {
	sch, err := s.Schedule(ctx, q)
	if err != nil {
		return State{}, err
	}
	c, err := s.Closure(ctx, q)
	if err != nil {
		return State{}, err
	}
	return Decide(s.now(), s.settings.GetBool(ctx, EnforcedKey), sch, c), nil
}

// SetSchedule **يستبدل الجدولَ كلَّه في معاملةٍ واحدة.**
//
// **وكلُّه لا بعضُه**: **جدولٌ يُعدَّل صفّاً صفّاً يمرّ بحالاتٍ متداخلةٍ
// وسطَ التعديل** — **ونداءُ زبونٍ يقع في تلك اللحظة يُجاب بجدولٍ نصفِ
// مكتوب.**
func (s *Service) SetSchedule(ctx context.Context, actorID string, ws []Window) (Schedule, error) {
	clean, err := Validate(ws)
	if err != nil {
		return nil, ErrBadSchedule
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM platform_hours`); err != nil {
		return nil, err
	}
	for _, w := range clean {
		if _, err := tx.Exec(ctx, `
			INSERT INTO platform_hours (day_of_week, starts_at, ends_at)
			VALUES ($1, make_time($2, $3, 0), make_time($4, $5, 0))`,
			w.Day, int(w.Start)/60, int(w.Start)%60,
			int(w.End)/60, int(w.End)%60); err != nil {
			return nil, ErrBadSchedule
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return clean, nil
}

// SetClosure **يضبط الإيقافَ المؤقّت.**
//
// **وموعدُ عودةٍ مضى يُردّ** — **وإيقافٌ ينتهي قبل أن يبدأ لا يوقف
// شيئاً**، **فيظنّ المالكُ الخدمةَ متوقّفةً وهي تعمل.**
func (s *Service) SetClosure(ctx context.Context, actorID string, c Closure) (Closure, error) {
	if c.Active && c.EndsAt != nil && !c.EndsAt.After(s.now()) {
		return Closure{}, ErrClosureRange
	}
	var actor any
	if actorID != "" {
		actor = actorID
	}
	_, err := s.db.Exec(ctx, `
		UPDATE service_closure
		   SET active = $1, message = $2, ends_at = $3,
		       updated_at = now(), updated_by = $4::uuid
		 WHERE id`, c.Active, c.Message, c.EndsAt, actor)
	if err != nil {
		return Closure{}, err
	}
	return c, nil
}
