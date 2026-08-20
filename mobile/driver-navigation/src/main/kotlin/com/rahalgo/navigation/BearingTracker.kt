package com.rahalgo.navigation

import kotlin.math.abs
import kotlin.math.exp

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اتّجاهُ السائق — لا يدور مع الضجيج ولا يتجمّد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «لا تعرض Raw Bearing مباشرة بدون معالجة…
 *  إذا كانت السرعة منخفضة لدرجة لا تجعل Bearing موثوقاً، لا تجعل سهم
 *  السائق يدور عشوائياً».)
 *
 * # ثلاثةُ مصادرَ مرتَّبة
 *
 * **١ · اتّجاهُ الجهاز** (`bearing`) حين تكون السرعةُ كافية — وهو
 * أدقُّها: يحسبه المستقبِلُ من إزاحة الطور لا من موضعين.
 *
 * **٢ · الاتّجاهُ بين قراءتين** حين لا يقوله الجهازُ وقد تحرّك تحرّكاً
 * يُعتدّ به.
 *
 * **٣ · آخرُ اتّجاهٍ عُرف** — يُثبَّت ولا يُبدَّل.
 *
 * # ولماذا لا يُصدَّق الاتّجاه عند البطء
 *
 * **مستقبِلُ الأقمار يحسب الاتّجاه من الحركة** — ومن يقف عند إشارة
 * تتحرّك قراءتُه مترين شرقاً ثمّ ثلاثةً غرباً بفعل الضجيج، **فيقول
 * الجهازُ إنّه استدار مئةً وثمانين درجة.**
 *
 * **والسهمُ يدور والسائقُ واقف** — وهو ما نهى عنه المالك نصّاً.
 *
 * **فدون مترين في الثانية (٧٫٢ كم/س) يُثبَّت الاتّجاه.**
 *
 * # ولا بوصلةَ مغناطيسيّة
 *
 * (أمرُ المالك: «لا تضف Magnetometer/Compass معقّداً من نفسك إلّا إذا
 *  أثبتّ أنّ الحاجة موجودة».)
 *
 * **ولم تثبت**: البوصلةُ تُفيد الواقفَ الذي يستدير بجسمه، **وسائقُنا
 * على درّاجةٍ محرّكُها معدنٌ يشوّش المغناطيس**، والهاتفُ في جيبٍ أو
 * حاملٍ لا يواجه جهةَ السير أصلاً. **فتُترك** — ومن أراد أن يعرف جهتَه
 * وهو واقفٌ ينظر إلى الخريطة لا إلى السهم.
 *
 * # والتنعيمُ زاويٌّ لا حسابيّ
 *
 * **الفرقُ بين ٣٥٩ و١ درجتان لا ٣٥٨** — ومن نعّم بالطرح المباشر جعل
 * الدرّاجةَ تدور دورةً كاملةً لتصحّح درجتين.
 *
 * **والمعامِلُ يتبع الزمنَ لا عددَ القراءات**: قراءتان بينهما ثانيةٌ
 * غيرُ قراءتين بينهما عشر، **ومعامِلٌ ثابتٌ يجعل التنعيمَ يشتدّ حين
 * تبطؤ القراءاتُ وهو وقتُ الحاجة إلى الاستجابة.**
 */
