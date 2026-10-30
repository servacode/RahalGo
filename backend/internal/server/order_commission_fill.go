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
		       COALESCE(m.commission_percent, o.snap_merchant_commission_percent)
		  FROM orders o JOIN merchants m ON m.id = o.merchant_id
		 WHERE o.id = ANY($1)`, ids)
	if err != nil {
		return
	}
	defer rows.Close()
	type row struct {
		base int64
		pct  *pricing.Percent2
	}
	got := map[string]row{}
	for rows.Next() {
		var id string
		var r row
		if rows.Scan(&id, &r.base, &r.pct) == nil {
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
		list[i].PlatformCommission = com.Of(r.base)
		list[i].MerchantNet = r.base - list[i].PlatformCommission
		list[i].CommissionPct = int(p.RoundedPercent())
		list[i].CommissionExact = pricing.Percent2Ptr(p)
	}
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
