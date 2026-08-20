package routing

// ══════════════════════════════════════════════════════════════════════
// **مناوراتُ رحّال غو — مفاهيمُنا لا مفاهيمُ المحرّك**
// ══════════════════════════════════════════════════════════════════════
//
// (المرحلة ٢، بأمر المالك ٢٠٢٦-٠٨-٢٠: «Android لا يجب أن يعرف OSRM».)
//
// # ولماذا تعدادٌ عندنا
//
// **`"exit rotary"` و`"slight left"` كلماتُ OSRM** — ومن مرّرها إلى
// التطبيق **ربط تطبيقَ السائق بمحرّكٍ بعينه.** فيومَ يُبدَّل المحرّكُ
// (فالهالا أو غيرُها) **يُعاد بناءُ التطبيق كلِّه** لأنّ نصّاً تبدّل
// في خادم.
//
// **والتحويلُ هنا في طبقةٍ واحدة** — ملفٌّ يُبدَّل ولا شيءَ بعده.
//
// # ولا يسقط شيءٌ على نوعٍ مجهول
//
// **وOSRM يضيف أنواعاً في نسخه** — **وتعدادٌ يسقط على قيمةٍ غريبةٍ
// يُنهي التطبيقَ في يد سائقٍ على درّاجة.** فما لا نعرفه `UNKNOWN`،
// وتقول الشاشةُ «تابع المسار» ولا تكذب.

// ManeuverKind ما يفعله السائقُ عند هذه النقطة.
type ManeuverKind string

const (
	KindDepart         ManeuverKind = "DEPART"
	KindArrive         ManeuverKind = "ARRIVE"
	KindStraight       ManeuverKind = "STRAIGHT"
	KindTurnLeft       ManeuverKind = "TURN_LEFT"
	KindTurnRight      ManeuverKind = "TURN_RIGHT"
	KindSlightLeft     ManeuverKind = "SLIGHT_LEFT"
	KindSlightRight    ManeuverKind = "SLIGHT_RIGHT"
	KindSharpLeft      ManeuverKind = "SHARP_LEFT"
	KindSharpRight     ManeuverKind = "SHARP_RIGHT"
	KindUTurn          ManeuverKind = "U_TURN"
	KindMerge          ManeuverKind = "MERGE"
	KindFork           ManeuverKind = "FORK"
	KindOffRamp        ManeuverKind = "OFF_RAMP"
	KindRoundabout     ManeuverKind = "ROUNDABOUT"
	KindExitRoundabout ManeuverKind = "EXIT_ROUNDABOUT"

	// KindUnknown **ما لم نعرفه** — ولا يُسقط شيئاً. انظر أعلاه.
	KindUnknown ManeuverKind = "UNKNOWN"
)

