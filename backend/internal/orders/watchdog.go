package orders

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// Alert تنبيه تصعيد لطلب عالق — يظهر أحمر في غرفة العمليات.
type Alert struct {
	OrderID       string  `json:"order_id"`
	Number        int64   `json:"number"`
	Status        string  `json:"status"`
	MerchantName  string  `json:"merchant_name"`
	CustomerPhone string  `json:"customer_phone"`
	Reason        string  `json:"reason"` // emergency | no_accept | not_sent | no_driver | too_long
	Minutes       float64 `json:"minutes"`
	// Since **منذ متى يصدق السبب** — والشاشةُ تعدّ الدقائقَ منه حيّةً (البند ١٩).
	//
	// **وكان الرقمُ يُرسل مرّةً فيبقى «منذ ٣٢٣ دقيقة» ساعاتٍ** — الراصدُ لا
	// يبثّ إلّا إن تبدّلت القائمة.
	Since time.Time `json:"since"`
	// AckedAt **متى ضغط موظّفٌ «أنا عليه»** — والفارغُ لم يضغط أحد، فالتذكيرُ
	// يتكرّر (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»، البند ٤).
	AckedAt *time.Time `json:"acked_at"`
	// Reminders **كم مرّةً ذُكّر المكتبُ بالسبب نفسِه** بعد الإنذار الأوّل.
	Reminders int `json:"reminders"`
}

