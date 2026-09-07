package push

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// ══════════════════════════════════════════════════════════════════════
// **دفعٌ يُعاد أو يُسجَّل — ولا يضيع صامتاً** — `PF-09` · `R23`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **`SendToUser` تُنادى مرّةً واحدةً بعد الحفظ**: **ثلاثةُ إخفاقاتٍ
// وثلاثةُ نداءاتٍ ولا رابع.** **لا صفَّ انتظارٍ ولا حالٌ معلَّقةٌ ولا
// سطرٌ يقول إنّه لم يصل.**
//
// # وما صار — ثلاثُ طبقاتٍ لا تُخلَط
//
//	١ نيّةٌ دائمة        صفُّ `notifications` — وقد أُحكم في `PF-07`
//	٢ محاولةُ نقلٍ وقبولُ المزوّد   **هذه**
//	٣ عرضُ الجهاز        **يحتاج جهازاً** — وعقدُه `P-8`
//
// **ولا يُدَّعى أنّ هاتفاً عرض شيئاً** — **المقبولُ أنّ المزوّدَ قَبِل
// الطلب**، **والاسمُ يقول ذلك.**
//
// # ودلالةُ التسليم: محاولةٌ مرّةً على الأقلّ
//
// **نداءُ شبكةٍ وتثبيتُ قاعدةٍ لا يجتمعان ذرّيّاً.** **فمزوّدٌ قَبِل ثمّ
// مات المنفّذُ قبل أن يُقيَّد القبول ⇒ تُعاد المحاولة.** **وهذا مقبولٌ
// ومُعلَن، ولا يُدَّعى تسليمٌ مرّةً واحدةً بالضبط.**
//
// # ولماذا رمزاً رمزاً
//
// **`Transport.Send` تأخذ رموزاً وتردّ خطأً واحداً** — **فلو أُرسلت
// دفعةً لم يُعرف أيُّها قَبِل وأيُّها انقطع.** **فيُنادى لكلّ صفٍّ
// برمزه**، **وعقدُ النقل لا يتبدّل** — و`FCM` يلفّ عليها واحداً واحداً
// في داخله أصلاً.

// backoff **جدولُ التراجع — قرارُ المالك ٢٠٢٦-٠٩-٠٧.**
//
//	المحاولةُ ١  فوراً
//	  ثمّ ٣٠ث · دقيقة · دقيقتان · ٤ · ٨ · ١٦
//
// **وسبعٌ إجمالاً** — **ونصفُ ساعةٍ تكفي انقطاعاً عابراً**، وما بعدها
// عطبٌ يُعرَض لا يُنتظَر.
var backoff = []time.Duration{
	30 * time.Second,
	1 * time.Minute,
	2 * time.Minute,
	4 * time.Minute,
	8 * time.Minute,
	16 * time.Minute,
}

// MaxAttempts **سبعُ محاولاتٍ إجمالاً** — الأولى فوراً وستُّ إعادات.
const MaxAttempts = 7

// claimLease **أجلُ الحجز** — **فمنفّذٌ مات لا يحبس عملاً أبداً.**
//
// **وأطولُ من أبطأ نداء**: مهلةُ `FCM` عندنا ثوانٍ، **والدقيقتان تحتملان
// تعثّراً بلا أن تُطلق العملَ لمنفّذٍ ثانٍ وهو يُرسل.**
const claimLease = 2 * time.Minute

// deliveryBatch **ما يُطالَب به في الجولة الواحدة.**
const deliveryBatch = 50

// errorClass أصنافُ الخطأ — **تُعَدُّ ويُبحَث فيها، بخلاف نصِّه.**
const (
	classTransport = "TRANSPORT"    // انقطاعٌ أو مهلةٌ أو خمسمئة
	classDeadToken = "DEAD_TOKEN"   // المنصّةُ رفضته نهائيّاً
	classExhausted = "MAX_ATTEMPTS" // نفدت المحاولات
	classNoTransp  = "NO_TRANSPORT" // لا ناقلَ لهذه المنصّة
)

// deliveryJob صفُّ نقلٍ مطالَبٌ به مع مضمون إشعاره.
type deliveryJob struct {
	ID       string
	Token    string
	Platform string
	Attempts int
	Title    string
	Body     string
	Kind     string
	Entity   string
	EntityID string
}

