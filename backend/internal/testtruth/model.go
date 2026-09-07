// حقيقةُ الاختبار — **الرابطُ بين ما تقوله الوثائقُ وما تثبته الشيفرة.**
//
// (المرحلةُ `P-2` من منظومة الاختبار الدائمة — قرارُ المالك ٢٠٢٦-٠٩-٠٥.)
//
// # المشكلةُ التي يحلّها
//
// **المستودعُ فيه مئاتُ الاختبارات ولا واحدٌ منها يعرف ماذا يحرس.**
// **فمن بدّل ملفّاً لا يعرف ما يُعيد تشغيلَه، ومن أصلح عيباً لا يعرف أيَّ
// اختبارٍ يمنع عودتَه، ومن أضاف إعداداً لا يعرف ما يمسّه.**
//
// # والمبدأُ الحاكم
//
//	DERIVE WHAT CAN BE DERIVED
//	DECLARE ONLY WHAT CANNOT BE DERIVED
//	DRIFT MUST FAIL
//
// **فما يُستخرَج من الشيفرة لا يُكتب بيد** — الأبوابُ والانتقالاتُ وأنواعُ
// القيد ومواضعُ الإشعار وغرفُ البثّ والإعداداتُ والاختباراتُ نفسُها.
// **وما لا يُستخرَج يُعلَن صراحةً** — التدفّقاتُ الأربعةُ والثلاثون وما
// يحرسه كلُّ اختبار.
//
// # ولا حقيقةَ ثانية
//
// **العيوبُ والمخاطرُ من السجلّ المجمَّد** · **والإعداداتُ من معجمها** ·
// **وعقدُ الخصوصيّة من `P-1`** — **تُقرأ ولا تُنسَخ.**
package testtruth

// Status حالُ تغطيةٍ أو تحقّق.
type Status string

const (
	StatusCovered        Status = "COVERED"
	StatusPartial        Status = "PARTIALLY_COVERED"
	StatusExpectedFail   Status = "EXPECTED_FAIL"
	StatusNotImplemented Status = "NOT_IMPLEMENTED"
	StatusRuntimeOnly    Status = "RUNTIME_ONLY"
	StatusNoTestYet      Status = "NO_REGRESSION_TEST_YET"
	StatusPresent        Status = "REGRESSION_TEST_PRESENT"
	StatusFixedPassing   Status = "FIXED_AND_PASSING"
)

// Level طبقةُ الاختبار — **الاثنتا عشرةُ المعتمدةُ في التصميم.**
type Level string

const (
	L1  Level = "L1"  // وحدة
	L2  Level = "L2"  // خدمة/مجال
	L3  Level = "L3"  // مستودع/قاعدة
	L4  Level = "L4"  // واجهةُ برمجة
	L5  Level = "L5"  // عقود
	L6  Level = "L6"  // متصفّح
	L7  Level = "L7"  // جهاز
	L8  Level = "L8"  // طرفٌ لطرف
	L9  Level = "L9"  // حقنُ عطل
	L10 Level = "L10" // حِمل
	L11 Level = "L11" // أمن
	L12 Level = "L12" // إصدار
)

// Purpose **غرضُ اختبارٍ لا يحرس ميزةً** — فلا يُعدّ يتيماً بلا سبب.
type Purpose string

const (
	PurposeFeature        Purpose = ""                    // يحرس ميزةً أو تدفّقاً
	PurposeInfrastructure Purpose = "INFRASTRUCTURE_TEST" // بنيةٌ تحتيّة
	PurposeGenerator      Purpose = "GENERATOR_TEST"      // يحرس مولِّداً
	PurposeHarnessSelf    Purpose = "HARNESS_SELF_TEST"   // المِسنَدُ يختبر نفسَه
	// PurposeStale اختبارٌ يقيس عقداً بُدِّل — **ولا يحرس شيئاً قائماً.**
	//
	// **وسقوطُه ليس عيبَ منتج**: يقيس ما قرّر المالكُ تبديلَه، **فيُوسَم
	// ولا يُحذَف بيدي** — والحذفُ أو إعادةُ الصياغة قرارُ منتجٍ لا قرارُ
	// مِسنَد.
	PurposeStale Purpose = "STALE_CONTRACT_MISMATCH"
)

