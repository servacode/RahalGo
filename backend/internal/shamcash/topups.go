package shamcash

// ══════════════════════════════════════════════════════════════════════
// **شحن المحفظة بشام كاش** — قرار المالك ٢٠٢٦-١١-٢٥ (هجرة ٠٥٦٢)
// ══════════════════════════════════════════════════════════════════════
//
//	الزبون    بيحوّل على حساب المنصة ⇒ بيكتب المبلغ ورقم العملية (وصورة الإيصال اختياريّة) ⇒ «معلّق»
//	المالية   بتشوف التحويل بحسابها ⇒ «تم، اشحن» (قيد `topup` بالمحفظة + حركة `customer_topup`
//	          داخلة بهالسجلّ، بنفس المعاملة) أو «مرفوض» بسبب
//
// **وهون الحركة الوحيدة بالسجلّ اللي إلها قيد بالدفتر** — لأنّ المال فعلاً دخل حساب المنصة
// وصار للزبون بمحفظته. والثابت FI-05.m بيتأكّد إنّ لكلّ شحن مقبول قيد واحد وحركة وحدة بالضبط.
//
// # والتأكيد الآليّ لاحقاً (`Verifier`)
//
// اليوم التأكيد يدويّ: `ManualVerifier` بيقول «ما بعرف» والقرار للموظّف. **وبكرا لمّا يصير في سجلّ
// تجاري** وربط رسميّ مع شام كاش (createBill/webhook)، بينكتب `Verifier` تاني بيقول «وصل»، والخادم
// بيوافق لحاله بنفس دالّة الموافقة — **وشاشة الزبون ما بتتغيّر ولا حرف.**

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// KindCustomerTopup **حركة «شحن زبون»** بالسجلّ — داخلة بالليرة.
const KindCustomerTopup = "customer_topup"

// حالات طلب الشحن.
const (
	TopupPending  = "pending"
	TopupApproved = "approved"
	TopupRejected = "rejected"
)

var (
	// ErrTopupDisabled **الشحن بشام كاش مطفي** من الإعدادات.
	ErrTopupDisabled = httpx.NewError(http.StatusConflict, "shamcash_topup_disabled", "errors.shamcash_topup_disabled")
	// ErrTopupMin **أقل من الحدّ الأدنى.**
	ErrTopupMin = httpx.NewError(http.StatusBadRequest, "shamcash_topup_min", "errors.shamcash_topup_min")
	// ErrTxnDuplicate **رقم العملية مستعمل** بطلب معلّق أو مقبول.
	ErrTxnDuplicate = httpx.NewError(http.StatusConflict, "shamcash_txn_duplicate", "errors.shamcash_txn_duplicate")
	// ErrTxnInvalid **رقم العملية فاضي أو غريب.**
	ErrTxnInvalid = httpx.NewError(http.StatusBadRequest, "shamcash_txn_invalid", "errors.shamcash_txn_invalid")
	// ErrReceiptInvalid **صورة الإيصال مو إلو أو مو من نوعها.**
	ErrReceiptInvalid = httpx.NewError(http.StatusBadRequest, "shamcash_receipt_invalid", "errors.validation")
	// ErrTopupDecided **الطلب انقرّر** (مرفوض ورجع حدا كبس «اشحن»، أو العكس).
	ErrTopupDecided = httpx.NewError(http.StatusConflict, "request_decided", "errors.request_decided")
	// ErrTooManyPending **كتير طلبات معلّقة لنفس الزبون** — حماية من الإغراق.
	ErrTooManyPending = httpx.NewError(http.StatusTooManyRequests, "shamcash_topup_pending_limit", "errors.shamcash_topup_pending_limit")
)

// MaxPendingPerUser **سقف الطلبات المعلّقة لزبون واحد.**
const MaxPendingPerUser = 5

// ── الفاصل للتأكيد الآليّ ──────────────────────────────────────────────

// Verdict **شو قال المتحقّق عن التحويل.**
type Verdict int

const (
	// VerdictUnknown **ما بعرف** — بيضل معلّق والقرار لموظّف.
	VerdictUnknown Verdict = iota
	// VerdictConfirmed **التحويل وصل بالمبلغ نفسه** — الخادم بيوافق لحاله.
	VerdictConfirmed
	// VerdictRejected **التحويل ما في منه** — الخادم بيرفض لحاله بالسبب.
	VerdictRejected
)

// Verification **جواب المتحقّق.**
type Verification struct {
	Verdict Verdict
	// Reason **سبب الرفض** كما بيوصل للزبون (مع VerdictRejected).
	Reason string
}

// Verifier **مين بيتأكّد إنّ التحويل وصل.** اليوم `ManualVerifier`؛ وبكرا ربط رسميّ.
type Verifier interface {
	// Name **بينكتب بـ`verifier`** على الطلب لما يتقرّر — `manual` للموظّف.
	Name() string
	// Verify **بيتنادى بعد ما ينكتب الطلب** (برّا معاملته). ما بيغيّر شي بالقاعدة.
	Verify(ctx context.Context, t Topup) (Verification, error)
}

