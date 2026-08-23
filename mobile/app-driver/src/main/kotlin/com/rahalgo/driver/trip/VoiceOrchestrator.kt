package com.rahalgo.driver.trip

import com.rahalgo.navigation.CueId
import com.rahalgo.navigation.CueStage
import com.rahalgo.navigation.Speaker
import com.rahalgo.navigation.VoiceCue

/**
 * ══════════════════════════════════════════════════════════════════════
 * **منسّقُ الكلام — يقرّر أيُقال الآن، وبأيّ ترتيب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٤، أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
 *
 *     VoiceCue  →  VoiceOrchestrator  →  Speaker  →  Android TTS
 *
 * # ولماذا في التطبيق لا في وحدة الملاحة
 *
 * **أمرُ المالك رسم الحدَّ**: وحدةُ الملاحة تقرّر **ماذا ومتى**،
 * والتطبيقُ يقرّر **كيف**. **والطابورُ والأولويّةُ شأنُ نطقٍ لا شأنُ
 * طريق.**
 *
 * # وسِجِلُّ ما قيل يعيش هنا
 *
 * **وهذا هو ما ينجّي من إعادة إنشاء الشاشة** (أمرُ المالك، البند ٢١:
 * «لا تعيد `DEPART` أو Cue قديمة فقط لأنّ Activity أُعيد إنشاؤها»).
 *
 * **و`NavigationSession` تعيش في التركيب** — تموت بدوران الجهاز
 * وتُبنى من جديد، **فيُصفَّر عبورُ العتبات ويُعاد إطلاق ما مضى.**
 * **وهذا المنسّقُ يعيش في `ViewModel`** فيتجاوز ذلك: **ما قيل مرّةً لا
 * يُقال ثانية.**
 *
 * # ولا شبكةَ ولا أندرويد هنا
 *
 * **صنفٌ خالصٌ يُختبر بناطقٍ وهميّ** — و`Speaker` واجهةٌ بدالّتين.
 */
class VoiceOrchestrator(
    private val speaker: Speaker,
    private val maxQueue: Int = MAX_QUEUE,
) {

    /**
     * **الكتمُ حالٌ مستقلّة.**
     *
     * (أمرُ المالك، البند ٢٠: «Navigation يستمرّ · Cue logic يستمرّ ·
     *  لا Speech».)
     *
     * **والتعليماتُ تُولَّد وهو مكتومٌ وتُوسَم «قيلت» ثمّ تُطرح** —
     * **فمن أعاد الصوتَ لا يسمع سيلاً ممّا فات.**
     */
    var muted: Boolean = false
        set(value) {
            field = value
            if (value) {
                queue.clear()
                speaker.stop()
                speaking = null
            }
        }

    /** **عدّاداتُ القياس** — تُقرأ في الاختبار والتقرير. */
    var spoken = 0
        private set
    var dropped = 0
        private set
    var staleDropped = 0
        private set

    private val said = HashSet<CueId>()
    private val queue = ArrayList<VoiceCue>(4)
    private var speaking: VoiceCue? = null

    /** **ما يُقال الآن** — للتشخيص. */
    val current: VoiceCue? get() = speaking

    val pending: Int get() = queue.size

    /**
     * **يُغذّى ما قرّره المخطِّط، ومعه التقدّمُ الآن.**
     *
     * **والتقدّمُ يُمرَّر لأنّ الصلاحيّةَ تُفحص عند التسليم لا عند
     * التوليد** — **وبينهما قد ينعطف السائقُ وينتهي.**
     */
    fun offer(cues: List<VoiceCue>, progressM: Double, navigating: Boolean = true) {
        for (cue in cues) {
            // **وما قيل مرّةً لا يُقال ثانية** — مهما تكرّر توليدُه.
            if (!said.add(cue.id)) {
                dropped++
                continue
            }
            if (muted) continue
            queue += cue
        }
        // **وطابورٌ صغيرٌ محكومٌ لا طابورُ أندرويد المفتوح** — أمرُ
        // المالك، البند ٢٣. **والأقدمُ الأدنى يسقط أوّلا.**
        if (queue.size > maxQueue) {
            queue.sortByDescending { it.priority }
            while (queue.size > maxQueue) {
                queue.removeAt(queue.size - 1)
                dropped++
            }
        }
        pump(progressM, navigating)
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **ولا يُسلَّم للناطق إلّا ما صحّ الآن**
     * ══════════════════════════════════════════════════════════════════
     *
     * (أمرُ المالك، البند ٢٢.)
     */
    fun pump(progressM: Double, navigating: Boolean = true) {
        if (muted || !navigating || !speaker.available) {
            if (queue.isNotEmpty()) {
                dropped += queue.size
                queue.clear()
            }
            return
        }
        // **والمنتهيةُ صلاحيّتُها تُطرح قبل النطق** — لا بعده.
        val expired = queue.filter { progressM > it.validUntilProgressM }
        if (expired.isNotEmpty()) {
            queue.removeAll(expired)
            staleDropped += expired.size
        }
        if (queue.isEmpty()) return

        queue.sortByDescending { it.priority }
        val next = queue[0]

        val busy = speaking
        if (busy != null) {
            // ══════════════════════════════════════════════════════════
            // **ولا يُقطع الكلامُ إلّا لما هو أنفعُ منه**
            // ══════════════════════════════════════════════════════════
            //
            // (أمرُ المالك: «لا أريد جملاً تتقطّع باستمرار في Close
            //  Maneuvers».)
            //
            // **فالقطعُ بالأولويّة لا بالوصول**: `NOW` تقطع تمهيداً
            // قديماً، **وتمهيدٌ لا يقطع «الآن».**
            if (next.priority <= busy.priority) return
        }
        queue.removeAt(0)
        speaking = next
        spoken++
        val flush = busy != null
        speaker.speak(next.id.toString(), next.text, flush) {
            if (speaking?.id == next.id) speaking = null
            // **ولا يُنادى `pump` من هنا بتقدّمٍ قديم** — من يقرأ
            // التقدّمَ هو المحرّك في القراءة التالية.
        }
    }

    /**
     * **ينتهي كلُّ شيء** — عند إغلاق الملاحة.
     *
     * **والسجِلُّ يبقى** — فجلسةٌ تُفتح ثانيةً على المسار نفسِه لا
     * تُعيد ما قيل. **ويمحوه الجيلُ الجديدُ وحدَه.**
     */
    fun stop() {
        queue.clear()
        speaker.stop()
        speaking = null
    }

    /** **يُنسى ما قيل** — عند رحلةٍ جديدةٍ لا عند دورانِ جهاز. */
    fun forget() {
        said.clear()
        stop()
        spoken = 0
        dropped = 0
        staleDropped = 0
    }

    private companion object {
        /**
         * **ثلاثُ جملٍ تنتظر لا أكثر.**
         *
         * **وما زاد قديمٌ بالضرورة** — عند هرتزٍ واحدٍ ثلاثُ جملٍ
         * تعني ثلاثَ ثوانٍ من التأخّر.
         */
        const val MAX_QUEUE = 3
    }
}
