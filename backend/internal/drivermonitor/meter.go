package drivermonitor

import "math"

// ══════════════════════════════════════════════════════════════════════
// **المسافة اللي مشيتها الدراجة فعلاً** (طلب المالك ٢٠٢٦-١١-١٨: «لازم تكون رقم حقيقي»)
// ══════════════════════════════════════════════════════════════════════
//
// القاعدة القديمة (`RealDrive`) كانت تحسب المقطع بس إذا سرعته ≥ ١٠٫٨ كم/س. **ومختبر الحقيقة**
// (`roadmatch/truth_test.go`، مشاوير ع شوارع الرقة بطول معروف) قاس: الزحمة ٥–١٥ كم/س كانت تطلع
// **−٨٥٪ لـ −٩٠٪**، ويوم توصيل فيه زحمة **−٢٨٪**، والوقوف جوّا محل ٤٥ دقيقة بيطلع **٠٫٤–٠٫٨ كم** رجفة.
//
// **فالحكم صار بالإزاحة لا بالسرعة** (`Moved`) — الخطوة من «المرساة» لنقطة بتنحسب إذا:
//
//  1. **بعيدة**: أبعد من نصف قطر الرجفة max(١٥ م، مجموع دقّة النقطتين).
//  2. **مؤكَّدة**: أول نقطة صالحة بعدها كمان بعيدة عن المرساة — فالنتوء اللي بيروح ويرجع ما بيصير أمتار.
//  3. **والمكان تغيّر فعلاً**: متوسّط النقاط من المرساة لقبل النقطة، ومتوسّط الدقيقتين من النقطة لقدّام،
//     بعاد عن بعض أكتر من max(٥٠ م، ٢ × مجموع متوسّط دقّتهن). الرجفة جوّا محل مترابطة وبتطلع نقطتين
//     بعاد ورا بعض، **بس متوسّطها بيضل بمكانه** — والدراجة الماشية متوسّطها بيمشي معها مهما كانت بطيئة
//     (مختبر الحقيقة: الماشي ≥ ٢٫٨ × العتبة، والرجفة ≤ ١٫٥).
//     **أو القيادة واضحة** (`fast`): القاعدة القديمة (`IsDrive`) والخطوة أبعد من ١٫٥ × مجموع الدقّتين —
//     لفّة حول بلوك متوسّطها ما بيتغيّر وهي قيادة.
//     **أو تكملة** (`cont`): المرساة آخر خطوة انحسبت والدراجة ما وقفت عندها، والخطوة ≤ ٣٠ ث وباتجاهها —
//     آخر المشوار قبل الوقفة (متوسّط ما بعده هو مكان الوقفة نفسه).
//  4. **ورجعة U وهو ماشي** (رايح وراجع بفتحة الجزيرة) بتشبه النتوء: بتنحسب إذا المرساة إجت بخطوة،
//     والقيادة واضحة، والجوال بيقول ماشي بالنقطة اللي بعدها.
//
// وآخر نقطة قبل صمت طويل أو بآخر الأثر ما في شي يأكّدها: بتنحسب إذا القيادة واضحة بس. يعني:
//
//   - **الواقف جوّا محل**: صفر تقريباً. والدقّة أسوأ من ٣٠ م ما بتحرّك المرساة ولا بتدخل بالمتوسّط.
//   - **الزحمة البطيئة والإشارة الحمرا**: كل متر بينحسب مهما كانت السرعة.
//   - **انقطاع الإشارة ≤ ٦ د**: من آخر موضع قبله لأول موضع بعده (خط مستقيم، وع الطريق مع المطابقة)،
//     **ونبضات الواقف بالنص مو مواضع** (`Resent`). وأطول من ٦ د: صفر متل قبل (بيظهر بقصة اليوم).
//   - **الوقفة الطويلة**: رجفتها جوّا نصف القطر ما بتنحسب. وآخر المشوار اللي دخل بنصف قطرها (لحد ٤٠ م
//     قبل مكانها) بينحسب — `Analyze` بيحطّه بتصنيف الطلب لا «واقف».
//   - **المزيَّف**: ما بيحرّك ولا بيتحسب، والمرساة بتبلّش من جديد بعده.
//
// والأمتار بتتوزّع ع مقاطع الخطوة من آخر نقطة جوّا نصف القطر للنقطة، بنسبة طولها — فتصنيف بطلب/بلا
// طلب بيضل لكل مقطع، ورجفة الواقف قبلها ما بتاخد شي. **ومع المطابقة الخطوة كلها ع الشارع**
// (`MetersOnRoad`): موضع النقطة ناقص موضع المرساة ع الطريق المطابق، إذا معقول.

