package authz

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ══════════════════════════════════════════════════════════════════════
// **معجمُ الأفعال التي تحتاج تأكيداً بكلمةِ صاحبها** — `ADG-3`
// ══════════════════════════════════════════════════════════════════════
//
// # العقد
//
//	جلسةٌ صالحة · وقدرةٌ قائمة · وإثباتُ تأكيدٍ لهذا الفعل بعينه
//
// **وثلاثتُها شروطٌ لا بدائل** — **والإثباتُ ليس تخويلاً**، والتخويلُ
// ليس جلسة.
//
// # ولماذا معجمٌ واحدٌ لا رايةٌ في كلّ معالِج
//
// **`requiresStepUp = true` مبعثرةً تُنسى في المسار القادم** —
// **ومسارٌ نُسي لا يُكتشَف إلّا بعد أن يُستعمَل.** **والمعجمُ يُقرأ
// كلُّه في مكانٍ واحد**، ويحرسه فحصٌ يمشي على الموجِّه.
//
// # ولا يُدرَج فعلٌ لأنّه إداريّ
//
// **قراءةٌ وتحليلٌ وتذكرةُ دعمٍ ولافتةٌ وإعدادٌ عامّ ومراجعةُ متجرٍ
// وتوثيقُ سائقٍ وإيقافٌ عاديّ** — **كلُّها تمضي بلا تأكيد.** **ومن
// طلب كلمةً في كلّ ضغطةٍ علّم الناسَ أن يكتبوها بلا نظر.**

// Sensitive فعلٌ حسّاسٌ يلزمه إثباتُ تأكيد.
type Sensitive struct {
	// Method فعلُ HTTP — وفارغٌ يعني كلَّها.
	Method string
	// Pattern نمطُ المسار بعد `/api/v1/admin` — و`{}` معلَمةٌ متغيّرة.
	Pattern string
	// Action المعرّفُ الكانونيّ — **هو نفسُه فعلُ التدقيق حيث وُجد**،
	// فلا يُقرأ سجلّان باسمين لفعلٍ واحد.
	Action string
	// TargetType صنفُ الهدف — للقراءة في التدقيق.
	TargetType string
	// TargetSeg ترتيبُ المقطع الذي يحمل معرّفَ الهدف في المسار
	// (صفرٌ للأوّل بعد `/`)، **و`-1` لفعلٍ بلا هدفٍ في المسار.**
	TargetSeg int
	// Params حقولُ الجسم الجوهريّة التي تدخل البصمة.
	//
	// **ولا تُدرَج حقولُ تعليقٍ ولا سببٍ نصّيّ** — **تبديلُها لا
	// يبدّل معنى الفعل.**
	Params []string
	// Conditional اسمُ مصنِّفٍ يحسم أحسّاسٌ هذا النداءُ أم لا.
	//
	// **وفارغٌ يعني «دائماً»** — **وغيرُ الفارغ يعني أنّ الاسمَ لا
	// يكفي**: `PATCH /users/{id}` حظرٌ أو إيقافٌ عاديّ بحسب جسمه.
	Conditional string
}

// أسماءُ المصنِّفات المشروطة — **تُقرأ في الخادم حيث تعيش.**
const (
	// CondSettingSensitivity **تصنيفُ الإعداد بأثره** — الماليُّ
	// والأمنيُّ يلزمهما تأكيد، والعامُّ لا. **وهو تصنيفُ دورةِ ٢١
	// نفسُه** (`criticalSettingKey`) — ولا يُخترَع ثانيةً.
	CondSettingSensitivity = "settingSensitivity"
	// CondStatusIsStrong **الحظرُ والحذفُ يلزمهما تأكيد**،
	// **والإيقافُ العاديُّ لا** — وهو فرقُ دورةِ ١٧ بعينه.
	CondStatusIsStrong = "statusIsStrong"
)

// sensitiveActions **الجدولُ الكانونيّ** — كلُّ سطرٍ مسارٌ قائم.
var sensitiveActions = []Sensitive{
	// ── سياسةُ الصلاحيّات — أشدُّ ما في المنصّة ──────────────────
	//
	// **ومن ملك `roles.manage` ملك كلَّ شيءٍ بالتعريف** — **فتأكيدُ
	// كلّ منحٍ هو الحدُّ الوحيدُ الباقي.**
	{"POST", "/roles/{code}/capabilities", "admin.role_capability_grant",
		"role", 1, []string{"capability"}, ""},
	{"DELETE", "/roles/{code}/capabilities/{cap}", "admin.role_capability_revoke",
		"role", 1, nil, ""},
	{"POST", "/users/{id}/roles", "admin.role_grant",
		"user", 1, []string{"role"}, ""},
	{"DELETE", "/users/{id}/roles/{role}", "admin.role_revoke",
		"user", 1, nil, ""},

	// ── استعادةُ الاعتماد وحالُ الحساب ──────────────────────────
	{"POST", "/users/{id}/password", "admin.password_reset", "user", 1, nil, ""},
	{"PATCH", "/users/{id}", "admin.user_update", "user", 1,
		[]string{"status"}, CondStatusIsStrong},

	// ── مالٌ يتحرّك ─────────────────────────────────────────────
	//
	// **والقراءةُ الماليّةُ لا تُؤكَّد** — **ولا يتحرّك بها شيء.**
	{"POST", "/users/{id}/wallet", "finance.wallet_apply", "user", 1,
		[]string{"amount", "kind"}, ""},
	{"POST", "/users/{id}/incentive", "finance.incentive", "user", 1,
		[]string{"amount"}, ""},
	{"POST", "/payouts/{id}/decide", "finance.payout_decide", "payout", 1,
		[]string{"approve", "amount"}, ""},
	{"POST", "/drivers/{id}/settle", "finance.driver_settle", "user", 1,
		[]string{"amount"}, ""},
	{"POST", "/expenses", "finance.expense_added", "expense", -1,
		[]string{"amount", "category_id"}, ""},
	{"POST", "/expenses/{id}/void", "finance.expense_voided", "expense", 1, nil, ""},
	{"POST", "/orders/{id}/compensate-driver", "finance.compensate_driver",
		"order", 1, []string{"amount"}, ""},

	// ── الإعداداتُ ذاتُ الأثر ───────────────────────────────────
	{"PUT", "/settings/{key}", "admin.setting_update", "setting", 1,
		[]string{"value"}, CondSettingSensitivity},
}

