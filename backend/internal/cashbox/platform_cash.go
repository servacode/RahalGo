package cashbox

// ══════════════════════════════════════════════════════════════════════
// **سقفُ نقد الكابتن يعدّ مالَ المنصّة وحدَه** — قرارُ المالك ٢٠٢٦-١١-١٠ (القرار ٧، الدفعة B9)
// ══════════════════════════════════════════════════════════════════════
//
// **ثمنُ البضاعة ليس مالَ المنصّة**: الكابتنُ يدفعه للمتجر عند الاستلام ويقبضه من الزبون عند
// التسليم. **فالسقفُ يعدّ الأجرةَ والعمولةَ والهامشَ وحدَها** — لكلّ الطلبات (والقديمةُ معها، جوابُ
// المالك):
//
//	بعد الاستلام  نقدُ الطلب ناقصَ ما دفعه للمتاجر (صفوفُ التسوية بتصحيحاتها)
//	قبل الاستلام  نقدُ الطلب ناقصَ تقدير حقّ المتاجر: كلفةُ البنود ناقصَ عمولتها بنسبة المتجر
//	              (أو لقطةِ الطلب)
//	المشتريات     نقدُ الطلب ناقصَ ما دُفع للمتجر (أو ثمنِ البضاعة الموثَّق)
//	الدولار       لا يُعدّ — ثمنُ بضاعةٍ يمرّ بيده ويخرج منها (`goods_usd_syp` خارجَ الحسبة)
//
// **ولا يهبط تحت الصفر.** و`held` (ما قبضه ولم يسلّمه) كما هو: **قبض عند التسليم نقدَ الطلب ناقصَ
// ما دفعه للمتجر** — فهو مالُ المنصّة أصلاً.

import "context"

// PlatformCashSQL **مالُ المنصّة في نقد طلبٍ مفتوح** — تعبيرٌ على اسم جدول الطلبات `o`.
func PlatformCashSQL(o string) string {
	return `GREATEST(` + o + `.cash_due - CASE
		WHEN ` + o + `.kind = 'custom' THEN
			COALESCE(` + o + `.custom_store_paid,
			         CASE WHEN ` + o + `.custom_mode = 'purchase' THEN ` + o + `.custom_goods_amount END, 0)
		WHEN ` + o + `.kind = 'standard' THEN
			CASE WHEN EXISTS (SELECT 1 FROM merchant_settlements pms WHERE pms.order_id = ` + o + `.id)
			     THEN COALESCE((SELECT sum(pms.amount + pms.adjusted_amount - pms.reversed_amount)
			                      FROM merchant_settlements pms WHERE pms.order_id = ` + o + `.id), 0)
			     ELSE COALESCE((SELECT sum(poi.merchant_price * poi.qty
			                               - floor(poi.merchant_price * poi.qty
			                                       * COALESCE(pm.commission_percent,
			                                                  ` + o + `.snap_merchant_commission_percent, 0) / 100))
			                      FROM order_items poi
			                      JOIN merchants pm ON pm.id = COALESCE(poi.merchant_id, ` + o + `.merchant_id)
			                     WHERE poi.order_id = ` + o + `.id), 0)
			END
		ELSE 0 END, 0)`
}

// InflightSQL **مالُ المنصّة في طلباتٍ مُسندةٍ لسائقٍ لم تُغلَق** — تعبيرُ السائق `drv`.
func InflightSQL(drv string) string {
	return `COALESCE((SELECT sum(` + PlatformCashSQL("ifo") + `) FROM orders ifo
		WHERE ifo.driver_id = ` + drv + ` AND ifo.closed_at IS NULL), 0)`
}

// PlatformCashOf **مالُ المنصّة في نقد طلبٍ واحد** — لما يدخل يدَ السائق حين يُسنَد إليه.
func PlatformCashOf(ctx context.Context, q Querier, orderID string) (int64, error) {
	var v int64
	err := q.QueryRow(ctx, `SELECT COALESCE((SELECT `+PlatformCashSQL("o")+`::bigint
		FROM orders o WHERE o.id = $1::uuid), 0)`, orderID).Scan(&v)
	return v, err
}
