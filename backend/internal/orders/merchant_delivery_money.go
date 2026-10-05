package orders

// ══════════════════════════════════════════════════════════════════════
// **مالُ «لدي توصيلة»** — قراراتُ المالك ٢٠٢٦-٠٩-٢٩
// ══════════════════════════════════════════════════════════════════════
//
// # ولا مالَ يظهر من لا شيء
//
// **كلُّ ليرةٍ لها مصدرٌ ووجهة**، **وثنائيّةُ القيد بنيويّةٌ لا فحصيّة**:
// `creditTreasury` تُخرج البقيّةَ (مدفوعٌ − مستردٌّ − ما وصل الأطراف)،
// **فما لم يصل طرفاً بلغ الخزينةَ من نفسِه.**
//
// # ودافعان لا ثالث
//
//	المتجرُ    ⇐ **من محفظته عند قبول المكتب** (لا عند الإنشاء — ٢٠٢٦-١٠-٠٥)،
//	             فإن لم تكفِ فدينٌ مضبوطٌ بسقفٍ إداريّ
//	المستلِمُ  ⇐ **نقداً بيد السائق** عند التسليم (`cash_due`)
//
// **ولا محفظةَ سالبةَ أبداً** (قرارُ المالك ٢) — **والدينُ مقيسٌ في
// `financial_obligations` القائم**، لا رصيدٌ ينزلق تحت الصفر بصمت.
//
// # والسقفُ صفرٌ افتراضاً
//
// **فالمتجرُ الجديدُ لا يُدين المنصّةَ بحرف** — ادفع من محفظتك أو لا
// توصيلة. **والإدارةُ وحدَها ترفعه لمتجرٍ بعينه.**

