package orders

import (
	"context"
	"errors"
	"fmt"
)

// ══════════════════════════════════════════════════════════════════════
// **الطلب ما بيعلق عند المكتب ولا عند الواتساب** (قرار المالك ٢٠٢٦-١٠-٣١)
// ══════════════════════════════════════════════════════════════════════
//
// بوضع المنصّة: إذا مرّ `orders.no_whatsapp_dispatch_min` دقيقة على إنشاء الطلب
// وما حدا قبله، أو انقبل وما انبعت للمتجر على الواتساب (`sent_to_merchant_at`
// فاضي) — **بينزل للسائقين لحاله بلا واتساب**، والسائق بيطلب من المتجر لمّا يوصل.
//
// وبيشمل كل الأنواع: العادي، والخاص (اطلب أي شي/مشتريات/أمانة)، و«لدي توصيلة»
// و«إحضار». **وصفره يعني «لا».** وبوضع المتاجر ما بيعمل شي — المتجر بيدير طلباته.
//
// الجولة بخطوتين بنفس الدورة:
//
//	أ) معلّق أقدم من المهلة ⇐ مقبول (والخاص ⇐ الطابور رأساً)
//	ب) مقبول وما انبعت للمتجر وعمره من الإنشاء أكبر من المهلة ⇐ الطابور
//
// **والساعة من إنشاء الطلب بالخطوتين** — فالانتظار الكلّي للزبون هو المهلة نفسها.

// noWhatsAppAcceptNote **سطر السجل لما ينقبل الطلب لحاله** — والرقم هو المهلة.
const noWhatsAppAcceptNote = "انقبل لحاله — ما حدا قبله خلال %d دقيقة"

// noWhatsAppDispatchNote **سطر السجل لما ينزل للسائقين بلا واتساب** — والرقم هو المهلة.
const noWhatsAppDispatchNote = "نزل للسائقين لحاله — ما انبعت للمتجر خلال %d دقيقة"

// sweepNoWhatsAppDispatch — انظر رأس الملف.
func (s *Service) sweepNoWhatsAppDispatch(ctx context.Context) {
	if s.settings == nil || s.MerchantsSelfManage(ctx) {
		return
	}
	mins := s.settingInt(ctx, "orders.no_whatsapp_dispatch_min")
	if mins <= 0 {
		return
	}
	// **وما بيمسك الطلبات القديمة** — طلبٌ معلّقٌ أقدمُ من مهلة التوصيل منسيٌّ بيد المكتب، ونزولُه
	// للسائقين بعد ساعات (أو أوّلَ ما ينرفع هالتعديل) بيبعت سائقاً لطلبٍ ما حدا ناطره.
	maxAge := s.settingInt(ctx, "orders.delivery_timeout_min")
	if maxAge <= mins {
		maxAge = 60
	}

	// ── أ) معلّق ما حدا قبله ──
	type pend struct{ id, kind string }
	pending := make([]pend, 0, 8)
	rows, err := s.db.Query(ctx, `
		SELECT id, kind FROM orders
		 WHERE status = 'pending'
		   AND closed_at IS NULL
		   AND created_at < now() - make_interval(mins => $1::int)
		   AND created_at > now() - make_interval(mins => $2::int)
		 ORDER BY created_at DESC
		 LIMIT 50`, mins, maxAge)
	if err != nil {
		s.logger.Warn("النزول بلا واتساب: تعذّرت قراءة المعلّق", "error", err)
		return
	}
	for rows.Next() {
		var p pend
		if err := rows.Scan(&p.id, &p.kind); err == nil {
			pending = append(pending, p)
		}
	}
	rows.Close()

	acceptNote := fmt.Sprintf(noWhatsAppAcceptNote, mins)
	for _, p := range pending {
		// **والخاص ما إله «مقبول»** (`customTransitions`) — قبوله هو نزوله للطابور.
		to := StAccepted
		note := acceptNote
		if p.kind == KindCustom {
			to = StDispatching
			note = fmt.Sprintf(noWhatsAppDispatchNote, mins)
		}
		if _, err := s.Transition(ctx, "", []string{"ops"}, p.id, to, note); err != nil {
			// **وتوصيلة رصيد متجرها ما بيكفي بتضل معلّقة — والمكتب بينخبر مرّة.**
			if errors.Is(err, ErrMerchantDeliveryUnpaid) {
				s.flagMerchantDeliveryUnpaid(ctx, p.id)
			}
			s.logger.Warn("النزول بلا واتساب: القبول تعثّر", "order", p.id, "error", err)
			continue
		}
		s.logger.Info("انقبل لحاله بعد المهلة", "order", p.id, "minutes", mins)
	}

	// ── ب) مقبول وما انبعت للمتجر ──
	//
	// **بعد (أ) بنفس الدورة** — فالمعلّق اللي انقبل هلّق بينزل بنفس الجولة.
	// والتوصيلة («لدي توصيلة»/«إحضار») منها: ما بتنبعت لمتجر، فبتنزل متل غيرها.
	accepted := make([]string, 0, 8)
	rows, err = s.db.Query(ctx, `
		SELECT id FROM orders
		 WHERE status = 'accepted'
		   AND kind <> 'custom'
		   AND sent_to_merchant_at IS NULL
		   AND closed_at IS NULL
		   AND created_at < now() - make_interval(mins => $1::int)
		   AND created_at > now() - make_interval(mins => $2::int)
		 ORDER BY created_at DESC
		 LIMIT 50`, mins, maxAge)
	if err != nil {
		s.logger.Warn("النزول بلا واتساب: تعذّرت قراءة المقبول", "error", err)
		return
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			accepted = append(accepted, id)
		}
	}
	rows.Close()

	dispatchNote := fmt.Sprintf(noWhatsAppDispatchNote, mins)
	for _, id := range accepted {
		if _, err := s.Transition(ctx, "", []string{"ops"}, id, StDispatching, dispatchNote); err != nil {
			s.logger.Warn("النزول بلا واتساب: الإنزال تعثّر", "order", id, "error", err)
			continue
		}
		s.logger.Info("نزل للسائقين بلا واتساب", "order", id, "minutes", mins)
	}
}

// SweepNoWhatsAppDispatchOnce **جولة واحدة** — مِعراض الفحص متل `SweepAutoAcceptOnce`، ولا مسار شبكة إله.
func (s *Service) SweepNoWhatsAppDispatchOnce(ctx context.Context) { s.sweepNoWhatsAppDispatch(ctx) }
