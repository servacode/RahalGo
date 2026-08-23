package com.rahalgo.navigation

import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.sin
import kotlin.math.sqrt

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قيادةٌ مصنوعةٌ على مسارٍ حقيقيّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «تخلّي شاشةَ الرحلة قدّامي وتشوفني شلون
 *  الطلبُ يمشي عالخريطة على المتجر وعلى الزبون بدون أن أتحرّك أنا».)
 *
 * # ولماذا على هندسة المسار لا على خطٍّ مستقيم
 *
 * **خطٌّ مستقيمٌ بين نقطتين يمرّ فوق البيوت** — فيراه كاشفُ الخروج عن
 * المسار خروجاً في الثانية الأولى، **فتُعاد الحوسبةُ بلا انقطاع
 * ويُقرأ مختبَرٌ فاشلٌ وهو ناجح.**
 *
 * **والمسارُ الذي يردّه المحرّكُ هو الشوارعُ نفسُها** — فالمشيُ عليه
 * مشيٌ صحيح، **وانعطافاتُه انعطافاتٌ حقيقيّةٌ تُجرَّب بها البوصلة.**
 *
 * # والسرعةُ فوق العتبة قصداً
 *
 * **`BearingTracker` يجمّد الاتّجاه دون مترين في الثانية** (٧٫٢ كم/س)
 * — بأمر المالك: «لا تجعل سهمَ السائق يدور عشوائيّاً».
 *
 * **وخرج المالكُ يجرّب ماشياً** (٢٠٢٦-٠٨-٢٣) **فبلغت سرعتُه ١٫١ م/ث**
 * — دون العتبة كلَّها، **فلم يدر السهمُ ولم تدر الخريطةُ وبدا كأنّ
 * شيئاً لا يعمل.** فالافتراضُ هنا ثلاثون كم/س: سرعةُ درّاجةٍ في حيّ.
 *
 * # والاتّجاهُ يُحسَب ولا يُترك فارغاً
 *
 * **قراءةٌ بلا اتّجاهٍ تُجبر `BearingTracker` على المصدر الثاني**
 * (الاتّجاهُ بين قراءتين) — **وهو ما نريد أن نجرّبه أحياناً لا
 * دائماً.** فيُكتب هنا كما يكتبه مستقبِلُ الأقمار حين يتحرّك صاحبُه.
 *
 * # ولا ضجيجَ فيها
 *
 * **قراءاتٌ نظيفةٌ تماماً لا تشبه الأقمار** — والضجيجُ يُضاف حين
 * نجرّب الترشيحَ نفسَه. **وأوّلُ ما يُثبَت أنّ الشيءَ يمشي في أحسن
 * الأحوال**، ثمّ يُقسى عليه.
 */
object ReplayDrive {

    private const val R = 6_371_000.0

    /**
     * **يصنع قراءاتٍ تمشي على المسار.**
     *
     * @param route المسارُ كما ردّه المحرّك.
     * @param speedMps سرعةُ السير — الافتراضُ ٨٫٣ م/ث ≈ ٣٠ كم/س.
     * @param stepSec كم ثانيةً بين قراءتين — كفاصل الملاحة الحقيقيّ.
     * @param startMs لحظةُ أوّل قراءة (رتيبة).
     */
    fun fixes(
        route: NavRoute,
        speedMps: Float = 8.3f,
        stepSec: Double = 1.0,
        startMs: Long = 0L,
    ): List<NavFix> = fixes(route.geometry, speedMps, stepSec, startMs)

