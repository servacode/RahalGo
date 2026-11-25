package orders

// ══════════════════════════════════════════════════════════════════════
// **إكراميةُ الكابتن** — قرارُ المالك ٢٠٢٦-١١-٢٥ (هجرة ٠٥٥٩)
// ══════════════════════════════════════════════════════════════════════
//
// الزبونُ اللي بيدفع من محفظته (`full` أو `delivery`) بيقدر يزيد إكرامية — اختياريّة دايماً.
//
//	الاختيار   **بتنحجز** من محفظته (`ReserveTx`) — `tip_reserved` — متل حجز الطلب الخاصّ
//	التسليم    الحجزُ بيتسوّى: سطرُ `tip` سالب عالزبون، وموجبٌ للكابتن اللي سلّم، ونسبةُ
//	           المنصّة (لقطة `tip_platform_percent`) للخزينة — **ومجموعُ السطور صفر**، فلا
//	           `creditTreasury` بتحسبها ولا حفظُ الطلب (FI-06.a · FI-17.e) بيتغيّر
//	الإلغاء    الحجزُ بينفكّ — ما في ولا سطر بالدفتر
//	تبديلُ الكابتن  ما بيهمّ: بتنقيّد لمين بيسلّم فعلاً
//	تصحيحُ تسليمٍ خاطئ  بتنعكس السطور التلاتة (`reverseTipTx`)
//
// **ومصدرُ الحقيقة السطور لا العمود** — والثوابتُ (FI-03.f · FI-04.g · FI-05.l · FI-11.n)
// بتقيس إنّها ما انقيّدت مرّتين ولا ضاعت.

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

const (
	// KeyTipsEnabled · KeyTipsPlatformPercent — بـ`settings/catalog.go`.
	KeyTipsEnabled         = "tips.enabled"
	KeyTipsPlatformPercent = "tips.platform_percent"
	// MaxTip **حارسٌ تقنيٌّ فقط** — **الإكراميةُ حسب ما بدّه الزبون بلا سقف** (قرارُ المالك ٢٠٢٦-١١-٢٥:
	// «الاكرامية حسب مابدو مانقدر نحط حد»)؛ والحدُّ الفعليُّ رصيدُ محفظته (`insufficient_balance`).
	MaxTip int64 = 1_000_000_000
	// tipKind نوعُ القيد بالدفتر.
	tipKind = "tip"
)

// TipPresets **أزرارُ الإكرامية الجاهزة** (قرارُ المالك ٢٠٢٦-١١-٢٥) — و«مبلغ تاني» حرّ حتّى `MaxTip`.
var TipPresets = []int64{5, 10, 20}

var (
	// ErrTipInvalid **مبلغٌ خارج المدى.**
	ErrTipInvalid = httpx.NewError(http.StatusBadRequest, "tip_invalid", "errors.tip_invalid")
	// ErrTipNotAllowed **الإكراميةُ مطفأة، أو الدفعُ مو من المحفظة، أو طلبٌ ما بينفع** (واتساب،
	// توصيلةُ متجر، منتهٍ، أو انقيّدت خلص).
	ErrTipNotAllowed = httpx.NewError(http.StatusConflict, "tip_not_allowed", "errors.tip_not_allowed")
)

// TipsEnabled **المفتاحُ الرئيسيّ** — للعرض العامّ (`/public/platform`).
func (s *Service) TipsEnabled(ctx context.Context) bool {
	return s.settings != nil && s.settings.GetBool(ctx, KeyTipsEnabled)
}

// tipNotice **إكراميةٌ انقيّدت لكابتن** — يُخبَر بعد الإيداع.
type tipNotice struct {
	driverID string
	amount   int64
	number   int64
}

