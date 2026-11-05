package server

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// fillOrderCommission **عمولةُ المنصّة من المتجر على بطاقة الطلب في اللوحة** (قرارُ المالك ٢٠٢٦-١٠-٢٦:
// «عمولة المنصّة مو موجودة — ما بتبيّن»). بالقاعدة نفسِها التي تُسوّي عند التسليم (`settleMerchant`):
// **نسبةُ المتجر الخاصّةُ إن كانت، وإلّا نسبةُ لقطة الطلب**، على سعر شراء أصنافه، **مقرّبةً لأقرب ليرة**.
// تقديرٌ قبل التسليم، **والدفترُ هو الحَكَم بعده.**
//
// **ومتجرٌ سُوّي في الطلب يُعرض بصفّ تسويته** (`settledMoney`) — لا بنسبته الحيّة: بُدّلت بعد
// الاستلام فالدفترُ على نسبة الاستلام (خلل PD-1)، **والكرتُ يتبعه لا يخالفه.**
func (s *Server) fillOrderCommission(ctx context.Context, list []orders.Order) {
	ids := make([]string, 0, len(list))
	for i := range list {
		if list[i].MerchantID != "" {
			ids = append(ids, list[i].ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	rows, err := s.pg.Query(ctx, `
		SELECT o.id::text,
		       COALESCE((SELECT sum(oi.merchant_price * oi.qty) FROM order_items oi
		                  WHERE oi.order_id = o.id AND COALESCE(oi.merchant_id, o.merchant_id) = o.merchant_id), 0),
		       COALESCE(m.commission_percent, o.snap_merchant_commission_percent),
		       ms.commission_percent, ms.amount + ms.adjusted_amount
		  FROM orders o JOIN merchants m ON m.id = o.merchant_id
		  LEFT JOIN merchant_settlements ms ON ms.order_id = o.id AND ms.merchant_id = o.merchant_id
		 WHERE o.id = ANY($1)`, ids)
	if err != nil {
		return
	}
	defer rows.Close()
	type row struct {
		base        int64
		pct, frozen *pricing.Percent2
		due         *int64
	}
	got := map[string]row{}
	for rows.Next() {
		var id string
		var r row
		if rows.Scan(&id, &r.base, &r.pct, &r.frozen, &r.due) == nil {
			got[id] = r
		}
	}
	for i := range list {
		r, ok := got[list[i].ID]
		if !ok || r.base <= 0 {
			continue
		}
		com := pricing.MerchantCommission(ctx, s.settings, r.pct)
		p := com.Percent()
		c := com.Of(r.base)
		if sc, sp, settled := settledMoney(r.base, p, r.frozen, r.due); settled {
			c, p = sc, sp
		}
		list[i].PlatformCommission = c
		list[i].MerchantNet = r.base - c
		list[i].CommissionPct = int(p.RoundedPercent())
		list[i].CommissionExact = pricing.Percent2Ptr(p)
	}
}

// settledMoney **عمولةُ متجرٍ سُوّي في الطلب كما قُيّدت في صفّ تسويته** (خلل PD-1، ٢٠٢٦-١١-٠٥).
//
// يُسوّى المتجرُ عند الاستلام، **ونسبتُه تُحفظ في صفّه** (`merchant_settlements.commission_percent`،
// الهجرة 0506) فيُطابَق التسليمُ بها ولو بُدّلت بعده. **فالعرضُ يقرأ الصفَّ** — وإلّا رأى المكتبُ
// والمتجرُ عمولةً بالحيّة غيرَ ما قُيّد ودفعه السائق:
//
//	نسبةُ الصفّ        ⇒ بها على سعر الشراء، كما يُعيدها `merchantShares`
//	صفٌّ بلا نسبة     ⇒ أقدمُ من 0506: العمولةُ = سعرُ الشراء − المستحقّ المقيَّد (`amount + adjusted_amount`)،
//	                    ونسبتُها الحيّةُ إن أعادتها بعينها وإلّا تُشتقّ منها
//	لا صفّ            ⇒ `false` — لم يُستلم منه بعد، **فالنسبةُ الحاليّةُ تحكمه**
//
// و`live` النسبةُ الحاليّة، و`due` المستحقُّ المقيَّد (فراغُه لا صفّ).
func settledMoney(base int64, live pricing.Percent2, frozen *pricing.Percent2, due *int64) (int64, pricing.Percent2, bool) {
	if frozen != nil {
		return pricing.Amount{Mode: pricing.ModePercent2, Value: int64(*frozen)}.Of(base), *frozen, true
	}
	if due == nil || base <= 0 {
		return 0, 0, false
	}
	c := min(max(base-*due, 0), base)
	if (pricing.Amount{Mode: pricing.ModePercent2, Value: int64(live)}).Of(base) == c {
		return c, live, true
	}
	return c, pricing.Percent2(c * 10000 / base), true
}

// fillCommissionTaken **العمولةُ المخزّنةُ وما مُحي منها للطلب المسلَّم** — كرتُ السجلّ في اللوحة
// (قرارُ المالك ٢٠٢٦-١٠-٣٠: «أجرة X · عمولة Y» تحت الإجمالي). قراءةٌ للعرض، وتعذّرُها لا يُسقط القائمة.
func (s *Server) fillCommissionTaken(ctx context.Context, list []orders.Order) {
	ids := make([]string, 0, len(list))
	for i := range list {
		if list[i].Status == orders.StDelivered {
			ids = append(ids, list[i].ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	rows, err := s.pg.Query(ctx, `
		SELECT id::text, platform_commission, commission_waived FROM orders WHERE id = ANY($1)`, ids)
	if err != nil {
		return
	}
	defer rows.Close()
	type row struct{ taken, waived int64 }
	got := map[string]row{}
	for rows.Next() {
		var id string
		var r row
		if rows.Scan(&id, &r.taken, &r.waived) == nil {
			got[id] = r
		}
	}
	for i := range list {
		if r, ok := got[list[i].ID]; ok {
			taken := r.taken
			list[i].CommissionTaken = &taken
			list[i].CommissionWaived = r.waived
		}
	}
}
