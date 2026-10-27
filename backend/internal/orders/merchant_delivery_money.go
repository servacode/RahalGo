package orders

// ══════════════════════════════════════════════════════════════════════
// **مالُ «لدي توصيلة»** — قراراتُ المالك ٢٠٢٦-٠٩-٢٩ ثمّ ٢٠٢٦-١٠-١٥
// ══════════════════════════════════════════════════════════════════════
//
// # ولا مالَ يظهر من لا شيء
//
// **كلُّ ليرةٍ لها مصدرٌ ووجهة**، **وثنائيّةُ القيد بنيويّةٌ لا فحصيّة**:
// `creditTreasury` تُخرج البقيّةَ (مدفوعٌ − مستردٌّ − ما وصل الأطراف)،
// **فما لم يصل طرفاً بلغ الخزينةَ من نفسِه.**
//
// # وثلاثةُ دافعين — ولا دين
//
//	المتجرُ (محفظة) ⇐ **من محفظته عند قبول المكتب** (لا عند الإنشاء — ٢٠٢٦-١٠-٠٥)،
//	                  **فإن لم يكفِ المتاحُ رُدّ الطلبُ والقبول** (٢٠٢٦-١٠-١٥)
//	المتجرُ (نقداً)  ⇐ **بيد السائق عند الاستلام** — ويُقيَّد في صندوقه ساعتَها (٢٠٢٦-١٠-١٥)
//	المستلِمُ        ⇐ **نقداً بيد السائق** عند التسليم (`cash_due`)
//
// **ولا محفظةَ سالبةَ أبداً** (قرارُ المالك ٢) — **ولا دينَ على أجرة توصيلة** (قرارُ المالك
// ٢٠٢٦-١٠-١٥: كان «الدينُ المفتوح» مشتعلاً افتراضاً فيُقيَّد على المتجر بلا حدٍّ ولا يُخبَر).
// **والاستثناءُ الوحيد**: توصيلةٌ فشلت بذنب المستلِم والمستلِمُ دافعُها — انظر
// `merchantBearsFailedDelivery`.

