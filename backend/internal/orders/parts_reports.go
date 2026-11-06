package orders

// ══════════════════════════════════════════════════════════════════════
// **التقاريرُ والعدّاداتُ لطلبٍ بعدّة محطّات — كلُّ متجرٍ بقسمه** (الدفعة P9)
// ══════════════════════════════════════════════════════════════════════
//
// **كان كلُّ عدّادٍ يقرأ `orders.merchant_id`** — فالمتجرُ الثاني لا يرى طلبَه في عدّاد مندوبه
// ولا في الخريطة ولا في غرفة الطوارئ، **والأوّلُ يُنسب له مالُ الطلب كلُّه.**
//
// **وطلبُ المتجر الواحد بلا قسمٍ فيبقى على `merchant_id` حرفاً.**

// LiveStoreOrdersSQL **طلباتُ متجرٍ حيّةٌ له**: طلبُه الوحيد، **أو قسمٌ حيٌّ له في طلبٍ بعدّة
// محطّات** (لم يُرفض ولم يُشَل). على الاسم المستعار `o`؛ و`p` تعبيرُ معرّف المتجر.
//
// **وبمصفوفةٍ تُحسب مرّةً لا بـ`EXISTS` لكلّ صفّ** — كـ`StoreOrdersSQL`.
func LiveStoreOrdersSQL(p string) string {
	return `((o.stops = 1 AND o.merchant_id = ` + p + `) OR o.id = ANY (ARRAY(
	         SELECT lp.order_id FROM order_parts lp WHERE lp.merchant_id = ` + p + `
	            AND lp.status NOT IN ('rejected', 'removed'))))`
}

// RepLiveOrdersSQL **طلباتُ متاجرِ مندوبٍ حيّةٌ لها** — `rep` تعبيرُ معرّفه. على الاسم المستعار `o`.
func RepLiveOrdersSQL(rep string) string {
	return `((o.stops = 1 AND o.merchant_id IN (SELECT rm.id FROM merchants rm
	                                          WHERE rm.sales_rep_user_id = ` + rep + `))
	         OR o.id = ANY (ARRAY(
	            SELECT lp.order_id FROM order_parts lp JOIN merchants rm ON rm.id = lp.merchant_id
	             WHERE rm.sales_rep_user_id = ` + rep + ` AND lp.status NOT IN ('rejected', 'removed'))))`
}

// StoreGoodsSQL **بضاعةُ متجرٍ في الطلب بسعر البيع** — مجموعُ بنوده هو. على الاسم المستعار `o`.
func StoreGoodsSQL(p string) string {
	return `COALESCE((SELECT sum(sg.unit_price * sg.qty) FROM order_items sg
	                  WHERE sg.order_id = o.id AND COALESCE(sg.merchant_id, o.merchant_id) = ` + p + `), 0)`
}

// StoreMarginSQL **هامشُ بنود متجرٍ في الطلب** — `OrderMarginSQL` مقصوراً عليه. على الاسم المستعار `o`.
func StoreMarginSQL(p string) string {
	return `COALESCE((SELECT sum((sm.unit_price - sm.merchant_price) * sm.qty) FROM order_items sm
	                  WHERE sm.order_id = o.id AND COALESCE(sm.merchant_id, o.merchant_id) = ` + p + `), 0)`
}

// RepStoreShareSQL **نصيبُ المندوب `rep` من الطلب عن المتجر `p` — صافياً من الدفتر.**
//
// طلبُ المتجر الواحد: سطورُ عمولته وتسوياتها بمرجع الطلب كما كانت. **وطلبٌ بعدّة محطّات**:
// صافيه على الطلب مقسوماً بنسبة تفصيل هذا المتجر (`order_parts.rep_commission`) إلى مجموع
// تفصيله — **فما عُكس عن الطلب يُعكس عن متاجره بنسبتها.** على الاسم المستعار `o`.
func RepStoreShareSQL(p, rep string) string {
	net := `(SELECT COALESCE(sum(rt.amount), 0) FROM wallet_transactions rt
	         WHERE rt.user_id = ` + rep + ` AND rt.ref = o.id::text
	           AND rt.kind IN ('commission', 'adjustment'))`
	return `(CASE WHEN o.stops > 1 THEN COALESCE((
	           SELECT ` + net + ` * rp.rep_commission
	                  / NULLIF((SELECT sum(rp2.rep_commission) FROM order_parts rp2
	                            WHERE rp2.order_id = o.id AND rp2.rep_id = ` + rep + `), 0)
	             FROM order_parts rp
	            WHERE rp.order_id = o.id AND rp.merchant_id = ` + p + ` AND rp.rep_id = ` + rep + `), 0)
	        ELSE ` + net + ` END)::bigint`
}
