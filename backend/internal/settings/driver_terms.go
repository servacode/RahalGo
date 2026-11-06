package settings

import "strings"

// DriverTermsClause **بندُ التعامل برّا المنصّة** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٧) —
// في شروط السائق تحت «سلوكُك»، **وتطبيقُه الجديد يطلب الموافقةَ عليه قبل فتح الدوام.**
const DriverTermsClause = "أخذ زبون أو رقمه للتعامل برّا المنصّة يعني إيقاف الحساب."

// DriverMapTraceLine **سطرُ نقاط الخريطة** (قرارُ المالك ٢٠٢٦-١١-٠٦) — نقاطُ `map_trace_points`
// (الهجرة ٠٤٦٦) بلا هويّة، تُجمع لتحسين الخريطة. في خصوصيّة السائق وشروطه.
const DriverMapTraceLine = "ونحفظ من طريقك نقاطاً بلا اسمك ولا ساعة ولا رقم طلب — موضعاً ويوماً فقط — لنحسّن خريطة المدينة ونضيف إليها الحارات الناقصة، وتُمحى بعد تسعين يوماً."

// driverMapTraceTitle عنوانُ كتلة السطر حين يُلحَق بنصٍّ عدّله المالك.
const driverMapTraceTitle = "تحسين الخريطة"

// EffectiveDriverPrivacy **خصوصيّةُ السائق كما تُعرض** — النصُّ (المعدَّل من اللوحة أو الافتراض)
// **وفيه سطرُ نقاط الخريطة أبداً**: إن خلا منه أُلحق كتلةً بآخره (كبند `EffectiveDriverTerms`).
func EffectiveDriverPrivacy(text string) string {
	if strings.Contains(text, DriverMapTraceLine) {
		return text
	}
	block := driverMapTraceTitle + "\n" + DriverMapTraceLine
	if strings.TrimSpace(text) == "" {
		return block
	}
	return strings.TrimRight(text, "\n ") + "\n\n" + block
}

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
