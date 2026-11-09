package orders

// ══════════════════════════════════════════════════════════════════════
// **شو بيندفع من المحفظة** — قرارُ المالك ٢٠٢٦-١١-٠٩ (هجرة ٠٥٢٦)
// ══════════════════════════════════════════════════════════════════════
//
// الزبونُ يختار واحداً من ثلاثة:
//
//	none      كاش كلّه                       payment_method = cash
//	full      «الكل من المحفظة»              payment_method = wallet
//	delivery  «التوصيل من المحفظة والباقي كاش» payment_method = wallet
//
// **و`payment_method` يبقى كما هو** — التطبيقاتُ القديمةُ تقرأ `== "wallet"`. والعمودُ
// الجديدُ (`wallet_scope`) يقول **قدّيش** من المحفظة، **ومصدرُ الرقم دالّةٌ واحدة**:
// `WalletPortion`. لا حسبةٌ ثانيةٌ في موضعٍ ثانٍ.
//
// # والطلبُ العاديّ يُخصم عند الإنشاء
//
// كما كان: **الحصّةُ المختارةُ تُخصم لحظةَ الإنشاء** (`order_payment`)، والباقي `cash_due`
// يقبضه الكابتن. **ويتبدّل الاختيارُ حتّى يوصل الكابتن للباب** (`ChangeWalletScope`):
// الفرقُ يُخصم (`order_payment`) أو يُردّ (`refund`) — **المسارُ نفسُه لا مسارٌ ثانٍ.**
//
// # والطلبُ الخاصُّ يُحجز عند التأكيد
//
// **الحصّةُ تُحجز لا تُخصم** (`ReserveTx`)، والبضاعةُ `cash_due`، **والتسليمُ يسوّي
// الحجزَ ويقبض النقدَ معاً** (`settleCustomWallet`). **ولا يُكتب `wallet_paid` قبل
// الخصم الحقيقيّ** — الخزينةُ تقرأ العمودَ حين لا قيد.

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// نطاقاتُ الدفع من المحفظة — قيمُ `orders.wallet_scope`.
const (
	ScopeNone     = "none"
	ScopeFull     = "full"
	ScopeDelivery = "delivery"
)

var (
	// ErrWalletScopeLocked **الكابتن وصل للباب** — الاختيارُ صار مقفولاً.
	ErrWalletScopeLocked = httpx.NewError(http.StatusConflict,
		"wallet_scope_locked", "errors.wallet_scope_locked")
	// ErrOrderChanged **تغيّر مجموعُ الطلب تحت يد الزبون** — يقرأ الجديدَ ويختار ثانيةً.
	ErrOrderChanged = httpx.NewError(http.StatusConflict,
		"order_changed", "errors.order_changed")
	// ErrWalletScopeNotAllowed **طلبٌ لا يُدفع من المحفظة** (واتساب، توصيلةُ متجر) أو
	// خاصٌّ لم يُؤكَّد سعرُه بعد.
	ErrWalletScopeNotAllowed = httpx.NewError(http.StatusConflict,
		"wallet_scope_not_allowed", "errors.wallet_scope_not_allowed")
)

// WalletPortion **قدّيش بيندفع من المحفظة** — الدالّةُ الواحدةُ لكلّ مسار.
//
// `deliveryFee` **أجرةُ التوصيل المدفوعةُ فعلاً** بعد كود الخصم والتوصيل المجانيّ —
// وهي عمودُ `orders.delivery_fee` في العاديّ والخاصّ (`custom_fee − promo_delivery_waived`).
// **ولا تتجاوز المجموع** (خصمٌ يُنزل المجموعَ تحت الأجرة).
func WalletPortion(scope string, total, deliveryFee int64) int64 {
	if total < 0 {
		total = 0
	}
	switch scope {
	case ScopeFull:
		return total
	case ScopeDelivery:
		p := deliveryFee
		if p > total {
			p = total
		}
		if p < 0 {
			p = 0
		}
		return p
	}
	return 0
}

