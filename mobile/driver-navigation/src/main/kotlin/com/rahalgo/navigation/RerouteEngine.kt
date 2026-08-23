package com.rahalgo.navigation

import kotlin.math.min

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إعادةُ الحساب — فعلُ شبكةٍ لا دليلُ ملاحة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٣ب، أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
 *
 *     IDLE  ──خروجٌ مؤكَّد──▶  REQUESTING  ──✓──▶  تركيبٌ ذرّيّ  ──▶  IDLE
 *                                  │
 *                                  ✗
 *                                  ▼
 *                             COOLDOWN  ──مهلةٌ ومعنىً للمحاولة──▶  IDLE
 *
 * # ولا يعرف هذا المحرّكُ «خارجَ المسار»
 *
 * (أمرُ المالك: «لا تخلط `OFF_ROUTE` مع `REROUTING`. الأوّلُ Navigation
 *  evidence والثاني Network action».)
 *
 * **يُعطى حالاً ويقرّر فعلاً** — ولا يحكم على طريقٍ ولا على قراءة.
 *
 * # ومرورُ الوقت وحدَه لا يكفي
 *
 * (أمرُ المالك، البند ٩: «لكن لا تجعل مرورَ الوقت وحدَه كافياً
 *  دائماً».)
 *
 * **ولا كلُّ إخفاقٍ سواء**: **شبكةٌ انقطعت تُعاد المحاولةُ فيها بمجرّد
 * عودتها** ولو كان السائقُ واقفاً — **ومسارٌ ردّه المحرّكُ غيرَ صالحٍ
 * لا تُصلحه إعادةُ الطلب من الموضع نفسِه**، فيُطلب معه معنىً: حركةٌ
 * ذاتُ بال أو مهلةٌ أطول. (البند ١٠.)
 */
