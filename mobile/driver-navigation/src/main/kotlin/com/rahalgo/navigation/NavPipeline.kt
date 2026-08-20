package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سلسلةُ الملاحة — من قراءةٍ خامٍّ إلى ما يُرسم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ١، بأمر المالك ٢٠٢٦-٠٨-٢٠.)
 *
 *     قراءةٌ خامّة
 *       ↓  GpsQuality      →  مقبولة · متدهورة · مرفوضة
 *       ↓  BearingTracker  →  اتّجاهٌ منعَّمٌ لا يدور عند الوقوف
 *       ↓  PathSmoother    →  كم تدوم حركةُ الأيقونة
 *       ↓  NavCamera       →  أين تنظر الكاميرا
 *     هدفٌ يُرسَم
 *
 * # ولماذا صنفٌ بلا أندرويد
 *
 * **كلُّ قرارٍ في الملاحة هنا** — والجهازُ يعطي قراءاتٍ ويأخذ أهدافاً.
 * **فتُعاد رحلةٌ مسجَّلةٌ كلُّها في اختبارٍ بلا هاتف**، وهو ما تقوم
 * عليه مكتبةُ `navigation-fixtures` في المراحل التالية.
 *
 * **ولا يعرف طلباً ولا سائقاً** — يعرف نقاطاً وأزمنة.
 */
class NavPipeline(
    /** **الفاصلُ المتوقَّع بين قراءتين** — يُستعمل حين لا يُعرف الفعليّ. */
    private val expectedIntervalMs: Long = 1_000L,
    private val bearing: BearingTracker = BearingTracker(),
    /**
     * **مِسْبَرُ التشخيص** — يُنادى بكلّ قراءةٍ بحُكمها.
     *
     * (أمرُ المالك ٢٠٢٦-٠٨-٢٠، البند ٩: «سجّل سبب رفض القراءات في
     *  Debug/Telemetry المناسب **بدون إغراق Logs الإنتاجية**».)
     *
     * **ومِسْبَرٌ يُمرَّر لا سجلٌّ يُكتب هنا**: هذا الصنفُ حسابٌ محضٌ
     * يُختبر بلا أندرويد، **وسطرُ `Log` واحدٌ فيه يربطه بالنظام
     * فيموت الاختبار.**
     *
     * **وفارغٌ يعني لا تشخيص** — وهو الافتراض.
     */
    private val probe: ((NavFix, FixGrade, RejectReason) -> Unit)? = null,
) {

    /** **ما يُرسَم بعد قراءةٍ واحدة.** */
    data class Step(
        val grade: FixGrade,
        val reason: RejectReason,
        /** **الهدفُ الذي تتحرّك إليه الأيقونة** — فارغٌ إن رُفضت القراءة. */
        val targetLat: Double?,
        val targetLng: Double?,
        /** **الاتّجاهُ المعروض** — فارغٌ إن لم يُعرف بعد. */
        val bearingDeg: Float?,
        val animationMs: Long,
    )

    private var lastAccepted: NavFix? = null

    /** **آخرُ قراءةٍ صُدِّقت** — للتشخيص والاختبار. */
    val accepted: NavFix? get() = lastAccepted

    /**
     * **الاتّجاهُ المعروضُ الآن** — يُقرأ ولا يُغيّر شيئا.
     *
     * **وقارئٌ يُفسد ما يقرؤه ليس قارئا**: كتبتُ الاختبارَ أوّلاً
     * يحقن قراءةً وهميّةً ليسأل عن الاتّجاه، **فصارت القراءةُ
     * الوهميّةُ آخرَ ما قُبل** — فرُفض ما بعدها بـ`STALE` وسقط
     * الاختبارُ على مسباره لا على المقيس. (٢٠٢٦-٠٨-٢٠.)
     */
    val headingDeg: Float? get() = bearing.smoothed

    /** **عدّاداتُ التشخيص** — تُقرأ في التقرير لا في الإنتاج. */
    var seen: Int = 0
        private set
    var rejected: Int = 0
        private set
    var degraded: Int = 0
        private set

    /** **ولماذا رُفضت** — عدٌّ لكلّ سبب. (البند ٩.) */
    var rejectedAccuracy: Int = 0
        private set
    var rejectedTeleport: Int = 0
        private set
    var rejectedStale: Int = 0
        private set

    /** **مدى الدقّة المرصود** — ليُعرف أمناسبةٌ حدودُنا للواقع. */
    var worstAccuracyM: Float = 0f
        private set
    var bestAccuracyM: Float = Float.MAX_VALUE
        private set
    private var accuracySum: Double = 0.0

    /** **متوسّطُ الدقّة** — صفرٌ إن لم تصل قراءةٌ بعد. */
    val meanAccuracyM: Double get() = if (seen == 0) 0.0 else accuracySum / seen

    /**
     * **يُغذّى قراءةً فيردّ ما يُرسم.**
     *
     * **والمرفوضةُ تردّ هدفاً فارغاً** — فلا تُحرَّك الأيقونةُ ولا
     * الكاميرا. **وهذا هو «لا تسمح لقراءةٍ سيّئةٍ أن تقفز شارعاً».**
     */
    fun onFix(fix: NavFix): Step {
        seen++
        val previous = lastAccepted
        val (grade, reason) = GpsQuality.grade(fix, previous)
        probe?.invoke(fix, grade, reason)
        if (fix.accuracyM < bestAccuracyM) bestAccuracyM = fix.accuracyM
        if (fix.accuracyM > worstAccuracyM) worstAccuracyM = fix.accuracyM
        accuracySum += fix.accuracyM.toDouble()

        if (grade == FixGrade.REJECTED) {
            rejected++
            when (reason) {
                RejectReason.ACCURACY -> rejectedAccuracy++
                RejectReason.TELEPORT -> rejectedTeleport++
                RejectReason.STALE -> rejectedStale++
                RejectReason.NONE -> Unit
            }
            return Step(grade, reason, null, null, bearing.smoothed, 0L)
        }
        if (grade == FixGrade.DEGRADED) degraded++

        val heading = bearing.update(fix, previous, grade)
        val dtMs = if (previous != null) fix.atMs - previous.atMs else 0L
        val anim = PathSmoother.animationMs(dtMs, expectedIntervalMs, grade)
        lastAccepted = fix
        return Step(grade, reason, fix.lat, fix.lng, heading, anim)
    }

    /**
     * **حالُ الكاميرا لهذه الخطوة** — أو فارغٌ إن لم يكن ثمّة هدف.
     *
     * **ودورانُ الخريطة الحاليُّ يُمرَّر** لأنّ الغيابَ يعني «لا
     * تُدِرها»: انظر `NavCamera.follow`.
     */
    fun cameraFor(step: Step, currentMapBearing: Float): NavCameraState? {
        val lat = step.targetLat ?: return null
        val lng = step.targetLng ?: return null
        return NavCamera.follow(lat, lng, step.bearingDeg, currentMapBearing, step.animationMs)
    }

    /** **يُنسى كلُّ شيء** — عند بدء جلسةٍ جديدة. */
    fun reset() {
        lastAccepted = null
        bearing.reset()
        seen = 0
        rejected = 0
        degraded = 0
        rejectedAccuracy = 0
        rejectedTeleport = 0
        rejectedStale = 0
        worstAccuracyM = 0f
        bestAccuracyM = Float.MAX_VALUE
        accuracySum = 0.0
    }
}
