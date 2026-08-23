package routing

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strconv"
)

// ══════════════════════════════════════════════════════════════════════
// **مجموعةُ المسارات — موصًى بها وبدائلُ اختياريّة**
// ══════════════════════════════════════════════════════════════════════
//
// (المرحلة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
//
// # **ولا «الأسرع» ولا «الأقصر»**
//
// **أمرُ المالك نصّاً**: «في Phase 7: لا FASTEST ولا SHORTEST».
//
// **والسببُ مقيس**: `weight_name` في محرّكنا **`routability` لا
// `duration`** — وقِيس الفرقُ فعليّاً: `weight ≠ duration` في **٧٫٢٪
// من المسارات** (١٦ من ٢٢١، ٢٠٢٦-٠٨-٢١). **فمن سمّى الأساسَ «الأسرع»
// وصف ما لم يُحسَب.**
//
// **وكذلك «الأقصر»**: جُرّب `exclude=motorway` كبديلٍ فأطال المسار
// (+٢٢٫٨كم الرقّة←دمشق، +٤٩كم حلب←دمشق). **والأقصرُ الحقيقيُّ يحتاج
// محرّكاً ثانياً بوزن المسافة** — وذلك مؤجَّلٌ إلى المرحلة ٨.
//
// **فالوصفُ بالأرقام لا بالصفات**: «أقصرُ بـ٤٫٥كم · أطولُ زمناً
// بـ١٢د».

// RouteSet ما يُعرض للسائق — **موصًى بها وبدائلُ نجت من الترشيح.**
type RouteSet struct {
	// Primary **هي `routes[0]` من المحرّك ولا تُرقّى غيرُها.**
	//
	// **أمرُ المالك نصّاً**: «حتى لو routes[1] كانت أقصر + أقل
	// duration، لا ترقيها تلقائيًا إلى Primary».
	//
	// **ووقع ذلك مرّتين في القياس** (٢٠٢٦-٠٨-٢١): بديلٌ أقصرُ بـ٣٧كم
	// وأسرعُ بـ٧د. **ومع ذلك يبقى `routes[0]` هو الموصى به** —
	// فالمحرّكُ يوازن ما لا نراه في رقمين.
	Primary *Route `json:"primary"`

	// Alternatives **مرتَّبةٌ بترتيب توصية المحرّك** (البند ١٣).
	Alternatives []*Route `json:"alternatives"`

	// Diagnostics **ما يُسجَّل ولا يُعرض** — البند ٢ و٣٢.
	Diagnostics SetDiagnostics `json:"diagnostics"`
}

// SetDiagnostics **أثرُ القرار** — ليُفهم لماذا نجا ما نجا.
type SetDiagnostics struct {
	// EngineReturned كم ردَّ المحرّك.
	EngineReturned int `json:"engine_returned"`
	// AfterValidation كم نجا من فحص السلامة.
	AfterValidation int `json:"after_validation"`
	// AfterFilter كم نجا من الترشيح.
	AfterFilter int `json:"after_filter"`
	// Shown كم عُرض بعد السقف.
	Shown int `json:"shown"`

	// PrimaryOutranked **الأساسُ ليس الأفضلَ في المحورين.**
	//
	// **يُسجَّل ولا يُبدَّل** — أمرُ المالك: «سجل هذه الحالة في
	// metadata/QA لأنها مهمة، لكن لا تتجاوز recommendation rank
	// الخاص بالمحرك».
	PrimaryOutranked bool `json:"primary_outranked"`

	// Dropped أسبابُ الحذف — للتشخيص.
	Dropped []string `json:"dropped,omitempty"`
}

