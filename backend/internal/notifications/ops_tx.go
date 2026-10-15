package notifications

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// ══════════════════════════════════════════════════════════════════════
// **نيّةُ الإنذار دائمةٌ مع وسمِها** — `PF-07` · `R22`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **الراصدُ يسم الطلبَ `alerted_at` ثمّ يُشعِر** — **والوسمُ يمنع
// إعادةَ المحاولة.** فإن سقط الإشعارُ أو مات المنفّذُ بينهما **صمت
// الإنذارُ إلى الأبد**، **ولا يراه زبونٌ ولا إدارة.**
//
// **والوسمُ يقول «أُنذر» وهو لا يعني إلّا «قرّرنا أن نُنذر».**
//
// # وما صار
//
//	DETECTED → DURABLE ALERT INTENT (مع الوسم في معاملةٍ واحدة)
//	         → DELIVERY ATTEMPT (بعد التثبيت)
//
// **ولا صندوقةَ جديدةٌ تُبنى**: **صفُّ الإشعار في `notifications` هو
// النيّةُ الدائمة** — قائمٌ ومقروءٌ في التطبيق. **وجدولُ عملٍ ثانٍ
// حقيقةٌ ثانيةٌ تشيخ.**
//
// # وما لا يضمنه هذا
//
// **وصولُ الدفعة إلى هاتف** — **تلك شبكةٌ وطرفٌ ثالث، وعقدُها
// `PF-09`.** **والمضمونُ أنّ النيّةَ لا تضيع، والتسليمُ محاولةٌ على
// الأقلّ مرّة.**

// NotifyRolesTx يُنشئ إشعاراتٍ دائمةً لأدوارٍ **في معاملةٍ مُمرَّرة**.
//
// **ويُرجع عددَ من أُشعِروا** — **فالمنادي يعرف أوقعت النيّةُ أم لا،
// ولا يسم ما لم يقع.**
//
// **ولا بثَّ فيها ولا دفعة**: **نداءٌ خارجيٌّ داخل معاملةٍ يُطيل قفلَها
// بقدر بُعد الطرف الآخر** — **وهما بعد التثبيت.**
func (s *Service) NotifyRolesTx(ctx context.Context, q dbtx.Querier,
	roles []string, in Input) (int, error) {
	if s == nil || in.Title == "" {
		return 0, nil
	}
	rows, err := q.Query(ctx, `
		SELECT DISTINCT u.id FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_code = ANY($1) AND u.status = 'active'`, roles)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		// **ولا مكتبَ يُنذَر** — **ووسمٌ بلا مُنذَرٍ كذب.**
		return 0, nil
	}
	if in.Transient {
		return len(ids), nil
	}
	tag, err := q.Exec(ctx, `
		INSERT INTO notifications (user_id, kind, title, body, entity,
		            entity_id, href, push_pending, push_apps)
		SELECT u, $2, $3, $4, $5, $6, $7, $8, $9
		  FROM unnest($1::uuid[]) AS u`,
		ids, in.Kind, in.Title, in.Body, in.Entity, in.EntityID, in.Href,
		!in.Silent, apps(in.Apps))
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// NotifyOpsTx كـ`NotifyOps` **في معاملةٍ مُمرَّرة**.
func (s *Service) NotifyOpsTx(ctx context.Context, q dbtx.Querier, in Input) (int, error) {
	return s.NotifyRolesTx(ctx, q, OpsDesk, in)
}

// PublishToUsers يبثّ إلى من أُشعِروا — **بعد التثبيت.**
//
// **وبثٌّ خرج ثمّ ارتدّت المعاملةُ كذبٌ لا يُسحَب.**
func (s *Service) PublishToUsers(ctx context.Context, q dbtx.Querier, roles []string, in Input) {
	if s == nil || s.hub == nil {
		return
	}
	rows, err := q.Query(ctx, `
		SELECT DISTINCT u.id FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_code = ANY($1) AND u.status = 'active'`, roles)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			s.hub.Publish("user:"+id, map[string]any{"type": "notification_refresh"})
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **الإنذارُ بالقدرة لا باسم الدور** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٦)
// ══════════════════════════════════════════════════════════════════════
//
// **كان العالقُ والطارئُ يذهبان إلى `admin` و`ops`** — **و`ops` دورٌ إرثيٌّ
// فارغٌ على التجهيز**، وموظّفُ العمليّات الحقيقيُّ (`operations`) والدعمُ
// (`customer_support`) لا يصلهما شيء. **فمن يملك القدرةَ يُنذَر أيَّ دورٍ حمل.**

