package orders

// ══════════════════════════════════════════════════════════════════════
// **من يرى الطلبَ في نمط «للجميع» — شرطٌ واحدٌ يقرؤه بابان** (٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// **بابُ الطابور يسأل: «ما الذي يراه هذا السائق؟»** — **ودفعُ العرض يسأل:
// «من يرى هذا الطلب؟»** — وهما سؤالٌ واحدٌ من جهتين.
//
// **وكان الدفعُ يرنّ عند كلّ سائقٍ على ورديّة في المدينة** — قريباً كان أو
// بعيداً، يستطيع أخذَه أو لا يستطيع. **فيفتح من في آخر المدينة طلباً لا يراه
// في طابوره أصلاً**، ومن رنّ هاتفُه عشرَ مرّاتٍ بلا طلبٍ يطفئ الإشعارات.
//
// **فالشرطُ هنا نصٌّ واحد** — `drv` تعبيرٌ عن السائق (`$1::uuid` في الطابور،
// `u.id` في الدفع) و`o` الطلبُ و`m` متجرُه. **وشرطان منسوخان يفترقان عند أوّل
// تعديل**: يرنّ لمن لا يرى، أو يُرى لمن لا يرنّ.

import (
	"context"
	"strconv"
)

// QueueBaseSQL **ما يحجب الطلبَ عن سائقٍ بعينه في كلّ حال** — عرضٌ لغيره،
// أو رفضه هو من قبل.
func QueueBaseSQL(drv string) string {
	return `(o.offered_driver_id IS NULL OR o.offered_driver_id = ` + drv + `)
		  -- **وما رفضه لا يعود إليه** (الرفضُ في «للجميع» إخفاءٌ لا نقل).
		  AND NOT (` + drv + ` = ANY(o.offer_passed))
		  -- **وما تركه لا يعود إليه أبداً** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢).
		  AND NOT (` + drv + ` = ANY(o.excluded_drivers))`
}

// QueueDriverFreshLocSQL **موضعُ السائق إن كان حديثاً** — وإلّا `NULL`.
func QueueDriverFreshLocSQL(drv, freshArg string) string {
	return `(SELECT du.last_location FROM users du
	              WHERE du.id = ` + drv + ` AND du.last_location_at > now() - make_interval(secs => ` + freshArg + `))`
}

// QueueVisibleSQL **شرطُ رؤية الطلب بالقرب** — الأساسُ وأهليّةُ السائق
// والحلقةُ التي تتوسّع بانتظار الطلب.
//
// `p` رقمُ أوّل وسيطٍ من سبعة بالترتيب: سقفُ النقد · أقصى الطلبات · حداثةُ
// الموضع · أقصى نصف القطر · أوّلُه · خطوتُه · مهلةُ العرض (`DispatchProximity`).
func QueueVisibleSQL(drv string, p int) string {
	arg := func(i int) string { return "$" + strconv.Itoa(p+i) }
	cash, maxActive, fresh, maxM, initM, stepM, timeout :=
		arg(0), arg(1), arg(2), arg(3), arg(4), arg(5), arg(6)
	radius := `LEAST(` + maxM + `::float8, ` + initM + `::float8 + ` + stepM + `::float8 *
		floor(GREATEST(0, EXTRACT(EPOCH FROM (now() - COALESCE(o.dispatched_at, o.created_at)))) / GREATEST(` + timeout + `::float8, 1)))`
	freshLoc := QueueDriverFreshLocSQL(drv, fresh)
	return QueueBaseSQL(drv) + `
		  -- **السائقُ مؤهَّلٌ فعلاً** — دوامٌ وحالةٌ وسقفُ نقدٍ وعددُ طلبات.
		  AND EXISTS (SELECT 1 FROM users du WHERE du.id = ` + drv + ` AND du.on_shift AND du.status = 'active')
		  AND COALESCE((SELECT b.held FROM driver_cash_boxes b WHERE b.driver_id = ` + drv + `), 0)
		      -- **والمُسنَدُ الذي لم يُسلَّم يُحسب** — صيغةُ cashbox.Exposure.
		      + COALESCE((SELECT sum(oi.cash_due) FROM orders oi
		                  WHERE oi.driver_id = ` + drv + ` AND oi.closed_at IS NULL), 0)
		      + o.cash_due <= COALESCE((SELECT cu.cash_limit_override FROM users cu
		                                WHERE cu.id = ` + drv + `), ` + cash + `)
		  -- **والمتأخّرُ بتسليم نقده لا يرى نقديّاً إن أُشعل الإيقاف** (٢٠٢٦-١٠-٠٤).
		  AND (o.cash_due = 0 OR NOT driver_cash_overdue_stopped(` + drv + `))
		  AND (SELECT count(*) FROM orders oo WHERE oo.driver_id = ` + drv + ` AND oo.closed_at IS NULL) < ` + maxActive + `
		  -- **حديثُ الموقع — شرطٌ لا يسقط** (قرارُ المالك ٢٠٢٦-٠٩-٢٨).
		  AND ` + freshLoc + ` IS NOT NULL
		  -- **داخلَ الحلقة، أو طلبٌ بلغ أقصى التوسّع، أو بلا نقطةِ التقاطٍ تُقاس.**
		  AND (
		    ` + DispatchAnchorSQL + ` IS NULL
		    OR ST_DWithin(` + freshLoc + `, ` + DispatchAnchorSQL + `, ` + radius + `)
		    OR ` + stepM + ` <= 0
		    OR ` + radius + ` >= ` + maxM + `::float8
		  )`
}

// queueAudience **من يرى هذا الطلبَ في طابوره الآن** — وإليهم وحدَهم يُدفَع
// العرضُ في «للجميع». **السؤالُ نفسُه الذي يسأله بابُ الطابور** من الجهة الأخرى.
func (s *Service) queueAudience(ctx context.Context, orderID string) ([]string, error) {
	q := `
		SELECT u.id::text FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		JOIN orders o ON o.id = $1::uuid
		LEFT JOIN merchants m ON m.id = o.merchant_id
		WHERE u.on_shift AND u.status = 'active'
		  AND o.status = 'dispatching' AND o.driver_id IS NULL
		  AND `
	args := []any{orderID}
	if s.ProximityEnabled(ctx) {
		dp := s.DispatchProximity(ctx)
		q += QueueVisibleSQL("u.id", 2)
		args = append(args, dp.CashLimit, dp.MaxActive, dp.FreshSec,
			dp.MaxM, dp.InitialM, dp.StepM, dp.TimeoutSec)
	} else {
		q += QueueBaseSQL("u.id")
	}
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