// Alerts يفحص الطلبات العالقة وفق المهل الديناميكية (PLAN §6.1).
//
// ══════════════════════════════════════════════════════════════════════
// **والشرطُ من `board.go` لا من هنا** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٥)
// ══════════════════════════════════════════════════════════════════════
//
// **كان نصٌّ هنا ونصٌّ في بطاقة الرئيسيّة ونصٌّ في رابطها** — فتقول البطاقةُ
// اثنين وتعرض القائمةُ خمسة. **وصار الشرطُ دالّةً واحدة** (`StuckReasonSQL`)
// يقرؤها الصندوقُ والإنذارُ وفلترُ «عالق» في اللوحة.
//
// **والخاصُّ بلا متجر والتوصيلةُ بلا زبون** — والوصلتان يسريان لذلك: وصلةٌ
// داخليّةٌ تُسقطهما فيبقيان عالقين ولا يعلم بهما المكتب. (قِيس ٢٠٢٦-٠٩-٢٩.)
func (s *Service) Alerts(ctx context.Context) ([]Alert, error) {
	l := s.StuckLimitsOf(ctx)
	rows, err := s.db.Query(ctx, `
		SELECT o.id, o.number, o.status,
			-- **والخاصُّ لا متجرَ له** — فاسمُه طلبٌ خاصٌّ لا NULL.
			COALESCE(m.name, 'طلبٌ خاصّ') AS merchant_name, COALESCE(cu.phone, o.recipient_phone, ''),
			r.reason, r.since,
			round(EXTRACT(EPOCH FROM now() - r.since) / 60),
			-- **«أنا عليه» تخصّ السببَ الذي أُنذر به** — سببٌ جديدٌ لم يستلمه أحد.
			CASE WHEN o.alerted_reason IS NOT DISTINCT FROM r.reason THEN o.alert_ack_at END,
			CASE WHEN o.alerted_reason IS NOT DISTINCT FROM r.reason THEN o.alert_repeats ELSE 0 END
		FROM orders o
		CROSS JOIN LATERAL (SELECT `+StuckReasonSQL(l)+` AS reason,
		                           `+StuckSinceSQL(l)+` AS since) r
		LEFT JOIN merchants m ON m.id = o.merchant_id
		LEFT JOIN users cu ON cu.id = o.customer_id
		WHERE o.closed_at IS NULL AND r.reason IS NOT NULL
		ORDER BY r.since, o.number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := []Alert{}
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.OrderID, &a.Number, &a.Status, &a.MerchantName,
			&a.CustomerPhone, &a.Reason, &a.Since, &a.Minutes, &a.AckedAt, &a.Reminders); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

// RunWatchdog حلقة الراصد: فحص دوري وبث التنبيهات لغرفة العمليات.
// يبث فقط عند تغير مجموعة التنبيهات — لا إزعاج متكرراً بلا جديد.
//
// **والبثّ يصل الشاشات المفتوحة وحدها.** فمن أغلق اللوحة ليلاً لم يصله شيء،
// والطلب يبقى عالقاً حتى الصباح. فصار الراصد يُنشئ كذلك **إشعاراً باقياً**
// يجده الموظّف حين يفتح، مرّةً واحدة لكل طلب (`alerted_at` علامةٌ في القاعدة
// لا في الذاكرة: ذاكرةٌ تُفقد بإعادة التشغيل فيُعاد إنذار الجميع دفعةً).
func (s *Service) RunWatchdog(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastKey := ""
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// **انتقالُ الدور مع نبضة الراصد** — لا مع نداءِ سائقٍ للطابور:
			// لو انتظرنا من يسأل لبقي طلبٌ محجوزاً لسائقٍ نائمٍ حتى يفتح
			// غيرُه التطبيق. **والزبونُ لا ينتظر أن يتذكّر أحدٌ أن ينظر.**
			//
			// **والعروضُ المنقضيةُ لها حلقتُها السريعة** (`RunOfferSweeper`،
			// ٢٠٢٦-١٠-٠٢) — **وهنا ما ينتظر بلا عرضٍ وحدَه.**
			s.SweepWaitingOffers(ctx)
			// **ويُقبل ما نُسي في انتظار المكتب** — انظر `sweepAutoAccept`.
			s.sweepAutoAccept(ctx)
			alerts, err := s.Alerts(ctx)
			if err != nil {
				s.logger.Error("watchdog scan failed", "error", err)
				continue
			}
			// **والإنذارُ في كلّ نبضة** (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة
			// التشغيل»، البند ٤): التذكيرُ يحين بالوقت لا بتبدّل القائمة،
			// **والقاعدةُ تقرّر من يستحقّه** (`escalateOne`) — فلا تكرارَ قبل وقته.
			if len(alerts) > 0 {
				s.escalate(ctx, alerts)
			}
			key := ""
			for _, a := range alerts {
				key += a.OrderID + a.Reason + "|"
			}
			if key == lastKey {
				continue
			}
			lastKey = key
			s.pub.Publish("ops", map[string]any{"type": "alerts", "alerts": alerts})
			if len(alerts) > 0 {
				s.logger.Warn("watchdog escalation", "count", len(alerts))
			}
		}
	}
}

// escalate يُنشئ إشعاراً باقياً لكل طلبٍ عَلِق ولم يُنذَر بعد.
//
// والعلامة تُوضع **قبل** الإشعار لا بعده: لو أُشعر ثم فشلت الكتابة لأُعيد
// الإشعار كل ثلاثين ثانية إلى الأبد. وإشعارٌ ضائع أهون من إشعارٍ يتكرّر
// مئتي مرّة — الثاني يُدرَّب المستخدم على تجاهله فيصير كالصمت.
// EscalateAlertsOnce جولةُ إنذارٍ واحدة — **مِعراضُ الفحص.**
//
// **والمصفوفةُ كانت تقول `TESTABILITY SEAM REQUIRED`** — **ولا بابَ
// إداريٌّ للراصد، فبقي `PF-07` غيرَ مُثبَت.**
//
// **ولا تفتح باباً في المنتَج**: **دالّةٌ مصدَّرةٌ ينادها الفحصُ
// مباشرةً**، ولا مسارَ شبكةٍ لها. **ومن فتح نقطةً لأجل اختبارٍ فتحها
// لغيره.**
func (s *Service) EscalateAlertsOnce(ctx context.Context) {
	alerts, err := s.Alerts(ctx)
	if err != nil {
		s.logger.Error("watchdog scan failed", "error", err)
		return
	}
	s.escalate(ctx, alerts)
}

func (s *Service) escalate(ctx context.Context, alerts []Alert) {
	if s.notify == nil {
		return
	}
	for _, a := range alerts {
		s.escalateOne(ctx, a)
	}
}

// escalateOne **وسمٌ ونيّةٌ في معاملةٍ واحدة** — `PF-07` · `R22`.
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا لا يُسبَق الوسمُ الإشعارَ**
// ══════════════════════════════════════════════════════════════════════
//
// **كان الوسمُ يُكتب أوّلاً ثمّ يُشعَر** — **والوسمُ يمنع إعادةَ
// المحاولة.** فإن سقط الإشعارُ أو مات المنفّذُ بينهما **صمت الإنذارُ
// إلى الأبد**، ولا يراه زبونٌ ولا إدارة.
//
// **و`alerted_at` يقول «أُنذر» وهو لا يعني إلّا «قرّرنا أن نُنذر».**
//
// **فصارا معاً**: **الوسمُ يُكتب والنيّةُ الدائمةُ تُنشأ في معاملةٍ
// واحدةٍ تُثبَّت مرّةً** — **فإن سقطت النيّةُ سقط الوسمُ وأُعيدت
// المحاولةُ في الجولة التالية.**
//
// **والوسمُ أوّلاً داخلَ المعاملة عمداً**: **قفلُه يمنع راصدين من
// إنشاء نيّتين** — والثاني يجد `RowsAffected == 0` أو ينتظر ثمّ يجده.
//
// **ولا مكتبَ يُنذَر ⇒ لا وسم**: **وسمٌ بلا مُنذَرٍ كذبٌ يمنع إنذاراً
// حين يُوظَّف أحد.**
//
// **والدفعةُ والبثُّ بعد التثبيت** — **شبكةٌ لا تدخل معاملة**، وعقدُ
// وصولها `PF-09`.
func (s *Service) escalateOne(ctx context.Context, a Alert) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Error("watchdog: تعذّر فتحُ معاملة", "order", a.OrderID, "error", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// ══════════════════════════════════════════════════════════════════
	// **والإنذارُ يتجدّد حين يتبدّل سببُه** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٩)
	// ══════════════════════════════════════════════════════════════════
	//
	// كان `alerted_at IS NULL` — **مرّةً للطلب كلِّه**: طلبٌ أُنذر لأنّه لم يُقبل
	// ثمّ علق «بلا سائق» لا يصل عنه إنذارٌ ثانٍ. **والسببُ يُحفظ مع الوسم**،
	// والسببُ نفسُه لا يُنذَر مرّتين.
	// ══════════════════════════════════════════════════════════════════
	// **والتذكيرُ يتكرّر حتّى يتصرّف أحد** (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»، البند ٤)
	// ══════════════════════════════════════════════════════════════════
	//
	// كان الإنذارُ مرّةً للسبب — **فمن لم يرَه ترك ‎#1400‎ عالقاً خمسَ ساعات.**
	// وصار السببُ نفسُه يُعاد كلَّ `ops.stuck_reminder_min` دقيقة **ما لم**:
	//
	//	تتغيّر حالةُ الطلب منذ آخر إنذار   (alerted_status)
	//	يتغيّر سائقُه                     (alerted_driver_id)
	//	يضغط موظّفٌ «أنا عليه»             (alert_ack_at)
	//
	// **والسببُ الجديدُ كما كان**: إنذارٌ فوراً، ويمسح «أنا عليه» القديمة.
	// **والطارئُ لا يُكرَّر من هنا** — له شريطُه وصوتُه حتّى يُستلَم.
	// **وما أُنذر قبل هذه الأعمدة** (`alerted_status` فارغ) يُذكَّر به — لا نعرف
	// أتغيّر أم لا، **وتذكيرٌ زائدٌ أهونُ من طلبٍ منسيّ.**
	every := s.settingInt(ctx, "ops.stuck_reminder_min")
	var repeats int
	err = tx.QueryRow(ctx,
		`UPDATE orders SET alerted_at = now(), alerted_reason = $2,
		        alerted_status = status, alerted_driver_id = driver_id,
		        alert_repeats = CASE WHEN alerted_at IS NOT NULL
		                              AND alerted_reason IS NOT DISTINCT FROM $2
		                             THEN alert_repeats + 1 ELSE 0 END,
		        alert_ack_at = CASE WHEN alerted_reason IS DISTINCT FROM $2
		                            THEN NULL ELSE alert_ack_at END,
		        alert_ack_by = CASE WHEN alerted_reason IS DISTINCT FROM $2
		                            THEN NULL ELSE alert_ack_by END
		  WHERE id = $1 AND (alerted_at IS NULL OR alerted_reason IS DISTINCT FROM $2
		        OR ($3::int > 0 AND $2 <> '`+StuckEmergency+`'
		            AND alerted_at < now() - make_interval(mins => $3::int)
		            AND alert_ack_at IS NULL
		            AND (alerted_status IS NULL
		                 OR (alerted_status = status
		                     AND alerted_driver_id IS NOT DISTINCT FROM driver_id))))
		  RETURNING alert_repeats`,
		a.OrderID, a.Reason, every).Scan(&repeats)
	if errors.Is(err, pgx.ErrNoRows) {
		return // أُنذر سابقاً — ولم يحِن التذكير
	}
	if err != nil {
		s.logger.Error("watchdog: تعذّر وسم الإنذار", "order", a.OrderID, "error", err)
		return
	}
	// **والطارئُ أُنذر لحظةَ وقوعه** (`handleDriverEmergency`) — فيُوسَم ولا
	// يُعاد إنذارُه من الراصد.
	if a.Reason == StuckEmergency {
		if err := tx.Commit(ctx); err != nil {
			s.logger.Error("watchdog: تعذّر تثبيتُ الوسم", "order", a.OrderID, "error", err)
		}
		return
	}
	in := notifications.Input{
		Kind:     notifications.KindOrder,
		Title:    alertTitle(a.Reason, repeats),
		Body:     a.MerchantName,
		Entity:   "order",
		EntityID: a.OrderID,
		// **والرابطُ يفتح الطلبَ نفسَه** (البند ٢١) — لا اللوحةَ كلَّها.
		Href: "/dashboard/orders?id=" + a.OrderID,
	}
	// ══════════════════════════════════════════════════════════════════
	// **ومن يُنذَر بقدرته لا باسم دوره** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٦)
	// ══════════════════════════════════════════════════════════════════
	//
	// كان `admin` و`ops` — **و`ops` فارغٌ على التجهيز**، وموظّفُ العمليّات
	// الحقيقيُّ بدور `operations` لا يصله شيء. **فكلُّ من يملك
	// `orders.intervene` يُنذَر بالعالق.**
	var n int
	if cn, ok := s.notify.(capNotifier); ok {
		n, err = cn.NotifyCapsTx(ctx, tx, StuckAlertCaps, in)
	} else {
		n, err = s.notify.NotifyOpsTx(ctx, tx, in)
	}
	if err != nil {
		s.logger.Error("watchdog: تعذّرت نيّةُ الإنذار", "order", a.OrderID, "error", err)
		return
	}
	if n == 0 {
		s.logger.Warn("watchdog: لا مكتبَ يُنذَر — لا وسم", "order", a.OrderID)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.logger.Error("watchdog: تعذّر تثبيتُ الإنذار", "order", a.OrderID, "error", err)
		return
	}
	if cn, ok := s.notify.(capNotifier); ok {
		cn.PublishToCaps(ctx, s.db, StuckAlertCaps, in)
		return
	}
	s.notify.PublishToUsers(ctx, s.db, notifications.OpsDesk, in)
}

// StuckAlertCaps **من يُنذَر بالطلب العالق** — من يملك التدخّل فيه (البند ٦).
var StuckAlertCaps = []string{"orders.intervene"}

// EmergencyAlertCaps **من يُنذَر بالطارئ** — من يتدخّل في الطلب ومن يملك الدعم
// (البند ٦)، **ومن مُنح الطوارئَ وحدَها** (البند ٧).
var EmergencyAlertCaps = []string{"orders.intervene", "support.manage", "emergencies.manage"}

// capNotifier **إشعارٌ بالقدرة** — تنفّذه خدمةُ الإشعارات، **والجواسيسُ في
// الفحوص القديمة لا تنفّذه فيسقط النداءُ على الأدوار كما كان.**
type capNotifier interface {
	NotifyCapsTx(ctx context.Context, q dbtx.Querier, caps []string, in notifications.Input) (int, error)
	PublishToCaps(ctx context.Context, q dbtx.Querier, caps []string, in notifications.Input)
}

// alertTitles نصوص التصعيد — مصدرٌ واحد بجانب بقية نصوص الإشعارات.
//
// **والمكتبُ يقبل أوّلاً في الوضعين** (قرارُ المالك ٢٠٢٦-٠٨-٢٩) — فكان «لم يقبله
// متجرُه» يتّهم من لم يُسأل بعد (البند ٢٣).
var alertTitles = map[string]string{
	StuckNoAccept: "طلبٌ ينتظر قبولَ المكتب",
	StuckNotSent:  "طلبٌ مقبولٌ لم يُرسَل للمتجر",
	StuckNoDriver: "طلبٌ بلا سائق",
	StuckTooLong:  "طلبٌ تأخّر عن موعده",
}

// reminderPrefix **يسبق عنوانَ التذكير** — فيُعرف أنّه الطلبُ نفسُه لم يتحرّك.
const reminderPrefix = "تذكير: "

// alertTitle **عنوانُ الإنذار** — والتذكيرُ بالسبب نفسِه يُقال تذكيراً.
func alertTitle(reason string, repeats int) string {
	if repeats > 0 {
		return reminderPrefix + alertTitles[reason]
	}
	return alertTitles[reason]
}

// AckAlert **«أنا عليه» على طلبٍ عالق** — يوقف تكرارَ التذكير بسببه الحاليّ
// (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»، البند ٤).
//
// **ولا يُغلق شيئاً ولا يُخفي الطلبَ من قائمة العالق** — يقول «رآه أحدٌ وهو
// عليه». **وسببٌ جديدٌ يمسحها** فيُنذَر المكتبُ من جديد. ويُردّ `false` لطلبٍ
// لم يُنذَر بعد أو أُغلق أو استُلم سلفاً.
func (s *Service) AckAlert(ctx context.Context, orderID, actorID string) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE orders SET alert_ack_at = now(), alert_ack_by = NULLIF($2, '')::uuid
		 WHERE id = $1 AND closed_at IS NULL AND alerted_at IS NOT NULL
		   AND alert_ack_at IS NULL`, orderID, actorID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() > 0 && s.pub != nil {
		s.pub.Publish("ops", map[string]any{"type": "order"})
	}
	return tag.RowsAffected() > 0, nil
}

// autoAcceptNote **يُكتب في سجلّ الطلب** — فيُعرف أنّ يداً لم تقبله.
const autoAcceptNote = "قُبل تلقائيّاً بعد انتهاء مهلة المكتب"

// sweepAutoAccept **يقبل ما نُسي في انتظار المكتب.**
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا وُجد**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-٢٩.)
//
// **صار الطلبُ يصل المنصّةَ أوّلاً في الوضعين** (`modes.go`) — وذاك يضع
// كلَّ طلبٍ على يقظة موظّف. **ومن طلب الساعةَ الثانية ليلاً والمكتبُ
// نائمٌ ينتظر ولا يُطبخ طلبُه**، ولا يعرف لماذا.
//
// # وفرقُه عن التنبيه
//
// **`accept_timeout_min` ينبّه ولا يقبل** — يضع الطلبَ في شاشة
// التنبيهات. **وهذا يقبل.**
//
// # والقبولُ بدور المكتب
//
// **فيمضي الطلبُ كما لو ضغطه موظّف**: يُخطَر المتجرُ ويدخل التحضيرَ في
// وضع المتاجر، أو يبقى بيد المكتب في وضع المنصّة. **ولا مسارَ ثانياً
// يُصان.**
//
// # وما لا يُلمَس
//
// **الشرطُ `status = 'pending'` وحدَه** — فما تحرّك بيدٍ لا يُقبل
// تلقائيّاً. **وخمسون في الدورة الواحدة** لئلّا تُغرق دفعةٌ الراصد.
func (s *Service) sweepAutoAccept(ctx context.Context) {
	mins := s.autoAcceptAfter(ctx)
	if mins <= 0 {
		return
	}
	rows, err := s.db.Query(ctx, `
		SELECT id FROM orders
		WHERE status = 'pending'
		  AND closed_at IS NULL
		  AND created_at < now() - make_interval(mins => $1::int)
		-- **الأحدثُ ممّا تجاوز المهلةَ أوّلاً** — طلبٌ قديمٌ يتعثّر قبولُه في كلّ
		-- جولةٍ لا يحجز الخمسين عن طلباتٍ وصلت الليلة.
		ORDER BY created_at DESC
		LIMIT 50`, mins)
	if err != nil {
		s.logger.Warn("القبولُ التلقائيّ: تعذّرت القراءة", "error", err)
		return
	}
	ids := make([]string, 0, 8)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()

	for _, id := range ids {
		if _, err := s.Transition(ctx, "", []string{"ops"}, id,
			StAccepted, autoAcceptNote); err != nil {
			s.logger.Warn("القبولُ التلقائيّ تعثّر", "order", id, "error", err)
			continue
		}
		s.logger.Info("قُبل تلقائيّاً بعد المهلة", "order", id, "minutes", mins)
	}
}

