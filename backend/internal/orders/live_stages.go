package orders

// ══════════════════════════════════════════════════════════════════════
// **مراحلُ الطلبات الجارية — لرئيسيّة المدير ولمُرشِّح الطلبات معاً**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤: «لازم أفهم كلّ شي عم يصير بالمنصّة بدون ما
// أتنقّل بين الصفحات».)
//
// **الطلباتُ الجاريةُ مقسومةٌ على خمس مراحل** — ومجموعُها كلُّ ما لم يُغلق:
//
//	at_store_prep  جديدٌ أو مقبولٌ أو قيد التحضير (قبل السائق)
//	dispatching    يبحث عن سائق
//	to_store       السائقُ في طريقه إلى المتجر
//	at_store       السائقُ عند المتجر
//	to_customer    استلم وفي الطريق إلى الزبون أو عند بابه
//
// **وقائمةٌ واحدةٌ للعدّ وللمُرشِّح** — فالرقمُ في الرئيسيّة هو عددُ ما
// تعرضه شاشةُ الطلبات حين تُفتح على المرحلة نفسِها.

// LiveStages **ترتيبُ المراحل كما تُعرض.**
var LiveStages = []string{"at_store_prep", "dispatching", "to_store", "at_store", "to_customer"}

// LiveStageStatuses **حالاتُ كلّ مرحلة** — و`nil` لمرحلةٍ لا تُعرف.
func LiveStageStatuses(stage string) []string {
	switch stage {
	case "at_store_prep":
		return []string{StPending, StAccepted, StPreparing}
	case "dispatching":
		return []string{StDispatching}
	case "to_store":
		return []string{StAssigned}
	case "at_store":
		return []string{StAtPickup}
	case "to_customer":
		return []string{StPickedUp, StOnTheWay, StAtDropoff}
	}
	return nil
}

// LiveStageSQL **شرطُ المرحلة نصّاً** على جدول الطلبات باسم `o` — للعدّ
// وللقائمة. **والحالاتُ ثوابتُ الحزمة** لا مُدخَلُ أحد.
func LiveStageSQL(stage string) string {
	return `o.closed_at IS NULL AND o.status IN ` + sqlTextList(LiveStageStatuses(stage))
}
