package orders

// ══════════════════════════════════════════════════════════════════════
// **عقدُ عرضِ السعر المخصَّص — نسخةٌ وتأكيدٌ وحجزٌ مملوكٌ للطلب** — Batch 2a
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨: الطلبُ المخصَّصُ عرضٌ يُقفَل قبل الشراء.)
//
// # لماذا كلُّ هذا
//
// **كان الاتفاقُ يُدهَس صامتاً**: السائقُ يكتب رقماً فيُكتب، ثمّ يكتب غيرَه
// فيُدهَس، **بلا نسخةٍ ولا تأكيدٍ من الزبون ولا أثرٍ ماليٍّ محفوظ.** فصار:
//
//  1. **نسخةٌ تتزايد مع كلّ تغييرٍ حقيقيّ** — إعادةُ الاتفاق بالقيم نفسِها
//     لا تُزيدها، **فلا يُبطَل تأكيدٌ بلا سبب.**
//  2. **تأكيدُ الزبون مربوطٌ بنسخةٍ ومبلغٍ** — يُقفَل السعرُ الذي رآه هو.
//  3. **حجزٌ للمحفظة مملوكٌ للطلب** (`custom_reserved_amount`) — لا مجمَّعاً
//     عارياً، **فيُطابقه الحارسُ الماليُّ ويُطلَق ويُسوّى لكلّ طلبٍ على حدة.**
//
// # ومسارُ السعر عند التغيير (قرارُ المالك)
//
//   - **زيادةٌ فوق المؤكَّد** → يُبطَل التأكيد، ويُفكّ الحجز، ويُعاد التأكيد،
//     ويُمنَع الاستلامُ حتّى يؤكّد الزبونُ الجديد.
//   - **نقصٌ دون المؤكَّد** → يبقى التأكيد، ويُخفَّض الحجزُ بالفرق، ويُحدَّث
//     المبلغ/النسخة إلى الأدنى — بلا إعادة تأكيد.
//   - **مكوّناتٌ تبدّلت والمجموعُ ثابت** → يُعاد التأكيدُ كذلك (رآه الزبونُ غيرَه).

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// customQuoteRow **لقطةٌ مقفولةٌ من صفّ الطلب** لتطبيق تغييرِ العرض عليها.
type customQuoteRow struct {
	id                 string
	customerID         string
	driverID           *string
	kind               string
	status             string
	paymentMethod      string
	goods              int64 // custom_goods_amount الحاليّة (0 إن لم يُتَّفق بعد)
	fee                int64 // custom_fee الحاليّة
	hasGoods           bool  // أاتُّفق مرّةً على الأقلّ؟
	quoteVersion       int64
	confirmedAt        *time.Time
	confirmedTotal     *int64
	confirmedVersion   *int64
	reserved           int64 // custom_reserved_amount المملوكُ لهذا الطلب
	feeSource          string
	feeSnapshot        *int64
	driverMayChangeFee bool
	pickedUpAt         *time.Time
}

