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
    /**
     * **ما قُطع على الخطّ بالمتر** — ليُطوى خلفَ السائق.
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-٢٤.)
     *
     * **وكان يُحسب ولا يصل إلى الرسم**: `RouteProgress` تعطيه كلَّ
     * قراءة، **والصوتُ يستعمله والخريطةُ لا تراه** — فيبقى الخطُّ
     * كاملاً خلفَ السائق ويكذب على عينه.
     *
     * **وسالبٌ يعني «لا تقصَّ شيئاً»** — قبل أن تبدأ الملاحة.
     */
    val progressM: Double = -1.0,
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
        /**
         * **ومن حالِ المحرّك كذلك** — المرحلة ٣أ.
         *
         * **والحقولُ المرسومةُ هي هي**: `NavState` تزيد التقدّمَ
         * وحالَ الخروج، **والخريطةُ لا شأنَ لها بهما.**
         */
        fun of(state: NavState, stepId: Long): NavRender? {
            if (state.lat == null || state.lng == null) return null
            return NavRender(
                stepId = stepId,
                targetLat = state.lat,
                targetLng = state.lng,
                bearingDeg = state.bearingDeg,
                durationMs = state.animationMs,
                progressM = state.progress?.progressM ?: -1.0,
            )
        }

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
