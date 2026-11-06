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
//   - **وصل الموقع أو العنوانُ المكتوب** (ردٌّ نصّيّ، ٢٠٢٦-١١-٠٦)، أو انتهى الرابط، أو الصفرُ في الإعداد.
//   - **رسالةُ الطلب الأولى لم تصل** (`whatsapp_sent`) — ثانيةٌ بلا أولى لا تُفهم.
//   - **الطلبُ المربوطُ انتهى** (سُلّم أو أُلغي) — لا يُسأل عن عنوانٍ لا يُوصَل إليه.
//   - **الطلبُ المربوطُ له نقطةٌ معروفة** — حدّدها الموظّفُ على الخريطة، فلا يُسأل عن عنوانه.
//   - **طلبٌ قديمٌ قبل اشتعال الإعداد** — نافذةُ ساعتين، فمن أشعله اليوم لا يُراسل طلباتِ الأمس.
//
// # وجولةُ إصلاحٍ قبلها
//
// **«وصل» يُثبَّت على طلب الموقع ثمّ تُكتب النقطةُ في الطلب** (`locReqReceived`) — خطوتان، فإن
// سقطت الثانية (عطلُ القاعدة) بقي الطلبُ بلا نقطةٍ وطلبُ الموقع يقول «وصل». **فالجولةُ تكتبها**
// (`repairWALocations`) — كلَّ دورةٍ، **مهما كان الإعدادُ أو البوت.**

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

// waLocRepairGrace **ما يُنتظر قبل أن تُعدّ الكتابةُ ساقطة** — الكتابةُ الحيّةُ تلي «وصل» في لحظتها.
const waLocRepairGrace = time.Minute

// repairWALocations **موقعٌ وصل ولم يُكتب في طلبه الجاري** — يُكتب الآن (`FillWhatsAppShared`:
// لا فوق نقطةٍ معروفة) ويُبثّ للسائق كما لو وصل الآن. ويردّ كم كُتب.
func (s *Server) repairWALocations(ctx context.Context) int {
	rows, err := s.pg.Query(ctx, `
		SELECT l.order_id::text, l.lat, l.lng
		  FROM recipient_location_requests l
		  JOIN orders o ON o.id = l.order_id
		 WHERE l.mode = 'whatsapp' AND l.received_at IS NOT NULL
		   AND l.lat IS NOT NULL AND l.lng IS NOT NULL
		   AND l.received_at <  now() - make_interval(secs => $1)
		   AND l.received_at >  now() - interval '1 day'
		   AND o.origin = 'whatsapp' AND NOT o.dropoff_known
		   AND o.delivered_at IS NULL AND o.closed_at IS NULL
		 ORDER BY l.received_at
		 LIMIT $2`, int(waLocRepairGrace.Seconds()), waLocFollowUpBatch)
	if err != nil {
		s.logger.Warn("إصلاحُ موقع طلب الواتساب: قراءةُ المرشَّحين سقطت", "error", err)
		return 0
	}
	type cand struct {
		order    string
		lat, lng float64
	}
	var cands []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.order, &c.lat, &c.lng); err != nil {
			rows.Close()
			s.logger.Warn("إصلاحُ موقع طلب الواتساب: قراءةُ صفٍّ سقطت", "error", err)
			return 0
		}
		cands = append(cands, c)
	}
	rows.Close()
	n := 0
	for _, c := range cands {
		if ctx.Err() != nil {
			break
		}
		ok, err := s.orders.FillWhatsAppShared(ctx, c.order, c.lat, c.lng)
		if err != nil {
			s.logger.Warn("إصلاحُ موقع طلب الواتساب: الكتابةُ سقطت ثانيةً", "order", c.order, "error", err)
			continue
		}
		if ok {
			s.logger.Info("إصلاحُ موقع طلب الواتساب: كُتب الموقعُ الواصل", "order", c.order)
			n++
		}
	}
	return n
}

// SweepWALocationFollowUp **جولةٌ واحدة** — ويردّ عددَ الرسائل التي وصلت. **يُنادى من الفحص مباشرةً.**
// **وأوّلُها الإصلاح** (`repairWALocations`) — قبل الإعداد والبوت، فلا يوقفه إطفاؤهما.
//
// **الحجزُ قبل الإرسال** (`followup_sent_at`) — جولتان متزامنتان لا تبعثان مرّتين، **ورسالةٌ لم
// تصل لا تُعاد** (كرسالة الرأي): المستلِمُ لا يُلاحَق.
func (s *Server) SweepWALocationFollowUp(ctx context.Context) int {
	s.repairWALocations(ctx)
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
		   AND l.address_received_at IS NULL
		   AND l.whatsapp_sent AND l.expires_at > now()
		   AND l.created_at <= now() - make_interval(mins => $1)
		   AND l.created_at >  now() - make_interval(secs => $2)
		   AND (l.order_id IS NULL OR EXISTS (
		        SELECT 1 FROM orders o
		         WHERE o.id = l.order_id AND o.delivered_at IS NULL AND o.closed_at IS NULL
		           AND NOT o.dropoff_known))
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
			 WHERE id = $1 AND followup_sent_at IS NULL AND received_at IS NULL
			   AND address_received_at IS NULL`, c.id)
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
