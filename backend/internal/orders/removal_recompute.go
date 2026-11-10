package orders

// ══════════════════════════════════════════════════════════════════════
// **بضاعةٌ خرجت من الطلب — وإعادةُ حسابه واحدة** (صنفٌ غيرُ متوفّر · قسمٌ انشال)
// ══════════════════════════════════════════════════════════════════════
//
// **بابان يُخرجان بضاعةً قبل الاستلام**: صنفٌ نفد (`MarkItemUnavailable`) ومتجرٌ رفض
// قسمَه أو شاله المكتب (`RemovePart`). **والحسبةُ بعدهما واحدة** — فلا تفترق يوماً:
//
//	البضاعة · رسمُ المصادر (لا يرتفع) · أجرُ السائق · متجرُ الطلب · الكود · التوصيلُ المجانيّ
//	· المجموعُ (لا يرتفع) · الدفعُ بقاعدة تعديل الطلب
//
// **وأجرُ السائق يتبع الأجرةَ دائماً** حين يخرج متجر (قرارُ المالك ٢٠٢٦-١١-٠٦، ١ب):
// **الأجرةُ الأساسيّةُ وحدَها ولو وصل السائقُ للمتجر الذي انشال.**

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// removalMoney **ما صار إليه مالُ الطلب بعد خروج البضاعة.**
type removalMoney struct {
	oldTotal, newTotal, refunded int64
	sourceGone                   bool
}

