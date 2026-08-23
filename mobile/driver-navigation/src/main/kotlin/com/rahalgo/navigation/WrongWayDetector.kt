package com.rahalgo.navigation

import kotlin.math.abs

/**
 * ══════════════════════════════════════════════════════════════════════
 * **يسير في المسار — بعكسه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٥، أمرُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 * # وما تعنيه هذه الكلمةُ عندنا — بالضبط
 *
 * (قرارُ المالك، البند ١: **«لا تعني السيرَ المخالفَ قانونيّاً لاتّجاه
 *  الشارع»**.)
 *
 *     السائقُ يتحرّك بعكس اتّجاه المسار الحاليّ
 *     مع بقائه قريباً هندسيّاً منه.
 *
 * **ولا نملك قاعدةَ اتّجاهاتِ شوارعَ موثوقة** — فلا ندّعي مخالفةً
 * قانونيّة، **ولا نقول للسائق إنّه يسير عكس الشارع.** نقول إنّه يسير
 * عكس **مسارنا**.
 *
 * **وقد يكون على شارعٍ موازٍ قريب** — والإسقاطُ لا يفرّق. **وهذا قيدٌ
 * مسجَّلٌ لا خللٌ مخفيّ** (`TD-PARALLEL-SAMEDIR`).
 *
 * # والفرقُ عن «خرج عن المسار»
 *
 *     بعيدٌ هندسيّاً    ⇒  OFF_ROUTE   — تركه
 *     قريبٌ ومعاكس      ⇒  WRONG_WAY   — فيه ومعكوس
 *
 * **وتأكيدُ الخروج صار يشترط دليلَ مسافة** (المرحلة ٥، `OffRouteDetector`)
 * — **فلا يتنازع الكاشفان على الحال نفسِها.**
 *
 * # ولا عدَّ قراءات
 *
 * (قرارُ المالك، البند ١١: «لا أريد Detector يعمل أسرعَ فقط لأنّ
 *  الهاتف يعطي Fixes أكثر».)
 *
 * **الإثباتُ زمنٌ رتيبٌ ومسافةٌ مقطوعة** — فرحلةٌ عند نصف هرتزٍ
 * وأخرى عند اثنين تعطيان الحكمَ نفسَه.
 */
class WrongWayDetector(val tuning: Tuning = Tuning()) {

    /**
     * ══════════════════════════════════════════════════════════════════
     * **كلُّ رقمٍ هنا — ولا رقمَ خارجَه**
     * ══════════════════════════════════════════════════════════════════
     */
    data class Tuning(
        /**
         * **ممرُّ الاتّجاه — أضيقُ من عتبة الخروج.**
         *
         * **وأساسُ عتبة الخروج ثلاثون** (`OffRouteDetector`) — **وما
         * بينهما منطقةُ شكٍّ لا منطقةُ اتّجاه.**
         */
        val corridorM: Double = 25.0,

        /**
         * **وفوق هذه الزاوية يُعدُّ معاكساً.**
         *
         * **ودعوى «معاكس» أقوى من دعوى «غيرُ محاذٍ»** — والمرحلةُ ٣
         * تسأل الثانيةَ عند ستّين درجة. **وتسعون عموديٌّ لا معاكس**:
         * تقاطعٌ أو انعطافٌ جارٍ.
         *
         * **ومئةٌ وخمسٌ وثلاثون قيمةٌ أوّليّةٌ تُقاس** — لا تُقدَّس.
         */
        val oppositeDeg: Float = 135f,

        /** **ودونها لا يُصدَّق اتّجاه** — كـ`BearingTracker`. */
        val minTrustedSpeedMps: Float = BearingTracker.MIN_TRUSTED_MPS,

        /** **ودونها هو واقف** — تجميدٌ لا محو. */
        val stationaryMps: Float = 1f,

        /** **زمنُ الشكّ ومسافتُه** — معاً لا أحدُهما. */
        val suspectMs: Long = 3_000L,
        val suspectM: Double = 10.0,

        /** **وزمنُ التأكيد ومسافتُه.** */
        val confirmMs: Long = 8_000L,
        val confirmM: Double = 30.0,

        /**
         * **وتراجعُ التقدّم يُعجّل التأكيد.**
         *
         * **و`RouteProgress` رشّحه سلفاً**: يبتلع دون اثني عشرَ متراً
         * مع الدقّة، ويطلب ثلاثَ قراءاتٍ للكبير. **فحين ينقص فعلاً
         * يكون قد مرّ بحارس.**
         *
         * **ولا يؤكّد وحدَه** — يقترن بالزاوية والحركة والقرب.
         */
        val confirmMsWithRegression: Long = 5_000L,
        val confirmMWithRegression: Double = 18.0,

        /** **والعودةُ تحتاج سيراً أماميّاً موثوقاً.** */
        val recoverMs: Long = 4_000L,
        val recoverM: Double = 15.0,

        /** **كبتٌ حولَ الدوران المطلوب.** */
        val uTurnSuppressM: Double = 60.0,
        val roundaboutSuppressM: Double = 50.0,
        val sharpSuppressM: Double = 25.0,

        /** **ونافذةُ التصحيح قبل إعادة الحساب** — زمناً ومسافة. */
        val correctionWindowMs: Long = 10_000L,
        val correctionTravelM: Double = 40.0,

        /** **وشهادةُ القراءة المقبولة** — بالمقياس الرتيب. */
        val freshAcceptedMs: Long = 3_000L,
    )

