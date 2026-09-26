// Package androidmap مصفوفتا أندرويد — **الأجهزةُ والاختبارات.**
//
// **وهنا لا في `mobile/`** — **لأنّ `P-9` و`P-10` يقرآن من هنا**، ومصفوفةٌ
// في شجرةٍ ثانيةٍ بلغةٍ ثانيةٍ تحتاج أداةً ثانيةً تقرؤها.
//
// # والفصلُ الذي طلبه المالك (البند ٥٠)
//
//	HARNESS IMPLEMENTED      ← ما بُني ويُشغَّل
//	DEVICE MATRIX EXECUTED   ← ما شُغّل على جهازٍ حقيقيّ
//
// **وجهازٌ غيرُ متوفّرٍ ليس `PASS`** (البند ٤٤).
package androidmap

import "sort"

// DeviceStatus حالُ الجهاز.
type DeviceStatus string

const (
	Available    DeviceStatus = "AVAILABLE"
	NotAvailable DeviceStatus = "NOT_AVAILABLE"
)

// RunStatus حالُ التشغيل على الجهاز.
type RunStatus string

const (
	Executed    RunStatus = "EXECUTED"
	PartialRun  RunStatus = "PARTIAL"
	NotExecuted RunStatus = "NOT_EXECUTED"
)

// Device صنفُ جهازٍ في مصفوفة القبول.
type Device struct {
	ID string `json:"id"`
	// Class فئتُه في `TQ-2`.
	Class string `json:"class"`
	// Model الطرازُ المعتمَد — فارغٌ إن لم يُسمَّ بعد.
	Model string `json:"model,omitempty"`
	// Android إصدارُه.
	Android string `json:"android,omitempty"`
	// Primary أهو جهازُ القبول المرجعيّ.
	Primary bool `json:"primary"`
	// Status متوفّرٌ أم لا — **مقيسٌ بـ`adb devices`.**
	Status DeviceStatus `json:"status"`
	// Run ما شُغّل عليه.
	Run RunStatus `json:"run"`
	// Why لماذا يلزم هذا الصنف.
	Why string `json:"why"`
	// Evidence سطرُ الدليل.
	Evidence string `json:"evidence"`
}

// Devices مصفوفةُ القبول — **`TQ-2`.**
var Devices = []Device{
	{
		ID: "DEV-01", Class: "SAMSUNG BASELINE",
		Model: "Samsung SM-A525F", Android: "14", Primary: true,
		Status: NotAvailable, Run: NotExecuted,
		Why:      "**جهازُ القبول المرجعيّ** — وقرارُ المالك يُثبته ولا يُبدَّل",
		Evidence: "adb devices ⇒ لا جهازَ متّصل (٢٠٢٦-٠٩-٠٥)",
	},
	{
		ID: "DEV-02", Class: "XIAOMI/REDMI RESTRICTIVE BACKGROUND",
		Status: NotAvailable, Run: NotExecuted,
		Why:      "**أقسى قيودِ خلفيّةٍ في السوق** — تقتل الخدماتِ اللاصقةَ بلا إنذار",
		Evidence: "لا جهازَ من هذا الصنف",
	},
	{
		ID: "DEV-03", Class: "LOW-COST INFINIX/TECNO",
		Status: NotAvailable, Run: NotExecuted,
		Why: "**شريحةُ سائقي الرقّة الأوسع** — ذاكرةٌ صغيرةٌ وموتُ عمليّةٍ متكرّر. " +
			"**و`Infinix X6833B` لا يُعاد اعتمادُه تلقائيّاً** (قرارُ المالك)",
		Evidence: "لا جهازَ من هذا الصنف · ولا اعتمادَ تلقائيّ",
	},
	{
		ID: "DEV-04", Class: "OLDER ANDROID",
		Status: NotAvailable, Run: NotExecuted,
		Why:      "**أذوناتُ الخلفيّةِ والموقعِ تختلف قبل `Android 12`**",
		Evidence: "لا جهازَ من هذا الصنف",
	},
	{
		ID: "DEV-05", Class: "HUAWEI (بلا خدمات جوجل)",
		Status: NotAvailable, Run: NotExecuted,
		Why:      "**لا FCM ولا خرائطَ جوجل** — والدفعُ والموقعُ مسارٌ آخرُ كلّيّاً",
		Evidence: "لا جهازَ · ويدخل حين يلزم",
	},
}

