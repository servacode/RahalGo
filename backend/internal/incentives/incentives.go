package incentives

// الأهدافُ والمكافآتُ والعقوبات — **ما لا معادلةَ له.**
//
// # القسمة
//
//	الأجرُ والعمولة  ←  معادلةٌ تقع وحدَها (`orders`)
//	المكافأةُ والعقوبة ←  تقديرُ إنسانٍ يُقيَّد بكلمة
//
// **والهدفُ بينهما**: يُحسب آلياً **ومكافأتُه تنصرف لحالها** عند بلوغ كلّ
// مرحلة (قرارُ المالك ٢٠٢٦-٠٨-٠٩ ثمّ ٢٠٢٦-٠٨-٣١: ثلاثُ مراحل) — `target.go`.
// **والمكافأةُ والعقوبةُ اليدويّة طلبٌ يوافق عليه شخصٌ ثانٍ** (قرارُ المالك
// ٢٠٢٦-١٠-٠٤) — `requests.go`، **وهي «تقدير» دائماً لا «عن الهدف».**
//
// # والشهرُ يبدأ بتوقيت دمشق
//
// **لا بغرينتش**: شهرُ السائق ينتهي عنده لا في لندن، **وطلبٌ سُلّم الساعةَ
// الواحدةَ بعد منتصف الليل يُحسب على ليلته لا على شهرٍ جديد.**

import (
	"context"
	"errors"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var (
	// ErrBadKind لا ثالثَ لهما.
	ErrBadKind = httpx.NewError(http.StatusBadRequest, "bad_incentive_kind", "errors.validation")
	// ErrNeedsReason **مالٌ يخرج بلا كلمةٍ لا يُراجَع** — ولا يُقاس من يُكثره.
	ErrNeedsReason = httpx.NewError(http.StatusBadRequest, "incentive_needs_reason", "errors.validation")
	// ErrBadAmount والمبلغُ موجبٌ في الحالين — **الإشارةُ في النوع لا في الرقم.**
	ErrBadAmount = httpx.NewError(http.StatusBadRequest, "bad_incentive_amount", "errors.validation")
	// ErrNoBalance عقوبةٌ لا تحتملها محفظتُه.
	//
	// **ومحفظةٌ لا تنزل تحت الصفر** (قيدُ القاعدة) — فتُردّ العقوبةُ صراحةً
	// **لا تُقبل نصفَها بصمت**: من عوقب بعشرةٍ وخُصمت منه أربعةٌ يظنّ أنّه
	// عوقب بأربعة.
	ErrNoBalance = httpx.NewError(http.StatusConflict, "insufficient_balance", "errors.insufficient_balance")
)

// أنواعُ الحافز.
const (
	KindReward  = "reward"
	KindPenalty = "penalty"
)

// Service يقرأ الأهدافَ ويقيّد المكافآت.
type Service struct {
	db       *pgxpool.Pool
	wallet   *wallet.Service
	settings Settings
	treasury func(ctx context.Context) string
	// logger **لِما يقع في الخلفية** — مكافأةُ الهدف تُدفع بلا فاعلٍ بشريّ،
	// **وإخفاقُها لا يُسقط تسليماً** فلا يبقى له أثرٌ إلّا هنا.
	logger *slog.Logger
}

// SetLogger يُحقن مرّةً عند الإقلاع — **وبلاه تصمت الحزمة ولا تنهار.**
func (s *Service) SetLogger(l *slog.Logger) { s.logger = l }

// Settings ما يلزم من الإعدادات — **واجهةٌ ضيّقة**: هذه الحزمةُ تقرأ رقمين.
type Settings interface {
	GetInt(ctx context.Context, key string) int64
}

func New(db *pgxpool.Pool, w *wallet.Service, st Settings,
	treasury func(ctx context.Context) string) *Service {
	return &Service{db: db, wallet: w, settings: st, treasury: treasury}
}

// Standing حالُ شخصٍ في شهرٍ — هدفُه وما بلغ وما ناله.
type Standing struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	// Done ما أنجزه في الشهر — طلباتٌ سُلّمت، أو متاجرُ فتحها.
	Done int `json:"done"`
	// Target مرحلتُه القادمة — **وصفرٌ يعني لا هدف**، فلا شارةَ ولا شريط.
	Target int `json:"target"`
	// Reached بلغ كلَّ المراحل — **يقوله الخادمُ ولا يُستنتج في الشاشة.**
	Reached bool `json:"reached"`
	// Level **كم مرحلةً بلغ** و Levels كم مرحلةً مضبوطة — «وصل للمرحلة ٢ من ٣».
	//
	// **كانت الشارةُ لا تظهر إلّا لمن أتمّ الكلّ** — فمن بلغ الأولى وقبضها
	// يظهر كمن لم يبلغ شيئاً.
	Level  int `json:"level"`
	Levels int `json:"levels"`
	// Rewarded كلُّ ما نال في الشهر، و Penalized ما خُصم منه.
	Rewarded  int64 `json:"rewarded"`
	Penalized int64 `json:"penalized"`
	// AutoPaid مكافآتُ الهدف الآليّة، و ManualPaid اليدويّةُ («تقدير»).
	AutoPaid   int64 `json:"auto_paid"`
	ManualPaid int64 `json:"manual_paid"`
	// Balance **المتاحُ في محفظته** (الرصيدُ ناقصَ المحجوز) — يُرى قبل العقوبة.
	Balance int64 `json:"balance"`
}

