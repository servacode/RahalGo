package com.rahalgo.navigation

import java.io.BufferedWriter
import java.io.File
import java.io.FileWriter
import java.util.concurrent.LinkedBlockingQueue
import java.util.concurrent.TimeUnit

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مسجّلُ الرحلة — أداةُ فحصٍ لا جزءٌ من الملاحة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «أستطيع تشغيل جلسة الملاحة، الخروج من مدى
 *  Wi-Fi بحرّيّة، القيام برحلة حقيقيّة، ثمّ العودة وسحب ملف الرحلة».)
 *
 * # ولماذا يلزم أصلاً
 *
 * **الاختبارُ الميدانيُّ يحتاج أن يتحرّك الهاتفُ في الشارع، واللابتوبُ
 * لا يتبعه** — **فوصلةٌ لاسلكيّةٌ تنقطع عند باب البيت.**
 *
 * **فالجهازُ يكتب رحلتَه بيده**، ويُسحب الملفُّ عند العودة. **وكلُّ
 * رحلةٍ تصير رفيدةً تُعاد آليّاً إلى الأبد** — وهو أساسُ
 * `navigation-fixtures`.
 *
 * # ويُسجَّل الرديءُ كما يُسجَّل الجيّد
 *
 * (أمرُ المالك: «لا تسجّل Accepted فقط؛ نحتاج القراءات السيّئة نفسَها
 *  لإعادة اختبار الفلاتر لاحقاً».)
 *
 * **ورحلةٌ فيها المقبولُ وحدَه لا تختبر مرشِّحاً** — **تختبر ما نجا
 * منه.**
 *
 * # وفشلُه لا يُسقط شيئاً
 *
 * **القرصُ يمتلئ، والملفُّ يُقفل، والإذنُ يُسحب** — **ورحلةُ سائقٍ
 * تنهار لأنّ أداةَ فحصٍ عجزت عن الكتابة** خطأٌ أفدحُ من ضياع رفيدة.
 *
 * **فيُطفأ عند أوّل فشلٍ ويُسجَّل السبب، وتمضي الملاحة.**
 *
 * # ولا أندرويدَ فيه
 *
 * **يأخذ مجلّداً لا `Context`** — فيُختبر بلا جهازٍ ولا محاكٍ.
 * **والتطبيقُ يمرّر `filesDir`.**
 *
 * # ولا بياناتٍ شخصيّة
 *
 * (أمرُ المالك: «لا اسمَ سائقٍ ولا رقمَ هاتفٍ ولا تفاصيلَ طلب».)
 *
 * **ما يُكتب موقعٌ ووقتٌ وجودةٌ لا غير** — **ورفيدةٌ تحمل اسمَ إنسانٍ
 * لا تُشارَك ولا تُحفظ.**
 */
