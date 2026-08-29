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

    /**
     * **بابُ السجلّ — ولا يعرف المنسّقُ أندرويد.**
     *
     * **و`android.util.Log` غيرُ موجودةٍ في اختبار الحاسوب**، فتُلقي
     * `Method not mocked` وتُسقط سبعةَ اختباراتٍ سليمة (وقع
     * ٢٠٢٦-٠٨-٢٤). **ومنطقٌ يستدعي أندرويد منطقٌ لا يُختبر.**
     */
    var log: (String) -> Unit = {}

    /** **متى عُرضت كلُّ تعليمة** — لقياس انتظارها في الطابور. */
    private val offeredNs = HashMap<com.rahalgo.navigation.CueId, Long>()

    /** **آخرُ تقدّمٍ عُرف** — للقياس لا للمنطق. */
    private var progressM: Double = 0.0

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
        for (c in queue) offeredNs.putIfAbsent(c.id, System.nanoTime())
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
        this.progressM = progressM
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
        // **واللاحقةُ تُمرَّر لمن يعرفها** — ومن لا يعرفها يقول الأولى
        // وحدَها ولا يسقط.
        val talker = speaker
        val hand: ((Boolean) -> Unit) -> Unit = { cb ->
            if (talker is ClipSpeaker) {
                talker.speak(next.id.toString(), next.text, next.clip, next.thenClip, flush, cb)
            } else {
                talker.speak(next.id.toString(), next.text, next.clip, flush, cb)
            }
        }
        // ══════════════════════════════════════════════════════════════
        // **ويُقاس التأخّرُ بالمتر لا بالانطباع**
        // ══════════════════════════════════════════════════════════════
        //
        // (بلاغُ المالك ٢٠٢٦-٠٨-٢٤: «الصوتُ متأخّرٌ عن الطريق».)
        //
        // **وثلاثةُ أرقامٍ تكفي للحكم**: كم بقي بينه وبين المناورة حين
        // بدأ الكلام · وكم بقي حين انتهى · وكم انتظرت التعليمةُ في
        // الطابور. **والسالبُ يعني أنّه جاوز المنعطفَ والصوتُ يتكلّم.**
        val startedAtM = next.maneuver?.let { it.atDistanceM - progressM } ?: Double.NaN
        val startedNs = System.nanoTime()
        val waitedMs = (startedNs - offeredNs.getOrDefault(next.id, startedNs)) / 1_000_000
        offeredNs.remove(next.id)
        log(
            "بدأ «${next.clip}»${next.thenClip?.let { "+«$it»" } ?: ""} " +
                "طور=${next.stage} بُعد=${"%.0f".format(startedAtM)}م انتظار=${waitedMs}ملّي",
        )
        hand {
            // **والسرعةُ من التعليمة نفسِها** — هي التي حُسبت بها
            // عتبتُها، **فالقياسُ والقرارُ على رقمٍ واحد.**
            val v = if (next.firedSpeedMps > 0) next.firedSpeedMps else 8.0
            val spentM = (System.nanoTime() - startedNs) / 1e9 * v
            log(
            "انتهى «${next.clip}» وبقي ${"%.0f".format(startedAtM - spentM)}م",
            )
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
