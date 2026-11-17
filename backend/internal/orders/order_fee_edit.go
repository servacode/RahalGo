package orders

// ══════════════════════════════════════════════════════════════════════
// **«تعديل الأجرة» — بيد المكتب على كلّ طلبٍ لحتّى التسليم** (قرارُ المالك ٢٠٢٦-١١-١٨)
// ══════════════════════════════════════════════════════════════════════
//
// المكتبُ يكتب **الأجرةَ الجديدةَ قبل أيّ إعفاء** (ما يقرؤه المكتبُ «أجرة التوصيل»)، وسبباً
// إجباريّاً، **ويرى قبل التأكيد** ما يصير (`DryRun` — يُحسب كلُّه في الخادم ثمّ يُطوى):
// أجرُ السائق، وحصّةُ المنصّة، والمجموعُ الجديدُ على الدافع، وما يُردّ أو يُخصم من محفظته.
//
// # لماذا يكفي صفُّ الطلب قبل التسليم
//
// **أجرُ السائق وحصّةُ المنصّة لا يُقيَّدان إلّا عند التسليم** (`payDriver` يقرأ `driver_fee`،
// و`creditTreasury` تُخرج البقيّةَ من الوقائع). فتعديلُ الأجرة قبله يكتب `driver_fee` الجديدَ
// بلقطة حصّة الطلب (`DriverFeeWithShare`) **ولا قيدَ يُعكس** — إلّا ما دفعه الدافعُ مسبقاً من
// محفظته: يُردّ فرقُه أو يُخصم. **وبعد التسليم لا تعديل** — بابُه «تعويض السائق».
//
// # ولكلّ نوعٍ مسارُه
//
//	العاديّ      ⇐ الكودُ والتوصيلُ المجانيُّ يُعاد تطبيقُهما على الأجرة الجديدة، والمحفظةُ بحصّتها
//	              (`WalletPortion`): نقصٌ يُردّ، وزيادةٌ تُخصم بما يكفي الرصيدُ والباقي نقد.
//	              **ورسمُ المصدر الإضافيّ** (`sources_fee`) يبقى جزءاً منها لا يزيد عليها — فحذفُ
//	              متجرٍ بعدها (`EditOrder`) يُنزل الأجرةَ به ولا يُنزلها تحت الصفر.
//	الخاصّ       ⇐ **بابُ «تعديل العرض» نفسُه** (`adminOverrideCustomTx`) والثمنُ كما هو — فإعادةُ
//	              التأكيد والحجزُ وحصّةُ المنصّة بقواعده هو، لا نسخةٌ ثانية.
//	التوصيلة     ⇐ المتجرُ «محفظة»: بعد القبول يُردّ له الفرقُ أو يُخصم (ولا يكفي ⇒ يُردّ التعديل).
//	              «أنا نقداً» قبل الاستلام والمستلِمُ عند التسليم: يتبدّل ما يقبضه السائق.
//	              **ونقدٌ قُبض فعلاً** (عند الاستلام) **يُردّ تعديلُه** — صندوقُ السائق قبض المبلغَ
//	              القديم، وتصحيحُه بعد التسليم بابُه التعويض.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

var (
	// ErrFeeEditRange **أجرةٌ لا يبقى منها للسائق شيء** — حصّةُ المنصّة تأكلها، أو رقمٌ خارجَ المعقول.
	ErrFeeEditRange = httpx.NewError(http.StatusBadRequest,
		"fee_edit_range", "errors.fee_edit_range")
	// ErrFeeEditCashCollected **الأجرةُ قُبضت نقداً عند الاستلام** — لا يُعدَّل ما في صندوق السائق.
	ErrFeeEditCashCollected = httpx.NewError(http.StatusConflict,
		"fee_edit_cash_collected", "errors.fee_edit_cash_collected")
	// ErrFeeEditMerchantWallet **محفظةُ المتجر لا تكفي الزيادة** — كما يُردّ القبولُ نفسُه.
	ErrFeeEditMerchantWallet = httpx.NewError(http.StatusPaymentRequired,
		"fee_edit_merchant_wallet", "errors.fee_edit_merchant_wallet")
	// ErrFeeEditNotAgreed **طلبٌ خاصٌّ لم تُوثَّق أجرتُه بعد** — الكابتنُ يوثّقها أوّلاً.
	ErrFeeEditNotAgreed = httpx.NewError(http.StatusConflict,
		"fee_edit_not_agreed", "errors.fee_edit_not_agreed")
)