// RunDeliveryWorker **حلقةُ النقل** — تُنادى مرّةً عند الإقلاع.
func (s *Service) RunDeliveryWorker(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.DeliverOnce(ctx)
		}
	}
}

// DeliverOnce **جولةٌ واحدة** — تفريعٌ ثمّ نقل. **ومِعراضُ الفحص.**
//
// **ولا مسارَ شبكةٍ لها** — **ومن فتح نقطةً لأجل اختبارٍ فتحها لغيره.**
func (s *Service) DeliverOnce(ctx context.Context) (fanned, sent, retried, failed int) {
	if s == nil {
		return
	}
	fanned = s.fanOut(ctx)
	sent, retried, failed = s.deliverClaimed(ctx)
	return
}

// ══════════════════════════════════════════════════════════════════════
// **التفريع: من إشعارٍ معلَّقٍ إلى صفوفِ أهدافه**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُفرَّع عند الإدراج** — **لأنّ الأجهزةَ تتبدّل، ولأنّ الإدراجَ
// في معاملة عملٍ لا يحتمل استعلاماً ثانياً على `device_tokens`.**
//
// **والعلامةُ `push_pending` تُكتب في صفّ الإشعار نفسِه** — **فلا فجوةَ
// بين كتابتين تُضيّع التنبيه** (وهي علّةُ `PF-07` بعينها).
func (s *Service) fanOut(ctx context.Context) int {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, user_id::text FROM notifications
		 WHERE push_pending
		 ORDER BY created_at
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED`, deliveryBatch)
	if err != nil {
		s.logger.Error("الدفع: تعذّرت قراءةُ المعلَّق", "error", err)
		return 0
	}
	type pend struct{ id, user string }
	var list []pend
	for rows.Next() {
		var p pend
		if rows.Scan(&p.id, &p.user) == nil {
			list = append(list, p)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		s.logger.Error("الدفع: تعذّرت قراءةُ المعلَّق", "error", err)
		return 0
	}

	var n int
	for _, p := range list {
		if s.fanOutOne(ctx, p.id, p.user) {
			n++
		}
	}
	return n
}

// fanOutOne **صفوفُ الأهداف والعلامةُ تُطفأ في معاملةٍ واحدة.**
//
// **فإن سقط الإدراجُ بقيت العلامةُ** — **والجولةُ التاليةُ تُعيد.**
func (s *Service) fanOutOne(ctx context.Context, notifID, userID string) bool {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// **والأجهزةُ الحيّةُ وحدَها** — والميّتُ بالزمن لا يُقرأ، كما في
	// `tokensOf`. **ومنصّةٌ بلا ناقلٍ تُفرَّع أيضاً**: **الناقلُ يُضاف
	// غداً والصفُّ ينتظره**، ولا يُخترَع صمتٌ اليوم.
	//
	// **وشرطُ التطبيق كما في `tokensOf` حرفاً**: **فارغةٌ تعني كلَّ
	// الأجهزة**، **والجهازُ المجهولُ تطبيقُه (`app = ''`) داخلٌ في كلّ
	// حال** — سُجّل من نسخةٍ قديمةٍ لا تُصرّح، **ويجب أن يستقبل لا أن
	// يصمت.**
	if _, err := tx.Exec(ctx, `
		INSERT INTO notification_deliveries
		       (notification_id, token, platform)
		SELECT n.id, d.token, d.platform
		  FROM device_tokens d
		  JOIN notifications n ON n.id = $1::uuid
		 WHERE d.user_id = $2::uuid
		   AND d.last_seen_at > now() - $3::interval
		   AND (cardinality(n.push_apps) = 0 OR d.app = ''
		        OR d.app = ANY(n.push_apps))
		ON CONFLICT (notification_id, token) DO NOTHING`,
		notifID, userID, staleAfter.String()); err != nil {
		s.logger.Error("الدفع: تعذّر تفريعُ الأهداف",
			"notification", notifID, "error", err)
		return false
	}
	if _, err := tx.Exec(ctx,
		`UPDATE notifications SET push_pending = false WHERE id = $1::uuid`,
		notifID); err != nil {
		return false
	}
	return tx.Commit(ctx) == nil
}

// ══════════════════════════════════════════════════════════════════════
// **النقل: مطالبةٌ تُثبَّت · نداءٌ خارجَ المعاملة · نتيجةٌ تُقيَّد**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُمسَك معاملةٌ على شبكة** — **قفلٌ ينتظر غوغل قفلٌ ينتظر الأبد.**
func (s *Service) deliverClaimed(ctx context.Context) (sent, retried, failed int) {
	jobs := s.claim(ctx)
	for _, j := range jobs {
		switch s.attempt(ctx, j) {
		case outcomeAccepted:
			sent++
		case outcomeRetry:
			retried++
		default:
			failed++
		}
	}
	return
}

type outcome int

const (
	outcomeAccepted outcome = iota
	outcomeRetry
	outcomeTerminal
)

// claim **يطالب بعملٍ مستحقٍّ ويثبّت المطالبةَ قبل أيّ نداء.**
//
// **والحجزُ بأجلٍ لا بعلَم**: **منفّذٌ مات وعلَمُه مرفوعٌ يحبس العملَ
// أبداً** — **والأجلُ يُطلقه من تلقائه.**
func (s *Service) claim(ctx context.Context) []deliveryJob {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		WITH due AS (
			SELECT d.id
			  FROM notification_deliveries d
			 WHERE d.state = 'pending'
			   AND d.next_attempt_at <= now()
			   AND (d.claimed_until IS NULL OR d.claimed_until < now())
			 ORDER BY d.next_attempt_at
			 LIMIT $1
			 FOR UPDATE SKIP LOCKED
		)
		UPDATE notification_deliveries d
		   SET attempts      = d.attempts + 1,
		       claimed_until = now() + $2::interval,
		       updated_at    = now()
		  FROM due, notifications n
		 WHERE d.id = due.id AND n.id = d.notification_id
		RETURNING d.id::text, d.token, d.platform, d.attempts,
		          n.title, n.body, n.kind, n.entity, n.entity_id`,
		deliveryBatch, claimLease.String())
	if err != nil {
		s.logger.Error("الدفع: تعذّرت المطالبة", "error", err)
		return nil
	}
	var out []deliveryJob
	for rows.Next() {
		var j deliveryJob
		if err := rows.Scan(&j.ID, &j.Token, &j.Platform, &j.Attempts,
			&j.Title, &j.Body, &j.Kind, &j.Entity, &j.EntityID); err != nil {
			rows.Close()
			return nil
		}
		out = append(out, j)
	}
	rows.Close()
	if rows.Err() != nil {
		return nil
	}
	// **والمطالبةُ تُثبَّت قبل النداء** — **فمن نادى ثمّ ثبّت أعاد
	// النداءَ بعد موتٍ بلا عدٍّ، وهي حلقةٌ لا تنتهي.**
	if tx.Commit(ctx) != nil {
		return nil
	}
	return out
}