// capHoldersSQL **من يملك إحدى القدرات** — حسابٌ نشطٌ وأدوارُه × قدراتُ أدواره.
const capHoldersSQL = `
	SELECT DISTINCT u.id FROM users u
	JOIN user_roles ur ON ur.user_id = u.id
	JOIN role_capabilities rc ON rc.role_code = ur.role_code
	WHERE rc.capability_code = ANY($1) AND u.status = 'active'`

// CapHolders **معرّفاتُ من يملك إحدى القدرات** — للفحص ولمن يحتاج القائمة.
func (s *Service) CapHolders(ctx context.Context, q dbtx.Querier, caps []string) ([]string, error) {
	rows, err := q.Query(ctx, capHoldersSQL, caps)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// NotifyCaps **يبلّغ من يملك إحدى القدرات** — مرّةً لكلّ شخص.
func (s *Service) NotifyCaps(ctx context.Context, caps []string, in Input) {
	if s == nil {
		return
	}
	s.notifyQuery(ctx, in, capHoldersSQL, caps)
}

// NotifyCapsRoles **يبلّغ من يملك إحدى القدرات أو يحمل أحدَ الأدوار** — مرّةً لكلّ شخص.
//
// (قرارُ المالك ٢٠٢٦-١٠-١٥: «للمالك وللماليّة».) **ونداءان منفصلان يُشعران المالكَ
// مرّتين** — فهو يحمل الدورَ ويملك القدرةَ معاً.
func (s *Service) NotifyCapsRoles(ctx context.Context, caps, roles []string, in Input) {
	if s == nil {
		return
	}
	s.notifyQuery(ctx, in, `
		SELECT id FROM (`+capHoldersSQL+`) caps
		UNION
		SELECT u.id FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_code = ANY($2) AND u.status = 'active'`, caps, roles)
}

// NotifyCapsTx كـ`NotifyCaps` **في معاملةٍ مُمرَّرة** — ويُرجع عددَ من أُشعِروا،
// **فلا يُوسَم إنذارٌ لم يقع.**
func (s *Service) NotifyCapsTx(ctx context.Context, q dbtx.Querier, caps []string, in Input) (int, error) {
	if s == nil || in.Title == "" {
		return 0, nil
	}
	ids, err := s.CapHolders(ctx, q, caps)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	if in.Transient {
		return len(ids), nil
	}
	tag, err := q.Exec(ctx, `
		INSERT INTO notifications (user_id, kind, title, body, entity,
		            entity_id, href, push_pending, push_apps)
		SELECT u, $2, $3, $4, $5, $6, $7, $8, $9
		  FROM unnest($1::uuid[]) AS u`,
		ids, in.Kind, in.Title, in.Body, in.Entity, in.EntityID, in.Href,
		!in.Silent, apps(in.Apps))
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// PublishToCaps يبثّ إلى من يملك القدرة — **بعد التثبيت.**
func (s *Service) PublishToCaps(ctx context.Context, q dbtx.Querier, caps []string, in Input) {
	if s == nil || s.hub == nil {
		return
	}
	ids, err := s.CapHolders(ctx, q, caps)
	if err != nil {
		return
	}
	for _, id := range ids {
		s.hub.Publish("user:"+id, map[string]any{"type": "notification_refresh"})
	}
}

// NotifyUserTx **إشعارٌ دائمٌ لشخصٍ واحدٍ في معاملةٍ مُمرَّرة** — بلا خدمة.
//
// (قرارُ المالك ٢٠٢٦-١٠-١٥، الديون: «صار عليك دين… انقطع من دينك…».) **الدينُ
// ينشأ ويُقتطع في معاملة المال نفسِها**، فالخبرُ يُكتب معه: **إن سقطت المعاملةُ
// سقط الخبر، ولا يُخبَر أحدٌ بدينٍ لم يقع.** والدفعُ تلتقطه الجولةُ الدوريّة.
func NotifyUserTx(ctx context.Context, q dbtx.Querier, in Input) error {
	if in.UserID == "" || in.Title == "" || in.Transient {
		return nil
	}
	_, err := q.Exec(ctx, `
		INSERT INTO notifications (user_id, kind, title, body, entity,
		            entity_id, href, push_pending, push_apps)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		in.UserID, in.Kind, in.Title, in.Body, in.Entity, in.EntityID, in.Href,
		!in.Silent, apps(in.Apps))
	return err
}
