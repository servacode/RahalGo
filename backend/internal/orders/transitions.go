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
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/pricing"
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
	var customerID, promoCode, kind string
	err = tx.QueryRow(ctx, `
		SELECT status, driver_id, wallet_paid, cash_due, delivery_fee, customer_id,
		       COALESCE(promo_code,''), kind
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&from, &driverID, &walletPaid, &cashDue, &deliveryFee, &customerID, &promoCode,
			&kind)
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
	selfManage := s.MerchantsSelfManage(ctx)
	effRoles := rolesUnderMode(selfManage, from, to, actorRoles, driverID != nil)
	if !canTransition(kind, from, to, effRoles) {
		return nil, ErrBadTransition
	}
	endedBy := authorizingRole(kind, from, to, effRoles)
	// لا استلام بلا سائق مسند
	if (to == StAtPickup || to == StPickedUp) && driverID == nil {
		return nil, ErrNeedsDriver
	}

	// **ولا يبدأ الخاصُّ قبل أن يُوثَّق ما اتُّفق عليه.**
	//
	// (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «لازم فتح دردشة مع الزبون لمعرفة التفاصيل…
	//  وبعد الاتّفاق وتوثيق البيانات يبدأ السائق».)
	//
	// **والتوثيقُ هو الحجّة**: من اشترى بلا اتّفاقٍ مكتوبٍ يقف عند الباب
	// **بكلمةٍ ضدّ كلمة** — لا العملياتُ تحكم ولا هو يستردّ.
	//
	// **وفي المحرّك لا في الشاشة**: زرٌّ يُخفى يُلتفّ عليه.
	if kind == KindCustom && to == StPickedUp {
		var agreed *time.Time
		if err := tx.QueryRow(ctx,
			`SELECT custom_agreed_at FROM orders WHERE id = $1`, orderID).
			Scan(&agreed); err != nil {
			return nil, err
		}
		if agreed == nil {
			return nil, ErrCustomNotAgreed
		}
	}

	// **وتعذّرٌ عند باب المتجر ليس فشلَ تسليم.**
	//
	// # المسألة
	//
	// السائقُ يقف عند المطعم فيجده مغلقاً أو يرفض التسليم، فيضغط «تعذّر».
	// **فيُغلق الطلبُ ويصل الزبونَ إشعارُ «فشل التسليم»** — وهو لم يُخطئ، ولم
	// يخرج من المطبخ شيءٌ بعد، **وللمنصة بديلٌ في يدها: تبديلُ المتجر.**
	//
	// **وزبونٌ يُقال له «فشل» وثمّة حلٌّ زبونٌ خسرناه بلا سبب.** (شهده المالك
	// ٢٠٢٦-٠٨-٠٥: «يجب أن يصل التنبيهُ للمنصة، والزبونُ يبقى طبيعياً ولا
	// يتأثّر بشيء لأنّه يوجد حلٌّ بديل».)
	//
	// # فيعود إلى المكتب لا يُغلق
	//
	// الحالةُ `accepted` — **حيث يظهر زرّا التحويل وتبديل المتجر.** ويُحرَّر
	// السائقُ ويُصفَّر وسمُ الإبلاغ، **فيبدأ التحويلُ من أوّله إلى متجرٍ آخر.**
	//
	// **والسائقُ يُعوَّض عن مشواره** كما لو فشل: قاد وعاد بلا شيء، **والذنبُ
	// ليس ذنبَه.**
	if to == StFailed && from == StAtPickup {
		return s.merchantBlocked(ctx, tx, orderID, actorID, from, failReason, note, driverID, deliveryFee)
	}

	set := `status = $2, updated_at = now()`
	switch to {
	case StAccepted:
		set += `, accepted_at = now()`
		// **ومدّةُ التحضير تُكتب هنا لا في معالجٍ بعينه.**
		//
		// (كشفه جردُ لوحة السائق ٢٠٢٦-٠٨-٠٩: `prep_minutes` فارغةٌ في كلّ
		//  طلبٍ في القاعدة.)
		//
		// **كانت تُكتب في مسار قبول المتجر وحدَه** — **والمنصةُ تعمل بوضع
		// «المنصة تدير»**، فالمتجرُ لا يقبل أصلاً: يقبل المكتب. **فلم تُكتب
		// المدّةُ لطلبٍ قطّ.**
		//
		// **وأثرُه ثلاثةُ أشياء صامتة**: السائقُ لا يرى متى يجهز الطلب ·
		// والزبونُ لا يرى وقتاً متوقَّعاً أصلاً (`hasEta` تشترطها) · وبطاقةُ
		// «طلبات قادمة» تُخفي شارةَ الجهوز. **وثلاثُ شاشاتٍ تصمت لأنّ عموداً
		// لم يُكتب.**
		//
		// **ولا يُكتب في المعالجات**: كلُّ من يقبل يجب أن يكتبها — **وقاعدةٌ
		// في معالجٍ تُنسى في المعالج الثاني**، وهو ما وقع بعينه.
		//
		// **والمكتوبُ لا يُدهَس**: من مرّر مدّةً صريحةً (المتجرُ يضبطها قبل
		// النداء) تبقى له.
		set += `, prep_minutes = COALESCE(prep_minutes,
			(SELECT m.default_prep_minutes FROM merchants m WHERE m.id = orders.merchant_id))`
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
		custom: kind == KindCustom,
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
	// **وحديثُ الطلب الخاصّ يبدأ بنفسه لحظةَ الإسناد** — (قرارُ المالك
	// ٢٠٢٦-٠٨-١٠). **وقناةٌ تُفتح ولا يقول فيها أحدٌ شيئاً تُبقي الطرفين
	// ينتظر كلٌّ منهما الآخر.**
	//
	// **وقبل النشر** — فالفقّاعةُ تُحدَّث بحدث الطلب، **ولو كُتب الحديثُ بعده
	// لَبقيت الشارةُ صفراً حتّى الحدث التالي.**
	if to == StAssigned && driverID != nil {
		if kind == KindCustom {
			s.openCustomChat(ctx, orderID, customerID, *driverID)
		} else {
			// **والعاديُّ يُفتح كذلك** — (قرارُ المالك ٢٠٢٦-٠٨-١١).
			s.openPlainChat(ctx, orderID, *driverID)
		}
	}

	// **وسطرٌ في الحديث عند كلّ خطوةٍ تهمّ الزبون** — (قرارُ المالك
	// ٢٠٢٦-٠٨-١٢). **والإشعارُ يمرّ ويُمحى، والحديثُ يبقى.**
	if driverID != nil {
		s.stepLine(ctx, orderID, *driverID, to)
	}

	// ══════════════════════════════════════════════════════════════════
	// **وزمنُ الطريق يُسأل عنه هنا لأنّ هنا وقعت اللحظة**
	// ══════════════════════════════════════════════════════════════════
	//
	// **موضعُ السائق يتحرّك** — فمن سأل الخريطةَ بعد يومين قاس من بيته.
	//
	// **وبعد أن يتمّ كلُّ شيء، في خيطٍ مستقلّ**: نداءٌ خارجيٌّ في طريق
	// الإسناد يجعل تعطّلَ الخريطة تعطّلاً في المنصّة — **سائقٌ يضغط
	// «أقبل» فينتظر مهلةَ اتّصالٍ ثمّ يُردّ بخطأ.**
	switch to {
	case StAssigned:
		s.captureToStoreETA(orderID)
	case StPickedUp:
		s.captureToDoorETA(orderID)
	}

	s.publishOrder(updated)
	// **وكلُّ من تحرّكت محفظتُه يُبلَّغ** — بعد الإيداع لا داخلَه.
	s.publishWalletsOf(ctx, orderID)
	// بعد الإيداع: فشل الإشعار لا يُبطل تسليماً وقع فعلاً
	s.notifyTransition(ctx, orderID, to, note, endedBy)
	s.notifyCommission(ctx, done.repID, orderID, done.commissionPaid)
	// **وكلُّ من تحرّكت محفظتُه يعرف عمّاذا** — (قرارُ المالك ٢٠٢٦-٠٨-١١:
	// «الرصيد يتغيّر وما حدا بيعرف ليش»).
	//
	// **بعد الإيداع لا داخلَه**: معاملةٌ تُرجَع بعد أن أُرسل إشعارُها
	// **تُخبر صاحبَها بمالٍ لم يصله.**
	s.notifyCredits(ctx, orderID, done.credits)

	// **ومكافأةُ من دعا هذا الزبون** — عند أوّل طلبٍ يُسلَّم له.
	//
	// **بعد الإيداع لا داخلَه**: مكافأةُ دعوةٍ لا تُبطل تسليماً وقع، **وقيدٌ
	// يُضاف إلى معاملة التسليم يجعل خطأً في الترويج يُسقط طلباً.**
	//
	// **وتخرج صامتةً إن لم يكن ثمّة ما يُصرف** — والحارسُ في موضعٍ واحدٍ خيرٌ
	// من شرطٍ يُكتب في كلّ نداء.
	if to == StDelivered && s.referrals != nil {
		s.referrals.SettleFirstOrder(ctx, customerID, orderID, actorID)
	}

	// **ومكافأةُ هدف الشهر إن بلغه بهذا الطلب.**
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «تُدفع بشكل آليّ عند إتمام الهدف».)
	//
	// **وهنا لأنّها اللحظةُ الوحيدةُ التي يتبدّل فيها العدّاد** — ونداؤها في
	// كلّ فتحةِ شاشةٍ يجعل القراءةَ تكتب، **وشاشةٌ تصرف مالاً بمجرّد أن تُفتح
	// لا تُراجَع.**
	//
	// **وبعد الإيداع لا داخلَه**: مكافأةٌ لا تُبطل تسليماً وقع فعلاً.
	// **والتكرارُ تمنعه القاعدةُ لا هذا السطر** — فهرسٌ فريدٌ لكلّ شهر.
	if to == StDelivered && driverID != nil && s.targets != nil {
		s.targets.GrantTargetIfReached(ctx, *driverID, "driver")
	}

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
		s.MerchantsSelfManage(ctx)
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
func (s *Service) compensateDriverOnFail(ctx context.Context, q wallet.Querier, in settlement, out *settled) error {
	if in.driverID == nil || in.deliveryFee <= 0 || s.settings == nil {
		return nil
	}
	var fault string
	if err := q.QueryRow(ctx,
		`SELECT COALESCE(fault, '') FROM orders WHERE id = $1`, in.orderID).
		Scan(&fault); err != nil {
		return err
	}
	// **ويُعوَّض في الحالين: ذنبُ الزبون وذنبُ المتجر.**
	//
	// قرارُ المالك (٢٠٢٦-٠٨-٠٣): **«نعم، المنصة تعوّضه — وبفتح نزاع مع المتجر
	// لحلّ القصة.»**
	//
	// وكانت القاعدةُ سابقاً «ذنبُ المتجر لا تعويضَ فيه» — **وهي تظلم السائق**:
	// قاد المشوارَ كاملاً **بسبب متجرٍ اعتذر متأخّراً**، وهو لا يملك من أمر
	// ذلك شيئاً. **ومن قاد بلا مقابلٍ مرّةً يتردّد في الثانية.**
	//
	// **وذنبُ السائق وحدَه لا تعويضَ فيه** — ومن أخّر فبرد الطعامُ لا يُؤجَر
	// على تأخيره.
	if fault != FaultCustomer && fault != FaultMerchant {
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
	who := "الحقُّ على الزبون"
	if fault == FaultMerchant {
		who = "المتجرُ اعتذر"
	}
	if _, err := s.wallet.ApplyTx(ctx, q, *in.driverID, amount, "compensation",
		in.orderID, "تعويضٌ عن تعذّر التسليم — "+who, &in.actorID); err != nil {
		return err
	}
	out.credit(*in.driverID, amount, t.compensated, notifications.AppDriver)
	if err := s.DebitTreasury(ctx, q, amount, in.orderID,
		"تعويضُ سائقٍ عن تعذّر تسليم", in.actorID); err != nil {
		return err
	}
	// **والمطالبةُ تُفتح على المتجر** — تُدفع الآن وتُحسم في مسارها.
	//
	// **والمنصةُ تدفع أوّلاً لا بعد الحسم**: نزاعٌ يستغرق يوماً يترك من قاد
	// مشوارَه بلا مقابلٍ يومَه كلَّه.
	if fault == FaultMerchant {
		return s.openMerchantClaim(ctx, q, in.orderID, amount)
	}
	return nil
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
	// ══════════════════════════════════════════════════════════════════
	// **credits ما تحرّكت به محفظةُ كلّ طرفٍ في هذه المعاملة**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١١: «الرصيد يتغيّر وما حدا بيعرف ليش».)
	//
	// **الأجرُ والمستحقُّ والاسترجاعُ والتعويض كانت تُقيَّد بلا إشعار.**
	// السائقُ يوصّل ويقبض، **فيرى الرقمَ في شريطه يزيد ولا يعرف عمّاذا**
	// — إلّا إن فتح المحفظةَ وقرأ السطر.
	//
	// **وتُجمَع هنا ولا تُرسَل في موضعها**: كلُّ هذه القيود داخلَ معاملةِ
	// الانتقال، **ومعاملةٌ تُرجَع بعد أن أُرسل إشعارُها تُخبر صاحبَها
	// بمالٍ لم يصله.** فتُجمَع، ثمّ تُرسَل بعد الإيداع.
	//
	// **وهو النمطُ نفسُه الذي تسير عليه `repID` منذ زمن.**
	credits []walletCredit
}

// walletCredit حركةُ محفظةٍ واحدةٌ تستحقّ أن يُخبَر بها صاحبُها.
type walletCredit struct {
	userID string
	amount int64
	// title العنوانُ كما يُقرأ — لا اسمُ النوع في القاعدة.
	title string
	// app التطبيقُ الذي يخصّه — فصاحبُ المتجر يحمل تطبيقين.
	app string
}

// credit يُسجّل حركةً للإخبار بها بعد الإيداع — **والصفرُ لا يُسجَّل.**
func (d *settled) credit(userID string, amount int64, title, app string) {
	if userID == "" || amount == 0 {
		return
	}
	d.credits = append(d.credits, walletCredit{userID: userID, amount: amount, title: title, app: app})
}

// settlement مدخلات التسوية المالية لانتقال واحد.
type settlement struct {
	orderID, from, to, actorID, customerID string
	driverID                               *string
	walletPaid, cashDue, deliveryFee       int64
	// custom **طلبٌ خاصّ — لا أثرَ ماليَّ له في المنصّة.**
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «السعر والأجرة لن تدخل بالحسابات، لأنّها
	//  خدمة للسائق فقط».)
	custom bool
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
	// (0) **والطلبُ الخاصّ لا يمرّ من هنا.**
	//
	// **السائقُ يدفع من جيبه ويستردّ عند التسليم** — والمنصّةُ توثّق ولا
	// تقبض: لا مستحقَّ متجرٍ ولا عمولةَ ولا أجرَ توصيلٍ ولا قيدَ صندوق.
	//
	// **وخروجٌ صريحٌ لا اعتمادٌ على أنّ الأعمدةَ أصفار**: هي أصفارٌ اليوم،
	// **وعمودٌ يُملأ غداً بسهوٍ يجعل مالاً يتحرّك بلا قرار.** والصمتُ في المال
	// أخطرُ منه في غيره.

	if in.custom {
		// **إلّا نقلَ المحفظة عند التسليم** — إن اختار الزبونُ الدفعَ منها.
		//
		// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «إذا تمّ الدفع من المحفظة يُحوَّل المبلغ
		//  بشكلٍ تلقائيٍّ إلى السائق».)
		//
		// **ونقلٌ لا كسب**: يُخصم من الزبون ما اتّفقا عليه ويُودَع للسائق —
		// **والمنصّةُ تعبر بلا أن تأخذ.**
		if in.to == StDelivered {
			return s.settleCustomWallet(ctx, q, in)
		}
		return nil
	}

	// (1) نهاية غير التسليم قبل التسليم — استرجاع ما دُفع من المحفظة
	if refundOnEnter(in.to) && in.from != StDelivered && in.walletPaid > 0 {
		if _, err := s.wallet.ApplyTx(ctx, q, in.customerID, in.walletPaid, "refund",
			in.orderID, fmt.Sprintf("استرجاع طلب %s", statusAr(in.to)), &in.actorID); err != nil {
			return err
		}
		out.credit(in.customerID, in.walletPaid, t.refunded2, notifications.AppCustomer)
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
		if err := s.settleMerchant(ctx, q, in.orderID, in.actorID, out); err != nil {
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
		if err := s.settleMerchant(ctx, q, in.orderID, in.actorID, out); err != nil {
			return err
		}
		if err := s.payDriver(ctx, q, in, out); err != nil {
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
		if err := s.compensateDriverOnFail(ctx, q, in, out); err != nil {
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
			out.credit(in.customerID, total, t.refunded2, notifications.AppCustomer)
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
func (s *Service) settleMerchant(ctx context.Context, q wallet.Querier, orderID, actorID string, out *settled) error {
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
		merchantID string
		ownerID    *string
		commission int64
		due        int64
	}
	shares := []share{}
	var totalCommission int64
	for rows.Next() {
		var merchantID string
		// **وعمودُ العمولة تجاوزٌ لا نسخة** — فراغُه «اتبع العامّ».
		var pct *int64
		var owner *string
		var cost int64
		if err := rows.Scan(&merchantID, &pct, &owner, &cost); err != nil {
			rows.Close()
			return err
		}
		c := pricing.MerchantCommission(ctx, s.settings, pct).Of(cost)
		totalCommission += c
		shares = append(shares, share{merchantID: merchantID, ownerID: owner, commission: c, due: cost - c})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// **وطلبٌ بلا بنود يقع على القديم** — طلباتُ ما قبل السعرين، أو ما أُنشئ
	// بلا `order_items`. **وصفرٌ هنا يعني «لا مستحقّ» وهو أسوأُ من الخطأ.**
	if len(shares) == 0 {
		var fallbackID string
		var subtotal int64
		var pct *int64
		var owner *string
		if err := q.QueryRow(ctx, `
			SELECT m.id::text, o.subtotal, m.commission_percent, m.owner_user_id::text
			FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE o.id = $1`, orderID).Scan(&fallbackID, &subtotal, &pct, &owner); err != nil {
			return err
		}
		c := pricing.MerchantCommission(ctx, s.settings, pct).Of(subtotal)
		totalCommission = c
		shares = append(shares, share{merchantID: fallbackID, ownerID: owner, commission: c, due: subtotal - c})
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
		out.credit(*sh.ownerID, sh.due, t.merchantEarned, notifications.AppMerchant)
		if _, err := s.wallet.ApplyTx(ctx, q, *sh.ownerID, sh.due, "merchant_earning",
			orderID, "مستحقّ عن بضاعةٍ سُلّمت للسائق", &actorID); err != nil {
			return err
		}
		// **ودَينُ بضاعةٍ رُدّت يُقتطع من أوّل مستحقٍّ قادم** (قرارُ المالك
		// ٢٠٢٦-٠٨-٠٤). وهو ما بقي حين لم تحتمله محفظتُه يومَ الردّ.
		if err := s.offsetMerchantDebt(ctx, q, sh.merchantID, *sh.ownerID,
			sh.due, orderID, actorID); err != nil {
			return err
		}
	}
	return nil
}

// offsetMerchantDebt يقتطع من مستحقٍّ جديدٍ ما بقي من دَينِ بضاعةٍ رُدّت.
//
// # ولماذا قيدان لا مبلغٌ منقوص
//
// كان يمكن أن يُقيَّد المستحقُّ ناقصاً من أوّله. **ومتجرٌ يرى رقماً أصغرَ بلا
// سطرٍ يقول لماذا يظنّ أنّه غُبن** — ويسأل، ولا يجد في كشفه ما يجيبه.
//
// **فسطرٌ يُعطي وسطرٌ يستردّ**، وكلاهما بمرجع طلبه.
//
// # وبالنوع نفسِه سالباً
//
// الخزينةُ تجمع `merchant_earning` لتعرف ما خرج للأطراف (`treasury.go`).
// **ونوعٌ لا تعرفه يجعلها تحسب أنّ المالَ خرج وهو عاد** — فتُقرأ خاسرةً وقد
// استُرِدّ لها.
func (s *Service) offsetMerchantDebt(ctx context.Context, q wallet.Querier,
	merchantID, ownerID string, available int64, orderID, actorID string) error {
	var debt int64
	if err := q.QueryRow(ctx,
		`SELECT debt FROM merchants WHERE id = $1 FOR UPDATE`, merchantID).
		Scan(&debt); err != nil {
		return err
	}
	if debt <= 0 {
		return nil
	}
	take := debt
	if available < take {
		take = available
	}
	if take <= 0 {
		return nil
	}
	if _, err := s.wallet.ApplyTx(ctx, q, ownerID, -take, "merchant_earning",
		orderID, "اقتطاعُ دَينٍ عن بضاعةٍ رُدّت سابقاً", &actorID); err != nil {
		return err
	}
	_, err := q.Exec(ctx,
		`UPDATE merchants SET debt = debt - $2 WHERE id = $1`, merchantID, take)
	return err
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
		       COALESCE(m.sales_rep_user_id = o.customer_id, false)
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
	if err := q.QueryRow(ctx,
		`SELECT `+OrderMarginSQL("$1"), orderID).Scan(&margin); err != nil {
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

// **ومتجرٌ بلا مندوبٍ لا يُسقط استرجاعاً.**
//
// المقارنةُ `sales_rep_user_id = customer_id` تُنتج NULL لا false حين لا مندوبَ
// للمتجر، **فيسقط المسحُ في bool ويعود الاسترجاعُ بخطأ** — ولا يُردّ للزبون
// شيء. **ولم يظهر في التجربة الحيّة لأنّ متجرَ الميدان له مندوب**، وكلُّ متجرٍ
// بلا مندوبٍ كان استرجاعُ طلبه مستحيلاً. كشفه أوّلُ اختبارٍ نادى هذا المسار.
//
// reverseCommissions يعكس أثر التسليم المالي عند استرجاع طلب مُسلَّم:
// قيد مضاد لعمولة المندوب (الدفاتر لا تُعدَّل ولا تُحذف — تُصحَّح بقيد مقابل)
// وتصفير لقطة عمولة المنصة كي لا تتضخّم التقارير وفواتير المتاجر.
func (s *Service) reverseCommissions(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	var platformCommission, subtotal int64
	var repID, ownerID *string
	var repIsBuyer bool
	err := q.QueryRow(ctx, `
		SELECT o.platform_commission, o.subtotal, m.sales_rep_user_id,
		       COALESCE(m.sales_rep_user_id = o.customer_id, false), m.owner_user_id
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
	// **ولكلِّ مطبخٍ عكسُه هو — لا مجموعُهما على واحد.**
	//
	// كشفته التجربةُ الحيّة (طلب #1009): كان يجمع قيودَ `merchant_earning`
	// كلَّها ثمّ **يخصم المجموعَ من صاحب المحطّة الأولى وحدَه**. ففي طلبٍ من
	// مطبخين قبض الأوّلُ ٤٣٬٢٠٠ **وخُصم منه ٥٠٬٨٠٠** — فصار رصيدُه سالباً
	// بسبعة آلافٍ لم يقبضها قطّ، **والثاني احتفظ بماله كاملاً على طلبٍ
	// مُسترَدّ.**
	//
	// **وهي علّةُ التسوية الأمامية نفسُها في مرآتها**: أُصلحت هناك ونُسيت هنا.
	// **والقيدُ يُعكس إلى المحفظة التي خرج منها**، لا إلى محفظةٍ يُظنّ أنها
	// صاحبتُه.
	mrows, err := q.Query(ctx, `
		SELECT user_id::text, sum(amount) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'merchant_earning'
		GROUP BY user_id`, orderID)
	if err != nil {
		return err
	}
	type paidTo struct {
		userID string
		amount int64
	}
	paidList := []paidTo{}
	for mrows.Next() {
		var x paidTo
		if err := mrows.Scan(&x.userID, &x.amount); err != nil {
			mrows.Close()
			return err
		}
		paidList = append(paidList, x)
	}
	mrows.Close()
	if err := mrows.Err(); err != nil {
		return err
	}
	for _, p := range paidList {
		if p.amount <= 0 {
			continue
		}
		// **والقيدُ المضادُّ يُكتب في الحساب الذي يعكسه.**
		//
		// كان يُكتب `adjustment` — **نوعاً جامعاً** يكتبه أيضاً التعديلُ
		// اليدويُّ من لوحة المحفظة ومطالبةُ المنصة على المتجر. **وحسبةُ نصيب
		// المنصة تجمع `merchant_earning` و`driver_earning` و`commission` ولا
		// ترى `adjustment`** — فيبقى في دفترها أنّها دفعت للمتجر وقد استردّت.
		//
		// وقع فعلاً في `#1003`: خسرت الخزينةُ ٣٩٬٨٠٠ على طلبٍ قدرُه ٢٢٬٠٠٠،
		// **وعشرةُ آلافٍ وثمانمئةٍ منها عكسٌ مكتوبٌ لا تراه.**
		//
		// **وضمُّ `adjustment` إلى الجمع يفتح باباً أسوأ**: تعديلٌ يدويٌّ
		// بمرجع طلبٍ يُحرّك أرباحَه بلا قصد. **والصوابُ أن يعود القيدُ إلى
		// حسابه** — فتصير كلُّ حسبةٍ تجمع بالنوع صحيحةً من نفسها، **ولا
		// يبقى نوعٌ يُنسى.**
		//
		// **وفائدةٌ ثانيةٌ تأتي معه**: من يقرأ `sum(merchant_earning)` يقرأ
		// **الصافي** — فعكسٌ ثانٍ يجد صفراً فلا يقع، **والتكرارُ يمتنع من
		// نفسه** بدل أن يُخصم من متجرٍ مرّتين.
		if _, err := s.wallet.ApplyTx(ctx, q, p.userID, -p.amount, "merchant_earning",
			orderID, "عكس مستحقّ متجر — طلب مُسترجَع", &actorID); err != nil {
			return err
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

	// **وعمولةُ المندوب تبقى له — لا تُعكس.**
	//
	// # قرارُ المالك (٢٠٢٦-٠٨-٠٣) نصّاً
	//
	// **«لا يوجد شي اسمه طعام فاسد بعد يوم او ساعات، واذا اصبحت هكذا مشكلة
	// فالمنصة سوف تتحمل المسؤولية، ويكون النزاع بين المنصة والمتجر —
	// والمندوب لا علاقة له بذلك، ولا السائق، ولا الزبون أيضاً.»**
	//
	// # وكان يُعكس بقرارٍ منّي لا منه
	//
	// قِسْتُه على «المنصةُ ردّت المال فلا يبقى لأحدٍ نصيبٌ منه» — **وهو قياسٌ
	// يناقض نفسَه**: أجرُ السائق يبقى بالحجّة المقابلة («أدّى الخدمة فعلاً»)،
	// **فصار طرفان أدّيا دورَيهما يُعامَلان بقاعدتين.**
	//
	// **والمندوبُ لا يملك جودةَ الطعام ولا سرعةَ التوصيل** — دورُه جلبُ العميل،
	// وقد جلبه، وباع المتجرُ فعلاً. **ومن يُعاقَب على ما لا يملكه يكفّ عن
	// العمل لا عن الخطأ.**
	//
	// **والنزاعُ بين المنصة والمتجر**: مستحقُّ المتجر يُسترَدّ (أعلاه)،
	// **والباقي على المنصة** — وهو ما يظهر في تقرير الخسائر الفعلية.
	//
	// ويبقى `repID` و`repIsBuyer` مقروءين أعلاه لأن الاستعلام واحد.
	_, _ = repID, repIsBuyer
	return nil
}

// repShare نصيب المندوب من عمولة المنصة — **من مخزن الإعدادات لا من SQL.**
//
// كان الاستعلامُ مكتوباً بيده هنا وفي `admin_financials` وفي `rep_merchant_detail`
// — **ثلاثةُ نسخٍ لقاعدةٍ واحدة**. فلمّا صار للعمولة نمطٌ (نسبةٌ أو مقطوع)
// لزم أن يُعدَّل ثلاثةُ مواضع، **ومن نسي واحداً دفع للمندوب غيرَ ما يُعرض له.**
//
// **والقراءةُ عند كلّ تسوية** — فتغييرُ المالك يسري على الطلب التالي.
func (s *Service) repShare(ctx context.Context, _ wallet.Querier, platformCommission int64) (int64, error) {
	return pricing.RepCommission(ctx, s.settings).Of(platformCommission), nil
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
// **وحصّةُ السائق هي أجرةُ التوصيل — كلُّها.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «حصّةُ السائق هي أجرة التوصيل».)
//
// # ولا نسبةَ ولا نمطَ ولا مفتاح
//
// كانت ثلاثةَ مفاتيح: نمطٌ ونسبةٌ ومقطوع. **وحُذفت مع الإعدادات ولم تُعَد** —
// فقُرئت أصفاراً، **فسلّم السائقُ ولم يُقيَّد له قرش**، وابتلعت الخزينةُ
// أجرةَ التوصيل كاملة. (وقع في `#1001`.)
//
// **والقاعدةُ الآن لا تحتاج مفتاحاً**: ما يدفعه الزبونُ توصيلاً يقبضه من
// أوصل. **ورقمٌ لا يُشتقّ لا يفترق.**
//
// # وربحُ المنصة من غيره
//
//	الهامشُ         ←  يُضاف على الزبون فوق سعر الشراء
//	عمولةُ المتاجر  ←  تُقتطع من سعر شراء المتجر
//	التوصيلُ        ←  **يمرّ إلى السائق ولا يبقى منه شيء**
//
// **والمنصةُ لا تربح من مشوارٍ لم تقده.**

func (s *Service) payDriver(ctx context.Context, q wallet.Querier, in settlement, out *settled) error {
	if in.driverID == nil {
		return nil
	}
	if in.deliveryFee <= 0 {
		return nil
	}
	if _, err := s.wallet.ApplyTx(ctx, q, *in.driverID, in.deliveryFee, "driver_earning",
		in.orderID, "أجر توصيل طلب مُسلَّم", &in.actorID); err != nil {
		return err
	}
	out.credit(*in.driverID, in.deliveryFee, t.driverEarned, notifications.AppDriver)
	return nil
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
	// **والعتبةُ من المخزن لا من SQL خام** — الافتراضُ في الفهرس وحدَه.
	threshold := s.settingInt(ctx, "sales.activation_orders")
	if threshold <= 1 {
		return true, nil
	}
	var delivered int64
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
	// **والطلبُ الخاصُّ بلا مهلةٍ إلى أن يُشترى** — (٢٠٢٦-٠٨-١٠).
	//
	// **حدُّه حدثٌ لا ساعة**: خروجُ المال من جيب السائق. **ورقمٌ ينقص
	// على الشاشة يستعجل صاحبَه بلا سبب** — لا مطبخَ بدأ يطبخ.
	if o.Kind == KindCustom {
		switch o.Status {
		case StPending, StDispatching, StAssigned:
			return -1
		default:
			return 0
		}
	}
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
