package orders

// ══════════════════════════════════════════════════════════════════════
// **تدويرُ السائقين على الزبون نفسِه** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٤)
// ══════════════════════════════════════════════════════════════════════
//
// **سائقٌ يصل الزبونَ نفسَه كلَّ يومٍ يعرف بابَه ورقمَه وموعدَه** — فيعرض عليه أن
// يوصله برّا التطبيق. **فإن وصّل له `drivers.same_customer_weekly_max` طلباتٍ
// مسلَّمةً خلال سبعة أيّام** («فوق الحدّ»)، **ذهب الطلبُ التالي لهذا الزبون إلى
// سائقٍ آخر — إن وُجد.**
//
// # تفضيلٌ ليّنٌ لا منع
//
// **المؤهَّلون لا ينقصون أبداً** — الترتيبُ وحدَه يتبدّل: «فوق الحدّ» آخرُ الصفّ في
// كلّ استعلامٍ يختار صاحبَ الدور، **فإن لم يبقَ غيرُه أخذه فوراً.** **فلا طلبَ
// يبقى بلا سائقٍ بسبب القاعدة.** وصفرُ الإعداد يطفئها.
//
// **وطلبٌ بلا زبون** («لدي توصيلة») لا تنطبق عليه — `customer_id` فارغ.

import (
	"context"
)

// sameCustomerOverSQL **صحيحٌ إن كان السائق `drv` فوق الحدّ مع الزبون `cust`** —
// طلباتٌ مسلَّمةٌ خلال سبعة أيّام **بلا الطلب نفسِه** (`self`)، و`capArg` الحدّ.
//
// **ونصٌّ لا وسيط** — يُركَّب في استعلاماتٍ مختلفة الوسائط: الدورُ والقربُ والعدلُ
// ونفسُ المسار والطابور. **وفارغُ الزبون أو صفرُ الحدّ كاذبٌ أبداً.**
func sameCustomerOverSQL(drv, cust, self, capArg string) string {
	return `(` + cust + ` IS NOT NULL AND ` + capArg + `::int > 0 AND (
		SELECT count(*) FROM orders h
		 WHERE h.customer_id = ` + cust + ` AND h.driver_id = ` + drv + `
		   AND h.id <> ` + self + ` AND h.status = 'delivered'
		   AND h.delivered_at > now() - interval '7 days') >= ` + capArg + `::int)`
}

// sameCustomerCap **الحدُّ من الإعدادات** — وصفرٌ أو سالبٌ يعني «مطفأ».
func (s *Service) sameCustomerCap(ctx context.Context) int64 {
	n := s.settingInt(ctx, "drivers.same_customer_weekly_max")
	if n < 0 {
		return 0
	}
	return n
}

// overQuota **أهذا السائقُ فوق الحدّ مع زبون هذا الطلب؟**
func (s *Service) overQuota(ctx context.Context, driverID, orderID string) bool {
	capN := s.sameCustomerCap(ctx)
	if capN <= 0 || driverID == "" {
		return false
	}
	var over bool
	if err := s.db.QueryRow(ctx, `SELECT `+sameCustomerOverSQL("$1::uuid", "o.customer_id", "o.id", "$3")+`
		FROM orders o WHERE o.id = $2`, driverID, orderID, capN).Scan(&over); err != nil {
		return false
	}
	return over
}

// sameCustomerAlternative **أفي سائقٌ آخرُ مؤهَّلٌ الآن ليس فوق الحدّ مع الزبون؟**
//
// **سؤالُ نفس المسار قبل أن يتخطّى صاحبَه** — يُسأل بسؤال التوزيع نفسِه في النمط
// القائم: «بالدور» مرشّحُ الدور بلا صاحب المسار، و«للجميع» من يرى الطلبَ في طابوره.
// **ولا بديلَ ⇒ لا تخطّي**: نفسُ المسار كما كان.
func (s *Service) sameCustomerAlternative(ctx context.Context, orderID, except string) bool {
	if s.AssignmentMode(ctx) == "rotation" {
		limit := s.settingInt(ctx, "drivers.cash_limit")
		maxActive := s.settingInt(ctx, "drivers.max_active_orders")
		skip := append([]string{except}, s.excludedDrivers(ctx, orderID)...)
		id, _, err := s.pickRotationCandidate(ctx, orderID, skip, limit, maxActive)
		return err == nil && id != "" && id != except && !s.overQuota(ctx, id, orderID)
	}
	ids, err := s.queueAudience(ctx, orderID)
	if err != nil {
		return false
	}
	for _, id := range ids {
		if id != except && !s.overQuota(ctx, id, orderID) {
			return true
		}
	}
	return false
}
