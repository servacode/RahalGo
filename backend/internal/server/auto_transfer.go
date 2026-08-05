package server

// التحويلُ التلقائيّ — **طلبٌ كبيرٌ لا ينتظر يداً.**
//
// # المسألة
//
// في وضع المنصة يقبل المكتبُ الطلبَ ثمّ يحوّله إلى المتجر. **وطلبٌ بمئتَي ألفٍ
// ينتظر موظّفاً مشغولاً خسارةٌ مضاعفة**: الزبونُ الأكبرُ هو أوّلُ من يمَلّ.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «يتحوّل الطلبُ تلقائياً إذا تجاوز مبلغُه رقماً
// يضعه صاحبُ المنصة، أو عددَ أصنافٍ — بنفس السلّة بنفس الطلب الواحد».)
//
// # ولا يُوسَم مُرسَلاً ما لم يُرسَل
//
// **وهذا هو الحارسُ الذي يمنع الكارثة.**
//
// «التحويل» ثلاثةُ أفعالٍ لا واحد: **قبولٌ، وإبلاغُ المتجر، وإنزالٌ إلى
// السائقين.** ولو وُسم الطلبُ مُرسَلاً بلا إبلاغٍ فعليّ **لَنزل إلى الطابور
// فأخذه سائقٌ ووقف أمام مطبخٍ لم يسمع بالطلب أصلاً** — والزبونُ ينتظر، والمطعمُ
// يُتَّهم، والسائقُ يضيع مشوارُه.
//
// **فالتحويلُ لا يقع إلّا إذا وُجدت قناةٌ تُبلّغ فعلاً:**
//
//	وضعُ المتاجر  ←  المتجرُ يرى الطلبَ في بوابته — **البوابةُ هي القناة**
//	وضعُ المنصة   ←  رسالةٌ نصّيةٌ إن كان المرسِلُ مهيّأً، وإلّا **لا تحويل**
//
// **وما لم يُحوَّل يبقى `pending` بيد المكتب** كما هو اليوم — لا يضيع، ولا
// يُخترع له مسارٌ لا يعرفه أحد.

import (
	"context"
	"strings"
)

// autoTransfer يحوّل طلباً بلغ عتبةَ المبلغ أو العدد — **أو لا يفعل شيئاً.**
//
// **ولا تُرجع خطأً**: طلبٌ أُنشئ نجح، **وتعثّرُ التحويل لا يُبطل إنشاءً وقع**
// — يبقى بيد المكتب.
func (s *Server) autoTransfer(ctx context.Context, orderID, actorID string) {
	minTotal := s.settings.GetInt(ctx, "orders.auto_transfer_min_total")
	minItems := s.settings.GetInt(ctx, "orders.auto_transfer_min_items")
	// **وصفرُهما لا تحويل** — لا «صفرٌ يعني كلَّ طلب».
	if minTotal <= 0 && minItems <= 0 {
		return
	}

	var total, items int64
	var status string
	if err := s.pg.QueryRow(ctx, `
		SELECT o.total, o.status,
		       COALESCE((SELECT sum(oi.qty) FROM order_items oi WHERE oi.order_id = o.id), 0)
		FROM orders o WHERE o.id = $1`, orderID).Scan(&total, &status, &items); err != nil {
		return
	}
	if status != "pending" {
		return
	}
	// **وأيُّهما بلغ يكفي** — عائلةٌ تطلب عشرين صحناً رخيصاً طلبٌ كبيرٌ أيضاً،
	// **والمبلغُ وحدَه يُغفلها.**
	if !(minTotal > 0 && total >= minTotal) && !(minItems > 0 && items >= int64(minItems)) {
		return
	}

	// **والقناةُ تُفحص قبل القبول لا بعده.**
	//
	// ولو قُبل ثمّ تعذّر الإبلاغُ **لَبقي الطلبُ في «مقبول» بلا أن يعلم به
	// المتجر** — وهي حالٌ أسوأُ من `pending`: الشاشةُ تقول إنّ شيئاً جرى.
	selfManage := s.orders.MerchantsSelfManage(ctx)
	if !selfManage && !s.textSender.Configured() {
		s.logger.Info("التحويلُ التلقائيّ: لا قناةَ تُبلّغ المتجر — يبقى بيد المكتب",
			"order", orderID)
		return
	}

	if _, err := s.orders.Transition(ctx, actorID, []string{"ops"},
		orderID, "accepted", autoTransferNote); err != nil {
		s.logger.Warn("التحويلُ التلقائيّ: تعذّر القبول", "order", orderID, "error", err)
		return
	}

	// **وفي وضع المنصة تُرسَل الرسالةُ فعلاً** — والبوابةُ هي القناة في الوضع
	// الآخر، فلا رسالةَ تلزم.
	if !selfManage {
		msg, ph, err := s.loadOrderMessage(ctx, orderID)
		if err != nil || strings.TrimSpace(ph.SMS) == "" {
			s.logger.Warn("التحويلُ التلقائيّ: لا هاتفَ للمتجر — قُبل ولم يُبلَّغ",
				"order", orderID)
			return
		}
		if err := s.textSender.SendText(ctx, ph.SMS, buildMerchantMessage(msg)); err != nil {
			s.logger.Error("التحويلُ التلقائيّ: تعذّر الإبلاغ", "order", orderID, "error", err)
			return
		}
		if _, err := s.pg.Exec(ctx,
			`UPDATE orders SET sent_to_merchant_at = now() WHERE id = $1`, orderID); err != nil {
			s.logger.Error("التحويلُ التلقائيّ: تعذّر وسمُ الإرسال", "order", orderID, "error", err)
		}
	}

	// **والآن — لا قبل الآن — يُستدعى السائق.**
	//
	// المطعمُ علم بالطلب فبدأ عدّادُ التحضير عنده، **واستدعاءٌ قبل ذلك يرسل
	// السائقَ إلى بابٍ لم يُطبخ خلفه شيء.**
	if err := s.orders.AutoDispatch(ctx, actorID, orderID); err != nil {
		s.logger.Warn("التحويلُ التلقائيّ: تعثّر الإنزال", "order", orderID, "error", err)
	}
	s.touch("order", "ops")
}

// autoTransferNote يُقرأ في سجلّ الطلب — **ومن رأى «مقبول» بلا فاعلٍ يسأل من.**
const autoTransferNote = "تحويلٌ تلقائيٌّ — طلبٌ بلغ العتبة"
