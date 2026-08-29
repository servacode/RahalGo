package com.rahalgo.map

import android.animation.ValueAnimator
import android.view.animation.LinearInterpolator

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مُحرِّكُ الأيقونة — يمشي بها ولا يقفز**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «بدل القفز A → B يجب أن تتحرّك أيقونة
 *  السائق بصريّاً بسلاسة… **وعدم استمرار Animation قديم إذا وصلت قراءة
 *  أحدث**».)
 *
 * # ولا يعرف ما يُحرِّك
 *
 * يتلقّى نقطتين وزاويتين ومدّة، **ويستدعي مُستقبِلاً في كلّ إطار.**
 * **ومن أراد أن يعرف أنّها درّاجةُ سائقٍ فليقرأ وحدةَ الملاحة.**
 *
 * # والقديمُ يُلغى قبل أن يبدأ الجديد
 *
 * **ومن ترك حركتين تعملان معاً رأى الأيقونةَ ترتجف بينهما** — كلٌّ
 * تسحبها إلى هدفها. **وهو ما نهى عنه المالك نصّاً.**
 *
 * # والاستيفاءُ خطّيٌّ هنا
 *
 * **التخفيفُ في المنطق لا في المشغّل** (`PathSmoother.ease`) —
 * **ومُخفِّفان أحدُهما فوق الآخر يجعلان الحركةَ تتباطأ مرّتين.**
 */
class MarkerAnimator {

    private var running: ValueAnimator? = null

    /**
     * **يمشي من موضعٍ وزاويةٍ إلى موضعٍ وزاوية.**
     *
     * **و`onFrame` تُنادى على خيط الشاشة** — فلا يُوضع فيها عملٌ ثقيل.
     */
    fun animate(
        fromLat: Double,
        fromLng: Double,
        toLat: Double,
        toLng: Double,
        fromBearing: Float,
        toBearing: Float,
        durationMs: Long,
        /**
         * **و`t` كسرُ الإطار** — من صفرٍ إلى واحد.
         *
         * **وبه يُعرف موضعُ الخطّ تحت السهم في هذه اللحظة بعينها** —
         * (بلاغُ المالك ٢٠٢٦-٠٨-٢٤: «يجب ألّا يلاحظ الشخصُ قصَّها»).
         */
        onFrame: (lat: Double, lng: Double, bearing: Float, t: Float) -> Unit,
    ) {
        cancel()
        // **ومدّةٌ صفرٌ تعني «ضعها هناك»** — لا حركةَ تُرى في إطار.
        if (durationMs <= 0L) {
            onFrame(toLat, toLng, toBearing, 1f)
            return
        }
        val delta = shortestDelta(fromBearing, toBearing)
        running = ValueAnimator.ofFloat(0f, 1f).apply {
            duration = durationMs
            interpolator = LinearInterpolator()
            addUpdateListener { a ->
                val t = a.animatedValue as Float
                onFrame(
                    fromLat + (toLat - fromLat) * t,
                    fromLng + (toLng - fromLng) * t,
                    normalize(fromBearing + delta * t),
                    t,
                )
            }
            start()
        }
    }

    /** **يُلغى ما يعمل** — عند وصول قراءةٍ أحدثَ أو مغادرة الشاشة. */
    fun cancel() {
        running?.cancel()
        running = null
    }

    /** **أثمّةَ حركةٌ تعمل؟** — للتشخيص. */
    val isRunning: Boolean get() = running?.isRunning == true

    private companion object {
        /**
         * **أقصرُ فرقٍ بين زاويتين.**
         *
         * **ومن استوفى بالطرح المباشر جعل الأيقونةَ تلفّ لفّةً كاملة**
         * حين تعبر الشمال — والحسابُ نفسُه في `PathSmoother`، **وهو
         * هناك للمنطق وهنا للرسم.**
         */
        fun shortestDelta(from: Float, to: Float): Float {
            var d = (to - from) % 360f
            if (d > 180f) d -= 360f
            if (d < -180f) d += 360f
            return d
        }

        fun normalize(deg: Float): Float {
            var d = deg % 360f
            if (d < 0) d += 360f
            return d
        }
    }
}
