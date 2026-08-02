package orders

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// Transition ينفّذ انتقال حالة بعد التحقق من شرعيته للأدوار الفاعلة،
// ويسجل الحدث، ويطلق التسويات (استرجاع المحفظة، تحرير كود الخصم) عند الإغلاق.
// Transition ينقل الطلبَ في مساره.
//
// `failReason` رمزُ سببٍ من `FailReasons` — يلزم عند `failed` ومنه يُشتقّ
// الذنبُ الذي يقرّر التعويض. **ويُهمَل في غيرها.**
func (s *Service) Transition(ctx context.Context, actorID string, actorRoles []string, orderID, to, note string) (*Order, error) {
	return s.TransitionWithReason(ctx, actorID, actorRoles, orderID, to, note, "")
}

// TransitionWithReason كالسابقة، ومعها سببُ التعذّر المُصنَّف.
func (s *Service) TransitionWithReason(ctx context.Context, actorID string, actorRoles []string, orderID, to, note, failReason string) (*Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var from string
	var driverID *string
	var walletPaid, cashDue, deliveryFee int64
	var customerID, promoCode string
	err = tx.QueryRow(ctx, `
		SELECT status, driver_id, wallet_paid, cash_due, delivery_fee, customer_id,
		       COALESCE(promo_code,'')
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&from, &driverID, &walletPaid, &cashDue, &deliveryFee, &customerID, &promoCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// **الوضعُ يُنقّي الأدوارَ ثم تُسأل الخريطة** — فحكمُ السياسة في الخادم
	// لا في الشاشة. (انظر modes.go)
	//
	// وبلا مخزن إعدادات يُفترض «المتجر يدير»: هو الأصل، **والافتراضُ عند
	// الجهل يجب أن يكون أقلَّ الوضعين تدخّلاً من المنصة**.
	selfManage := s.settings == nil ||
		s.settings.GetBool(ctx, "merchants.self_manage_orders")
	effRoles := rolesUnderMode(selfManage, from, to, actorRoles, driverID != nil)
	if !canTransition(from, to, effRoles) {
		return nil, ErrBadTransition
	}
	endedBy := authorizingRole(from, to, effRoles)
	// لا استلام بلا سائق مسند
	if (to == StAtPickup || to == StPickedUp) && driverID == nil {
		return nil, ErrNeedsDriver
	}

	set := `status = $2, updated_at = now()`
	switch to {
	case StAccepted:
		set += `, accepted_at = now()`
	case StPickedUp:
		set += `, picked_up_at = now()`
	case StDelivered:
		set += `, delivered_at = now(), closed_at = now()`
	case StDispatching:
		// **متى نزل إلى الطابور** — ومنه يُقاس انتظارُ الإسناد.
		//
		// **ويُعاد كتابتُه عند فكّ الإسناد**: طلبٌ أخذه سائقٌ ثمّ تركه **عاد
		// إلى أوّل الصفّ لا إلى وسطه** — فمهلةُ انتظاره تبدأ من جديد، ولا
		// يُنبَّه عنه فوراً لأن أوّلَ نزولٍ له كان قبل ساعة.
		set += `, dispatched_at = now()`
		// **فكُّ الإسناد من أيّ موضعٍ قبل الاستلام** — لا من `assigned` وحدَها.
		//
		// ولو بقي الشرطُ على الأولى **لَعاد الطلبُ إلى الطابور وسائقُه ملتصقٌ
		// به**: يظهر للجميع مسنَداً فلا يأخذه أحد، ويبقى في سقف الأوّل النقديِّ
		// وهو لا يعمل عليه. **طلبٌ في الطابور وله سائق أسوأُ من طلبٍ لا سائق
		// له** — ذاك يُرى فارغاً فيُلتقط، وهذا يُرى مأخوذاً فيُترك.
		// **وكلُّ عودةٍ إلى الطابور تُصفّي سائقَها** — لا الأولَيَين وحدَهما.
		//
		// وقد صار للطلب مخرجٌ بعد الاستلام (طارئُ السائق، أو اختفاؤه)، **ولو
		// بقي الشرطُ على حالِه لَعاد إلى الطابور وسائقُه الغائبُ ملتصقٌ به.**
		if from != StPreparing && from != StAccepted {
			set += `, driver_id = NULL`
		}
	}
	if terminal(to) && to != StDelivered {
		set += `, closed_at = now(), cancel_reason = ` + "$3"
	}

	args := []any{orderID, to}
	if terminal(to) && to != StDelivered {
		args = append(args, note)
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET `+set+` WHERE id = $1`, args...); err != nil {
		return nil, err
	}

	// **سببُ التعذّر وذنبُه** — يُكتبان قبل التسوية لأن التعويضَ يقرأهما.
	//
	// **والذنبُ من القائمة لا من تقدير أحد**: كلُّ سببٍ يحمل ذنبَه
	// (`failreasons.go`)، **فلا يُترك حكمٌ ماليٌّ لاجتهادٍ في لحظة.**
	if to == StFailed && failReason != "" {
		if _, err := tx.Exec(ctx,
			`UPDATE orders SET fail_reason = $2, fault = NULLIF($3, '') WHERE id = $1`,
			orderID, failReason, FaultOf(failReason)); err != nil {
			return nil, err
		}
	}

	// **من أنهى الطلب** — يُسجَّل مع الإغلاق ويُقرأ في عدّ المخالفات.
	//
	// وبتحديثٍ ثانٍ لا بحشره في الأوّل: الأوّلُ يبني نصَّه بالتركيب، **ودمجُ
	// قيمةٍ آتيةٍ من رمز الدخول في نصٍّ يُركَّب بابُ حقنٍ لا داعيَ له** — والصفُّ
	// مقفولٌ في المعاملة نفسها فلا يراه أحدٌ بين التحديثين.
	if terminal(to) && endedBy != "" {
		if _, err := tx.Exec(ctx,
			`UPDATE orders SET ended_by = $2 WHERE id = $1`, orderID, endedBy); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, $2, $3, $4, $5)`, orderID, from, to, actorID, note); err != nil {
		return nil, err
	}

	// تحرير كود الخصم عند الإلغاء/الرفض/الفشل (لا عند refunded — الخدمة قُدمت)
	if (to == StCancelled || to == StRejected || to == StFailed) && promoCode != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE promo_codes SET used_count = greatest(used_count - 1, 0) WHERE code = $1`,
			promoCode); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM promo_redemptions WHERE order_id = $1`, orderID); err != nil {
			return nil, err
		}
	}

	// نافذة إلغاء الزبون بعد القبول — تُفرض هنا لا في الخارطة لأنها **زمنية**
	// لا دورية: الخارطة تقول من يملك الانتقال، والزمن يقول متى.
	if to == StCancelled && from == StAccepted && slices.Contains(actorRoles, "customer") &&
		!slices.Contains(actorRoles, "ops") && !slices.Contains(actorRoles, "admin") {
		// **الافتراضيُّ من الفهرس لا من الاستعلام.**
		//
		// كان `COALESCE(..., 120)` مكتوباً هنا **والفهرسُ يحمل افتراضَه أيضاً**
		// — رقمان لمعنًى واحد. ولو غيّر المالكُ الافتراضَ في الفهرس لبقي هذا
		// الاستعلامُ يعمل بالقديم عند غياب الصفّ. **وافتراضٌ في موضعين
		// افتراضٌ لا يُعتمد عليه.**
		var withinWindow bool
		if err := tx.QueryRow(ctx, `
			SELECT accepted_at IS NOT NULL
			   AND accepted_at > now() - make_interval(secs => $2)
			FROM orders WHERE id = $1`,
			orderID, s.cancelWindowSec(ctx)).Scan(&withinWindow); err != nil {
			return nil, err
		}
		if !withinWindow {
			return nil, ErrCancelWindowPassed
		}
	}

	// كل التسويات المالية **داخل** معاملة الانتقال: إمّا تتم الحالة والمال معاً
	// أو لا يتم شيء. كانت تُنفَّذ بعد الإيداع وأخطاؤها تُبتلع في السجل، فيصير
	// الطلب مُسلَّماً بلا عمولة ولا نقد مقيَّد — خلل مالي صامت لا أثر له.
	var done settled
	if err := s.settle(ctx, tx, settlement{
		orderID: orderID, from: from, to: to, actorID: actorID,
		customerID: customerID, driverID: driverID,
		walletPaid: walletPaid, cashDue: cashDue, deliveryFee: deliveryFee,
	}, &done); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	updated, err := s.GetByID(ctx, orderID)
	if err != nil {
		return updated, err
	}
	s.publishOrder(updated)
	// بعد الإيداع: فشل الإشعار لا يُبطل تسليماً وقع فعلاً
	s.notifyTransition(ctx, orderID, to, note, endedBy)
	s.notifyCommission(ctx, done.repID, orderID, done.commissionPaid)

	// **أوّلُ عرضٍ في نمط «بالترتيب»** — لحظةَ نزول الطلب إلى الطابور.
	//
	// وبعد الإيداع: العرضُ ترتيبٌ لا مال، **وتعثّرُه يترك الطلبَ مشاعاً للجميع
	// لا يُلغي نزولَه.**
	if to == StDispatching {
		// **ومن حرّره لا يُعرض عليه ثانيةً في اللحظة نفسها.**
		//
		// كان يُمرَّر `nil` دائماً — **فالسائقُ الذي فكّ إسنادَه يجده أمامه من
		// جديد قبل أن يُغلق الشاشة**: يرفضه فيعود، ويرفضه فيعود. **ودورةٌ لا
		// تتقدّم أسوأُ من طابورٍ لا يُخدَم** — هذه تُشغل الطلبَ وتوهم أنّه
		// يُعرض.
		//
		// **ولا يُحرم منه أبداً**: الاستثناءُ لهذه الجولة وحدَها، فإن دار
		// الطابورُ ولم يأخذه أحد عاد إليه مع الجميع.
		var skip []string
		if driverID != nil && (from == StAssigned || from == StAtPickup) {
			skip = []string{*driverID}
		}
		if err := s.OfferNext(ctx, orderID, skip); err != nil {
			s.logger.Error("الترتيب: تعذّر أوّل عرض", "order", orderID, "error", err)
		}
	}

	// **حظرُ المتجر كثيرِ الإلغاء** — بعد الإيداع لا داخله.
	//
	// الحظرُ قرارٌ قائمٌ بذاته، وتعثّرُه يجب ألّا يُلغي إلغاءً وقع فعلاً:
	// **وطلبٌ أُلغي ثم رُدَّ إلغاؤه لأن الحظر تعثّر يترك الزبونَ ينتظر طعاماً
	// لن يأتي.**
	if endedBy == "merchant" && (to == StCancelled || to == StRejected) {
		s.enforceMerchantViolations(ctx, orderID)
	}

	// **وامتناعُ المتجر يُنذَر كما يُنذَر إلغاؤه.**
	//
	// كان الفشلُ بذنبه يمرّ بلا أثر: إشعارٌ يُقرأ ويُنسى، **ولا عدٌّ ولا سجلّ.**
	// فمن أغلق بابَه عشر مرّاتٍ والسائقُ عنده بقي بلا مخالفةٍ واحدة.
	if to == StFailed && failReason != "" {
		s.warnMerchantOnFault(ctx, orderID, FaultOf(failReason), failReason)
	}

	// **الإنزال التلقائيّ إلى طابور السائقين.**
	//
	// **خارج المعاملة عمداً، وبعد بثّ الأوّل وإشعاره**: هو انتقالٌ ثانٍ قائمٌ
	// بذاته له تسوياتُه وإشعاراتُه. وحشرُه في معاملة الأوّل يجعل تعثّرَه يُلغي
	// انتقالاً وقع فعلاً — **وطلبٌ قُبل ثم رُدَّ قبولُه لأن الطابور تعثّر أسوأ
	// من طلبٍ ينتظر ضغطةً يدوية.**
	//
	// فإن تعثّر: يبقى الطلبُ حيث هو، وتراه العملياتُ بزرّ «طلب سائق» كما كان.
	// **تعطُّلُ الأتمتة يعيدنا إلى اليد، لا إلى لا شيء.**
	if s.autoDispatch(ctx, to) {
		dispatched, derr := s.Transition(ctx, actorID, []string{"ops"},
			orderID, StDispatching, autoDispatchNote)
		if derr != nil {
			s.logger.Warn("الإنزال التلقائي تعثّر — الطلب ينتظر إسناداً يدوياً",
				"order", orderID, "from", to, "error", derr)
		} else {
			return dispatched, nil
		}
	}
	return updated, nil
}

// autoDispatchNote يُكتب في أثر الأحداث — **فيُعرف أن الآلة نقلته لا الموظّف.**
const autoDispatchNote = "إنزالٌ تلقائيّ إلى طابور السائقين"

// autoDispatch أيُنزَّل الطلبُ إلى الطابور بعد هذا الانتقال؟
//
// **ومن أيّ حالةٍ يُنزَّل يتبع من يدير الطلبات:**
//
//   - **المتجر يدير**: من `preparing` — وهنا. المتجرُ قال «بدأتُ الطبخ»،
//     فيُنادى السائق ليصل مع الجاهزية لا بعدها بربع ساعة.
//   - **المنصة تدير**: **بعد إبلاغ المتجر بالرسالة** — لا هنا، بل في
//     `server.handleSendOrderToMerchant`. **فالقبولُ وحده لا يعني أن المطعم
//     يعلم**، واستدعاءُ سائقٍ قبل أن يعلم يرسله إلى بابٍ لم يُطبخ خلفه شيء.
//
// ولا يُنزَّل من `accepted` هنا البتّة: في وضع «المتجر يدير» يعني ذلك أن
// السائق يسبق الطبخَ فيقف عند الباب، وفي وضع «المنصة تدير» يسبق الإبلاغ.
func (s *Service) autoDispatch(ctx context.Context, to string) bool {
	if s.settings == nil || to != StPreparing {
		return false
	}
	return s.settings.GetBool(ctx, "orders.auto_dispatch") &&
		s.settings.GetBool(ctx, "merchants.self_manage_orders")
}

// AutoDispatchEnabled أمُشغَّلٌ الإنزالُ التلقائيّ؟ يسأله الإبلاغُ بالرسالة.
func (s *Service) AutoDispatchEnabled(ctx context.Context) bool {
	return s.settings != nil && s.settings.GetBool(ctx, "orders.auto_dispatch")
}

// AutoDispatch ينزل الطلبَ إلى الطابور باسم النظام — يُنادى بعد إبلاغ المتجر.
func (s *Service) AutoDispatch(ctx context.Context, actorID, orderID string) error {
	_, err := s.Transition(ctx, actorID, []string{"ops"}, orderID,
		StDispatching, autoDispatchNote)
	return err
}

// compensateDriverOnFail يعوّض السائقَ عن مشوارٍ لم يُثمر — **بلا يد**.
//
// # ولماذا نسبةٌ من رسم التوصيل
//
// **الثابتُ يظلم طرفاً حتماً**: خمسةُ آلافٍ كثيرةٌ على مشوارٍ في الحيّ وقليلةٌ
// على مشوارٍ عبر المدينة. **والنسبةُ تتبع المسافةَ لأن رسم التوصيل يتبعها.**
//
// # ولماذا نصفٌ لا كلّ
//
// **لا يُعدل أن تتحمّل المنصةُ الخسارةَ وحدها** — وقد خسرت بضاعةَ المتجر
// أصلاً. **والنصفُ يقسم ما لا ذنبَ لأحدٍ منهما فيه.**
//
// # ويخرج من الخزينة في القيد نفسه
//
// **تعويضٌ يُقيَّد للسائق وحده يجعل المنصةَ تظهر رابحةً وهي تدفع.**
func (s *Service) compensateDriverOnFail(ctx context.Context, q wallet.Querier, in settlement) error {
	if in.driverID == nil || in.deliveryFee <= 0 || s.settings == nil {
		return nil
	}
	var fault string
	if err := q.QueryRow(ctx,
		`SELECT COALESCE(fault, '') FROM orders WHERE id = $1`, in.orderID).
		Scan(&fault); err != nil {
		return err
	}
	if fault != FaultCustomer {
		return nil
	}

	pct := s.settings.GetInt(ctx, "drivers.failed_compensation_percent")
	if pct <= 0 {
		return nil
	}
	amount := in.deliveryFee * pct / 100
	if amount <= 0 {
		return nil
	}
	if _, err := s.wallet.ApplyTx(ctx, q, *in.driverID, amount, "compensation",
		in.orderID, "تعويضٌ عن تعذّر التسليم — الحقُّ على الزبون", &in.actorID); err != nil {
		return err
	}
	return s.DebitTreasury(ctx, q, amount, in.orderID,
		"تعويضُ سائقٍ عن تعذّر تسليم", in.actorID)
}

// pastPickup حالاتٌ صار الطعامُ فيها بيد السائق — والمتجرُ قبض ثمنَه.
var pastPickup = map[string]bool{
	StPickedUp: true, StOnTheWay: true, StAtDropoff: true,
}

// settled ما وقع فعلاً من تسويات — يُملأ داخل المعاملة ويُقرأ بعد نجاحها
// لإطلاق الإشعارات. متغيّر محلي لكل طلب: الخدمة مشتركة بين كل الطلبات المتزامنة
// فلا يجوز أن تحمل حالة طلب بعينه.
type settled struct {
	repID          string
	commissionPaid int64
}

// settlement مدخلات التسوية المالية لانتقال واحد.
type settlement struct {
	orderID, from, to, actorID, customerID string
	driverID                               *string
	walletPaid, cashDue, deliveryFee       int64
}

// settle ينفّذ كل الأثر المالي لانتقال الحالة داخل معاملة المستدعي.
//
// القاعدة المحاسبية المعتمدة — الاسترجاع يعكس ما قيّده التسليم بالضبط:
//   - نهاية غير التسليم **قبل** التسليم: يُعاد المدفوع من المحفظة فقط
//     (النقد لم يُحصَّل أصلاً).
//   - استرجاع **بعد** التسليم: يُعاد **كامل المبلغ** إلى محفظة الزبون — لأنه
//     دفع النقد فعلاً للسائق — وتُعكس عمولة المندوب وتُصفَّر عمولة المنصة.
//     صندوق السائق يبقى كما هو عن قصد: النقد الذي قبضه ما زال بحوزته ويدين به
//     للمنصة، والمنصة هي من ردّت للزبون. هكذا يتوازن الطرفان بلا رصيد سالب.
func (s *Service) settle(ctx context.Context, q wallet.Querier, in settlement, out *settled) error {
	// (1) نهاية غير التسليم قبل التسليم — استرجاع ما دُفع من المحفظة
	if refundOnEnter(in.to) && in.from != StDelivered && in.walletPaid > 0 {
		if _, err := s.wallet.ApplyTx(ctx, q, in.customerID, in.walletPaid, "refund",
			in.orderID, fmt.Sprintf("استرجاع طلب (%s)", in.to), &in.actorID); err != nil {
			return err
		}
	}

	// (2) **الاستلامُ من المتجر — وهنا يقبض المتجر.**
	//
	// **المتجرُ ليس طرفاً في التوصيل**: باع وسلّم وانتهى، وما يجري بعد ذلك
	// بين المنصة والسائق والزبون لا يخصّه. **فمستحقُّه لحظةَ خروج البضاعة من
	// يده لا لحظةَ وصولها.**
	//
	// وأثرُه أن **الخزينةَ تهبط تحت الصفر بين الاستلام والتسليم** — دفعت ولم
	// تقبض. **والسالبُ هناك ليس خطأً، هو الواقع**: مقدارُه ما في يد المتجر
	// من مال المنصة.
	//
	// **وهذا يُسقط سؤالاً كنّا نبنيه**: «أاستردّ المتجرُ بضاعتَه أم تتحمّلها
	// المنصة؟» — فالمنصةُ **اشترت** الطعامَ لحظةَ خروجه، **وهو ملكُها**،
	// والخسارةُ تقع تلقائياً حيث يجب بلا قرارٍ من أحد.
	if in.to == StPickedUp {
		if err := s.settleMerchant(ctx, q, in.orderID, in.actorID); err != nil {
			return err
		}
		return s.creditTreasury(ctx, q, in.orderID, in.actorID)
	}

	// (3) التسليم — تحصيل النقد وأجرُ السائق ونصيبُ المندوب
	//
	// **ولا مستحقَّ متجرٍ هنا**: قُيّد عند الاستلام. ولو أُعيد لَقُيّد مرّتين.
	if in.to == StDelivered {
		if in.cashDue > 0 && in.driverID != nil {
			if err := s.cashbox.CollectTx(ctx, q, *in.driverID, in.cashDue, in.orderID, &in.actorID); err != nil {
				return err
			}
		}
		// **احتياطٌ لا تكرار**: تُهمل إن قُيّد المتجرُ عند الاستلام، وتُدرك
		// ما فات إن قُفز فوقه.
		if err := s.settleMerchant(ctx, q, in.orderID, in.actorID); err != nil {
			return err
		}
		if err := s.payDriver(ctx, q, in); err != nil {
			return err
		}
		if err := s.settleRep(ctx, q, in.orderID, in.actorID, out); err != nil {
			return err
		}
		// **الطرفُ الرابع** — بعد أن تُقيَّد أنصبةُ الجميع، فيقرأ ما وقع
		// لا ما نُوي. (treasury.go)
		return s.creditTreasury(ctx, q, in.orderID, in.actorID)
	}

	// (4) تعذّرُ التسليم — **تعويضُ السائق تلقائياً حين يكون الحقُّ على الزبون**
	//
	// **بلا يد** (قرار المالك). ولو تُرك لتقديرٍ لاحق **لَصار قاعدةً تُنفَّذ
	// بيدٍ — وقاعدةٌ تُنفَّذ بيدٍ ليست قاعدة، هي عادة.**
	//
	// **وذنبُ السائق لا تعويضَ فيه**، وذنبُ المتجر كذلك: المنصةُ تتحمّل
	// بضاعتَه وتعوّض سائقَها، **ولا تجمع عليها الاثنين بلا سبب**.
	if in.to == StFailed {
		if err := s.compensateDriverOnFail(ctx, q, in); err != nil {
			return err
		}
	}

	// (5) نهايةٌ فاشلةٌ بعد الاستلام — **الخسارةُ تُقيَّد بلا قرارٍ من أحد**
	//
	// المتجرُ قبض عند الاستلام والزبونُ لم يدفع (أو رُدّ له). **فالفرقُ على
	// المنصة** — وتقيّده الخزينةُ وحدها حين تُعيد الحساب.
	if refundOnEnter(in.to) && in.from != StDelivered && pastPickup[in.from] {
		return s.creditTreasury(ctx, q, in.orderID, in.actorID)
	}
	// وقبل الاستلام: لا مالَ تحرّك، فلا خزينةَ تُحدَّث — **إلّا إن عُوّض سائق.**
	if in.to == StFailed {
		return s.creditTreasury(ctx, q, in.orderID, in.actorID)
	}

	// (6) استرجاع بعد التسليم — عكس كل ما سبق
	if refundOnEnter(in.to) && in.from == StDelivered {
		if total := in.walletPaid + in.cashDue; total > 0 {
			if _, err := s.wallet.ApplyTx(ctx, q, in.customerID, total, "refund",
				in.orderID, "استرجاع طلب مُسلَّم", &in.actorID); err != nil {
				return err
			}
		}
		if err := s.reverseCommissions(ctx, q, in.orderID, in.actorID); err != nil {
			return err
		}
		// **بعد عكس الأنصبة لا قبله**: الخزينةُ تقرأ ما بقي مقيَّداً، فلو
		// قُرئت قبل العكس لحسبت أنصبةً ستُلغى بعد سطر.
		return s.creditTreasury(ctx, q, in.orderID, in.actorID)
	}
	return nil
}

// settleMerchant يقيّد مستحقَّ المتجر وعمولةَ المنصة — **عند الاستلام**.
//
// **المتجرُ ليس طرفاً في التوصيل**: باع وسلّم وانتهى. فمستحقُّه لحظةَ خروج
// البضاعة من يده، **وما يجري بعدها لا يخصّه** — لا فشلُ تسليمٍ ولا رفضُ زبون.
//
// **والعمولةُ معه**: هي تكلفةُ بيعته، والبيعةُ وقعت. (السياسة §٣-١)
func (s *Service) settleMerchant(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	// **لا يُقيَّد مرّتين.**
	//
	// تُنادى عند الاستلام، **وتُنادى ثانيةً عند التسليم احتياطاً**: الخريطةُ
	// تمنع بلوغَ التسليم بلا استلام، **لكنّ تجاوزَ أدمنٍ أو إصلاحَ بياناتٍ قد
	// يقفز فوقه** — وحينها يبقى المتجرُ بلا مستحقّ وعمولةُ المنصة صفراً،
	// **فيُقرأ الطلبُ ربحاً كاملاً وهو لم يُدفع ثمنُه.**
	//
	// **ومنعُ التكرار بالدفتر لا بالحالة**: يُسأل عن قيدٍ وقع، لا عن حالةٍ
	// يُظنّ أنها مرّت.
	var already bool
	if err := q.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM wallet_transactions
		               WHERE ref = $1 AND kind = 'merchant_earning')`,
		orderID).Scan(&already); err != nil {
		return err
	}
	if already {
		return nil
	}

	// **الأساسُ سعرُ الشراء لا سعرُ البيع.**
	//
	// `o.subtotal` صار **ما يدفعه الزبون** — سعرَ الشراء زائدَ هامشِنا.
	// **والمتجرُ لا يبيع بذلك السعر ولا يراه**، فمستحقُّه منه يعطيه هامشَنا،
	// **وعمولتُنا عليه تخصم منه على مالٍ لم يقبضه.**
	//
	// **وعمولةٌ على سعرٍ لا يراه المتجرُ عمولةٌ لا يفهمها**: يُقال له «٢٪»
	// فيحسبها على رقمه، فإن حُسبت على رقمنا **وجد خصماً لم يتوقّعه ولم
	// يُخبَر به** — والثقةُ تنكسر هكذا لا بالهامش المُعلَن.
	//
	// و`merchant_price` لقطةٌ في بند الطلب: **تكلفةُ الأمس تُقرأ كما كانت.**
	// **ولكلِّ مصدرٍ مستحقُّه وعمولتُه.**
	//
	// كان الطلبُ من مصدرٍ واحد فيُقرأ `orders.merchant_id`. **وبعد مصدرين صار
	// ذلك يدفع لصاحب المحطّة الأولى ثمنَ بضاعةِ الثاني** — فيربح من لم يبع،
	// **ويُحرم من باع.**
	//
	// **والعمولةُ لكلِّ متجرٍ بنسبته**: متجرٌ اتُّفق معه على ٢٪ وآخرُ على ٥٪
	// **لا تجمعهما نسبةٌ واحدة**، ونسبةُ صاحب المحطّة الأولى ليست عقداً على
	// غيره.
	//
	// وتُجمع البنودُ بمصدرها من **لقطةِ البند** لا من `menu_items` اليوم:
	// **يُنقل صنفٌ فتُعاد قراءةُ طلبات الأمس بمصدرٍ لم يحضّرها.**
	rows, err := q.Query(ctx, `
		SELECT COALESCE(oi.merchant_id, o.merchant_id)::text,
		       m.commission_percent, m.owner_user_id::text,
		       sum(oi.merchant_price * oi.qty)
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN merchants m ON m.id = COALESCE(oi.merchant_id, o.merchant_id)
		WHERE oi.order_id = $1
		GROUP BY 1, 2, 3`, orderID)
	if err != nil {
		return err
	}
	type share struct {
		ownerID    *string
		commission int64
		due        int64
	}
	shares := []share{}
	var totalCommission int64
	for rows.Next() {
		var merchantID string
		var pct int
		var owner *string
		var cost int64
		if err := rows.Scan(&merchantID, &pct, &owner, &cost); err != nil {
			rows.Close()
			return err
		}
		c := cost * int64(pct) / 100
		totalCommission += c
		shares = append(shares, share{ownerID: owner, commission: c, due: cost - c})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// **وطلبٌ بلا بنود يقع على القديم** — طلباتُ ما قبل السعرين، أو ما أُنشئ
	// بلا `order_items`. **وصفرٌ هنا يعني «لا مستحقّ» وهو أسوأُ من الخطأ.**
	if len(shares) == 0 {
		var subtotal int64
		var pct int
		var owner *string
		if err := q.QueryRow(ctx, `
			SELECT o.subtotal, m.commission_percent, m.owner_user_id::text
			FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE o.id = $1`, orderID).Scan(&subtotal, &pct, &owner); err != nil {
			return err
		}
		c := subtotal * int64(pct) / 100
		totalCommission = c
		shares = append(shares, share{ownerID: owner, commission: c, due: subtotal - c})
	}

	if _, err := q.Exec(ctx,
		`UPDATE orders SET platform_commission = $2 WHERE id = $1`,
		orderID, totalCommission); err != nil {
		return err
	}

	// مستحقّ كلِّ متجر: **سعرُ شرائه ناقصَ عمولته** — لا سعرُ البيع.
	//
	// **لا `total`**: رسم التوصيل أجرُ خدمةٍ تؤدّيها المنصة بسائقها فليس من
	// نصيبه — والعمولة نفسها محسوبة على بضاعته، فالأساسان متسقان.
	for _, sh := range shares {
		if sh.ownerID == nil || sh.due <= 0 {
			continue
		}
		if _, err := s.wallet.ApplyTx(ctx, q, *sh.ownerID, sh.due, "merchant_earning",
			orderID, "مستحقّ عن بضاعةٍ سُلّمت للسائق", &actorID); err != nil {
			return err
		}
	}
	return nil
}