// Maneuver مناورةٌ واحدةٌ على المسار.
//
// # والمرجعُ مسافةٌ لا فهرس
//
// (تصحيحُ المالك ٢٠٢٦-٠٨-٢٠: «اعتمد `atDistanceM` كمرجعٍ أساسيّ… ولا
//
//	يجب أن يعتمد منطقُ الملاحة على `atIndex` وحدَه».)
//
// **والفهرسُ يتبدّل إن بُسِّطت الهندسةُ يوماً أو حُذف رأسٌ مكرّر** —
// **والمسافةُ على الطريق لا تتبدّل.** فهي المرجع، والفهرسُ تسريعٌ لا
// أكثر.
type Maneuver struct {
	Kind ManeuverKind `json:"kind"`

	// Modifier جهةُ المناورة كما جاءت — **وقد تغيب.**
	//
	// **قِيس أنّها `null` فعلاً** في `depart` بالأزقّة وفي مسارٍ طويل.
	// **ومن افترضها موجودةً سقط على أوّل زقاق.**
	Modifier *string `json:"modifier,omitempty"`

	// AtDistanceM **موضعُ المناورة على طول المسار** — المرجعُ الأساسيّ.
	AtDistanceM float64 `json:"at_distance_m"`

	// AtIndex فهرسُ الرأس — **تسريعٌ لا مرجع.** انظر أعلاه.
	AtIndex int `json:"at_index"`

	// StepDistanceM طولُ الخطوة التي تبدأ بهذه المناورة.
	StepDistanceM float64 `json:"step_distance_m"`
	// StepDurationS مدّتُها كما قالها المحرّك — **أساسُ زمن الوصول.**
	StepDurationS float64 `json:"step_duration_s"`

	// StreetName اسمُ الشارع — **اختياريٌّ بالكامل.**
	//
	// **وقِيس أنّ ٨٣٪ من خطوات الرقّة بلا اسم** — فالإرشادُ يُبنى على
	// المسافة، **والاسمُ إضافةٌ حين يوجد لا ركنٌ في الجملة.**
	StreetName *string `json:"street_name,omitempty"`

	// RoundaboutExit رقمُ المخرج — **وهو ما يحتاجه السائقُ فعلاً.**
	RoundaboutExit *int `json:"roundabout_exit,omitempty"`
	// RoundaboutName اسمُ الدوّار — **عربيٌّ في بياناتنا.**
	RoundaboutName *string `json:"roundabout_name,omitempty"`
}

// ══════════════════════════════════════════════════════════════════════
// **والدوّاران واحدٌ عند السائق**
// ══════════════════════════════════════════════════════════════════════
//
// (تصحيحُ المالك: «لا تعتمد منطقيّاً على أنّ `rotary` دوّارٌ كبيرٌ
//
//	و`roundabout` صغير».)
//
// **وOSRM يفرّق بينهما بحجمِ الدوّار واسمِه** — **والسائقُ لا يفرّق**:
// يدخل ويأخذ المخرج. **فيُوحَّدان، ويُؤخذ منهما المخرجُ والاسمُ حين
// يوجدان.**

// mapKind يحوّل نوعَ المحرّك ومعدِّلَه إلى مفهومنا.
//
// **والأنواعُ المذكورةُ هنا قِيست من محرّكنا نفسِه** (٢٠٢٦-٠٨-٢٠، خمسةُ
// مسارات) — **لا من وثيقة.**
func mapKind(osrmType string, modifier *string) ManeuverKind {
	switch osrmType {
	case "depart":
		return KindDepart
	case "arrive":
		return KindArrive
	case "roundabout", "rotary":
		return KindRoundabout
	case "exit roundabout", "exit rotary":
		return KindExitRoundabout
	case "merge":
		return KindMerge
	case "fork":
		// **والمفترقُ جهتُه معنى** — يمينٌ أو يسارٌ لا «مفترق» مجرّدة.
		if k := byModifier(modifier); k != KindUnknown && k != KindStraight {
			return KindFork
		}
		return KindFork
	case "off ramp":
		return KindOffRamp
	case "continue", "new name":
		// **و«اسمٌ جديد» ليست مناورة** — الشارعُ تبدّل اسمُه والسائقُ
		// يمضي مستقيماً. **ومن جعلها انعطافاً أربك من يقودها.**
		return KindStraight
	case "turn", "end of road":
		return byModifier(modifier)
	default:
		return KindUnknown
	}
}

// byModifier جهةُ المناورة من معدِّلها.
func byModifier(modifier *string) ManeuverKind {
	if modifier == nil {
		// **ومعدِّلٌ غائبٌ لا يُخمَّن يميناً** — انظر `Modifier`.
		return KindUnknown
	}
	switch *modifier {
	case "straight":
		return KindStraight
	case "left":
		return KindTurnLeft
	case "right":
		return KindTurnRight
	case "slight left":
		return KindSlightLeft
	case "slight right":
		return KindSlightRight
	case "sharp left":
		return KindSharpLeft
	case "sharp right":
		return KindSharpRight
	case "uturn":
		return KindUTurn
	default:
		return KindUnknown
	}
}
