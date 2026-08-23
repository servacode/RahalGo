package com.rahalgo.navigation

import kotlin.math.max
import kotlin.math.min

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تقدّمُ السائق على المسار — حالٌ واحدةٌ تقرؤها الشاشة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٢، أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
 *
 *     أين هو؟ · كم قطع؟ · كم بقي؟ · كم بقي زمناً؟
 *     ما المناورةُ الآن؟ · وما التالية؟ · وكم إليها؟
 *
 * # ولا نداءَ شبكةٍ في الحلقة
 *
 * **كلُّ ما هنا حسابٌ محلّيّ** — والمسارُ يُجلب مرّةً ويُسقَط عليه ألفُ
 * قراءة.
 *
 * # والتقدّمُ ليس رتيباً مطلقاً
 *
 * (أمرُ المالك: «لا تجعل Progress Monotonic مطلقاً، لأنّ السائقَ قد
 *  يرجع فعليّاً».)
 *
 * **ورجفةُ GPS ليست رجوعاً**: تراجعٌ صغيرٌ يُبتلع، **وتراجعٌ كبيرٌ
 * يُصدَّق بعد أن يتكرّر.** فمن رجع فعلاً يُرى، ومن ارتجّ لا يُرى.
 */
class RouteProgress(private val route: NavRoute) {

    /** **حالُ اللحظة** — ما تعرضه الشاشة. */
    data class State(
        val progressM: Double,
        val remainingM: Double,
        val remainingSec: Double,
        val offRouteM: Double,

        /**
         * **الإزاحةُ الموقّعة** — موجبٌ يسارَ السير، سالبٌ يمينَه.
         *
         * **تمريرٌ لا حساب** (المرحلة ٨ب): [RouteProjector] يحسبها
         * أصلاً **وكانت تُرمى.** ولا حكمَ هنا يتبدّل بها.
         */
        val lateralSignedM: Double = 0.0,
        val fraction: Double,
        /** **المناورةُ الجاريةُ الآن** — وفارغةٌ قبل أوّل إسقاط. */
        val current: NavManeuver?,
        /** **التالية** — وفارغةٌ عند آخر المسار. */
        val next: NavManeuver?,
        /** **كم متراً إلى المناورة الجارية.** */
        val distanceToManeuverM: Double,
        /**
         * **أتُعرض المناورتان معاً؟**
         *
         * (تصحيحُ المالك: «مسافةُ ٥٠م خاصّةٌ بطريقة العرض فقط. **لا
         *  تستخدمها للحكم على أنّ المناورة الحالية انتهت**».)
         */
        val showNextTogether: Boolean,
        val nearDestination: Boolean,
        val arrived: Boolean,
    )

    private var progressM = 0.0
    private var maneuverIdx = 0
    private var lastFixMs = 0L
    private var backCount = 0
    private var arriveCount = 0
    private var started = false

    /** **آخرُ تقدّمٍ مقيس** — للتشخيص. */
    val currentProgressM: Double get() = progressM

    /**
     * **يُغذّى قراءةً مقبولةً فيردّ الحال.**
     *
     * **ولا تُغذّى المرفوضة** — فالمرشِّحُ قبله (`NavPipeline`).
     */
    fun onFix(fix: NavFix, headingDeg: Float?): State? {
        if (!route.usable) return null
        val dt = if (lastFixMs == 0L) 0.0 else (fix.atMs - lastFixMs) / 1000.0
        lastFixMs = fix.atMs

        // **وأوّلُ إسقاطٍ يمسح المسارَ كلَّه** — انظر `RouteProjector`:
        // **من فتح التطبيقَ في منتصف الطريق لا يبقى سهمُه عند
        // بدايته.**
        val hit = RouteProjector.project(
            route, fix.lat, fix.lng, progressM, dt, fix.speedMps, headingDeg,
            fullScan = !started,
        ) ?: return null

        progressM = settle(hit.progressM, fix)
        started = true
        advanceManeuvers()
        return snapshot(hit.offRouteM, fix, hit.lateralSignedM)
    }

    /**
     * **يقرّر أيُصدَّق التقدّمُ الجديد.**
     *
     * **وقفزةٌ إلى الأمام أكبرُ ممّا يمكن قطعُه تُقصّ** — لا تُرفض:
     * **رفضُها يجمّد الأيقونةَ، وقصُّها يُبقيها تسير.**
     */
    private fun settle(candidate: Double, fix: NavFix): Double {
        if (!started) return candidate
        val delta = candidate - progressM
        if (delta >= 0) {
            backCount = 0
            return candidate
        }
        // **والتراجعُ الصغيرُ رجفةٌ لا رجوع** — يُبتلع.
        if (-delta <= BACK_TOLERANCE_M + fix.accuracyM) {
            return progressM
        }
        // **والكبيرُ يُصدَّق بعد أن يتكرّر** — فمن التفّ فعلاً يُرى،
        // **ومن قفزت قراءتُه مرّةً لا يُرجَع به.**
        backCount++
        if (backCount >= BACK_CONFIRM) {
            backCount = 0
            return candidate
        }
        return progressM
    }