// ══════════════════════════════════════════════════════════════════════
// **العتباتُ — مشتقّةٌ من قياسٍ لا من رأي**
// ══════════════════════════════════════════════════════════════════════
//
// **مصدرُها**: `testdata/syria-alternatives.golden.json` — ١٣٤ طلباً
// و٨٧ بديلاً في خمس محافظاتٍ وبين اثنتَي عشرةَ مدينة (٢٠٢٦-٠٨-٢١،
// OSRM v26.8.0).
const (
	// SimilarityStepM **يُعاد تقطيعُ المسار كلَّ ٢٥م.**
	//
	// **فالتقطيعُ لا يؤثّر** — قِيس: نصفُ الرؤوس يعطي ٩٨٫٦٪ تشابهاً
	// مع الأصل، وربعُها ٩٦٫٧٪.
	SimilarityStepM = 25.0

	// SimilarityToleranceM **مسافةُ «على المسار نفسِه».**
	//
	// **قِيست**: إزاحةُ ٢٠م تعطي ٩٩٪، وإزاحةُ ٤٠م تعطي ٣٠٫٧٪.
	SimilarityToleranceM = 30.0

	// NearDuplicateShared **شبهُ المكرَّر — يُحذف.**
	//
	// **قِيس التوزيع**: من ٨٧ بديلاً، **اثنان فقط تشابهُهما ≥٨٠٪**،
	// ولا واحدَ ≥٩٠٪. **فالعتبةُ تحذف ما هو مكرَّرٌ فعلاً ولا تمسّ
	// غيرَه.**
	NearDuplicateShared = 0.80

	// GrossPenaltyRatio **العقوبةُ الفاحشة — زمناً أو مسافة.**
	//
	// **قِيست ثلاثُ سياسات** (البند ١٢):
	//
	//	أ · شبهُ المكرَّر وحدَه      تغطية ٤١٫٨٪ · أقصى مسافة +٥٢٫٣٪
	//	ج · + زمنٌ ومسافةٌ ≤ +٥٠٪    تغطية ٤١٫٠٪ · أقصى مسافة +٤٣٫٢٪
	//	د · + زمنٌ ومسافةٌ ≤ +٢٥٪    تغطية ٣٣٫٦٪ ← تُسقط ١٨ بديلاً
	//
	// **فـ«ج» تكلّف ٠٫٨ نقطةِ تغطيةٍ وتحذف بديلاً واحداً** أطولَ
	// بـ٥٢٪ مسافةً (٤كم التفافاً في رحلةِ ٧٫٥كم). **و«د» تُسقط
	// الخُمسَ بلا مقابل.**
	//
	// **وحدُّ الزمن لم يُطلَق في المجموعة كلِّها** — أقصى عقوبةٍ
	// مقيسةٍ +٣٢٫٥٪. **وهو حارسٌ لبياناتٍ قادمةٍ لا لهذه.**
	GrossPenaltyRatio = 0.50

	// MaxAlternatives **سقفُ العرض** — البند ٢٢.
	MaxAlternatives = 2

	// EngineMaxAlternatives **ما يُطلب من المحرّك.**
	//
	// **و`alternatives=5` يردّ `TooBig`** — سقفُ OSRM ثلاثة (قِيس).
	EngineMaxAlternatives = 3
)

// ══════════════════════════════════════════════════════════════════════
// **الهويّةُ الثابتة — لا فهرسُ مصفوفة**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ١٤.)
//
// **`alternatives[0]` قد يصير `alternatives[1]` في الطلب التالي** —
// فالسائقُ يضغط ما ظنّه الأوّلَ فيُختار غيرُه.
//
// **والمعرِّفُ من الهندسة والسياق**: هندسةٌ مكمَّمةٌ إلى ١e-5 (≈١م)
// **مع الوجهة ومعرِّف الطلب.** فمسارٌ إلى الاستلام ومسارٌ إلى التسليم
// بالهندسة نفسِها **معرِّفاهما مختلفان** — وهو ما يحسم سباقَ الوجهة
// (البند ٢١).
//
// **والجيلُ ليس فيه** — أمرُ المالك: «routeGeneration لا تدخل في
// routeId». فلو دخل **لتبدّل معرِّفُ المسار نفسِه بلا أن يتبدّل
// المسار.** والجيلُ في `RouteSet` والتحقّقُ عنده.