// setTipTx **يثبّت إكراميةَ الطلب على مبلغٍ جديد** — داخل معاملة المستدعي.
//
// الصفرُ يفكّ الحجزَ دايماً (ولو انطفت الإكرامية). والموجبُ بيشترط: المفتاحَ شغّال، طلباً
// عاديّاً (مو واتساب) أو خاصّاً، دفعاً من المحفظة، طلباً حيّاً ما انقيّدت إكراميتُه، ومتاحاً
// يغطّيه. **ونفسُ المبلغ لا شي.**
func (s *Service) setTipTx(ctx context.Context, q wallet.Querier, orderID, customerID string, amount int64) error {
	if amount < 0 || amount > MaxTip {
		return ErrTipInvalid
	}
	var kind, origin, status, method, scope, owner string
	var cur, reserved int64
	var paidAt *time.Time
	var payShamcash bool
	err := q.QueryRow(ctx, `
		SELECT kind, origin, status, payment_method, wallet_scope, COALESCE(customer_id::text, ''),
		       tip_amount, tip_reserved, tip_paid_at, pay_shamcash
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&kind, &origin, &status, &method, &scope, &owner, &cur, &reserved, &paidAt, &payShamcash)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if owner != customerID {
		return httpx.NewError(http.StatusForbidden, "not_your_order", "errors.not_your_order")
	}
	if amount == cur && (amount == 0 || reserved == amount) {
		return nil
	}
	if paidAt != nil || terminal(status) {
		return ErrTipNotAllowed
	}
	if amount > 0 {
		allowedKind := (kind == KindStandard && origin != "whatsapp") || kind == KindCustom
		// **وشام كاش عند الاستلام متل الكاش** (٢٠٢٦-١١-٢٥) — ما إلو إكرامية.
		if !s.TipsEnabled(ctx) || !allowedKind || method != "wallet" || payShamcash ||
			(scope != ScopeFull && scope != ScopeDelivery) {
			return ErrTipNotAllowed
		}
	}
	if reserved > 0 {
		if err := s.wallet.ReleaseTx(ctx, q, customerID, reserved); err != nil {
			return err
		}
	}
	pct := int64(0)
	if amount > 0 {
		avail, err := s.wallet.AvailableTx(ctx, q, customerID)
		if err != nil {
			return err
		}
		if avail < amount {
			return wallet.ErrInsufficient
		}
		if err := s.wallet.ReserveTx(ctx, q, customerID, amount); err != nil {
			if errors.Is(err, wallet.ErrReservedTooMuch) {
				return wallet.ErrInsufficient
			}
			return err
		}
		if s.settings != nil {
			pct = min(max(s.settings.On(q).GetInt(ctx, KeyTipsPlatformPercent), 0), 100)
		}
	}
	if _, err := q.Exec(ctx, `
		UPDATE orders SET tip_amount = $2, tip_reserved = $2, tip_platform_percent = $3, updated_at = now()
		WHERE id = $1`, orderID, amount, pct); err != nil {
		return err
	}
	return s.auditQuoteTx(ctx, q, &customerID, "order.tip_set", orderID, map[string]any{
		"old_tip": cur, "new_tip": amount, "platform_percent": pct, "status": status,
		"actor_role": "customer",
	})
}

// releaseTipTx **نهايةٌ لغير التسليم تفكّ حجزَ الإكرامية** — والطلبُ بلا حجزٍ ما بيلمس شي.
func (s *Service) releaseTipTx(ctx context.Context, q wallet.Querier, orderID string) error {
	var customerID string
	var reserved int64
	err := q.QueryRow(ctx, `
		SELECT COALESCE(customer_id::text, ''), tip_reserved FROM orders WHERE id = $1`, orderID).
		Scan(&customerID, &reserved)
	if errors.Is(err, pgx.ErrNoRows) || reserved <= 0 || customerID == "" {
		return nil
	}
	if err != nil {
		return err
	}
	if err := s.wallet.ReleaseTx(ctx, q, customerID, reserved); err != nil {
		return err
	}
	_, err = q.Exec(ctx, `UPDATE orders SET tip_reserved = 0, updated_at = now() WHERE id = $1`, orderID)
	return err
}

// tipNote **«إكرامية الطلب #N»** — نصُّ السطر بالدفتر للطرفين.
func tipNote(number int64) string { return fmt.Sprintf("إكرامية الطلب #%d", number) }

// settleTipTx **التسليمُ يقيّد الإكرامية** — للكابتن اللي سلّم فعلاً (`driverID` الانتقال).
//
// **مرّة وحدة**: الحجزُ يصير صفر و`tip_paid_at` ينكتب بنفس المعاملة، والصفُّ مقفول من
// الانتقال — فتسليمٌ تاني ما بيلاقي شي. **وبلا خزينة** نسبةُ المنصّة للكابتن (المجموع صفر دايماً).
func (s *Service) settleTipTx(ctx context.Context, q wallet.Querier, orderID string, driverID *string,
	actorID string, out *settled) error {
	var customerID string
	var number, reserved, pct int64
	var paidAt *time.Time
	err := q.QueryRow(ctx, `
		SELECT COALESCE(customer_id::text, ''), number, tip_reserved, tip_platform_percent, tip_paid_at
		FROM orders WHERE id = $1`, orderID).Scan(&customerID, &number, &reserved, &pct, &paidAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if reserved <= 0 || paidAt != nil || customerID == "" {
		return nil
	}
	// **بلا كابتن ما في مين ياخدها** — بترجع للزبون.
	if driverID == nil || *driverID == "" {
		return s.releaseTipTx(ctx, q, orderID)
	}
	var by *string
	if actorID != "" {
		by = &actorID
	}
	note := tipNote(number)
	if _, err := s.wallet.SettleReservedTx(ctx, q, customerID, reserved, tipKind, orderID, note, by); err != nil {
		return err
	}
	cut := reserved * pct / 100
	tid := s.treasuryOn(ctx, q)
	if tid == "" {
		cut = 0
	}
	share := reserved - cut
	if share > 0 {
		if _, err := s.wallet.ApplyTx(ctx, q, *driverID, share, tipKind, orderID, note, by); err != nil {
			return err
		}
	}
	if cut > 0 {
		if _, err := s.wallet.ApplyTx(ctx, q, tid, cut, tipKind, orderID,
			"نسبة المنصة من "+note, by); err != nil {
			return err
		}
	}
	if _, err := q.Exec(ctx, `
		UPDATE orders SET tip_reserved = 0, tip_platform_cut = $2, tip_paid_at = now(), updated_at = now()
		WHERE id = $1`, orderID, cut); err != nil {
		return err
	}
	if out != nil && share > 0 {
		out.tip = &tipNotice{driverID: *driverID, amount: share, number: number}
	}
	return nil
}

// reverseTipTx **تصحيحُ تسليمٍ سُجّل خطأً** — الإكراميةُ ترجع كاملةً للزبون.
//
// الخزينةُ بتردّ نسبتها، والكابتنُ بيردّ حصّته بقدّ متاحه — **والباقي بتتحمّله الخزينة**
// (مبالغُ صغيرة وتصحيحٌ بإيد الأدمن). **والمجموعُ يبقى صفر**، و`tip_paid_at` بينمحى.
func (s *Service) reverseTipTx(ctx context.Context, q wallet.Querier, orderID, actorID string) (int64, error) {
	var customerID string
	var number, amount int64
	var paidAt *time.Time
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(customer_id::text, ''), number, tip_amount, tip_paid_at
		FROM orders WHERE id = $1`, orderID).Scan(&customerID, &number, &amount, &paidAt); err != nil {
		return 0, err
	}
	if paidAt == nil || customerID == "" {
		return 0, nil
	}
	// **ما انقيّد فعلاً** — من الدفتر لا من العمود.
	type line struct {
		userID string
		amount int64
	}
	rows, err := q.Query(ctx, `
		SELECT user_id::text, sum(amount)::bigint FROM wallet_transactions
		 WHERE ref = $1 AND kind = $2 GROUP BY user_id ORDER BY user_id`, orderID, tipKind)
	if err != nil {
		return 0, err
	}
	var lines []line
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.userID, &l.amount); err != nil {
			rows.Close()
			return 0, err
		}
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	tid := s.treasuryOn(ctx, q)
	note := "إلغاء " + tipNote(number) + " — طلبٌ أُلغي بعد تسجيل تسليمه خطأً"
	var refund, shortfall int64
	for _, l := range lines {
		switch {
		case l.userID == customerID:
			refund = -l.amount
		case l.amount > 0 && l.userID != tid:
			available, err := lockedInt(ctx, q,
				`SELECT balance - reserved FROM wallets WHERE user_id = $1 FOR UPDATE`, l.userID)
			if err != nil {
				return 0, err
			}
			take := max(min(l.amount, available), 0)
			if take > 0 {
				if _, err := s.wallet.ApplyTx(ctx, q, l.userID, -take, tipKind, orderID, note, &actorID); err != nil {
					return 0, err
				}
			}
			shortfall += l.amount - take
		}
	}
	var treasuryBack int64
	for _, l := range lines {
		if l.userID == tid && l.userID != customerID {
			treasuryBack = l.amount
		}
	}
	if back := treasuryBack + shortfall; back != 0 {
		if tid == "" {
			return 0, errors.New("tip reversal: shortfall without a treasury")
		}
		if _, err := s.wallet.ApplyTx(ctx, q, tid, -back, tipKind, orderID, note, &actorID); err != nil {
			return 0, err
		}
	}
	if refund > 0 {
		if _, err := s.wallet.ApplyTx(ctx, q, customerID, refund, tipKind, orderID, note, &actorID); err != nil {
			return 0, err
		}
	}
	if _, err := q.Exec(ctx, `
		UPDATE orders SET tip_paid_at = NULL, tip_platform_cut = 0, tip_amount = 0, updated_at = now()
		WHERE id = $1`, orderID); err != nil {
		return 0, err
	}
	return refund, nil
}

// notifyTip **الكابتن يُخبَر بإكراميته** بعد الإيداع — ويرنّ (مو متل أجره الصامت).
func (s *Service) notifyTip(ctx context.Context, orderID string, n *tipNotice) {
	if s.notify == nil || n == nil || n.driverID == "" || n.amount <= 0 {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: n.driverID, Kind: notifications.KindWallet,
		Title:  "وصلتك إكرامية",
		Body:   fmt.Sprintf("إكرامية %s ل.س من زبون الطلب #%d — بمحفظتك", groupDigits(n.amount), n.number),
		Entity: "wallet", EntityID: orderID, Href: "/portal/wallet",
		Apps: []string{notifications.AppDriver},
	})
}