// ManualVerifier **التأكيد بإيد موظّف** — دايماً «ما بعرف».
type ManualVerifier struct{}

// Name «manual».
func (ManualVerifier) Name() string { return "manual" }

// Verify **ما بيعرف** — الموظّف بيشوف حسابه وبيقرّر.
func (ManualVerifier) Verify(context.Context, Topup) (Verification, error) {
	return Verification{Verdict: VerdictUnknown}, nil
}

// ── الطلب ────────────────────────────────────────────────────────────────

// Topup **طلب شحن** كما بينعرض.
type Topup struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	UserName       string     `json:"user_name"`
	UserPhone      string     `json:"user_phone,omitempty"`
	Currency       string     `json:"currency"`
	Amount         int64      `json:"amount"`
	TxnRef         string     `json:"txn_ref"`
	ReceiptMediaID *string    `json:"receipt_media_id"`
	ReceiptPath    *string    `json:"-"`
	ReceiptURL     *string    `json:"receipt_url"`
	Status         string     `json:"status"`
	RejectReason   string     `json:"reject_reason"`
	Verifier       string     `json:"verifier"`
	DecidedByName  *string    `json:"decided_by_name"`
	DecidedAt      *time.Time `json:"decided_at"`
	MovementID     *string    `json:"movement_id"`
	CreatedAt      time.Time  `json:"created_at"`
	// RejectedBefore **نفس رقم العملية انرفض قبل** — علامة للموظّف يدقّق.
	RejectedBefore bool `json:"rejected_before"`
}

// NormalizeTxn **رقم العملية بعد التنظيف**: بلا فراغات وشرطات، أحرف كبيرة، والأرقام العربية لاتينية.
// ويرجع فاضي إذا فيه شي غير أرقام وأحرف لاتينية أو طوله برّا ٣..٦٤.
func NormalizeTxn(v string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(v) {
		switch {
		case r == ' ' || r == '-' || r == '_' || r == ' ':
			continue
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		case r < unicode.MaxASCII && (unicode.IsDigit(r) || unicode.IsLetter(r)):
			b.WriteRune(unicode.ToUpper(r))
		default:
			return ""
		}
	}
	out := b.String()
	if len(out) < 3 || len(out) > 64 {
		return ""
	}
	return out
}

// CreateTopup **طلب شحن جديد** — المبلغ والحدّ الأدنى انفحصوا عند المستدعي.
func CreateTopup(ctx context.Context, q dbtx.Querier, userID string, amount int64, txnRef, receiptID string) (string, error) {
	if amount <= 0 {
		return "", ErrTopupMin
	}
	if txnRef == "" {
		return "", ErrTxnInvalid
	}
	// **قفل للزبون** — طلبان بنفس اللحظة ما بيتخطّوا سقف المعلّق.
	if _, err := q.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('shamcash-topup:' || $1))`, userID); err != nil {
		return "", err
	}
	var pending int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM shamcash_topups WHERE user_id = $1::uuid AND status = 'pending'`,
		userID).Scan(&pending); err != nil {
		return "", err
	}
	if pending >= MaxPendingPerUser {
		return "", ErrTooManyPending
	}
	if receiptID != "" {
		var ok bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM media m WHERE m.id::text = $1 AND m.kind = 'shamcash_receipt'
			                 AND m.created_by = $2::uuid)
			   AND NOT EXISTS (SELECT 1 FROM shamcash_topups t WHERE t.receipt_media_id::text = $1)`,
			receiptID, userID).Scan(&ok); err != nil {
			return "", err
		}
		if !ok {
			return "", ErrReceiptInvalid
		}
	}
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO shamcash_topups (user_id, amount, txn_ref, receipt_media_id)
		VALUES ($1::uuid, $2, $3, NULLIF($4, '')::uuid) RETURNING id::text`,
		userID, amount, txnRef, receiptID).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return "", ErrTxnDuplicate
	}
	return id, err
}

// TopupFilter **مرشّحات قائمة اللوحة** — والفارغ «الكل».
type TopupFilter struct {
	Status   string // pending | approved | rejected
	Search   string // رقم العملية أو اسم الزبون أو رقمه
	From, To string // YYYY-MM-DD بيوم دمشق
	UserID   string
}

func (f TopupFilter) where() (string, []any) {
	conds := []string{"TRUE"}
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "$?", "$"+strconv.Itoa(len(args))))
	}
	if f.Status != "" {
		add("t.status = $?", f.Status)
	}
	if f.UserID != "" {
		add("t.user_id = $?::uuid", f.UserID)
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		// **والعلامة نفسها تلات مرّات** — `add` بيبدّل كل `$?` بنفس الرقم.
		add("(t.txn_ref ILIKE '%' || $? || '%' OR u.full_name ILIKE '%' || $? || '%' OR u.phone ILIKE '%' || $? || '%')", s)
	}
	if f.From != "" {
		add("(t.created_at AT TIME ZONE 'Asia/Damascus')::date >= $?::date", f.From)
	}
	if f.To != "" {
		add("(t.created_at AT TIME ZONE 'Asia/Damascus')::date <= $?::date", f.To)
	}
	return strings.Join(conds, " AND "), args
}

