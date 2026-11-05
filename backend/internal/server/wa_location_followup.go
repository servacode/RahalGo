package server

// ══════════════════════════════════════════════════════════════════════
// **«ما قدرت تشارك الموقع؟ اكتبلنا عنوانك»** (ملحقُ المالك ٢٠٢٦-١١-٠٥، د)
// ══════════════════════════════════════════════════════════════════════
//
// «واذا ماشارك الموقع نطلب منه الموقع التفصيلي، بحيث اذا ما بعث الموقع ناخذ العنوان التفصيلي»
// ثمّ «مايصير عشر دقايق كثيرة، 3 دقايق». **بعد `orders.whatsapp_location_followup_min` دقيقة
// من طلب الموقع بلا موقعٍ وصل، البوتُ يبعت رسالةً ثانيةً مرّةً وحدة** يطلب العنوانَ مكتوباً.
//
// # لطلب الواتساب من المكتب وحدَه
//
// **الردُّ يصل رقمَ المنصّة ويقرؤه المكتب** — فهو من يكتبه في الطلب (`/address`). **وتوصيلةُ
// المتجر و«إحضار» والأمانةُ رسالةٌ واحدة** — الردُّ هناك لا يقرؤه أحد. **ولا التقاطَ تلقائيّاً
// للنصّ**: المكتبُ يقرأ ويعبّي بيده («نحن رح نعبيه يدوي»).
//
// # ومتى لا يُبعث
//
//   - **وصل الموقع**، أو انتهى الرابط، أو الصفرُ في الإعداد.
//   - **رسالةُ الطلب الأولى لم تصل** (`whatsapp_sent`) — ثانيةٌ بلا أولى لا تُفهم.
//   - **الطلبُ المربوطُ انتهى** (سُلّم أو أُلغي) — لا يُسأل عن عنوانٍ لا يُوصَل إليه.
//   - **طلبٌ قديمٌ قبل اشتعال الإعداد** — نافذةُ ساعتين، فمن أشعله اليوم لا يُراسل طلباتِ الأمس.

import (
	"context"
	"time"
)

// waLocFollowUpText **نصُّ المالك حرفاً** (٢٠٢٦-١١-٠٥).
const waLocFollowUpText = "إذا ما قدرت تشارك الموقع، اكتبلنا عنوانك بالتفصيل برسالة وحدة: الحي، الشارع، وأقرب معلم 🌷"

// waLocFollowUpWindow **أقدمُ طلبٍ يُتابَع** — ساعتان.
const waLocFollowUpWindow = 2 * time.Hour

// waLocFollowUpBatch **حدُّ الجولة الواحدة.**
const waLocFollowUpBatch = 50

// RunWALocationFollowUp **يتابع طلباتِ الموقع من المكتب** — دورةٌ كلَّ نصف دقيقة تكفي لثلاث دقائق.
func (s *Server) RunWALocationFollowUp(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SweepWALocationFollowUp(ctx)
		}
	}
}

// SweepWALocationFollowUp **جولةٌ واحدة** — ويردّ عددَ الرسائل التي وصلت. **يُنادى من الفحص مباشرةً.**
//
// **الحجزُ قبل الإرسال** (`followup_sent_at`) — جولتان متزامنتان لا تبعثان مرّتين، **ورسالةٌ لم
// تصل لا تُعاد** (كرسالة الرأي): المستلِمُ لا يُلاحَق.
func (s *Server) SweepWALocationFollowUp(ctx context.Context) int {
	minutes := s.settings.GetInt(ctx, "orders.whatsapp_location_followup_min")
	if minutes <= 0 {
		return 0
	}
	// **البوتُ غيرُ جاهز ⇒ لا حجزَ ولا إرسال** — يُتابَع حين يعود ما دام في النافذة.
	if !s.merchantReady() {
		return 0
	}
	rows, err := s.pg.Query(ctx, `
		SELECT l.id::text, l.phone
		  FROM recipient_location_requests l
		 WHERE l.mode = 'whatsapp'
		   AND l.received_at IS NULL AND l.followup_sent_at IS NULL
		   AND l.whatsapp_sent AND l.expires_at > now()
		   AND l.created_at <= now() - make_interval(mins => $1)
		   AND l.created_at >  now() - make_interval(secs => $2)
		   AND (l.order_id IS NULL OR EXISTS (
		        SELECT 1 FROM orders o
		         WHERE o.id = l.order_id AND o.delivered_at IS NULL AND o.closed_at IS NULL))
		 ORDER BY l.created_at
		 LIMIT $3`, minutes, int(waLocFollowUpWindow.Seconds()), waLocFollowUpBatch)
	if err != nil {
		s.logger.Warn("متابعةُ طلب الموقع: قراءةُ المرشَّحين سقطت", "error", err)
		return 0
	}
	type cand struct{ id, phone string }
	var cands []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.id, &c.phone); err != nil {
			rows.Close()
			s.logger.Warn("متابعةُ طلب الموقع: قراءةُ صفٍّ سقطت", "error", err)
			return 0
		}
		cands = append(cands, c)
	}
	rows.Close()

	sent := 0
	for _, c := range cands {
		if ctx.Err() != nil {
			break
		}
		tag, err := s.pg.Exec(ctx, `
			UPDATE recipient_location_requests SET followup_sent_at = now()
			 WHERE id = $1 AND followup_sent_at IS NULL AND received_at IS NULL`, c.id)
		if err != nil {
			s.logger.Warn("متابعةُ طلب الموقع: الحجزُ سقط", "request", c.id, "error", err)
			continue
		}
		if tag.RowsAffected() == 0 {
			continue
		}
		sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		ok := s.sendText(sendCtx, c.phone, waLocFollowUpText)
		cancel()
		if !ok {
			s.logger.Warn("متابعةُ طلب الموقع: الرسالةُ لم تصل — لا إعادة", "request", c.id)
			continue
		}
		sent++
	}
	if sent > 0 {
		s.touch("orders", "ops")
	}
	return sent
}