// RouteID **معرِّفٌ ثابتٌ لمسارٍ في سياقه.**
func RouteID(r *Route, orderID, target string) string {
	if r == nil {
		return ""
	}
	h := sha256.New()
	h.Write([]byte(orderID))
	h.Write([]byte{0})
	h.Write([]byte(target))
	h.Write([]byte{0})
	for _, p := range r.Geometry {
		h.Write([]byte(strconv.FormatInt(int64(math.Round(p.Lat*1e5)), 10)))
		h.Write([]byte{','})
		h.Write([]byte(strconv.FormatInt(int64(math.Round(p.Lng*1e5)), 10)))
		h.Write([]byte{';'})
	}
	h.Write([]byte(strconv.FormatInt(int64(math.Round(r.DistanceM)), 10)))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// ══════════════════════════════════════════════════════════════════════
// **التشابهُ بالإسقاط — لا بمطابقة الرؤوس**
// ══════════════════════════════════════════════════════════════════════
//
// (البندان ٩ و٣١.)
//
// **أمرُ المالك نصّاً**: «لا أريد مقارنة vertex index حرفيًا لأن
// geometries قد تختلف في sampling».
//
// **والفرقُ ليس نظريّاً**: مسارٌ وآخرُ على الطريق نفسِه قد يكون
// لأحدهما ضعفُ نقاط الآخر، **فمقارنةُ الفهارس تعلن اختلافاً وهو
// طريقٌ واحد.**
//
// # **وهذه ليست البصمة**
//
// **`RouteFingerprint` هويّةٌ** — تطابقٌ أو لا. **وهذا تدرّج.**
// ولا يُخلطان: **الأولى تجيب «أهو المسارُ نفسُه؟»، والثاني «كم
// يشتركان؟»**

// Similarity **نسبةُ `b` التي تقع على `a`** — بين صفرٍ وواحد.
//
// **غيرُ متماثلة**: قصيرٌ داخلَ طويلٍ يعطي نسبةً عاليةً في اتّجاهٍ
// ومنخفضةً في الآخر. **ومن أراد التماثلَ أخذ الأدنى.**
func Similarity(a, b []Point) float64 {
	if len(a) < 2 || len(b) < 2 {
		return 0
	}
	idx := buildIndex(a)
	samples := resample(b, SimilarityStepM)
	if len(samples) == 0 {
		return 0
	}
	on := 0
	for _, p := range samples {
		if onRoute(idx, a, p) {
			on++
		}
	}
	return float64(on) / float64(len(samples))
}

// SharedRatio **الأدنى في الاتّجاهين** — فلا يُخدع بمسارٍ قصير.
func SharedRatio(a, b []Point) float64 {
	return math.Min(Similarity(a, b), Similarity(b, a))
}

// resample **عيّناتٌ متساويةُ المسافة** — فالتقطيعُ لا يؤثّر.
func resample(g []Point, step float64) []Point {
	if len(g) < 2 || step <= 0 {
		return g
	}
	out := []Point{g[0]}
	carry := 0.0
	for i := 1; i < len(g); i++ {
		a, b := g[i-1], g[i]
		d := MetersBetween(a, b)
		if d <= 0 {
			continue
		}
		for t := step - carry; t <= d; t += step {
			f := t / d
			out = append(out, Point{
				Lat: a.Lat + (b.Lat-a.Lat)*f,
				Lng: a.Lng + (b.Lng-a.Lng)*f,
			})
		}
		carry = math.Mod(carry+d, step)
	}
	return append(out, g[len(g)-1])
}

// **شبكةٌ مكانيّةٌ خشنة** — فالمقارنةُ خطّيّةٌ لا تربيعيّة.
//
// **مسارُ الرقّة←دمشق أربعةُ آلاف نقطة**، ومقارنتُه بآخرَ بلا شبكةٍ
// **ستّةَ عشرَ مليونَ عمليّة.**
const indexCellDeg = 0.005

type segIndex map[[2]int][]int

// ══════════════════════════════════════════════════════════════════════
// **والقطعةُ تُسجَّل في كلّ خليّةٍ تعبرها — لا عند طرفيها**
// ══════════════════════════════════════════════════════════════════════
//
// (إغلاقُ صحّة ٧، ٢٠٢٦-٠٨-٢١.)
//
// # **العيبُ الذي أُصلح**
//
// **كانت تُسجَّل عند الطرفين وجوارِهما** — وذلك يكفي حين تكون الرؤوسُ
// متقاربة. **ورؤوسُ الطرق السريعة متباعدة**: قطعةٌ واحدةٌ قد تمتدّ
// كيلومترات بلا رأسٍ بينهما.
//
// **فنقطةٌ في وسط تلك القطعة تقع في خليّةٍ لم يُسجَّل فيها شيء** —
// **فتُعلَن «خارجَ المسار» وهي عليه تماماً.**
//
// **وقِيس الأثر**: مساران متطابقان بايتاً ببايت (٤٣٢٬٣٩٦م لكليهما)
// **أعطيا تشابهاً ٠٫٩٠٦ و٠٫٨٤٩** بدل ١٫٠٠٠.
//
// **وذلك يفسد ثلاثةَ أشياء**: كشفَ شبه المكرَّر (فيمرّ مكرَّر)،
// **ونقطةَ التفرّع (فتُعلَن مبكّرةً كاذبة)**، وكلَّ قرارٍ مبنيٍّ
// عليهما.
//
// **فتُمشى الخلايا على طول القطعة** بخطوةٍ نصفِ خليّة — **فلا تفوت
// خليّةٌ تعبرها.**
func buildIndex(g []Point) segIndex {
	idx := make(segIndex, len(g))

	add := func(lat, lng float64, i int) {
		key := [2]int{int(math.Floor(lat / indexCellDeg)), int(math.Floor(lng / indexCellDeg))}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				k := [2]int{key[0] + dy, key[1] + dx}
				list := idx[k]
				if len(list) > 0 && list[len(list)-1] == i {
					continue
				}
				idx[k] = append(list, i)
			}
		}
	}

	for i := 0; i < len(g)-1; i++ {
		a, b := g[i], g[i+1]
		add(a.Lat, a.Lng, i)
		add(b.Lat, b.Lng, i)

		// **وما بينهما** — بخطوةٍ نصفِ خليّةٍ فلا تُقفَز واحدة.
		span := math.Max(math.Abs(b.Lat-a.Lat), math.Abs(b.Lng-a.Lng))
		steps := int(span/(indexCellDeg/2)) + 1
		if steps <= 1 {
			continue
		}
		// **وسقفٌ يمنع انفجاراً** — قطعةٌ بعرض درجةٍ كاملةٍ لا تقع في
		// بياناتِ طرقٍ، **والسقفُ حارسٌ لا سياسة.**
		if steps > 4096 {
			steps = 4096
		}
		for s := 1; s < steps; s++ {
			f := float64(s) / float64(steps)
			add(a.Lat+(b.Lat-a.Lat)*f, a.Lng+(b.Lng-a.Lng)*f, i)
		}
	}
	return idx
}