// attempt **نداءٌ واحدٌ خارجَ المعاملة، ثمّ تُقيَّد نتيجتُه.**
func (s *Service) attempt(ctx context.Context, j deliveryJob) outcome {
	t, ok := s.transports[j.Platform]
	if !ok {
		// **ولا ناقلَ لهذه المنصّة** — **لا يُعاد ولا يُدَّعى قبول.**
		s.finish(ctx, j, "failed", classNoTransp,
			fmt.Sprintf("لا ناقلَ لمنصّة %q", j.Platform))
		return outcomeTerminal
	}

	msg := Message{
		Title: j.Title,
		Body:  j.Body,
		Data: map[string]string{
			"kind":      j.Kind,
			"entity":    j.Entity,
			"entity_id": j.EntityID,
		},
		Urgent: j.Kind == kindOrder,
	}
	dead, err := t.Send(ctx, []string{j.Token}, msg)

	switch {
	// ══════════════════════════════════════════════════════════════
	// **رمزٌ ماتَ — يُحذف ولا يُعاد**
	// ══════════════════════════════════════════════════════════════
	//
	// **ورمزٌ مرفوضٌ يبقى يُنادى عليه في كلّ حدثٍ إلى الأبد** — يستهلك
	// حصّةً ويبطئ كلَّ إشعارٍ لصاحبه.
	//
	// **وأخطاءُ التهيئة ليست موتَ رمز** (`٤٠٣` مثلاً): **الناقلُ
	// يميّزها ولا يعدّها ميتة** — **وحذفُ الرموز عليها يمحو أجهزةَ
	// المنصّة كلَّها والسببُ سطرٌ في إعدادات غوغل.**
	case len(dead) > 0:
		s.dropDead(ctx, dead)
		s.finish(ctx, j, "dead_token", classDeadToken, "المنصّةُ رفضت الرمزَ نهائيّاً")
		return outcomeTerminal

	case err != nil:
		// **عابرٌ**: مهلةٌ أو انقطاعٌ أو خمسمئةٌ أو خنقُ حصّة.
		if j.Attempts >= MaxAttempts {
			s.finish(ctx, j, "failed", classExhausted, errText(err))
			s.logger.Warn("الدفع: نفدت المحاولات",
				"delivery", j.ID, "attempts", j.Attempts, "error", err)
			return outcomeTerminal
		}
		s.reschedule(ctx, j, errText(err))
		return outcomeRetry

	default:
		s.accept(ctx, j)
		return outcomeAccepted
	}
}

