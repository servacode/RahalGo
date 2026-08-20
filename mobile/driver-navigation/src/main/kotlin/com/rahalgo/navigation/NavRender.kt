package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما تُعطاه الخريطةُ لترسم خطوةَ ملاحة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ١، بأمر المالك ٢٠٢٦-٠٨-٢٠.)
 *
 * # ولماذا صنفٌ بين المنطق والرسم
 *
 * **`NavPipeline.Step` تحمل التشخيصَ أيضاً** — سببَ الرفض وعدّادات.
 * **والخريطةُ لا شأنَ لها بذلك**، فتُعطى ما ترسمه وحدَه.
 *
 * # و`stepId` هو ما يمنع إعادةَ الحركة
 *
 * **`AndroidView` تُنادى مع كلّ رسمٍ لأيّ سبب** — دورانُ جهازٍ، أو
 * تبدّلُ حقلٍ آخرَ في الشاشة. **ومن بدأ الحركةَ في كلّ نداءٍ أعادها من
 * أوّلها فتجمّدت الأيقونةُ في مكانها ولم تصل أبدا.**
 *
 * **ورقمٌ يزيد مع كلّ قراءةٍ مقبولةٍ أصدقُ إشارةٍ على «هذه جديدة»** —
 * وهو الأسلوبُ نفسُه الذي يعمل به زرُّ «ردّني إلى موضعي».
 */
data class NavRender(
    val stepId: Long,
    val targetLat: Double?,
    val targetLng: Double?,
    val bearingDeg: Float?,
    val durationMs: Long,
) {
    /**
     * **حالُ الكاميرا لهذه الخطوة.**
     *
     * **والقرارُ هنا لا في `map-core`** — أمرُ المالك: «قرار الملاحة
     * يبقى داخل `driver-navigation`».
     */
    fun camera(currentMapBearing: Float): NavCameraState = NavCamera.follow(
        lat = targetLat ?: 0.0,
        lng = targetLng ?: 0.0,
        bearingDeg = bearingDeg,
        currentBearingDeg = currentMapBearing,
        durationMs = durationMs,
    )

    companion object {
        /**
         * **يبني ما يُرسم من خطوةٍ ومعرّفها.**
         *
         * **والمرفوضةُ تردّ فارغاً** — فلا تُحرَّك الأيقونةُ ولا
         * الكاميرا.
         */
        fun of(step: NavPipeline.Step, stepId: Long): NavRender? {
            if (step.targetLat == null || step.targetLng == null) return null
            return NavRender(
                stepId = stepId,
                targetLat = step.targetLat,
                targetLng = step.targetLng,
                bearingDeg = step.bearingDeg,
                durationMs = step.animationMs,
            )
        }
    }
}
