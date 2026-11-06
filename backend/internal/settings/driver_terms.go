package settings

import "strings"

// DriverTermsClause **بندُ التعامل برّا المنصّة** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٧) —
// في شروط السائق تحت «سلوكُك»، **وتطبيقُه الجديد يطلب الموافقةَ عليه قبل فتح الدوام.**
const DriverTermsClause = "أخذ زبون أو رقمه للتعامل برّا المنصّة يعني إيقاف الحساب."

// driverTermsClauseTitle **عنوانُ كتلة البند** حين يُلحَق بنصٍّ عدّله المالك.
const driverTermsClauseTitle = "التعامل برّا المنصّة"

// EffectiveDriverTerms **شروطُ السائق كما تُعرض وتُوقَّع** — النصُّ (المعدَّل من اللوحة
// أو الافتراض) **وفيه البندُ أبداً**: إن خلا منه أُلحق كتلةً بآخره.
//
// **فالمالكُ قد عدّل النصَّ على الإنتاج قبل البند** — وتعديلُ الافتراض لا يصله. **والبحثُ
// عن الجملة حرفاً**: من أعاد صياغتها بكلماتٍ أخرى أُلحقت الأصليّةُ بعدها.
func EffectiveDriverTerms(text string) string {
	if strings.Contains(text, DriverTermsClause) {
		return text
	}
	block := driverTermsClauseTitle + "\n" + DriverTermsClause
	if strings.TrimSpace(text) == "" {
		return block
	}
	return strings.TrimRight(text, "\n ") + "\n\n" + block
}