// SweepAutoAcceptOnce **جولةُ قبولٍ تلقائيٍّ واحدة** — مِعراضُ الفحص كأخيه
// `EscalateAlertsOnce`، ولا مسارَ شبكةٍ له.
func (s *Service) SweepAutoAcceptOnce(ctx context.Context) { s.sweepAutoAccept(ctx) }

// ══════════════════════════════════════════════════════════════════════
// **والقبولُ التلقائيُّ ليلاً — إن لم يكن في المكتب أحد** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٨)
// ══════════════════════════════════════════════════════════════════════
//
// «طلبٌ الساعةَ الثانيةَ ليلاً يبقى بانتظار القبول» — **والقبولُ التلقائيُّ
// مطفأٌ نهاراً لأنّ في المكتب من يقبل.** فإن لم يكن أحد قُبل بعد مهلته.
//
// # ومن «في المكتب»؟
//
// **لا ورديّةَ للموظّفين في المنصّة** — فالإشارةُ الصادقةُ الوحيدة: **آخرُ
// ظهورٍ لمن يملك `orders.intervene`** (`users.last_seen_at`، يُحدَّث مع كلّ
// نداءٍ موثَّقٍ مرّةً في الدقيقة، ولوحةُ الطلبات تنادي كلَّ دقيقة). **فمن لم
// يظهر منهم أحدٌ في `orders.staff_presence_min` فالمكتبُ فارغ.**
//
// # والمهلتان
//
//	orders.auto_accept_min             دائماً — وصفرُه «لا»
//	orders.unattended_auto_accept_min  حين يفرغ المكتبُ وحدَه — وصفرُه «لا»
//
// **والأقصرُ يحكم** حين يصدق الاثنان.
func (s *Service) autoAcceptAfter(ctx context.Context) int64 {
	mins := s.settingInt(ctx, "orders.auto_accept_min")
	away := s.settingInt(ctx, "orders.unattended_auto_accept_min")
	if away > 0 && (mins <= 0 || away < mins) && !s.StaffPresent(ctx) {
		return away
	}
	return mins
}

// StaffPresent **أفي المكتب أحدٌ يملك قبولَ الطلب الآن؟** — انظر `autoAcceptAfter`.
//
// **وتعذّرُ القراءة يُقرأ «حاضر»**: قبولٌ آليٌّ بسبب عطبٍ في القاعدة يفاجئ
// المكتبَ وهو جالس.
func (s *Service) StaffPresent(ctx context.Context) bool {
	win := s.settingInt(ctx, "orders.staff_presence_min")
	if win <= 0 {
		win = 5
	}
	var present bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users u
			JOIN user_roles ur ON ur.user_id = u.id
			JOIN role_capabilities rc ON rc.role_code = ur.role_code
			WHERE rc.capability_code = 'orders.intervene'
			  AND u.status = 'active'
			  AND u.last_seen_at > now() - make_interval(mins => $1::int))`, win).
		Scan(&present); err != nil {
		return true
	}
	return present
}
