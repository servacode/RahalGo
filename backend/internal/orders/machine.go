package orders

// **آلةُ الحالات مقروءةً بالآلة** — لا جدولاً يُقرأ بالعين وحدَها.
//
// # ولماذا صيغةٌ ثانيةٌ بعد `TruthTable`
//
// **`TruthTable` يُخرج جدولاً بشريّاً** — تسمياتٌ عربيّةٌ ووضعٌ واحدٌ ونوعٌ
// واحد (`truth.go:83` يثبّت `KindStandard`). **فلا يُسأل برنامجٌ عن
// «هل يملك السائقُ `at_dropoff → delivered` في وضع المنصّة؟»** إلّا بقراءةِ
// نصٍّ عربيٍّ وتحليلِه — **وذلك ليس عقداً، ذلك تخمين.**
//
// **ولا شيءَ في المستودع يُخرج قواعدَ الانتقال بصيغةٍ تُقرأ آليّاً** (مُقاسٌ
// ٢٠٢٦-٠٩-٢٩). و`docs/testing/ORDER_TRANSITIONS_55.md` **يزعم في سطره
// الثالث أنّه مولَّد ولا مولِّدَ له** — يحرسه عدُّ صفوفٍ فقط
// (`testtruth/extract.go:116`)، **فيشيخ محتواه صامتاً.** وهذا بعينه ما
// تحذّر منه `truth.go:13`: **«وثيقةٌ تكذب أخطرُ من غياب الوثيقة».**
//
// # وما يُشتَقّ وما يُصرَّح — ولا يُخلطان في سطر
//
// **المُشتَقُّ يُقرأ من الخرائط ذاتِها**: الحالاتُ والحدودُ والأدوارُ الخامُّ
// والأدوارُ الفعليّةُ في كلّ وضعٍ ونهائيّةُ الحالة. **فإن بدّلتَ الشيفرةَ
// تبدّل الخرَج، وأسقط الحارسُ البناءَ إن لم يُعَد التوليد.**
//
// **والمُصرَّحُ سلوكٌ لا تحمله بِنيةُ بيانات**: الحرّاسُ والانتقالاتُ
// التلقائيّة. **وكلُّ مُصرَّحٍ يحمل `source` بملفِّه وسطرِه** ويُعلَّم
// `declared: true` — **فمن قرأ الخرَجَ عرف أيَّ سطرٍ يثق به بلا مراجعة،
// وأيَّ سطرٍ يفتح الملفَّ ليتحقّق.**

import (
	"slices"
	"sort"
)

// MachineStatus حالةٌ واحدةٌ بصفاتها المُشتَقّة.
type MachineStatus struct {
	Code          string `json:"code"`
	Label         string `json:"label"`
	Terminal      bool   `json:"terminal"`
	RefundOnEnter bool   `json:"refund_on_enter"`
	FlowIndex     int    `json:"flow_index"`
}

// MachineEdge حدٌّ واحدٌ — بأدواره الخامّ وأدواره الفعليّة في كلّ وضع.
type MachineEdge struct {
	Kind string `json:"kind"`
	From string `json:"from"`
	To   string `json:"to"`

	// DeclaredRoles **الأدوارُ كما في الخريطة** — قبل أيّ تنقيةٍ بالوضع.
	// **وفارغةٌ تعني «المالكَ وحدَه»** (`statuses.go:182`)، لا «الجميع».
	DeclaredRoles []string `json:"declared_roles"`

	// EffectivePlatform / EffectiveMerchants **من يملكها فعلاً** بعد
	// `rolesUnderMode`. **وفارغةٌ تعني لا أحد** — وهي حالةٌ واقعةٌ لا نظريّة.
	EffectivePlatform  []string `json:"effective_platform"`
	EffectiveMerchants []string `json:"effective_merchants"`

	// Inherited **حدٌّ ورثه النوعُ المخصَّصُ من القياسيّ** لا صُرِّح له.
	Inherited bool `json:"inherited,omitempty"`

	// NobodyInPlatform **لا يملكها أحدٌ في وضع المنصّة** — تُبرَز لأنّها
	// **قيدٌ تشغيليٌّ يُقرأ خطأً أنّه سهو.**
	NobodyInPlatform bool `json:"nobody_in_platform,omitempty"`
}

// MachineDeclared سلوكٌ مُصرَّحٌ بمصدره — **لا يُشتَقّ، فيُراجَع.**
type MachineDeclared struct {
	ID     string `json:"id"`
	What   string `json:"what"`
	Source string `json:"source"`
}

