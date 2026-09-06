package gate

// ══════════════════════════════════════════════════════════════════════
// **قراراتُ المالك ومتطلّباتٌ لا تُقاس من الشيفرة**
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا يُعلَن هذا ولا يُشتقّ
//
// **الغيابُ لا يُقرأ من شيفرة.** **لا يوجد سطرٌ يقول «النسخُ الاحتياطيُّ
// لم يُجرَّب»** — **وغيابُ الدليل هو الدليلُ نفسُه، ولا بدَّ من إعلانه.**
//
// **وكلُّ سطرٍ هنا يحمل مصدرَه** — قرارَ مالكٍ بتاريخه أو مرحلةً أثبتته.
// **فما لا مصدرَ له لا يدخل.**

// Supersede **خطرٌ تأكّد فصار عيباً** (البند ٧).
//
// **والغرضُ ألّا يُحتسب سببٌ واحدٌ مانعَين** — `R22` و`D26` واقعةٌ واحدة.
var Supersede = map[string]string{
	// **قرارُ المالك ٢٠٢٦-٠٩-٠٥ · إغلاقُ ما قبل `P-6`.**
	"R10": "D24",
	// **قرارُ المالك ٢٠٢٦-٠٩-٠٥ · إغلاقُ ما قبل `P-8`.**
	"R22": "D26",
	"R23": "D27",
}

// ExternalReq متطلَّبٌ لا يُثبَت محلّيّاً.
type ExternalReq struct {
	ID          string
	Category    Category
	Requirement string
	// Source **من أوجبه** — عقدٌ أو قرارُ مالك.
	Source   string
	Severity Severity
	// Evidence **ما يلزم لإثباته** — لا أقلّ.
	Evidence string
	// Status **حالُه اليومَ** — ولا يُكتب `PASS` هنا أبداً.
	Status State
	Why    string
	// Blocking **هل يمنع الإطلاق؟**
	Blocking bool
	Waiver   WaiverPolicy
}

