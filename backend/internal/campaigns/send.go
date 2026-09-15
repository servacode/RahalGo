package campaigns

// ══════════════════════════════════════════════════════════════════════
// **الإرسال — مرّةً واحدةً مهما تكرّر النداء** (`NT-06`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// # وكيف تصير الحملةُ إرسالاً واحداً
//
// **وثلاثةُ أقفالٍ متتابعة:**
//
//	١ · `idem_key`                 ⇒ ضغطتان تصنعان صفّاً واحداً
//	٢ · الانتقالُ المشروطُ للحال    ⇒ منفّذٌ واحدٌ يملك التنفيذ
//	٣ · `campaign_recipients`      ⇒ إعادةٌ بعد سقوطٍ لا تُرسل ثانيةً
//
// **والثالثُ هو الذي ينجو من موتٍ في منتصف التوزيع** — **فمن أُرسل
// إليه صفُّه مكتوب، ومن لم يُرسَل لا صفَّ له.**

import (
	"context"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **الحجزُ بأجلٍ لا بعلَم** — **كحجز صفوف الدفع** (`PF-09`)
// ══════════════════════════════════════════════════════════════════════
const claimLease = 5 * time.Minute

// claim **ينتقل بالحملة إلى `sending` إن ملكها.**
//
// **والانتقالُ مشروطٌ في `UPDATE` نفسِه** — **ولا يُقرأ ثمّ يُكتب**:
// **وبين القراءة والكتابة يدخل منفّذٌ ثانٍ فيُرسل الحملةَ مرّتين.**
//
// **ويُقبَل المستحقُّ من `draft` و`scheduled`** — **ومن `sending`
// هجرها منفّذُها**: **مضى أجلُ حجزها فهي عملٌ متروك.**
func (s *Service) claim(ctx context.Context, id string) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE campaigns
		   SET status = 'sending', claimed_at = now(), updated_at = now()
		 WHERE id = $1::uuid
		   AND (status IN ('draft', 'scheduled')
		        OR (status = 'sending' AND claimed_at < now() - $2::interval))`,
		id, claimLease.String())
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Send **يُرسل حملةً الآن.**
//
// **والنداءُ الثاني على حملةٍ أُرسلت لا يُرسل شيئاً** — **يردّها كما
// هي**: **وهو ما يقع حين تسقط الشبكةُ بعد الإرسال فيُعيد الأدمن.**
func (s *Service) Send(ctx context.Context, id string) (*Campaign, error) {
	c, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	switch c.Status {
	case StatusSent:
		// **وقد أُرسلت** — **فلا تُرسل ثانية.**
		return c, nil
	case StatusCancelled:
		return nil, ErrNotCancellable
	}
	ok, err := s.claim(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		// **ومنفّذٌ آخرُ يملكها الآن** — **وتُقرأ حالُها كما هي.**
		return s.Get(ctx, id)
	}
	return s.execute(ctx, c)
}

// execute **التوزيعُ نفسُه** — **وقد صارت الحملةُ ملكَ هذا المنفّذ.**
func (s *Service) execute(ctx context.Context, c *Campaign) (*Campaign, error) {
	// ══════════════════════════════════════════════════════════════════
	// **وخبرُ الوصول يُعاد تقييمُه قبل أن يُقال** (`SI-N-06`)
	// ══════════════════════════════════════════════════════════════════
	//
	// **ومن جدول «وصلت الخدمةُ إلى منطقتك» أمسِ ثمّ أُغلقت المنطقةُ
	// اليومَ يرسل وعداً كاذباً** — **ويفتح الزبونُ التطبيقَ فلا يجد
	// شيئاً.**
	//
	// **والاشتراكُ لا يُعلَّم مُخبَراً حينئذٍ** (`SI-N-07`) — **فيبقى
	// قائماً ليُخبَر حين تصل حقّاً.**
	if c.AudienceType == AudienceInterest && s.Serviceable != nil {
		ok, err := s.Serviceable(ctx, c.AudienceRef)
		if err != nil || !ok {
			s.fail(ctx, c.ID, "target not serviceable: "+c.AudienceRef)
			return s.Get(ctx, c.ID)
		}
	}

	// ══════════════════════════════════════════════════════════════════
	// **والعرضُ يُسأل عنه ثانيةً لحظةَ الإرسال** (`EN-02`)
	// ══════════════════════════════════════════════════════════════════
	//
	// **وحملةٌ جُدولت لعرضٍ انتهى بينهما وعدٌ كاذب** — **والسؤالُ عند
	// الإنشاء وحدَه لا يكفي.**
	if c.DestType != nil && *c.DestType == DestOffer && c.DestID != nil && s.LiveOffer != nil {
		live, err := s.LiveOffer(ctx, *c.DestID)
		if err != nil || !live {
			s.fail(ctx, c.ID, "offer not live")
			return s.Get(ctx, c.ID)
		}
	}

	ids, err := s.Audience(ctx, c.AudienceType, c.AudienceRef)
	if err != nil {
		s.fail(ctx, c.ID, "audience: "+err.Error())
		return nil, err
	}

	quiet := s.QuietOf(ctx)
	cap := s.CapOf(ctx)
	now := s.Now()
	// **وساعةُ الهدوء تُقاس مرّةً للحملة كلِّها** — **ولا يُفرَّق بين
	// اثنين وصلتهما في الثانية نفسِها.**
	deferByQuiet := Engagement(c.Category) && quiet.InQuiet(now)

	var inbox, deferred int
	for _, uid := range ids {
		if deferByQuiet {
			deferred++
			continue
		}
		// ══════════════════════════════════════════════════════════
		// **والسقفُ يُقرأ لكلّ حساب** (`NT-14`)
		// ══════════════════════════════════════════════════════════
		//
		// **ومن بلغ سقفَه اليومَ لا يُزعَج ثانيةً** — **ولا يُعدّ
		// خبرُ طلبٍ في هذا السقف**: **هذا الجدولُ يعدّ التفاعلَ
		// وحدَه.**
		if Engagement(c.Category) {
			sent, err := s.sentToday(ctx, uid, now)
			if err == nil && !UnderCap(sent, cap) {
				deferred++
				continue
			}
		}
		// ══════════════════════════════════════════════════════════
		// **وصفُّ المتلقّي يُكتب أوّلاً** (`NT-19`، `SI-N-09`)
		// ══════════════════════════════════════════════════════════
		//
		// **ومن كُتب صفُّه لا يُرسَل إليه ثانيةً في إعادةٍ بعد
		// سقوط** — **والإدراجُ هو القفل.**
		claimed, err := s.claimRecipient(ctx, c.ID, uid)
		if err != nil || !claimed {
			continue
		}
		s.notif.Notify(ctx, uid, kindOf(c), c.Title, c.Body, entityOf(c), idOf(c))
		inbox++
		// ══════════════════════════════════════════════════════════
		// **ويُعلَّم الاشتراكُ بعد الإرسال لا قبله** (`SI-N-08`)
		// ══════════════════════════════════════════════════════════
		//
		// **ومن علّم قبل أن يُرسل ثمّ سقط ترك إنساناً مكتوباً أنّه
		// أُخبر ولم يُخبَر** — **ولا يعود إليه الخبرُ أبداً.**
		if c.AudienceType == AudienceInterest {
			_, _ = s.db.Exec(ctx, `
				UPDATE coverage_requests SET notified_at = now(), updated_at = now()
				 WHERE user_id = $1::uuid AND kind = 'service_interest'
				   AND target_key = $2 AND active`, uid, c.AudienceRef)
		}
	}

	// **وما أُجّل كلُّه يبقى مجدولاً لأوّل وقتٍ مسموح** — **ولا
	// يُقال «أُرسلت» لحملةٍ لم تُرسَل.**
	if deferByQuiet && inbox == 0 {
		next := quiet.NextAllowed(now)
		_, _ = s.db.Exec(ctx, `
			UPDATE campaigns SET status = 'scheduled', scheduled_at = $2,
			       claimed_at = NULL, targeted = $3, deferred = $4, updated_at = now()
			 WHERE id = $1::uuid`, c.ID, next, len(ids), deferred)
		return s.Get(ctx, c.ID)
	}

	if _, err := s.db.Exec(ctx, `
		UPDATE campaigns SET status = 'sent', sent_at = now(), claimed_at = NULL,
		       targeted = $2, inbox_created = $3, deferred = $4, updated_at = now()
		 WHERE id = $1::uuid`, c.ID, len(ids), inbox, deferred); err != nil {
		return nil, err
	}
	return s.Get(ctx, c.ID)
}

// claimRecipient **يحجز إنساناً لهذه الحملة** — **ويردّ `false` لمن حُجز.**
func (s *Service) claimRecipient(ctx context.Context, campaignID, userID string) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		INSERT INTO campaign_recipients (campaign_id, user_id)
		VALUES ($1::uuid, $2::uuid)
		ON CONFLICT (campaign_id, user_id) DO NOTHING`, campaignID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// sentToday **كم حملةَ تفاعلٍ بلغته منذ فجر اليوم بتوقيت الخدمة.**
func (s *Service) sentToday(ctx context.Context, userID string, now time.Time) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM campaign_recipients
		 WHERE user_id = $1::uuid
		   AND sent_at >= date_trunc('day', now() AT TIME ZONE 'Asia/Damascus')
		       AT TIME ZONE 'Asia/Damascus'`, userID).Scan(&n)
	return n, err
}

func (s *Service) fail(ctx context.Context, id, why string) {
	_, _ = s.db.Exec(ctx, `
		UPDATE campaigns SET status = 'failed', error = $2, claimed_at = NULL,
		       updated_at = now() WHERE id = $1::uuid`, id, clip(why, 300))
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ══════════════════════════════════════════════════════════════════════
// **وخبرُ الحملة نوعُه `promo`** — **لا `account` ولا `order`**
// ══════════════════════════════════════════════════════════════════════
//
// **ومن رشّح صندوقَه ليرى أخبارَ حسابه لا يريد عرضاً بينها** —
// **والفرزُ في الصندوق يقوم على النوع.**
const KindPromo = "promo"

func kindOf(*Campaign) string { return KindPromo }

// entityOf و idOf **الوجهةُ تُحمَل في الخبر نفسِه.**
//
// **و`entity`/`entity_id` حقلان قائمان في الإشعار** — **ولا حقلَ
// ثالثٌ يُخترَع لشيءٍ له موضع.**
func entityOf(c *Campaign) string {
	if c.DestType == nil || *c.DestType == DestHome {
		return ""
	}
	return *c.DestType
}

func idOf(c *Campaign) string {
	if c.DestID == nil {
		return ""
	}
	return *c.DestID
}

// ══════════════════════════════════════════════════════════════════════
// **العاملُ — يلتقط ما استحقّ** (البند ٧)
// ══════════════════════════════════════════════════════════════════════
//
// **ولا جدولةَ في ذاكرة العمليّة** — **ومن أعاد تشغيلَ الخادم وجد
// المستحقَّ في القاعدة كما تركه.**
//
// **والالتقاطُ بـ`SKIP LOCKED`** — **فعاملان لا يأخذان حملةً واحدة**،
// **وهو نمطُ عامل الدفع نفسُه.**
func (s *Service) DueOnce(ctx context.Context) int {
	rows, err := s.db.Query(ctx, `
		SELECT id::text FROM campaigns
		 WHERE status = 'scheduled' AND scheduled_at <= now()
		 ORDER BY scheduled_at
		 LIMIT 20
		 FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return 0
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()

	var n int
	for _, id := range ids {
		if _, err := s.Send(ctx, id); err == nil {
			n++
		}
	}
	return n
}

// RunWorker **حلقةُ المستحقّ** — **بإيقاع المنصّة نفسِه.**
func (s *Service) RunWorker(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.DueOnce(ctx)
		}
	}
}
