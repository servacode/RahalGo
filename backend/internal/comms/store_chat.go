package comms

// ══════════════════════════════════════════════════════════════════════
// **حديثُ «لدي توصيلة» في قائمة المتجر** (قرارُ المالك ٢٠٢٦-١١-٢٦)
// ══════════════════════════════════════════════════════════════════════
//
// **ولماذا لا يُقرأ من «محادثاتي»** (`Threads`): صاحبُ المتجر يحمل تطبيقَ الزبون أيضاً
// ليطلب عشاءه، **و«محادثاتي» في تطبيق الزبون تصير فيها توصيلاتُ متجره** لو أُضيف إليها.
// **فيُقرأ العددُ مع التوصيلة نفسِها** — في قائمة التوصيلات وشاشة متابعتها — ولا يمسّ
// قائمةَ الزبون.

import "context"

// StoreChat **ما ينتظر المتجرَ في حديث توصيلة** — وكم سطراً فيه.
type StoreChat struct {
	// Unread **رسائلُ الكابتن التي لم يقرأها المتجرُ بعد** — في ولاية الكابتن الحاليّ
	// (أو آخرِ من تكلّم إن لم يكن لها كابتنٌ الآن)، كما يَسِم `MarkRead`.
	Unread int
	// Lines **كم سطراً في تلك الولاية** — فيُعرض زرُّ الحديث لتوصيلةٍ انتهت وفيها كلام.
	Lines int
}

// StoreChats **عدُّ الحديث لكلّ توصيلةٍ من هذه التوصيلات** — والغائبةُ بلا حديث.
//
// **ولا يسأل عن الملكيّة**: المنادي حرس التوصيلاتِ بمتجره (`ownsMerchant`) قبل أن يبلغ هنا.
func (s *Service) StoreChats(ctx context.Context, orderIDs []string) (map[string]StoreChat, error) {
	out := map[string]StoreChat{}
	if len(orderIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx, `
		WITH t AS (
		    SELECT o.id,
		           COALESCE(o.driver_id,
		                    (SELECT y.driver_id FROM order_messages y
		                      WHERE y.order_id = o.id AND y.driver_id IS NOT NULL
		                      ORDER BY y.created_at DESC LIMIT 1)) AS tenure
		    FROM orders o
		    WHERE o.id = ANY($1::uuid[]) AND o.kind = $2
		)
		SELECT t.id::text,
		       count(*) FILTER (WHERE x.sender_role <> $3 AND x.read_at IS NULL),
		       count(*)
		FROM t
		JOIN order_messages x ON x.order_id = t.id AND x.driver_id = t.tenure
		-- **صورةٌ محجوزةٌ لا تُعدّ ولا تُعرض** حتّى يمرّرها المكتب (٢٠٢٦-١١-٢٠).
		WHERE x.held_at IS NULL OR x.released_at IS NOT NULL
		GROUP BY t.id`, orderIDs, kindMerchantDelivery, string(RoleMerchant))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var c StoreChat
		if err := rows.Scan(&id, &c.Unread, &c.Lines); err != nil {
			return nil, err
		}
		out[id] = c
	}
	return out, rows.Err()
}