class BearingTracker(
    /** **دون هذه السرعة لا يُصدَّق اتّجاه** — م/ث. */
    private val minTrustedMps: Float = MIN_TRUSTED_MPS,
    /**
     * **زمنُ الاستجابة** بالثواني — كم يلزم ليقطع أكثرَ الفرق.
     *
     * **وثلاثةُ أعشارٍ توازنٌ مقيس**: أقلُّ منها يرتجف مع الضجيج،
     * **وأكثرُ منها يجعل السهمَ يكمل الانعطافَ بعد أن أكمله صاحبُه.**
     */
    private val responseSec: Double = RESPONSE_SEC,
) {

    /** **الاتّجاهُ المعروضُ الآن** — وفارغٌ يعني «لم يُعرف بعد». */
    var smoothed: Float? = null
        private set

    /** **آخرُ اتّجاهٍ خامٍّ صُدِّق** — للتشخيص. */
    var lastTrustedRaw: Float? = null
        private set

    private var lastAtMs: Long = 0L

    /**
     * **يُغذّى قراءةً فيردّ الاتّجاهَ المعروض.**
     *
     * **و`grade` يقرّر أيُصدَّق الاتّجاه**: المتدهورةُ يُؤخذ موضعُها
     * ولا يُصدَّق اتّجاهُها (انظر `GpsQuality`).
     */
    fun update(fix: NavFix, previous: NavFix?, grade: FixGrade): Float? {
        val raw = trustedRawOf(fix, previous, grade)
        if (raw == null) {
            // **ولا يُبدَّل شيءٌ حين لا يُصدَّق مصدر** — يبقى السهمُ
            // حيث كان. **وهذا هو «لا يدور عشوائيّا».**
            lastAtMs = fix.atMs
            return smoothed
        }
        lastTrustedRaw = raw
        val current = smoothed
        if (current == null) {
            // **وأوّلُ اتّجاهٍ يُؤخذ كما هو** — لا شيءَ يُنعَّم إليه.
            smoothed = raw
            lastAtMs = fix.atMs
            return raw
        }
        val dtSec = if (lastAtMs == 0L) responseSec else (fix.atMs - lastAtMs) / 1000.0
        lastAtMs = fix.atMs
        // **ومعامِلٌ أُسّيٌّ يتبع الزمن** — انظر أعلاه.
        val alpha = 1.0 - exp(-(dtSec.coerceAtLeast(0.0)) / responseSec)
        val delta = GpsQuality.angleDelta(current, raw)
        smoothed = GpsQuality.normalize(current + (delta * alpha).toFloat())
        return smoothed
    }

    /** **يُنسى كلُّ شيء** — عند بدء جلسةٍ جديدة. */
    fun reset() {
        smoothed = null
        lastTrustedRaw = null
        lastAtMs = 0L
    }

    /**
     * **أيُّ اتّجاهٍ خامٍّ يُصدَّق — إن وُجد.**
     *
     * **والمتدهورةُ لا تُعطي اتّجاهاً أبداً**: دقّةُ أربعين متراً
     * تعني موضعاً في دائرةٍ قطرُها ثمانون، **والاتّجاهُ المحسوبُ من
     * دائرتين يدور مع الضجيج.**
     */
    private fun trustedRawOf(fix: NavFix, previous: NavFix?, grade: FixGrade): Float? {
        if (grade != FixGrade.ACCEPTED) return null
        val speed = fix.speedMps
        if (speed == null || speed < minTrustedMps) return null

        fix.bearingDeg?.let { return GpsQuality.normalize(it) }

        // **وحين يصمت الجهازُ يُحسب من قراءتين** — بشرط أن تكون
        // الإزاحةُ أكبرَ من الشكّ، **وإلّا كان الاتّجاهُ اتّجاهَ ضجيج.**
        if (previous == null) return null
        val moved = GpsQuality.metersBetween(previous, fix)
        if (moved < (fix.accuracyM + previous.accuracyM)) return null
        return GpsQuality.courseBetween(previous, fix)
    }

    companion object {
        /**
         * **مترانِ في الثانية** ≈ ٧٫٢ كم/س.
         *
         * **وهي عتبةُ المشي السريع** — دونها لا يُميَّز السيرُ من ضجيج
         * الأقمار.
         */
        const val MIN_TRUSTED_MPS = 2f

        const val RESPONSE_SEC = 0.3

        /**
         * **قفزةٌ زاويّةٌ مستحيلةٌ عند السرعة** — تُستعمل في التشخيص.
         *
         * **ودرّاجةٌ بستّين كم/س لا تستدير تسعين درجةً في ثانية** —
         * وقوسُ الانعطاف يمنعها.
         */
        fun implausibleTurn(deltaDeg: Float, dtSec: Double, speedMps: Float?): Boolean {
            if (speedMps == null || dtSec <= 0) return false
            if (speedMps < MIN_TRUSTED_MPS) return false
            // **وكلّما أسرع قلَّ ما يستطيع أن يستدير** — ٣٦٠ درجة/ث
            // للواقف تقريباً، وتضيق مع السرعة.
            val maxRate = 900.0 / speedMps
            return abs(deltaDeg) / dtSec > maxRate
        }
    }
}
