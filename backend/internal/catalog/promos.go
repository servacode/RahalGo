package catalog

// ══════════════════════════════════════════════════════════════════════
// **أكوادُ الخصم** — قراراتُ المالك ٢٠٢٦-١٠-٠٤ (قسمُ «العروض والخصومات»)
// ══════════════════════════════════════════════════════════════════════
//
//	١ · دورُ المحتوى يعمل كوداً بحرّيّةٍ حتّى ٢٠٪ و٥٠ استخداماً، وفوقه
//	    يُنشأ الكودُ موقوفاً بانتظار الماليّة (`promo_approvals`).
//	٢ · نسبةُ الكود ١–٩٠، وسقفٌ بالليرة إجباريٌّ لكود النسبة — يُفحص هنا.
//	  · الحالةُ الحقيقيّة يقولها الخادم: سارٍ · منتهٍ · خلص سقفه · موقوف ·
//	    بانتظار الموافقة · مرفوض.
//	  · التعديلُ يمسح التاريخَ أو السقف، والكودُ تُشال فراغاتُه، والقيمةُ
//	    السالبةُ رسالةٌ واضحةٌ لا خطأٌ داخليّ.

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/offers"
)

var ErrCodeTaken = httpx.NewError(http.StatusConflict, "code_taken", "errors.code_taken")

// **ورسائلُ التحقّق في `offers`** — لوحةُ الإدارة وحدَها تقرؤها، والتطبيقاتُ
// لا تنادي هذا الباب.
var (
	ErrPromoCode     = offers.ErrPromoCode
	ErrPromoKind     = offers.ErrPromoKind
	ErrPromoPercent  = offers.ErrPromoPercent
	ErrPromoCap      = offers.ErrPromoCap
	ErrPromoAmount   = offers.ErrPromoAmount
	ErrPromoNegative = offers.ErrPromoNegative
	ErrPromoExpired  = offers.ErrPromoExpired
)

// حالاتُ الكود المشتقّة — **تُشتقّ ولا تُخزَّن.**
const (
	PromoActive    = "active"
	PromoExpired   = "expired"
	PromoExhausted = "exhausted"
	PromoPaused    = "paused"
	PromoPending   = "pending_approval"
	PromoRejected  = "rejected"
)

type PromoCode struct {
	ID             string     `json:"id"`
	Code           string     `json:"code"`
	Kind           string     `json:"kind"`
	Value          int64      `json:"value"`
	MaxDiscount    *int64     `json:"max_discount"`
	MinOrder       int64      `json:"min_order"`
	FirstOrderOnly bool       `json:"first_order_only"`
	OncePerUser    bool       `json:"once_per_user"`
	MaxUses        *int       `json:"max_uses"`
	UsedCount      int        `json:"used_count"`
	ExpiresAt      *time.Time `json:"expires_at"`
	Active         bool       `json:"active"`
	CreatedAt      time.Time  `json:"created_at"`
	ApprovalState  string     `json:"approval_state"`
	CreatedByName  string     `json:"created_by_name"`
	// Status **الحالةُ الحقيقيّة** — `PromoStatusAt`.
	Status string `json:"status"`
	// Cost **ما كلّفه الكودُ لحدّ الآن** — خصمُ الطلبات المسلَّمة + ما
	// أُعفي منه الزبونُ من التوصيل.
	Cost int64 `json:"cost"`
}

// PromoStatusAt **حالةُ الكود عند لحظة** — دالّةٌ خالصةٌ تُقاس بلا قاعدة.
//
// الموافقةُ أوّلاً (ما ينتظر الماليّة لا يسري أيّاً كان)، ثمّ الانتهاء، ثمّ
// السقف، ثمّ الإيقاف.
func PromoStatusAt(active bool, approval string, expiresAt *time.Time, maxUses *int, used int, now time.Time) string {
	switch {
	case approval == offers.ApprovalPending:
		return PromoPending
	case approval == offers.ApprovalRejected:
		return PromoRejected
	case expiresAt != nil && !expiresAt.After(now):
		return PromoExpired
	case maxUses != nil && used >= *maxUses:
		return PromoExhausted
	case !active:
		return PromoPaused
	}
	return PromoActive
}

// PromoLiveSQL **شرطُ السريان في SQL** — بحرف `PromoStatusAt` لحالة «سارٍ»،
// يقرؤه عدّادُ الرئيسيّة والملخّص.
const PromoLiveSQL = `(p.active AND p.approval_state = 'ok'
	AND (p.expires_at IS NULL OR p.expires_at > now())
	AND (p.max_uses IS NULL OR p.used_count < p.max_uses))`

