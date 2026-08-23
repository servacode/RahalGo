package com.rahalgo.navigation

import kotlin.math.abs
import kotlin.math.max
import kotlin.math.min

/**
 * ══════════════════════════════════════════════════════════════════════
 * **هل خرج السائقُ فعلاً — أم أنّ القمرَ الصناعيَّ كذب؟**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٣أ، أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
 *
 * # ولا يقول «إعادةَ حساب»
 *
 * **يقول جملةً واحدةً: خرج أو لم يخرج.** ومن يقرّر ماذا يُفعل بعدها
 * محرّكٌ آخرُ في المرحلة ٣ب — **وخلطُ الحكم بالفعل هو ما يجعل الحالَ
 * تتذبذب**: حالُ الطريق شيءٌ، وحالُ الطلب شيءٌ آخر.
 *
 * # ولا مسافةَ وحدَها
 *
 * (أمرُ المالك: «لا نريد مجرّد `distance from route > X → reroute`».)
 *
 *     العتبةُ  =  أساسٌ  +  شكُّ القراءة  +  سماحُ المناورة
 *     والحكمُ  =  دليلٌ يتراكم، لا قراءةٌ واحدةٌ تحسم
 *
 * # وأربعةُ شواهدَ لا شاهدٌ واحد
 *
 *     البعدُ عن الخطّ      ·  ويتضاعف وزنُه إن كان صريحاً
 *     مخالفةُ الاتّجاه      ·  شارعٌ موازٍ لا يُكشف بالمسافة
 *     تقدّمٌ لا يتقدّم       ·  يتحرّك ولا يقطع من المسار
 *     قراءةٌ على المسار     ·  تخصم — والعودةُ أسرعُ من الخروج
 *
 * # والأرقامُ تُضبط ولا تُقدَّس
 *
 * (أمرُ المالك: «`Initial Tunable Parameters` وليست Constants مقدّسة…
 *  مركزيّة وواضحة وليست Magic Numbers موزّعة».)
 *
 * **فكلُّها في `Tuning` وحدَها** — تُبدَّل بعد الرفائد والميدان **بلا
 * أن يُمسّ سطرٌ من الخوارزميّة.**
 */