class RerouteEngine(
    private val source: RouteSource,
    val tuning: Tuning = Tuning(),
    /**
     * **يُتحقَّق من المسار الجديد ويُركَّب** — ويردّ ما وقع.
     *
     * **والتحقّقُ ليس ترفاً** (البند ١٦): **مسارٌ يترك السائقَ خارجَه
     * فورَ تركيبه يُطلق حلقةً** — `install → OFF_ROUTE → reroute`.
     */
    private val installer: (NavRoute, NavFix) -> InstallOutcome,
    /** **جيلُ المسار الحاليّ** — يُقرأ لحظةَ الطلب ولحظةَ الجواب. */
    private val generationOf: () -> Long,
) {

    data class Tuning(
        /** **سلّمُ التهدئة** بالملّي — ولا يتجاوز آخرَها. */
        val cooldownStepsMs: List<Long> = listOf(5_000L, 10_000L, 20_000L, 30_000L),

        /**
         * **وحركةٌ ذاتُ معنىً لإعادة المحاولة** بعد إخفاقٍ ليس شبكيّاً.
         *
         * **وخمسون متراً أربعةُ أضعاف خليّة المخبأ** (قِيست: ١١م
         * شمالاً و٩م شرقاً) — **فطلبٌ من داخل الخليّة نفسِها يعود
         * بالجواب المخزَّن نفسِه**، وهي إعادةٌ بلا فائدة.
         */
        val minRetryMoveM: Double = 50.0,

        /**
         * **أو مهلةٌ أطول تُغني عن الحركة.**
         *
         * (أمرُ المالك: «إذا الشبكةُ عادت والسائقُ ما زال واقفاً
         *  OFF_ROUTE، يجب ألّا نمنع Reroute إلى الأبد لمجرّد أنّه لم
         *  يتحرّك ٥٠م».)
         */
        val unusableRetryMs: Long = 60_000L,

        /**
         * **وأقصى بعدٍ يُقبل بين السائق وبداية المسار الجديد.**
         *
         * **وضِعفُ الأساس** — ومسارٌ يبدأ أبعدَ من ذلك يترك السائقَ
         * خارجَه لحظةَ تركيبه.
         */
        val maxStartOffsetM: Double = 60.0,
    )

    enum class Phase { IDLE, REQUESTING, COOLDOWN }

    var phase = Phase.IDLE
        private set

    var status = RerouteStatus.NONE
        private set

    var lastFailure: RerouteFailure? = null
        private set

    /** **سببُ آخر طلب** — يُقرأ في التشخيص والاختبار. */
    var lastReason: RerouteReason? = null
        private set

    /** **عدّاداتُ القياس** — تُقرأ في التقرير والاختبار. */
    var requests = 0
        private set
    var installs = 0
        private set
    var rejected = 0
        private set

    private var seq = 0L
    private var pendingSeq = -1L
    private var pendingGeneration = -1L
    private var attempts = 0
    private var cooldownUntilMs = 0L
    private var failedAtMs = 0L
    private var lastRequestFix: NavFix? = null

    /** **لحظةُ إطلاق آخر طلبٍ ولحظةُ تركيب آخر مسار** — للقياس. */
    var lastRequestAtMs = 0L
        private set
    var lastInstallAtMs = 0L
        private set

    /**
     * **يُنظَر في الحال بعد كلّ قراءة** — فيُطلق طلباً أو لا يفعل.
     *
     * **ولا يُطلق شيئاً وهو في `REQUESTING`** (البند ٨): **عشرون قراءةً
     * خارجَ المسار وطلبٌ واحد.**
     */
    fun consider(fix: NavFix, state: NavState, nowMs: Long): RerouteStatus =
        consider(fix, state.rerouteReason, nowMs)

    /**
     * **ويُنظَر في الحال بسببٍ معلن.**
     *
     * (المرحلة ٥، أمرُ المالك، البند ٢١: «لا تنشئ محرّكاً جديداً…
     *  `RerouteReason.OFF_ROUTE` و`RerouteReason.WRONG_WAY`».)
     *
     * **وحرّاسُ ٣ب كلُّها تبقى**: طلبٌ واحد · تهدئة · جيل · جوابٌ
     * متأخّر · استبدالٌ ذرّيّ · تحقّقٌ من الجديد.
     */
    fun consider(fix: NavFix, reason: RerouteReason?, nowMs: Long): RerouteStatus {
        // ══════════════════════════════════════════════════════════════
        // **والعودةُ إلى المسار تُلغي كلَّ شيء**
        // ══════════════════════════════════════════════════════════════
        //
        // **حتّى وطلبٌ في الطريق** (البند ١٥ من التحليل): **جوابٌ يصل
        // بعد أن عاد السائقُ لا يستبدل مساراً صالحا.** والإلغاءُ هنا،
        // **والطرحُ عند الوصول** — انظر `onReply`.
        if (reason == null) {
            if (phase == Phase.REQUESTING) {
                pendingSeq = -1L
                lastFailure = RerouteFailure.CANCELLED
            }
            phase = Phase.IDLE
            attempts = 0
            status = RerouteStatus.NONE
            return status
        }

        if (phase == Phase.REQUESTING) return status
        if (phase == Phase.COOLDOWN && !mayRetry(fix, nowMs)) return status

        lastReason = reason
        fire(fix, nowMs)
        return status
    }

    /**
     * **أيجوز أن نعيد المحاولة الآن؟**
     *
     * **والمهلةُ شرطٌ لا يكفي** — انظر أعلى الملفّ.
     */
    private fun mayRetry(fix: NavFix, nowMs: Long): Boolean {
        if (nowMs < cooldownUntilMs) return false
        return when (lastFailure) {
            // **الشبكةُ تعود فتُعاد المحاولةُ ولو كان واقفا** — فما
            // منعنا كان الشبكةَ لا الموضع.
            RerouteFailure.NETWORK, RerouteFailure.TIMEOUT -> true
            // **وما عداها يحتاج معنىً**: موضعٌ جديدٌ أو صبرٌ أطول.
            else -> movedEnough(fix) || (nowMs - failedAtMs) >= tuning.unusableRetryMs
        }
    }

    private fun movedEnough(fix: NavFix): Boolean {
        val prev = lastRequestFix ?: return true
        return GpsQuality.metersBetween(prev, fix) >= tuning.minRetryMoveM
    }

    private fun fire(fix: NavFix, nowMs: Long) {
        seq++
        pendingSeq = seq
        pendingGeneration = generationOf()
        lastRequestFix = fix
        lastRequestAtMs = nowMs
        phase = Phase.REQUESTING
        status = RerouteStatus.REROUTING
        requests++
        val mySeq = seq
        val myGen = pendingGeneration
        source.request(fix.lat, fix.lng, mySeq) { reply -> onReply(mySeq, myGen, reply, fix) }
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **والجوابُ يُقاس بطلبه وبجيله معاً**
     * ══════════════════════════════════════════════════════════════════
     *
     * (البندان ١٧ و١٨.)
     *
     * **رقمُ الطلب** يقطع الأجوبةَ المتأخّرةَ لطلبٍ أُلغي.
     * **وجيلُ المسار** يقطع جواباً بُني على مسارٍ صار قديماً —
     * **ومنه تغيّرُ طور الرحلة**: من طلب طريقاً إلى المتجر ثمّ استلم،
     * **جوابُه القديمُ لا يستبدل طريقاً إلى الزبون.** (البند ١٩.)
     *
     * **ووقتُ الجواب يُقرأ من `at` نفسِها** — لا ساعةَ جدارٍ هنا.
     */
    fun onReply(replySeq: Long, replyGeneration: Long, reply: RouteReply, at: NavFix) {
        if (replySeq != pendingSeq || replyGeneration != generationOf()) {
            // **ولا عقوبةَ على السباق** — لم يُخفق شيء.
            lastFailure = RerouteFailure.STALE_RESPONSE
            if (phase == Phase.REQUESTING && replySeq == pendingSeq) {
                phase = Phase.IDLE
                status = RerouteStatus.NONE
            }
            return
        }
        pendingSeq = -1L
        when (reply) {
            is RouteReply.Failed -> fail(reply.reason, at.atMs)
            is RouteReply.Ok -> when (installer(reply.route, at)) {
                InstallOutcome.INSTALLED -> {
                    installs++
                    lastInstallAtMs = at.atMs
                    attempts = 0
                    phase = Phase.IDLE
                    status = RerouteStatus.NONE
                    lastFailure = null
                }
                InstallOutcome.INVALID -> { rejected++; fail(RerouteFailure.INVALID_ROUTE, at.atMs) }
                InstallOutcome.UNUSABLE -> { rejected++; fail(RerouteFailure.INVALID_ROUTE, at.atMs) }
            }
        }
    }

    private fun fail(reason: RerouteFailure, nowMs: Long) {
        lastFailure = reason
        failedAtMs = nowMs
        phase = Phase.COOLDOWN
        status = RerouteStatus.REROUTE_FAILED
        val step = tuning.cooldownStepsMs[min(attempts, tuning.cooldownStepsMs.size - 1)]
        attempts++
        cooldownUntilMs = nowMs + step
    }

    /** **مهلةُ التهدئة الباقيةُ بالملّي** — للتشخيص. */
    fun cooldownLeftMs(nowMs: Long): Long = (cooldownUntilMs - nowMs).coerceAtLeast(0L)

    /** **يُنسى كلُّ شيء** — عند جلسةٍ جديدةٍ أو مسارٍ يُسلَّم من خارج. */
    fun reset() {
        phase = Phase.IDLE
        status = RerouteStatus.NONE
        lastFailure = null
        pendingSeq = -1L
        pendingGeneration = -1L
        attempts = 0
        cooldownUntilMs = 0L
        failedAtMs = 0L
        lastRequestFix = null
    }
}