import (
	"context"
	"errors"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// ErrDeliveryCreditExhausted **بلغ المتجرُ سقفَ دينه.**
//
// **ونصٌّ يقول ما يُفعل** — «تعذّر» وحدَها تجعله يعيد المحاولة عشراً.
var ErrDeliveryCreditExhausted = httpx.NewError(http.StatusPaymentRequired,
	"delivery_credit_exhausted", "errors.delivery_credit_exhausted")

// ErrMerchantDeliveryUnpaid **المكتبُ يقبل توصيلةً لا يقدر متجرُها على أجرتها**
// (قرارُ المالك ٢٠٢٦-١٠-٠٥): **يُردّ القبولُ** وتبقى معلَّقةً — يرفضها المكتبُ
// أو ينتظر شحنَ المحفظة.
var ErrMerchantDeliveryUnpaid = httpx.NewError(http.StatusPaymentRequired,
	"merchant_delivery_unpaid", "errors.merchant_delivery_unpaid")

// chargeMerchantDelivery **يُحصّل أجرةَ التوصيلة من المتجر** — محفظةً أو ديناً.
//
// # لماذا داخلَ المعاملة
//
// **قراءةُ الرصيد ثمّ الخصمُ بلا قفلٍ تمرّان معاً**: توصيلتان في اللحظة
// نفسِها تقرآن الرصيدَ كافياً فتخصمان مرّتين، **فيهبط تحت الصفر** — وهو ما
// يمنعه القرار. **فالقراءةُ والكتابةُ في معاملةٍ واحدة.**
//
// # والترتيبُ كما أمر المالك
//
//	١ · **المحفظةُ أوّلاً** — ما دام فيها ما يكفي فلا دينَ يُكتب.
//	٢ · **فإن لم تكفِ: دينٌ بكامل الأجرة** — لا نصفٌ من هنا ونصفٌ من هناك.
//	٣ · **والدينُ يُردّ إن تجاوز السقف** — ولا توصيلة.
//
// **ولا يُخلَط المصدران في توصيلةٍ واحدة**: مبلغٌ نصفُه خصمٌ ونصفُه دينٌ
// **يحتاج عكسَين مختلفَين عند الإلغاء**، **وأوّلُ إلغاءٍ يكشف أنّ أحدَهما
// نُسي.**
func (s *Service) chargeMerchantDelivery(ctx context.Context, q wallet.Querier,
	orderID, merchantID, actorID string, fee int64) error {
	if fee <= 0 {
		return nil
	}
	// **والقبولُ التلقائيُّ بلا فاعل** (`sweepAutoAccept`) — **والفارغُ ليس
	// `uuid`** فيُسقط القيدَ والمعاملةَ معه. **فيُكتب `NULL`**: لا إنسانَ فعلها.
	var actor *string
	if actorID != "" {
		actor = &actorID
	}
	var ownerID string
	if err := q.QueryRow(ctx,
		`SELECT owner_user_id::text FROM merchants WHERE id = $1`, merchantID).
		Scan(&ownerID); err != nil {
		return err
	}

	// **والرصيدُ يُقرأ بقفلِ صفِّه** — انظر أعلاه.
	var balance int64
	if err := q.QueryRow(ctx,
		`SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE`, ownerID).
		Scan(&balance); err != nil {
		// **ولا محفظةَ بعد** — يُعامَل رصيداً صفراً فيُحاوَل الدين.
		balance = 0
	}

	if balance >= fee {
		// **خصمٌ من محفظة المتجر** — ونوعُه يقول لماذا نقص الرصيد
		// (قرارُ المالك ٢٠٢٦-٠٨-١١: «الرصيد يتغيّر وما حدا بيعرف ليش»).
		if _, err := s.wallet.ApplyTx(ctx, q, ownerID, -fee, "order_payment",
			orderID, "أجرةُ توصيلةٍ من متجرك", actor); err != nil {
			return err
		}
		// ══════════════════════════════════════════════════════════════
		// **ويُكتب على الطلب أنّه دُفع** — وإلّا موّلت الخزينةُ السائقَ
		// ══════════════════════════════════════════════════════════════
		//
		// **`creditTreasury` تقرأ `wallet_paid` و`cash_due` من الصفّ**،
		// وتُخرج البقيّةَ (مدفوعٌ − مستردٌّ − ما وصل الأطراف).
		//
		// **فلو بقي `wallet_paid` صفراً والمالُ خرج من محفظة المتجر
		// فعلاً**: يُقرأ المدفوعُ صفراً، **فتصير البقيّةُ سالبةً بمقدار
		// أجر السائق** — **فتُقيَّد الخزينةُ خسارةً وهي التي كسبت.**
		//
		// **والخصمُ وحدَه لا يُخبر الدفترَ** — الدفترُ يقرأ الطلبَ لا
		// المحفظة. **فيُكتب هنا حيث وقع الخصم، لا في مكانٍ ثانٍ يُنسى.**
		if _, err := q.Exec(ctx,
			`UPDATE orders SET wallet_paid = $2 WHERE id = $1`, orderID, fee); err != nil {
			return err
		}
		return nil
	}

	// ── الدين — بسقفٍ لا بلا حدّ ───────────────────────────────────────
	var limit int64
	if err := q.QueryRow(ctx,
		`SELECT delivery_credit_limit FROM merchants WHERE id = $1`, merchantID).
		Scan(&limit); err != nil {
		return err
	}
	owed, err := obligations.Balance(ctx, q, obligations.PartyMerchant, merchantID)
	if err != nil {
		return err
	}
	// **والقائمُ والجديدُ يُقاسان معاً** — **ومن قاس الجديدَ وحدَه سمح
	// بألفٍ مرّةً بعد مرّةٍ وسقفُه ألف.**
	if owed+fee > limit {
		return ErrDeliveryCreditExhausted
	}
	if _, err := obligations.Create(ctx, q, obligations.PartyMerchant, merchantID,
		fee, causeMerchantDeliveryFee, orderID, actor); err != nil {
		return err
	}
	return nil
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
//	التسليم          ⇐ نقدُ المستلِم إلى صندوق السائق (إن دفع هو) · أجرُ السائق ·
//	                   والبقيّةُ للخزينة (`creditTreasury`)
//	نهايةٌ بلا تسليم ⇐ **يُعاد المالُ لمن دفعه**: محفظةُ المتجر إن خُصمت، أو
//	                   يُسقَط الدينُ إن قُيّد. **ولا أجرَ لسائقٍ لم يُسلّم.**
func (s *Service) settleMerchantDelivery(ctx context.Context, q wallet.Querier,
	in settlement, out *settled) error {
	switch {
	// ══════════════════════════════════════════════════════════════════
	// **قبولُ المكتب — وفيه وحدَه تُخصم أجرةُ «المتجر يدفع»** (٢٠٢٦-١٠-٠٥)
	// ══════════════════════════════════════════════════════════════════
	//
	// **في معاملة القبول نفسِها**: إن لم تكفِ المحفظةُ ولا السقفُ سقط القبولُ
	// كلُّه وبقيت التوصيلةُ معلَّقةً بيد المكتب — **فلا تنزل الطابورَ توصيلةٌ
	// لم يُدفع أجرُها.** والنقدُ (المستلِمُ أو «أنا نقداً») لا شيءَ يُخصم له.
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

	case in.to == StDelivered:
		if in.cashDue > 0 && in.driverID != nil {
			if err := s.cashbox.CollectTx(ctx, q, *in.driverID, in.cashDue, in.orderID, &in.actorID); err != nil {
				return err
			}
		}
		if err := s.payDriver(ctx, q, in, out); err != nil {
			return err
		}
		return s.creditTreasury(ctx, q, in.orderID, in.actorID)

	case refundOnEnter(in.to) && in.from != StDelivered:
		var merchantID, ownerID string
		if err := q.QueryRow(ctx, `
			SELECT o.merchant_id::text, m.owner_user_id::text
			  FROM orders o JOIN merchants m ON m.id = o.merchant_id
			 WHERE o.id = $1`, in.orderID).Scan(&merchantID, &ownerID); err != nil {
			return err
		}
		if in.walletPaid > 0 {
			if _, err := s.wallet.ApplyTx(ctx, q, ownerID, in.walletPaid, "refund",
				in.orderID, "استرجاعُ أجرةِ توصيلةٍ لم تُسلَّم", &in.actorID); err != nil {
				return err
			}
			out.credit(ownerID, in.walletPaid, t.refunded2, notifications.AppMerchant)
		}
		// **والدينُ يُسقَط كما يُردّ المال** — دينٌ على توصيلةٍ لم تقع باطل.
		if err := obligations.VoidForOrder(ctx, q, obligations.PartyMerchant, merchantID,
			in.orderID, causeMerchantDeliveryFee, &in.actorID); err != nil {
			return err
		}
		// ══════════════════════════════════════════════════════════════
		// **وفشلُها بعد الاستلام يُعوَّض فيه السائقُ من المنصّة** (قرارُ المالك
		// مساءَ ٢٠٢٦-١٠-٠٢، البند ٨: «المنصّة تدفع طبعاً»)
		// ══════════════════════════════════════════════════════════════
		//
		// **كان هذا الفرعُ بلا تعويضٍ أصلاً** — فسائقٌ قاد التوصيلةَ إلى بابٍ لم
		// يستلم يعود بلا شيء. **وأجرُها على المتجر حين تُسلَّم**، فإن لم تُسلَّم
		// **فالمنصّةُ تعوّض** — بطلبٍ معلَّقٍ كغيره (`compensation_requests.go`)
		// يوافق عليه إنسان، **بذنبٍ «المنصّة»** فلا تُفتح مطالبةٌ على المتجر.
		// **وذنبُ السائق لا تعويضَ فيه** — كما في كلّ طلب.
		if in.to == StFailed && in.driverID != nil && pastPickup[in.from] {
			var fault, reason string
			if err := q.QueryRow(ctx,
				`SELECT COALESCE(fault, ''), COALESCE(fail_reason, '') FROM orders WHERE id = $1`,
				in.orderID).Scan(&fault, &reason); err != nil {
				return err
			}
			if fault != FaultDriver {
				created, err := s.requestDriverCompensation(ctx, q, in.orderID, in.driverID,
					FaultPlatform, reason, in.deliveryFee)
				if err != nil {
					return err
				}
				out.compensationRequested = out.compensationRequested || created
			}
		}
		return s.creditTreasury(ctx, q, in.orderID, in.actorID)
	}
	return nil
}

// causeMerchantDeliveryFee **سببُ دينِ أجرة التوصيلة** — موضعٌ واحدٌ للنصّ.
const causeMerchantDeliveryFee = "merchant_delivery_fee"

// MerchantDeliveryCredit **سقفُ دينِ متجرٍ ورصيدُه** — لشاشة الإدارة.
//
// **والرقمان يُقرآن معاً أو لا يُقرآن**: سقفٌ بلا رصيدٍ لا يقول أبقي شيء،
// **ورصيدٌ بلا سقفٍ لا يقول أقريبٌ من الحدّ.**
func (s *Service) MerchantDeliveryCredit(ctx context.Context, merchantID string) (limit, owed int64, err error) {
	if err = s.db.QueryRow(ctx,
		`SELECT delivery_credit_limit FROM merchants WHERE id = $1`, merchantID).
		Scan(&limit); err != nil {
		return 0, 0, err
	}
	owed, err = obligations.Balance(ctx, s.db, obligations.PartyMerchant, merchantID)
	return limit, owed, err
}