// Kind نوعُ الاختبار — **والفصلُ بينها هو صدقُ التقرير.**
type Kind string

const (
	// Structural يُثبت بنيةَ المسار من مصدره — **بلا جهاز.**
	Structural Kind = "STRUCTURAL"
	// Unit اختبارُ JVM على منطقٍ صِرف.
	Unit Kind = "UNIT"
	// Replay إعادةُ أثرٍ حتميّة.
	Replay Kind = "REPLAY"
	// Device يحتاج جهازاً حقيقيّاً.
	OnDevice Kind = "DEVICE"
	// Drive يحتاج قيادةً في الشارع.
	Drive Kind = "REAL_WORLD_DRIVE"
)

// Automation درجةُ الأتمتة (البند ٤٣).
type Automation string

const (
	Automated Automation = "AUTOMATED"
	SemiAuto  Automation = "SEMI_AUTOMATED"
	Manual    Automation = "MANUAL_EVIDENCE_REQUIRED"
)

// Result نتيجةٌ مقيسة.
type Result string

const (
	Pass          Result = "PASS"
	ExpectedFail  Result = "EXPECTED_FAIL"
	RiskConfirmed Result = "RISK_CONFIRMED"
	NotRun        Result = "NOT_RUN"
	Partial       Result = "PARTIAL"
)

// Case حالةُ اختبارٍ في أندرويد.
type Case struct {
	ID   string `json:"id"`
	App  string `json:"app"`
	Area string `json:"area"`
	// Title ما يُثبته.
	Title string `json:"title"`
	// Kind نوعُه.
	Kind Kind `json:"kind"`
	// Automation درجتُه.
	Automation Automation `json:"automation"`
	// Registers ما يمسّه من السجلّات.
	Registers []string `json:"registers,omitempty"`
	// Test اسمُ الاختبار حيث وُجد.
	Test string `json:"test,omitempty"`
	// Result ما قِيس.
	Result Result `json:"result"`
	// Evidence سطرُ الدليل — **ولا نتيجةَ بلا دليل.**
	Evidence string `json:"evidence"`
	// Needs ما يلزم لتشغيله إن لم يُشغَّل.
	Needs string `json:"needs,omitempty"`
}

