package comms

/*
**أحاديثُ شخصٍ كلُّها — في ملفّه لا في بحثٍ عن طلباته.**

(قرارُ المالك ٢٠٢٦-٠٨-١٥: «نسينا سجلَّ الدردشات الخاصّ به — وأيضاً السائق،
 لأنّها مهمّةٌ في حال حدوث أيّ مشكلةٍ أو نزاع».)

# ولماذا لا يكفي حديثُ الطلب

**الحديثُ كان يُقرأ من الطلب وحدَه** — ومن يحكم في نزاعٍ **لا يعرف رقمَ
الطلب بعد**: يعرف اسمَ الإنسان. **فيمشي طلباته واحداً واحداً يفتح كلَّ
حديثٍ يبحث عن سطر.**

**والسؤالُ عند الخلاف عن شخصٍ لا عن طلب**: «هل أساء هذا السائقُ من قبل؟» —
وجوابُه في عشرين حديثاً متفرّقاً، **لا يُجمع إلّا هنا.**

# والموسومُ أوّلاً — لا الأحدثُ أوّلاً

**السطرُ الموسومُ هو ما يُبحث عنه** (`flagged`): «قال لي كذا» كانت كلمةً
ضدّ كلمة. **وترتيبٌ بالزمن يدفنه تحت «أنا تحت البناية»** — فيُعدّ في رأس
كلّ حديثٍ ويُقدَّم ما فيه موسوم.

# ولا وسمَ قراءةٍ يُكتب

**ما تقرؤه الإدارةُ لا يصير «قُرئ» عند صاحبه** — وعلامةٌ زرقاءُ يضعها طرفٌ
ثالثٌ كذبٌ يُحتجّ به.
*/

import (
	"context"
	"time"
)

// UserThread **حديثُ طلبٍ واحدٍ في ملفّ صاحبه — رأسُه لا سطورُه.**
//
// **والسطورُ تُطلب بالطلب** (`Audit`) — **وجلبُها كلَّها مع كلّ فتحةِ ملفٍّ
// يحمل مئاتِ الأسطر لسؤالٍ لم يُسأل بعد.**
type UserThread struct {
	OrderID     string `json:"order_id"`
	OrderNumber int64  `json:"order_number"`
	// Peer **الطرفُ الآخر** — من كان يحدّثه، **وهو ما يُبحث عنه في نزاع.**
	Peer     string `json:"peer"`
	PeerRole string `json:"peer_role"`
	// Count **كم سطراً**، وFlagged **كم موسوماً منها.**
	Count   int `json:"count"`
	Flagged int `json:"flagged"`
	// Last **آخرُ ما قيل** — يُقرأ بلمحةٍ فيُعرف أيُفتح أم لا.
	Last     string    `json:"last"`
	LastRole string    `json:"last_role"`
	LastAt   time.Time `json:"last_at"`
}

// UserThreads **أحاديثُ هذا الحساب — زبوناً كان أو سائقاً.**
//
// **والمعرّفُ يُقارن بطرفَي الطلب لا بمرسِل الرسائل**: من فُتح له حديثٌ
// فلم يكتب فيه حرفاً **له فيه سطورُ الآخر** — وهي ما يُحتجّ به عليه أو له.
//
// **والمتجرُ لا طرفَ له هنا**: القناةُ بين الزبون والسائق وحدَهما.
func (s *Service) UserThreads(ctx context.Context, userID string, limit, offset int) ([]UserThread, int, error) {
	// **والعدُّ بشرط القائمة نفسِه** — نصّان يفترقان يوماً **فيقول العنوانُ
	// عشرين وتعرض القائمةُ خمسةَ عشر.**
	const scope = `
		FROM orders o
		LEFT JOIN users cu ON cu.id = o.customer_id
		LEFT JOIN users dr ON dr.id = o.driver_id
		WHERE (o.customer_id = $1 OR o.driver_id = $1)
		  AND EXISTS (SELECT 1 FROM order_messages m WHERE m.order_id = o.id)`

	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*)`+scope, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// **والطرفُ الآخرُ يُشتقّ من موقعه في الطلب** — فمن كان زبوناً رأى
	// سائقَه، ومن كان سائقاً رأى زبونَه. **وسائقٌ سُحب منه الطلبُ يبقى
	// اسمُه على سطوره**، لكنّ رأسَ الحديث يقول حاملَه اليوم.
	rows, err := s.db.Query(ctx, `
		SELECT o.id::text, o.number,
		       CASE WHEN o.customer_id = $1
		            THEN COALESCE(dr.full_name, dr.phone::text, '')
		            ELSE COALESCE(cu.full_name, cu.phone::text, '') END,
		       CASE WHEN o.customer_id = $1 THEN 'driver' ELSE 'customer' END,
		       (SELECT count(*) FROM order_messages m WHERE m.order_id = o.id),
		       (SELECT count(*) FROM order_messages m WHERE m.order_id = o.id AND m.flagged),
		       COALESCE((SELECT m.body FROM order_messages m WHERE m.order_id = o.id
		                 ORDER BY m.created_at DESC LIMIT 1), ''),
		       COALESCE((SELECT m.sender_role FROM order_messages m WHERE m.order_id = o.id
		                 ORDER BY m.created_at DESC LIMIT 1), ''),
		       (SELECT m.created_at FROM order_messages m WHERE m.order_id = o.id
		        ORDER BY m.created_at DESC LIMIT 1)`+scope+`
		-- **والموسومُ يتقدّم** — هو ما يُفتح الملفُّ لأجله.
		ORDER BY (SELECT count(*) FROM order_messages m
		          WHERE m.order_id = o.id AND m.flagged) > 0 DESC,
		         o.created_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []UserThread{}
	for rows.Next() {
		var t UserThread
		if err := rows.Scan(&t.OrderID, &t.OrderNumber, &t.Peer, &t.PeerRole,
			&t.Count, &t.Flagged, &t.Last, &t.LastRole, &t.LastAt); err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}
