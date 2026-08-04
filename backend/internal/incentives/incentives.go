package incentives

// الأهدافُ والمكافآتُ والعقوبات — **ما لا معادلةَ له.**
//
// # القسمة
//
//	الأجرُ والعمولة  ←  معادلةٌ تقع وحدَها (`orders`)
//	المكافأةُ والعقوبة ←  تقديرُ إنسانٍ يُقيَّد بكلمة
//
// **والهدفُ بينهما**: يُحسب آلياً ولا يدفع شيئاً — **يقول من يستحقّ، ولا
// يُعطي.** (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «بيدك، والشاشةُ تقول من بلغ».)
//
// # ولماذا لا تقع المكافأةُ وحدَها
//
// **رقمٌ يدفع بلا يدٍ لا يُراجَع.** ومن بلغ الهدفَ بثلاثين طلباً صغيراً ليس
// كمن بلغه بثلاثين في ليالي المطر — **والفرقُ يراه إنسانٌ ولا تراه معادلة.**
// وخطأٌ في رقمٍ تلقائيٍّ يُصرف على الجميع قبل أن يُلاحظ.
//
// # والشهرُ يبدأ بتوقيت دمشق
//
// **لا بغرينتش**: شهرُ السائق ينتهي عنده لا في لندن، **وطلبٌ سُلّم الساعةَ
// الواحدةَ بعد منتصف الليل يُحسب على ليلته لا على شهرٍ جديد.**

import (
	"context"
	"errors"
	"net/http"
	"strings"

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
}

// Settings ما يلزم من الإعدادات — **واجهةٌ ضيّقة**: هذه الحزمةُ تقرأ رقمين.
type Settings interface {
	GetInt(ctx context.Context, key string) int64
}

func New(db *pgxpool.Pool, w *wallet.Service, st Settings,
	treasury func(ctx context.Context) string) *Service {
	return &Service{db: db, wallet: w, settings: st, treasury: treasury}
}

// Standing حالُ شخصٍ هذا الشهر — هدفُه وما بلغ وما ناله.
type Standing struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	// Done ما أنجزه هذا الشهر — طلباتٌ سُلّمت.
	Done int `json:"done"`
	// Target هدفُه — **وصفرٌ يعني لا هدف**، فلا شارةَ ولا شريط.
	Target int `json:"target"`
	// Reached بلغ أم لا — **يقوله الخادمُ ولا يُستنتج في الشاشة**: شرطٌ
	// يُحسب في موضعين يفترق يوماً، فتُهنّئ الشاشةُ من لم يبلغ.
	Reached bool `json:"reached"`
	// Rewarded ما نال هذا الشهر، و Penalized ما خُصم منه.
	Rewarded  int64 `json:"rewarded"`
	Penalized int64 `json:"penalized"`
}

// monthStart بدايةُ الشهر بتوقيت دمشق — **تعبيرٌ واحدٌ يُعاد استعماله.**
const monthStart = `date_trunc('month', now() AT TIME ZONE 'Asia/Damascus')`

