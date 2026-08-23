package com.rahalgo.navigation

import kotlin.math.abs

/**
 * **حكمُ الخادم — محايدٌ تماماً.**
 *
 * **ولا يعرف الجوّالُ من أين جاء**: لا محرّكَ ولا عقدَ ولا ثقةَ خام
 * (البند ١٤).
 */
enum class RoadCorrelation {
    /** **على المسار المخطَّط.** */
    ON_PLANNED_ROUTE,

    /** **على طريقٍ آخر.** */
    PARALLEL_ROUTE,

    /** **لا يُحسم.** */
    AMBIGUOUS,

    /** **لا سياقَ أو لا شبكةَ أو لا هويّةَ مسار.** */
    INSUFFICIENT_DATA,
}

/** **ما يُرسَل إلى الخادم** — قراءاتٌ ومعرّفُ مسارٍ لا غير. */
data class CorrelationRequest(
    val routeId: String,
    val fixes: List<NavFix>,
)

/**
 * ══════════════════════════════════════════════════════════════════
 * **مُحلِّلُ الطريق الموازي — آلةُ حالٍ لا تُقرّر بنفسها**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٨ب، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 *     NO_SUSPICION → SUSPECTED → RESOLVING_FIRST
 *                  → WAITING_FRESH_FIX → RESOLVING_SECOND
 *                  → CONFIRMED_PARALLEL | AMBIGUOUS | INSUFFICIENT_DATA
 *                  → COOLDOWN → NO_SUSPICION
 *
 * # ولا إعادةَ حسابٍ من ردّ شبكة
 *
 * **البند ٢١**: الردُّ يُغيّر حالَ المُحلِّل، **والحالُ وحدَها تُنتج
 * سببَ إعادةِ الحساب** — كما تُنتج `NavState` أسبابَها.
 *
 * # والرايةُ مطفأةٌ افتراضاً
 *
 * **البند ١**: المعايرةُ تقوم على فرضٍ لم يُقَس في الرقّة — **أنّ
 * الدقّةَ المبلَّغةَ تقارب الخطأَ الحقيقيّ.** وحساسيّةُ التصميم له
 * قاتلة: عند ١٠م الاشتباهُ الكاذبُ ٢٢٪ من الرحلات، **وعند ١٥م
 * يصير ٥٦٪.**
 */