// ExternalReqs المتطلّباتُ التي لا مِسنَدَ محلّيّاً يثبتها.
//
// **ولا واحدٌ منها `PASS`** — **وليس ذلك تشاؤماً: لم يُشغَّل أيٌّ منها.**
var ExternalReqs = []ExternalReq{
	// ── أندرويد · قبولُ الجهاز الحقيقيّ (البند ١٨) ────────────────
	{
		ID: "GATE-AND-01", Category: CatAndroid,
		Requirement: "قبولُ الأجهزة الحقيقيّة على الجهاز المرجعيّ `SM-A525F / Android 14`",
		Source:      "`P-8` · قرارُ المالك: الجهازُ المرجعيُّ لا يُبدَّل",
		Severity:    Blocker,
		Evidence:    "تنفيذُ مصفوفة الأجهزة على جهازٍ متّصلٍ مع سجلٍّ لكلّ بند",
		Status:      NotRun,
		Why:         "**لا جهازَ متّصلٌ أصلاً** — `DevicesAvailable = 0` في مصفوفة الأجهزة",
		Blocking:    true, Waiver: WaiverOwnerOnly,
	},
	{
		ID: "GATE-NAV-01", Category: CatNavigation,
		Requirement: "قبولُ الملاحة النهائيُّ — إعادةٌ حتميّةٌ **وقيادةٌ في الشارع**",
		Source:      "`P-8` البند ١٩ · مختبرُ اختبار الملاحة ثلاثُ طبقات",
		Severity:    Critical,
		Evidence:    "إعادةُ مسارٍ مسجَّلٍ + سَوقةٌ حقيقيّةٌ موثَّقة",
		Status:      NotRun,
		Why:         "**الإعادةُ وحدَها لا تكفي** — ولم تُنفَّذ سَوقةٌ حقيقيّة",
		Blocking:    true, Waiver: WaiverOwnerOnly,
	},
	{
		ID: "GATE-BG-01", Category: CatBackground,
		Requirement: "الخلفيّةُ والغفوةُ وموتُ العمليّة لتطبيق السائق على جهازٍ حقيقيّ",
		Source:      "`P-8` البند ٢٠",
		Severity:    Blocker,
		Evidence:    "خدمةٌ أماميّةٌ · موقعٌ في الخلفيّة · `Doze` · قيدُ البطّاريّة · موتُ العمليّة",
		Status:      NotRun,
		Why:         "**الإثباتُ البنيويُّ ليس قبولَ جهاز** — خمسةُ إثباتاتٍ بنيويّةٍ لا تُغني",
		Blocking:    true, Waiver: WaiverOwnerOnly,
	},

	// ── بيئةُ التجهيز · `P-0` (البند ٢١) ──────────────────────────
	{
		ID: "GATE-STG-01", Category: CatStaging,
		Requirement: "إبطالُ الجلسة عبر عُقَدٍ متعدّدةٍ — `R16`",
		Source:      "`R16` في سجلّ المخاطر · و`P-5` أثبته محلّيّاً بعُقدةٍ واحدة",
		Severity:    Critical,
		Evidence:    "خادمان يشتركان في `Redis` ويُبطَل رمزٌ على أحدهما فيُرفَض على الآخر",
		Status:      ReqStaging,
		Why: "**عُقدةٌ واحدةٌ لا تُثبت الإبطالَ الموزَّع** — والنشرُ عُقَد. " +
			"⚠️ **و`P-0` أثبت وجهاً آخرَ من `R16`**: **سقوطُ `Redis` يجعل " +
			"التوثيقَ يسقط مفتوحاً** (`TestFAIL_R16_RedisDownFailsOpen`) — " +
			"**وذلك أخطرُ من الإبطال الموزَّع ولا يُغني عنه.**",
		Blocking: true, Waiver: WaiverOwnerOnly,
	},
	{
		ID: "GATE-STG-02", Category: CatStaging,
		Requirement: "تحقّقُ نشرٍ شبيهٍ بالإنتاج — هجراتٌ وضبطٌ وشهاداتٌ وتوجيه",
		Source:      "`P-0` · دَينُ البيئة",
		Severity:    Critical,
		Evidence:    "نشرٌ كاملٌ على بيئةٍ مخصَّصةٍ ثمّ فحوصُ دخانٍ موثَّقة",
		Status:      ReqStaging,
		Why: "**الحزمةُ بُنيت وقِيست محلّيّاً** (`P-0` ٢٠٢٦-٠٩-٠٦: هجراتٌ " +
			"نظيفةٌ ١٢٨ · عزلُ أحجامٍ مُثبَت · حرّاسُ إنتاجٍ خمسةٌ · تدريبُ " +
			"استعادة). **ولم تُنشَر على خادمٍ بعد** — **ومكدّسٌ محلّيٌّ " +
			"ليس بيئةَ تجهيزٍ مخصَّصة** (البند ٤٢).",
		Blocking: true, Waiver: WaiverOwnerOnly,
	},
	{
		ID: "GATE-OPS-02", Category: CatBackup,
		Requirement: "نسخٌ احتياطيٌّ واستعادةٌ مُثبَتان — `OPS2`",
		Source:      "`OPS2` · دَينُ التشغيل · و`P-0` البند ٢٩",
		Severity:    Blocker,
		Evidence:    "إنشاءُ نسخةٍ · استعادتُها · فحصُ سلامةٍ · إجراءٌ موثَّق",
		// **نُفِّذ على التجهيز ٢٠٢٦-٠٩-٠٦** — والخطواتُ الخمسُ كلُّها.
		Status: Pass,
		Why: "**تدريبٌ كاملٌ على التجهيز**: ٧ مستخدمين ومنطقتان ⇒ نسخةٌ " +
			"١٩٦٤٨٠ بايت ⇒ محوٌ إلى صفرٍ ⇒ استعادةٌ ⇒ ٧ ومنطقتان و٦١ جدولاً " +
			"و١٢٨ هجرةً و`PostGIS`. **والإجراءُ موثَّقٌ في `deploy/staging/RUNBOOK.md`.** " +
			"⚠️ **ولم تُجرَّب استعادةُ إنتاجٍ** — **ولا إنتاجَ بعد.**",
		Blocking: true, Waiver: WaiverOwnerOnly,
	},

	// ── الأداءُ والحِمل (البندان ٢٩ و٣٠) ──────────────────────────
	{
		ID: "GATE-PERF-01", Category: CatPerf,
		Requirement: "خطُّ أساسٍ للأداء — ثلاثُ دوراتٍ مستقرّةٍ قبل تجميد `SLO`",
		Source:      "`TQ-3` · ولم يُعتمد `SLO` مطلقٌ بعد",
		Severity:    High,
		Evidence:    "ثلاثُ دوراتِ قياسٍ متقاربةٍ على بيئةٍ ثابتة",
		Status:      NotRun,
		Why:         "**`BASELINE 0/3`** — **ولا يُخترَع رقمٌ ليمرّ**",
		Blocking:    false, Waiver: WaiverOwnerOnly,
	},
	{
		ID: "GATE-LOAD-01", Category: CatLoad,
		Requirement: "اختبارُ حِملٍ ونقعٍ قبل الإطلاق",
		Source:      "`TQ-3` · تصميمُ منظومة الاختبار",
		Severity:    High,
		Evidence:    "حِملٌ متدرّجٌ ونقعٌ طويلٌ مع رصد التسرّب",
		Status:      NotRun,
		Why:         "**لم يُنفَّذ** — ويحتاج بيئةً شبيهةً بالإنتاج",
		Blocking:    false, Waiver: WaiverOwnerOnly,
	},

	// ── الجهوزيّةُ التشغيليّة (البند ٢٢) ──────────────────────────
	{
		ID: "GATE-OPS-05", Category: CatOps,
		Requirement: "عقدُ الجهوزيّة التشغيليّة للإدارة — `AQ-5`",
		Source:      "`AQ-5` · `NOT READY / READY WITH WARNINGS / READY`",
		Severity:    High,
		Evidence:    "فحصُ الضبط التجاريِّ الإلزاميِّ على بيانات إطلاقٍ حقيقيّة",
		Status:      NotImplemented,
		Why:         "**`AQ-5` غيرُ منفَّذ** — **والنسبةُ وحدَها ليست قراراً**",
		Blocking:    false, Waiver: WaiverOwnerOnly,
	},
	{
		ID: "GATE-RBAC-01", Category: CatRBAC,
		Requirement: "أقلُّ صلاحيّةٍ في الخادم لا في الواجهة — `AQ-1`",
		Source:      "`AQ-1` · غيرُ منفَّذٍ بالكامل",
		Severity:    Critical,
		Evidence:    "موظّفُ تشغيلٍ يُمنَع من تبديل إعدادٍ ماليٍّ حسّاسٍ **من الخادم**",
		Status:      NotImplemented,
		Why:         "**الأدوارُ خشنة** — **وواجهةٌ جميلةٌ لا تُغني عن منعٍ في الخادم**",
		Blocking:    true, Waiver: WaiverOwnerOnly,
	},

	// ── أوضاعُ التشغيل الإلزاميّة (البند ٣٦) ──────────────────────
	{
		ID: "GATE-MODE-01", Category: CatCoverage,
		Requirement: "تشغيلُ الأوضاع الإلزاميّة كاملةً على المرشَّح — `FULL` و`FINANCIAL` و`SECURITY` و`CROSS_SYSTEM` و`WEB` و`ANDROID` و`RELEASE`",
		Source:      "`P-10` البند ٣٦",
		Severity:    Blocker,
		Evidence:    "تشغيلٌ كاملٌ مسجَّلٌ على مرشَّحٍ ثابتٍ لا على شجرة عمل",
		Status:      NotRun,
		Why:         "**`FAST` و`IMPACTED` لا تكفيان للإطلاق** — ولم يُسجَّل تشغيلٌ كاملٌ على مرشَّح",
		Blocking:    true, Waiver: WaiverOwnerOnly,
	},
}