const promoSelect = `
	SELECT p.id, p.code, p.kind, p.value, p.max_discount, p.min_order, p.first_order_only,
	       p.once_per_user, p.max_uses, p.used_count, p.expires_at, p.active, p.created_at,
	       p.approval_state, COALESCE(u.full_name, ''),
	       COALESCE((SELECT sum(o.discount + o.promo_delivery_waived)
	                   FROM promo_redemptions r JOIN orders o ON o.id = r.order_id
	                  WHERE r.promo_id = p.id AND o.status = 'delivered'), 0)
	FROM promo_codes p
	LEFT JOIN users u ON u.id = p.created_by`

func scanPromo(row pgx.Row) (*PromoCode, error) {
	var p PromoCode
	err := row.Scan(&p.ID, &p.Code, &p.Kind, &p.Value, &p.MaxDiscount, &p.MinOrder, &p.FirstOrderOnly,
		&p.OncePerUser, &p.MaxUses, &p.UsedCount, &p.ExpiresAt, &p.Active, &p.CreatedAt,
		&p.ApprovalState, &p.CreatedByName, &p.Cost)
	if err != nil {
		return nil, err
	}
	p.Status = PromoStatusAt(p.Active, p.ApprovalState, p.ExpiresAt, p.MaxUses, p.UsedCount, time.Now())
	return &p, nil
}

func (s *Service) ListPromos(ctx context.Context) ([]PromoCode, error) {
	rows, err := s.db.Query(ctx, promoSelect+` ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PromoCode{}
	for rows.Next() {
		p, err := scanPromo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (s *Service) getPromo(ctx context.Context, q pgxQuerier, id string) (*PromoCode, error) {
	p, err := scanPromo(q.QueryRow(ctx, promoSelect+` WHERE p.id = $1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	return p, err
}

type pgxQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PromoInput struct {
	Code           *string    `json:"code"`
	Kind           *string    `json:"kind"`
	Value          *int64     `json:"value"`
	MaxDiscount    *int64     `json:"max_discount"`
	MinOrder       *int64     `json:"min_order"`
	FirstOrderOnly *bool      `json:"first_order_only"`
	OncePerUser    *bool      `json:"once_per_user"`
	MaxUses        *int       `json:"max_uses"`
	ExpiresAt      *time.Time `json:"expires_at"`
	// ExpiresOn **يومُ الانتهاء** `YYYY-MM-DD` — آخرَ اليوم بتوقيت دمشق.
	ExpiresOn *string `json:"expires_on"`
	Active    *bool   `json:"active"`
	// ClearExpiry و ClearMaxUses **مسحُ التاريخ أو السقف في التعديل** —
	// `null` في JSON لا يُفرَّق عن الغياب.
	ClearExpiry  bool `json:"clear_expiry"`
	ClearMaxUses bool `json:"clear_max_uses"`
}

// normCode **الكودُ كما يُحفظ** — بلا فراغٍ حوله وبحروفٍ كبيرة، ولا فراغَ
// أو رمزٌ داخله (الزبونُ يكتبه في السلّة كما يُطبع).
func normCode(raw string) (string, error) {
	c := strings.ToUpper(strings.TrimSpace(raw))
	if len(c) < 3 || len(c) > 32 {
		return "", ErrPromoCode
	}
	for _, r := range c {
		if !(r == '-' || r == '_' || (r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)))) {
			return "", ErrPromoCode
		}
	}
	return c, nil
}

// promoState **الكودُ بعد تطبيق المُدخَل** — يُفحص كلُّه معاً.
type promoState struct {
	kind        string
	value       int64
	maxDiscount *int64
	minOrder    int64
	maxUses     *int
	expiresAt   *time.Time
}

func (st promoState) validate() error {
	if st.value < 0 || st.minOrder < 0 || (st.maxUses != nil && *st.maxUses < 1) {
		return ErrPromoNegative
	}
	switch st.kind {
	case "percent":
		if st.value < 1 || st.value > 90 {
			return ErrPromoPercent
		}
		if st.maxDiscount == nil || *st.maxDiscount <= 0 {
			return ErrPromoCap
		}
	case "fixed":
		if st.value < 1 {
			return ErrPromoAmount
		}
	case "free_delivery":
	default:
		return ErrPromoKind
	}
	return nil
}