import (
	"context"
	"errors"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// ErrDeliveryCreditExhausted **المتاحُ في محفظة المتجر لا يكفي الأجرة** — عند الطلب.
//
// **والاسمُ القديمُ باقٍ ورمزُه** (التطبيقاتُ تعرفه)، **ومعناه اليوم**: «رصيد محفظتك ما
// بيكفي — اشحنها أو اختر الدفع نقداً». ولا سقفَ دينٍ بعد ٢٠٢٦-١٠-١٥.
var ErrDeliveryCreditExhausted = httpx.NewError(http.StatusPaymentRequired,
	"delivery_credit_exhausted", "errors.delivery_credit_exhausted")

// ErrMerchantDeliveryUnpaid **المكتبُ يقبل توصيلةً لا يكفي متاحُ متجرها أجرتَها**
// (قرارُ المالك ٢٠٢٦-١٠-٠٥ ثمّ ٢٠٢٦-١٠-١٥): **يُردّ القبولُ** وتبقى معلَّقةً — يرفضها
// المكتبُ أو ينتظر شحنَ المحفظة. **ولا دينَ يُكتب مكانَ الخصم.**
var ErrMerchantDeliveryUnpaid = httpx.NewError(http.StatusPaymentRequired,
	"merchant_delivery_unpaid", "errors.merchant_delivery_unpaid")

// chargeMerchantDelivery **يخصم أجرةَ التوصيلة من محفظة المتجر** — أو يردّ.
//
// # لماذا داخلَ المعاملة
//
// **قراءةُ الرصيد ثمّ الخصمُ بلا قفلٍ تمرّان معاً**: توصيلتان في اللحظة
// نفسِها تقرآن الرصيدَ كافياً فتخصمان مرّتين، **فيهبط تحت الصفر** — وهو ما
// يمنعه القرار. **فالقراءةُ والكتابةُ في معاملةٍ واحدة.**
//
// # ولا دين (قرارُ المالك ٢٠٢٦-١٠-١٥)
//
// **كان يُقيَّد دينٌ بكامل الأجرة إن لم تكفِ المحفظة** — بسقفٍ إداريّ ثمّ بلا سقفٍ
// (`delivery.merchant_debt_open`). **وصار**: المتاحُ (الرصيدُ − المحجوز) يكفي ⇐ يُخصم،
// وإلّا ⇐ `ErrDeliveryCreditExhausted` **ولا يُكتب شيء.** والمفتاحُ وسقفُ المتجر لا
// يُقرآن بعد اليوم.
func (s *Service) chargeMerchantDelivery(ctx context.Context, q wallet.Querier,
	orderID, merchantID, actorID string, fee int64) error {
	if fee <= 0 {
		return nil
	}
	ok, err := s.debitMerchantWallet(ctx, q, orderID, merchantID, actorID, fee,
		"أجرةُ توصيلةٍ من متجرك")
	if err != nil {
		return err
	}
	if !ok {
		return ErrDeliveryCreditExhausted
	}
	return nil
}

// debitMerchantWallet **يخصم من محفظة صاحب المتجر إن كفى المتاح** — ويردّ `false` بلا
// كتابةٍ إن لم يكفِ.
//
// **ويُكتب على الطلب أنّه دُفع** (`wallet_paid`) — **وإلّا موّلت الخزينةُ السائقَ**:
// `creditTreasury` تقرأ المدفوعَ من الصفّ لا من المحفظة، **فخصمٌ لا يُكتب على الطلب
// يُقرأ صفراً فتُقيَّد الخزينةُ خسارةً وهي التي كسبت.**
func (s *Service) debitMerchantWallet(ctx context.Context, q wallet.Querier,
	orderID, merchantID, actorID string, amount int64, note string) (bool, error) {
	// **والقبولُ التلقائيُّ بلا فاعل** — **والفارغُ ليس `uuid`** فيُسقط القيدَ
	// والمعاملةَ معه. **فيُكتب `NULL`**: لا إنسانَ فعلها.
	var actor *string
	if actorID != "" {
		actor = &actorID
	}
	var ownerID string
	if err := q.QueryRow(ctx,
		`SELECT owner_user_id::text FROM merchants WHERE id = $1`, merchantID).
		Scan(&ownerID); err != nil {
		return false, err
	}
	// **والمتاحُ يُقرأ بقفلِ صفِّه** — انظر أعلاه. **والمحجوزُ ليس للدفع** (سحبٌ معلَّق).
	_, available, err := merchantWallet(ctx, q, merchantID, true)
	if err != nil {
		return false, err
	}
	if !merchantCanCover(available, amount) {
		return false, nil
	}
	// **ونوعُه يقول لماذا نقص الرصيد** (قرارُ المالك ٢٠٢٦-٠٨-١١: «الرصيد يتغيّر وما حدا
	// بيعرف ليش»).
	if _, err := s.wallet.ApplyTx(ctx, q, ownerID, -amount, "order_payment",
		orderID, note, actor); err != nil {
		return false, err
	}
	if _, err := q.Exec(ctx,
		`UPDATE orders SET wallet_paid = wallet_paid + $2 WHERE id = $1`, orderID, amount); err != nil {
		return false, err
	}
	return true, nil
}

// merchantDeliveryCollected **ما قُيّد في الصناديق تحصيلاً عن هذه التوصيلة.**
//
// **وهو الحارسُ من التحصيل مرّتين** (قرارُ المالك ٢٠٢٦-١٠-١٥): «أنا نقداً» يُقبض عند
// الاستلام، **فالتسليمُ لا يقبضه ثانيةً** — ولا استلامٌ ثانٍ بعد طارئ سائق.
func merchantDeliveryCollected(ctx context.Context, q wallet.Querier, orderID string) (int64, error) {
	var n int64
	err := q.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0) FROM driver_cash_entries
		 WHERE ref = $1 AND kind = 'order_collection'`, orderID).Scan(&n)
	return n, err
}

// cashAtPickup **أيُقبض النقدُ عند الاستلام؟** — حين يقف دافعُه عند نقطة الاستلام:
// المتجرُ في «توصيل» («أنا نقداً»)، **والمستلِمُ في «إحضار».** وما سواه عند التسليم.
func cashAtPickup(payer, mode string) bool {
	if mode == DeliveryModeFetch {
		return payer == FeePayerRecipient
	}
	return payer == FeePayerMerchantCash
}

// collectMerchantDeliveryCash **يُقيّد نقدَ الأجرة في صندوق السائق — مرّةً واحدة.**
func (s *Service) collectMerchantDeliveryCash(ctx context.Context, q wallet.Querier, in settlement) error {
	if in.cashDue <= 0 || in.driverID == nil {
		return nil
	}
	got, err := merchantDeliveryCollected(ctx, q, in.orderID)
	if err != nil || got > 0 {
		return err
	}
	return s.cashbox.CollectTx(ctx, q, *in.driverID, in.cashDue, in.orderID, &in.actorID)
}

// ══════════════════════════════════════════════════════════════════════
// **تسويةُ التوصيلة — مسارٌ ثالثٌ لا يمرّ بالعامّ** (الخطوة ١٨، ٢٠٢٦-١٠-٠١)
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا لا يمرّ بالعامّ
//
// **العامُّ يفترض زبوناً وبضاعة**: يُعيد المحفظةَ إلى `customer_id` عند
// الإلغاء (**وهو فارغٌ هنا — فيضيع مالُ المتجر**)، **ويُقيّد مستحقَّ المتجر عند
// الاستلام** (لا بضاعةَ باعها)، **ويصرف عمولةَ المندوب عند التسليم** (لا مبيعات).
// **وكلُّ واحدةٍ منها خطأٌ ماليٌّ صامتٌ لو مرّت التوصيلةُ من هناك.**
//
// # وما يقع
//
//	القبول           ⇐ يُخصم «المتجر يدفع» من محفظته أو يُردّ القبول
//	الاستلام         ⇐ نقدُ «أنا نقداً» إلى صندوق السائق (٢٠٢٦-١٠-١٥)
//	التسليم          ⇐ نقدُ المستلِم إلى صندوق السائق (إن دفع هو) · أجرُ السائق ·
//	                   والبقيّةُ للخزينة (`creditTreasury`)
//	فشلٌ بعد الاستلام ⇐ **بالذنب الحقيقيّ** (٢٠٢٦-١٠-١٥) — انظر الفرعَ أدناه
//	نهايةٌ بلا تسليم ⇐ **يُعاد المالُ لمن دفعه**. **ولا أجرَ لسائقٍ لم يُسلّم.**
func (s *Service) settleMerchantDelivery(ctx context.Context, q wallet.Querier,
	in settlement, out *settled) error {
	switch {
	// ══════════════════════════════════════════════════════════════════
	// **قبولُ المكتب — وفيه وحدَه تُخصم أجرةُ «المتجر يدفع»** (٢٠٢٦-١٠-٠٥)
	// ══════════════════════════════════════════════════════════════════
	//
	// **في معاملة القبول نفسِها**: إن لم يكفِ المتاحُ سقط القبولُ كلُّه وبقيت
	// التوصيلةُ معلَّقةً بيد المكتب — **فلا تنزل الطابورَ توصيلةٌ لم يُدفع أجرُها،
	// ولا يُكتب دين** (٢٠٢٦-١٠-١٥). والنقدُ لا شيءَ يُخصم له.
	case in.to == StAccepted && in.from == StPending:
		var merchantID, payer string
		var fee int64
		if err := q.QueryRow(ctx, `
			SELECT merchant_id::text, COALESCE(fee_payer, ''), delivery_fee
			  FROM orders WHERE id = $1`, in.orderID).Scan(&merchantID, &payer, &fee); err != nil {
			return err
		}
		if payer != FeePayerMerchant {
			return nil
		}
		err := s.chargeMerchantDelivery(ctx, q, in.orderID, merchantID, in.actorID, fee)
		if errors.Is(err, ErrDeliveryCreditExhausted) {
			// **والمكتبُ يُقال له ما يفعل** — لا نصُّ المتجر «اشحن محفظتك».
			return ErrMerchantDeliveryUnpaid
		}
		return err

	// ══════════════════════════════════════════════════════════════════
	// **«أنا نقداً» يُقبض عند الاستلام ويُقيَّد ساعتَها** (قرارُ المالك ٢٠٢٦-١٠-١٥)
	// ══════════════════════════════════════════════════════════════════
	//
	// **كان يُقيَّد عند التسليم** — والسائقُ قبضه من المتجر قبل أن يتحرّك، **فصندوقُه
	// يحمل مالاً لا يراه أحدٌ ساعةً كاملة، ويضيع أثرُه إن فشلت التوصيلة.**
	//
	// **والقاعدةُ: من يقف عند نقطة الاستلام يدفع عند الاستلام.** وفي «إحضار» تنقلب
	// الوقفتان: **المستلِمُ عند نقطة الاستلام** فيُقبض منه عندها، **والمتجرُ عند التسليم.**
	case in.to == StPickedUp:
		var payer, mode string
		if err := q.QueryRow(ctx,
			`SELECT COALESCE(fee_payer, ''), delivery_mode FROM orders WHERE id = $1`,
			in.orderID).Scan(&payer, &mode); err != nil {
			return err
		}
		if !cashAtPickup(payer, mode) {
			return nil
		}
		return s.collectMerchantDeliveryCash(ctx, q, in)

	case in.to == StDelivered:
		// **نقدُ المستلِم عند التسليم** — **وما قُبض عند الاستلام لا يُقبض ثانيةً.**
		if err := s.collectMerchantDeliveryCash(ctx, q, in); err != nil {
			return err
		}
		if err := s.payDriver(ctx, q, in, out); err != nil {
			return err
		}
		return s.creditTreasury(ctx, q, in.orderID, in.actorID)

	case refundOnEnter(in.to) && in.from != StDelivered:
		var merchantID, ownerID, payer, fault, reason string
		var fee int64
		if err := q.QueryRow(ctx, `
			SELECT o.merchant_id::text, m.owner_user_id::text, COALESCE(o.fee_payer, ''),
			       COALESCE(o.fault, ''), COALESCE(o.fail_reason, ''), o.delivery_fee
			  FROM orders o JOIN merchants m ON m.id = o.merchant_id
			 WHERE o.id = $1`, in.orderID).Scan(&merchantID, &ownerID, &payer, &fault, &reason, &fee); err != nil {
			return err
		}
		collected, err := merchantDeliveryCollected(ctx, q, in.orderID)
		if err != nil {
			return err
		}
		afterPickup := in.to == StFailed && in.driverID != nil && pastPickup[in.from]

		// ══════════════════════════════════════════════════════════════
		// **فشلٌ بعد الاستلام بذنب المستلِم أو المتجر ⇒ كأنّها سُلّمت** (قرارُ المالك ٢٠٢٦-١٠-١٥)
		// ══════════════════════════════════════════════════════════════
		//
		// **كان كلُّ فشلٍ بعد الاستلام يُردّ للمتجر كاملاً ويُعوَّض السائقُ بذنبٍ «المنصّة»
		// مفروض** — فمستلِمٌ أغلق بابَه تحمّلت المنصّةُ مشوارَه. **وصار الذنبُ الحقيقيُّ
		// يحكم**: المتجرُ يدفع الأجرة، والسائقُ يقبض أجرَه، والمنصّةُ حصّتَها، **ولا طلبَ
		// تعويض.**
		if afterPickup && (fault == FaultCustomer || fault == FaultMerchant) {
			return s.merchantBearsFailedDelivery(ctx, q, in, out, merchantID, fee, collected)
		}

		// ── وما سواه: يُعاد المالُ لمن دفعه ──────────────────────────────
		//
		// **محفظتُه إن خُصمت، ونقدُه إن قبضه السائقُ عند الاستلام** («أنا نقداً» —
		// ٢٠٢٦-١٠-١٥): يُردّ إلى محفظته، **والنقدُ في صندوق السائق يُسلَّم للمكتب كسائر
		// النقد.** ونقدُ المستلِم في «إحضار» يُردّ له بيد السائق — انظر تحت.
		refund := in.walletPaid
		if payer == FeePayerMerchantCash {
			refund += collected
		}
		if refund > 0 {
			if _, err := s.wallet.ApplyTx(ctx, q, ownerID, refund, "refund",
				in.orderID, "استرجاعُ أجرةِ توصيلةٍ لم تُسلَّم", &in.actorID); err != nil {
				return err
			}
			out.credit(ownerID, refund, t.refunded2, notifications.AppMerchant)
		}
		// ══════════════════════════════════════════════════════════════
		// **ونقدُ المستلِم يرجع له إن لم يكن الذنبُ ذنبَه** (قرارُ المالك ٢٠٢٦-١٠-٢٧)
		// ══════════════════════════════════════════════════════════════
		//
		// **في «إحضار» يدفع المستلِمُ للسائق عند الاستلام** (`cashAtPickup`). وكان التعليقُ
		// هنا يقول «لا يُقبض قبل التسليم» — **وذاك قبل ٢٠٢٦-١٠-١٥**؛ فصار نقدُه يبقى في صندوق
		// السائق وتقرؤه الخزينةُ ربحاً على توصيلةٍ فشلت بذنبها أو بذنب السائق.
		//
		// **فالسائقُ يردّه للمستلِم بيده**: سطرٌ سالبٌ في صندوقه (`recipient_cash_returned`)
		// يقابل سطرَ القبض — **فلا يبقى بذمّته، ولا تحسبه الخزينةُ مدفوعاً** (`creditTreasury`
		// تجمع السطرين). **وذنبُ المستلِم** (`customer`) يبقى كما كان: نقدُه للسائق والمنصّة.
		if payer == FeePayerRecipient && fault != FaultCustomer && collected > 0 {
			if err := s.returnRecipientCash(ctx, q, in.orderID, collected, in.actorID); err != nil {
				return err
			}
		}
		// **والدينُ القديمُ يُسقَط كما يُردّ المال** — دينٌ على توصيلةٍ لم تقع باطل.
		// (لا دينَ جديدٌ منذ ٢٠٢٦-١٠-١٥؛ والقائمُ قبله يُسقَط هنا.)
		if err := obligations.VoidForOrder(ctx, q, obligations.PartyMerchant, merchantID,
			in.orderID, causeMerchantDeliveryFee, &in.actorID); err != nil {
			return err
		}
		// ══════════════════════════════════════════════════════════════
		// **وذنبُ المنصّة يُعوَّض فيه السائق — وذنبُ السائق لا** (قرارُ المالك ٢٠٢٦-١٠-١٥)
		// ══════════════════════════════════════════════════════════════
		//
		// **ذنبُ السائق**: لا أجرَ ولا تعويض، **ومن يتحمّل الكلفةَ يختاره المكتبُ عند الباب**
		// كالطلب العاديّ (`chargeDriverLoss` في `door.go` — المنصّةُ أو السائقُ كلَّها أو
		// بنسبة). **وما سواه** (المنصّةُ، أو إنهاءٌ بلا ذنبٍ مكتوب) ⇒ طلبُ تعويضٍ معلَّقٌ
		// بذنب المنصّة (`compensation_requests.go`) يوافق عليه إنسان.
		if afterPickup && fault != FaultDriver {
			created, err := s.requestDriverCompensation(ctx, q, in.orderID, in.driverID,
				FaultPlatform, reason, in.deliveryFee)
			if err != nil {
				return err
			}
			out.compensationRequested = out.compensationRequested || created
		}
		return s.creditTreasury(ctx, q, in.orderID, in.actorID)
	}
	return nil
}

// merchantBearsFailedDelivery **المتجرُ يدفع الأجرةَ كأنّها سُلّمت** — فشلٌ بعد الاستلام
// بذنب المستلِم (أو المتجر). (قرارُ المالك ٢٠٢٦-١٠-١٥.)
//
//	محفظة     ⇐ الخصمُ باقٍ — لا ردّ
//	أنا نقداً  ⇐ النقدُ المقبوضُ عند الاستلام باقٍ
//	المستلِم   ⇐ لم يُقبض شيء: **يُخصم من محفظة المتجر إن كفى المتاح، وإلّا فدينٌ عليه**
//	            بسبب `delivery_failed_recipient`
//
// **والدينُ هنا الاستثناءُ الوحيدُ من «لا دينَ على أجرة توصيلة»**: ليس تسهيلاً يطلبه المتجر،
// **بل كلفةٌ وقعت** — سائقٌ قاد وقبض أجرَه، والمتجرُ هو من أرسل إلى مستلِمٍ لم يستلم.
//
// **والسائقُ يقبض أجرَه كاملاً والمنصّةُ حصّتَها** — ولا طلبَ تعويض.
func (s *Service) merchantBearsFailedDelivery(ctx context.Context, q wallet.Querier,
	in settlement, out *settled, merchantID string, fee, collected int64) error {
	if owed := fee - in.walletPaid - collected; owed > 0 {
		paid, err := s.debitMerchantWallet(ctx, q, in.orderID, merchantID, in.actorID, owed,
			"أجرةُ توصيلةٍ فشلت بذنب المستلِم")
		if err != nil {
			return err
		}
		if !paid {
			if _, err := obligations.Create(ctx, q, obligations.PartyMerchant, merchantID,
				owed, obligations.CauseDeliveryFailedRecipient, in.orderID, &in.actorID); err != nil {
				return err
			}
		}
	}
	if err := s.payDriver(ctx, q, in, out); err != nil {
		return err
	}
	return s.creditTreasury(ctx, q, in.orderID, in.actorID)
}

// returnRecipientCash **السائقُ الذي قبض يردّ** — لا السائقُ الحاليّ: طارئٌ بعد الاستلام
// قد يبدّل السائق، **والنقدُ بيد من قبضه.** (قرارُ المالك ٢٠٢٦-١٠-٢٧.)
//
// **ولا يُردّ أكثرُ ممّا بذمّته**: سائقٌ سلّم صندوقَه للمكتب قبل أن تنتهي التوصيلةُ لم يبقَ
// النقدُ معه — وسطرٌ يُهبط صندوقَه تحت الصفر يُسقط الانتقالَ كلَّه (`ErrOverSettle`). فيُردّ
// ما بذمّته، **والباقي يُكتب تحذيراً** يردّه المكتبُ من الدرج بيده.
func (s *Service) returnRecipientCash(ctx context.Context, q wallet.Querier,
	orderID string, amount int64, actorID string) error {
	var driverID string
	var held int64
	if err := q.QueryRow(ctx, `
		SELECT e.driver_id::text, COALESCE(b.held, 0)
		  FROM driver_cash_entries e
		  LEFT JOIN driver_cash_boxes b ON b.driver_id = e.driver_id
		 WHERE e.ref = $1 AND e.kind = 'order_collection'
		 ORDER BY e.id LIMIT 1`, orderID).Scan(&driverID, &held); err != nil {
		return err
	}
	give := amount
	if give > held {
		give = held
		s.logger.Warn("ردُّ نقد المستلِم: الصندوقُ أقلّ من المقبوض — الباقي يردّه المكتب",
			"order", orderID, "driver", driverID, "collected", amount, "held", held)
	}
	var actor *string
	if actorID != "" {
		actor = &actorID
	}
	return s.cashbox.ReturnRecipientCashTx(ctx, q, driverID, give, orderID, actor)
}

// causeMerchantDeliveryFee **سببُ دينِ أجرة التوصيلة القديم** — لا يُكتب جديدٌ منه منذ
// ٢٠٢٦-١٠-١٥، **ويُسقَط القائمُ منه إن انتهت توصيلتُه بلا تسليم.**
const causeMerchantDeliveryFee = "merchant_delivery_fee"

// MerchantDeliveryCredit **سقفُ دينِ متجرٍ ورصيدُه** — لشاشة الإدارة.
//
// **والسقفُ لا يحكم شيئاً منذ ٢٠٢٦-١٠-١٥** (لا دينَ على أجرة توصيلة) — يُقرأ لمن فتح
// الشاشةَ القديمة، **والقائمُ يبقى حقيقيّاً**: ديونٌ قديمةٌ تُتابَع في «الديون».
func (s *Service) MerchantDeliveryCredit(ctx context.Context, merchantID string) (limit, owed int64, err error) {
	if err = s.db.QueryRow(ctx,
		`SELECT delivery_credit_limit FROM merchants WHERE id = $1`, merchantID).
		Scan(&limit); err != nil {
		return 0, 0, err
	}
	owed, err = obligations.Balance(ctx, s.db, obligations.PartyMerchant, merchantID)
	return limit, owed, err
}
