package routing

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **الافتراقُ ذو المعنى — البند ١١، الحالاتُ السبع**
// ══════════════════════════════════════════════════════════════════════
//
// (إغلاقُ صحّة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١.)

// branch **يفترق عند `atM` ويبقى مفترقاً إلى `untilM`.**
//
// **والإزاحةُ ٢٠٠م** — أوسعُ من تسامح الثلاثين بكثير.
func branch(lengthM, atM, untilM, offsetM, stepM float64) []Point {
	var out []Point
	for d := 0.0; d <= lengthM; d += stepM {
		lat := lat0
		if d >= atM && d < untilM {
			lat = north(offsetM)
		}
		out = append(out, Point{Lat: lat, Lng: east(d)})
	}
	return out
}

// twoBranches **يفترق مرّتين بينهما التقاء** — مثالُ المالك في البند ٧.
func twoBranches(lengthM, at1, until1, at2, until2, offsetM, stepM float64) []Point {
	var out []Point
	for d := 0.0; d <= lengthM; d += stepM {
		lat := lat0
		if (d >= at1 && d < until1) || (d >= at2 && d < until2) {
			lat = north(offsetM)
		}
		out = append(out, Point{Lat: lat, Lng: east(d)})
	}
	return out
}

// ── أ · مشتركان ٥٠٠م ثمّ يفترقان ────────────────────────────────────

func TestDivergence_A_SharedThenSplit(t *testing.T) {
	base := straightEast(3000, 20)
	alt := branch(3000, 500, 3000, 200, 20)

	got := MeaningfulDivergence(base, alt)
	if got < 450 || got > 560 {
		t.Errorf("أ: القرارُ عند %.0fم — والمنتظَرُ نحو ٥٠٠", got)
	}
}

// ── ب · ضجيجُ ٢٥م ثمّ يعودان ───────────────────────────────────────

func TestDivergence_B_NoiseIsNotDecision(t *testing.T) {
	// **أمرُ المالك**: «noise لمدة 25m ثم يعودان → لا يعتبر decision
	// divergence».
	//
	// **وقِيس أنّ أقصرَ فترةٍ حقيقيّةٍ ٩٩٩م** — فضجيجُ ٢٥م دونها
	// بأربعين ضعفاً.
	base := straightEast(3000, 20)
	alt := branch(3000, 1000, 1025, 200, 20)

	if got := MeaningfulDivergence(base, alt); got >= 0 {
		t.Errorf("ب: ضجيجُ ٢٥م عُدَّ قراراً عند %.0fم", got)
	}
	// **ولا فترةَ أصلاً في هذه الرفيدة**: نتوءُ خمسةٍ وعشرين متراً
	// **يصل بقطعتين مائلتين تمرّان بجوار المسار**، فيبقى الموصى به
	// داخلَ تسامح البديل. **والرؤيةُ في التشخيص تُقاس في `B2`** حيث
	// الفترةُ ظاهرةٌ ودونَ العتبة.
}

func TestDivergence_B2_JustUnderThresholdIsNoise(t *testing.T) {
	base := straightEast(4000, 20)
	// **تسعون متراً** — دونَ المئة.
	shortIv := branch(4000, 1000, 1090, 200, 20)
	if got := MeaningfulDivergence(base, shortIv); got >= 0 {
		t.Errorf("فترةُ ٩٠م عُدَّت قراراً عند %.0fم", got)
	}
	// **ومئةٌ وخمسون فوقها** — فتُعدّ.
	longIv := branch(4000, 1000, 1150, 200, 20)
	if got := MeaningfulDivergence(base, longIv); got < 900 || got > 1100 {
		t.Errorf("فترةُ ١٥٠م: القرارُ عند %.0fم — والمنتظَرُ نحو ١٠٠٠", got)
	}
}

// ── ج · يفترقان ١٢٥م ثمّ يلتقيان ثمّ يفترقان ٩٠٠ ────────────────────

func TestDivergence_C_FirstNotLast(t *testing.T) {
	// **مثالُ المالك بعينه** (البند ٧).
	base := straightEast(3000, 20)
	alt := twoBranches(3000, 125, 600, 900, 3000, 200, 20)

	ivs := DivergenceIntervals(base, alt)
	if len(ivs) != 2 {
		t.Fatalf("ج: %d فترةً والمنتظَرُ اثنتان: %+v", len(ivs), ivs)
	}
	got := MeaningfulDivergence(base, alt)
	if got < 100 || got > 180 {
		t.Errorf("ج: القرارُ عند %.0fم — والمنتظَرُ نحو ١٢٥ لا ٩٠٠", got)
	}
	// **ولو أُخذت الأخيرةُ لكان ٩٠٠** — وهو ما رُفض.
	if got > 500 {
		t.Error("ج: أُخذت الفترةُ الأخيرة — والمطلوبُ الأولى")
	}
}

// ── د · الهندسةُ نفسُها بتقطيعٍ مختلف ───────────────────────────────

func TestDivergence_D_SamplingIsNotDivergence(t *testing.T) {
	dense := straightEast(3000, 10)
	sparse := straightEast(3000, 250)
	if got := MeaningfulDivergence(dense, sparse); got >= 0 {
		t.Errorf("د: تقطيعٌ مختلفٌ عُدَّ قراراً عند %.0fم", got)
	}
	if got := MeaningfulDivergence(sparse, dense); got >= 0 {
		t.Errorf("د: وبالعكس %.0fم", got)
	}
}

// ── هـ · طريقان متوازيان من البداية ────────────────────────────────

func TestDivergence_E_ParallelFromStart(t *testing.T) {
	base := straightEast(3000, 20)
	parallel := offsetNorth(base, 60)
	got := MeaningfulDivergence(base, parallel)
	if got < 0 || got > 60 {
		t.Errorf("هـ: متوازيان بستّين متراً — القرارُ عند %.0fم والمنتظَرُ نحو الصفر", got)
	}
}