// normalizePayment **يوحّد الطريقةَ والنطاق** — والفارغان يعنيان ما كانا يعنيانه.
//
//	("", "") · ("cash", "")          ⇒ cash · none
//	("wallet", "")                   ⇒ wallet · full   (التطبيقُ القديم)
//	("", "full") · ("wallet","full") ⇒ wallet · full
//	("", "delivery") · …             ⇒ wallet · delivery
//
// **والمتناقضان يُردّان** (`cash` مع `full`).
func normalizePayment(method, scope string) (string, string, error) {
	switch scope {
	case "":
		switch method {
		case "", "cash":
			return "cash", ScopeNone, nil
		case "wallet":
			return "wallet", ScopeFull, nil
		}
	case ScopeNone:
		if method == "" || method == "cash" {
			return "cash", ScopeNone, nil
		}
	case ScopeFull, ScopeDelivery:
		if method == "" || method == "wallet" {
			return "wallet", scope, nil
		}
	}
	return "", "", ErrBadPayment
}

// walletScopeOpen **أيتبدّل الاختيارُ بعد؟** — حتّى يوصل الكابتن للباب.
func walletScopeOpen(status string) bool {
	return !terminal(status) && status != StAtDropoff
}

// customPaymentTx **يثبّت دفعَ الطلب الخاصّ على الحجز** — من التأكيد ومن تبديل الاختيار.
//
// يفكّ الحجزَ القائم، **ويفحص حارسَي النقد إن طُلب** (قفلُ النقد وسقفُ COD على ما سيُقبض
// نقداً)، ويتحقّق أنّ المتاحَ يغطّي حصّةَ المحفظة، ثمّ يكتب الطريقةَ والنطاقَ والنقد
// **ويحجز الحصّة.** **ولا يكتب `wallet_paid`** — يُكتب عند التسوية بعد الخصم الحقيقيّ.
//
// **والمستدعي كتب التأكيدَ قبله** (`quote_confirmed_at`) — القيدُ يشترطه لأيّ حجز.
func (s *Service) customPaymentTx(ctx context.Context, tx dbtx.Querier, row customQuoteRow,
	method, scope string, current int64, cashGuards bool) (int64, error) {
	portion := WalletPortion(scope, current, row.fee-row.waived)
	cashDue := current - portion

	if row.reserved > 0 {
		if err := s.releaseCustomReservationTx(ctx, tx, row.customerID, row.id, row.reserved); err != nil {
			return 0, err
		}
	}
	if cashGuards && cashDue > 0 {
		if err := s.cashGuardsTx(ctx, tx, row.customerID, row.id, cashDue); err != nil {
			return 0, err
		}
	}
	if portion > 0 {
		avail, err := s.wallet.AvailableTx(ctx, tx, row.customerID)
		if err != nil {
			return 0, err
		}
		if avail < portion {
			return 0, wallet.ErrInsufficient
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE orders SET payment_method = $2, wallet_scope = $3, cash_due = $4, updated_at = now()
		WHERE id = $1`, row.id, method, scope, cashDue); err != nil {
		return 0, err
	}
	if portion > 0 {
		if err := s.reserveCustomTx(ctx, tx, row.customerID, row.id, portion); err != nil {
			if errors.Is(err, wallet.ErrReservedTooMuch) {
				return 0, wallet.ErrInsufficient
			}
			return 0, err
		}
	}
	return cashDue, nil
}

// cashGuardsTx **حارسا النقد على الزبون** — قفلُ النقد (`cashBlocked`) وسقفُ ما بذمّته
// (`checkCODLimit`) على ما سيُقبض نقداً من هذا الطلب. **وتحت قفل `customer-admit:`**
// الذي يعقده الإنشاءُ والتأكيد — فلا يسابق أحدُهما الآخرَ على المتّسع.
func (s *Service) cashGuardsTx(ctx context.Context, tx dbtx.Querier, customerID, orderID string, cashDue int64) error {
	unit := s.on(tx)
	blocked, err := unit.cashBlocked(ctx, tx, customerID)
	if err != nil {
		return err
	}
	if blocked {
		return ErrCashBlocked
	}
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1))`, "customer-admit:"+customerID); err != nil {
		return err
	}
	return unit.checkCODLimit(ctx, tx, customerID, orderID, cashDue)
}

// ChangeWalletScope **الزبونُ يبدّل شو بيندفع من المحفظة** — حتّى يوصل الكابتن للباب.
//
// يُقفل الصفّ (`FOR UPDATE`)، **ويُطابق المجموعَ الذي رآه الزبون** (`expectedTotal`)،
// **ونفسُ الاختيار لا شيء** (idempotent). **وكلُّ زيادةِ نقدٍ تمرّ بحرّاسها**: قفلُ النقد
// وسقفُ COD، **وسقفُ نقدِ الكابتن** إن كان مُسنَداً (`AdmitDriverTx` على الفرق وحدَه —
// نقدُ الطلب الحاليُّ محسوبٌ في تعرّضه أصلاً).
//
//	العاديّ  الفرقُ يُخصم (`order_payment`) أو يُردّ (`refund`) الآن — كالإنشاء والتعديل
//	الخاصّ   يُفكّ الحجزُ ويُحجز الجديد (`customPaymentTx`) — ويشترط سعراً مؤكَّداً حاليّاً
//
// **وبعد الاستلام تُعاد حسبةُ الخزينة** (`creditTreasury`) — قيدُ ردٍّ بعد المحاسبة يكسر
// FI-06.a لولاها. **ويُنشر الطلبُ ويُخبَر الكابتن** — نقدُه تبدّل.
func (s *Service) ChangeWalletScope(ctx context.Context, orderID, customerID, scope string,
	expectedTotal int64) (*Order, error) {
	// **السقفُ يُقرأ قبل فتح المعاملة** (`XG-46`) — قراءتُه داخلها تطلب اتّصالاً ثانياً.
	limit := s.DriverCashLimit(ctx)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	o, after, err := s.ChangeWalletScopeTx(ctx, tx, orderID, customerID, scope, expectedTotal, limit)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	after()
	return o, nil
}

// DriverCashLimit **سقفُ نقد الكابتن** — يُقرأ قبل فتح المعاملة (`XG-46`).
func (s *Service) DriverCashLimit(ctx context.Context) int64 {
	if s.cashbox == nil {
		return 0
	}
	return s.cashbox.Limit(ctx)
}

// ChangeWalletScopeTx **التبديلُ داخل معاملة المستدعي** — بابُ الخادم يلفّه بـ`WithIdempotentTx`
// فتُثبَّت علامةُ منع التكرار مع المال في معاملةٍ واحدة (`XG-33`). **و`after` يُنادى بعد الإيداع**:
// نشرُ الطلب وإخبارُ الكابتن.
func (s *Service) ChangeWalletScopeTx(ctx context.Context, tx dbtx.Querier, orderID, customerID, scope string,
	expectedTotal, cashLimit int64) (*Order, func(), error) {
	noop := func() {}
	if scope == "" {
		return nil, noop, ErrBadPayment
	}
	method, scope, err := normalizePayment("", scope)
	if err != nil {
		return nil, noop, err
	}

	var o struct {
		kind, status, customerID, origin, scope, method string
		total, deliveryFee, walletPaid, cashDue         int64
		driverID                                        *string
	}
	err = tx.QueryRow(ctx, `
		SELECT kind, status, COALESCE(customer_id::text, ''), origin, wallet_scope, payment_method,
		       total, delivery_fee, wallet_paid, cash_due, driver_id::text
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&o.kind, &o.status, &o.customerID, &o.origin, &o.scope, &o.method,
			&o.total, &o.deliveryFee, &o.walletPaid, &o.cashDue, &o.driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, noop, httpx.ErrNotFound
	}
	if err != nil {
		return nil, noop, err
	}
	if o.customerID != customerID {
		return nil, noop, httpx.NewError(http.StatusForbidden, "not_your_order", "errors.not_your_order")
	}
	if !(o.kind == KindStandard && o.origin != "whatsapp") && o.kind != KindCustom {
		return nil, noop, ErrWalletScopeNotAllowed
	}
	if !walletScopeOpen(o.status) {
		return nil, noop, ErrWalletScopeLocked
	}
	if expectedTotal != o.total {
		return nil, noop, ErrOrderChanged
	}
	if o.scope == scope {
		same, err := s.getByID(ctx, tx, orderID)
		return same, noop, err
	}

	var newCash, charged, refunded int64
	if o.kind == KindCustom {
		row, err := s.lockCustomRow(ctx, tx, orderID)
		if err != nil {
			return nil, noop, err
		}
		// **الاختيارُ على سعرٍ مؤكَّدٍ حاليّ** — قبله يختار من شاشة التأكيد نفسِها.
		if row.confirmedAt == nil || row.confirmedVersion == nil || *row.confirmedVersion != row.quoteVersion {
			return nil, noop, ErrWalletScopeNotAllowed
		}
		current := row.goods + row.fee - row.waived
		newCash = current - WalletPortion(scope, current, row.fee-row.waived)
		if newCash > o.cashDue {
			if err := s.admitDriverCashTx(ctx, tx, o.driverID, newCash-o.cashDue, cashLimit); err != nil {
				return nil, noop, err
			}
		}
		if _, err := s.customPaymentTx(ctx, tx, row, method, scope, current, newCash > o.cashDue); err != nil {
			return nil, noop, err
		}
	} else {
		target := WalletPortion(scope, o.total, o.deliveryFee)
		newCash = o.total - target
		if newCash > o.cashDue {
			if err := s.cashGuardsTx(ctx, tx, customerID, orderID, newCash); err != nil {
				return nil, noop, err
			}
			if err := s.admitDriverCashTx(ctx, tx, o.driverID, newCash-o.cashDue, cashLimit); err != nil {
				return nil, noop, err
			}
		}
		note := "تبديل الدفع من المحفظة"
		switch diff := target - o.walletPaid; {
		case diff > 0:
			avail, err := s.wallet.AvailableTx(ctx, tx, customerID)
			if err != nil {
				return nil, noop, err
			}
			if avail < diff {
				return nil, noop, wallet.ErrInsufficient
			}
			if _, err := s.wallet.ApplyTx(ctx, tx, customerID, -diff, "order_payment",
				orderID, note, &customerID); err != nil {
				return nil, noop, err
			}
			charged = diff
		case diff < 0:
			if _, err := s.wallet.ApplyTx(ctx, tx, customerID, -diff, "refund",
				orderID, note, &customerID); err != nil {
				return nil, noop, err
			}
			refunded = -diff
		}
		if _, err := tx.Exec(ctx, `
			UPDATE orders SET payment_method = $2, wallet_scope = $3, wallet_paid = $4, cash_due = $5,
			       updated_at = now()
			WHERE id = $1`, orderID, method, scope, target, newCash); err != nil {
			return nil, noop, err
		}
		if pastPickup[o.status] {
			if err := s.creditTreasury(ctx, tx, orderID, customerID); err != nil {
				return nil, noop, err
			}
		}
	}

	if err := s.auditQuoteTx(ctx, tx, &customerID, "order.wallet_scope_changed", orderID, map[string]any{
		"old_scope": o.scope, "new_scope": scope, "status": o.status,
		"old_cash_due": o.cashDue, "new_cash_due": newCash,
		"charged": charged, "refunded": refunded, "actor_role": "customer",
	}); err != nil {
		return nil, noop, err
	}
	out, err := s.getByID(ctx, tx, orderID)
	if err != nil {
		return nil, noop, err
	}
	return out, func() {
		s.publishOrder(out)
		s.publishWalletsOf(ctx, orderID)
		s.notifyWalletScopeDriver(ctx, out)
	}, nil
}

// admitDriverCashTx **سقفُ نقدِ الكابتن على الزيادة وحدَها** — نقدُ الطلب الحاليُّ في
// تعرّضه أصلاً (`cashbox.Exposure`). وبلا كابتنٍ لا شيء: يُفحص حين يُسنَد.
func (s *Service) admitDriverCashTx(ctx context.Context, tx dbtx.Querier, driverID *string, incoming, limit int64) error {
	if driverID == nil || *driverID == "" || incoming <= 0 {
		return nil
	}
	return s.AdmitDriverTx(ctx, tx, *driverID, Admission{CashDue: incoming, CashLimit: limit, ActiveMax: -1})
}

// notifyWalletScopeDriver **الكابتن يُخبَر أنّ ما يقبضه تبدّل** — بالرقم الجديد.
func (s *Service) notifyWalletScopeDriver(ctx context.Context, o *Order) {
	if s.notify == nil || o == nil || o.DriverID == nil || *o.DriverID == "" {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: *o.DriverID, Kind: notifications.KindOrder,
		Title:  t.walletScopeDriver,
		Body:   fmt.Sprintf("#%d — اقبض %s ل.س", o.Number, groupDigits(o.CashDue)),
		Entity: "order", EntityID: o.ID, Href: "/portal",
		Apps: []string{notifications.AppDriver},
	})
}