// maxEditFee **سقفُ ما يُكتب** — حارسُ خطأ إصبع لا قرارُ تسعير.
const maxEditFee = 1_000_000_000

// FeePayerCustomerSide **دافعُ الأجرة في العاديّ والخاصّ** — وفي التوصيلة `fee_payer` نفسُه
// (`merchant` · `merchant_cash` · `recipient`).
const FeePayerCustomerSide = "customer"

// EditFeeRequest **الأجرةُ الجديدة قبل أيّ إعفاء، والسبب** — والمعاينةُ لا تترك أثراً.
type EditFeeRequest struct {
	Fee    int64  `json:"fee"`
	Reason string `json:"reason"`
	DryRun bool   `json:"dry_run"`
}

// EditFeeResult **ما يقع** — يُعرض قبل التأكيد ويُكتب في التدقيق.
type EditFeeResult struct {
	Kind  string `json:"kind"`
	Payer string `json:"payer"`
	// OldFee/NewFee **الأجرةُ قبل الإعفاء** — ما يقسمه السائقُ والمنصّة.
	OldFee int64 `json:"old_fee"`
	NewFee int64 `json:"new_fee"`
	// OldDriverFee/DriverFee **أجرُ السائق** عند التسليم.
	OldDriverFee int64 `json:"old_driver_fee"`
	DriverFee    int64 `json:"driver_fee"`
	// OldPlatformShare/PlatformShare **حصّةُ المنصّة من الأجرة.**
	OldPlatformShare int64 `json:"old_platform_share"`
	PlatformShare    int64 `json:"platform_share"`
	// Waived **ما أُعفي منه الدافعُ** (كودٌ أو توصيلٌ مجانيّ) — والخزينةُ تتحمّله.
	Waived     int64 `json:"waived"`
	OldTotal   int64 `json:"old_total"`
	NewTotal   int64 `json:"new_total"`
	OldCashDue int64 `json:"old_cash_due"`
	CashDue    int64 `json:"cash_due"`
	// Refunded/Charged **ما رُدّ لمحفظة الدافع أو خُصم منها** (الزبون أو المتجر).
	Refunded int64 `json:"refunded"`
	Charged  int64 `json:"charged"`
	// Reconfirm **الخاصّ: الزبونُ يؤكّد المجموعَ الجديد** قبل الشراء أو التسليم.
	Reconfirm   bool   `json:"reconfirm"`
	AfterPickup bool   `json:"after_pickup"`
	Reason      string `json:"reason"`
	DryRun      bool   `json:"dry_run"`
}

// feeOrderRow **صفُّ الطلب مقفولاً** — ما يلزم كلَّ المسارات.
type feeOrderRow struct {
	kind, status, customerID, merchantID, promoCode, scope string
	payer, mode, origin                                    string
	number                                                 int64
	subtotal, deliveryFee, discount, total, walletPaid     int64
	cashDue, waived, sourcesFee, driverFee                 int64
	sharePct, shareFixed                                   int64
	driverID                                               *string
}