    /** **ثلاثُ حالاتٍ — ولا `REROUTING` فيها.** */
    enum class State { CORRECT_DIRECTION, SUSPECTED_WRONG_WAY, WRONG_WAY }

    /** **ولماذا لم تُحسب هذه القراءة.** */
    enum class Skip {
        NONE,

        /** **مرفوضةٌ — لا تضيف ولا تخصم.** */
        REJECTED,

        /** **لا مسارَ ولا إسقاطٌ صالح.** */
        NO_ROUTE,

        /** **خارجَ الممرّ** — تلك حالُ الخروج لا حالُ الاتّجاه. */
        OUTSIDE_CORRIDOR,

        /** **واقفٌ** — تجميدٌ لا محو. */
        STATIONARY,

        /** **اتّجاهٌ لا يُصدَّق** — سرعةٌ دون الحدّ أو قراءةٌ متدهورة. */
        NO_BEARING,

        /** **سياقُ مناورةٍ تُدير الاتّجاهَ بحقّ.** */
        MANEUVER,
    }

    /** **حكمُ اللحظة.** */
    data class Verdict(
        val state: State,
        /** **فرقُ الزاوية** — `-180..180`، وفارغٌ إن لم يُصدَّق اتّجاه. */
        val bearingDeltaDeg: Float?,
        /** **كم دام السيرُ معاكساً** بالملّي. */
        val oppositeMs: Long,
        /** **وكم متراً قُطع معاكساً.** */
        val oppositeM: Double,
        /** **أتراجع التقدّمُ في هذه النوبة؟** */
        val progressRegressed: Boolean,
        val skip: Skip,
        /** **رقمُ النوبة** — يتبدّل فيتكلّم الصوتُ من جديد. */
        val episode: Long,
    )

    var state: State = State.CORRECT_DIRECTION
        private set

    /** **رقمُ النوبة الجارية** — يزيد مع كلّ تأكيدٍ جديد. */
    var episode: Long = 0L
        private set

    /** **عدّاداتُ القياس.** */
    var suspicions = 0
        private set
    var confirmations = 0
        private set

    private var oppositeMs = 0L
    private var oppositeM = 0.0
    private var forwardMs = 0L
    private var forwardM = 0.0
    private var regressed = false
    private var lastFix: NavFix? = null
    private var lastProgressM = Double.NaN
    private var confirmedAtMs = 0L
    private var confirmedTravelM = 0.0
    private var lastGeneration = -1L