func onRoute(idx segIndex, g []Point, p Point) bool {
	key := [2]int{int(math.Floor(p.Lat / indexCellDeg)), int(math.Floor(p.Lng / indexCellDeg))}
	for _, i := range idx[key] {
		if i+1 >= len(g) {
			continue
		}
		if segmentDistanceM(p, g[i], g[i+1]) <= SimilarityToleranceM {
			return true
		}
	}
	return false
}

// segmentDistanceM **بُعدُ نقطةٍ عن قطعة** — بتقريبٍ مستوٍ يكفي لعشرات الأمتار.
func segmentDistanceM(p, a, b Point) float64 {
	kx := math.Cos(p.Lat*math.Pi/180) * 111320.0
	ky := 110540.0
	px, py := p.Lng*kx, p.Lat*ky
	ax, ay := a.Lng*kx, a.Lat*ky
	bx, by := b.Lng*kx, b.Lat*ky
	dx, dy := bx-ax, by-ay
	if dx == 0 && dy == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	t := ((px-ax)*dx + (py-ay)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}

// ══════════════════════════════════════════════════════════════════════
// **الافتراقُ ذو المعنى — أوّلُ قرارٍ حقيقيّ**
// ══════════════════════════════════════════════════════════════════════
//
// (إغلاقُ صحّة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ٦ إلى ١٠.)
//
// **أمرُ المالك نصّاً**: «نريد انتهاء صلاحية البديل عند تجاوز FIRST
// MEANINGFUL DECISION DIVERGENCE على المسار المختار/Recommended».
//
// # **ولماذا الأوّلُ لا الأخير**
//
// **مثالُ المالك**: مساران يشتركان حتّى ١٢٥م، ثمّ يفترقان، ثمّ
// يلتقيان عند ٦٠٠، ثمّ يفترقان ثانيةً عند ٩٠٠.
//
// **فمن تجاوز الفرعَ الأوّل لم يعد البديلُ خياراً له** — وقرارُه
// الأوّلُ اتُّخذ فعلاً. **ولو بقي معروضاً حتّى التسعمئة لأعطينا
// السائقَ مساراً يحاول إرجاعَه إلى فرعٍ فات.**
//
// # **ولماذا لا تكفي أوّلُ عيّنةٍ تتجاوز التسامح**
//
// **أمرُ المالك**: «لا تستخدم أول Sample تتجاوز 30m فقط».
//
// **فعيّنةٌ واحدةٌ شاردةٌ ليست قراراً**: جسرٌ، أو طريقُ خدمةٍ موازٍ،
// أو فرقُ تمثيلٍ في تقاطع. **والقرارُ يبقى قراراً مسافةً.**
//
// # **والعتبةُ مقيسةٌ لا مفترضة**
//
// **قِيست فتراتُ الافتراق على ثماني عيّناتٍ حقيقيّة** (٢٠٢٦-٠٨-٢١،
// OSRM v26.8.0) — **عشرُ فتراتٍ**:
//
//	أدنى طولٍ مقيس        ٩٩٩م
//	ربع                  ١٬٨٣٥م
//	وسيط                ٢١٬٦٧٥م
//	أقصى               ٢٢٠٬٧٥٨م
//
// **ولا فترةَ واحدةً دونَ ٩٩٩م.** **وضجيجُ العيّنة عيّنةٌ أو اثنتان**
// — أي ٢٥م إلى ٥٠م بخطوة `SimilarityStepM`.
//
// **فمئةُ مترٍ تفصل بينهما بأمان**: **عشرُ مرّاتٍ دونَ أصغر فترةٍ
// حقيقيّة، وضِعفا أطولِ ضجيجٍ ممكن.**

// MeaningfulDivergenceM **العتبةُ** — فترةٌ أقصرُ منها ضجيجٌ لا قرار.
const MeaningfulDivergenceM = 100.0

// DivergenceInterval **فترةُ افتراقٍ واحدة** — بالمسافة على المسار
// الذي يقوده السائق.
type DivergenceInterval struct {
	StartM float64
	EndM   float64
}

// LengthM **طولُ الفترة.**
func (d DivergenceInterval) LengthM() float64 { return d.EndM - d.StartM }

// DivergenceIntervals **كلُّ فترات الافتراق** — مقيسةً على `a`.
//
// **والاتّجاهُ مهمّ**: يُمشى على `a` — **وهو المسارُ الذي يقوده
// السائق** — ويُسأل «أما زلتُ على `b`؟». **ولو قِيس على `b` لزادت
// المسافةُ بقدر انحرافه** عن طريقٍ لم يسلكه.
func DivergenceIntervals(a, b []Point) []DivergenceInterval {
	if len(a) < 2 || len(b) < 2 {
		return nil
	}
	samples := resample(a, SimilarityStepM)
	if len(samples) < 2 {
		return nil
	}
	idx := buildIndex(b)

	dist := make([]float64, len(samples))
	for i := 1; i < len(samples); i++ {
		dist[i] = dist[i-1] + MetersBetween(samples[i-1], samples[i])
	}

	var out []DivergenceInterval
	start := -1.0
	for i, p := range samples {
		on := onRoute(idx, b, p)
		switch {
		case !on && start < 0:
			start = dist[i]
		case on && start >= 0:
			out = append(out, DivergenceInterval{StartM: start, EndM: dist[i]})
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, DivergenceInterval{StartM: start, EndM: dist[len(dist)-1]})
	}
	return out
}

// MeaningfulDivergenceM **بدايةُ أوّل فترةٍ ذاتِ معنى** — أو سالبٌ إن
// لم توجد.
//
// **الأولى لا الأخيرة** (البند ٧)، **وذاتُ المعنى لا أوّلُ شاردة**
// (البند ٨).
func MeaningfulDivergence(a, b []Point) float64 {
	for _, iv := range DivergenceIntervals(a, b) {
		if iv.LengthM() >= MeaningfulDivergenceM {
			return iv.StartM
		}
	}
	return -1
}

// FirstDivergenceM **أوّلُ افتراقٍ أيّاً كان طولُه** — للتشخيص.
//
// **ولا يُبنى عليه قرارُ تقادم** — `MeaningfulDivergence` هي التي
// تُبنى عليها. **وبقي لأنّ الفرقَ بينهما يُقرأ في التشخيص**: افتراقٌ
// عند ١٢٠م طولُه ٣٠م **يُرى ولا يُبطل.**
func FirstDivergenceM(a, b []Point) float64 {
	ivs := DivergenceIntervals(a, b)
	if len(ivs) == 0 {
		return -1
	}
	return ivs[0].StartM
}

// ══════════════════════════════════════════════════════════════════════
// **الترشيح — سياسةُ «ج» المعايَرة**
// ══════════════════════════════════════════════════════════════════════
//
// # **والهيمنةُ إشارةُ جودةٍ لا سببَ حذف** (البندان ٧ و١١)
//
// **أمرُ المالك نصّاً**: «المسافة والزمن لا يمثلان كل معرفة الطريق،
// والسائق قد يستفيد من Route مختلفة بوضوح إذا كان يعرف إغلاقًا أو
// مشكلةً محليةً غير موجودة في البيانات».
//
// **والقياسُ يسنده**: من ٥٦ بديلاً مهيمَناً عليه، **٤٤ تشابهُها دونَ
// ٥٠٪** — طرقٌ مختلفةٌ حقّاً. **وحذفُها يُنزل التغطيةَ من ٤١٪ إلى
// ١٦٪** ويرمي أكثرَ القيمة.

// FilterReason **لماذا حُذف بديل** — نصٌّ للتشخيص لا للعرض.
type FilterReason string

const (
	ReasonNearDuplicate FilterReason = "near_duplicate"
	ReasonGrossTime     FilterReason = "gross_time_penalty"
	ReasonGrossDistance FilterReason = "gross_distance_penalty"
	ReasonInvalid       FilterReason = "invalid"
	ReasonOverCap       FilterReason = "over_cap"
)

// FilterAlternatives **يُرشّح البدائلَ ويحافظ على ترتيب المحرّك.**
//
// **أمرُ المالك** (البند ١٣): «حافظ قدر الإمكان على OSRM
// recommendation order. ولا تعيد ترتيب Routes حسب duration وحدها أو
// distance وحدها».
func FilterAlternatives(primary *Route, alts []*Route) ([]*Route, []string) {
	if primary == nil {
		return nil, nil
	}
	kept := make([]*Route, 0, len(alts))
	dropped := make([]string, 0, len(alts))

	for i, a := range alts {
		if reason := rejectReason(primary, a); reason != "" {
			dropped = append(dropped, string(reason))
			continue
		}
		// **والمقارنةُ مع ما نجا أيضاً** — فبديلان متشابهان بينهما
		// **لا يُعرضان معاً** ولو اختلف كلٌّ منهما عن الأساس.
		dup := false
		for _, k := range kept {
			if SharedRatio(k.Geometry, a.Geometry) >= NearDuplicateShared {
				dup = true
				break
			}
		}
		if dup {
			dropped = append(dropped, string(ReasonNearDuplicate))
			continue
		}
		if len(kept) >= MaxAlternatives {
			dropped = append(dropped, string(ReasonOverCap))
			continue
		}
		_ = i
		kept = append(kept, a)
	}
	return kept, dropped
}

func rejectReason(primary, a *Route) FilterReason {
	if a == nil || len(a.Geometry) < 2 || a.DistanceM <= 0 || a.DurationS <= 0 {
		return ReasonInvalid
	}
	if SharedRatio(primary.Geometry, a.Geometry) >= NearDuplicateShared {
		return ReasonNearDuplicate
	}
	// **العقوبةُ الفاحشة** — حارسٌ ضدّ العبث، لا ترشيحٌ للأفضل.
	if primary.DurationS > 0 &&
		(a.DurationS-primary.DurationS)/primary.DurationS > GrossPenaltyRatio {
		return ReasonGrossTime
	}
	if primary.DistanceM > 0 &&
		(a.DistanceM-primary.DistanceM)/primary.DistanceM > GrossPenaltyRatio {
		return ReasonGrossDistance
	}
	return ""
}

// PrimaryOutranked **أثمّة بديلٌ أفضلُ من الأساس في المحورين؟**
//
// **يُسجَّل ولا يُغيّر شيئاً** (البندان ٢ و٣٢) — وقع مرّتين في ١٣٤
// طلباً مقيساً.
func PrimaryOutranked(primary *Route, alts []*Route) bool {
	if primary == nil {
		return false
	}
	for _, a := range alts {
		if a != nil && a.DistanceM < primary.DistanceM && a.DurationS < primary.DurationS {
			return true
		}
	}
	return false
}