class OffRouteDetector(
    /** **كلُّ رقمٍ في الخوارزميّة هنا** — ولا رقمَ خارجَها. */
    val tuning: Tuning = Tuning(),
) {

    /**
     * ══════════════════════════════════════════════════════════════════
     * **الأرقامُ الأوّليّة — مقيسةٌ من المرحلتين لا مخترعة**
     * ══════════════════════════════════════════════════════════════════
     */
    data class Tuning(
        /**
         * **أساسُ العتبة.**
         *
         * **ومشتقٌّ لا مخترع**: `RouteProjector.BACK_WINDOW_M = 30`
         * نافذةُ الرجوع، **ومن خرج أبعدَ منها خرج عن نافذة الإسقاط
         * نفسِها.**
         */
        val baseThresholdM: Double = 30.0,

        /**
         * **كم يُطرح من شكّ القراءة.**
         *
         * **و`GpsQuality` تطرح الشكَّ قبل الحكم** في القفزات — والمنطقُ
         * نفسُه هنا: **قراءةٌ دقّتُها أربعون قد تبعد أربعين بحقّ.**
         */
        val accuracySlackFactor: Double = 1.0,

        /**
         * ══════════════════════════════════════════════════════════════
         * **وسقفُ ذلك الشكّ — خمسةٌ وعشرون لا ستّون**
         * ══════════════════════════════════════════════════════════════
         *
         * (قرارُ المالك ٢٠٢٦-٠٨-٢٠، بعد قياس `OFFR-014`.)
         *
         * **وكان ستّين فقِيس أثرُه**: بدقّةِ ٥٥م لا يُشَكُّ في السائق
         * حتّى يبتعد مئةً وستّين متراً، **ومسقوفاً بخمسةٍ وعشرين يُشَكُّ
         * عند مئةٍ وعشرة — بصفر إنذارٍ كاذبٍ في الحالين.**
         *
         * **وخمسةٌ وعشرون هي `GOOD_ACCURACY_M` نفسُها**: ما فوقها
         * قراءةٌ متدهورةٌ **وزنُها منصَّفٌ أصلاً**، فلا تُكافأ مرّتين.
         *
         * **ولا ممرَّ بمئةِ مترٍ حولَ الطريق داخلَ مدينة** — أمرُ
         * المالك نصّاً.
         */
        val maxAccuracySlackM: Double = 25.0,

        /**
         * **وعند هذه النقاط يُعلَن الشكّ.**
         *
         * **وقياسٌ كشف لزومَها** (٢٠٢٦-٠٨-٢٠): بلاها تدخل الحالُ
         * الشكَّ بأوّل نقطة، **فقراءاتٌ تتناوب حولَ العتبة تعطي سبعةً
         * وأربعين انتقالاً في إحدى وخمسين قراءة** — وهو التذبذبُ
         * الذي نهى عنه المالك نصّاً.
         *
         * **وتجاوزٌ حدّيٌّ واحدٌ لا يكفي** — والخصمُ ضِعفُه، فقراءةٌ
         * نظيفةٌ واحدةٌ تمحو تجاوزين.
         */
        val suspectScore: Double = 2.0,

        /** **وعند هذه النقاط يُعلَن الخروج.** */
        val confirmScore: Double = 4.0,

        /** **وكم قراءةً نظيفةً متتاليةً تُعيده** بعد إعلانِ الخروج. */
        val returnConfirm: Int = 3,

        /**
         * ══════════════════════════════════════════════════════════════
         * **وكم تبقى شهادةُ القراءة المقبولة صالحة**
         * ══════════════════════════════════════════════════════════════
         *
         * (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «لا تجعل قراءة `ACCEPTED` قديمةً
         *  جدّاً تسمح بتأكيدٍ متأخّرٍ بعد سلسلةٍ طويلةٍ من DEGRADED…
         *  يجب أن يكون Accepted evidence حديثاً ومرتبطاً بنفس Episode
         *  من الاشتباه».)
         *
         * **وبالزمن الرتيب لا بعدد القراءات** (تصحيحُ المالك
         * ٢٠٢٦-٠٨-٢٠): **«لا أريد ربطَ صحّة القرار بتردّد GPS
         * المفترَض»**.
         *
         * **وخمسُ قراءاتٍ في ثانيةٍ ليست خمسَ ثوانٍ** — وجهازٌ يعطي
         * خمسَ قراءاتٍ في الثانية كان يمنح الشهادةَ خُمسَ عمرها،
         * **وجهازٌ يتلكّأ عشرَ ثوانٍ بين قراءتين كان يمنحها ضِعفَينِ
         * وخمسةً.**
         *
         * **والمقياسُ `atMs` الرتيبُ من إقلاع الجهاز** — لا ساعةُ
         * الجدار: **تلك تقفز حين يضبطها النظامُ من الشبكة.**
         */
        val acceptedEvidenceWindowMs: Long = 5_000L,

        /** **دون هذه المسافة إلى المناورة نحن في سياقها.** */
        val maneuverNearM: Double = 40.0,

        /** **وسماحُها.** */
        val maneuverSlackM: Double = 20.0,

        /**
         * **وسماحُ الدوّار أوسع.**
         *
         * **وقِيس في المرحلة ٢**: دوّاراتُ الرقّة كثيفة، **وقوسُ
         * الدوّار يُبعد الدبّوسَ عن الخطّ بطبيعته.**
         */
        val roundaboutSlackM: Double = 25.0,

        /** **ومناورتان تُعرضان معاً** — الطريقُ ملتوٍ هنا. */
        val closeManeuverSlackM: Double = 15.0,

        /** **ودونها تُعدّ المناورتان متقاربتين** — كـ`SHOW_TOGETHER_M`. */
        val closeManeuverGapM: Double = 50.0,

        /** **وضِعفُ العتبة خروجٌ صريحٌ لا حدّيّ.** */
        val grossFactor: Double = 2.0,

        /** **وزنُ تجاوزٍ حدّيّ.** */
        val overWeight: Double = 1.0,

        /** **ووزنُ الصريح.** */
        val grossWeight: Double = 2.0,

        /**
         * **وزيادةٌ للسرعة العالية.**
         *
         * **من يقطع ثلاثين متراً في الثانية في شارعٍ آخرَ خرج فعلاً** —
         * ولا ينتظر كما ينتظر الماشي.
         */
        val fastBonus: Double = 1.0,

        /** **ووزنُ مخالفة الاتّجاه.** */
        val bearingWeight: Double = 1.0,

        /** **ووزنُ تقدّمٍ لا يتقدّم.** */
        val stalledWeight: Double = 1.0,

        /** **وما تخصمه القراءةُ النظيفة** — والعودةُ أسرعُ من الخروج. */
        val clearWeight: Double = -2.0,

        /** **وحصّةُ المتدهورة من كلّ وزن** — تُسهم ولا تحسم. */
        val degradedFactor: Double = 0.5,

        /** **وفوق هذا الانحرافِ يُعدّ الاتّجاهُ مخالفاً.** */
        val bearingOffDeg: Float = 60f,

        /** **ودون هذه السرعة لا يُصدَّق اتّجاه** — كـ`BearingTracker`. */
        val minBearingSpeedMps: Float = BearingTracker.MIN_TRUSTED_MPS,

        /** **ودونها هو واقفٌ** — والانجرافُ ليس خروجا. */
        val stationaryMps: Float = 1f,

        /** **وفوقها سريع** — ثمانيةُ أمتارٍ ≈ ٢٩ كم/س. */
        val fastMps: Float = 8f,

        /** **وأقلُّ تقدّمٍ يُعدّ تقدّماً** في الثانية الواحدة. */
        val stalledProgressM: Double = 3.0,

        /**
         * ══════════════════════════════════════════════════════════════
         * **والخروجُ دعوى هندسيّةٌ — لا تُثبَت باتّجاهٍ وحدَه**
         * ══════════════════════════════════════════════════════════════
         *
         * (المرحلة ٥، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٣: «لا ينبغي أن
         *  يستطيع `OffRouteDetector` إعلانَ `OFF_ROUTE` **بسبب
         *  Bearing/Progress evidence وحدَها** داخل هذا الممرّ
         *  الضيّق».)
         *
         * **و«خرج عن المسار» تعني أنّه ليس عليه** — ومن كان على بُعد
         * عشرين متراً يسير بعكسه **لم يتركه، بل يسير فيه معكوسا.**
         * **وذاك سؤالُ `WrongWayDetector` لا سؤالُنا.**
         *
         * **فالتأكيدُ يشترط أن يكون البعدُ قد تجاوز العتبةَ مرّةً في
         * النوبة** — والاتّجاهُ والتجمّدُ يبقيان شاهدَين يُعجّلان، **لا
         * دليلَين يحسمان وحدَهما.**
         *
         * **ولا يمسّ هذا `real-deviation`**: من انحرف حقّاً يتجاوز
         * العتبةَ بطبيعته. (وأُثبت بالانحدار.)
         */
        val requireDistanceToConfirm: Boolean = true,
    ) {
        /** **وسقفُ النقاط** — فلا تتراكم إلى ما لا نهاية فتؤخّر العودة. */
        val maxScore: Double get() = confirmScore * 2
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **ثلاثُ حالاتٍ — ولا `REROUTING` هنا**
     * ══════════════════════════════════════════════════════════════════
     *
     * (أمرُ المالك: «اعتمد فصلَ حال الطريق عن حال Reroute… ولا نريد
     *  `REROUTING` داخل `OffRouteDetector`».)
     */
    enum class State {
        ON_ROUTE,
        SUSPECTED_OFF_ROUTE,
        OFF_ROUTE,
    }

    /** **ولماذا لم تُحسب هذه القراءة** — للتشخيص والاختبار. */
    enum class Skip {
        /** **حُسبت.** */
        NONE,

        /** **مرفوضةٌ — تُتجاهل كأنّها لم تصل.** (أمرُ المالك.) */
        REJECTED,

        /** **لا مسارَ بعد** — فلا خروجَ عمّا لا وجودَ له. */
        NO_ROUTE,

        /** **واقفٌ — يُجمَّد الدليلُ ولا يُمحى.** (أمرُ المالك.) */
        STATIONARY,
    }

    /** **حكمُ اللحظة.** */
    data class Verdict(
        val state: State,
        val score: Double,
        /** **العتبةُ النافذةُ لهذه القراءة** — تُقرأ في القياس. */
        val thresholdM: Double,
        val offRouteM: Double,
        val skip: Skip,
        /** **أدلّةُ هذه القراءة** — للتقرير لا للمنطق. */
        val overThreshold: Boolean,
        val grossly: Boolean,
        val bearingAgainst: Boolean,
        val stalled: Boolean,
    )

    var state: State = State.ON_ROUTE
        private set

    var score: Double = 0.0
        private set

    private var cleanRun = 0
    /** **أتجاوز البعدُ العتبةَ مرّةً في هذه النوبة؟** — انظر `Tuning`. */
    private var sawDistanceEvidence = false
    private var lastProgressM = Double.NaN
    private var lastAtMs = 0L

    /**
     * ══════════════════════════════════════════════════════════════════
     * **عمرُ آخرِ شهادةٍ من قراءةٍ مقبولة**
     * ══════════════════════════════════════════════════════════════════
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «**مهما تراكمت قراءاتُ DEGRADED
     *  وحدَها، لا تنتقل الحالُ إلى `OFF_ROUTE`**».)
     *
     * **وهذا كلُّ ما لزم من حالٍ جديدة** — لحظةُ آخر قراءةٍ مقبولةٍ
     * مريبة، **بالمقياس الرتيب.** وصفرٌ يعني «لا شهادةَ».
     *
     * **وتُمحى بانتهاء نوبة الشكّ** — فلا تُستأنف شهادةٌ من نوبةٍ
     * مضت.
     */
    private var evidenceAtMs = 0L

    /** **أثمّةَ شهادةٌ مقبولةٌ حيّةٌ عند هذه اللحظة؟** */
    fun hasFreshAcceptedEvidence(nowMs: Long): Boolean =
        evidenceAtMs != 0L && (nowMs - evidenceAtMs) <= tuning.acceptedEvidenceWindowMs

    /** **وعند آخر قراءةٍ محسوبة** — تُقرأ في الاختبار والتشخيص. */
    val hasFreshAcceptedEvidence: Boolean get() = hasFreshAcceptedEvidence(lastAtMs)

    /** **عدّاداتُ القياس** — تُقرأ في تقرير المرحلة. */
    var suspicions = 0
        private set
    var confirmations = 0
        private set

    /**
     * **يُغذّى قراءةً وحالَ التقدّم فيردّ الحكم.**
     *
     * **ولا يعرف شبكةً ولا خريطةً ولا طلبا** — حسابٌ محضٌ يُعاد تشغيلُه
     * في اختبارٍ بلا هاتف.
     */
    fun onFix(
        fix: NavFix,
        grade: FixGrade,
        headingDeg: Float?,
        route: NavRoute?,
        progress: RouteProgress.State?,
    ): Verdict {
        // **والمرفوضةُ لا تزيد ولا تنقص ولا تُعيد حالاً** — أمرُ المالك
        // نصّاً: **«تُتجاهل في قرار Off-route كأنّها لم تصل».**
        if (grade == FixGrade.REJECTED) return verdict(Skip.REJECTED, 0.0, 0.0)
        if (route == null || !route.usable || progress == null) {
            return verdict(Skip.NO_ROUTE, 0.0, 0.0)
        }

        val threshold = thresholdFor(fix, route, progress)
        val off = progress.offRouteM

        // **والزمنُ يُحدَّث دائماً** — حتّى في المتخطّاة، **وإلّا حُسب
        // فاصلٌ ضخمٌ بعد وقوفٍ طويلٍ فبدا التقدّمُ متجمّدا.**
        val dtSec = if (lastAtMs == 0L) 0.0 else (fix.atMs - lastAtMs) / 1000.0
        val prevProgress = lastProgressM
        lastAtMs = fix.atMs
        lastProgressM = progress.progressM

        // **والواقفُ يُجمَّد ولا يُمحى** — أمرُ المالك: «الوقوفُ
        // يجمّد/يخفّف تراكمَ الدليل، لا يمحو التاريخ».
        val speed = fix.speedMps
        if (speed != null && speed < tuning.stationaryMps) {
            return verdict(Skip.STATIONARY, threshold, off)
        }

        val nearManeuver = inManeuverContext(route, progress.progressM)
        val grossly = off > threshold * tuning.grossFactor
        val over = off > threshold
        val fast = speed != null && speed >= tuning.fastMps

        // ══════════════════════════════════════════════════════════════
        // **والاتّجاهُ شاهدٌ لا حكم**
        // ══════════════════════════════════════════════════════════════
        //
        // (أمرُ المالك: «لا يجوز أن يعلن Bearing وحدَه Off-route… ولا
        //  تبنِ `WrongWayDetector`».)
        //
        // **ويُسكَت قربَ المناورة**: عند الانعطاف يسبق اتّجاهُ السائق
        // القطعةَ المسقَطَ عليها بطبيعته، **فمخالفةٌ هناك متوقَّعةٌ لا
        // دليل.**
        val bearingAgainst = !nearManeuver &&
            grade == FixGrade.ACCEPTED &&
            headingDeg != null &&
            speed != null && speed >= tuning.minBearingSpeedMps &&
            segmentBearing(route, progress.progressM)?.let {
                abs(GpsQuality.angleDelta(headingDeg, it)) > tuning.bearingOffDeg
            } == true

        // ══════════════════════════════════════════════════════════════
        // **ويتحرّك ولا يقطع من المسار**
        // ══════════════════════════════════════════════════════════════
        //
        // **وهذا ما يكشف من سار في شارعٍ يوازي المسارَ ولا يقترب من
        // وجهته** — والمسافةُ عنه صغيرةٌ والاتّجاهُ قد يوافق.
        val stalled = !prevProgress.isNaN() && dtSec > 0 &&
            speed != null && speed >= tuning.minBearingSpeedMps &&
            (progress.progressM - prevProgress) < tuning.stalledProgressM

        var weight = 0.0
        if (grossly) {
            weight += tuning.grossWeight + if (fast) tuning.fastBonus else 0.0
        } else if (over) {
            weight += tuning.overWeight
        }
        if (bearingAgainst) weight += tuning.bearingWeight
        if (stalled) weight += tuning.stalledWeight
        if (grade == FixGrade.DEGRADED) weight *= tuning.degradedFactor

        if (over) sawDistanceEvidence = true
        if (weight > 0) {
            cleanRun = 0
            score = min(tuning.maxScore, score + weight)
            // **والمقبولةُ المريبةُ تُجدّد الشهادة** — والمتدهورةُ
            // تُشيخها كما يُشيخها أيُّ مرورِ قراءة.
            if (grade == FixGrade.ACCEPTED) evidenceAtMs = fix.atMs
        } else {
            // **ولا تُخصم إلّا وهو داخلَ العتبة** — **ومن كان خارجَها
            // بلا شاهدٍ آخرَ لا يُكافأ بخصم.**
            if (!over) {
                cleanRun++
                score = max(0.0, score + tuning.clearWeight)
            }
        }

        advance(over, fix.atMs)
        return Verdict(
            state = state,
            score = score,
            thresholdM = threshold,
            offRouteM = off,
            skip = Skip.NONE,
            overThreshold = over,
            grossly = grossly,
            bearingAgainst = bearingAgainst,
            stalled = stalled,
        )
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **والدخولُ أصعبُ من الخروج — وهو ما يمنع التذبذب**
     * ══════════════════════════════════════════════════════════════════
     *
     * (أمرُ المالك: «لا تجعل النظامَ يتذبذب `ON OFF ON OFF` كلَّ
     *  ثانية».)
     *
     *     score ≥ suspectScore  :  شكّ
     *     score ≥ confirmScore  :  خروجٌ مؤكَّد
     *     دونهما                 :  على المسار
     *     OFF_ROUTE → ON_ROUTE  :  ثلاثُ قراءاتٍ نظيفةٍ **متتالية**
     *
     * **والخروجُ المؤكَّد بابٌ يُدخل ولا يُخرَج منه بنقطةٍ تصفّرت** —
     * يحتاج دليلاً كما احتاجه الدخول. **وما دونه يتبع النقاطَ
     * مباشرةً**، فلا تُؤخَّر إدانةٌ صريحةٌ درجةً درجة: **من قفز إلى
     * أربع نقاطٍ في قراءةٍ واحدةٍ خرج، ولا يُنتظر به دورٌ ثانٍ.**
     */
    private fun advance(over: Boolean, nowMs: Long) {
        if (state == State.OFF_ROUTE) {
            if (!over && cleanRun >= tuning.returnConfirm) {
                state = State.ON_ROUTE
                score = 0.0
            }
            return
        }
        val next = when {
            // ══════════════════════════════════════════════════════════
            // **ولا يُعلَن خروجٌ بشهادةِ ضبابٍ وحدَها**
            // ══════════════════════════════════════════════════════════
            //
            // **دقّةُ خمسين متراً تعني موضعاً في دائرةٍ قطرُها مئة** —
            // **ومئةُ قراءةٍ كهذه تبقى مئةَ دائرةٍ لا نقطةً واحدة.**
            // فتُبقي الشكَّ حيّاً ولا تحسم.
            //
            // **والحسمُ يحتاج قراءةً مقبولةً مريبةً حديثةً** من نوبة
            // الشكّ نفسِها — انظر `evidenceAtMs`.
            score >= tuning.confirmScore && hasFreshAcceptedEvidence(nowMs) &&
                (!tuning.requireDistanceToConfirm || sawDistanceEvidence) -> State.OFF_ROUTE
            score >= tuning.suspectScore -> State.SUSPECTED_OFF_ROUTE
            else -> State.ON_ROUTE
        }
        if (next == State.SUSPECTED_OFF_ROUTE && state != next) suspicions++
        if (next == State.OFF_ROUTE) confirmations++
        // **وانتهاءُ النوبة يمحو شهادتَها** — فلا تُستأنف من نوبةٍ مضت.
        if (next == State.ON_ROUTE) {
            evidenceAtMs = 0L
            sawDistanceEvidence = false
        }
        state = next
    }

    /**
     * **العتبةُ النافذةُ لهذه القراءة.**
     *
     * **والسماحاتُ لا تُجمَع بل يُؤخذ أوسعُها** — **وجمعُها يفتح
     * العتبةَ إلى مئةِ مترٍ عند دوّارٍ فيه مناورتان متقاربتان**، فلا
     * يبقى حارس.
     */
    fun thresholdFor(fix: NavFix, route: NavRoute, progress: RouteProgress.State): Double {
        val slack = min(
            tuning.maxAccuracySlackM,
            fix.accuracyM.toDouble() * tuning.accuracySlackFactor,
        )
        return tuning.baseThresholdM + slack + maneuverSlack(route, progress.progressM)
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **سماحُ سياق المناورة — ويزول بمغادرتها**
     * ══════════════════════════════════════════════════════════════════
     *
     * (أمرُ المالك: «لا تجعل Roundabout tolerance يبقى فعّالاً بعد
     *  مغادرة سياق الدوّار».)
     *
     * # ولماذا يُحسب من المسار لا من `distanceToManeuverM`
     *
     * **قِيس في معايرة ٣أ (٢٠٢٦-٠٨-٢٠) أنّ السماحَ لم يعمل قطّ عند
     * الانعطاف** — واختبارُ الأطوار الأربعة كشفه: `قربَها = 36م`.
     *
     * **والعلّةُ أنّ `RouteProgress` تُقدّم المناورةَ الجاريةَ بمجرّد
     * مجاوزة السابقة**، ثمّ تقيس المسافةَ إلى **التالية**. فعلى مسارٍ
     * مناوراتُه عند `0` و`300` و`600`: **من بلغ اثني عشر متراً صارت
     * «الجارية» عنده انعطافَ الثلاثمئة، والمسافةُ المعروضةُ إلى
     * الوصول عند الستّمئة.** — فلا تهبط دون الأربعين إلّا عند الوصول.
     *
     * **وذاك شأنُ عرضٍ لا شأنُ حكم** — والمرحلةُ الثانيةُ معتمدةٌ لا
     * تُمسّ. **وما نحتاجه سؤالٌ هندسيٌّ محض**: أنحن قربَ نقطةِ مناورةٍ
     * على الطريق؟ **وأقربُ مناورةٍ بالمسافة تجيبه** — قبلَها وبعدَها
     * معاً، **فالسائقُ يخرج عن الخطّ في قوس الانعطاف كما يخرج قبله.**
     */
    private fun maneuverSlack(route: NavRoute, progressM: Double): Double {
        var nearest = Double.MAX_VALUE
        var roundabout = false
        var close = false
        for (i in route.maneuvers.indices) {
            val m = route.maneuvers[i]
            val d = abs(m.atDistanceM - progressM)
            if (d > tuning.maneuverNearM) continue
            if (d < nearest) nearest = d
            if (m.isRoundabout) roundabout = true
            val nxt = route.maneuvers.getOrNull(i + 1)
            if (nxt != null && nxt.atDistanceM - m.atDistanceM <= tuning.closeManeuverGapM) {
                close = true
            }
        }
        if (nearest == Double.MAX_VALUE) return 0.0
        var slack = if (roundabout) tuning.roundaboutSlackM else tuning.maneuverSlackM
        if (close) slack = max(slack, tuning.closeManeuverSlackM)
        return slack
    }

    /** **أنحن في سياق مناورة؟** — وجودُ سماحٍ هو الجواب. */
    private fun inManeuverContext(route: NavRoute, progressM: Double): Boolean =
        maneuverSlack(route, progressM) > 0.0

    /**
     * **اتّجاهُ القطعة التي يقف عليها التقدّم.**
     *
     * **ويُحسب هنا لا في `RouteProgress`** — **فالمرحلةُ الثانيةُ
     * معتمدةٌ ولا تُمسّ**، وهذا حسابٌ مشتقٌّ لا حالٌ جديدة.
     */
    fun segmentBearing(route: NavRoute, meters: Double): Float? {
        val g = route.geometry
        if (g.size < 2) return null
        val i = min(g.size - 2, RouteProjector.indexAtOrBefore(route.cumulativeM, meters))
        return GpsQuality.courseBetween(
            NavFix(g[i].lat, g[i].lng, 0f, null, null, 0),
            NavFix(g[i + 1].lat, g[i + 1].lng, 0f, null, null, 0),
        )
    }

    private fun verdict(skip: Skip, threshold: Double, off: Double) = Verdict(
        state = state,
        score = score,
        thresholdM = threshold,
        offRouteM = off,
        skip = skip,
        overThreshold = false,
        grossly = false,
        bearingAgainst = false,
        stalled = false,
    )

    /** **يُنسى كلُّ شيء** — عند مسارٍ جديدٍ أو جلسةٍ جديدة. */

    /**
     * **حكمٌ ساكن — بلا تغذيةٍ ولا تبديلِ حال.**
     *
     * **يُنادى حين ينتهي إرشادُ الطريق** (آخرُ ميل، ٢٠٢٦-٠٨-٢١):
     * الهندسةُ استُنفدت، **فالقياسُ عليها لا يعني شيئاً.**
     *
     * **ولا شرطَ داخلَ الكاشف** — طبقةُ التنسيق هي من تقرّر متى
     * يُسأل ومتى يُترك (البند ٥).
     */
    fun idle(): Verdict = Verdict(
        state = state,
        score = score,
        thresholdM = 0.0,
        offRouteM = 0.0,
        skip = Skip.NO_ROUTE,
        overThreshold = false,
        grossly = false,
        bearingAgainst = false,
        stalled = false,
    )

    fun reset() {
        state = State.ON_ROUTE
        score = 0.0
        cleanRun = 0
        sawDistanceEvidence = false
        lastProgressM = Double.NaN
        lastAtMs = 0L
        evidenceAtMs = 0L
        suspicions = 0
        confirmations = 0
    }
}