    /**
     * **يتقدّم بالمناورات بحسب التقدّم على المسار.**
     *
     * (تصحيحُ المالك: «لا تعتمد `progress > atDistanceM + 15m` كشرطٍ
     *  ثابت… **ولا تسمح للـTolerance بأن تبتلع المناورة التالية**».)
     *
     * **وقِيس أنّ بين مناورتين ثمانيةَ أمتارٍ فعلاً** — فسماحٌ ثابتٌ
     * بخمسةَ عشرَ متراً **يبتلع مناورةً كاملة.**
     *
     * **فالسماحُ ديناميكيٌّ محدودٌ بنصف المسافة إلى التالية** — فلا
     * يبلغها أبداً.
     */
    private fun advanceManeuvers() {
        while (maneuverIdx < route.maneuvers.size - 1) {
            val cur = route.maneuvers[maneuverIdx]
            val nxt = route.maneuvers[maneuverIdx + 1]
            val gap = max(0.0, nxt.atDistanceM - cur.atDistanceM)
            // **والسماحُ لا يبلغ نصفَ الفجوة** — فمناورتان بينهما ٨م
            // تُفصلان بأربعة، **ولا تُبتلع الثانية.**
            val tolerance = min(PASS_TOLERANCE_M, gap * 0.5)
            if (progressM >= cur.atDistanceM + tolerance) {
                maneuverIdx++
            } else {
                break
            }
        }
    }

    private fun snapshot(
        offRouteM: Double,
        fix: NavFix,
        lateralSignedM: Double = 0.0,
    ): State {
        val total = route.totalM
        val remaining = max(0.0, total - progressM)
        val cur = route.maneuvers.getOrNull(maneuverIdx)
        val nxt = route.maneuvers.getOrNull(maneuverIdx + 1)

        // **والمسافةُ إلى المناورة تُقاس إلى التالية لا إلى الجارية**:
        // **الجاريةُ هي ما ننفّذه الآن**، ونقطتُها أمامنا — وحين
        // نتجاوزها تصير الجاريةُ هي التالية.
        val target = nxt ?: cur
        val toManeuver = max(0.0, (target?.atDistanceM ?: total) - progressM)

        val gapToNext = if (nxt != null && cur != null) {
            max(0.0, nxt.atDistanceM - (cur.atDistanceM))
        } else {
            Double.MAX_VALUE
        }

        val near = remaining <= max(NEAR_M, 2.0 * fix.accuracyM)
        if (remaining <= max(ARRIVE_M, fix.accuracyM.toDouble())) arriveCount++ else arriveCount = 0

        return State(
            progressM = progressM,
            remainingM = remaining,
            remainingSec = remainingSeconds(),
            offRouteM = offRouteM,
            lateralSignedM = lateralSignedM,
            fraction = if (total <= 0) 0.0 else (progressM / total).coerceIn(0.0, 1.0),
            current = cur,
            next = nxt,
            distanceToManeuverM = toManeuver,
            // **والعرضُ المزدوجُ شأنُ عرضٍ لا شأنُ حكم** — انظر `State`.
            showNextTogether = nxt != null && gapToNext <= SHOW_TOGETHER_M,
            nearDestination = near,
            arrived = arriveCount >= ARRIVE_CONFIRM ||
                (route.totalM > 0 && progressM >= route.totalM - ARRIVE_M &&
                    (fix.speedMps ?: 0f) < 1f),
        )
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **زمنُ الوصول — من مدد المحرّك لا من سرعةِ اللحظة**
     * ══════════════════════════════════════════════════════════════════
     *
     * (تصحيحُ المالك ٢٠٢٦-٠٨-٢٠: «**لا تجعل سرعةَ السائق اللحظيّة
     *  تغيّر ETA مباشرة**… الوقوفُ عند إشارةٍ وGPS speed لحظيٌّ غيرُ
     *  مستقرّ… قد تجعل ETA تتذبذب أو ترتفع بصورةٍ غيرِ واقعيّة».)
     *
     *     الخطواتُ المكتملة   ←  لا تدخل
     *     الخطوةُ الجارية     ←  ما بقي منها بنسبة التقدّم
     *     الخطواتُ التالية    ←  مدّةُ المحرّك كما هي
     *
     * **ومن وقف عند إشارةٍ لا يرتفع زمنُه إلى ساعة** — الطريقُ لم
     * يتبدّل.
     */
    fun remainingSeconds(): Double {
        var sec = 0.0
        for (i in route.maneuvers.indices) {
            val m = route.maneuvers[i]
            val stepEnd = m.atDistanceM + m.stepDistanceM
            if (stepEnd <= progressM) continue // **مكتملةٌ — لا تدخل.**
            if (m.atDistanceM >= progressM) {
                sec += m.stepDurationS // **لم تبدأ بعد.**
                continue
            }
            // **والجاريةُ بنسبتها** — ما بقي منها.
            val left = stepEnd - progressM
            val frac = if (m.stepDistanceM > 0) (left / m.stepDistanceM).coerceIn(0.0, 1.0) else 0.0
            sec += m.stepDurationS * frac
        }
        return sec
    }

    /** **يُنسى كلُّ شيء** — عند مسارٍ جديد. */
    fun reset() {
        progressM = 0.0
        maneuverIdx = 0
        lastFixMs = 0L
        backCount = 0
        arriveCount = 0
        started = false
    }

    companion object {
        /** **تراجعٌ دونه رجفةٌ لا رجوع.** */
        const val BACK_TOLERANCE_M = 12.0

        /** **وكم قراءةً تُصدِّق رجوعاً كبيرا.** */
        const val BACK_CONFIRM = 3

        /** **سقفُ سماحِ تجاوز المناورة** — ويُقصّ بنصف الفجوة. */
        const val PASS_TOLERANCE_M = 12.0

        /** **مسافةُ عرضِ مناورتين معاً** — عرضٌ لا حكم. */
        const val SHOW_TOGETHER_M = 50.0

        const val NEAR_M = 50.0
        const val ARRIVE_M = 25.0

        /**
         * **ولا وصولَ بقراءةٍ واحدة.**
         *
         * **ودقّةُ أربعين متراً تجعل «خمسةَ أمتارٍ» بلا معنى** — فيُطلب
         * تكرار.
         */
        const val ARRIVE_CONFIRM = 3
    }
}