// accept **المزوّدُ قَبِل الطلب** — **ولا يعني أنّ هاتفاً عرضه.**
func (s *Service) accept(ctx context.Context, j deliveryJob) {
	if _, err := s.db.Exec(ctx, `
		UPDATE notification_deliveries
		   SET state = 'accepted', provider_accepted_at = now(),
		       claimed_until = NULL, last_error = '', last_error_class = '',
		       updated_at = now()
		 WHERE id = $1::uuid`, j.ID); err != nil {
		// **وقبولٌ لم يُقيَّد يُعاد إرسالُه** — **وتلك دلالةُ «مرّةً
		// على الأقلّ» المُعلَنة، لا عطبٌ خفيّ.**
		s.logger.Error("الدفع: تعذّر قيدُ القبول", "delivery", j.ID, "error", err)
	}
}

// reschedule **موعدٌ تالٍ بتراجعٍ أُسّيّ** — **ولا حلقةَ محمومة.**
func (s *Service) reschedule(ctx context.Context, j deliveryJob, why string) {
	// **المحاولةُ الأولى تنتظر `backoff[0]`** — والفهرسُ من صفر.
	i := j.Attempts - 1
	if i < 0 {
		i = 0
	}
	if i >= len(backoff) {
		i = len(backoff) - 1
	}
	if _, err := s.db.Exec(ctx, `
		UPDATE notification_deliveries
		   SET next_attempt_at = now() + $2::interval,
		       claimed_until = NULL,
		       last_error = $3, last_error_class = $4, updated_at = now()
		 WHERE id = $1::uuid`,
		j.ID, backoff[i].String(), truncErr(why), classTransport); err != nil {
		s.logger.Error("الدفع: تعذّرت جدولةُ الإعادة", "delivery", j.ID, "error", err)
	}
}

// finish **حالٌ نهائيّةٌ تُقيَّد بسببها** — **ولا تختفي صامتة.**
//
// **والإشعارُ نفسُه يبقى في التطبيق** — **فسقوطُ النقل لا يمحو الخبر.**
func (s *Service) finish(ctx context.Context, j deliveryJob, state, class, why string) {
	if _, err := s.db.Exec(ctx, `
		UPDATE notification_deliveries
		   SET state = $2, claimed_until = NULL,
		       last_error = $3, last_error_class = $4, updated_at = now()
		 WHERE id = $1::uuid`,
		j.ID, state, truncErr(why), class); err != nil {
		s.logger.Error("الدفع: تعذّر قيدُ الحال النهائيّة",
			"delivery", j.ID, "error", err)
	}
}

// PendingSummary **ما ينتظر وما أخفق** — **لِيُقرأ لا لِيختفي.**
//
// **ولا لوحةَ جديدةٌ تُبنى في هذه الدورة** — **والحالُ مقروءةٌ ومقيسة،
// وهو أدنى ما يُشترَط.**
type PendingSummary struct {
	Pending   int    `json:"pending"`
	Failed    int    `json:"failed"`
	DeadToken int    `json:"dead_token"`
	Accepted  int    `json:"accepted"`
	OldestDue string `json:"oldest_due,omitempty"`
}

