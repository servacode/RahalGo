package orders

// ══════════════════════════════════════════════════════════════════════
// **شروطُ لوحة الطلبات — تُكتب مرّةً هنا وتُقرأ في كلّ مكان**
// ══════════════════════════════════════════════════════════════════════
//
// (قراراتُ المالك ٢٠٢٦-١٠-٠٤ — قسمُ «الطلبات»، البنود ١ و٣ و٥.)
//
// # المسألة
//
// **ثلاثةُ مواضعَ كانت تعدّ «العالق» بثلاثة شروط**: بطاقةُ الرئيسيّة تقيس من
// `updated_at` بلا شرط السائق، وصندوقُ التنبيهات من `dispatched_at` بشرطه،
// ورابطُ البطاقة يفتح الطابورَ كلَّه. **فتقول البطاقةُ اثنين وتعرض القائمةُ
// خمسة** — ولا أحدَ يعرف أيُّهما الصواب.
//
// # والحلّ
//
// **كلُّ شرطٍ دالّةٌ تُرجع نصَّ SQL على الاسم المستعار `o`** (جدول `orders`):
//
//	StuckReasonSQL   سببُ العلوق أو NULL — الصندوقُ الأحمر والإنذارُ والفلتر «عالق»
//	NoDriverSQL      «بلا سائق» بعد مهلته — بطاقةُ الرئيسيّة تقرؤها بعينها
//	BoardFilterSQL   عدّاداتُ المراحل وفلاترُها — العدُّ والقائمةُ بنصٍّ واحد
//	PriorityOrderSQL ترتيبُ اللوحة — المُنذَرُ أوّلاً ثمّ المتأخّرُ ثمّ الأقدم
//
// **والمهلُ أعدادٌ صحيحةٌ تُكتب في النصّ** (`StuckLimits`) — من الإعدادات لا
// من يد كاتب، **ولا بابَ حقنٍ في عددٍ صحيح.**

import (
	"context"
	"strconv"
)

// StuckLimits **مهلُ العلوق كما ضبطها المالك** — بالدقائق.
type StuckLimits struct {
	// AcceptMin **مهلةُ قبول المكتب** — بعدها «بانتظار قبول المكتب» و«مقبولٌ لم يُرسَل».
	AcceptMin int
	// DriverMin **مهلةُ إيجاد سائق** — بعدها «بلا سائق».
	DriverMin int
	// DeliveryMin **عمرُ الطلب كلِّه** — بعده «متأخّر».
	DeliveryMin int
}

// StuckLimitsOf **المهلُ من الإعدادات** — والافتراضُ من الفهرس بلا مخزن.
//
// **ويقرؤها كلُّ من يعدّ**: الراصدُ واللوحةُ والرئيسيّة.
func (s *Service) StuckLimitsOf(ctx context.Context) StuckLimits {
	return StuckLimits{
		AcceptMin:   int(s.settingInt(ctx, "orders.accept_timeout_min")),
		DriverMin:   int(s.settingInt(ctx, "orders.driver_timeout_min")),
		DeliveryMin: int(s.settingInt(ctx, "orders.delivery_timeout_min")),
	}
}

func mins(n int) string {
	if n < 0 {
		n = 0
	}
	return `make_interval(mins => ` + strconv.Itoa(n) + `)`
}

// EmergencyOpenSQL **طارئٌ مفتوحٌ على الطلب** — لم يُغلقه إنسان.
const EmergencyOpenSQL = `EXISTS (SELECT 1 FROM driver_emergencies de
	WHERE de.order_id = o.id AND de.status = 'open')`

// NoAcceptSQL **بانتظار قبول المكتب** بعد مهلته — والمكتبُ يقبل أوّلاً في الوضعين
// (قرارُ المالك ٢٠٢٦-٠٨-٢٩)، **فالتأخيرُ عند المكتب لا عند المتجر.**
func NoAcceptSQL(l StuckLimits) string {
	return `(o.status = 'pending' AND o.created_at < now() - ` + mins(l.AcceptMin) + `)`
}

