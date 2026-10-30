package orders

// ══════════════════════════════════════════════════════════════════════
// **ملفُّ الطلب في سجلّ اللوحة** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
// ══════════════════════════════════════════════════════════════════════
//
// «لمّا افتح السجلّ لازم افهم لكلّ طلب: شو صار، كيف مشي، وحركة المال.»
//
// **شرطٌ واحدٌ للفلتر وللعلامة على الكرت** — «متأخّر» و«عمولته ممحيّة» و«مُحوَّل»
// تُكتب هنا مرّةً، **فلا يقول الكرتُ «متأخّر» والفلترُ لا يجده.**

// StoreTransferNote **أوّلُ ملاحظة حدث «تحويلٌ إلى متجرٍ آخر»** — يكتبها التحويلُ
// (`server/order_transfer.go`) ويقرؤها السجلّ ليقول «مُحوَّل».
const StoreTransferNote = "تحويلٌ إلى متجرٍ آخر"

// LateDeliverySQL **سُلّم بعد مهلة عمر الطلب** (`orders.delivery_timeout_min`) — المهلةُ نفسُها
// التي تقول بها اللوحةُ «متأخّر» للطلب الجاري (`TooLongSQL`)، **مقيسةً على وقت التسليم.**
func LateDeliverySQL(l StuckLimits) string {
	return `(o.delivered_at IS NOT NULL AND o.delivered_at > o.created_at + ` + mins(l.DeliveryMin) + `)`
}

// TransferredSQL **حُوّل الطلبُ إلى متجرٍ آخر** — من مساره.
const TransferredSQL = `EXISTS (SELECT 1 FROM order_events te
	WHERE te.order_id = o.id AND te.note LIKE '` + StoreTransferNote + `%')`

// CasesSQL **كم شكوى ومطالبةً وطارئاً على الطلب** — مفتوحةً أو مغلقة.
const CasesSQL = `((SELECT count(*) FROM tickets tk WHERE tk.order_id = o.id)
	+ (SELECT count(*) FROM disputes dp WHERE dp.order_id = o.id)
	+ (SELECT count(*) FROM driver_emergencies em WHERE em.order_id = o.id))::int`

// payFilters **فلترُ طريقة الدفع** — نقدٌ: على الزبون مبلغٌ نقديّ · محفظةٌ: دُفع منها شيء.
// **والمختلطُ يظهر في الاثنين** — فيه الطريقتان.
var payFilters = map[string]string{
	"cash":   `(o.cash_due > 0 OR (o.wallet_paid = 0 AND o.payment_method = 'cash'))`,
	"wallet": `(o.wallet_paid > 0)`,
}

// staffSearchPhone و staffSearchName **بحثُ الموظّفين الأعرض** — هاتفُ السائق والمتجر،
// واسمُ السائق والمتجر وكودُ الخصم. `p` رقمُ معاملٍ مُعدٌّ سلفاً.
func staffSearchPhone(p string) string {
	return `EXISTS (SELECT 1 FROM users sd WHERE sd.id = o.driver_id AND sd.phone::text LIKE '%' || ` + p + ` || '%' ESCAPE '\')
		OR EXISTS (SELECT 1 FROM merchants sm WHERE sm.id = o.merchant_id AND sm.phone::text LIKE '%' || ` + p + ` || '%' ESCAPE '\')`
}

func staffSearchName(p string) string {
	return `EXISTS (SELECT 1 FROM users sd WHERE sd.id = o.driver_id AND sd.full_name ILIKE '%' || ` + p + ` || '%' ESCAPE '\')
		OR EXISTS (SELECT 1 FROM merchants sm WHERE sm.id = o.merchant_id AND sm.name ILIKE '%' || ` + p + ` || '%' ESCAPE '\')
		OR COALESCE(o.promo_code, '') ILIKE '%' || ` + p + ` || '%' ESCAPE '\'`
}