// DeliveryHealth حالُ النقل — **يُقرأ في السجلّ وفي الفحص.**
func (s *Service) DeliveryHealth(ctx context.Context) (PendingSummary, error) {
	var out PendingSummary
	var oldest *time.Time
	err := s.db.QueryRow(ctx, `
		SELECT
		  count(*) FILTER (WHERE state = 'pending'),
		  count(*) FILTER (WHERE state = 'failed'),
		  count(*) FILTER (WHERE state = 'dead_token'),
		  count(*) FILTER (WHERE state = 'accepted'),
		  min(next_attempt_at) FILTER (WHERE state = 'pending')
		  FROM notification_deliveries`).
		Scan(&out.Pending, &out.Failed, &out.DeadToken, &out.Accepted, &oldest)
	if oldest != nil {
		out.OldestDue = oldest.UTC().Format(time.RFC3339)
	}
	return out, err
}

// MarkPending **يُعلِّم إشعاراتٍ للدفع — في معاملة كاتبِها.**
//
// **ولا يُنادى بعد التثبيت** — **الفجوةُ بين كتابتين هي `PF-07` نفسُها.**
func MarkPending(ctx context.Context, q dbtx.Querier, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := q.Exec(ctx, `
		UPDATE notifications SET push_pending = true
		 WHERE id = ANY($1::uuid[])`, ids)
	return err
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "مهلةٌ انقضت: " + err.Error()
	}
	return err.Error()
}

func truncErr(s string) string {
	const max = 500
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// kindOrder **نسخةٌ من `notifications.KindOrder`** — **ولا تُستورَد
// الحزمةُ**: `notifications` تعرف `push` ولا العكس، **ودورةُ استيرادٍ
// أسوأُ من ثابتٍ مكرَّرٍ يحرسه فحص.**
const kindOrder = "order"

// ══════════════════════════════════════════════════════════════════════
// **النبضة: محاولةُ الفور بلا انتظارِ صاحب العمل**
// ══════════════════════════════════════════════════════════════════════
//
// **قرارُ المالك «المحاولةُ ١ فوراً»** — **ولا يُنتظَر غوغل في مسار
// إنشاء طلب.** **فتُوقَظ جولةٌ في خيطٍ مستقلّ.**
//
// **ولا شيءَ يعتمد عليها**: **النيّةُ مكتوبةٌ في الصفّ**، **والجولةُ
// الدوريّةُ تلتقط ما فات.** **فنبضةٌ ضاعت تأخيرٌ لا ضياع.**
//
// **وواحدةٌ تكفي**: **خانةٌ واحدةٌ لا طابور** — **وألفُ إشعارٍ في ثانيةٍ
// لا يُولّد ألفَ جولة**، والجولةُ الواحدةُ تأخذ الدفعةَ كلَّها.
func (s *Service) Kick(ctx context.Context) {
	if s == nil {
		return
	}
	select {
	case s.kick <- struct{}{}:
	default:
		// **جولةٌ موقظةٌ سلفاً** — ولا تُكدَّس.
		return
	}
	// **وسياقٌ مستقلٌّ لا سياقُ الطلب**: **طلبُ HTTP ينتهي فيُلغى
	// سياقُه، والنقلُ عملٌ خلفَه لا فيه.**
	go func() {
		defer func() { <-s.kick }()
		bg, cancel := context.WithTimeout(context.Background(), kickBudget)
		defer cancel()
		s.DeliverOnce(bg)
	}()
}

// kickBudget **سقفُ الجولة الموقظة** — **ولا خيطٌ يعيش أبداً.**
//
// **وقصيرٌ عمداً**: **النبضةُ تشارك مَسبَحَ الاتّصالات مع مسارات
// الطلبات** — **وجولةٌ تمسك اتّصالاً دقيقتين تُجوّع طلباً حيّاً.**
//
// **وما لم تُنجزه تُنجزه الجولةُ الدوريّة** — **فالنبضةُ تعجيلٌ لا
// مصدرُ حقيقة.**
const kickBudget = 15 * time.Second
