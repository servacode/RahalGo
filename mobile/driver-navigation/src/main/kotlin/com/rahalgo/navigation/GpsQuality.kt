package com.rahalgo.navigation

import kotlin.math.abs
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.sin
import kotlin.math.sqrt

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حارسُ جودة القراءة — لا كلُّ ما يقوله الجهاز صحيح**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «لا تعامل `accuracy = 5m` مثل `accuracy =
 *  80m`، ولا تسمح لقراءةٍ واحدةٍ سيّئةٍ أن تجعل الدرّاجة تقفز شارعاً
 *  كاملاً».)
 *
 * # ثلاثةُ أحكامٍ لا حكمان
 *
 * **«اقبل أو ارفض» يخسر نصفَ القراءات**: دقّةُ ثلاثين متراً بين
 * الأبنية **موضعٌ صالحٌ واتّجاهٌ كاذب** — فتُقبل نقطتُها ويُهمَل
 * اتّجاهُها. **ومن رفضها كلَّها جمّد الدرّاجةَ في حيٍّ ضيّق.**
 *
 * # والقفزةُ تُقاس بالسرعة لا بالمسافة
 *
 * **مئةُ مترٍ في ثانيةٍ خطأٌ، ومئةُ مترٍ في عشرٍ سيرٌ عاديّ.** فالحدُّ
 * سرعةٌ ضمنيّةٌ لا مسافةٌ مطلقة، **ومن حدَّ بالمسافة وحدَها رفض سائقاً
 * على أوتوستراد.**
 *
 * **ويُتسامح مع الشكّ**: قراءةٌ دقّتُها أربعون متراً قد تبعد أربعين
 * متراً بحقّ، **فتُطرح الدقّةُ من المسافة قبل الحكم** — وإلّا رُفض
 * الصوابُ لأنّه غيرُ دقيق.
 */
object GpsQuality {

    /**
     * **حدُّ التصديق الكامل** — دقّةٌ أفضلُ منه تُصدَّق موضعاً واتّجاها.
     *
     * **وخمسةٌ وعشرون متراً حدُّ الشارع**: دونها تعرف أيَّ شارعٍ أنت
     * فيه، **وفوقها تعرف الحيَّ ولا تعرف الشارع.**
     */
    const val GOOD_ACCURACY_M = 25f

    /** **وما فوقه يُرمى** — لا موضعَ فيه ولا اتّجاه. */
    const val MAX_ACCURACY_M = 60f

    /**
     * **أقصى سرعةٍ ضمنيّةٍ تُصدَّق** — ٤٠ م/ث ≈ ١٤٤ كم/س.
     *
     * **ولا درّاجةَ توصيلٍ تبلغها** — وما فوقها قفزةُ قمرٍ صناعيّ.
     */
    const val MAX_IMPLIED_MPS = 40.0

    /**
     * **يحكم على قراءةٍ في ضوء ما قبلها.**
     *
     * **و`previous` فارغةٌ في أوّل قراءة** — فلا قفزةَ تُقاس، ويبقى
     * حكمُ الدقّة وحدَه.
     */
    fun grade(fix: NavFix, previous: NavFix?): Pair<FixGrade, RejectReason> {
        if (fix.accuracyM > MAX_ACCURACY_M) {
            return FixGrade.REJECTED to RejectReason.ACCURACY
        }
        if (previous != null) {
            // **والأقدمُ يُرمى** — الدفعاتُ تصل مختلطةَ الترتيب أحيانا،
            // **وقراءةٌ ماضيةٌ تُعرض تُرجع الدرّاجةَ إلى الوراء.**
            if (fix.atMs <= previous.atMs) {
                return FixGrade.REJECTED to RejectReason.STALE
            }
            val dtSec = (fix.atMs - previous.atMs) / 1000.0
            if (dtSec > 0) {
                // **والشكُّ يُطرح قبل الحكم** — انظر أعلاه.
                val slack = (fix.accuracyM + previous.accuracyM).toDouble()
                val moved = (metersBetween(previous, fix) - slack).coerceAtLeast(0.0)
                if (moved / dtSec > MAX_IMPLIED_MPS) {
                    return FixGrade.REJECTED to RejectReason.TELEPORT
                }
            }
        }
        val grade = if (fix.accuracyM <= GOOD_ACCURACY_M) FixGrade.ACCEPTED else FixGrade.DEGRADED
        return grade to RejectReason.NONE
    }

    /**
     * **المسافةُ بالأمتار** — بصيغة هافرساين.
     *
     * **ولا تُستعمل في قرارِ ترشيحٍ ولا تسعير** — هي لحكم الجودة وحدَه،
     * **وقرارُ المسافة في المحرّك بـPostGIS.**
     */
    fun metersBetween(a: NavFix, b: NavFix): Double = metersBetween(a.lat, a.lng, b.lat, b.lng)

    fun metersBetween(aLat: Double, aLng: Double, bLat: Double, bLng: Double): Double {
        val r = 6371000.0
        val p1 = Math.toRadians(aLat)
        val p2 = Math.toRadians(bLat)
        val dp = Math.toRadians(bLat - aLat)
        val dl = Math.toRadians(bLng - aLng)
        val h = sin(dp / 2) * sin(dp / 2) + cos(p1) * cos(p2) * sin(dl / 2) * sin(dl / 2)
        return 2 * r * atan2(sqrt(h), sqrt(1 - h))
    }

    /**
     * **اتّجاهُ السير بين نقطتين** — درجةٌ من الشمال.
     *
     * **يُستعمل حين لا يقول الجهازُ اتّجاهاً** وقد تحرّك تحرّكاً
     * يُعتدّ به. (انظر `BearingTracker`.)
     */
    fun courseBetween(a: NavFix, b: NavFix): Float {
        val p1 = Math.toRadians(a.lat)
        val p2 = Math.toRadians(b.lat)
        val dl = Math.toRadians(b.lng - a.lng)
        val y = sin(dl) * cos(p2)
        val x = cos(p1) * sin(p2) - sin(p1) * cos(p2) * cos(dl)
        return normalize(Math.toDegrees(atan2(y, x)).toFloat())
    }

    /** **يردّ الزاويةَ إلى `0..360`** — والسالبُ شمالٌ أيضاً. */
    fun normalize(deg: Float): Float {
        var d = deg % 360f
        if (d < 0) d += 360f
        return d
    }

    /**
     * **أقصرُ فرقٍ بين زاويتين** — `-180..180`.
     *
     * **والفرقُ بين ٣٥٩ و١ درجتان لا ٣٥٨** — ومن طرح مباشرةً جعل
     * الدرّاجةَ تدور دورةً كاملةً لتصحّح درجتين.
     */
    fun angleDelta(from: Float, to: Float): Float {
        var d = (to - from) % 360f
        if (d > 180f) d -= 360f
        if (d < -180f) d += 360f
        return d
    }

    /** **ألزاويتان متقاربتان؟** — يُستعمل في الاختبار والتشخيص. */
    fun angleClose(a: Float, b: Float, withinDeg: Float): Boolean =
        abs(angleDelta(a, b)) <= withinDeg
}