// needsApproval **فوق حدّ المحتوى؟** — بلا سقف استخدامات، أو فوق ٥٠، أو
// نسبةٌ فوق ٢٠، أو مبلغٌ ثابتٌ فوق ٢٠٪ من الحدّ الأدنى للطلب.
func (st promoState) needsApproval() bool {
	if st.maxUses == nil || *st.maxUses > offers.FreeUses {
		return true
	}
	switch st.kind {
	case "percent":
		return st.value > offers.FreePercent
	case "fixed":
		return st.value*100 > st.minOrder*offers.FreePercent
	}
	return false
}

// maxCost **أقصى كلفةٍ معروفة** — صفرٌ إن لم تُعرف (توصيلٌ مجّانيّ أو بلا سقف).
func (st promoState) maxCost() int64 {
	if st.maxUses == nil {
		return 0
	}
	switch st.kind {
	case "percent":
		if st.maxDiscount != nil {
			return *st.maxDiscount * int64(*st.maxUses)
		}
	case "fixed":
		return st.value * int64(*st.maxUses)
	}
	return 0
}

func (st promoState) approvalNote(code string) string {
	var b strings.Builder
	b.WriteString("كود خصم " + code + ": ")
	switch st.kind {
	case "percent":
		b.WriteString("نسبة " + strconv.FormatInt(st.value, 10) + "٪")
		if st.maxDiscount != nil {
			b.WriteString(" بسقف " + strconv.FormatInt(*st.maxDiscount, 10) + " ل.س")
		}
	case "fixed":
		b.WriteString("مبلغ ثابت " + strconv.FormatInt(st.value, 10) + " ل.س")
	case "free_delivery":
		b.WriteString("توصيل مجاني")
	}
	if st.maxUses == nil {
		b.WriteString(" · بلا سقف استخدامات")
	} else {
		b.WriteString(" · " + strconv.Itoa(*st.maxUses) + " استخدام")
	}
	return b.String()
}

func resolveExpiry(in PromoInput) (*time.Time, error) {
	if in.ExpiresOn != nil && strings.TrimSpace(*in.ExpiresOn) != "" {
		t, err := offers.EndOfDay(*in.ExpiresOn)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}
	return in.ExpiresAt, nil
}

