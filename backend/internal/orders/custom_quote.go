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
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
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
	waived             int64 // promo_delivery_waived — التوصيلُ المجانيّ (٢٠٢٦-١٠-٢٣)
	hasGoods           bool  // أاتُّفق مرّةً على الأقلّ؟
	hasFee             bool  // أوُثّقت الأجرةُ عموداً؟ — شرطُ خطوة الثمن
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
		       custom_driver_may_change_fee, picked_up_at, promo_delivery_waived
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&r.customerID, &r.driverID, &r.kind, &r.status, &r.paymentMethod,
			&goods, &fee, &r.quoteVersion,
			&r.confirmedAt, &r.confirmedTotal, &r.confirmedVersion,
			&r.reserved, &r.feeSource, &r.feeSnapshot,
			&r.driverMayChangeFee, &r.pickedUpAt, &r.waived)
	if err != nil {
		return customQuoteRow{}, err
	}
	if goods != nil {
		r.goods, r.hasGoods = *goods, true
	}
	if fee != nil {
		r.fee, r.hasFee = *fee, true
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
	// **والتوصيلُ المجانيّ بعرضه** (قرارُ المالك ٢٠٢٦-١٠-٢٣) — يُحسب قبل المجموع: الزبونُ
	// يؤكّد ما يدفعه فعلاً. **و`custom_fee` يبقى الأجرةَ المتّفقَ عليها** (أجرُ السائق
	// منها)، و`delivery_fee` ما يدفعه الزبون، والفرقُ في `promo_delivery_waived`.
	var mode string
	if err := q.QueryRow(ctx, `SELECT custom_mode FROM orders WHERE id = $1`, row.id).Scan(&mode); err != nil {
		return false, err
	}
	freeKind := FreeKindPurchase
	if mode == CustomModeAmanah {
		freeKind = FreeKindAmanah
	}
	waive, err := s.freeDeliveryForQuote(ctx, q, row.id, row.customerID, freeKind, newFee)
	if err != nil {
		return false, err
	}
	newTotal := newGoods + newFee - waive
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
	// ══════════════════════════════════════════════════════════════════
	// **وبعد الاستلام: الزيادةُ تنتظر تأكيداً ولا تمسّ التأكيدَ القائم** (قرارُ المالك ٢٠٢٦-١٠-١٥)
	// ══════════════════════════════════════════════════════════════════
	//
	// **البضاعةُ في يد السائق** — فلا يُفكّ حجزٌ ولا يُمحى تأكيد: **المؤكَّدُ القديمُ
	// يبقى عقداً قائماً بمحجوزه**، والنسخةُ الجديدةُ تسبقه فيُغلق التسليمُ حتّى يؤكّدها
	// الزبون (`ErrQuoteReconfirmPending`). **ومن فكّ الحجزَ هنا ترك الزبونَ يصرف
	// رصيدَه والبضاعةُ في الطريق إليه.** والنقصُ يقع فوراً كما قبل الاستلام.
	reconfirm := false
	if invalidate && row.pickedUpAt != nil {
		invalidate, reconfirm = false, true
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
	// **و`driver_fee = custom_fee` أيضاً**: الطلبُ الخاصُّ لا عرضَ فيه يُصفّر
	// دفعَ الزبون، **فأجرُ السائقِ هو الأجرةُ المتّفقُ عليها نفسُها** — والسائقُ
	// يُدفَع من `driver_fee` (٢٠٢٦-٠٩-٢٧). فلولا هذا لقاد بأجرٍ صفرٍ (افتراضِ العمود).
	// ══════════════════════════════════════════════════════════════════
	// **ونصيبُ المنصّةِ من الأجرةِ وحدَها** — قرارُ المالك النهائيُّ ٢٠٢٦-٠٩-٢٩
	// ══════════════════════════════════════════════════════════════════
	//
	// **كان `driver_fee = custom_fee`** — الأجرةَ كاملةً، **والمنصّةُ تعبر
	// بلا أن تأخذ** (قرارُ المالك ٢٠٢٦-٠٨-٠٩). **ثمّ شهد المالكُ خزينتَه
	// صفراً ونقض قرارَه**: «خزينة المنصة فارغة مازالت صفر».
	//
	// # والبضاعةُ ردٌّ لا كسب
	//
	// **`goods_amount` لا تُؤخذ منه نسبةٌ البتّة** (نصُّ المالك: «لا تأخذ
	// نسبة المنصة من goods_amount»): **السائقُ دفع ثمنَها من جيبه**،
	// **فأخذُ نسبةٍ منه اقتطاعٌ من رأس ماله لا من ربحه** — ويجعله يخسر
	// بكلّ طلب.
	//
	// **والأجرةُ وحدَها ربحٌ** — فهي وحدَها تُقسَم.
	//
	// # واللقطةُ هنا لا عند التسوية
	//
	// **تُثبَّت لحظةَ تثبيت الاتّفاق** (نصُّ المالك: «snapshot … وقت تثبيت
	// الاتفاق، بحيث تغيير النسبة لاحقاً لا يؤثر على أي طلب قديم»):
	// **ولو قُرئ الإعدادُ وقتَ التسوية لتبدّل مالُ طلبٍ اتُّفق عليه أمس**،
	// **ولاختلف ما وُعد به السائقُ عمّا قُبض.**
	//
	// **ويُعاد الحسابُ مع كلّ تعديلِ اتّفاق** — فالأجرةُ تتبدّل، **ونصيبٌ
	// محسوبٌ على أجرةٍ قديمةٍ يكذب.**
	// **ولكلّ نوعٍ حصّتُه** (٢٠٢٦-١٠-١٣) — الأمانةُ غيرُ المشتريات.
	shareKind := SharePurchase
	if mode == CustomModeAmanah {
		shareKind = ShareAmanah
	}
	pct, fixed := s.platformShare(ctx, q, shareKind)
	driverFee := DriverFeeWithShare(newFee, pct, fixed)

	set := `custom_goods_amount = $2, custom_fee = $3,
	        subtotal = $2, delivery_fee = $3::bigint - $8::bigint, driver_fee = $5,
	        promo_delivery_waived = $8,
	        snap_platform_delivery_percent = $6, snap_platform_delivery_fixed = $7,
	        total = $2::bigint + $3::bigint - $8::bigint,
	        quote_version = $4, custom_agreed_at = now(), updated_at = now(),
	        quote_rejected_at = NULL`
	// **والثابتان يُضافان قبل الشرطيّ** — **ورقمُ وسيطٍ يتزحزح بحسب فرعٍ
	// يُنفَّذ أو لا يُنفَّذ يكتب قيمةً في عمودٍ آخر بصمت.**
	args := []any{row.id, newGoods, newFee, newVersion, driverFee, pct, fixed, waive}
	switch {
	case invalidate:
		set += `, quote_confirmed_at = NULL, quote_confirmed_total = NULL, quote_confirmed_version = NULL`
	case reconfirm:
		// **التأكيدُ القديمُ باقٍ بمبلغه ونسخته** — والنسخةُ الجديدةُ تنتظر.
	case row.confirmedAt != nil: // نقصٌ مع تأكيدٍ باقٍ: يُحدَّث المبلغُ والنسخة
		// **والمستحقُّ النقديُّ يتبع المبلغ** — كان يبقى على المؤكَّد القديم فيقبض
		// السائقُ ما لم يعُد مستحقّاً. (والمحفظةُ صفرٌ فيها، فلا يُمَسّ.)
		set += `, quote_confirmed_total = $9, quote_confirmed_version = $4,
		        cash_due = CASE WHEN payment_method = 'cash' THEN $9 ELSE cash_due END`
		args = append(args, newTotal)
	}
	if _, err := q.Exec(ctx, `UPDATE orders SET `+set+` WHERE id = $1`, args...); err != nil {
		return false, err
	}

	// ── التدقيق: العرضُ القديمُ والجديد، والفاعل، والدور، والمصدر، والسبب ──
	if err := s.auditQuoteTx(ctx, q, &m.actorID, "order.custom_quote_changed", row.id, map[string]any{
		"old_goods": row.goods, "old_fee": row.fee, "old_total": row.goods + row.fee - row.waived,
		"new_goods": newGoods, "new_fee": newFee, "new_total": newTotal,
		"quote_version": newVersion, "actor_role": m.role, "source": m.source,
		"reason": m.reason, "invalidated_confirmation": invalidate,
		"reconfirm_pending": reconfirm,
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
	return s.confirmQuote(ctx, orderID, customerID, paymentMethod, expectedTotal, expectedVersion, false)
}

// confirmQuote **والصامتُ منه** لأجرة المنصّة (`autoAgreeFixedFee`) — لا سطرَ «وافقتُ»
// ولا إشعارَ للسائق: **هو من أخذ الطلبَ للتوّ.**
func (s *Service) confirmQuote(ctx context.Context, orderID, customerID, paymentMethod string,
	expectedTotal, expectedVersion int64, quiet bool) (*Order, error) {
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
	// **لا تأكيدَ قبل اتفاقٍ، ولا بعد نهاية.**
	if !row.hasGoods {
		return nil, ErrCustomNotAgreed
	}
	if terminal(row.status) {
		return nil, ErrCustomLocked
	}
	// **العرضُ الذي يؤكّده هو الحاليُّ نفسُه** — نسخةً ومبلغاً، وإلّا فليقرأ الجديد.
	current := row.goods + row.fee - row.waived
	if expectedVersion != row.quoteVersion || expectedTotal != current {
		return nil, ErrQuoteChanged
	}

	// **تأكيدٌ متكرّرٌ بالنسخة والطريقة نفسِها — لا شيء** (idempotent). **ومحفظةً
	// كان فالحجزُ قائمٌ أصلاً** — يُعاد الطلبُ كما هو (التراجعُ يُطلق القفل).
	if row.confirmedAt != nil && row.confirmedVersion != nil &&
		*row.confirmedVersion == row.quoteVersion && row.paymentMethod == paymentMethod {
		return s.GetByID(ctx, orderID)
	}
	// ══════════════════════════════════════════════════════════════════
	// **وبعد الاستلام يُؤكَّد السعرُ الجديدُ وحدَه** (قرارُ المالك ٢٠٢٦-١٠-١٥)
	// ══════════════════════════════════════════════════════════════════
	//
	// **المكتبُ رفع السعرَ والبضاعةُ في الطريق** — فيؤكّده الزبونُ هنا، **ولا شيءَ
	// غيرُه يُفتح بعد الاستلام**: عرضٌ مؤكَّدٌ لا يُبدَّل دفعُه في الطريق.
	reconfirm := row.pickedUpAt != nil
	if reconfirm && row.confirmedVersion != nil && *row.confirmedVersion == row.quoteVersion {
		return nil, ErrCustomLocked
	}

	// **وأيُّ حجزٍ سابقٍ يُفكّ قبل التأكيد الجديد** — تبديلُ الطريقة أو إعادةُ
	// التأكيد لا يُبقي حجزاً قديماً معلَّقاً.
	if row.reserved > 0 {
		if err := s.releaseCustomReservationTx(ctx, tx, row.customerID, row.id, row.reserved); err != nil {
			return nil, err
		}
	}

	// **وإعادةُ التأكيد نقداً على طلبٍ نقديٍّ في الطريق لا تُسأل عن قفل النقد وسقفه**:
	// **الطلبُ قائمٌ نقداً والبضاعةُ اشتُريت** — ومن رُدّ هنا بقيت بضاعتُه بيد السائق
	// لا تُسلَّم ولا تُردّ. والسقفان حارسا طلبٍ جديدٍ لا تعديلِ طلبٍ في الطريق.
	if paymentMethod == "cash" && !(reconfirm && row.paymentMethod == "cash") {
		// **ومن قُفل عليه النقدُ لا يؤكّد نقداً** — الحدُّ نفسُه لبابٍ آخر.
		blocked, err := s.on(tx).cashBlocked(ctx, tx, customerID)
		if err != nil {
			return nil, err
		}
		if blocked {
			return nil, ErrCashBlocked
		}
		// **وسقفُ النقد غيرِ المسدَّدِ بذمّة الزبون** (`COD`) — عند التأكيد
		// لا عند الإنشاء: **السعرُ لا يُعرف قبل عرض السائق**، وهنا صار
		// `current` هو المتّفَقَ عليه. **ويُعقد قفلُ `customer-admit:` نفسُه**
		// الذي يعقده إنشاءُ الطلب العاديّ، فلا يسابق تأكيدٌ خاصٌّ إنشاءً
		// عاديّاً على المتّسع (`COD-07`)؛ والصفُّ مقفولٌ (`FOR UPDATE`) فلا
		// تأكيدان لطلبٍ واحد. **والطلبُ القائمُ يُستثنى من المجموع** بمعرّفه
		// فلا يُحسب مرّتين، **وقبل كتابة أعمدة التأكيد** فالردُّ لا يخلّف أثراً.
		if _, err := tx.Exec(ctx,
			`SELECT pg_advisory_xact_lock(hashtext($1))`,
			"customer-admit:"+customerID); err != nil {
			return nil, err
		}
		if err := s.on(tx).checkCODLimit(ctx, tx, customerID, orderID, current); err != nil {
			return nil, err
		}
	} else if paymentMethod == "wallet" {
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
	// ══════════════════════════════════════════════════════════════════
	// **والنقدُ يُكتب مستحقّاً على الطلب** — وإلّا بقي فراغاً ماليّاً
	// ══════════════════════════════════════════════════════════════════
	//
	// **كان `cash_due` يبقى صفراً للخاصِّ النقديِّ أبداً** — والمبلغُ لا
	// يُعرف إلّا بعد الاتّفاق، **ولم يُكتب حين عُرف.**
	//
	// **فكان الخاصُّ النقديُّ فراغاً ماليّاً كاملاً** (قُيس ٢٠٢٦-٠٩-٢٩):
	// لا صندوقَ نقدٍ للسائق، ولا قيدَ في الدفتر، ولا نصيبَ للمنصّة —
	// **والسائقُ يقبض المبلغَ ولا أثرَ له في المنصّة إطلاقاً.**
	//
	// **و`cash_due` هو ما تقرؤه المنصّةُ كلُّها**: سقفُ نقدِ السائق في
	// الأهليّة، وصندوقُه عند التسليم، و`creditTreasury` في الدفتر.
	// **فبصفرِه لا يُحرَس سقفٌ ولا يُقيَّد قبض.**
	//
	// **ويُصفَّر إن كان الدفعُ بالمحفظة** — تبديلُ الطريقة يقع
	// (`payment_method = $2`)، **ومستحقٌّ نقديٌّ باقٍ على طلبٍ صار
	// محفظيّاً يجعل السائقَ يقبض مرّةً والمحفظةَ تُخصم مرّةً.**
	cashDue := int64(0)
	if paymentMethod == "cash" {
		cashDue = current
	}
	if _, err := tx.Exec(ctx, `
		UPDATE orders SET payment_method = $2,
		       quote_confirmed_at = now(), quote_confirmed_total = $3,
		       quote_confirmed_version = $4, cash_due = $5, updated_at = now(),
		       quote_rejected_at = NULL
		WHERE id = $1`, orderID, paymentMethod, current, row.quoteVersion, cashDue); err != nil {
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
		"after_pickup": reconfirm,
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
		// ══════════════════════════════════════════════════════════════
		// **والدورُ صار على السائق — فيُخبَر أنّ الدورَ عليه**
		// ══════════════════════════════════════════════════════════════
		//
		// **وكان البثُّ وحدَه**، وهو يصل شاشةً مفتوحةً لا جيباً مغلقاً:
		// **فالسائقُ ينتظر التأكيدَ ولا يعلم أنّ انتظارَه انتهى**، والزبونُ
		// يرى «بانتظار الشراء» ولا أحدَ يشتري.
		if o.DriverID != nil && !quiet {
			s.notifyCustomConfirmed(ctx, orderID, *o.DriverID, o.Total, reconfirm)
			// **وتأكيدُ الزبون يُقال في الحديث أيضاً** (٢٠٢٦-١٠-٠٣) — بمجموعه.
			line := fmt.Sprintf("وافقتُ على المجموع: %s ل.س.", groupDigits(o.Total))
			if o.CustomGoodsPending || o.CustomMode == CustomModeAmanah {
				line = fmt.Sprintf("وافقتُ على أجرة التوصيل: %s ل.س.", groupDigits(o.Total))
			}
			s.quoteLine(ctx, orderID, customerID, "customer", line)
		}
	}
	return o, err
}

// AdminOverrideCustomQuote **تدخّلُ الأدمن على العرض** — صعوداً ونزولاً حتّى التسليم،
// **موثَّقاً بسبب** (قرارُ المالك ٢٠٢٦-١٠-١٥، ناقضاً «بعد الاستلام نقصاً فقط»).
//
// # الحادثةُ التي فتحته
//
// **وثّق السائقُ ثمنَ البضاعة ٥٠ بدل ٥٠٠ ثمّ استلم** — فلم يملك المكتبُ تصحيحَه:
// الزيادةُ بعد الاستلام كانت تُردّ (`ErrCustomLocked`)، **والمنصّةُ تخسر الفرقَ كلَّه.**
//
// # والقاعدةُ الآن
//
//	نقصٌ      ←  يقع فوراً، ويُخبَر الزبونُ بالمجموع الجديد
//	زيادةٌ    ←  قبل الاستلام كما كانت (يُعاد التأكيد)، **وبعده يبقى المؤكَّدُ القديمُ
//	             قائماً والتسليمُ موقوفٌ حتّى يؤكّد الزبونُ الجديد** (`ErrQuoteReconfirmPending`)
//
// **والسائقُ لا يعدّل بعد الاستلام** — بابُه `AgreeCustom` ما زال مقفلاً.
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
	changed, err := s.applyCustomQuoteTx(ctx, tx, row, newGoods, newFee, quoteMutator{
		actorID: adminID, role: "admin", source: "admin_override", reason: reason,
	})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	o, err := s.GetByID(ctx, orderID)
	if err == nil {
		s.publishOrder(o)
		s.publishWalletsOf(ctx, orderID)
		if changed {
			s.notifyCustomPriceChanged(ctx, o, reason, row.pickedUpAt != nil)
		}
	}
	return o, err
}

// QuotePending **أينتظر العرضُ الحاليُّ تأكيدَ الزبون؟** — نسخةٌ لم تُؤكَّد بعد.
func (o *Order) QuotePending() bool {
	return o.CustomGoodsAmount != nil &&
		(o.QuoteConfirmedVersion == nil || *o.QuoteConfirmedVersion != o.QuoteVersion)
}

// notifyCustomPriceChanged **الزبونُ يُخبَر بما عدّله المكتبُ وسببِه** (قرارُ المالك ٢٠٢٦-١٠-١٥).
//
// **وإشعارٌ باقٍ لا عابر**: سعرٌ تغيّر في طلبه — نقصاً يقرؤه، وزيادةً يُطلب منه
// تأكيدُها. **والسائقُ يُخبَر بعد الاستلام** أنّ التسليمَ موقوفٌ حتّى يؤكّد الزبون.
func (s *Service) notifyCustomPriceChanged(ctx context.Context, o *Order, reason string, afterPickup bool) {
	if s.notify == nil || o == nil || o.CustomerID == "" {
		return
	}
	pending := o.QuotePending()
	title := t.customPriceLowered
	if pending {
		title = t.customPriceRaised
	}
	body := fmt.Sprintf("#%d — المجموع الجديد %s ل.س — السبب: %s", o.Number, groupDigits(o.Total), reason)
	s.notify.Notify(ctx, notifications.Input{
		UserID: o.CustomerID, Kind: notifications.KindOrder,
		Title: title, Body: body,
		Entity: "order", EntityID: o.ID, Href: "/portal/orders",
		Apps: []string{notifications.AppCustomer},
	})
	if pending && afterPickup && o.DriverID != nil && *o.DriverID != "" {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *o.DriverID, Kind: notifications.KindOrder,
			Title:  t.customPriceRaisedDriver,
			Body:   fmt.Sprintf("#%d — المجموع الجديد %s ل.س", o.Number, groupDigits(o.Total)),
			Entity: "order", EntityID: o.ID, Href: "/portal",
			Apps: []string{notifications.AppDriver},
		})
	}
}

// RejectQuote **الزبونُ يرفض السعرَ الجديد** (قرارُ المالك ٢٠٢٦-١٠-١٥).
//
// **والطلبُ يبقى حيث هو** — لا إلغاءَ ولا فكَّ حجز: البضاعةُ قد تكون في يد السائق،
// **والمكتبُ يقرّر** (يعدّل ثانيةً أو يُنهي). **فتُكتب علامةٌ يراها المكتب**
// (`quote_rejected_at`) ويُنذَر بها.
func (s *Service) RejectQuote(ctx context.Context, orderID, customerID string) (*Order, error) {
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
	if terminal(row.status) {
		return nil, ErrCustomLocked
	}
	if !row.hasGoods || (row.confirmedVersion != nil && *row.confirmedVersion == row.quoteVersion) {
		return nil, ErrQuoteNothingPending
	}
	if _, err := tx.Exec(ctx, `
		UPDATE orders SET quote_rejected_at = COALESCE(quote_rejected_at, now()),
		       updated_at = now() WHERE id = $1`, orderID); err != nil {
		return nil, err
	}
	if err := s.auditQuoteTx(ctx, tx, &customerID, "order.custom_quote_rejected", orderID, map[string]any{
		"total": row.goods + row.fee - row.waived, "quote_version": row.quoteVersion,
		"actor_role": "customer", "source": "customer_reject",
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	o, err := s.GetByID(ctx, orderID)
	if err == nil {
		s.publishOrder(o)
		if s.notify != nil {
			s.notify.NotifyOps(ctx, notifications.Input{
				Kind:   notifications.KindOrder,
				Title:  t.customQuoteRejected,
				Body:   fmt.Sprintf("#%d — %s ل.س", o.Number, groupDigits(o.Total)),
				Entity: "order", EntityID: orderID, Href: "/dashboard/orders",
			})
			// **والسائقُ يُخبَر أيضاً** (قرارُ المالك ٢٠٢٦-١٠-٢١) — كان المكتبُ وحدَه يعلم،
			// والسائقُ ينتظر تأكيداً لن يأتي. **فيتواصل مع الزبون ويعدّل ويُرسل ثانيةً.**
			if o.DriverID != nil {
				s.notify.Notify(ctx, notifications.Input{
					UserID: *o.DriverID, Kind: notifications.KindOrder,
					Title:  t.customQuoteRejectedDriver,
					Body:   fmt.Sprintf("#%d — %s ل.س", o.Number, groupDigits(o.Total)),
					Entity: "order", EntityID: orderID, Href: "/portal",
					Apps: []string{notifications.AppDriver},
				})
			}
		}
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