// EditDeliveryFee **يعدّل أجرةَ التوصيل أو يعاينها** — ذرّيّاً، لكلّ أنواع الطلب.
//
// `hook` **أثرُ التدخّل في المعاملة نفسِها** (التدقيق) — ولا يُنادى في المعاينة.
func (s *Service) EditDeliveryFee(ctx context.Context, actorID, orderID string, req EditFeeRequest,
	hook func(context.Context, dbtx.Querier, EditFeeResult) error) (*Order, EditFeeResult, error) {
	var res EditFeeResult
	res.DryRun = req.DryRun
	req.Reason = strings.TrimSpace(req.Reason)
	res.Reason = req.Reason
	if req.Reason == "" {
		return nil, res, ErrOrderEditReason
	}
	if utf8.RuneCountInString(req.Reason) > 500 {
		return nil, res, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if req.Fee < 1 || req.Fee > maxEditFee {
		return nil, res, ErrFeeEditRange
	}
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, res, httpx.ErrNotFound
	}
	res.NewFee = req.Fee

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, res, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var o feeOrderRow
	err = tx.QueryRow(ctx, `
		SELECT kind, status, COALESCE(customer_id::text, ''), COALESCE(merchant_id::text, ''),
		       COALESCE(promo_code, ''), wallet_scope, COALESCE(fee_payer, ''), delivery_mode, origin,
		       number, subtotal, delivery_fee, discount, total, wallet_paid, cash_due,
		       promo_delivery_waived, sources_fee, driver_fee,
		       snap_platform_delivery_percent, snap_platform_delivery_fixed, driver_id::text
		  FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&o.kind, &o.status, &o.customerID, &o.merchantID, &o.promoCode, &o.scope, &o.payer,
			&o.mode, &o.origin, &o.number, &o.subtotal, &o.deliveryFee, &o.discount, &o.total,
			&o.walletPaid, &o.cashDue, &o.waived, &o.sourcesFee, &o.driverFee,
			&o.sharePct, &o.shareFixed, &o.driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, res, httpx.ErrNotFound
	}
	if err != nil {
		return nil, res, err
	}
	// **لحتّى التسليم وحدَه** — وبعده «تعويض السائق».
	if terminal(o.status) {
		return nil, res, ErrOrderEditClosed
	}
	res.Kind = o.kind
	res.AfterPickup = pastPickup[o.status]
	res.OldTotal, res.OldCashDue = o.total, o.cashDue
	res.OldDriverFee = o.driverFee

	note := fmt.Sprintf("تعديل أجرة التوصيل: %s", req.Reason)
	var merchantOwner string
	switch o.kind {
	case KindStandard:
		err = s.feeEditStandardTx(ctx, tx, actorID, orderID, o, req, note, &res)
	case KindCustom:
		err = s.feeEditCustomTx(ctx, tx, actorID, orderID, req, &res)
	case KindMerchantDelivery:
		merchantOwner, err = s.feeEditMerchantDeliveryTx(ctx, tx, actorID, orderID, o, req, note, &res)
	default:
		err = ErrOrderEditKind
	}
	if err != nil {
		return nil, res, err
	}

	// ── ما صار: من الصفّ نفسِه لا من الحساب — فالمعاينةُ هي ما يقع ────────
	var quotePending bool
	if err := tx.QueryRow(ctx, `
		SELECT total, cash_due, driver_fee, promo_delivery_waived,
		       kind = 'custom' AND custom_goods_amount IS NOT NULL
		         AND (quote_confirmed_version IS NULL OR quote_confirmed_version <> quote_version)
		  FROM orders WHERE id = $1`, orderID).
		Scan(&res.NewTotal, &res.CashDue, &res.DriverFee, &res.Waived, &quotePending); err != nil {
		return nil, res, err
	}
	res.Reconfirm = quotePending
	// **ولا يُرسَل سائقٌ بلا أجر** — حصّةٌ ثابتةٌ تأكل الأجرةَ كلَّها تُردّ.
	if res.DriverFee <= 0 {
		return nil, res, ErrFeeEditRange
	}
	res.PlatformShare = res.NewFee - res.DriverFee
	res.OldPlatformShare = res.OldFee - res.OldDriverFee

	if err := s.feeEditTrailTx(ctx, tx, actorID, orderID, o, merchantOwner, res); err != nil {
		return nil, res, err
	}
	if hook != nil && !req.DryRun {
		if err := hook(ctx, tx, res); err != nil {
			return nil, res, err
		}
	}
	// **والمعاينةُ تُحسب كلُّها ثمّ تُطوى** — الرقمُ الذي يراه المكتبُ هو ما سيقع بالضبط.
	if req.DryRun {
		return nil, res, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, res, err
	}
	updated, err := s.GetByID(ctx, orderID)
	if err != nil {
		return nil, res, err
	}
	s.publishOrder(updated)
	s.publishWalletsOf(ctx, orderID)
	// **والخاصُّ يُخبَر زبونُه بإشعار «تعديل العرض» نفسِه** — نقصاً يقرؤه وزيادةً يؤكّدها.
	if o.kind == KindCustom {
		s.notifyCustomPriceChanged(ctx, updated, req.Reason, res.AfterPickup)
	}
	return updated, res, nil
}

// feeEditStandardTx **الطلبُ العاديّ** — كما يُعيد `EditOrder` حسابَه، والبضاعةُ كما هي.
func (s *Service) feeEditStandardTx(ctx context.Context, tx pgx.Tx, actorID, orderID string,
	o feeOrderRow, req EditFeeRequest, note string, res *EditFeeResult) error {
	res.Payer = FeePayerCustomerSide
	feeBeforePromo := o.deliveryFee + o.waived
	res.OldFee = feeBeforePromo
	if req.Fee == feeBeforePromo {
		return ErrOrderEditNoChange
	}
	// **رسمُ المصدر جزءٌ من الأجرة لا يزيد عليها** — فحذفُ متجرٍ بعدها يطرحه ولا يُنزلها تحت الصفر.
	sourcesFee := min64(o.sourcesFee, req.Fee)
	driverFee := DriverFeeWithShare(req.Fee, o.sharePct, o.shareFixed)

	deliveryFee := req.Fee
	discount := o.discount
	var waived int64
	if o.promoCode != "" {
		var kind string
		var value int64
		var maxDiscount *int64
		err := tx.QueryRow(ctx, `
			SELECT kind, value, max_discount FROM promo_codes WHERE code = $1`, o.promoCode).
			Scan(&kind, &value, &maxDiscount)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// **كودٌ حُذف بعد الطلب** — يبقى إعفاؤه كما كان ولا يزيد على الأجرة.
			waived = min64(o.waived, req.Fee)
			deliveryFee = req.Fee - waived
		case err != nil:
			return err
		default:
			discount = promoCut(kind, value, maxDiscount, o.subtotal, &deliveryFee)
			if req.Fee > deliveryFee {
				waived = req.Fee - deliveryFee
			}
		}
	}
	if fw, err := s.freeDeliveryReapply(ctx, tx, orderID, deliveryFee); err != nil {
		return err
	} else if fw > 0 {
		deliveryFee -= fw
		waived += fw
	}
	total := o.subtotal - discount + deliveryFee
	if total < 0 {
		total = 0
	}

	// **وحصّةُ المحفظة من الدالّة الواحدة** — نقصٌ يُردّ، وزيادةٌ تُخصم بما يكفي والباقي نقد.
	walletPaid := o.walletPaid
	target := WalletPortion(o.scope, total, deliveryFee)
	if target < walletPaid {
		diff := walletPaid - target
		if _, err := s.wallet.ApplyTx(ctx, tx, o.customerID, diff, "refund", orderID, note, &actorID); err != nil {
			return err
		}
		res.Refunded = diff
		walletPaid = target
	} else if target > walletPaid {
		avail, err := s.wallet.AvailableTx(ctx, tx, o.customerID)
		if err != nil {
			return err
		}
		charge := min64(target-walletPaid, avail)
		if charge > 0 {
			if _, err := s.wallet.ApplyTx(ctx, tx, o.customerID, -charge, "order_payment",
				orderID, note, &actorID); err != nil {
				return err
			}
			res.Charged = charge
			walletPaid += charge
		}
	}
	_, err := tx.Exec(ctx, `
		UPDATE orders SET delivery_fee = $2, discount = $3, total = $4, wallet_paid = $5,
		       cash_due = $6, promo_delivery_waived = $7, sources_fee = $8, driver_fee = $9,
		       updated_at = now()
		 WHERE id = $1`, orderID, deliveryFee, discount, total, walletPaid, total-walletPaid,
		waived, sourcesFee, driverFee)
	return err
}

// feeEditCustomTx **الطلبُ الخاصّ** — بابُ «تعديل العرض» والثمنُ بعملته كما هو.
func (s *Service) feeEditCustomTx(ctx context.Context, tx pgx.Tx, actorID, orderID string,
	req EditFeeRequest, res *EditFeeResult) error {
	res.Payer = FeePayerCustomerSide
	row, err := s.lockCustomRow(ctx, tx, orderID)
	if err != nil {
		return err
	}
	res.OldFee = row.fee
	// **ولا أجرةَ تُعدَّل قبل أن تُوثَّق** — وإلّا كُتب عرضٌ بثمنٍ صفرٍ لم يتّفق عليه أحد.
	if !row.hasGoods || !row.hasFee {
		return ErrFeeEditNotAgreed
	}
	if req.Fee == row.fee {
		return ErrOrderEditNoChange
	}
	_, err = s.adminOverrideCustomTx(ctx, tx, row, actorID, row.goods, req.Fee, row.goodsUSD, req.Reason)
	return err
}

// feeEditMerchantDeliveryTx **التوصيلة** («لدي توصيلة» وطلبُ الواتساب) — بدافعها. ويردّ مالكَ
// المتجر ليُخبَر (فراغٌ لطلب الواتساب أو المستلِم).
func (s *Service) feeEditMerchantDeliveryTx(ctx context.Context, tx pgx.Tx, actorID, orderID string,
	o feeOrderRow, req EditFeeRequest, note string, res *EditFeeResult) (string, error) {
	feeBefore := o.deliveryFee + o.waived
	res.OldFee = feeBefore
	if req.Fee == feeBefore {
		return "", ErrOrderEditNoChange
	}
	var owner string
	if o.merchantID != "" {
		if err := tx.QueryRow(ctx, `SELECT COALESCE(owner_user_id::text, '') FROM merchants WHERE id = $1`,
			o.merchantID).Scan(&owner); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}
	// **والتوصيلُ المجانيُّ بعرضه يُعاد على الأجرة الجديدة** (بسقفه إن كان).
	waive, err := s.freeDeliveryReapply(ctx, tx, orderID, req.Fee)
	if err != nil {
		return "", err
	}
	charge := req.Fee - waive
	driverFee := DriverFeeWithShare(req.Fee, o.sharePct, o.shareFixed)
	walletPaid, cashDue := o.walletPaid, int64(0)

	switch o.payer {
	case FeePayerMerchant:
		res.Payer = FeePayerMerchant
		// **قبل قبول المكتب لم يُخصم شيء** — والقبولُ يخصم الأجرةَ الجديدة من الصفّ.
		if o.status != StPending {
			if charge < walletPaid {
				diff := walletPaid - charge
				if _, err := s.wallet.ApplyTx(ctx, tx, owner, diff, "refund", orderID, note, &actorID); err != nil {
					return "", err
				}
				res.Refunded = diff
				walletPaid = charge
			} else if charge > walletPaid {
				diff := charge - walletPaid
				// **ولا دين** (قرارُ المالك ٢٠٢٦-١٠-١٥) — المتاحُ يكفي وإلّا يُردّ التعديل.
				ok, err := s.takeFromMerchantWallet(ctx, tx, orderID, o.merchantID, actorID, diff, note)
				if err != nil {
					return "", err
				}
				if !ok {
					return "", ErrFeeEditMerchantWallet
				}
				res.Charged = diff
				walletPaid = charge
			}
		}
	case FeePayerMerchantCash, FeePayerRecipient:
		res.Payer = o.payer
		// **ونقدٌ قُبض فعلاً لا يُعدَّل** — «أنا نقداً» يُقبض عند الاستلام (أو المستلِمُ في «إحضار»).
		got, err := merchantDeliveryCollected(ctx, tx, orderID)
		if err != nil {
			return "", err
		}
		if got > 0 {
			return "", ErrFeeEditCashCollected
		}
		cashDue = charge
	default:
		return "", ErrOrderEditKind
	}
	_, err = tx.Exec(ctx, `
		UPDATE orders SET delivery_fee = $2, promo_delivery_waived = $3, driver_fee = $4,
		       total = $5, cash_due = $6, wallet_paid = $7, updated_at = now()
		 WHERE id = $1`, orderID, charge, waive, driverFee, charge, cashDue, walletPaid)
	if o.payer == FeePayerRecipient {
		owner = ""
	}
	return owner, err
}

// feeEditTrailTx **سطرٌ في سجلّ الطلب وإشعاراتُ الأطراف** — في المعاملة نفسِها، فالمعاينةُ
// المطويّةُ لا تُخبر أحداً.
func (s *Service) feeEditTrailTx(ctx context.Context, q dbtx.Querier, actorID, orderID string,
	o feeOrderRow, merchantOwner string, res EditFeeResult) error {
	change := fmt.Sprintf("أجرة التوصيل %s ← %s ل.س", groupDigits(res.OldFee), groupDigits(res.NewFee))
	var actor any
	if actorID != "" {
		actor = actorID
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $2, $3, $4, $5)`, orderID, o.status, actor,
		fmt.Sprintf("تعديل الأجرة: %s — السبب: %s — المجموع %s ← %s ل.س",
			change, res.Reason, groupDigits(res.OldTotal), groupDigits(res.NewTotal)),
		o.driverID); err != nil {
		return err
	}
	// **الزبونُ في العاديّ** — والخاصُّ إشعارُه «تعديل العرض» بعد التثبيت.
	if o.kind == KindStandard && o.customerID != "" {
		body := fmt.Sprintf("طلب #%d: %s — السبب: %s. صار مجموع طلبك %s ل.س",
			o.number, change, res.Reason, groupDigits(res.NewTotal))
		if res.Refunded > 0 {
			body += fmt.Sprintf("، ورجع %s ل.س إلى محفظتك", groupDigits(res.Refunded))
		}
		if res.Charged > 0 {
			body += fmt.Sprintf("، وخُصم %s ل.س من محفظتك", groupDigits(res.Charged))
		}
		if err := notifications.NotifyUserTx(ctx, q, notifications.Input{
			UserID: o.customerID, Kind: notifications.KindOrder,
			Title: t.feeEdited, Body: body,
			Entity: "order", EntityID: orderID, Href: "/portal/orders",
			Apps: []string{notifications.AppCustomer},
		}); err != nil {
			return err
		}
	}
	// **والمتجرُ الدافعُ في التوصيلة.**
	if merchantOwner != "" {
		body := fmt.Sprintf("توصيلة #%d: %s — السبب: %s", o.number, change, res.Reason)
		if res.Refunded > 0 {
			body += fmt.Sprintf("، ورجع %s ل.س إلى محفظتك", groupDigits(res.Refunded))
		}
		if res.Charged > 0 {
			body += fmt.Sprintf("، وانخصم %s ل.س من محفظتك", groupDigits(res.Charged))
		}
		if res.Payer == FeePayerMerchantCash {
			body += fmt.Sprintf(". المبلغ اللي بتدفعه للكابتن صار %s ل.س", groupDigits(res.CashDue))
		}
		if err := notifications.NotifyUserTx(ctx, q, notifications.Input{
			UserID: merchantOwner, Kind: notifications.KindOrder,
			Title: t.feeEdited, Body: body,
			Entity: "order", EntityID: orderID, Href: "/portal",
			Apps: []string{notifications.AppMerchant},
		}); err != nil {
			return err
		}
	}
	// **والسائقُ بأجره الجديد وبما يقبضه** — إلّا زيادةَ خاصٍّ بعد الاستلام: يُخبره «تعديل العرض»
	// أنّ التسليمَ ينتظر تأكيدَ الزبون (`notifyCustomPriceChanged`)، فلا إشعاران.
	skipDriver := o.kind == KindCustom && res.Reconfirm && res.AfterPickup
	if o.driverID != nil && *o.driverID != "" && !skipDriver {
		body := fmt.Sprintf("طلب #%d: %s. أجرتك صارت %s ل.س، والمبلغ المطلوب قبضه صار %s ل.س",
			o.number, change, groupDigits(res.DriverFee), groupDigits(res.CashDue))
		if err := notifications.NotifyUserTx(ctx, q, notifications.Input{
			UserID: *o.driverID, Kind: notifications.KindOrder,
			Title: t.feeEditedDriver, Body: body,
			Entity: "order", EntityID: orderID, Href: "/portal",
			Apps: []string{notifications.AppDriver},
		}); err != nil {
			return err
		}
	}
	return nil
}