// recomputeAfterRemovalTx **يعيد حسابَ الطلب بعد حذف بنودٍ منه** — في المعاملة نفسِها.
//
// `removedStore` متجرُ البنود التي خرجت — **ومنه يُعرف أخرج متجرٌ كاملٌ من الطلب.**
// `refundNote` نصُّ قيد ردّ المحفظة إن وقع. **والبنودُ محذوفةٌ قبل النداء.**
func (s *Service) recomputeAfterRemovalTx(ctx context.Context, tx pgx.Tx, orderID, removedStore,
	actorID, refundNote string) (removalMoney, error) {
	var res removalMoney
	var o struct {
		customerID, promoCode, merchantID, scope       string
		deliveryFee, discount, total, walletPaid       int64
		waived, sourcesFee, driverFee, sharePct, share int64
	}
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(customer_id::text, ''), COALESCE(promo_code, ''), COALESCE(merchant_id::text, ''),
		       delivery_fee, discount, total, wallet_paid,
		       promo_delivery_waived, sources_fee, driver_fee,
		       snap_platform_delivery_percent, snap_platform_delivery_fixed, wallet_scope
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&o.customerID, &o.promoCode, &o.merchantID, &o.deliveryFee, &o.discount, &o.total,
			&o.walletPaid, &o.waived, &o.sourcesFee, &o.driverFee, &o.sharePct, &o.share, &o.scope); err != nil {
		return res, err
	}

	// ── ١ · البضاعةُ من البنود — المعادلةُ نفسُها التي سعّرت الطلب ───────
	// (`priceItems`: سعرُ البيع الملقوطُ × الكمّيّة.)
	var subtotal int64
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(sum(unit_price * qty), 0) FROM order_items WHERE order_id = $1`, orderID).
		Scan(&subtotal); err != nil {
		return res, err
	}

	// ── ٢ · أخرج متجرٌ كاملٌ من الطلب؟ ───────────────────────────────────
	feeBeforePromo := o.deliveryFee + o.waived
	sourcesFee, driverFee, merchantID := o.sourcesFee, o.driverFee, o.merchantID
	var stillThere bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM order_items
		              WHERE order_id = $1 AND COALESCE(merchant_id::text, $2) = $3)`,
		orderID, o.merchantID, removedStore).Scan(&stillThere); err != nil {
		return res, err
	}
	if !stillThere {
		res.sourceGone = true
		remaining, err := remainingSources(ctx, tx, orderID, o.merchantID)
		if err != nil {
			return res, err
		}
		// **ومتجرُ الطلب إن خرج يخلفه أكبرُ الباقين** — وإلّا بقي الطلبُ على متجرٍ لا بضاعةَ له فيه.
		if removedStore == o.merchantID && len(remaining.merchants) > 0 {
			merchantID = remaining.merchants[0]
		}
		// **ورسمُ المصادر يُعاد بالمعادلة نفسِها** (`extraSourceFee`) على ما بقي،
		// **ولا يرتفع أبداً** — إعداداتُ اليوم لا تُحمَّل على طلب الأمس.
		if o.sourcesFee > 0 {
			base := feeBeforePromo - o.sourcesFee
			var extra int64
			if len(remaining.items) > 0 {
				src, err := s.SourcesOf(ctx, tx, remaining.items)
				if err != nil {
					return res, err
				}
				extra = s.extraSourceFee(ctx, src, base)
			}
			if extra > o.sourcesFee {
				extra = o.sourcesFee
			}
			sourcesFee = extra
			feeBeforePromo = base + extra
			// **وأجرُ السائق يتبع الأجرة** — وقفةٌ أقلّ، **بحصّة المنصّة الملقوطة.**
			driverFee = DriverFeeWithShare(feeBeforePromo, o.sharePct, o.share)
		}
	}

	// ── ٣ · الكود: يُعاد حسابُه على البضاعة الجديدة — ولا يسقط ─────────────
	//
	// **لا يُسأل عن حدّه الأدنى ولا عن صلاحيّته اليوم**: وُعد به الزبونُ يومَ الطلب،
	// **وحذفُ ما نفد عند المتجر لا ينقض الوعد.** النسبةُ تُعاد بسقفها، والثابتُ يُقصَر على
	// البضاعة (`promoCut`)، **والاستعمالُ وعدّادُه لا يُمسّان.**
	deliveryFee := feeBeforePromo
	var discount, waived int64
	promoCode := o.promoCode
	if promoCode != "" {
		var kind string
		var value int64
		var maxDiscount *int64
		err := tx.QueryRow(ctx, `
			SELECT kind, value, max_discount FROM promo_codes
			WHERE code = $1`, promoCode).Scan(&kind, &value, &maxDiscount)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// **كودٌ مُحي بعد الطلب** — يبقى خصمُه كما وُعد، لا يُخترَع غيرُه.
			discount, waived, deliveryFee = min64(o.discount, subtotal), o.waived, o.deliveryFee
		case err != nil:
			return res, err
		default:
			discount = promoCut(kind, value, maxDiscount, subtotal, &deliveryFee)
			if feeBeforePromo > deliveryFee {
				waived = feeBeforePromo - deliveryFee
			}
		}
	}
	// **والتوصيلُ المجانيّ يبقى وعداً** — يُعاد على الأجرة الجديدة بسقف عرضه.
	if fw, err := s.freeDeliveryReapply(ctx, tx, orderID, deliveryFee); err != nil {
		return res, err
	} else if fw > 0 {
		deliveryFee -= fw
		waived += fw
	}
	total := subtotal - discount + deliveryFee
	if total < 0 {
		total = 0
	}
	// **وحارسٌ بنيويّ**: الحسبةُ أعلاه لا ترفع المجموعَ (البضاعةُ تنقص والخصمُ يتبعها
	// صعوداً ونزولاً)، **فإن ارتفع يوماً بقاعدةٍ تُضاف بعدُ امتصّه الخصمُ** — ولا يدفع
	// الزبونُ أكثرَ ممّا وافق عليه.
	if total > o.total {
		extra := min64(total-o.total, subtotal-discount)
		discount += extra
		total -= extra
	}
	res.oldTotal, res.newTotal = o.total, total

	// ── ٤ · الدفع: بقاعدة تعديل الطلب — المحفظةُ يُردّ لها ما زاد فوراً ──────
	//
	// **المحفظةُ لا تزيد على ما دُفع منها، وما زاد منها على المجموع الجديد يُردّ لها، والباقي
	// نقد** — القاعدةُ نفسُها في `EditOrder` (`order_edit.go`).
	//
	// **وكان طلبُ المحفظة يُكتب كلُّه من المحفظة** (`wallet_paid = total`، `cash_due = 0`) — فطلبٌ
	// زِيد عليه بتعديلٍ لم يكفِه رصيدُ الزبون (فبقي عليه نقد) **يضيع باقيه النقديّ بحذف صنف**: لا
	// يقبضه السائق، **والطلبُ يقول دُفع من المحفظة ما لم يُدفع منها.** (خلل PD-2، ٢٠٢٦-١١-٠٥.)
	// **والحصّةُ من الدالّة الواحدة** (`WalletPortion`، قرارُ المالك ٢٠٢٦-١١-٠٩) — في «التوصيل من
	// المحفظة» لا تتجاوز الأجرةَ الجديدة. **والإزالةُ لا تُخصم شيئاً** — المجموعُ لا يزيد هنا.
	walletPaid := min64(o.walletPaid, WalletPortion(o.scope, total, deliveryFee))
	if diff := o.walletPaid - walletPaid; diff > 0 {
		if _, err := s.wallet.ApplyTx(ctx, tx, o.customerID, diff, "refund",
			orderID, refundNote, &actorID); err != nil {
			return res, err
		}
		res.refunded = diff
	}
	cashDue := total - walletPaid

	if _, err := tx.Exec(ctx, `
		UPDATE orders SET subtotal = $2, delivery_fee = $3, discount = $4, total = $5,
		       wallet_paid = $6, cash_due = $7, promo_delivery_waived = $8,
		       sources_fee = $9, driver_fee = $10, promo_code = NULLIF($11, ''),
		       merchant_id = $12::uuid, updated_at = now()
		WHERE id = $1`, orderID, subtotal, deliveryFee, discount, total,
		walletPaid, cashDue, waived, sourcesFee, driverFee, promoCode, merchantID); err != nil {
		return res, err
	}
	return res, nil
}