// lockCustomRow يقرأ صفَّ الطلب **بقفلٍ** (`FOR UPDATE`) — فلا يتبدّل تحته.
func (s *Service) lockCustomRow(ctx context.Context, q wallet.Querier, orderID string) (customQuoteRow, error) {
	var r customQuoteRow
	r.id = orderID
	var goods, fee *int64
	err := q.QueryRow(ctx, `
		SELECT customer_id::text, driver_id::text, kind, status, payment_method,
		       custom_goods_amount, custom_fee, quote_version,
		       quote_confirmed_at, quote_confirmed_total, quote_confirmed_version,
		       custom_reserved_amount, custom_fee_source, custom_fee_snapshot,
		       custom_driver_may_change_fee, picked_up_at
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&r.customerID, &r.driverID, &r.kind, &r.status, &r.paymentMethod,
			&goods, &fee, &r.quoteVersion,
			&r.confirmedAt, &r.confirmedTotal, &r.confirmedVersion,
			&r.reserved, &r.feeSource, &r.feeSnapshot,
			&r.driverMayChangeFee, &r.pickedUpAt)
	if err != nil {
		return customQuoteRow{}, err
	}
	if goods != nil {
		r.goods, r.hasGoods = *goods, true
	}
	if fee != nil {
		r.fee = *fee
	}
	return r, nil
}

// reserveCustomTx **حجزٌ مملوكٌ للطلب** — يحجز في المحفظة ويقيّده على الطلب معاً.
//
// **ويُنادى بعد كتابة أعمدة التأكيد** — القيدُ يشترط محفظةً مؤكَّدةً لأيّ حجز.
func (s *Service) reserveCustomTx(ctx context.Context, q wallet.Querier, customerID, orderID string, amount int64) error {
	if amount <= 0 {
		return nil
	}
	if err := s.wallet.ReserveTx(ctx, q, customerID, amount); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `
		UPDATE orders SET custom_reserved_amount = custom_reserved_amount + $2,
		       updated_at = now() WHERE id = $1`, orderID, amount)
	return err
}

// releaseCustomReservationTx **فكُّ حجزٍ مملوكٍ للطلب** — يفكّ من المحفظة
// ويطرح من الطلب معاً، **وصفراً فأقلّ لا شيء** (فكٌّ لا محلَّ له).
func (s *Service) releaseCustomReservationTx(ctx context.Context, q wallet.Querier, customerID, orderID string, amount int64) error {
	if amount <= 0 {
		return nil
	}
	if err := s.wallet.ReleaseTx(ctx, q, customerID, amount); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `
		UPDATE orders SET custom_reserved_amount = custom_reserved_amount - $2,
		       updated_at = now() WHERE id = $1`, orderID, amount)
	return err
}

// releaseCustomReservationOnTerminal **طلبٌ انتهى إلى غير التسليم يُطلق حجزَه.**
//
// **ولا يبقى مالٌ محجوزٌ لطلبٍ أُغلق** — الحارسُ الماليُّ يمنعه، وهذا يمنحه
// إيّاه. **ويُقرأ بقفلٍ** — فلا يُفكّ حجزٌ فُكّ.
func (s *Service) releaseCustomReservationOnTerminal(ctx context.Context, q wallet.Querier, orderID, customerID string) error {
	var reserved int64
	if err := q.QueryRow(ctx,
		`SELECT custom_reserved_amount FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&reserved); err != nil {
		return err
	}
	return s.releaseCustomReservationTx(ctx, q, customerID, orderID, reserved)
}

// quoteMutator **من غيّر العرضَ ولماذا** — يُكتب في التدقيق.
type quoteMutator struct {
	actorID string
	role    string // "driver" | "admin"
	source  string // "driver_agree" | "admin_override"
	reason  string
}

// applyCustomQuoteTx **جوهرُ تغييرِ العرض** — يشترك فيه السائقُ والأدمن.
//
// **يكتب الأعمدة، ويُزيد النسخةَ عند تغييرٍ حقيقيٍّ وحدَه، ويطبّق سياسةَ
// التأكيد/الحجز، ويُدقّق.** **ويردّ `changed=false` بلا أثرٍ إن كانت القيمُ
// هي هي** (إعادةُ اتفاقٍ لا تُبطل تأكيداً ولا تُحرّك حجزاً).
//
// **والمنادي أمسك الصفَّ بقفلٍ** (`lockCustomRow`) ومرّر لقطتَه — فلا قراءةَ
// ثانيةً هنا، ولا يتبدّل الصفُّ بين القراءة والكتابة.
func (s *Service) applyCustomQuoteTx(ctx context.Context, q wallet.Querier, row customQuoteRow,
	newGoods, newFee int64, m quoteMutator) (bool, error) {
	if row.hasGoods && row.goods == newGoods && row.fee == newFee {
		return false, nil
	}
	newTotal := newGoods + newFee
	newVersion := row.quoteVersion + 1

	// ── سياسةُ التأكيد والحجز — لا تنطبق إلّا على تأكيدٍ حيّ ──────────────
	invalidate := false
	if row.confirmedAt != nil {
		confirmedTotal := int64(0)
		if row.confirmedTotal != nil {
			confirmedTotal = *row.confirmedTotal
		}
		switch {
		case newTotal > confirmedTotal:
			invalidate = true // زيادة → يُعاد التأكيد
		case newTotal < confirmedTotal:
			// نقص → يبقى التأكيد ويُخفَّض الحجزُ بالفرق (نقدٌ لا حجزَ له: لا شيء).
			if err := s.releaseCustomReservationTx(ctx, q, row.customerID, row.id, row.reserved-newTotal); err != nil {
				return false, err
			}
		default:
			invalidate = true // مجموعٌ ثابتٌ ومكوّناتٌ تبدّلت → يُعاد التأكيد
		}
	}
	if invalidate {
		if err := s.releaseCustomReservationTx(ctx, q, row.customerID, row.id, row.reserved); err != nil {
			return false, err
		}
	}

	// ── كتابةُ الأعمدة ──────────────────────────────────────────────────
	//
	// **والنوعُ يُقال صراحةً في الجمع** (`$2::bigint + $3`) — وإلّا سألت
	// بوستغرس «أيُّ + هذا؟» بين وسيطين مجهولين فسقط النداءُ بخمسمئة.
	set := `custom_goods_amount = $2, custom_fee = $3,
	        subtotal = $2, delivery_fee = $3, total = $2::bigint + $3::bigint,
	        quote_version = $4, custom_agreed_at = now(), updated_at = now()`
	args := []any{row.id, newGoods, newFee, newVersion}
	switch {
	case invalidate:
		set += `, quote_confirmed_at = NULL, quote_confirmed_total = NULL, quote_confirmed_version = NULL`
	case row.confirmedAt != nil: // نقصٌ مع تأكيدٍ باقٍ: يُحدَّث المبلغُ والنسخة
		set += `, quote_confirmed_total = $5, quote_confirmed_version = $4`
		args = append(args, newTotal)
	}
	if _, err := q.Exec(ctx, `UPDATE orders SET `+set+` WHERE id = $1`, args...); err != nil {
		return false, err
	}

	// ── التدقيق: العرضُ القديمُ والجديد، والفاعل، والدور، والمصدر، والسبب ──
	if err := s.auditQuoteTx(ctx, q, &m.actorID, "order.custom_quote_changed", row.id, map[string]any{
		"old_goods": row.goods, "old_fee": row.fee, "old_total": row.goods + row.fee,
		"new_goods": newGoods, "new_fee": newFee, "new_total": newTotal,
		"quote_version": newVersion, "actor_role": m.role, "source": m.source,
		"reason": m.reason, "invalidated_confirmation": invalidate,
	}); err != nil {
		return false, err
	}
	return true, nil
}

