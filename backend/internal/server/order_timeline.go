package server

import (
	"context"
	"time"
)

/*
**مسارُ الطلب بأوقاته.**

(البندُ الرابعَ عشر في قائمة المالك ٢٠٢٦-٠٨-١٢: «هذا المسار لازم الإدارة
 تشوفه بصفحة مراقبة الطلبات، وكلّ شيء واضح التوقيت والساعة والدقيقة …
 وبنفس الشيء الزبون ولكن لا يشوف كلّ التفاصيل».)

# والبياناتُ محفوظةٌ منذ اليوم الأوّل

**جدول `order_events` يكتب كلَّ انتقالٍ بوقته** — من أيّ حالٍ إلى أيّ حال
ومن فعله ومتى. **ولا أحدَ يعرضه**: لا الزبونُ يعرف متى استلم سائقُه
طلبَه، ولا الإدارةُ تعرف أين ضاع الوقت.

# والزبونُ يرى أقلَّ

**ستُّ خطواتٍ لا أربعَ عشرة**: قُبل · إلى المتجر · استلم · إليك · وصل ·
سُلّم. **وما بينها شأنُ العمليات** — `dispatching` و`at_pickup` أسماءُ
آلةٍ لا أخبارُ زبون.

**والإدارةُ ترى الكلّ** — بأسمائه ومن فعله.
*/

// TimelineStep خطوةٌ في مسار الطلب.
type TimelineStep struct {
	Status string    `json:"status"`
	At     time.Time `json:"at"`
	Actor  *string   `json:"actor,omitempty"`
	// ActorRole **صفةُ الفاعل في هذا الطلب** — customer · merchant · driver · staff · system
	// (ملفُّ الطلب، قرارُ المالك ٢٠٢٦-١٠-٣٠). **للإدارة وحدَها** — والزبونُ لا يراها.
	ActorRole string `json:"actor_role,omitempty"`
	Note      string `json:"note,omitempty"`
}

// actorRoleSQL **صفةُ فاعل الحدث في طلبه** — من الطلب أوّلاً ثمّ من أدواره:
// زبونُه · سائقُه (الحاليّ أو سائقُ الحدث) · صاحبُ متجرٍ فيه · موظّفٌ · ثمّ ما بقي من أدواره.
// **وبلا فاعلٍ فالنظام.**
const actorRoleSQL = `CASE
	WHEN e.actor_id IS NULL THEN 'system'
	WHEN e.actor_id = o.customer_id THEN 'customer'
	WHEN e.actor_id = o.driver_id OR e.actor_id = e.driver_id THEN 'driver'
	WHEN EXISTS (SELECT 1 FROM merchants am WHERE am.owner_user_id = e.actor_id
	             AND (am.id = o.merchant_id OR am.id = o.partner_merchant_id
	                  OR am.id IN (SELECT ai.merchant_id FROM order_items ai WHERE ai.order_id = o.id))) THEN 'merchant'
	WHEN EXISTS (SELECT 1 FROM user_roles ar WHERE ar.user_id = e.actor_id
	             AND ar.role_code NOT IN ('customer', 'driver', 'merchant', 'sales')) THEN 'staff'
	WHEN EXISTS (SELECT 1 FROM user_roles ar WHERE ar.user_id = e.actor_id AND ar.role_code = 'driver') THEN 'driver'
	WHEN EXISTS (SELECT 1 FROM user_roles ar WHERE ar.user_id = e.actor_id AND ar.role_code = 'merchant') THEN 'merchant'
	ELSE 'staff' END`

// withReadyStep **يُدخل «المتجر جهّز الطلب» في المسار بوقته** (ملفُّ الطلب ٢٠٢٦-١٠-٣٠).
//
// **وليس حدثاً في `order_events`** — ضغطةُ «جاهز» تكتب `orders.ready_at` وحدَه ولا تُبدّل الحال،
// فكان المسارُ لا يقول متى جهّز المتجر. **فيُقرأ من عموده ويوضع في موضعه من الزمن.**
func withReadyStep(steps []TimelineStep, readyAt *time.Time, store string) []TimelineStep {
	if readyAt == nil {
		return steps
	}
	step := TimelineStep{Status: "ready", At: *readyAt, ActorRole: "merchant"}
	if store != "" {
		step.Actor = &store
	}
	out := make([]TimelineStep, 0, len(steps)+1)
	put := false
	for _, st := range steps {
		if !put && st.At.After(*readyAt) {
			out = append(out, step)
			put = true
		}
		out = append(out, st)
	}
	if !put {
		out = append(out, step)
	}
	return out
}

// customerSteps ما يراه الزبون — **وما عداه شأنُ العمليات.**
var customerSteps = map[string]bool{
	"accepted":   true,
	"assigned":   true,
	"picked_up":  true,
	"on_the_way": true,
	"at_dropoff": true,
	"delivered":  true,
	"cancelled":  true,
	"failed":     true,
	"refunded":   true,
}

// timeline يقرأ مسارَ الطلب من `order_events`.
//
// **و`full` تفصل الجمهورين**: الإدارةُ ترى كلَّ انتقالٍ ومن فعله،
// **والزبونُ يرى ما يخصّه بلا اسمِ فاعل** — من نقل الطلب شأنُ المنصّة.
func (s *Server) timeline(ctx context.Context, orderID string, full bool) []TimelineStep {
	rows, err := s.pg.Query(ctx, `
		SELECT e.to_status, e.created_at, u.full_name, e.note, `+actorRoleSQL+`
		FROM order_events e
		JOIN orders o ON o.id = e.order_id
		LEFT JOIN users u ON u.id = e.actor_id
		WHERE e.order_id = $1
		ORDER BY e.id`, orderID)
	if err != nil {
		return []TimelineStep{}
	}
	defer rows.Close()

	out := []TimelineStep{}
	// **أكان ما قبله إسناداً؟** — منه يُعرف تبديلُ السائق.
	lastWasAssigned := false
	for rows.Next() {
		var step TimelineStep
		var actor *string
		var role string
		if err := rows.Scan(&step.Status, &step.At, &actor, &step.Note, &role); err != nil {
			return out
		}
		if !full {
			// ══════════════════════════════════════════════════════════
			// **و«حُوّل إلى سائقٍ آخر» ليست حالاً في المحرّك**
			// ══════════════════════════════════════════════════════════
			//
			// (البندُ الخامسَ عشر في قائمة المالك ٢٠٢٦-٠٨-١٢.)
			//
			// **هي عودةٌ إلى `dispatching` بعد `assigned`** — والزبونُ
			// يقرأ «قُبل طلبك» ثمّ لا شيء، **فيبقى ينتظر من لن يأتي.**
			//
			// **فتُقرأ من المسار لا من عمودٍ جديد**: رجوعٌ إلى الطابور
			// بعد إسنادٍ يُسمّى باسمه.
			if step.Status == "dispatching" && lastWasAssigned {
				step.Status = "driver_changed"
				step.Actor = nil
				step.Note = ""
				out = append(out, step)
				lastWasAssigned = false
				continue
			}
			lastWasAssigned = step.Status == "assigned"

			if !customerSteps[step.Status] {
				continue
			}
			// **ولا اسمَ فاعلٍ للزبون ولا ملاحظةَ عملياتٍ** — يعرف ما
			// جرى لطلبه، **لا من فعله ولا لماذا كُتب في السجلّ.**
			step.Actor = nil
			step.Note = ""
		} else {
			step.Actor = actor
			step.ActorRole = role
		}
		out = append(out, step)
	}
	return out
}