// Cases الحالاتُ المعرَّفة.
//
// **وكلُّ ما نوعُه `DEVICE` أو `REAL_WORLD_DRIVE` نتيجتُه `NOT_RUN`
// اليوم** — **ولا جهازَ متّصل.**
var Cases = []Case{
	// ── السائق · طبقةُ الموقع — **مُثبَتةٌ بنيويّاً** ─────────────
	{ID: "AND-01", App: "driver", Area: "location-queue",
		Title: "الطابورُ يُمحى كاملاً بعد إرسال ما قُرئ",
		Kind:  Structural, Automation: Automated, Registers: []string{"R19"},
		Test: "LocationContractTest.R19", Result: RiskConfirmed,
		Evidence: "all() ثمّ sendBatch() ثمّ clear() — والملفُّ يُحذف كاملاً"},
	{ID: "AND-02", App: "driver", Area: "location-queue",
		Title: "نجاحُ الدفعة يمسح المرفوضَ مع المقبول",
		Kind:  Structural, Automation: Automated, Registers: []string{"R20"},
		Test: "LocationContractTest.R20", Result: RiskConfirmed,
		Evidence: "لا تُقرأ حمولةُ الردّ لتمييز نقطةٍ رُفضت"},
	{ID: "AND-03", App: "driver", Area: "location-queue",
		Title: "طابورُ النقاط بلا صاحبٍ ولا يُمسح عند الخروج",
		Kind:  Structural, Automation: Automated, Registers: []string{"D16"},
		Test: "LocationContractTest.D16", Result: ExpectedFail,
		Evidence: "points.jsonl ملفٌّ واحدٌ بلا معرّفِ سائق · ولا مسارَ خروجٍ يمسحه"},
	{ID: "AND-04", App: "driver", Area: "location-service",
		Title: "الإقلاعُ اللاصق يفقد الفاصلَ المضبوط",
		Kind:  Structural, Automation: Automated, Registers: []string{"D18"},
		Test: "LocationContractTest.D18", Result: ExpectedFail,
		Evidence: "الفاصلُ يُقرأ من intent وحدَه · وأندرويد يُسلّم null عند الإقلاع اللاصق"},
	{ID: "AND-05", App: "driver", Area: "location-service",
		Title: "لا إعادةَ تحقّقٍ من الورديّة بعد الإقلاع اللاصق",
		Kind:  Structural, Automation: Automated, Registers: []string{"R17"},
		Test: "LocationContractTest.R17", Result: RiskConfirmed,
		Evidence: "onStartCommand تُقلع وتطلب الموقعَ بلا سؤالٍ عن ورديّةٍ أو جلسةٍ أو حال"},

	// ── ما يحتاج جهازاً ────────────────────────────────────────────
	{ID: "AND-10", App: "driver", Area: "shift",
		Title: "فتحُ الورديّة وإغلاقُها ونقرٌ متكرّرٌ وانقطاعُ شبكة",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},
	{ID: "AND-11", App: "driver", Area: "background-location",
		Title: "الخدمةُ حيّةٌ والشاشةُ مطفأةٌ والتطبيقُ في الخلفيّة",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},
	{ID: "AND-12", App: "driver", Area: "doze",
		Title: "الغفوةُ وموفّرُ البطّاريّة — أتستمرّ المواقع؟",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F + adb dumpsys deviceidle"},
	{ID: "AND-13", App: "driver", Area: "gps",
		Title: "GPS مطفأٌ أو ضعيفٌ لا يُنهي رحلةً",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},
	{ID: "AND-14", App: "driver", Area: "process-death",
		Title: "موتُ العمليّة أثناء ملاحةٍ نشطة",
		Kind:  OnDevice, Automation: SemiAuto, Registers: []string{"D17"},
		Result:   NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F + adb shell am force-stop"},
	{ID: "AND-15", App: "driver", Area: "reboot",
		Title: "إعادةُ الإقلاع وحالُ الخدمة بعدها",
		Kind:  OnDevice, Automation: Manual, Registers: []string{"R17"},
		Result:   NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},
	{ID: "AND-16", App: "driver", Area: "offline-maps",
		Title: "تنزيلُ الحزمة واستئنافُها وقطعُ الشبكة",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F + مساحةٌ كافية"},
	{ID: "AND-17", App: "driver", Area: "navigation",
		Title: "الملاحةُ الحيّة — مطابقةُ الخريطة والانحرافُ وإعادةُ المسار",
		Kind:  Drive, Automation: Manual, Result: NotRun,
		Evidence: "يحتاج قيادةً في الشارع", Needs: "REAL-WORLD DRIVE"},
	{ID: "AND-18", App: "driver", Area: "push",
		Title: "صوتُ طلبٍ جديدٍ وقناتُه",
		Kind:  OnDevice, Automation: Manual, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},

	{ID: "AND-20", App: "customer", Area: "process-death",
		Title: "السلّةُ والعنوانُ والطلبُ النشطُ بعد موتِ العمليّة",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},
	{ID: "AND-21", App: "customer", Area: "deep-link",
		Title: "النقرُ يفتح الكيانَ بعينه — باردٌ ودافئٌ وخلفيّة",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل · **وعقدُ الخادم مُثبَتٌ في `P-7`**",
		Needs:    "SM-A525F"},
	{ID: "AND-22", App: "customer", Area: "multi-order",
		Title: "طلبان لا يختلطان على الجهاز",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل · **والخادمُ مُثبَتٌ في `P-7`**", Needs: "SM-A525F"},

	{ID: "AND-30", App: "merchant", Area: "process-death",
		Title: "الطلبُ النشطُ يعود بعد موتِ العمليّة",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},
	{ID: "AND-31", App: "merchant", Area: "realtime",
		Title: "الوصلةُ تعود بعد انقطاعٍ وانتهاءِ رمز",
		Kind:  OnDevice, Automation: SemiAuto, Registers: []string{"D19"},
		Result:   NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},
	// ── جرسُ الطلب الجديد — **حالةُ متجرٍ خاصّةٌ لا AND-18 السائق** (B2) ──
	//
	// **وليست AND-18**: تلك صوتُ إشعارِ السائق وقناتُه (رنّةٌ واحدة). **وهذه
	// إنذارٌ داخليٌّ متكرّرٌ** يعيشه صاحبُ المطعم ويداه في العجين: يرنّ ويهتزّ
	// كلَّ ثوانٍ ما دام طلبٌ في `pending`، **ويتوقّف فورَ القبول أو الرفض أو
	// انقضاء المهلة أو معالجةِ الطلب من جهازٍ آخر** (يُوائم حقيقةَ الخادم عبر
	// القائمة الحيّة، فلا رنينَ يتيم ولا بعثٌ لطلبٍ عولج). **قناةُ
	// `rahalgo_urgent` العالية للخلفيّة والقفل، وبلا `full-screen intent`
	// مقيَّدة** (`MerchantOrderAlert`).
	{ID: "AND-32", App: "merchant", Area: "push",
		Title: "جرسُ الطلب الجديد يتكرّر ثمّ يتوقّف — ولا رنينَ يتيم",
		Kind:  OnDevice, Automation: Manual, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},

	// ── إعادةُ الوصل برمزٍ منتهٍ — `D19` · دورةُ ٥٨ ────────────────
	//
	// **ووحدةٌ مشتركةٌ لا تطبيقاً بعينه**: `LiveSocket` و`RealtimeAuth`
	// في `:shared` — **يستعملها الزبونُ والسائقُ والمتجرُ والمندوب.**
	//
	// **والمقيسُ بمقبسٍ حقيقيٍّ ومصافحةٍ حقيقيّة** — **يُسجَّل الرمزُ
	// الذي عُرض على السلك**، لا نيّةُ الشيفرة.
	{ID: "AND-60", App: "all", Area: "realtime",
		Title: "رمزٌ منتهٍ وعائلةٌ سليمة ⇒ تجديدٌ واحدٌ ثمّ وصلٌ بالجديد",
		Kind:  Structural, Automation: Automated, Registers: []string{"D19"},
		Test: "D19ReconnectTest", Result: Pass,
		Evidence: "المصافحةُ الأولى تحمل الجديد · وتجديدٌ واحدٌ لا أكثر · وحدثٌ يصل"},
	{ID: "AND-61", App: "all", Area: "realtime",
		Title: "انقطاعُ نقلٍ برمزٍ صالحٍ لا يُدوّر رمزَ التجديد",
		Kind:  Structural, Automation: Automated, Registers: []string{"D19"},
		Test: "D19ReconnectTest.t1", Result: Pass,
		Evidence: "٥٠٢ ثمّ وصلٌ ناجح · وصفرُ تجديدات"},
	{ID: "AND-62", App: "all", Area: "realtime",
		Title: "مرفوضٌ ومتعذّرٌ لا يمحوان اعتماداً ولا يطرقان التجديد",
		Kind:  Structural, Automation: Automated, Registers: []string{"D19", "R16"},
		Test: "D19StressTest.unavailable50AndRevoked50", Result: Pass,
		Evidence: "٥٠٣×٥٠ و٤٠١×٥٠ · صفرُ اعتماداتٍ مُحيت · صفرُ تجديدات"},
	{ID: "AND-63", App: "all", Area: "realtime",
		Title: "تكرارُ التعافي والتزامن",
		Kind:  Structural, Automation: Automated, Registers: []string{"D19"},
		Test: "D19StressTest", Result: Pass,
		Evidence: "تعافٍ×١٠٠ · وصلٌ عاديٌّ×١٠٠ · تزامنٌ×١٠٠ · صفرُ حلقاتٍ قديمة"},

	{ID: "AND-40", App: "rep", Area: "permissions",
		Title: "لا إذنَ موقعٍ قبل الدخول",
		Kind:  OnDevice, Automation: SemiAuto, Registers: []string{"REP-OWNER-02"},
		Result:   NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},
	{ID: "AND-41", App: "rep", Area: "session",
		Title: "انتهاءُ الجلسة وموتُ العمليّة أثناء استمارة",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},

	// ── ما يمسّ كلَّ التطبيقات ──────────────────────────────────────
	{ID: "AND-50", App: "all", Area: "auth",
		Title: "إجبارُ تبديل كلمة المرور",
		Kind:  OnDevice, Automation: SemiAuto, Registers: []string{"D11"},
		Result:   NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},
	{ID: "AND-51", App: "all", Area: "push",
		Title: "إلغاءُ تسجيل الرمز عند الخروج",
		Kind:  OnDevice, Automation: SemiAuto, Registers: []string{"D12"},
		Result:   NotRun,
		Evidence: "لا جهازَ متّصل · **وجانبُ الخادم مُثبَتٌ في `P-7`**",
		Needs:    "SM-A525F"},
	// ── الخروجُ يُنهي وجهةَ الدفع — `D12` · دورةُ ٥٩ ──────────────
	//
	// **والسلطةُ في المحرّك**: **بابُ الخروج يحذف وجهةَ هذا الجهاز
	// في معاملةِ إبطال العائلة نفسِها** — **ولا يُكتفى بأن يتجاهل
	// التطبيقُ ما يصله.**
	{ID: "AND-64", App: "all", Area: "push",
		Title: "الخروجُ يُنهي وجهةَ هذا الجهاز ويُبقي غيرَه",
		Kind:  Structural, Automation: Automated, Registers: []string{"D12"},
		Test: "TestD12_LogoutRemovesThisDeviceBinding", Result: Pass,
		Evidence: "المُرسِلُ لا يعود يختار الجهازَ · وجهازٌ ثانٍ للحساب نفسِه باقٍ"},
	{ID: "AND-65", App: "all", Area: "push",
		Title: "تبديلُ الحساب على الجهاز نفسِه لا يسرق وجهةً",
		Kind:  Structural, Automation: Automated, Registers: []string{"D12"},
		Test: "TestD12_AccountSwitchOnSameDevice", Result: Pass,
		Evidence: "إلغاءُ الأوّلِ لرمزٍ يملكه الثاني لا يمسّه — والحذفُ مقيَّدٌ بصاحبه"},
	{ID: "AND-67", App: "all", Area: "push",
		Title: "الخروجُ يُنهي وجهةَ عائلته ولو تعذّر رمزُ الجهاز",
		Kind:  Structural, Automation: Automated, Registers: []string{"D12"},
		Test: "TestD12_LogoutWithoutDeviceTokenStillRemovesBinding", Result: Pass,
		Evidence: "المِلكيّةُ عائلةُ جلسة · خروجٌ بلا رمزِ جهازٍ ⇒ صفرُ وجهات · والمُرسِلُ صامت"},
	{ID: "AND-66", App: "all", Area: "push",
		Title: "تكرارُ الخروج والتسجيل والسباق",
		Kind:  Structural, Automation: Automated, Registers: []string{"D12"},
		Test: "TestD12_Stress*", Result: Pass,
		Evidence: "خروجٌ×١٠٠ · جهازان×١٠٠ · سباقُ خروجٍ وتسجيل×١٠٠ · صفرُ وجهاتٍ ناجية"},
	{ID: "AND-52", App: "all", Area: "realtime",
		Title: "وصلةٌ مفتوحةٌ بعد إبطال الجلسة",
		Kind:  OnDevice, Automation: SemiAuto, Registers: []string{"R14"},
		Result:   NotRun,
		Evidence: "لا جهازَ متّصل · **وعزلُ الغرف مُثبَتٌ في `P-7`**",
		Needs:    "SM-A525F"},
	{ID: "AND-53", App: "all", Area: "push-targeting",
		Title: "حسابٌ بأكثر من تطبيق — أيصل الإشعارُ إلى الخطأ؟",
		Kind:  OnDevice, Automation: SemiAuto, Registers: []string{"XOB-4"},
		Result:   NotRun,
		Evidence: "لا جهازَ متّصل · **و٢٧ من ٤٧ موضعاً بلا توجيهٍ مقيسةٌ في `P-7`**",
		Needs:    "جهازان أو حسابٌ بتطبيقين"},
	{ID: "AND-54", App: "all", Area: "network",
		Title: "انقطاعُ الشبكة وعودتُها والتبديلُ بين واي فاي والبيانات",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F + شريحةُ بيانات"},
	{ID: "AND-55", App: "all", Area: "install",
		Title: "تثبيتٌ جديدٌ وترقيةٌ ومسحُ بياناتٍ وإعادةُ تثبيت",
		Kind:  OnDevice, Automation: SemiAuto, Result: NotRun,
		Evidence: "لا جهازَ متّصل", Needs: "SM-A525F"},
}

// Counts إحصاءٌ مولَّد.
type Counts struct {
	Devices, DevicesAvailable, DevicesExecuted         int
	Cases, Structural, UnitK, ReplayK, DeviceK, DriveK int
	AutomatedN, SemiAutoN, ManualN                     int
	PassN, ExpectedFailN, RiskConfirmedN, NotRunN      int
	Apps                                               int
}

// Snapshot صورةٌ آليّة.
func Snapshot() map[string]any {
	c := Counts{Devices: len(Devices), Cases: len(Cases)}
	apps := map[string]bool{}
	for _, d := range Devices {
		if d.Status == Available {
			c.DevicesAvailable++
		}
		if d.Run == Executed {
			c.DevicesExecuted++
		}
	}
	for _, x := range Cases {
		if x.App != "all" {
			apps[x.App] = true
		}
		switch x.Kind {
		case Structural:
			c.Structural++
		case Unit:
			c.UnitK++
		case Replay:
			c.ReplayK++
		case OnDevice:
			c.DeviceK++
		case Drive:
			c.DriveK++
		}
		switch x.Automation {
		case Automated:
			c.AutomatedN++
		case SemiAuto:
			c.SemiAutoN++
		case Manual:
			c.ManualN++
		}
		switch x.Result {
		case Pass:
			c.PassN++
		case ExpectedFail:
			c.ExpectedFailN++
		case RiskConfirmed:
			c.RiskConfirmedN++
		case NotRun:
			c.NotRunN++
		}
	}
	c.Apps = len(apps)
	dv := append([]Device(nil), Devices...)
	cs := append([]Case(nil), Cases...)
	sort.Slice(dv, func(i, j int) bool { return dv[i].ID < dv[j].ID })
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	return map[string]any{"devices": dv, "cases": cs, "counts": c}
}