// settleRep يقيّد نصيبَ المندوب — **عند التسليم**.
//
// **وعند الفشل لا يُقيَّد شيء**: عمولتُه على طلبٍ وصل، لا على طلبٍ خرج من
// المطبخ. (وهو قرارُ المالك: «عند الفشل يُشطب ويُكتب ملغي».)
func (s *Service) settleRep(ctx context.Context, q wallet.Querier, orderID, actorID string, out *settled) error {
	var platformCommission int64
	var repID *string
	var repIsBuyer bool
	if err := q.QueryRow(ctx, `
		SELECT o.platform_commission, m.sales_rep_user_id,
		       m.sales_rep_user_id = o.customer_id
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1`, orderID).Scan(&platformCommission, &repID, &repIsBuyer); err != nil {
		return err
	}
	if platformCommission == 0 || repID == nil {
		return nil
	}

	// المندوب لا يقبض عمولةً على شرائه هو.
	//
	// العمولة أُنشئت لتكافئ **جلب الزبائن**، وشراءُ المندوب من متجره ليس ترويجاً
	// بل استهلاك. دفعُها له يعني مالاً يخرج من المنصة بلا قيمة مقابلة، ويضخّم
	// أرقامه فتصير مقاييس أدائه تقيس إنفاقه لا عمله.
	//
	// وعمولة المنصة تبقى كاملة: المتجر باع فعلاً ويدين بها — الملغى هو **نصيب
	// المندوب** منها لا العمولة نفسها.
	if repIsBuyer {
		return nil
	}
	// عتبة التفعيل: لا عمولة عن عميلٍ لم يُثبت أنه يعمل.
	activated, err := s.merchantActivated(ctx, q, orderID)
	if err != nil || !activated {
		return err
	}

	// **نصيبُه من الهامش لا من العمولة.**
	//
	// **الـ٢٪ مالٌ مرصودٌ لخسارة** — للطلبات التي تفشل فتتحمّلها المنصة. ولو
	// أخذ منها **لربح على متجرٍ هامشُه صفر والمنصةُ تدفع له من جيبها.**
	//
	// **وبالهامش يربح حين تربح ولا يربح حين لا تربح** — وهو أعدلُ حافزٍ
	// بينهما. ومتجرٌ جلبه ثمّ لم يُوضع على أصنافه هامشٌ لا يُنتج عمولة،
	// **وهو الصدق: لم تربح المنصةُ منه شيئاً.**
	//
	// والهامشُ **يُحسب من اللقطتين لا يُخزَّن** — فلا يفترق عن مصدريه.
	var margin int64
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(sum((oi.unit_price - oi.merchant_price) * oi.qty), 0)
		FROM order_items oi WHERE oi.order_id = $1`, orderID).Scan(&margin); err != nil {
		return err
	}
	repCommission, err := s.repShare(ctx, q, margin)
	if err != nil || repCommission <= 0 {
		return err
	}
	if _, err := s.wallet.ApplyTx(ctx, q, *repID, repCommission, "commission",
		orderID, "عمولة مندوب عن طلب مسلَّم", &actorID); err != nil {
		return err
	}
	out.repID, out.commissionPaid = *repID, repCommission
	return nil
}

// reverseCommissions يعكس أثر التسليم المالي عند استرجاع طلب مُسلَّم:
// قيد مضاد لعمولة المندوب (الدفاتر لا تُعدَّل ولا تُحذف — تُصحَّح بقيد مقابل)
// وتصفير لقطة عمولة المنصة كي لا تتضخّم التقارير وفواتير المتاجر.
func (s *Service) reverseCommissions(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	var platformCommission, subtotal int64
	var repID, ownerID *string
	var repIsBuyer bool
	err := q.QueryRow(ctx, `
		SELECT o.platform_commission, o.subtotal, m.sales_rep_user_id,
		       m.sales_rep_user_id = o.customer_id, m.owner_user_id
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1`, orderID).
		Scan(&platformCommission, &subtotal, &repID, &repIsBuyer, &ownerID)
	if err != nil {
		return err
	}

	// عكس مستحقّ المتجر أولاً: المنصة ردّت للزبون ثمن البضاعة، فلا يبقى للمتجر
	// مستحقٌّ عن بيعٍ لم يتمّ. وبلا هذا العكس يبقى مالٌ في دفتره عن طلب مُسترجَع —
	// وهو نفس تسريب R-14 من باب المتجر.
	//
	// **ويُعكس ما قُيّد فعلاً لا ما تقول المعادلة.**
	//
	// كان يُحسب `subtotal - platform_commission`، **و`subtotal` صار سعرَ البيع
	// لا سعرَ الشراء** — فلو بقيت الحسبةُ لَخُصم من المتجر هامشُنا معه: **مالٌ
	// لم يقبضه يُسترَدّ منه.** وفوقها تتغيّر النسبةُ بين القيد والعكس فيبقى
	// فرقٌ في محفظته بلا سبب.
	//
	// **والدفترُ يقول كم دُفع** — ولا يحتاج أن يُسأل مرّتين.
	if ownerID != nil {
		var paid int64
		if err := q.QueryRow(ctx, `
			SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
			WHERE ref = $1 AND kind = 'merchant_earning'`, orderID).Scan(&paid); err != nil {
			return err
		}
		if paid > 0 {
			if _, err := s.wallet.ApplyTx(ctx, q, *ownerID, -paid, "adjustment",
				orderID, "عكس مستحقّ متجر — طلب مُسترجَع", &actorID); err != nil {
				return err
			}
		}
	}

	if _, err := q.Exec(ctx,
		`UPDATE orders SET platform_commission = 0 WHERE id = $1`, orderID); err != nil {
		return err
	}
	if platformCommission == 0 || repID == nil {
		return nil
	}
	// لم تُدفع له عمولة على شرائه هو، فلا شيء يُعكس.
	// **والتماثل هنا شرط لا تجميل**: عكسٌ بلا دفعٍ سابق يخصم من رصيده مالاً لم
	// يقبضه قط — خطأ محاسبي في الاتجاه المعاكس.
	if repIsBuyer {
		return nil
	}

	// **وعمولتُه تُعكس بما قُيّد لا بما يُحسب.**
	//
	// أساسُها صار الهامشَ لا العمولة، **ونسبةُ المندوب إعدادٌ قد يتغيّر بين
	// الطلب واسترجاعه** — فحسبةٌ جديدةٌ تعكس مبلغاً غيرَ الذي قُبض، فيبقى فرقٌ
	// في محفظته إلى الأبد.
	var repPaid int64
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'commission'`, orderID).Scan(&repPaid); err != nil {
		return err
	}
	if repPaid <= 0 {
		return nil
	}
	// قد يكون رصيد المندوب أقلّ من العمولة (سحبها) — عندها يُرفض القيد بـ
	// insufficient_balance، وهو رفض صحيح: الدَّين يُسوّى يدوياً من المالية.
	_, err = s.wallet.ApplyTx(ctx, q, *repID, -repPaid, "adjustment",
		orderID, "عكس عمولة مندوب — طلب مُسترجَع", &actorID)
	return err
}

