package orders

// ══════════════════════════════════════════════════════════════════════
// **«تصحيح: الطلب انلغى»** — قرارُ المالك ٢٠٢٦-١٠-٢٨
// ══════════════════════════════════════════════════════════════════════
//
// السائقُ ضغط «تم التسليم» خطأً، والطلبُ أُلغي فعلاً (#1099). **فيصحّحه الأدمن**، والنتيجةُ ما
// كان سيُسجَّل لو أنهى المكتبُ الطلبَ بعد الاستلام بلا تسليم (`settle` §5):
//
//	الطلب     ملغى، وسببُه وحدثُه (delivered → cancelled) ومن صحّحه.
//	السائق    يُعكَس تحصيلُ صندوقه (`collection_reversed`) وأجرُه (`driver_earning` سالباً).
//	المندوب   تُعكَس عمولتُه (`commission` سالباً)، والمحجوزةُ لموقوفٍ تسقط.
//	الدعوة    مكافأةُ أوّل طلبٍ صرفها هذا التسليمُ تُعكَس إن احتملها متاحُ الداعي.
//	المتجر    **يبقى له ما قبضه** — باع وسلّم البضاعة.
//	الزبون    يُردّ له ما دفعه من محفظته — والنقدُ لم يدفعه أصلاً.
//	الدرج     يردّ السلفةَ التي دفع بها السائقُ للمتجر (§5) — وما سلّمه السائقُ للمكتب عن هذا الطلب.
//	الخزينة   `creditTreasury` تُعيد الحسبةَ بعد كلّ ما سبق.
//
// **ويُعكَس ما قيّده الدفترُ لهذا الطلب لا ما تقوله المعادلة.** وما لا يحتمله متاحُ السائق أو
// المندوب دَينٌ عليه (`undelivered_order`) يُقتطع من أوّل مستحقٍّ قادم — **ولا رصيدَ سالب.**
//
// **ولا يمرّ بمسار الاسترجاع (§6)**: ذاك يفترض أنّ الزبونَ دفع النقدَ فيردّ له الطلبَ كلَّه.

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/officecash"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var (
	// ErrUndeliverReasonRequired **السببُ إلزاميّ.**
	ErrUndeliverReasonRequired = httpx.NewError(http.StatusBadRequest,
		"undeliver_reason_required", "errors.undeliver_reason_required")
	// ErrUndeliverAdminOnly **للأدمن وحدَه** — كالاسترجاع.
	ErrUndeliverAdminOnly = httpx.NewError(http.StatusForbidden,
		"undeliver_admin_only", "errors.undeliver_admin_only")
	// ErrUndeliverNotStandard الخاصُّ و«لدي توصيلة» لهما مسارُ مالٍ آخر.
	ErrUndeliverNotStandard = httpx.NewError(http.StatusConflict,
		"undeliver_not_standard", "errors.undeliver_not_standard")
	// ErrUndeliverNotDelivered الطلبُ ليس مُسلَّماً — ومنه يمتنع التكرار.
	ErrUndeliverNotDelivered = httpx.NewError(http.StatusConflict,
		"undeliver_not_delivered", "errors.undeliver_not_delivered")
	// ErrUndeliverCommissionWaived **عمولتُه مُحيت** — صندوقُ السائق والمتجرُ صُحّحا بطريقٍ آخر.
	ErrUndeliverCommissionWaived = httpx.NewError(http.StatusConflict,
		"undeliver_commission_waived", "errors.undeliver_commission_waived")
)

// UndeliverResult ما وقع — للردّ والتدقيق.
type UndeliverResult struct {
	OrderID string `json:"order_id"`
	Number  int64  `json:"number"`
	Status  string `json:"status"`
	// BoxReversed ما نقص من صندوق السائق · BoxHandedOver ما سلّمه للمكتب فيردّه الدرج.
	BoxReversed   int64 `json:"box_reversed"`
	BoxHandedOver int64 `json:"box_handed_over"`
	// DriverReversed · DriverDebt أجرُه: ما عُكس من محفظته وما صار ديناً.
	DriverReversed int64 `json:"driver_reversed"`
	DriverDebt     int64 `json:"driver_debt"`
	// RepReversed · RepDebt · RepHeldForfeited عمولةُ المندوب.
	RepReversed      int64 `json:"rep_reversed"`
	RepDebt          int64 `json:"rep_debt"`
	RepHeldForfeited int64 `json:"rep_held_forfeited"`
	// ReferralReversed مكافأةُ دعوةٍ عُكست · ReferralKept بقيت لأنّ متاحَ الداعي لا يحتملها.
	ReferralReversed int64 `json:"referral_reversed"`
	ReferralKept     int64 `json:"referral_kept"`
	// CustomerRefund ما رُدّ لمحفظة الزبون · DrawerOut ما خرج من الدرج.
	CustomerRefund int64 `json:"customer_refund"`
	DrawerOut      int64 `json:"drawer_out"`
	// DriverID · RepID للإشعار والتدقيق.
	DriverID string `json:"-"`
	RepID    string `json:"-"`
}