// Standings حالُ كلّ من في هذا الدور.
//
// # ولماذا استعلامٌ واحد
//
// نداءٌ لكلّ سائقٍ يعني عشرين نداءً لشاشةٍ واحدة. **وشاشةٌ بطيئةٌ لا تُفتح**،
// فلا تُقرأ الأهدافُ أصلاً.
func (s *Service) Standings(ctx context.Context, role string) ([]Standing, error) {
	var target int
	switch role {
	case "driver":
		target = int(s.settings.GetInt(ctx, "drivers.monthly_target"))
	case "sales":
		target = int(s.settings.GetInt(ctx, "sales.monthly_target"))
	default:
		return nil, httpx.ErrNotFound
	}

	// **والإنجازُ يختلف بالدور** — السائقُ بما وصّل، والمندوبُ بما بيع من
	// متاجرَ جلبها. **والمعنى واحد: ما أنتجه عملُه.**
	done := `(SELECT count(*) FROM orders o
	          WHERE o.driver_id = u.id AND o.status = 'delivered'
	            AND o.delivered_at AT TIME ZONE 'Asia/Damascus' >= ` + monthStart + `)`
	if role == "sales" {
		done = `(SELECT count(*) FROM orders o
		          JOIN merchants mm ON mm.id = o.merchant_id
		          WHERE mm.sales_rep_user_id = u.id AND o.status = 'delivered'
		            AND o.delivered_at AT TIME ZONE 'Asia/Damascus' >= ` + monthStart + `)`
	}

	rows, err := s.db.Query(ctx, `
		SELECT u.id::text, COALESCE(NULLIF(u.full_name, ''), ''), u.phone::text,
		       `+done+`,
		       COALESCE((SELECT sum(i.amount) FROM incentives i
		                 WHERE i.user_id = u.id AND i.kind = 'reward'
		                   AND i.created_at AT TIME ZONE 'Asia/Damascus' >= `+monthStart+`), 0),
		       COALESCE((SELECT sum(i.amount) FROM incentives i
		                 WHERE i.user_id = u.id AND i.kind = 'penalty'
		                   AND i.created_at AT TIME ZONE 'Asia/Damascus' >= `+monthStart+`), 0)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = $1
		WHERE u.deleted_at IS NULL
		ORDER BY 4 DESC, u.full_name`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Standing{}
	for rows.Next() {
		var x Standing
		if err := rows.Scan(&x.UserID, &x.Name, &x.Phone, &x.Done,
			&x.Rewarded, &x.Penalized); err != nil {
			return nil, err
		}
		x.Target = target
		// **وبلوغٌ بلا هدفٍ ليس بلوغاً** — صفرٌ يعني «لا هدف»، ومن أنجز
		// طلباً واحداً ليس بالغاً شيئاً.
		x.Reached = target > 0 && x.Done >= target
		out = append(out, x)
	}
	return out, rows.Err()
}

// MyStanding حالُ صاحب الحساب نفسِه.
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
// **ودفترٌ يأخذ من طرفٍ ولا يعطي آخرَ لا يتوازن**: مكافأةٌ تُقيَّد للسائق وحدَه
// تجعل المنصةَ تظهر رابحةً وهي تدفع، **وعقوبةٌ تُخصم منه ولا تعود إليها تجعلها
// تظهر خاسرةً وقد قبضت.**
func (s *Service) Grant(ctx context.Context, actorID, userID, kind string,
	amount int64, reason string, forTarget bool) (*Entry, error) {
	if kind != KindReward && kind != KindPenalty {
		return nil, ErrBadKind
	}
	if amount <= 0 {
		return nil, ErrBadAmount
	}
	if reason = strings.TrimSpace(reason); reason == "" {
		return nil, ErrNeedsReason
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// **والعقوبةُ تُفحص قبل أن تُقيَّد**: قيدُ القاعدة يرفض السالبَ كلَّه،
	// **فيُردّ الطلبُ برسالةٍ تُقرأ بدل خطأٍ لا يفهمه الموظّف.**
	signed := amount
	if kind == KindPenalty {
		var balance int64
		if err := tx.QueryRow(ctx,
			`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1`, userID).
			Scan(&balance); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if balance < amount {
			return nil, ErrNoBalance
		}
		signed = -amount
	}

	if _, err := s.wallet.ApplyTx(ctx, tx, userID, signed, kind, "", reason, &actorID); err != nil {
		return nil, err
	}
	// **والخزينةُ الطرفُ المقابل** — تدفع المكافأةَ وتقبض العقوبة.
	if tid := s.treasury(ctx); tid != "" {
		note := "مكافأةٌ صُرفت"
		if kind == KindPenalty {
			note = "عقوبةٌ حُصّلت"
		}
		if _, err := s.wallet.ApplyTx(ctx, tx, tid, -signed, kind, "", note, &actorID); err != nil {
			return nil, err
		}
	}

	var e Entry
	if err := tx.QueryRow(ctx, `
		INSERT INTO incentives (user_id, kind, amount, reason, for_target, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, kind, amount, reason, for_target, created_at::text`,
		userID, kind, amount, reason, forTarget, actorID).
		Scan(&e.ID, &e.Kind, &e.Amount, &e.Reason, &e.ForTarget, &e.CreatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &e, nil
}
