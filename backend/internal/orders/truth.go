package orders

// **جدولُ الحقيقة** — يُقرأ من المحرّك لا يُكتب بيد.
//
// # لماذا يُولَّد
//
// وثيقةٌ تُكتب بيدٍ **تشيخ في أوّل تعديل**: تُغيَّر قاعدةٌ في الشيفرة ويبقى
// السطرُ في الوثيقة يقول القديم. **وقد وقع فعلاً** (٢٠٢٦-٠٨-٠٢): كُتب في
// `POLICY-OPERATIONS.md` استثناءٌ **واحدٌ ضيّقٌ** للمالك، **ونُفِّذ في الشيفرة
// تجاوزاً شاملاً** — فبقيت ثلاثةُ أزرارٍ ظاهرةً له، **والوثيقةُ تقول إنّها
// نُزعت.**
//
// **ووثيقةٌ تكذب أخطرُ من غياب الوثيقة**: غيابُها يدفعك إلى قراءة الشيفرة،
// **وكذبُها يجعلك تبني على ما ليس.**
//
// فما يصف **سلوكَ المحرّك** يُولَّد منه، وما يصف **قرارَ المالك** يُكتب بيده
// نصّاً. **ولا يُخلط الاثنان في سطر.**

import (
	"fmt"
	"sort"
	"strings"
)

// TruthRole دورٌ يُسأل عنه في الجدول.
type TruthRole struct {
	Code  string
	Label string
}

var truthRoles = []TruthRole{
	{"customer", "الزبون"},
	{"merchant", "المتجر"},
	{"ops", "العمليات"},
	{"admin", "المالك"},
	{"driver", "السائق"},
}

// statusLabels تسمياتُ الحالات — **بلفظها في الشاشة لا برمزها.**
var statusLabels = map[string]string{
	StPending:     "بانتظار القبول",
	StAccepted:    "مقبول",
	StPreparing:   "قيد التحضير",
	StDispatching: "في الطابور",
	StAssigned:    "أُسند لسائق",
	StAtPickup:    "السائق عند المتجر",
	StPickedUp:    "استلم السائق",
	StOnTheWay:    "في الطريق",
	StAtDropoff:   "عند الزبون",
	StDelivered:   "سُلّم",
	StRejected:    "مرفوض",
	StCancelled:   "ملغى",
	StFailed:      "تعذّر التسليم",
	StRefunded:    "مُسترَدّ",
}

// flowOrder ترتيبُ الحالات كما تقع — لا أبجدياً.
var flowOrder = []string{
	StPending, StAccepted, StPreparing, StDispatching, StAssigned,
	StAtPickup, StPickedUp, StOnTheWay, StAtDropoff, StDelivered,
	StRejected, StCancelled, StFailed, StRefunded,
}

// TruthTable جدولُ «من يملك ماذا» في وضعٍ بعينه — **مقروءاً من المحرّك.**
//
// **ولا يُستثنى المالك من السؤال**: كان استثناؤه هو الخلل، **فيُسأل كما يُسأل
// غيرُه ويظهر جوابُه في الجدول.**
func TruthTable(selfManage bool) string {
	var b strings.Builder
	b.WriteString("| من | إلى | من يملكها |\n|---|---|---|\n")

	for _, from := range flowOrder {
		outs := allowedTransitions[from]
		if len(outs) == 0 {
			continue
		}
		for _, tr := range outs {
			owners := []string{}
			for _, role := range truthRoles {
				// **السائقُ المُسنَد يغيّر الجواب** في بعض الانتقالات، فيُسأل
				// عن الحالين ويُذكر القيدُ إن اختلفا.
				eff := rolesUnderMode(selfManage, from, tr.To, []string{role.Code}, true)
				if canTransition(from, tr.To, eff) {
					owners = append(owners, role.Label)
				}
			}
			who := strings.Join(owners, " · ")
			if who == "" {
				// **ولا يُحذف الصفُّ الفارغ**: «لا أحد» خبرٌ، **وغيابُ السطر
				// يُقرأ «لم يُفكَّر فيه».**
				who = "**لا أحد**"
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				label(from), label(tr.To), who))
		}
	}
	return b.String()
}

func label(st string) string {
	if l, ok := statusLabels[st]; ok {
		return fmt.Sprintf("%s `%s`", l, st)
	}
	return "`" + st + "`"
}

// FailReasonsTable أسبابُ التعذّر وذنوبُها — **مقروءةً من القائمة.**
//
// **والذنبُ يقرّر التعويض**، فجدولٌ يُكتب بيدٍ يخالفها يجعل من يقرأ الوثيقةَ
// يحسب تعويضاً غيرَ الذي يقع.
func FailReasonsTable() string {
	faultLabel := map[string]string{
		FaultCustomer: "الزبون", FaultDriver: "السائق",
		FaultMerchant: "المتجر", FaultPlatform: "المنصة",
	}
	var b strings.Builder
	b.WriteString("| السبب | الذنب على | يُعرض عند |\n|---|---|---|\n")
	for _, r := range FailReasons {
		b.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n",
			r.Code, faultLabel[r.Fault], label(r.At)))
	}
	return b.String()
}

// TerminalStates الحالاتُ التي تُغلق الطلب — **مقروءةً من `terminal()`.**
func TerminalStates() string {
	out := []string{}
	for _, st := range flowOrder {
		if terminal(st) {
			out = append(out, label(st))
		}
	}
	sort.Strings(out)
	return strings.Join(out, " · ")
}