// Undeliver **يصحّح طلباً سُجّل تسليمُه خطأً** — في معاملةٍ واحدة، و`hook` (التدقيق) قبل التثبيت
// كـ`TransitionAudited`. انظر رأسَ الملفّ.
func (s *Service) Undeliver(ctx context.Context, actorID string, actorRoles []string,
	orderID, reason string, hook func(context.Context, dbtx.Querier, UndeliverResult) error) (*Order, UndeliverResult, error) {
	var res UndeliverResult
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, res, ErrUndeliverReasonRequired
	}
	if !slices.Contains(actorRoles, "admin") && !slices.Contains(actorRoles, "owner_super_admin") {
		return nil, res, ErrUndeliverAdminOnly
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, res, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if res, err = s.undeliverTx(ctx, tx, actorID, orderID, reason); err != nil {
		return nil, res, err
	}
	if hook != nil {
		if err := hook(ctx, tx, res); err != nil {
			return nil, res, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, res, err
	}

	updated, err := s.GetByID(ctx, orderID)
	if err != nil {
		return updated, res, err
	}
	s.publishOrder(updated)
	s.publishWalletsOf(ctx, orderID)
	return updated, res, nil
}

func (s *Service) undeliverTx(ctx context.Context, q wallet.Querier,
	actorID, orderID, reason string) (UndeliverResult, error) {
	var res UndeliverResult

	// (١) **القفلُ على صفّ الطلب** — نداءان متزامنان: الثاني ينتظر ثمّ يجده ملغياً.
	var status, kind, customerID, promoCode string
	var driverID *string
	var number, walletPaid, waived int64
	err := q.QueryRow(ctx, `
		SELECT number, status, kind, driver_id::text, COALESCE(customer_id::text, ''),
		       COALESCE(promo_code, ''), wallet_paid, commission_waived
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&number, &status, &kind, &driverID, &customerID, &promoCode, &walletPaid, &waived)
	if errors.Is(err, pgx.ErrNoRows) {
		return res, httpx.ErrNotFound
	}
	if err != nil {
		return res, err
	}
	res.OrderID, res.Number = orderID, number
	if kind != KindStandard {
		return res, ErrUndeliverNotStandard
	}
	if status != StDelivered {
		return res, ErrUndeliverNotDelivered
	}
	if waived > 0 {
		return res, ErrUndeliverCommissionWaived
	}

	// (٢) **صندوقُ السائق** — كلُّ تحصيلٍ لهذا الطلب يُعكَس بقدر ما في صندوقه اليوم، **وما سلّمه
	// للمكتب منه يردّه الدرجُ له** (في (٨)) — لم يقبضه من الزبون.
	type collected struct {
		driverID string
		amount   int64
	}
	var cols []collected
	rows, err := q.Query(ctx, `
		SELECT driver_id::text, sum(amount)::bigint FROM driver_cash_entries
		 WHERE ref = $1 AND kind = 'order_collection'
		 GROUP BY driver_id ORDER BY driver_id`, orderID)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var c collected
		if err := rows.Scan(&c.driverID, &c.amount); err != nil {
			rows.Close()
			return res, err
		}
		cols = append(cols, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return res, err
	}
	for _, c := range cols {
		if c.amount <= 0 {
			continue
		}
		held, err := lockedInt(ctx, q,
			`SELECT held FROM driver_cash_boxes WHERE driver_id = $1 FOR UPDATE`, c.driverID)
		if err != nil {
			return res, err
		}
		take := min(held, c.amount)
		if take > 0 {
			if _, err := s.cashbox.ReverseCollectionTx(ctx, q, c.driverID, take, orderID, actorID); err != nil {
				return res, err
			}
		}
		res.BoxReversed += take
		res.BoxHandedOver += c.amount - take
	}

	// (٣) **أجرُ السائق** — الموجبُ المقيَّدُ لهذا الطلب (سطورُ اقتطاع الدَّين ليست نقصاً منه).
	drv, err := s.reverseEarnings(ctx, q, orderID, actorID, "driver_earning", obligations.PartyDriver,
		"إلغاءُ أجرِ توصيلِ طلبٍ سُجّل تسليمُه خطأً")
	if err != nil {
		return res, err
	}
	res.DriverID, res.DriverReversed, res.DriverDebt = drv.userID, drv.taken, drv.debt

	// (٤) **عمولةُ المندوب** — والمحجوزةُ لمندوبٍ موقوفٍ تسقط في مكانها.
	rep, err := s.reverseEarnings(ctx, q, orderID, actorID, "commission", obligations.PartyRep,
		"إلغاءُ عمولةِ طلبٍ سُجّل تسليمُه خطأً")
	if err != nil {
		return res, err
	}
	res.RepID, res.RepReversed, res.RepDebt = rep.userID, rep.taken, rep.debt
	if err := q.QueryRow(ctx, `
		WITH f AS (
			UPDATE rep_held_commissions SET status = 'forfeited', decided_at = now(), decided_by = $2
			 WHERE order_id = $1 AND status = 'held' RETURNING amount)
		SELECT COALESCE(sum(amount), 0)::bigint FROM f`,
		orderID, actorID).Scan(&res.RepHeldForfeited); err != nil {
		return res, err
	}

	// (٥) **مكافأةُ الدعوة** — إن صرفها هذا التسليمُ (مرجعُها رقمُ الطلب).
	if err := s.reverseReferral(ctx, q, orderID, customerID, actorID, &res); err != nil {
		return res, err
	}

	// (٦) **الطلب** — ملغى، وسببُه وحدثُه. **ولا يُعدّ مُسلَّماً بعد اليوم** (`delivered_at` يُمحى).
	if _, err := q.Exec(ctx, `
		UPDATE orders
		   SET status = $2, cancel_reason = $3, delivered_at = NULL, closed_at = now(),
		       ended_by = 'admin', updated_at = now()
		 WHERE id = $1`, orderID, StCancelled, reason); err != nil {
		return res, err
	}
	var actor any
	if actorID != "" {
		actor = actorID
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $3, $4, $5, $6)`, orderID, StDelivered, StCancelled, actor, reason, driverID); err != nil {
		return res, err
	}
	// **وكودُ الخصم يتحرّر كما في كلّ إلغاء** (`transitionTx`).
	if promoCode != "" {
		if _, err := q.Exec(ctx,
			`UPDATE promo_codes SET used_count = greatest(used_count - 1, 0) WHERE code = $1`, promoCode); err != nil {
			return res, err
		}
		if _, err := q.Exec(ctx, `DELETE FROM promo_redemptions WHERE order_id = $1`, orderID); err != nil {
			return res, err
		}
	}

	// (٧) **الزبون** — ما دفعه من محفظته يُردّ (§1). والنقدُ لم يدفعه.
	if walletPaid > 0 && customerID != "" {
		if _, err := s.wallet.ApplyTx(ctx, q, customerID, walletPaid, "refund", orderID,
			"استرجاعُ المدفوع من المحفظة — طلبٌ أُلغي بعد تسجيل تسليمه خطأً", &actorID); err != nil {
			return res, err
		}
		res.CustomerRefund = walletPaid
	}

	// (٨) **الدرج** — §5 يردّ السلفةَ كلَّها (ما دفعه السائقون للمتاجر)، والتسليمُ ردّ منها ما لم
	// يغطّه نقدُ الزبون. **فيُردّ الفرقُ، ومعه ما سلّمه السائقُ للمكتب من تحصيلٍ لم يقع.**
	if driverID != nil {
		all, _, err := driverPaidFor(ctx, q, orderID, *driverID)
		if err != nil {
			return res, err
		}
		var returned int64
		if err := q.QueryRow(ctx, `
			SELECT COALESCE(sum(amount), 0)::bigint FROM office_cash_entries
			 WHERE ref = $1 AND source IN ($2, $3)`,
			orderID, officecash.SourceAdvanceReturn, officecash.SourceUndeliveredReturn).Scan(&returned); err != nil {
			return res, err
		}
		if out := all + res.BoxHandedOver - returned; out > 0 {
			if err := officecash.Record(ctx, q, officecash.Entry{
				Direction: officecash.Out, Amount: out, Source: officecash.SourceUndeliveredReturn,
				Ref: orderID, UserID: *driverID, Actor: actorID,
				Note: "ردُّ ما دفعه السائقُ للمتجر — طلبٌ أُلغي بعد تسجيل تسليمه خطأً",
			}); err != nil {
				return res, err
			}
			res.DrawerOut = out
		}
	}

	// (٩) **الخزينة** — بعد كلّ ما سبق، فتقرأ ما بقي مقيَّداً.
	if err := s.creditTreasury(ctx, q, orderID, actorID); err != nil {
		return res, err
	}
	res.Status = StCancelled
	return res, nil
}

type reversedEarning struct {
	userID      string
	taken, debt int64
}

// reverseEarnings **يعكس ما قُيّد لطرفٍ عن هذا الطلب** — بالنوع نفسِه سالباً (فتراه الخزينةُ وكلُّ
// حسبةٍ تجمع بالنوع)، **وما لا يحتمله متاحُه دَينٌ عليه** (`undelivered_order`) — كعكس مستحقّ
// المتجر في `reverseCommissions` ونصيبِ المندوب في `reverseRepForWaiver`.
func (s *Service) reverseEarnings(ctx context.Context, q wallet.Querier, orderID, actorID,
	kind, party, note string) (reversedEarning, error) {
	var out reversedEarning
	rows, err := q.Query(ctx, `
		SELECT user_id::text, sum(amount)::bigint FROM wallet_transactions
		 WHERE ref = $1 AND kind = $2 AND amount > 0
		 GROUP BY user_id ORDER BY user_id`, orderID, kind)
	if err != nil {
		return out, err
	}
	type paid struct {
		userID string
		amount int64
	}
	var list []paid
	for rows.Next() {
		var p paid
		if err := rows.Scan(&p.userID, &p.amount); err != nil {
			rows.Close()
			return out, err
		}
		list = append(list, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	for _, p := range list {
		out.userID = p.userID
		available, err := lockedInt(ctx, q,
			`SELECT balance - reserved FROM wallets WHERE user_id = $1 FOR UPDATE`, p.userID)
		if err != nil {
			return out, err
		}
		take := max(min(p.amount, available), 0)
		if take > 0 {
			if _, err := s.wallet.ApplyTx(ctx, q, p.userID, -take, kind, orderID, note, &actorID); err != nil {
				return out, err
			}
		}
		if debt := p.amount - take; debt > 0 {
			if _, err := obligations.Create(ctx, q, party, p.userID, debt,
				obligations.CauseUndeliveredOrder, orderID, &actorID); err != nil {
				return out, err
			}
			out.debt += debt
		}
		out.taken += take
	}
	return out, nil
}

// reverseReferral **مكافأةُ أوّلِ طلبٍ صرفها هذا التسليم** — تُعكَس إلى الخزينة وتعود الدعوةُ
// معلّقةً، فيُصرف لها أوّلُ طلبٍ يُسلَّم فعلاً. **وإن لم يحتملها متاحُ الداعي** (سحبها) تبقى له
// والدعوةُ مختومة — الداعي زبونٌ لا دَينَ عليه في المنصّة، والطلبُ التالي لا يصرفها مرّةً ثانية.
func (s *Service) reverseReferral(ctx context.Context, q wallet.Querier,
	orderID, customerID, actorID string, res *UndeliverResult) error {
	if customerID == "" {
		return nil
	}
	var inviterID string
	var amount int64
	err := q.QueryRow(ctx, `
		SELECT r.inviter_id::text, t.amount FROM referrals r
		JOIN wallet_transactions t ON t.user_id = r.inviter_id AND t.ref = $1
		                          AND t.kind = 'reward' AND t.amount > 0
		WHERE r.invitee_id = $2 AND r.rewarded_at IS NOT NULL
		ORDER BY t.id LIMIT 1
		FOR UPDATE OF r`, orderID, customerID).Scan(&inviterID, &amount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	available, err := lockedInt(ctx, q,
		`SELECT balance - reserved FROM wallets WHERE user_id = $1 FOR UPDATE`, inviterID)
	if err != nil {
		return err
	}
	if available < amount {
		res.ReferralKept = amount
		return nil
	}
	if _, err := s.wallet.ApplyTx(ctx, q, inviterID, -amount, "reward", orderID,
		"إلغاءُ مكافأةِ دعوةٍ — الطلبُ أُلغي بعد تسجيل تسليمه خطأً", &actorID); err != nil {
		return err
	}
	if tid := s.treasuryOn(ctx, q); tid != "" {
		if _, err := s.wallet.ApplyTx(ctx, q, tid, amount, "reward", orderID,
			"مكافأةُ دعوةٍ عادت إلى الخزينة", &actorID); err != nil {
			return err
		}
	}
	if _, err := q.Exec(ctx, `
		UPDATE referrals SET rewarded_at = NULL, reward_amount = 0 WHERE invitee_id = $1`,
		customerID); err != nil {
		return err
	}
	res.ReferralReversed = amount
	return nil
}

// lockedInt **رقمٌ يُقرأ ويُقفل صفُّه** — وغيابُ الصفّ صفر (صندوقٌ أو محفظةٌ لم تُنشأ بعد).
func lockedInt(ctx context.Context, q wallet.Querier, sql string, arg any) (int64, error) {
	var v int64
	err := q.QueryRow(ctx, sql, arg).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return v, err
}