// TestFn اختبارٌ واحدٌ كما اكتُشف في الشجرة — **مستخرَجٌ لا معلَن.**
type TestFn struct {
	Name    string `json:"name"`
	File    string `json:"file"`
	Package string `json:"package"`

	// ما يلي **معلَنٌ** — يُملأ من `declared.go` عند المطابقة.
	Level    Level    `json:"level,omitempty"`
	Purpose  Purpose  `json:"purpose,omitempty"`
	Flows    []string `json:"flows,omitempty"`
	Defects  []string `json:"defects,omitempty"`
	Risks    []string `json:"risks,omitempty"`
	Gaps     []string `json:"gaps,omitempty"`
	Settings []string `json:"settings,omitempty"`
	Modes    []string `json:"modes,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
	Orphan   bool     `json:"orphan"`
}

// Flow تدفّقٌ عبر مكوّنين فأكثر — **معلَنٌ، ومصدرُه كتالوجُ التفاعل.**
type Flow struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Actor    string   `json:"actor"`
	Apps     []string `json:"apps"`
	Money    bool     `json:"money"`
	Realtime bool     `json:"realtime"`
	Severity string   `json:"severity"`
	// NeedLevels **الطبقاتُ اللازمةُ** — تُشتقّ بقواعدَ لا تُكتب.
	NeedLevels []Level  `json:"need_levels"`
	Defects    []string `json:"defects,omitempty"`
	Risks      []string `json:"risks,omitempty"`
	Gaps       []string `json:"gaps,omitempty"`
	// Tests **يُملأ بالمطابقة لا بالإعلان.**
	Tests  []string `json:"tests"`
	Status Status   `json:"status"`
}

// Defect عيبٌ مجمَّد — **العنوانُ والشدّةُ من السجلّ، والحالُ بالمطابقة.**
type Defect struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Domain **عمودُ «المجال»** — ومنه تُصنَّف بوّابةُ الإطلاق.
	Domain string `json:"domain"`
	// Severity **من عمود «الشدّةُ المرشَّحة» في السجلّ** — ولا تُكتب بيد.
	Severity string `json:"severity"`
	// Fixed **دليلُ الإصلاح** — كما في `Gap`.
	//
	// **والعيوبُ تُستخرَج من `FINAL_STATIC_CLOSEOUT.md` ولا تُعلَن هنا**،
	// **فدليلُ إصلاحها في `DefectFixed`** — والسجلُّ يبقى تاريخاً لا
	// يُعدَّل.
	Fixed  string   `json:"fixed,omitempty"`
	Tests  []string `json:"tests"`
	Status Status   `json:"status"`
}

// Risk خطرٌ مجمَّد — **ولا يصير عيباً تلقائيّاً.**
type Risk struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Domain **عمودُ «المجال»** — **وسجلُّ المخاطر لا شدّةَ فيه**، فالمجالُ
	// وحدَه ما يُصنِّفه.
	Domain   string   `json:"domain"`
	Strategy string   `json:"strategy"`
	Flows    []string `json:"flows,omitempty"`
	Tests    []string `json:"tests"`
	Status   Status   `json:"status"`
}

// Gap فجوةُ عقدٍ معتمدة — **وشدّتُها من `TQ-1`.**
type Gap struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Severity string `json:"severity"`
	// WokenBy **الإعدادُ الذي يوقظ الفجوةَ النائمة** — إن عُرف.
	WokenBy string `json:"woken_by,omitempty"`
	// Fixed **دليلُ الإصلاح** — تقرؤه البوّابةُ فلا تعدّها مانعة.
	Fixed string `json:"fixed,omitempty"`
	// PartOf **جذرُها جذرُ فجوةٍ أخرى** — **تُقرأ ولا تُعَدّ مانعاً
	// ثانياً**، وهو عقدُ `SupersededBy` في البوّابة.
	PartOf string   `json:"part_of,omitempty"`
	Tests  []string `json:"tests"`
	Status Status   `json:"status"`
}

// Setting إعدادٌ مُغيِّرٌ للسلوك — **مستخرَجٌ من المعجم، ومربوطٌ بالإعلان.**
type Setting struct {
	Key       string   `json:"key"`
	Group     string   `json:"group"`
	Kind      string   `json:"kind"`
	Sensitive bool     `json:"sensitive"`
	Money     bool     `json:"money"`
	Flows     []string `json:"flows"`
	Apps      []string `json:"apps"`
	Tests     []string `json:"tests"`
	Status    Status   `json:"status"`
}

// Derived **ما استُخرج من الشيفرة** — ولا سطرَ منه مكتوبٌ بيد.
type Derived struct {
	Routes            int      `json:"routes"`
	Transitions       int      `json:"transitions"`
	LedgerKinds       []string `json:"ledger_kinds"`
	NotifySites       int      `json:"notify_sites"`
	NotifyTargeted    int      `json:"notify_targeted"`
	PublishRooms      []string `json:"publish_rooms"`
	SettingDefs       int      `json:"setting_defs"`
	BehaviourSettings int      `json:"behaviour_settings"`
	OrderFields       int      `json:"order_fields"`
	TestFiles         int      `json:"test_files"`
	TestFuncs         int      `json:"test_funcs"`
}

// Truth الحقيقةُ كاملةً — **وهذا ما يُكتب `TEST_TRUTH.json`.**
type Truth struct {
	// Baseline **التزامُ الحقيقة** — يُملأ عند التوليد.
	Baseline string `json:"baseline"`

	Derived Derived `json:"derived"`

	Flows    []Flow    `json:"flows"`
	Defects  []Defect  `json:"defects"`
	Risks    []Risk    `json:"risks"`
	Gaps     []Gap     `json:"gaps"`
	Settings []Setting `json:"settings"`
	Tests    []TestFn  `json:"tests"`

	// Stale **مراجعُ تشير إلى ما لم يعد موجوداً** — وسقوطٌ لا تقرير.
	Stale []string `json:"stale"`
	// CoverageGaps **ما يحتاج اختباراً ولا اختبارَ له.**
	CoverageGaps []string `json:"coverage_gaps"`
}