// NotSentSQL **مقبولٌ ولم يُرسَل للمتجر** بعد مهلة القبول نفسِها.
//
// (قِيس على التجهيز ٢٠٢٦-١٠-٠٤: ‎#1400‎ «مقبول» منذ ٣٢٣ دقيقة — **ولم يظهر إلّا
// بعد ساعةٍ تحت «متأخّر جدّاً».**)
func NotSentSQL(l StuckLimits) string {
	return `(o.status = 'accepted' AND o.sent_to_merchant_at IS NULL
		AND COALESCE(o.accepted_at, o.created_at) < now() - ` + mins(l.AcceptMin) + `)`
}

// NoDriverSQL **بلا سائقٍ بعد مهلة إيجاده** — من نزوله إلى الطابور لا من إنشائه.
//
// **وهي الدالّةُ التي تقرؤها بطاقةُ «طلبات بلا سائق» في الرئيسيّة** — فيطابق
// رقمُها القائمةَ التي تفتحها (`?filter=no_driver`).
func NoDriverSQL(l StuckLimits) string {
	return `(o.status IN ('preparing','dispatching') AND o.driver_id IS NULL
		AND COALESCE(o.dispatched_at, o.updated_at) < now() - ` + mins(l.DriverMin) + `)`
}

// TooLongSQL **تجاوز عمرُ الطلب مهلتَه** — الشبكةُ الأخيرة.
func TooLongSQL(l StuckLimits) string {
	return `(o.created_at < now() - ` + mins(l.DeliveryMin) + `)`
}

// أسبابُ العلوق — **بترتيب الأولويّة**: الأوّلُ الذي يصدق هو السبب.
const (
	StuckEmergency = "emergency"
	StuckNoAccept  = "no_accept"
	StuckNotSent   = "not_sent"
	StuckNoDriver  = "no_driver"
	StuckTooLong   = "too_long"
)

// StuckReasonSQL **سببُ علوق الطلب أو NULL** — المصدرُ الواحدُ للإنذار والصندوق
// والفلتر «عالق».
//
// **والطارئُ أوّلاً**: سائقٌ أصابه شيءٌ أعجلُ من طلبٍ تأخّر.
func StuckReasonSQL(l StuckLimits) string {
	return `(CASE
		WHEN o.closed_at IS NOT NULL THEN NULL
		WHEN ` + EmergencyOpenSQL + ` THEN '` + StuckEmergency + `'
		WHEN ` + NoAcceptSQL(l) + ` THEN '` + StuckNoAccept + `'
		WHEN ` + NotSentSQL(l) + ` THEN '` + StuckNotSent + `'
		WHEN ` + NoDriverSQL(l) + ` THEN '` + StuckNoDriver + `'
		WHEN ` + TooLongSQL(l) + ` THEN '` + StuckTooLong + `'
	END)`
}

// StuckSinceSQL **منذ متى يصدق السبب** — فتعدّ الشاشةُ الدقائقَ حيّةً منه.
//
// (البند ١٩: كانت الدقائقُ من إنشاء الطلب حتّى لسبب «بلا سائق».)
func StuckSinceSQL(l StuckLimits) string {
	return `(CASE ` + StuckReasonSQL(l) + `
		WHEN '` + StuckEmergency + `' THEN COALESCE((SELECT min(de.created_at) FROM driver_emergencies de
			WHERE de.order_id = o.id AND de.status = 'open'), o.created_at)
		WHEN '` + StuckNotSent + `' THEN COALESCE(o.accepted_at, o.created_at)
		WHEN '` + StuckNoDriver + `' THEN COALESCE(o.dispatched_at, o.updated_at)
		ELSE o.created_at
	END)`
}

// فلاترُ اللوحة — **وكلُّ عدّادٍ فلترٌ بالاسم نفسِه** (البند ٣).
const (
	BoardAwaitingAccept = "awaiting_accept"
	BoardAwaitingDriver = "awaiting_driver"
	BoardOnTheWay       = "on_the_way"
	BoardAtDoor         = "at_door"
	BoardReports        = "reports"
	BoardStuck          = "stuck"
	BoardNoDriver       = "no_driver"
	BoardEmergency      = "emergency"
	// BoardNew **طلبٌ جديدٌ لم يضغط عليه أحدٌ «استلمتها»** — وهو ما يرنّ (البند ٥).
	BoardNew = "new"
)

