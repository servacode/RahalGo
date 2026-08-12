package com.rahalgo.driver.trip

import android.graphics.Bitmap
import java.io.ByteArrayOutputStream

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صورة التسليم — تُصغَّر قبل أن تُرسَل**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (البند السادس في قائمة المالك ٢٠٢٦-٠٨-١٢.)
 *
 * # لماذا تُصغَّر
 *
 * **صورة كاميرا اليوم أربعة ميغابايت** — وحزمة السائق تُدفع من جيبه،
 * **وشبكة الرقّة في الذروة تجعل رفعها دقيقة.** **وهو واقف عند الباب
 * والزبون ينتظر.**
 *
 * **وما نريده منها إثبات لا تفصيل**: باب وكيس، **وألف بكسل تكفي**
 * لتُقرأ في شكوى.
 *
 * # وأهمّ منها الإحداثيات
 *
 * **صورة باب قد تكون لأيّ باب** — والنقطة تقول أين وقف حين صوّر.
 * **فتُرسل معها**، والمحرّك يحفظها في `pod_at`.
 */
object Proof {

    /** **العرض الأقصى** — والارتفاع يتبعه. */
    private const val MAX_SIDE = 1080

    /** **جودة كافية لباب وكيس** — ولا تُضاعف الحجم بلا فائدة. */
    private const val QUALITY = 78

    fun shrink(source: Bitmap): ByteArray {
        val scale = MAX_SIDE.toFloat() / maxOf(source.width, source.height)
        val bitmap = if (scale < 1f) {
            Bitmap.createScaledBitmap(
                source,
                (source.width * scale).toInt(),
                (source.height * scale).toInt(),
                true,
            )
        } else {
            source
        }
        val out = ByteArrayOutputStream()
        bitmap.compress(Bitmap.CompressFormat.JPEG, QUALITY, out)
        return out.toByteArray()
    }
}