// Moved **المقياس المعتمد** — الإزاحة عن المرساة (فوق).
type Moved struct{ Cfg Config }

// movedEps حصّة مقطع طوله صفر بالخطوة — بس ليبيّن «قيادة» (فبياخد أمتار الطريق مع المطابقة).
const movedEps = 1e-6

// RoadDriveMeter مقياس بيعرف الطريق المطابق: `between(i, j)` أمتار الطريق من pts[i] لـpts[j] إذا الاتنين
// ع نفس القطعة المطابقة (`Road.Segment`)، وإلا false. `Analyze` بيناديه بدل `Meters` لما في طريق.
type RoadDriveMeter interface {
	MetersOnRoad(pts []Point, between func(i, j int) (float64, bool)) []float64
}

// عتبات معقوليّة الخطوة ع الطريق: أمتار الطريق للخطوة لازم تكون بين RoadStepMinFactor × الخط − RoadStepMinSlackM
// و RoadStepMaxFactor × الخط + RoadStepMaxExtraM — وإلا المطابق غلط (لفّة ع رجفة، أو موضع علق) ⇒ الخط
// المستقيم. **فالطريق ما بينزل عن الخط المستقيم بلا سبب** — بس الضجيج سبب: نتوء ٤٠ م بيطوّل الخط
// والطريق لأ (حدّ ٠٫٨ كان بيطلّع الضجيج +٧٪ بمختبر الحقيقة، ٠٫٥ بيطلّعه +٤٪).
const (
	RoadStepMaxFactor = 2.0
	RoadStepMaxExtraM = 50.0
	RoadStepMinFactor = 0.5
	RoadStepMinSlackM = 50.0
)

// Meters انظر DriveMeter — بلا طريق: الخط المستقيم من المرساة للنقطة.
func (m Moved) Meters(pts []Point) []float64 { return m.MetersOnRoad(pts, nil) }