func TestDivergence_E2_ParallelWithinToleranceIsShared(t *testing.T) {
	// **وعشرون متراً داخلَ التسامح** — الطريقُ نفسُه بتمثيلٍ مختلف.
	base := straightEast(3000, 20)
	if got := MeaningfulDivergence(base, offsetNorth(base, 20)); got >= 0 {
		t.Errorf("هـ٢: إزاحةُ ٢٠م عُدَّت قراراً عند %.0fم", got)
	}
}

// ── و · دوّار ──────────────────────────────────────────────────────

func TestDivergence_F_RoundaboutBranches(t *testing.T) {
	// **دوّارٌ نصفُ قطره ٣٠م عند ألفِ متر**، ثمّ مخرجان مختلفان.
	//
	// **فالقرارُ عند المخرج لا عند دخول الدوّار** — والدوّارُ نفسُه
	// مشتركٌ بينهما.
	base := make([]Point, 0, 200)
	alt := make([]Point, 0, 200)
	for d := 0.0; d <= 1000; d += 20 {
		p := Point{Lat: lat0, Lng: east(d)}
		base = append(base, p)
		alt = append(alt, p)
	}
	// **الدوّارُ نفسُه** — قوسٌ مشترك.
	for a := 0; a < 8; a++ {
		p := Point{Lat: north(15 * float64(a) / 8), Lng: east(1000 + 4*float64(a))}
		base = append(base, p)
		alt = append(alt, p)
	}
	// **ثمّ مخرجان** — الأساسُ شرقاً والبديلُ شمالاً.
	for d := 0.0; d <= 2000; d += 20 {
		base = append(base, Point{Lat: north(15), Lng: east(1032 + d)})
		alt = append(alt, Point{Lat: north(15 + d), Lng: east(1032)})
	}

	got := MeaningfulDivergence(base, alt)
	if got < 950 || got > 1200 {
		t.Errorf("و: القرارُ عند %.0fم — والمنتظَرُ عند مخرج الدوّار نحو ١٠٣٠", got)
	}
}

// ── ز · جسرٌ أو طريقُ خدمةٍ قصير ────────────────────────────────────

func TestDivergence_G_ShortBridgeIsNotDecision(t *testing.T) {
	// **أمرُ المالك**: «bridge/service road temporary geometry
	// difference → لا false expiry إن كان الاختلاف قصيرًا».
	base := straightEast(4000, 20)
	// **جسرٌ بإزاحةِ ٤٠م لثمانين متراً** — خارجَ التسامح وداخلَ العتبة.
	bridge := branch(4000, 1500, 1580, 40, 20)
	if got := MeaningfulDivergence(base, bridge); got >= 0 {
		t.Errorf("ز: جسرٌ ٨٠م عُدَّ قراراً عند %.0fم", got)
	}
}

// ── ولا افتراقَ أصلاً ──────────────────────────────────────────────

func TestDivergence_Identical(t *testing.T) {
	g := straightEast(3000, 20)
	if got := MeaningfulDivergence(g, g); got >= 0 {
		t.Errorf("مسارٌ مع نفسِه: %.0f", got)
	}
	if ivs := DivergenceIntervals(g, g); len(ivs) != 0 {
		t.Errorf("مسارٌ مع نفسِه: %d فترة", len(ivs))
	}
}

// ── والفتراتُ تُقاس على من يقودها ─────────────────────────────────

func TestDivergenceMeasuredOnRecommended(t *testing.T) {
	// **البند ٩** — «distance/progress to divergence يجب أن تكون على
	// المسار الذي يقوده السائق، لا على Alternative».
	//
	// **والمقصودُ أنّ الرقمَ يُقارَن بتقدّم السائق** — وتقدّمُه على
	// الموصى به. **فالقيمةُ يجب أن تساوي موضعَ الفرع على الموصى به.**
	//
	// **وحين يشتركان في البداية يستوي الاتّجاهان** — الفرعُ عند
	// المسافة نفسِها على كليهما. **والفرقُ يظهر حين يختلف الطول**،
	// وهو ما يقيسه الاختبارُ الثاني.
	base := straightEast(3000, 20)
	alt := branch(3000, 500, 3000, 200, 20)

	onBase := MeaningfulDivergence(base, alt)
	if onBase < 450 || onBase > 620 {
		t.Errorf("القرارُ عند %.0fم — والفرعُ عند ٥٠٠ على الموصى به", onBase)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ولماذا لا اختبارَ لتباين الاتّجاهين**
// ══════════════════════════════════════════════════════════════════════
//
// **جُرّب ثلاثَ مرّاتٍ وسقط ثلاثاً** (٢٠٢٦-٠٨-٢١): ادّعيتُ أنّ القياسَ
// على البديل يعطي رقماً أكبرَ من القياس على الموصى به. **والبياناتُ
// لم تسنده**: ٥٥٠ و٥٥٠، ثمّ ٢٢٥ و٢٢١.
//
// **والسببُ أنّ المسارين يشتركان في البداية دائماً** — كلاهما من موضع
// السائق. **فموضعُ الفرع الأوّل هو المسافةُ نفسُها على كليهما**، ولا
// يُغيّرها ما بعد الفرع.
//
// **والاتّجاهُ يبقى مهمّاً لموضعٍ آخر**: نهاياتُ الفترات ومواضعُ
// الالتقاء **تختلف** — وهي تُقاس على `a` بالبناء. **لكنّ ما يُبنى
// عليه التقادمُ هو البدايةُ**، وهي متساوية.
//
// **فلا يُختبر ما لا يصحّ.**