// CreatePromo ينشئ كوداً — **وفوق حدّ المحتوى يُنشأ موقوفاً بانتظار الماليّة.**
func (s *Service) CreatePromo(ctx context.Context, actorID string, in PromoInput, ip string) (*PromoCode, error) {
	if in.Code == nil || in.Kind == nil {
		return nil, ErrPromoCode
	}
	code, err := normCode(*in.Code)
	if err != nil {
		return nil, err
	}
	exp, err := resolveExpiry(in)
	if err != nil {
		return nil, err
	}
	if exp != nil && !exp.After(time.Now()) {
		return nil, ErrPromoExpired
	}
	st := promoState{kind: *in.Kind, maxUses: in.MaxUses, expiresAt: exp}
	if in.Value != nil {
		st.value = *in.Value
	}
	if in.MinOrder != nil {
		st.minOrder = *in.MinOrder
	}
	if st.kind == "percent" {
		st.maxDiscount = in.MaxDiscount
	}
	if st.kind == "free_delivery" {
		st.value = 0
	}
	if err := st.validate(); err != nil {
		return nil, err
	}
	pending := st.needsApproval()
	active := in.Active == nil || *in.Active
	approval := offers.ApprovalOK
	if pending {
		active = false
		approval = offers.ApprovalPending
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO promo_codes (code, kind, value, max_discount, min_order, first_order_only,
		                         once_per_user, max_uses, expires_at, active, approval_state, created_by)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6,false), COALESCE($7,true), $8, $9, $10, $11, $12::uuid)
		RETURNING id::text`,
		code, st.kind, st.value, st.maxDiscount, st.minOrder, in.FirstOrderOnly, in.OncePerUser,
		st.maxUses, st.expiresAt, active, approval, actorID).Scan(&id)
	if isUniqueViolationCatalog(err) {
		return nil, ErrCodeTaken
	}
	if err != nil {
		return nil, err
	}
	if pending {
		if _, err := tx.Exec(ctx, `
			INSERT INTO promo_approvals (target_kind, target_id, amount, note, proposed_by)
			VALUES ('promo', $1::uuid, $2, $3, $4::uuid)`,
			id, st.maxCost(), st.approvalNote(code), actorID); err != nil {
			return nil, err
		}
	}
	p, err := s.getPromo(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	s.auditDetail(ctx, actorID, "admin.promo_create", "promo", p.ID, ip,
		map[string]any{"code": code, "kind": st.kind, "value": st.value, "pending_approval": pending})
	return p, nil
}

// UpdatePromo يعدّل الكود — **وتعديلُ المال فوق الحدّ يعيده إلى الماليّة.**
//
// النوعُ والكودُ لا يتبدّلان (من بدّلهما عمل كوداً آخر). **والتاريخُ والسقفُ
// يُمسحان** بـ`clear_expiry` و`clear_max_uses`.
func (s *Service) UpdatePromo(ctx context.Context, actorID, id string, in PromoInput, ip string) (*PromoCode, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var st promoState
	var code, approval string
	var active bool
	err = tx.QueryRow(ctx, `
		SELECT code, kind, value, max_discount, min_order, max_uses, expires_at, active, approval_state
		FROM promo_codes WHERE id = $1::uuid FOR UPDATE`, id).
		Scan(&code, &st.kind, &st.value, &st.maxDiscount, &st.minOrder, &st.maxUses,
			&st.expiresAt, &active, &approval)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	before := st
	money := false
	if in.Value != nil && st.kind != "free_delivery" && *in.Value != st.value {
		st.value, money = *in.Value, true
	}
	if in.MaxDiscount != nil && st.kind == "percent" &&
		(st.maxDiscount == nil || *in.MaxDiscount != *st.maxDiscount) {
		st.maxDiscount, money = in.MaxDiscount, true
	}
	if in.MinOrder != nil && *in.MinOrder != st.minOrder {
		st.minOrder, money = *in.MinOrder, true
	}
	if in.ClearMaxUses {
		if st.maxUses != nil {
			st.maxUses, money = nil, true
		}
	} else if in.MaxUses != nil && (st.maxUses == nil || *in.MaxUses != *st.maxUses) {
		st.maxUses, money = in.MaxUses, true
	}
	expChanged := false
	if in.ClearExpiry {
		st.expiresAt, expChanged = nil, before.expiresAt != nil
	} else {
		exp, err := resolveExpiry(in)
		if err != nil {
			return nil, err
		}
		if exp != nil {
			if !exp.After(time.Now()) {
				return nil, ErrPromoExpired
			}
			st.expiresAt, expChanged = exp, true
		}
	}
	if err := st.validate(); err != nil {
		return nil, err
	}

	// **والموافقةُ تتبع المالَ لا كلَّ تعديل** — تفعيلٌ أو تاريخٌ لا يعيدها.
	switch {
	case money && st.needsApproval():
		approval = offers.ApprovalPending
		active = false
		if _, err := tx.Exec(ctx, `
			INSERT INTO promo_approvals (target_kind, target_id, amount, note, proposed_by)
			VALUES ('promo', $1::uuid, $2, $3, $4::uuid)
			ON CONFLICT (target_kind, target_id) WHERE status = 'pending'
			DO UPDATE SET amount = excluded.amount, note = excluded.note,
			              proposed_by = excluded.proposed_by, created_at = now()`,
			id, st.maxCost(), st.approvalNote(code), actorID); err != nil {
			return nil, err
		}
	case money && !st.needsApproval() && approval != offers.ApprovalOK:
		// **نزل تحت الحدّ** — لا يحتاج الماليّة بعد، والطلبُ المعلَّقُ يُسحب.
		approval = offers.ApprovalOK
		if _, err := tx.Exec(ctx, `
			DELETE FROM promo_approvals
			WHERE target_kind = 'promo' AND target_id = $1::uuid AND status = 'pending'`, id); err != nil {
			return nil, err
		}
	}
	if in.Active != nil && !(money && approval == offers.ApprovalPending) {
		if *in.Active && approval != offers.ApprovalOK {
			return nil, offers.ErrNeedsApproval
		}
		active = *in.Active
	}

	if _, err := tx.Exec(ctx, `
		UPDATE promo_codes SET
			value = $2, max_discount = $3, min_order = $4,
			first_order_only = COALESCE($5, first_order_only),
			once_per_user    = COALESCE($6, once_per_user),
			max_uses = $7, expires_at = $8, active = $9, approval_state = $10
		WHERE id = $1::uuid`,
		id, st.value, st.maxDiscount, st.minOrder, in.FirstOrderOnly, in.OncePerUser,
		st.maxUses, st.expiresAt, active, approval); err != nil {
		return nil, err
	}
	p, err := s.getPromo(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	s.auditDetail(ctx, actorID, "admin.promo_update", "promo", id, ip,
		map[string]any{"money_changed": money, "expiry_changed": expChanged,
			"approval_state": approval, "active": active})
	return p, nil
}