class TraceRecorder(
    /** **مجلّدُ الرحلات** — يُنشأ إن لم يكن. */
    private val dir: File,
    /** **معرّفُ الجلسة** — يدخل في اسم الملفّ. */
    val sessionId: String,
    /** **ساعةُ الجدار** — تُحقن في الاختبار. */
    private val nowMs: () -> Long = System::currentTimeMillis,
) {

    /** **الملفُّ الذي يُكتب فيه** — يُعرف بعد `start`. */
    var file: File? = null
        private set

    /** **أتعطّل المسجّل؟** — يُقرأ في التقرير. */
    var failed: Boolean = false
        private set

    /** **سببُ التعطّل** — أو فارغ. */
    var failure: String? = null
        private set

    /** **كم سطراً كُتب فعلاً** — للتحقّق. */
    @Volatile
    var written: Int = 0
        private set

    private var writer: BufferedWriter? = null
    private var worker: Thread? = null
    private val queue = LinkedBlockingQueue<String>()
    @Volatile private var running = false

    /**
     * **يفتح ملفَّ الرحلة ويشغّل كاتباً في خيطٍ خاصّ.**
     *
     * **والكتابةُ لا تقع على خيط الشاشة** — **وقرصٌ بطيءٌ في حلقةٍ
     * تعمل كلَّ ثانيةٍ يجعل الخريطةَ ترتجف.**
     */
    fun start(): Boolean {
        if (running || failed) return running
        return try {
            if (!dir.exists()) dir.mkdirs()
            val f = File(dir, fileNameFor(sessionId, nowMs()))
            writer = BufferedWriter(FileWriter(f, false), BUFFER)
            file = f
            running = true
            worker = Thread({ drain() }, "nav-trace").apply {
                isDaemon = true
                start()
            }
            true
        } catch (e: Throwable) {
            fail(e)
            false
        }
    }

    /**
     * **يضيف قراءةً بحكمها** — ولا يحجب المنادي.
     *
     * **والصفُّ في الذاكرة والكتابةُ في خيطٍ آخر**: من كتب على القرص
     * في مسار القراءة **علّق الملاحةَ على سرعة الذاكرة الوميضيّة.**
     */
    fun add(fix: NavFix, grade: FixGrade, reason: RejectReason) {
        if (!running || failed) return
        try {
            queue.offer(lineOf(sessionId, fix, grade, reason))
        } catch (e: Throwable) {
            fail(e)
        }
    }

    /**
     * **يُفرغ ما بقي ويُغلق الملفَّ إغلاقاً سليماً.**
     *
     * **وملفٌّ لم يُغلق يفقد آخرَ ما في مخزنه** — وهو غالباً أهمُّ ما
     * في الرحلة: نهايتُها.
     */
    fun stop(): File? {
        if (!running) return file
        running = false
        return try {
            // **ويُنتظر الكاتبُ قليلاً ليفرغ صفَّه** — ولا يُنتظر إلى
            // الأبد: **إغلاقٌ يعلّق الشاشةَ أسوأُ من سطرٍ ضائع.**
            worker?.join(JOIN_MS)
            drainRemaining()
            writer?.flush()
            writer?.close()
            writer = null
            file
        } catch (e: Throwable) {
            fail(e)
            file
        }
    }

    private fun drain() {
        while (running) {
            try {
                val line = queue.poll(POLL_MS, TimeUnit.MILLISECONDS) ?: continue
                writeLine(line)
            } catch (e: InterruptedException) {
                return
            } catch (e: Throwable) {
                fail(e)
                return
            }
        }
    }

    private fun drainRemaining() {
        while (true) {
            val line = queue.poll() ?: break
            writeLine(line)
        }
    }

    private fun writeLine(line: String) {
        writer?.let {
            it.write(line)
            it.write("\n")
            written++
        }
    }

    private fun fail(e: Throwable) {
        failed = true
        running = false
        failure = e.javaClass.simpleName + ": " + (e.message ?: "")
        // **ولا يُرمى شيءٌ إلى المنادي** — انظر أعلى الصنف.
        runCatching { writer?.close() }
        writer = null
    }

    companion object {
        /**
         * **نسخةُ الصيغة** — تُكتب في كلّ سطر.
         *
         * **ورفيدةٌ بلا نسخةٍ تُكسَر يومَ تتبدّل الصيغة** — ولا يُعرف
         * أيُّ الملفّات قديمٌ وأيُّها جديد.
         */
        const val SCHEMA = 1

        /** **أقدمُ نسخةٍ يقرؤها القارئ** — انظر `TraceReader`. */
        const val MIN_SCHEMA = 1

        private const val BUFFER = 16 * 1024
        private const val POLL_MS = 200L
        private const val JOIN_MS = 1_500L

        /** **اسمٌ يقول متى ولأيّ جلسة.** */
        fun fileNameFor(sessionId: String, startMs: Long): String =
            "nav-$startMs-$sessionId.jsonl"

        /**
         * **سطرُ JSONL واحد** — يُبنى بيدٍ لا بمكتبة.
         *
         * **والحقولُ مسطّحةٌ وقيمُها أرقامٌ ونصوصٌ قصيرة** — **ومكتبةٌ
         * تُضاف لسطرٍ كهذا تحمل الوحدةَ تبعيّةً لا تحتاجها.**
         *
         * **والفارغُ يُكتب `null` لا صفراً** — أمرُ المالك: «Bearing
         * الغائب يبقى `null` ولا يتحوّل إلى صفر».
         */
        fun lineOf(sessionId: String, fix: NavFix, grade: FixGrade, reason: RejectReason): String {
            val sb = StringBuilder(220)
            sb.append('{')
            sb.append("\"schema_version\":").append(SCHEMA)
            sb.append(",\"session_id\":").append(quote(sessionId))
            sb.append(",\"timestamp\":").append(fix.wallMs)
            sb.append(",\"elapsed_realtime_nanos\":").append(fix.elapsedNs)
            sb.append(",\"at_ms\":").append(fix.atMs)
            sb.append(",\"latitude\":").append(fix.lat)
            sb.append(",\"longitude\":").append(fix.lng)
            sb.append(",\"accuracy_m\":").append(fix.accuracyM)
            sb.append(",\"speed_mps\":").append(fix.speedMps?.toString() ?: "null")
            sb.append(",\"bearing_deg\":").append(fix.bearingDeg?.toString() ?: "null")
            sb.append(",\"has_bearing\":").append(fix.bearingDeg != null)
            sb.append(",\"quality_result\":").append(quote(grade.name))
            sb.append(",\"rejection_reason\":")
                .append(if (reason == RejectReason.NONE) "null" else quote(reason.name))
            sb.append(",\"provider\":").append(fix.provider?.let { quote(it) } ?: "null")
            sb.append('}')
            return sb.toString()
        }

        /** **تهريبُ ما يُكسر السطر** — والأسماءُ عندنا بسيطةٌ لكنّ الحارسَ يبقى. */
        fun quote(s: String): String {
            val sb = StringBuilder(s.length + 2)
            sb.append('"')
            for (c in s) {
                when (c) {
                    '"' -> sb.append("\\\"")
                    '\\' -> sb.append("\\\\")
                    '\n' -> sb.append("\\n")
                    '\r' -> sb.append("\\r")
                    '\t' -> sb.append("\\t")
                    else -> sb.append(c)
                }
            }
            sb.append('"')
            return sb.toString()
        }
    }
}
