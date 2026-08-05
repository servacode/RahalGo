package orders

// تعذّرٌ عند باب المتجر — **ليس فشلَ تسليم.**
//
// # المسألة
//
// السائقُ يقف عند المطعم فيجده مغلقاً أو يرفض التسليم، فيضغط «تعذّر». **وكان
// الطلبُ يُغلق ويصل الزبونَ إشعارُ «فشل التسليم»** — وهو لم يُخطئ، **ولم يخرج
// من المطبخ شيءٌ بعد**، وللمنصة بديلٌ في يدها: **تبديلُ المتجر.**
//
// **وزبونٌ يُقال له «فشل» وثمّة حلٌّ زبونٌ خسرناه بلا سبب.**
//
// (شهده المالك ٢٠٢٦-٠٨-٠٥: «يجب أن يصل التنبيهُ للمنصة، والزبونُ يبقى طبيعياً
// ولا يتأثّر بشيء لأنّه يوجد حلٌّ بديل».)
//
// # والفرقُ بين البابين
//
//	باب المتجر  ←  لا بضاعةَ خرجت · **بديلٌ قائم** · الطلبُ يعود للمكتب
//	باب الزبون  ←  البضاعةُ في الصندوق · لا بديل · **الطلبُ يُغلق**
//
// **وهما اليوم فعلٌ واحدٌ بزرٍّ واحدٍ في شاشة السائق** — وهو صواب: من يقف عند
// بابٍ لا يُسأل عن حالة النظام، **يقول ما وقع والمحرّكُ يقرّر ما يعنيه.**
//
// # ولماذا `accepted` لا `dispatching`
//
// **`accepted` حيث يظهر زرّا التحويل وتبديل المتجر.** ولو عاد `dispatching`
// **لَنزل إلى سائقٍ آخرَ يقف أمام المطعم المغلق نفسِه** — ويتكرّر إلى أن
// ينتبه أحد.
//
// # والسائقُ يُعوَّض
//
// قاد وعاد بلا شيء، **والذنبُ ليس ذنبَه.** وبالمعادلة نفسِها التي تعوّضه عند
// الفشل — **ولا حسبةَ ثانيةً لواقعةٍ من الجنس نفسِه.**

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// merchantBlockedNote يُقرأ في سجلّ الطلب — **ومن رأى الطلبَ يعود يسأل لماذا.**
const merchantBlockedNote = "المتجرُ تعذّر — عاد إلى المكتب للتبديل"

// merchantBlocked يردّ الطلبَ إلى المكتب بدل أن يُغلقه.
func (s *Service) merchantBlocked(ctx context.Context, tx wallet.Querier,
	orderID, actorID, from, failReason, note string,
	driverID *string, deliveryFee int64) (*Order, error) {
	// **ويُحرَّر السائقُ ويُصفَّر وسمُ الإبلاغ** — فيبدأ التحويلُ من أوّله.
	//
	// **و`dispatched_at` يُصفَّر معهما**: منه تُقاس مهلةُ إيجاد سائق، **ولو
	// بقي لَظهر الطلبُ متأخّراً منذ نزوله الأوّل وهو لم ينزل بعد.**
	if _, err := tx.Exec(ctx, `
		UPDATE orders
		SET status = 'accepted', driver_id = NULL, offered_driver_id = NULL,
		    offer_expires_at = NULL, dispatched_at = NULL,
		    sent_to_merchant_at = NULL, updated_at = now(),
		    fail_reason = $2, fault = 'merchant'
		WHERE id = $1`, orderID, failReason); err != nil {
		return nil, err
	}

	// **والحدثُ يقول ما وقع لا ما صار إليه فقط.**
	//
	// «إلى مقبول» وحدَها تُقرأ تراجعاً غامضاً — **والنصُّ يقول إنّ المتجرَ
	// تعذّر**، فتعرف العملياتُ أنّ عليها التبديلَ لا الانتظار.
	body := merchantBlockedNote
	if note != "" {
		body += " — " + note
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, $2, 'accepted', $3, $4)`, orderID, from, actorID, body); err != nil {
		return nil, err
	}

	// **ويُعوَّض السائق** — بالمعادلة نفسِها، ومن الخزينة كما دائماً.
	if err := s.compensateDriverOnFail(ctx, tx, settlement{
		orderID: orderID, actorID: actorID, driverID: driverID,
		deliveryFee: deliveryFee,
	}); err != nil {
		return nil, err
	}
	// **والخزينةُ تُعيد الحساب** — تعويضٌ خرج منها يُقرأ في ربح الطلب.
	if err := s.creditTreasury(ctx, tx, orderID, actorID); err != nil {
		return nil, err
	}

	if c, ok := tx.(interface{ Commit(context.Context) error }); ok {
		if err := c.Commit(ctx); err != nil {
			return nil, err
		}
	}

	updated, err := s.GetByID(ctx, orderID)
	if err != nil {
		return updated, err
	}
	s.publishOrder(updated)
	s.publishWalletsOf(ctx, orderID)

	// **والإنذارُ يُسجَّل كما كان — ووجودُ البديل لا يُبرّئ من أغلق بابَه.**
	//
	// **وهذا ما كاد يضيع في التصويب**: الطريقُ القديمُ كان يُنذر عند `failed`،
	// **وتحويلُ المسار قفز فوقه** — فمن أغلق بابَه عشر مرّاتٍ بقي بلا مخالفةٍ
	// واحدة، **والمنصةُ تدفع التعويضَ في كلّ مرّةٍ وتبتلعه صامتة.**
	//
	// **بل هو هنا أوجب**: الزبونُ لم يُخطَر، **فلا شكوى تكشف المتجرَ** — والعدُّ
	// وحدَه ما يُظهره.
	if failReason != "" {
		s.warnMerchantOnFault(ctx, orderID, FaultOf(failReason), failReason)
	}

	// **والتنبيهُ للمكتب وحدَه — لا للزبون.**
	//
	// **وهذا هو جوهرُ التصويب**: الزبونُ لا يعلم أنّ شيئاً وقع، **لأنّ شيئاً
	// لم يقع في حقّه** — طلبُه قائمٌ ويُحوَّل إلى مطبخٍ آخر.
	if s.notify != nil {
		s.notify.NotifyOps(ctx, notifications.Input{
			Kind:     notifications.KindOrder,
			Title:    "المتجرُ تعذّر — الطلبُ ينتظر تبديلاً",
			Body:     "#" + itoa(updated.Number) + " — " + body,
			Entity:   "order",
			EntityID: orderID,
			Href:     "/dashboard/orders",
		})
	}
	return updated, nil
}

// itoa رقمُ الطلب نصّاً — **بلا استيراد حزمةٍ لسطر.**
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
