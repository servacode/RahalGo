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
	Note   string    `json:"note,omitempty"`
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
		SELECT e.to_status, e.created_at, u.full_name, e.note
		FROM order_events e
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
		if err := rows.Scan(&step.Status, &step.At, &actor, &step.Note); err != nil {
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
		}
		out = append(out, step)
	}
	return out
}