// BoardFilters **أسماءُ الفلاتر بترتيب العرض** — العدّاداتُ تُرسَم بها.
var BoardFilters = []string{
	BoardAwaitingAccept, BoardAwaitingDriver, BoardOnTheWay, BoardAtDoor,
	BoardReports, BoardStuck, BoardNoDriver, BoardEmergency, BoardNew,
}

// BoardFilterSQL **شرطُ فلترٍ من فلاتر اللوحة** — و`ok=false` لاسمٍ لا يُعرف.
//
// **والعدُّ يقرأ هذا النصَّ نفسَه** (`BoardCounts`) — فلا يقول العدّادُ
// رقماً وتعرض القائمةُ غيرَه.
func BoardFilterSQL(name string, l StuckLimits) (string, bool) {
	switch name {
	case BoardAwaitingAccept:
		return `(o.closed_at IS NULL AND o.status = 'pending')`, true
	case BoardAwaitingDriver:
		return `(o.closed_at IS NULL AND o.status IN ('preparing','dispatching') AND o.driver_id IS NULL)`, true
	case BoardOnTheWay:
		return `(o.closed_at IS NULL AND o.status IN ('assigned','at_pickup','picked_up','on_the_way'))`, true
	case BoardAtDoor:
		return `(o.closed_at IS NULL AND o.status = 'at_dropoff')`, true
	case BoardReports:
		return `(` + awaitingOfficeSQL() + `)`, true
	case BoardStuck:
		return `(` + StuckReasonSQL(l) + ` IS NOT NULL)`, true
	case BoardNoDriver:
		return `(o.closed_at IS NULL AND ` + NoDriverSQL(l) + `)`, true
	case BoardEmergency:
		return `(o.closed_at IS NULL AND ` + EmergencyOpenSQL + `)`, true
	case BoardNew:
		return `(o.closed_at IS NULL AND o.status = 'pending' AND o.office_seen_at IS NULL)`, true
	}
	return "", false
}

// PriorityOrderSQL **ترتيبُ اللوحة** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ١):
//
//	٠  طارئٌ · بلاغٌ ينتظر قرارك · بانتظار القبول · مقبولٌ لم يُرسَل · بلا سائق
//	١  تجاوز عمرَه («متأخّر»)
//	٢  الباقي
//
// **ثمّ الأقدمُ أوّلاً** — فلا ينزل أخطرُ طلبٍ إلى الصفحة الثانية.
func PriorityOrderSQL(l StuckLimits) string {
	r := StuckReasonSQL(l)
	return `(CASE
		WHEN ` + r + ` IN ('` + StuckEmergency + `','` + StuckNoAccept + `','` +
		StuckNotSent + `','` + StuckNoDriver + `') OR (` + awaitingOfficeSQL() + `) THEN 0
		WHEN ` + r + ` = '` + StuckTooLong + `' THEN 1
		ELSE 2
	END), o.created_at ASC, o.number ASC`
}

// BoardCounts **عدّاداتُ اللوحة** — كلٌّ بشرط فلتره نفسِه.
//
// **ولا يتبع بحثاً ولا فلتراً**: العدّادُ يقول ما في اللوحة كلِّها، **وصفرٌ
// يكذب أسوأُ من رقمٍ غائب.**
func (s *Service) BoardCounts(ctx context.Context) (map[string]int, error) {
	l := s.StuckLimitsOf(ctx)
	sel := ""
	for i, name := range BoardFilters {
		cond, _ := BoardFilterSQL(name, l)
		if i > 0 {
			sel += ", "
		}
		sel += `count(*) FILTER (WHERE ` + cond + `)`
	}
	vals := make([]int, len(BoardFilters))
	dst := make([]any, len(BoardFilters))
	for i := range vals {
		dst[i] = &vals[i]
	}
	if err := s.db.QueryRow(ctx,
		`SELECT `+sel+` FROM orders o WHERE o.closed_at IS NULL`).Scan(dst...); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(BoardFilters))
	for i, name := range BoardFilters {
		out[name] = vals[i]
	}
	return out, nil
}