// repShare نصيب المندوب من عمولة المنصة — نسبة ديناميكية من الإعدادات.
func (s *Service) repShare(ctx context.Context, q wallet.Querier, platformCommission int64) (int64, error) {
	var repPct float64
	if err := q.QueryRow(ctx, `
		SELECT COALESCE((SELECT (value#>>'{}')::float8 FROM app_settings
		                 WHERE key = 'sales.commission_percent'), 10)`).Scan(&repPct); err != nil {
		return 0, err
	}
	return int64(float64(platformCommission) * repPct / 100), nil
}

// AssignDriver إسناد يدوي من العمليات: يتحقق أن الحساب سائق نشط ثم يسند وينقل الحالة.
func (s *Service) AssignDriver(ctx context.Context, actorID string, actorRoles []string, orderID, driverID, note string) (*Order, error) {
	var isDriver bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_roles ur JOIN users u ON u.id = ur.user_id
			WHERE ur.user_id = $1 AND ur.role_code = 'driver' AND u.status = 'active')`,
		driverID).Scan(&isDriver)
	if err != nil {
		return nil, err
	}
	if !isDriver {
		return nil, ErrNeedsDriver
	}

	var from string
	var cashDue int64
	err = s.db.QueryRow(ctx, `SELECT status, cash_due FROM orders WHERE id = $1`, orderID).
		Scan(&from, &cashDue)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	// الإسناد اليدوي مسموح من التحضير أو البحث عن سائق
	if from != StPreparing && from != StDispatching {
		return nil, ErrBadTransition
	}
	// السقف النقدي: لا طلبات نقدية لسائق تجاوز سقفه (PLAN §6.3)
	if cashDue > 0 {
		over, err := s.cashbox.OverLimit(ctx, driverID)
		if err != nil {
			return nil, err
		}
		if over {
			return nil, cashbox.ErrLimitExceed
		}
	}

	if _, err := s.db.Exec(ctx,
		`UPDATE orders SET driver_id = $2, updated_at = now() WHERE id = $1`,
		orderID, driverID); err != nil {
		return nil, err
	}
	// من التحضير: نمر عبر dispatching ثم assigned لسجل أحداث سليم
	if from == StPreparing {
		if _, err := s.Transition(ctx, actorID, actorRoles, orderID, StDispatching, note); err != nil {
			return nil, err
		}
	}
	return s.Transition(ctx, actorID, []string{"ops"}, orderID, StAssigned, note)
}

// payDriver يقيّد أجر السائق عن طلبٍ سلّمه.
//
// **الأجر خارج مبلغ الدَّين عمداً**: يُقيَّد في محفظته مستقلاً ويسلّم النقد كاملاً
// للمكتب. لو سلّم المبلغ ناقصاً أجره لما عاد مجموع ما حصّله يساوي مجموع ما
// سلّمه، فتصير تسوية الصندوق غير قابلة للمطابقة — وهي آخر ما يُراجَع عند الخلاف.
//
// والنمط من الإعدادات لا من الشيفرة: نسبةً من رسم التوصيل أو مبلغاً مقطوعاً،
// يُبدَّل بلا نشر. (ونمطٌ ثالث حسب المسافة حين تتوفّر مواقع السائقين الحيّة.)
// driverShare أجرُ السائق عن رسم توصيلٍ معلوم.
//
// **مصدرٌ واحد للحساب**: يستعمله قيدُ الأجر عند التسليم، وتستعمله غرفةُ
// العمليات لتعرف الأجر **قبل** الإسناد. ولو حُسب في موضعين لانحرف أحدهما يوماً
// — فتُسنِد العمليات على رقمٍ ويُقيَّد للسائق غيره.
//
// ومفتاحان لا مفتاح: النسبة والمبلغ المقطوع لكلٍّ منهما مداه. ومفتاحٌ واحد
// يعني معنيين لا يمكن حراسة مداه — كان يقبل ٢٠٠ لأنها مبلغٌ معقول، وهي نسبةٌ
// تجعل المنصة تدفع ضعف ما قبضت.
func driverShare(ctx context.Context, q wallet.Querier, deliveryFee int64) (int64, error) {
	var mode string
	var pct, fixed float64
	if err := q.QueryRow(ctx, `
		SELECT COALESCE((SELECT value#>>'{}' FROM app_settings WHERE key = 'drivers.share_mode'), 'percent'),
		       COALESCE((SELECT (value#>>'{}')::float8 FROM app_settings WHERE key = 'drivers.share_percent'), 70),
		       COALESCE((SELECT (value#>>'{}')::float8 FROM app_settings WHERE key = 'drivers.share_fixed'), 5000)`).
		Scan(&mode, &pct, &fixed); err != nil {
		return 0, err
	}
	if mode == "fixed" {
		return int64(fixed), nil
	}
	// percent — من رسم التوصيل لا من قيمة الطلب: أجرُ توصيلٍ لا حصةٌ من بيع
	return int64(float64(deliveryFee) * pct / 100), nil
}

func (s *Service) payDriver(ctx context.Context, q wallet.Querier, in settlement) error {
	if in.driverID == nil {
		return nil
	}
	share, err := driverShare(ctx, q, in.deliveryFee)
	if err != nil {
		return err
	}
	if share <= 0 {
		return nil
	}
	_, err = s.wallet.ApplyTx(ctx, q, *in.driverID, share, "driver_earning",
		in.orderID, "أجر توصيل طلب مُسلَّم", &in.actorID)
	return err
}

// merchantActivated هل بلغ عميلُ هذا الطلب عتبةَ التفعيل؟
//
// **طلبات المندوب نفسه لا تُحتسب في العتبة**. ولولا هذا الاستثناء لصارت العتبة
// بلا معنى: يشتري المندوب خمس مرّات من متجره فيُفعّله بيده، ثم يقبض عمّا بعدها.
// وقد مُنع من العمولة على شرائه فلا يُترك له بابٌ يفتحه بها.
//
// والعدّ **يشمل الطلب الحالي**: هو طلبٌ مُسلَّم فعلاً، فاستثناؤه يؤخّر التفعيل
// طلباً بلا سبب.
func (s *Service) merchantActivated(ctx context.Context, q wallet.Querier, orderID string) (bool, error) {
	var threshold int
	if err := q.QueryRow(ctx, `
		SELECT COALESCE((SELECT (value#>>'{}')::int FROM app_settings
		                 WHERE key = 'sales.activation_orders'), 5)`).Scan(&threshold); err != nil {
		return false, err
	}
	if threshold <= 1 {
		return true, nil
	}
	var delivered int
	if err := q.QueryRow(ctx, `
		SELECT count(*)
		FROM orders o
		JOIN merchants m ON m.id = o.merchant_id
		WHERE o.merchant_id = (SELECT merchant_id FROM orders WHERE id = $1)
		  AND o.status = 'delivered'
		  AND o.customer_id IS DISTINCT FROM m.sales_rep_user_id`, orderID).Scan(&delivered); err != nil {
		return false, err
	}
	return delivered >= threshold, nil
}

// cancelWindowSec مهلةُ تدارُك الزبون بالثواني.
//
// **تُقرأ من موضعين**: المحرّكُ يفرضها، والشاشةُ تعدّها تنازلياً. ولو حسبها
// كلٌّ بنفسه **لعدّ الزبونُ ثانيةً والخادمُ ثانيةً أخرى** — فيضغط على زرٍّ
// يراه حيّاً ويُردّ عليه بـ«انقضت المهلة».
func (s *Service) cancelWindowSec(ctx context.Context) int64 {
	if s.settings == nil {
		return 120
	}
	return s.settings.GetInt(ctx, "orders.customer_cancel_window_sec")
}

// CancelSecondsLeft ما بقي للزبون من مهلة الإلغاء — وصفرٌ إن لم يبقَ شيء.
//
// **يُرسل رقماً نسبياً لا موعداً مطلقاً**: ساعةُ الهاتف قد تسبق ساعةَ الخادم
// بدقائق، **فموعدٌ مطلقٌ يُقرأ عند المستخدم منقضياً وهو حيّ** أو حيّاً وهو
// منقضٍ. والنسبيُّ لا يعرف الساعتين.
func (s *Service) CancelSecondsLeft(ctx context.Context, o *Order) int {
	if o.Status != StAccepted || o.AcceptedAt == nil {
		if o.Status == StPending {
			// **قبل قبول المتجر لا مهلة أصلاً** — يُلغي متى شاء.
			return -1
		}
		return 0
	}
	left := s.cancelWindowSec(ctx) - int64(time.Since(*o.AcceptedAt).Seconds())
	if left < 0 {
		return 0
	}
	return int(left)
}