    /**
     * **ويقبل هندسةً عاريةً أيضاً.**
     *
     * **و`NavRoute` يشترط تراكميّاتٍ ومناوراتٍ لتكون `usable`** —
     * **والإعادةُ لا تحتاج منها شيئاً**: تمشي على الرؤوس وحدَها.
     * **فبناءُ مسارٍ كاملٍ لأجلها يعني اختلاقَ مناورةٍ لا وجودَ لها**،
     * وتلك تُقرأ في الشاشة تعليمةً كاذبة.
     */
    fun fixes(
        geometry: List<GeoPoint>,
        speedMps: Float = 8.3f,
        stepSec: Double = 1.0,
        startMs: Long = 0L,
    ): List<NavFix> {
        val g = geometry
        if (g.size < 2) return emptyList()

        // **وأطوالُ الأضلاع تُحسب مرّةً** — والمشيُ يقطعها بالترتيب.
        val legs = DoubleArray(g.size - 1)
        var total = 0.0
        for (i in 0 until g.size - 1) {
            legs[i] = meters(g[i], g[i + 1])
            total += legs[i]
        }
        if (total <= 0.0) return emptyList()

        val step = speedMps * stepSec
        val out = ArrayList<NavFix>((total / step).toInt() + 2)

        var leg = 0
        var doneInLeg = 0.0
        var travelled = 0.0
        var t = startMs

        while (travelled <= total && leg < legs.size) {
            val a = g[leg]
            val b = g[leg + 1]
            val f = if (legs[leg] <= 0.0) 0.0 else doneInLeg / legs[leg]
            out.add(
                NavFix(
                    lat = a.lat + (b.lat - a.lat) * f,
                    lng = a.lng + (b.lng - a.lng) * f,
                    // **ودقّةٌ ممتازةٌ قصداً** — **وقراءةٌ سيّئةٌ يرفضها
                    // `NavPipeline` فلا يمشي شيء**، والمرادُ هنا أن
                    // يمشي.
                    accuracyM = 5f,
                    speedMps = speedMps,
                    // **والاتّجاهُ اتّجاهُ الضلع** — وهو ما يقوله
                    // المستقبِلُ لمن يسير عليه.
                    bearingDeg = bearing(a, b),
                    atMs = t,
                    provider = "replay",
                ),
            )
            t += (stepSec * 1000).toLong()
            travelled += step
            doneInLeg += step
            // **ويُستهلك ما زاد عن الضلع في الذي يليه** — **ومن بدأ
            // كلَّ ضلعٍ من أوّله جعل السرعةَ تتبدّل بطول الضلع.**
            while (leg < legs.size && doneInLeg >= legs[leg]) {
                doneInLeg -= legs[leg]
                leg++
            }
        }

        // **وآخرُ نقطةٍ تُضاف صراحةً** — **ومن وقف قبل الوجهة بمترين
        // لم يصل**، فلا يُختبر حارسُ الوصول أصلاً.
        val last = g.last()
        val prev = g[g.size - 2]
        out.add(
            NavFix(
                lat = last.lat,
                lng = last.lng,
                accuracyM = 5f,
                speedMps = 0f,
                bearingDeg = bearing(prev, last),
                atMs = t,
                provider = "replay",
            ),
        )
        return out
    }

    /** **المسافةُ بالأمتار** — هافرساين، وهي كافيةٌ في حيٍّ واحد. */
    private fun meters(a: GeoPoint, b: GeoPoint): Double {
        val dLat = Math.toRadians(b.lat - a.lat)
        val dLng = Math.toRadians(b.lng - a.lng)
        val la1 = Math.toRadians(a.lat)
        val la2 = Math.toRadians(b.lat)
        val h = sin(dLat / 2) * sin(dLat / 2) +
            cos(la1) * cos(la2) * sin(dLng / 2) * sin(dLng / 2)
        return 2 * R * atan2(sqrt(h), sqrt(1 - h))
    }

    /** **الاتّجاهُ من `a` إلى `b`** بالدرجات من الشمال `0..360`. */
    private fun bearing(a: GeoPoint, b: GeoPoint): Float {
        val la1 = Math.toRadians(a.lat)
        val la2 = Math.toRadians(b.lat)
        val dLng = Math.toRadians(b.lng - a.lng)
        val y = sin(dLng) * cos(la2)
        val x = cos(la1) * sin(la2) - sin(la1) * cos(la2) * cos(dLng)
        val deg = Math.toDegrees(atan2(y, x))
        return ((deg + 360.0) % 360.0).toFloat()
    }
}