const topupCols = `
	t.id::text, t.user_id::text, COALESCE(NULLIF(u.full_name, ''), u.phone, ''), COALESCE(u.phone, ''),
	t.currency, t.amount, t.txn_ref, t.receipt_media_id::text, rm.path,
	t.status, t.reject_reason, t.verifier, NULLIF(COALESCE(NULLIF(d.full_name, ''), d.phone, ''), ''),
	t.decided_at, t.movement_id::text, t.created_at,
	EXISTS (SELECT 1 FROM shamcash_topups o WHERE o.txn_ref = t.txn_ref AND o.id <> t.id AND o.status = 'rejected')`

const topupFrom = `
	FROM shamcash_topups t
	JOIN users u ON u.id = t.user_id
	LEFT JOIN users d ON d.id = t.decided_by
	LEFT JOIN media rm ON rm.id = t.receipt_media_id`

func scanTopup(row pgx.Row) (Topup, error) {
	var t Topup
	err := row.Scan(&t.ID, &t.UserID, &t.UserName, &t.UserPhone, &t.Currency, &t.Amount, &t.TxnRef,
		&t.ReceiptMediaID, &t.ReceiptPath, &t.Status, &t.RejectReason, &t.Verifier, &t.DecidedByName,
		&t.DecidedAt, &t.MovementID, &t.CreatedAt, &t.RejectedBefore)
	return t, err
}

// ListTopups **الطلبات الأحدث أولاً** — والمعلّق فوق.
func ListTopups(ctx context.Context, q dbtx.Querier, f TopupFilter, limit, offset int) ([]Topup, int, error) {
	where, args := f.where()
	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM shamcash_topups t JOIN users u ON u.id = t.user_id WHERE `+where,
		args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := q.Query(ctx, `SELECT `+topupCols+topupFrom+` WHERE `+where+`
		ORDER BY (t.status = 'pending') DESC, t.created_at DESC
		LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Topup{}
	for rows.Next() {
		t, err := scanTopup(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

// GetTopup **طلب واحد.**
func GetTopup(ctx context.Context, q dbtx.Querier, id string) (Topup, error) {
	t, err := scanTopup(q.QueryRow(ctx, `SELECT `+topupCols+topupFrom+` WHERE t.id = $1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, httpx.ErrNotFound
	}
	return t, err
}

// CountPending **عدد المعلّق** — لشارة اللوحة.
func CountPending(ctx context.Context, q dbtx.Querier) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT count(*) FROM shamcash_topups WHERE status = 'pending'`).Scan(&n)
	return n, err
}

// LockTopup **يقفل الطلب ويرجّع صاحبه ومبلغه وحالته.**
func LockTopup(ctx context.Context, q dbtx.Querier, id string) (userID string, amount int64, txnRef, status string, err error) {
	err = q.QueryRow(ctx, `SELECT user_id::text, amount, txn_ref, status FROM shamcash_topups WHERE id = $1::uuid FOR UPDATE`, id).
		Scan(&userID, &amount, &txnRef, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		err = httpx.ErrNotFound
	}
	return
}

// RecordTopupMovement **حركة «شحن زبون» داخلة بالسجلّ** — مرّة وحدة لكل طلب (فهرس فريد).
func RecordTopupMovement(ctx context.Context, q dbtx.Querier, topupID string, amount int64, actor string) (string, error) {
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO shamcash_movements (kind, topup_id, currency, amount, note, created_by)
		VALUES ('customer_topup', $1::uuid, 'SYP', $2, 'شحن محفظة زبون', NULLIF($3, '')::uuid)
		RETURNING id::text`, topupID, amount, actor).Scan(&id)
	return id, err
}

// MarkApproved **يثبّت الموافقة وأثرها** على الطلب.
func MarkApproved(ctx context.Context, q dbtx.Querier, id string, walletTxID int64, movementID, verifier, actor string) error {
	_, err := q.Exec(ctx, `
		UPDATE shamcash_topups SET status = 'approved', decided_at = now(), decided_by = NULLIF($2, '')::uuid,
		       verifier = $3, wallet_tx_id = $4, movement_id = $5::uuid
		WHERE id = $1::uuid AND status = 'pending'`, id, actor, verifier, walletTxID, movementID)
	return err
}

// MarkRejected **يرفض بسبب.**
func MarkRejected(ctx context.Context, q dbtx.Querier, id, reason, verifier, actor string) error {
	_, err := q.Exec(ctx, `
		UPDATE shamcash_topups SET status = 'rejected', decided_at = now(), decided_by = NULLIF($2, '')::uuid,
		       verifier = $3, reject_reason = $4
		WHERE id = $1::uuid AND status = 'pending'`, id, actor, verifier, reason)
	return err
}