// MetersOnRoad انظر RoadDriveMeter. **الخطوة ع الطريق**: أمتار الطريق من المرساة للنقطة إذا كلها
// مطابقة ومعقولة؛ وإلا من آخر نقطة بنصف القطر (رجفة الواقف قبلها ممكن تلفّ ع الشوارع)؛ وإلا الخط.
func (m Moved) MetersOnRoad(pts []Point, between func(i, j int) (float64, bool)) []float64 {
	cfg := m.Cfg
	if cfg.GapAfter == 0 {
		cfg = DefaultConfig()
	}
	n := len(pts)
	out := make([]float64, n)
	if n < 2 {
		return out
	}
	usable := func(k int) bool {
		p := pts[k]
		return !p.Mocked && (p.AccuracyM == nil || *p.AccuracyM <= cfg.DriveMaxAccuracyM) && !Resent(pts, k)
	}
	acc := func(p Point) float64 {
		if p.AccuracyM == nil {
			return cfg.MoveNilAccM
		}
		return *p.AccuracyM
	}
	radius := func(a, b Point) float64 {
		return max(cfg.MoveMinM, cfg.MoveAccFactor*(acc(a)+acc(b)))
	}
	// broken بين k-1 و k ما بينحسب شي (صمت طويل أو مزيَّف).
	broken := func(k int) bool {
		return pts[k].At.Sub(pts[k-1].At) > cfg.GapAfter || pts[k].Mocked || pts[k-1].Mocked
	}
	// confirmed النقطة k بعيدة عن المرساة a: أول نقطة صالحة بعدها (بلا انقطاع) كمان بعيدة؟ و`last`
	// = ما في نقطة بعدها تأكّد (آخر الأثر، أو صمت طويل بعدها).
	confirmed := func(a, k int) (ok, last bool, next int) {
		for q := k + 1; q < n; q++ {
			if broken(q) {
				return false, true, -1
			}
			if !usable(q) {
				continue
			}
			return Dist(pts[a], pts[q]) >= radius(pts[a], pts[q]), false, q
		}
		return false, true, -1
	}
	// mean متوسّط المواضع الصالحة من `from` لـ`to` (ضمناً، بأيّ اتجاه) ومتوسّط دقّتها — لحد
	// `MoveMeanWindow` من `from` وبلا قطع.
	mean := func(from, to int) (Point, float64) {
		dir := 1
		if to < from {
			dir = -1
		}
		var la, ln, ac float64
		c := 0
		for j := from; ; j += dir {
			if j != from {
				if d := pts[j].At.Sub(pts[from].At); d > cfg.MoveMeanWindow || -d > cfg.MoveMeanWindow {
					break
				}
				if broken(max(j, j-dir)) {
					break
				}
			}
			if usable(j) {
				la += pts[j].Lat
				ln += pts[j].Lng
				ac += acc(pts[j])
				c++
			}
			if j == to {
				break
			}
		}
		if c == 0 {
			return pts[from], acc(pts[from])
		}
		f := float64(c)
		return Point{Lat: la / f, Lng: ln / f}, ac / f
	}
	// moved **المكان تغيّر**: متوسّط النقاط من المرساة لقبل النقطة (موضع المرساة كما شافتها النقاط
	// اللي بعدها)، ومتوسّط الدقيقتين من النقطة لقدّام.
	moved := func(a, k int) bool {
		pa, aa := mean(k-1, a)
		pk, ak := mean(k, n-1)
		return Dist(pa, pk) >= max(cfg.MoveMeanM, cfg.MoveMeanAccFactor*(aa+ak))
	}
	// fast **قيادة واضحة** ولو المتوسّط ما تغيّر (لفّة حول بلوك أو دوّار): القاعدة القديمة (`IsDrive`)
	// والخطوة أبعد من `MoveFastAccFactor` × مجموع الدقّتين — رجفة بدقّة ٣٠ م ما بتنطّ هالقدّ.
	fast := func(a, p Point, d float64) bool {
		return IsDrive(a, p, cfg) && d >= cfg.MoveFastAccFactor*(acc(a)+acc(p))
	}
	// last آخر نقطة صالحة ضمن نصف القطر من المرساة: الخطوة بتتوزّع ع المقاطع من بعدها بس — فرجفة
	// الواقف قبلها ما بتاخد شي.
	// cont **تكملة**: المرساة نفسها آخر خطوة انحسبت والدراجة ما وقفت عندها (ولا نقطة جوّا نصف القطر
	// بعدها)، والخطوة الجاية ضمن `MoveContinueMax` وباتجاهها (±٦٠°) — آخر المشوار قبل الوقفة: المتوسّط
	// بعد النقطة هو مكان الوقفة نفسه، فكان يضيع آخر ~٥٠ م من كل مشوار.
	stepFrom := -1 // من وين إجت آخر خطوة انحسبت لـ`anchor` (-1 = ما إجت بخطوة)
	cont := func(anchor, last, k int) bool {
		if stepFrom < 0 || last != anchor || pts[k].At.Sub(pts[anchor].At) > cfg.MoveContinueMax {
			return false
		}
		a, b, c := pts[stepFrom], pts[anchor], pts[k]
		ux, uy := b.Lng-a.Lng, b.Lat-a.Lat
		vx, vy := c.Lng-b.Lng, c.Lat-b.Lat
		n := math.Hypot(ux, uy) * math.Hypot(vx, vy)
		return n > 0 && (ux*vx+uy*vy)/n >= 0.5
	}
	anchor, last := -1, -1
	setAnchor := func(k int) { anchor, last, stepFrom = k, k, -1 }
	for k := 0; k < n; k++ {
		if k > 0 && broken(k) {
			anchor, last = -1, -1
		}
		p := pts[k]
		if !usable(k) {
			continue
		}
		if anchor < 0 {
			setAnchor(k)
			continue
		}
		a := pts[anchor]
		d := Dist(a, p)
		if d < radius(a, p) {
			last = k
			continue
		}
		// **وآخر نقطة قبل صمت أو بآخر الأثر** ما في شي يأكّدها: بتنحسب إذا القيادة واضحة (`fast`) بس.
		// **ورجعة U وهو ماشي** (رايح ٤٠ م وراجع — فتحة بالجزيرة) بتشبه النتوء: بتنحسب إذا المرساة
		// إجت بخطوة، والقيادة واضحة، والجوال بيقول ماشي بالنقطة اللي بعدها كمان.
		ok, end, next := confirmed(anchor, k)
		uturn := !ok && next >= 0 && stepFrom >= 0 && fast(a, p, d) &&
			pts[next].SpeedMps != nil && *pts[next].SpeedMps >= cfg.DriveMinPhoneSpeedMps
		switch {
		case end:
			if !fast(a, p, d) {
				continue
			}
		case uturn:
		case !ok || !(moved(anchor, k) || fast(a, p, d) || cont(anchor, last, k)):
			continue
		}
		if between != nil {
			d = stepOnRoad(between, anchor, last, k, d)
		}
		// الخطوة: d متر ع المقاطع (last, k] بنسبة طولها.
		sum := 0.0
		for j := last + 1; j <= k; j++ {
			sum += Dist(pts[j-1], pts[j])
		}
		for j := last + 1; j <= k; j++ {
			share := movedEps
			if sum > 0 {
				share = math.Max(movedEps, d*Dist(pts[j-1], pts[j])/sum)
			}
			out[j] += share
		}
		from := anchor
		setAnchor(k)
		stepFrom = from
	}
	return out
}