// Summary **كروتُ رأس الصفحة**: كم بلغ كلَّ مرحلة · الآليّ · اليدويّ · العقوبات.
type Summary struct {
	ReachedPerLevel []int `json:"reached_per_level"`
	AutoPaid        int64 `json:"auto_paid"`
	ManualPaid      int64 `json:"manual_paid"`
	Penalties       int64 `json:"penalties"`
}

// ErrBadMonth **شهرٌ بغير صيغة «2026-09»** أو في المستقبل.
var ErrBadMonth = httpx.NewError(http.StatusBadRequest, "bad_month", "errors.validation")

// NormalizeMonth **يقبل «2026-09» أو فراغاً (الجاري)** — ولا مستقبل.
func NormalizeMonth(m string) (string, error) {
	cur := PeriodOf(time.Now())
	if m == "" {
		return cur, nil
	}
	if _, err := time.Parse("2006-01", m); err != nil || len(m) != 7 || m > cur {
		return "", ErrBadMonth
	}
	return m, nil
}

// Standings حالُ كلّ من في هذا الدور للشهر الجاري.
func (s *Service) Standings(ctx context.Context, role string) ([]Standing, error) {
	return s.StandingsFor(ctx, role, PeriodOf(time.Now()))
}

// StandingsFor حالُ كلّ من في هذا الدور **في شهرٍ بعينه** (قرارُ المالك
// ٢٠٢٦-١٠-٠٤: اختيارُ شهرٍ وتصدير).
//
// # ولماذا استعلامٌ واحد
//
// نداءٌ لكلّ سائقٍ يعني عشرين نداءً لشاشةٍ واحدة. **وشاشةٌ بطيئةٌ لا تُفتح.**
//
// # والموقوفُ لا يظهر
//
// **(قرارُ المالك ٢٠٢٦-١٠-٠٤)**: الاستعلامُ كان يستثني المحذوفَ وحدَه، فيبقى
// الموقوفُ في القائمة وزرُّ المكافأة أمامه. **والآن الفعّالُ وحدَه.**
func (s *Service) StandingsFor(ctx context.Context, role, month string) ([]Standing, error) {
	levels := s.levelsFor(ctx, role)
	if len(levels) == 0 {
		return nil, httpx.ErrNotFound
	}
	// **والهدفُ المعروضُ مرحلتُه القادمة** (قرارُ المالك ٢٠٢٦-٠٨-٣١) — ومن
	// بلغ الكلَّ عُرضت الأخيرةُ مبلوغة.
	set := []int{}
	for _, l := range levels {
		if l.Target > 0 {
			set = append(set, int(l.Target))
		}
	}
	place := func(done int) (next, level int) {
		for _, t := range set {
			next = t
			if done < t {
				return t, level
			}
			level++
		}
		return next, level
	}

	inMonth := `i.created_at >= ` + monthFromSQL("$2::text") +
		` AND i.created_at < ` + monthToSQL("$2::text")
	rows, err := s.db.Query(ctx, `
		SELECT u.id::text, COALESCE(NULLIF(u.full_name, ''), ''), u.phone::text,
		       `+doneSQL(role, "u.id", "$2::text")+`,
		       COALESCE((SELECT sum(i.amount) FROM incentives i
		                 WHERE i.user_id = u.id AND i.kind = 'reward' AND i.for_target
		                   AND `+inMonth+`), 0),
		       COALESCE((SELECT sum(i.amount) FROM incentives i
		                 WHERE i.user_id = u.id AND i.kind = 'reward' AND NOT i.for_target
		                   AND `+inMonth+`), 0),
		       COALESCE((SELECT sum(i.amount) FROM incentives i
		                 WHERE i.user_id = u.id AND i.kind = 'penalty'
		                   AND `+inMonth+`), 0),
		       COALESCE((SELECT w.balance - w.reserved FROM wallets w WHERE w.user_id = u.id), 0)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = $1
		WHERE u.deleted_at IS NULL AND u.status = 'active'
		ORDER BY 4 DESC, u.full_name`, role, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Standing{}
	for rows.Next() {
		var x Standing
		if err := rows.Scan(&x.UserID, &x.Name, &x.Phone, &x.Done,
			&x.AutoPaid, &x.ManualPaid, &x.Penalized, &x.Balance); err != nil {
			return nil, err
		}
		x.Rewarded = x.AutoPaid + x.ManualPaid
		x.Target, x.Level = place(x.Done)
		x.Levels = len(set)
		// **وبلوغٌ بلا هدفٍ ليس بلوغاً** — صفرٌ يعني «لا هدف».
		x.Reached = x.Target > 0 && x.Done >= x.Target
		out = append(out, x)
	}
	return out, rows.Err()
}

// SummaryOf **كروتُ الشهر** — والمالُ من كلّ صاحب دور، موقوفاً كان أو فعّالاً:
// ما صُرف صُرف.
func (s *Service) SummaryOf(ctx context.Context, role, month string, rows []Standing) (Summary, error) {
	sum := Summary{ReachedPerLevel: make([]int, len(s.LevelsOf(ctx, role)))}
	for _, r := range rows {
		for n := 0; n < r.Level && n < len(sum.ReachedPerLevel); n++ {
			sum.ReachedPerLevel[n]++
		}
	}
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(sum(i.amount) FILTER (WHERE i.kind = 'reward' AND i.for_target), 0),
		       COALESCE(sum(i.amount) FILTER (WHERE i.kind = 'reward' AND NOT i.for_target), 0),
		       COALESCE(sum(i.amount) FILTER (WHERE i.kind = 'penalty'), 0)
		  FROM incentives i
		 WHERE EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = i.user_id AND ur.role_code = $1)
		   AND i.created_at >= `+monthFromSQL("$2::text")+`
		   AND i.created_at < `+monthToSQL("$2::text"), role, month).
		Scan(&sum.AutoPaid, &sum.ManualPaid, &sum.Penalties)
	return sum, err
}

// MyStanding حالُ صاحب الحساب نفسِه — للشهر الجاري.
func (s *Service) MyStanding(ctx context.Context, userID, role string) (*Standing, error) {
	all, err := s.Standings(ctx, role)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].UserID == userID {
			return &all[i], nil
		}
	}
	return &Standing{UserID: userID}, nil
}

// Entry سطرٌ في كشف مكافآته.
type Entry struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Amount    int64  `json:"amount"`
	Reason    string `json:"reason"`
	ForTarget bool   `json:"for_target"`
	By        string `json:"by"`
	CreatedAt string `json:"created_at"`
}

// List كشفُ مكافآت شخصٍ وعقوباته — الأحدثُ أوّلاً.
func (s *Service) List(ctx context.Context, userID string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT i.id::text, i.kind, i.amount, i.reason, i.for_target,
		       COALESCE(NULLIF(b.full_name, ''), ''), i.created_at::text
		FROM incentives i
		LEFT JOIN users b ON b.id = i.created_by
		WHERE i.user_id = $1
		ORDER BY i.created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Kind, &e.Amount, &e.Reason, &e.ForTarget,
			&e.By, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Grant يقيّد مكافأةً أو عقوبةً — **في معاملةٍ واحدة، والخزينةُ الطرفُ المقابل.**
//
// **ودفترٌ يأخذ من طرفٍ ولا يعطي آخرَ لا يتوازن.** ولا يُنادى من الصفحة:
// اليدويُّ طلبٌ يوافق عليه شخصٌ ثانٍ (`DecideRequest` ⇒ `GrantTx`).
func (s *Service) Grant(ctx context.Context, actorID, userID, kind string,
	amount int64, reason string) (*Entry, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	e, err := s.GrantTx(ctx, tx, actorID, userID, kind, amount, reason)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return e, nil
}

// AvailableOn **المتاحُ في محفظته** — الرصيدُ ناقصَ المحجوز (`XG-12`).
func AvailableOn(ctx context.Context, q dbtx.Querier, userID string) (int64, error) {
	var balance int64
	err := q.QueryRow(ctx,
		`SELECT COALESCE((SELECT balance  FROM wallets WHERE user_id = $1), 0)
		      - COALESCE((SELECT reserved FROM wallets WHERE user_id = $1), 0)`, userID).
		Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return balance, err
}

// GrantTx كـ`Grant` **في معاملةٍ مُمرَّرة** — `XG-33`.
//
// **واليدويُّ «تقدير» دائماً** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٦): لا
// `for_target` هنا أبداً — **مكافأةُ الهدف الآليّةُ وحدَها لها شهرٌ ومرحلة**،
// وتقاريرُ «كم صرفنا على الأهداف» تبقى صادقة.
func (s *Service) GrantTx(ctx context.Context, q dbtx.Querier,
	actorID, userID, kind string, amount int64, reason string) (*Entry, error) {
	if kind != KindReward && kind != KindPenalty {
		return nil, ErrBadKind
	}
	if amount <= 0 {
		return nil, ErrBadAmount
	}
	if reason = strings.TrimSpace(reason); reason == "" {
		return nil, ErrNeedsReason
	}

	// **والعقوبةُ تُفحص قبل أن تُقيَّد** — **من المتاح لا من المحجوز** (`XG-12`):
	// قيدُ القاعدة يرفض السالبَ كلَّه، **فيُردّ الطلبُ برسالةٍ تُقرأ.**
	signed := amount
	if kind == KindPenalty {
		balance, err := AvailableOn(ctx, q, userID)
		if err != nil {
			return nil, err
		}
		if balance < amount {
			return nil, ErrNoBalance
		}
		signed = -amount
	}

	if _, err := s.wallet.ApplyTx(ctx, q, userID, signed, kind, "", reason, &actorID); err != nil {
		return nil, err
	}
	// **والخزينةُ الطرفُ المقابل** — تدفع المكافأةَ وتقبض العقوبة.
	if tid := s.treasury(ctx); tid != "" {
		note := "مكافأةٌ صُرفت"
		if kind == KindPenalty {
			note = "عقوبةٌ حُصّلت"
		}
		if _, err := s.wallet.ApplyTx(ctx, q, tid, -signed, kind, "", note, &actorID); err != nil {
			return nil, err
		}
	}

	var e Entry
	if err := q.QueryRow(ctx, `
		INSERT INTO incentives (user_id, kind, amount, reason, for_target, created_by)
		VALUES ($1, $2, $3, $4, false, $5)
		RETURNING id::text, kind, amount, reason, for_target, created_at::text`,
		userID, kind, amount, reason, actorID).
		Scan(&e.ID, &e.Kind, &e.Amount, &e.Reason, &e.ForTarget, &e.CreatedAt); err != nil {
		return nil, err
	}
	return &e, nil
}