// BoardInfo **ما يحتاجه المكتبُ عن الطلب ولا يحتاجه طرفُه** — يُملأ في ردّ
// الإدارة وحدَه، **فلا يصل سعرُ الشراء ولا عددُ من رفضوا تطبيقَ الزبون.**
type BoardInfo struct {
	// AlertReason **سببُ العلوق** — وفارغٌ حين لا علوق.
	AlertReason string `json:"alert_reason"`
	// AlertSince **منذ متى يصدق السبب** — والشاشةُ تعدّ منه حيّاً.
	AlertSince *string `json:"alert_since"`
	// EmergencyOpen **طارئٌ مفتوحٌ لدى السائق.**
	EmergencyOpen bool `json:"emergency_open"`
	// OfferExpiresAt **متى ينقضي العرضُ الحيّ** — والفارغُ لا عرض.
	OfferExpiresAt *string `json:"offer_expires_at"`
	// OfferPassed **كم سائقاً مرّ عليهم العرضُ فلم يقبلوا.**
	OfferPassed int `json:"offer_passed"`
	// GoodsCost **سعرُ شراء البضاعة** — سقفُ تعويض المتجر (البند ١٣).
	GoodsCost int64 `json:"goods_cost"`
	// StoreCompensated **ما دُفع للمتجر تعويضاً عن بضاعته** — وصفرُه لا تعويض.
	StoreCompensated int64 `json:"store_compensated"`
	// SeenAt **متى ضغط موظّفٌ «استلمتها»** — والفارغُ مع «بانتظار القبول» يرنّ.
	SeenAt *string `json:"seen_at"`
}

// BoardInfoOf **معلوماتُ المكتب لصفحةٍ من الطلبات** — في رحلةٍ واحدة.
func (s *Service) BoardInfoOf(ctx context.Context, ids []string) (map[string]*BoardInfo, error) {
	out := map[string]*BoardInfo{}
	if len(ids) == 0 {
		return out, nil
	}
	l := s.StuckLimitsOf(ctx)
	rows, err := s.db.Query(ctx, `
		SELECT o.id::text,
		       COALESCE(`+StuckReasonSQL(l)+`, ''),
		       to_char(`+StuckSinceSQL(l)+` AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       `+EmergencyOpenSQL+`,
		       CASE WHEN o.offer_expires_at > now()
		            THEN to_char(o.offer_expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       COALESCE(cardinality(o.offer_passed), 0),
		       COALESCE((SELECT sum(oi.merchant_price * oi.qty) FROM order_items oi
		                 WHERE oi.order_id = o.id), 0),
		       COALESCE((SELECT sum(wt.amount) FROM wallet_transactions wt
		                 JOIN merchants mm ON mm.owner_user_id = wt.user_id
		                 WHERE wt.ref = o.id::text AND wt.kind = 'compensation'
		                   AND mm.id = o.merchant_id), 0),
		       to_char(o.office_seen_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM orders o WHERE o.id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		b := &BoardInfo{}
		var since *string
		if err := rows.Scan(&id, &b.AlertReason, &since, &b.EmergencyOpen,
			&b.OfferExpiresAt, &b.OfferPassed, &b.GoodsCost, &b.StoreCompensated, &b.SeenAt); err != nil {
			return nil, err
		}
		if b.AlertReason != "" {
			b.AlertSince = since
		}
		out[id] = b
	}
	return out, rows.Err()
}

// MarkSeen **«استلمتها» على طلبٍ جديد** — يُسكت رنينَه عند المكتب كلِّه (البند ٥).
//
// **ومرّةً واحدة**: الضغطةُ الثانيةُ لا تُبدّل من استلم ولا متى. **ويُردّ `false`
// لطلبٍ استُلم سلفاً أو لم يعد جديداً** — والشاشةُ تُحدَّث على أيّ حال.
func (s *Service) MarkSeen(ctx context.Context, orderID, actorID string) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE orders SET office_seen_at = now(), office_seen_by = NULLIF($2, '')::uuid
		 WHERE id = $1 AND office_seen_at IS NULL`, orderID, actorID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() > 0 && s.pub != nil {
		s.pub.Publish("ops", map[string]any{"type": "order"})
	}
	return tag.RowsAffected() > 0, nil
}