// stepOnRoad أمتار الخطوة ع الطريق (انظر `MetersOnRoad`)، أو `chord`.
func stepOnRoad(between func(i, j int) (float64, bool), anchor, last, k int, chord float64) float64 {
	sane := func(v float64) bool {
		return v <= RoadStepMaxFactor*chord+RoadStepMaxExtraM && v >= RoadStepMinFactor*chord-RoadStepMinSlackM
	}
	if v, ok := between(anchor, k); ok && sane(v) {
		return v
	}
	if last != anchor {
		if v, ok := between(last, k); ok && sane(v) {
			return v
		}
	}
	return chord
}

// Resent **نبضة الواقف** (`LocationService.kt`): التطبيق بيعيد آخر موضع بلا دقّة ولا سرعة والخادم بيحطّ
// وقته — بالانقطاع بتضل لحد ٥ دقايق. **مو موضع** ⇒ ما بتحرّك المرساة ولا بتدخل بالمتوسّط، فالانقطاع
// بينحسب من آخر موضع حقيقي لأول موضع بعده (لا من «واقف» مصنوع).
func Resent(pts []Point, k int) bool {
	p := pts[k]
	if k == 0 || p.AccuracyM != nil || p.SpeedMps != nil {
		return false
	}
	q := pts[k-1]
	return p.Lat == q.Lat && p.Lng == q.Lng
}

// DropResent الأثر بلا نبضات الواقف (`Resent`) — للمطابقة ع الطريق: Valhalla بيشوف المواضع الحقيقية بس،
// فالانقطاع بيتطابق من آخر موضع قبله لأول موضع بعده.
func DropResent(pts []Point) []Point {
	out := make([]Point, 0, len(pts))
	for k := range pts {
		if !Resent(pts, k) {
			out = append(out, pts[k])
		}
	}
	return out
}
