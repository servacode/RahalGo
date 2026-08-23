package com.rahalgo.navigation

import kotlin.math.abs

/**
 * ══════════════════════════════════════════════════════════════════
 * **الاشتباهُ المحلّيّ — لا يُقرّر شيئاً**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٨ب، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 * **غايتُه سؤالٌ واحد: أنسأل الخادم؟** — ولا يقول «هذا طريقٌ موازٍ»
 * مهما طالت الإزاحة (البند ١٠). **الحكمُ ليس هنا.**
 *
 * # ولماذا لا يستطيع أن يقرّر
 *
 * **قِيس** (٢٠٢٦-٠٨-٢١، ١٢٬٠٤٣ زوجَ طرقٍ من `syria.osm.pbf`): بتباعد
 * ١٢م ودقّةٍ ١٠م **يصيب الحكمُ المحلّيُّ ٦٧٪** — وذلك ليس كشفاً.
 *
 * **والسببُ أنّ خطأَ الموضع مترابطٌ لا مستقلّ**: ينحرف في جهةٍ واحدةٍ
 * دقائقَ، **فالمعدّلُ لا يُلغيه.** وسائقٌ سليمٌ بانحيازٍ ١٥م يبدو
 * كسائقٍ على طريقٍ يبعد ١٥م — **ولا فرقَ في القراءة.**
 *
 * # فما الذي يُميّزه إذاً
 *
 * **لا شيء.** ولذلك **بوّابةُ الدقّة هي كلُّ شيء**: قِيس أنّ الاشتباهَ
 * بلا بوّابةٍ ينطلق **١٠٤ مرّاتٍ في الساعة على المسار الصحيح**،
 * **وبها عند [MIN_OFFSET_M] وقراءةٍ دقّتُها ≤[MAX_ACCURACY_M]: صفر.**
 */
class ParallelSuspicion(
    private val tuning: Tuning = Tuning(),
) {

    /**
     * **العتباتُ مُعايَرةٌ لا مختارة** — ٢٥٢ سلسلةً على أزواجِ طرقٍ
     * حقيقيّةٍ بضجيجٍ مترابطٍ `AR(1)` τ=٣٠ث.
     */
    data class Tuning(
        /**
         * **حدُّ الإزاحة.**
         *
         * **قِيست نسبةُ الرحلات التي تُطلق اشتباهاً كاذباً** بدقّةٍ
         * مبلَّغةٍ ١٠م:
         *
         *     ١٢م → ٤٤٪   ·   **١٥م → ٢٢٪**   ·   ١٨م → ٣٪   ·   ٢٠م → ٠٪
         *
         * **وخمسةَ عشرَ توازن**: ما دونها عاصفة، وما فوقها يفوّت
         * أزواجاً حقيقيّةً تباعدُها ١٥–٢٠م.
         */
        val minOffsetM: Double = MIN_OFFSET_M,

        /**
         * **مدّةُ الاستمرار.**
         *
         * **وهي نافذةُ المطابقة الأولى نفسُها** (البند ١١): القراءاتُ
         * الثمانِ التي أنتجت الاشتباه **هي ما يُرسَل** — فلا تُجمع
         * ثمانٍ مرّتين.
         */
        val sustainSec: Int = SUSTAIN_SEC,

        /**
         * **ودقّةٌ أوسعُ من هذه لا يُبنى عليها اشتباه.**
         *
         * **بلا هذه البوّابة**: ١٠٤ نوبةً في الساعة على المسار
         * الصحيح. **وبها: صفر.**
         */
        val maxAccuracyM: Float = MAX_ACCURACY_M,
    )

    /** **حالُ الاشتباه** — اثنتان لا ثالثة (البند ١٠). */
    enum class State {
        /** **لا شيء.** */
        NONE,

        /** **يستحقّ سؤالاً** — ولا يستحقّ قراراً. */
        SUSPECTED,
    }

    /**
     * **القراءاتُ المستوفيةُ للشرط** — أحدثُها آخراً.
     *
     * **وهي الدليلُ الذي يُرسَل** — فلا نافذةَ ثانيةٌ تُجمع.
     */
    private val window = ArrayDeque<NavFix>()
    private val offsets = ArrayDeque<Double>()

    var state: State = State.NONE
        private set

    /** **متوسّطُ الإزاحة في النافذة** — للتشخيص. */
    var meanOffsetM: Double = 0.0
        private set

    /**
     * **يُغذّى قراءةً وإزاحتَها الموقّعة.**
     *
     * @param lateralSignedM من [RouteProjector.Hit] — **موجبٌ يساراً.**
     * @return الحالُ بعد هذه القراءة.
     */
    fun onFix(fix: NavFix, grade: FixGrade, lateralSignedM: Double): State {
        // ══════════════════════════════════════════════════════════
        // **ولا اشتباهَ إلّا من قراءةٍ مقبولةٍ دقيقة — البند ١٦**
        // ══════════════════════════════════════════════════════════
        //
        // **`REJECTED` لا دليلَ فيها**، **و`DEGRADED` موضعُها
        // تقريبيٌّ** — ومن بنى عليها اشتباهاً بنى عاصفةَ نداءات.
        if (grade != FixGrade.ACCEPTED || fix.accuracyM > tuning.maxAccuracyM) {
            reset()
            return state
        }
        if (abs(lateralSignedM) < tuning.minOffsetM) {
            reset()
            return state
        }

        // ── والجهةُ الواحدةُ شرطٌ ضمنيّ ──────────────────────────
        //
        // **ولا تُقاس نسبةَ ثباتٍ** (قِيس أنّها بلا قيمةٍ تمييزيّة:
        // ٧٥٪ و٩٥٪ تعطيان النتيجةَ نفسَها حرفيّاً، **لأنّ الضجيجَ
        // المترابطَ يُثبّت الجهةَ بنفسه**).
        //
        // **بل تُمسح النافذةُ عند انقلاب الجهة** — وهو أبسطُ وأصرم.
        val sign = if (lateralSignedM >= 0) 1.0 else -1.0
        if (offsets.isNotEmpty()) {
            val prev = if (offsets.last() >= 0) 1.0 else -1.0
            if (prev != sign) reset()
        }

        window.addLast(fix)
        offsets.addLast(lateralSignedM)
        // **ونافذةٌ متدحرجةٌ بطول المدّة** — لا تنمو بلا حدّ.
        while (window.size > tuning.sustainSec) {
            window.removeFirst()
            offsets.removeFirst()
        }

        meanOffsetM = offsets.sumOf { it } / offsets.size
        state = if (window.size >= tuning.sustainSec &&
            abs(meanOffsetM) >= tuning.minOffsetM
        ) {
            State.SUSPECTED
        } else {
            State.NONE
        }
        return state
    }

    /**
     * **الدليلُ الحاضر** — أحدثُ [Tuning.sustainSec] قراءةً مقبولة.
     *
     * **وهو ما يُرسَل** — ولا يُجمع دليلٌ ثانٍ من الصفر.
     */
    fun evidence(): List<NavFix> = window.toList()

    /** **أحدثُ قراءةٍ في النافذة** — يُعرف بها هل جاء دليلٌ طازج. */
    fun newestAtMs(): Long = window.lastOrNull()?.atMs ?: 0L

    fun reset() {
        window.clear()
        offsets.clear()
        meanOffsetM = 0.0
        state = State.NONE
    }

    companion object {
        const val MIN_OFFSET_M = 15.0
        const val SUSTAIN_SEC = 8
        const val MAX_ACCURACY_M = 10f
    }
}