class ParallelResolver(
    private val suspicion: ParallelSuspicion = ParallelSuspicion(),
    private val tuning: Tuning = Tuning(),
    /** **الرايةُ — والإطفاءُ يعني: لا نداءَ ولا حالَ ولا أثر.** */
    var enabled: Boolean = false,
) {

    data class Tuning(
        /** **تبريدُ نوبةٍ جديدة.** */
        val cooldownMs: Long = COOLDOWN_MS,

        /**
         * **وتقدّمٌ جديدٌ يلزم لفتح نوبةٍ بعد التباس** — البند ٢٣.
         *
         * **والتبريدُ وحدَه لا يكفي**: سائقٌ يبقى في ممرٍّ ملتبسٍ
         * عشرَ دقائقَ **ينتج نداءين كلَّ ثلاثين ثانية** — وذلك
         * استجوابٌ دوريٌّ لا تحليل.
         */
        val newProgressM: Double = NEW_PROGRESS_M,

        /** **أو تبدّلٌ معتبَرٌ في الإزاحة.** */
        val newOffsetM: Double = NEW_OFFSET_M,

        /**
         * **وسقفٌ صلبٌ للنوبات في الجيل الواحد** — البند ٢٤.
         *
         * **حارسٌ أخيرٌ ضدّ عيبٍ لا منطقٌ أساسيّ**: بعده يبقى
         * المُحلِّلُ ملتبساً حتّى جيلٍ جديدٍ أو وجهةٍ جديدة.
         */
        val maxEpisodesPerGeneration: Int = MAX_EPISODES,
    )

    enum class State {
        NO_SUSPICION,
        SUSPECTED,
        RESOLVING_FIRST,
        WAITING_FRESH_FIX,
        RESOLVING_SECOND,
        CONFIRMED_PARALLEL,
        AMBIGUOUS,
        INSUFFICIENT_DATA,
        COOLDOWN,
    }

    var state: State = State.NO_SUSPICION
        private set

    /** **عددُ النوبات في هذا الجيل** — للسقف الصلب. */
    var episodes: Int = 0
        private set

    private var generation: Long = -1L
    private var cooldownUntilMs: Long = 0L
    private var lastResolvedProgressM: Double = Double.NaN
    private var lastResolvedOffsetM: Double = Double.NaN
    private var firstVerdict: RoadCorrelation? = null
    private var firstAtMs: Long = 0L

    /**
     * **يُغذّى كلَّ قراءة.**
     *
     * @return طلبٌ يجب إرساله الآن، أو `null`.
     */
    fun onFix(
        fix: NavFix,
        grade: FixGrade,
        hit: RouteProgress.State?,
        nav: NavState?,
        routeId: String?,
        routeGeneration: Long,
    ): CorrelationRequest? {
        if (!enabled) {
            hardReset(routeGeneration)
            return null
        }
        // ══════════════════════════════════════════════════════════
        // **والأولويّةُ للكواشف الحاسمة — البندان ٢٧ و٢٩**
        // ══════════════════════════════════════════════════════════
        //
        // **من حُسم أمرُه لا يُستجوَب**: `OFF_ROUTE` و`WRONG_WAY`
        // قرارٌ قائم، **وإعادةُ الحساب جاريةٌ فلا معنى لمقارنةٍ
        // بمسارٍ يُستبدل.**
        //
        // **وآخرُ ميلٍ انتهى فيه الإرشاد** — فلا مسارَ يُقارَن به.
        if (nav != null && !usable(nav)) {
            softReset()
            return null
        }
        if (routeGeneration != generation) {
            hardReset(routeGeneration)
        }
        if (episodes >= tuning.maxEpisodesPerGeneration) {
            state = State.AMBIGUOUS
            return null
        }

        val lateral = hit?.lateralSignedM ?: 0.0
        val s = suspicion.onFix(fix, grade, lateral)

        when (state) {
            State.COOLDOWN -> {
                if (fix.atMs < cooldownUntilMs) return null
                if (!materialNewEvidence(hit)) return null
                state = State.NO_SUSPICION
            }

            State.CONFIRMED_PARALLEL -> return null

            State.WAITING_FRESH_FIX -> {
                // ══════════════════════════════════════════════════
                // **ودليلٌ طازجٌ واحدٌ يكفي — البندان ١٢ و١٣**
                // ══════════════════════════════════════════════════
                //
                // **ولا ندّعي استقلالاً إحصائيّاً**: الخطأُ مترابطٌ
                // τ≈٣٠ث، **ونافذتان متجاورتان تتقاسمان انحيازَه.**
                //
                // **والقياسُ قال إنّ التكرارَ لا يشتري شيئاً**: صفرُ
                // حالةٍ أخطأ فيها الأوّلُ ثمّ الثاني — **لأنّ الأوّلَ
                // لم يُخطئ.** فهذه **مراجعةُ اتّساقٍ بدليلٍ أحدث**
                // لا تأكيدٌ مستقلّ.
                if (s != ParallelSuspicion.State.SUSPECTED) {
                    finish(State.AMBIGUOUS, hit, fix)
                    return null
                }
                if (suspicion.newestAtMs() <= firstAtMs) return null
                state = State.RESOLVING_SECOND
                return request(routeId, hit, fix)
            }

            State.RESOLVING_FIRST, State.RESOLVING_SECOND -> return null

            else -> Unit
        }

        if (s != ParallelSuspicion.State.SUSPECTED) {
            if (state == State.SUSPECTED) state = State.NO_SUSPICION
            return null
        }
        if (state == State.NO_SUSPICION) {
            // **والقراءاتُ التي أنتجت الاشتباهَ هي نافذةُ المطابقة
            // الأولى** — البند ١١: لا تُجمع ثمانٍ مرّتين.
            state = State.RESOLVING_FIRST
            episodes++
            firstVerdict = null
            firstAtMs = suspicion.newestAtMs()
            return request(routeId, hit, fix)
        }
        return null
    }

    /** **يُسلَّم حكمَ الخادم.** */
    fun onResult(result: RoadCorrelation, hit: RouteProgress.State?, fix: NavFix?) {
        if (!enabled) return
        when (state) {
            State.RESOLVING_FIRST -> when (result) {
                RoadCorrelation.PARALLEL_ROUTE -> {
                    firstVerdict = result
                    state = State.WAITING_FRESH_FIX
                }
                RoadCorrelation.INSUFFICIENT_DATA -> finish(State.INSUFFICIENT_DATA, hit, fix)
                else -> finish(State.AMBIGUOUS, hit, fix)
            }

            State.RESOLVING_SECOND -> {
                if (result == firstVerdict && result == RoadCorrelation.PARALLEL_ROUTE) {
                    state = State.CONFIRMED_PARALLEL
                    lastResolvedProgressM = hit?.progressM ?: Double.NaN
                    lastResolvedOffsetM = hit?.lateralSignedM ?: Double.NaN
                } else if (result == RoadCorrelation.INSUFFICIENT_DATA) {
                    finish(State.INSUFFICIENT_DATA, hit, fix)
                } else {
                    finish(State.AMBIGUOUS, hit, fix)
                }
            }

            else -> Unit
        }
    }

    /**
     * **أثبتَ أنّه على طريقٍ موازٍ؟**
     *
     * **وهذا وحدَه سببُ إعادةِ حساب** (البند ٢١) — **ولا يُنادى من
     * ردّ شبكةٍ مباشرةً.**
     */
    val confirmedParallel: Boolean get() = enabled && state == State.CONFIRMED_PARALLEL

    private fun usable(nav: NavState): Boolean {
        if (nav.reroute != RerouteStatus.NONE) return false
        if (nav.arrivalPhase != ArrivalPhase.EN_ROUTE) return false
        return nav.situation == NavSituation.ON_ROUTE
    }

    private fun request(routeId: String?, hit: RouteProgress.State?, fix: NavFix): CorrelationRequest? {
        // **ولا هويّةَ فلا سؤال** — البند ١٩: لا ارتدادَ إلى مسارٍ مضى.
        if (routeId.isNullOrEmpty()) {
            finish(State.INSUFFICIENT_DATA, hit, fix)
            return null
        }
        val ev = suspicion.evidence()
        if (ev.size < MIN_FIXES) {
            finish(State.INSUFFICIENT_DATA, hit, fix)
            return null
        }
        return CorrelationRequest(routeId, ev)
    }

    private fun finish(next: State, hit: RouteProgress.State?, fix: NavFix?) {
        state = next
        lastResolvedProgressM = hit?.progressM ?: Double.NaN
        lastResolvedOffsetM = hit?.lateralSignedM ?: Double.NaN
        cooldownUntilMs = (fix?.atMs ?: 0L) + tuning.cooldownMs
        suspicion.reset()
        if (next != State.CONFIRMED_PARALLEL) state = State.COOLDOWN
    }

    /**
     * **دليلٌ جديدٌ معتبَرٌ — البند ٢٣.**
     *
     * **وبلاه يصير التبريدُ ساعةَ استجواب**: كلُّ ثلاثين ثانيةً
     * نداءان على الموقف نفسِه بلا معلومةٍ جديدة.
     */
    private fun materialNewEvidence(hit: RouteProgress.State?): Boolean {
        if (hit == null) return false
        if (lastResolvedProgressM.isNaN()) return true
        if (abs(hit.progressM - lastResolvedProgressM) >= tuning.newProgressM) return true
        if (!lastResolvedOffsetM.isNaN() &&
            abs(hit.lateralSignedM - lastResolvedOffsetM) >= tuning.newOffsetM
        ) {
            return true
        }
        return false
    }

    private fun softReset() {
        suspicion.reset()
        if (state != State.COOLDOWN && state != State.CONFIRMED_PARALLEL) {
            state = State.NO_SUSPICION
        }
    }

    private fun hardReset(gen: Long) {
        generation = gen
        episodes = 0
        state = State.NO_SUSPICION
        cooldownUntilMs = 0L
        lastResolvedProgressM = Double.NaN
        lastResolvedOffsetM = Double.NaN
        firstVerdict = null
        firstAtMs = 0L
        suspicion.reset()
    }

    companion object {
        const val COOLDOWN_MS = 30_000L
        const val NEW_PROGRESS_M = 300.0
        const val NEW_OFFSET_M = 15.0
        const val MAX_EPISODES = 4
        const val MIN_FIXES = 4
    }
}