// ConfirmQuote **تأكيدُ الزبون للعرض واختيارُ طريقة الدفع** — Batch 2a.
//
// (قرارُ المالك: العرضُ يُقفَل بتأكيدٍ من الزبون، ثمّ يبدأ الشراء.)
//
// **يُقفَل الصفّ، ويُتحقَّق أنّ العرضَ الحاليَّ هو ما يؤكّده** (نسخةً ومبلغاً)،
// **والمحفظةُ لا تُقبَل إلّا إن غطّى المتاحُ المبلغَ كاملاً — ويُحجَز فوراً.**
// **ومتكرّرٌ بالنسخة والطريقة نفسِها لا شيء** (idempotent).
func (s *Service) ConfirmQuote(ctx context.Context, orderID, customerID, paymentMethod string,
	expectedTotal, expectedVersion int64) (*Order, error) {
	if paymentMethod != "cash" && paymentMethod != "wallet" {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row, err := s.lockCustomRow(ctx, tx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if row.kind != KindCustom {
		return nil, ErrNotCustom
	}
	if row.customerID != customerID {
		return nil, httpx.NewError(http.StatusForbidden, "not_your_order", "errors.not_your_order")
	}
	// **لا تأكيدَ قبل اتفاقٍ، ولا بعد استلام** — العرضُ يُقفَل عند الاستلام.
	if !row.hasGoods {
		return nil, ErrCustomNotAgreed
	}
	if row.pickedUpAt != nil {
		return nil, ErrCustomLocked
	}
	// **العرضُ الذي يؤكّده هو الحاليُّ نفسُه** — نسخةً ومبلغاً، وإلّا فليقرأ الجديد.
	current := row.goods + row.fee
	if expectedVersion != row.quoteVersion || expectedTotal != current {
		return nil, ErrQuoteChanged
	}

	// **تأكيدٌ متكرّرٌ بالنسخة والطريقة نفسِها — لا شيء** (idempotent). **ومحفظةً
	// كان فالحجزُ قائمٌ أصلاً** — يُعاد الطلبُ كما هو (التراجعُ يُطلق القفل).
	if row.confirmedAt != nil && row.confirmedVersion != nil &&
		*row.confirmedVersion == row.quoteVersion && row.paymentMethod == paymentMethod {
		return s.GetByID(ctx, orderID)
	}

	// **وأيُّ حجزٍ سابقٍ يُفكّ قبل التأكيد الجديد** — تبديلُ الطريقة أو إعادةُ
	// التأكيد لا يُبقي حجزاً قديماً معلَّقاً.
	if row.reserved > 0 {
		if err := s.releaseCustomReservationTx(ctx, tx, row.customerID, row.id, row.reserved); err != nil {
			return nil, err
		}
	}

	if paymentMethod == "cash" {
		// **ومن قُفل عليه النقدُ لا يؤكّد نقداً** — الحدُّ نفسُه لبابٍ آخر.
		blocked, err := s.on(tx).cashBlocked(ctx, tx, customerID)
		if err != nil {
			return nil, err
		}
		if blocked {
			return nil, ErrCashBlocked
		}
	} else { // wallet
		avail, err := s.wallet.AvailableTx(ctx, tx, customerID)
		if err != nil {
			return nil, err
		}
		if avail < current {
			return nil, wallet.ErrInsufficient
		}
	}

	// **تُكتب أعمدةُ التأكيد أوّلاً** (الطريقة، اللحظة، المبلغ، النسخة) —
	// **قبل أن يصير الحجزُ موجباً**، فالقيدُ يشترط محفظةً مؤكَّدةً لأيّ حجز.
	if _, err := tx.Exec(ctx, `
		UPDATE orders SET payment_method = $2,
		       quote_confirmed_at = now(), quote_confirmed_total = $3,
		       quote_confirmed_version = $4, updated_at = now()
		WHERE id = $1`, orderID, paymentMethod, current, row.quoteVersion); err != nil {
		return nil, err
	}
	// **ثمّ يُحجَز كاملُ المبلغ فوراً** — فشلٌ مغلق (`reserved <= balance`).
	if paymentMethod == "wallet" {
		if err := s.reserveCustomTx(ctx, tx, customerID, orderID, current); err != nil {
			if errors.Is(err, wallet.ErrReservedTooMuch) {
				return nil, wallet.ErrInsufficient
			}
			return nil, err
		}
	}

	if err := s.auditQuoteTx(ctx, tx, &customerID, "order.custom_quote_confirmed", orderID, map[string]any{
		"total": current, "quote_version": row.quoteVersion,
		"payment_method": paymentMethod, "actor_role": "customer", "source": "customer_confirm",
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	o, err := s.GetByID(ctx, orderID)
	if err == nil {
		s.publishOrder(o)
		s.publishWalletsOf(ctx, orderID)
	}
	return o, err
}

// AdminOverrideCustomQuote **تدخّلُ الأدمن على العرض** — قبل الاستلام حرّاً،
// وبعده نقصاً أو تصحيحاً فقط، **موثَّقاً** (قرارُ المالك) — Batch 2a.
//
// **وبعد التسليم/الانتهاء لا تعديل** — التصحيحُ حينها تعويضٌ بابُه آخر.
func (s *Service) AdminOverrideCustomQuote(ctx context.Context, orderID, adminID string,
	newGoods, newFee int64, reason string) (*Order, error) {
	if newGoods < 0 || newFee < 0 {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row, err := s.lockCustomRow(ctx, tx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if row.kind != KindCustom {
		return nil, ErrNotCustom
	}
	// **بعد الانتهاء (تسليماً أو إلغاءً) لا تعديل** — والتصحيحُ حينها تعويضٌ موثَّق.
	if terminal(row.status) {
		return nil, ErrCustomLocked
	}
	// **وبعد الاستلام لا زيادة** — نقصٌ أو تصحيحٌ فقط (قرارُ المالك).
	if row.pickedUpAt != nil && newGoods+newFee > row.goods+row.fee {
		return nil, ErrCustomLocked
	}
	if _, err := s.applyCustomQuoteTx(ctx, tx, row, newGoods, newFee, quoteMutator{
		actorID: adminID, role: "admin", source: "admin_override", reason: reason,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	o, err := s.GetByID(ctx, orderID)
	if err == nil {
		s.publishOrder(o)
		s.publishWalletsOf(ctx, orderID)
	}
	return o, err
}

// auditQuoteTx **يقيّد تغييرَ العرض في `audit_log`** داخلَ المعاملة نفسِها —
// **فتغييرٌ ارتدّت معاملتُه لا يترك تدقيقاً، ومثبَّتٌ يحمل تدقيقَه دائماً.**
//
// **والـ`ip` فارغٌ هنا** — الفاعلُ ودورُه ومصدرُه ما يُقرأ عند الخلاف، لا موضعُه.
func (s *Service) auditQuoteTx(ctx context.Context, q wallet.Querier, actorID *string,
	action, orderID string, details map[string]any) error {
	_, err := q.Exec(ctx, `
		INSERT INTO audit_log (actor_user_id, action, entity, entity_id, ip, details)
		VALUES ($1, $2, 'order', $3, '', $4)`, actorID, action, orderID, details)
	return err
}