// KnownTest اختبارٌ معروفُ الحال — **مُصالَحٌ لا مُتجاهَل.**
type KnownTest struct {
	Name string
	// Verdict **الحكمُ بعد المصالحة** — ولا يُكتب بلا قياس.
	Verdict string
	Why     string
	// Contract **العقدُ الذي كان يحرسه** — وهل بقي محروساً.
	Contract  string
	Unguarded bool
	Fix       string
}

// KnownTests الاختباراتُ المُصالَحة (البندان ١١ و١٢).
var KnownTests = []KnownTest{
	{
		Name:    "TestRepTarget_GrantedOnDelivery",
		Verdict: "STALE / CONTRACT-MISMATCHED",
		Why: "**العقدُ تبدّل ٢٠٢٦-٠٨-٣١ بقرار المالك** (`5da3dbed`): " +
			"هدفُ المندوب صار بالمتاجر المفتوحة لا بالطلبات المسلَّمة. " +
			"**والاختبارُ يحرس عقداً لم يعد قائماً.**",
		Contract:  "منحُ هدف المندوب",
		Unguarded: false,
		Fix:       "**مُتخطّىً بـ`t.Skip` مع تاريخه** — والتتبّعُ محفوظٌ لا محذوف",
	},
	{
		// **قِيس في `P-10` ٢٠٢٦-٠٩-٠٦ ولم يُخمَّن.**
		Name:    "TestMerchantProfile_SeesHisStore",
		Verdict: "TEST INFRA DEBT",
		Why: "**الهجرةُ `0125_drop_rating_comments.sql` حذفت العمودَ عمداً** " +
			"(قرارُ المالك ٢٠٢٦-٠٨-٣١: «تقييمٌ بدون أيّ تعليق»). " +
			"**فالمخطَّطُ صحيحٌ والاختبارُ شائخ** — يُدرج عموداً لم يعد موجوداً. " +
			"**وليس عيبَ منتج.**",
		Contract:  "ملفُّ المتجر في الإدارة يُظهر الإنذاراتِ والنزاعاتِ وتقييماتِ السائقين",
		Unguarded: true,
		Fix:       "حذفُ `comment` من جملة الإدراج في الاختبار — **سطرٌ واحد، وقرارُه للمالك**",
	},
}