// Machine آلةُ الحالات كاملةً.
type Machine struct {
	Note string `json:"_note"`

	Statuses []MachineStatus `json:"statuses"`
	Terminal []string        `json:"terminal_statuses"`
	Kinds    []string        `json:"kinds"`
	Modes    []string        `json:"modes"`
	Roles    []string        `json:"roles"`

	Edges []MachineEdge `json:"edges"`

	// ModeRules قواعدُ تنقية الأدوار بالوضع — **مُصرَّحةٌ ومُشتَقٌّ أثرُها.**
	ModeRules []MachineDeclared `json:"mode_rules_declared"`
	// Guards الحرّاسُ بترتيب تنفيذهم.
	Guards []MachineDeclared `json:"guards_declared"`
	// AutoRules الانتقالاتُ التي تقع بلا فاعلٍ بشريّ.
	AutoRules []MachineDeclared `json:"auto_rules_declared"`
	// ImplicitEdges حدودٌ تقع ولا تظهر في أيّ خريطة.
	ImplicitEdges []MachineDeclared `json:"implicit_edges_declared"`

	FailReasons []MachineFailReason `json:"fail_reasons"`

	// RequiresReason حالاتٌ لا تُدخَل بلا علّةٍ مكتوبة.
	RequiresReason []string `json:"requires_reason"`
}

// MachineFailReason سببُ تعذّرٍ بذنبه وموضعه.
type MachineFailReason struct {
	Code  string `json:"code"`
	Fault string `json:"fault"`
	At    string `json:"at"`
}

// machineRoles الأدوارُ التي تُسأل — **والمالكُ فيها** (`truth.go:66`).
func machineRoles() []string {
	out := make([]string, 0, len(truthRoles))
	for _, r := range truthRoles {
		out = append(out, r.Code)
	}
	return out
}

// effectiveRoles **من يملك هذا الحدَّ في هذا الوضع** — بسؤال المحرّك.
//
// **ويُسأل عن كلّ دورٍ منفرداً** كما يفعل `TruthTable` (`truth.go:79`):
// **فمن سأل عن الأدوار مجتمعةً خلط من يملكها بمن يُحمَل معه.**
func effectiveRoles(kind, from, to string, selfManage bool) []string {
	out := []string{}
	for _, r := range machineRoles() {
		eff := rolesUnderMode(selfManage, from, to, []string{r}, true)
		if canTransition(kind, from, to, eff) {
			out = append(out, r)
		}
	}
	return out
}