// LookupSensitive **أيلزم هذا النداءَ تأكيد؟**
//
// **والأخصُّ يغلب** — كما في جدول القدرات.
func LookupSensitive(method, pattern string) (Sensitive, bool) {
	best, found, bestScore := Sensitive{}, false, -1
	for _, s := range sensitiveActions {
		if s.Method != "" && s.Method != method {
			continue
		}
		if !patternsEqual(s.Pattern, pattern) {
			continue
		}
		score := literalSegments(s.Pattern) * 2
		if s.Method != "" {
			score++
		}
		if score > bestScore {
			best, found, bestScore = s, true, score
		}
	}
	return best, found
}

// SensitiveActions المعجمُ كما هو — **تقرؤه الحرّاسُ واللوحة.**
func SensitiveActions() []Sensitive { return sensitiveActions }

// SensitiveCount عددُ الأفعال المؤكَّدة.
func SensitiveCount() int { return len(sensitiveActions) }

// ══════════════════════════════════════════════════════════════════════
// **بصمةُ ما نوى الفاعلُ فعلَه — تُحسَب في موضعٍ واحد**
// ══════════════════════════════════════════════════════════════════════
//
// **يقرؤها الخادمُ عند الإصدار والاستهلاك، ويقرؤها المِسنَدُ حين
// يصنع إثباتاً لفحصٍ يقيس عقداً آخر** — **وحسابان يفترقان يومَ
// يُضاف حقل.**

// Material حقولُ التبديل الجوهريّةُ في نصٍّ ثابتِ الترتيب.
//
// **ولو أُخذ الجسمُ كلُّه لَبطل الإثباتُ بتبديل تعليق** — **ولو
// أُهملت الحقولُ لَصار «أكّدتُ قيدَ خمسين» إذناً بخمسِ مئة.**
func Material(act Sensitive, body []byte) string {
	if len(act.Params) == 0 {
		return ""
	}
	var in map[string]any
	if err := json.Unmarshal(body, &in); err != nil {
		return ""
	}
	keys := append([]string(nil), act.Params...)
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		v, ok := in[k]
		if !ok {
			continue
		}
		// **والقيمةُ تُكتب بترميزٍ واحد** — فرقمٌ نصّاً ورقمٌ رقماً
		// لا يُقرآن سواءً.
		enc, _ := json.Marshal(v)
		fmt.Fprintf(&b, "%s=%s;", k, enc)
	}
	return b.String()
}

// Target معرّفُ الهدف من المسار — **للقراءة في التدقيق.**
//
// **والحدُّ الفعليُّ في البصمة** — وهذا اسمٌ يُقرأ لا حارس.
func Target(act Sensitive, path string) string {
	if act.TargetSeg < 0 {
		return ""
	}
	rest := strings.TrimPrefix(path, "/api/v1/admin")
	segs := strings.Split(strings.Trim(rest, "/"), "/")
	if act.TargetSeg >= len(segs) {
		return ""
	}
	return segs[act.TargetSeg]
}

// AdminPattern **يحوّل مساراً حيّاً إلى نمطٍ يطابق الجدولين.**
func AdminPattern(path string) string {
	const prefix = "/api/v1/admin"
	rest := strings.TrimPrefix(path, prefix)
	segs := strings.Split(strings.Trim(rest, "/"), "/")
	out := make([]string, 0, len(segs))
	for _, seg := range segs {
		if seg == "" {
			continue
		}
		if looksLikeID(seg) {
			out = append(out, "{}")
			continue
		}
		out = append(out, seg)
	}
	return "/" + strings.Join(out, "/")
}

// looksLikeID **أهذا المقطعُ معرّفٌ لا اسمُ مورد؟**
//
// **ومقاطعُ الجدول كلُّها كلماتٌ لاتينيّةٌ قصيرةٌ بشرطة** — **والمعرّفُ
// `uuid` أو رقمٌ أو مفتاحُ إعدادٍ فيه نقطة.**
func looksLikeID(s string) bool {
	if s == "" {
		return false
	}
	if strings.Count(s, "-") == 4 && len(s) == 36 {
		return true // uuid
	}
	if strings.ContainsAny(s, ".") {
		return true // مفتاحُ إعدادٍ مثل `merchants.commission_percent`
	}
	for _, r := range s {
		if r < 'a' || r > 'z' {
			if r != '-' {
				return true
			}
		}
	}
	return false
}
