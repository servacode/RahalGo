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
		INSERT INTO notifications (user_id, kind, title, body, entity, entity_id, href)
		SELECT u, $2, $3, $4, $5, $6, $7 FROM unnest($1::uuid[]) AS u`,
		ids, in.Kind, in.Title, in.Body, in.Entity, in.EntityID, in.Href)
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