// BuildMachine يبني الآلةَ من الخرائط الحيّة — **ولا يكتب حدّاً بيد.**
func BuildMachine() Machine {
	m := Machine{
		Note: "مولَّدٌ من internal/orders — لا يُحرَّر بيد. " +
			"go run ./cmd/machinedoc. " +
			"الحقولُ المنتهيةُ بـ_declared سلوكٌ مُصرَّحٌ بمصدره، وما عداها مُشتَقٌّ من الشيفرة.",
		Kinds: []string{KindStandard, KindCustom, KindMerchantDelivery},
		Modes: []string{ModePlatform, ModeMerchants},
		Roles: machineRoles(),
	}

	for i, st := range flowOrder {
		m.Statuses = append(m.Statuses, MachineStatus{
			Code:          st,
			Label:         statusLabels[st],
			Terminal:      terminal(st),
			RefundOnEnter: refundOnEnter(st),
			FlowIndex:     i,
		})
	}
	m.Terminal = append([]string{}, TerminalStatuses()...)

	// ── الحدود ───────────────────────────────────────────────────────
	//
	// **وتُمشى بترتيب `flowOrder` لا بترتيب الخريطة** — فخرَجٌ مستقرٌّ
	// **شرطُ أن يكون الفرقُ في `git diff` فرقاً حقيقيّاً**، لا إعادةَ ترتيب.
	for _, kind := range m.Kinds {
		for _, from := range flowOrder {
			ts := transitionsFor(kind, from)
			if len(ts) == 0 {
				continue
			}
			// **ترتيبُ الوجهات بترتيب الانسياب كذلك.**
			sorted := append([]transition{}, ts...)
			sort.SliceStable(sorted, func(i, j int) bool {
				return slices.Index(flowOrder, sorted[i].To) <
					slices.Index(flowOrder, sorted[j].To)
			})
			for _, t := range sorted {
				e := MachineEdge{
					Kind:               kind,
					From:               from,
					To:                 t.To,
					DeclaredRoles:      append([]string{}, t.Roles...),
					EffectivePlatform:  effectiveRoles(kind, from, t.To, false),
					EffectiveMerchants: effectiveRoles(kind, from, t.To, true),
				}
				if e.DeclaredRoles == nil {
					e.DeclaredRoles = []string{}
				}
				// **الموروثُ يُعلَّم** — فمن قرأ الحدَّ المخصَّصَ عرف أصُرِّح
				// له أم ورثه، **وهو الفرقُ الذي يشرح تغيّرَه مع القياسيّ.**
				if kind == KindCustom {
					if _, explicit := customTransitions[from]; !explicit {
						e.Inherited = true
					}
				}
				e.NobodyInPlatform = len(e.EffectivePlatform) == 0
				m.Edges = append(m.Edges, e)
			}
		}
	}

	for _, fr := range FailReasons {
		m.FailReasons = append(m.FailReasons, MachineFailReason{
			Code: fr.Code, Fault: fr.Fault, At: fr.At,
		})
	}

	m.RequiresReason = []string{StRejected, StCancelled, StFailed, StRefunded}

	m.ModeRules = []MachineDeclared{
		{"P1", "وضعُ المنصّة يُسقط دورَ المتجر دائماً", "internal/orders/modes.go:131"},
		{"P2", "الدخولُ إلى preparing يُسقط ops وadmin", "internal/orders/modes.go:144"},
		{"P3", "وجهةٌ في driverOnly تُسقط ops وadmin — **إلّا at_dropoff → failed**: المكتبُ يُنهي عند الباب (مساءَ ٢٠٢٦-١٠-٠٢)", "internal/orders/modes.go:171"},
		{"P4", "**مُبتَلَعةٌ في P3** — شرطُها جزءٌ من شرطِها فلا تُسقط شيئاً جديداً", "internal/orders/modes.go:208"},
		{"P5", "الإلغاءُ بعد التسليم للسائق يُسقط ops لغير المالك", "internal/orders/modes.go:224"},
		{"M0", "وضعُ المتاجر لا يُنقّي شيئاً — يردّ الأدوارَ كما هي", "internal/orders/modes.go:123"},
		{"A0", "المالكُ يملك كلَّ حدٍّ في الخريطة ولا يخترع حدّاً", "internal/orders/statuses.go:330"},
		{"D0", "**driverHolds يُمرَّر ولا يُقرأ** — فالجدولُ دالّةُ (kind, from, to, roles, mode) وحدَها", "internal/orders/modes.go:61"},
	}

	m.Guards = []MachineDeclared{
		{"G0", "قفلُ الصفِّ FOR UPDATE قبل أيّ قراءة", "internal/orders/transitions.go:70"},
		{"G1", "الوضعُ يُقرأ من داخل المعاملة لا قبلها", "internal/orders/transitions.go:106"},
		{"G2", "canTransition — وإلّا invalid_transition (409)", "internal/orders/transitions.go:109"},
		{"G3", "at_pickup وpicked_up يشترطان سائقاً مُسنَداً — driver_required (409)", "internal/orders/transitions.go:115"},
		{"G4", "المخصَّصُ إلى picked_up يشترط custom_agreed_at", "internal/orders/transitions.go:128"},
		{"G5", "ويشترط قفلَ السعر quote_confirmed_version == quote_version", "internal/orders/transitions.go:141"},
		{"G6", "at_pickup → failed **يُحوَّل** إلى merchantBlocked ولا يُنفَّذ", "internal/orders/transitions.go:168"},
		{"G8", "at_dropoff → failed **بلا ذنبٍ مكتوب** يُردّ door_needs_ops (409) — بابُه ResolveDoor وحدَه", "internal/orders/transitions.go:180"},
		{"G7", "نافذةُ إلغاء الزبون — وتُقاس على الأدوار الخامّ فيتجاوزها زبونٌ يحمل ops", "internal/orders/transitions.go:301"},
		{"H1", "الأدمن يحتاج القدرة orders.intervene", "internal/server/server.go:1176"},
		{"H2", "السائقُ إلى delivered يحتاج إثباتاً إن فُعّل drivers.require_delivery_photo", "internal/server/delivery_proof.go:189"},
		{"H3", "قبولُ العرض يشترط ورديّةً مفتوحةً وسقفَ نقدٍ وسقفَ طلبات", "internal/server/driver_handlers.go:614"},
	}

	m.AutoRules = []MachineDeclared{
		{"AUTO-PREPARING", "accepted → preparing — **وضعُ المتاجر وحدَه**، بأدوار الفاعل الأصليّ", "internal/orders/transitions.go:552"},
		{"AUTO-DISPATCH", "preparing → dispatching — وضعُ المتاجر وorders.auto_dispatch، بأدوار ops قسراً", "internal/orders/transitions.go:559"},
		{"AUTO-DISPATCH-MD", "accepted → dispatching لـ«لدي توصيلة» بعد قبول المكتب — orders.auto_dispatch، بأدوار ops قسراً، وفي الوضعين", "internal/orders/transitions.go:591"},
		{"AUTO-DISPATCH-EXPORTED", "AutoDispatch بلا شرطِ وضعٍ ولا علَم — **وهو طريقُ وضع المنصّة**", "internal/orders/transitions.go:568"},
		{"AUTO-ACCEPT", "pending → accepted بعد orders.auto_accept_min (افتراضُه صفرٌ ⇒ مطفأ)", "internal/orders/watchdog.go:257"},
		{"AUTO-TRANSFER", "pending → accepted ثمّ إنزال — orders.auto_transfer (افتراضُه لا)", "internal/server/auto_transfer.go:98"},
		{"AUTO-ASSIGN", "dispatching → assigned بالدور", "internal/orders/rotation.go:541"},
	}

	m.ImplicitEdges = []MachineDeclared{
		{"IMPLICIT-AT-PICKUP-ACCEPTED",
			"at_pickup → accepted — **لا تظهر في خريطةٍ ولا في TRUTH.md**: " +
				"merchantBlocked يردّ الطلبَ مقبولاً ويُفرّغ السائقَ ويُثبّت الذنبَ على المتجر",
			"internal/orders/merchant_blocked.go:46"},
	}

	return m
}