    /**
     * **يُغذّى قراءةً وحالَ التقدّم فيردّ الحكم.**
     *
     * **ولا يعرف شبكةً ولا صوتاً** — حسابٌ محضٌ يُعاد تشغيلُه بلا
     * هاتف.
     */
    fun onFix(
        fix: NavFix,
        grade: FixGrade,
        headingDeg: Float?,
        route: NavRoute?,
        progress: RouteProgress.State?,
        generation: Long,
    ): Verdict {
        if (generation != lastGeneration) {
            lastGeneration = generation
            hardReset()
        }

        val previous = lastFix
        val previousProgress = lastProgressM

        // **والمرفوضةُ لا تضيف ولا تخصم ولا تُعيد حالاً.**
        if (grade == FixGrade.REJECTED) return verdict(Skip.REJECTED, null)

        lastFix = fix
        if (progress != null) lastProgressM = progress.progressM

        if (route == null || !route.usable || progress == null) {
            return verdict(Skip.NO_ROUTE, null)
        }
        // **وخارجَ الممرّ ليست حالَنا** — أمرُ المالك، البند ٣.
        if (progress.offRouteM > tuning.corridorM) return verdict(Skip.OUTSIDE_CORRIDOR, null)

        val speed = fix.speedMps
        // **والواقفُ يُجمَّد ولا يُمحى.**
        if (speed != null && speed < tuning.stationaryMps) return verdict(Skip.STATIONARY, null)

        // **وسياقُ المناورة يُدير الاتّجاهَ بحقّ** — فيُجمَّد.
        val suppress = suppression(route, progress.progressM)
        if (suppress) return verdict(Skip.MANEUVER, null)

        val segment = segmentBearing(route, progress.progressM)
        val trusted = headingDeg != null && segment != null &&
            grade == FixGrade.ACCEPTED &&
            speed != null && speed >= tuning.minTrustedSpeedMps
        val degradedTrusted = headingDeg != null && segment != null &&
            grade == FixGrade.DEGRADED &&
            speed != null && speed >= tuning.minTrustedSpeedMps
        if (!trusted && !degradedTrusted) return verdict(Skip.NO_BEARING, null)

        val delta = GpsQuality.angleDelta(headingDeg!!, segment!!)
        val opposite = abs(delta) >= tuning.oppositeDeg
        val dtMs = if (previous == null) 0L else (fix.atMs - previous.atMs).coerceAtLeast(0L)
        val moved = if (previous == null) 0.0 else GpsQuality.metersBetween(previous, fix)

        if (!previousProgress.isNaN() && progress.progressM < previousProgress) regressed = true

        if (opposite) {
            oppositeMs += dtMs
            oppositeM += moved
            forwardMs = 0L
            forwardM = 0.0
        } else {
            forwardMs += dtMs
            forwardM += moved
        }
        if (state == State.WRONG_WAY) {
            confirmedAtMs += dtMs
            confirmedTravelM += moved
        }

        advance(grade, fix.atMs)
        return verdict(Skip.NONE, delta)
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **الدخولُ بالزمن والمسافة — والخروجُ بدليلٍ أماميّ**
     * ══════════════════════════════════════════════════════════════════
     *
     * **ولا تحسم المتدهورةُ وحدَها** (البند ٩): التأكيدُ يحتاج قراءةً
     * مقبولةً في اللحظة نفسِها. **ودقّةُ خمسين متراً تجعل الاتّجاهَ
     * المحسوبَ يدور مع الضجيج**، وهي دعوى ثقيلةٌ لا تُبنى عليه.
     */
    private fun advance(grade: FixGrade, nowMs: Long) {
        // **والعودةُ أوّلاً** — سيرٌ أماميٌّ موثوقٌ يُنهي النوبة.
        if (forwardMs >= tuning.recoverMs && forwardM >= tuning.recoverM) {
            if (state != State.CORRECT_DIRECTION) endEpisode()
            return
        }
        val confirmMs = if (regressed) tuning.confirmMsWithRegression else tuning.confirmMs
        val confirmM = if (regressed) tuning.confirmMWithRegression else tuning.confirmM
        when {
            oppositeMs >= confirmMs && oppositeM >= confirmM && grade == FixGrade.ACCEPTED -> {
                if (state != State.WRONG_WAY) {
                    state = State.WRONG_WAY
                    confirmations++
                    episode++
                    confirmedAtMs = 0L
                    confirmedTravelM = 0.0
                }
            }
            oppositeMs >= tuning.suspectMs && oppositeM >= tuning.suspectM -> {
                if (state == State.CORRECT_DIRECTION) {
                    state = State.SUSPECTED_WRONG_WAY
                    suspicions++
                }
            }
        }
    }

    /**
     * **أمضت نافذةُ التصحيح؟** — زمناً أو مسافة.
     *
     * (قرارُ المالك، البند ١٩: **«لا Reroute فورَ أوّل Confirmation»**.)
     *
     * **ومن دخل شارعاً معاكساً طويلاً لن يُصلحه الدوران** — المسارُ
     * الصحيحُ صار غيرَه. **ومن التفّ في موقفٍ يعود قبل أن تمضي.**
     */
    fun correctionWindowElapsed(): Boolean =
        state == State.WRONG_WAY &&
            (confirmedAtMs >= tuning.correctionWindowMs || confirmedTravelM >= tuning.correctionTravelM)

    /**
     * **كبتُ سياق المناورة.**
     *
     * **ويُحسب من المسار وموضع التقدّم** — بالطريقة التي صحّحت بها
     * سماحَ المناورة في معايرة ٣أ، **فلا يُمسّ `RouteProgress`.**
     *
     * **والدورانُ المطلوبُ أخطرُ ما يولّد إنذاراً كاذباً**: المسارُ
     * نفسُه يطلب مئةً وثمانين درجة.
     */
    private fun suppression(route: NavRoute, progressM: Double): Boolean {
        for (m in route.maneuvers) {
            val d = abs(m.atDistanceM - progressM)
            val window = when {
                m.kind == ManeuverKinds.U_TURN -> tuning.uTurnSuppressM
                m.isRoundabout -> tuning.roundaboutSuppressM
                m.kind == ManeuverKinds.SHARP_LEFT || m.kind == ManeuverKinds.SHARP_RIGHT ->
                    tuning.sharpSuppressM
                else -> continue
            }
            if (d <= window) return true
        }
        return false
    }

    /** **اتّجاهُ القطعة التي يقف عليها التقدّم** — كما في المرحلة ٣أ. */
    private fun segmentBearing(route: NavRoute, meters: Double): Float? {
        val g = route.geometry
        if (g.size < 2) return null
        val i = minOf(g.size - 2, RouteProjector.indexAtOrBefore(route.cumulativeM, meters))
        return GpsQuality.courseBetween(
            NavFix(g[i].lat, g[i].lng, 0f, null, null, 0),
            NavFix(g[i + 1].lat, g[i + 1].lng, 0f, null, null, 0),
        )
    }

    private fun endEpisode() {
        state = State.CORRECT_DIRECTION
        oppositeMs = 0L
        oppositeM = 0.0
        regressed = false
        confirmedAtMs = 0L
        confirmedTravelM = 0.0
    }

    private fun verdict(skip: Skip, delta: Float?) = Verdict(
        state = state,
        bearingDeltaDeg = delta,
        oppositeMs = oppositeMs,
        oppositeM = oppositeM,
        progressRegressed = regressed,
        skip = skip,
        episode = episode,
    )

    private fun hardReset() {
        state = State.CORRECT_DIRECTION
        oppositeMs = 0L
        oppositeM = 0.0
        forwardMs = 0L
        forwardM = 0.0
        regressed = false
        lastFix = null
        lastProgressM = Double.NaN
        confirmedAtMs = 0L
        confirmedTravelM = 0.0
    }

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
        bearingDeltaDeg = null,
        oppositeMs = 0L,
        oppositeM = 0.0,
        progressRegressed = false,
        skip = Skip.NO_ROUTE,
        episode = episode,
    )

    fun reset() {
        hardReset()
        episode = 0L
        lastGeneration = -1L
        suspicions = 0
        confirmations = 0
    }
}
