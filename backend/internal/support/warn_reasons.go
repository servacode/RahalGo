package support

// ══════════════════════════════════════════════════════════════════════
//  **أسبابُ الإنذار — مصنّفةٌ لا نصٌّ حرّ**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٥: «يجب أن يكون هناك أسبابٌ جاهزةٌ للإنذار».)
//
// # ولماذا لا يُترك نصّاً
//
// **كان الحقلُ نصّاً حرّاً يكتبه من يُنذر** — فيُكتب السببُ الواحدُ بعشرة
// ألفاظ. **و«هذا الزبونُ أُنذر ثلاثاً لنفس السبب» جملةٌ لا تُقال** إن كان
// كلُّ إنذارٍ بلفظٍ مختلف.
//
// **ومن السبب يُشتقّ التكرارُ والحكم** — وهو عينُ ما فُعل في أسباب الشكوى.
//
// # والأسبابُ تتبع الدور
//
// **«رفضُ الاستلام» لا يُنذَر به سائق، و«التأخّرُ في التسليم» لا يُنذَر به
// زبون.** **وقائمةٌ واحدةٌ للجميع تُري من يُنذر أسباباً لا تخصّ من أمامه**،
// فيختار أقربَها فيُكتب سببٌ لا يصف ما وقع.

// WarnReason **سببٌ ودورُ من يُنذَر به.**
type WarnReason struct {
	Code string `json:"code"`
	// Roles **الأدوارُ التي يصلح لها** — وفارغةٌ تعني الجميع.
	Roles []string `json:"roles,omitempty"`
}

// WarnReasons **ما يملك المكتبُ الإنذارَ به.**
//
// **والرمزُ لغةُ الآلة والاسمُ لغةُ الناس** — الترجمةُ في الشاشة، **والمحرّكُ
// لا يعرف لغةَ من يقرأ.**
var WarnReasons = []WarnReason{
	// ── ما يخصّ الزبون ────────────────────────────────────────────────
	{Code: "abuse_driver", Roles: []string{"customer"}},
	{Code: "repeated_cancel", Roles: []string{"customer"}},
	{Code: "refused_delivery", Roles: []string{"customer"}},
	{Code: "wrong_address", Roles: []string{"customer"}},
	{Code: "unreachable", Roles: []string{"customer"}},
	// ── وما يخصّ الجميع ───────────────────────────────────────────────
	{Code: "fraud_attempt"},
	{Code: "terms_violation"},
	{Code: "other"},
}

// WarnReasonsFor **ما يصلح لهذا الدور.**
func WarnReasonsFor(role string) []WarnReason {
	out := []WarnReason{}
	for _, r := range WarnReasons {
		if len(r.Roles) == 0 {
			out = append(out, r)
			continue
		}
		for _, x := range r.Roles {
			if x == role {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

// WarnReasonAr **اسمُ السبب بالعربيّة — لِما يُخزَّن في إشعارٍ يُقرأ.**
//
// **والإشعارُ يُخزَّن نصّاً في القاعدة ويُقرأ كما خُزّن** — **فترجمتُه في
// الواجهة تأتي بعد فوات الأوان**، ويقرأ صاحبُه رمزاً إنكليزيّاً في جملةٍ
// عربيّة. (وهو عينُ ما وقع في أسباب الشكوى ٢٠٢٦-٠٨-٠٨.)
func WarnReasonAr(code string) string {
	switch code {
	case "abuse_driver":
		return "إساءةٌ إلى سائق"
	case "repeated_cancel":
		return "إلغاءٌ متكرّرٌ للطلبات"
	case "refused_delivery":
		return "رفضُ استلام الطلب"
	case "wrong_address":
		return "عنوانٌ خاطئٌ متكرّر"
	case "unreachable":
		return "عدمُ الردّ على الهاتف"
	case "fraud_attempt":
		return "محاولةُ احتيال"
	case "terms_violation":
		return "مخالفةُ شروط الاستخدام"
	case "other":
		return "سببٌ آخر"
	}
	return code
}

// ValidWarnReason **أيُنذَر بهذا السبب هذا الدور؟**
//
// **والحكمُ في المحرّك لا في الشاشة** — من نادى الواجهةَ البرمجيّة مباشرةً
// تجاوز قائمةَ الاختيار.
func ValidWarnReason(code, role string) bool {
	for _, r := range WarnReasonsFor(role) {
		if r.Code == code {
			return true
		}
	}
	return false
}
