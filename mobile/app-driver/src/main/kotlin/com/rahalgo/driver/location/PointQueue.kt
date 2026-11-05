package com.rahalgo.driver.location

import android.content.Context
import android.util.Log
import com.rahalgo.shared.model.TrackPoint
import java.io.File
import kotlinx.serialization.json.Json

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طابور النقاط — ما لم يصل يُحفظ حتّى يصل**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ١ من `docs/DRIVER-APP-PLAN.md`.)
 *
 * # لماذا يلزم
 *
 * **الشبكة تنقطع في الشارع**: نفق، أو حيّ بلا تغطية، أو حزمة انتهت.
 * **وما لم يُحفظ ضاع** — فيظهر السائق للمكتب واقفاً عشر دقائق وهو يسير،
 * **ولا يعرف أحد أين مرّ.**
 *
 * # ولماذا ملفّ لا قاعدة بيانات
 *
 * **النقطة سطر واحد يُكتب ويُقرأ مرّة ثمّ يُمحى** — لا استعلام عليها ولا
 * ترتيب ولا بحث. **وقاعدة بيانات لهذا ثقلٌ بلا ثمن.**
 *
 * **وقاعدة البيانات موضعها حين نخزّن الطلبات نفسها** (المرحلة ٤): هناك
 * يُبحث ويُرتَّب ويُربط.
 *
 * # ساعةٌ كاملةٌ بلا شبكة (0.9.0، ٢٠٢٦-١١-٠٥)
 *
 * **كان يُقصّ إلى أحدث مئتين حين يبلغ الملفُّ ٦٤ ك.ب** — ربعُ ساعةٍ في طلب (نقطةٌ كلَّ أربع
 * ثوانٍ)، **ويُرسَل كلُّه في نداءٍ واحد** كان يردّه المحرّكُ فوق المئتين فيعلق. والآن:
 * - **قاعدةُ الأثر نفسُها قبل الحفظ** ([keep]، مرآةُ `drivermonitor.Keep`): نقطةٌ إن مرّت عشرون
 *   ثانيةً على آخر محفوظة أو تحرّك خمسةً وعشرين متراً — **فرجفةُ الواقف لا تملأ الطابور.**
 * - **والسقفُ بالعدد** ([MAX_POINTS]): ساعةٌ كاملةٌ على أسرع نبضةٍ تسمح بها اللوحة (ثلاث ثوانٍ).
 * - **وما شاخ ساعتين عن أحدث نقطةٍ لا يُرسَل** — المحرّكُ يرفضه أصلاً (`batchMaxAge`).
 * - **ويُرسَل أجزاءً** ([oldest])، **ويُمحى من كلّ جزءٍ ما أُرسل وحدَه** ([remove]) — فنقطةٌ أُضيفت
 *   أثناء نداء الشبكة تبقى (كان المسحُ يحذف الملفَّ كلَّه، `R19`).
 */
class PointQueue(
    private val file: File,
    private val warn: (String, Throwable) -> Unit = { _, _ -> },
) {

    constructor(context: Context) : this(
        File(context.filesDir, "points.jsonl"),
        { msg, e -> Log.w(TAG, msg, e) },
    )

    private val json = Json { ignoreUnknownKeys = true }

    /** **ما في الملفّ في الذاكرة** — الأقدمُ أوّلاً؛ يُقرأ مرّةً عند أوّل حاجة. */
    private var cache: MutableList<TrackPoint>? = null

    /** يضيف نقطةً إلى آخر الطابور — **إن مرّت بقاعدة الأثر.** يردّ هل حُفظت. */
    @Synchronized
    fun add(point: TrackPoint): Boolean {
        val list = load()
        if (!keep(list.lastOrNull(), point)) return false
        list.add(point)
        try {
            if (list.size > MAX_POINTS) {
                // **والقصُّ دفعةً لا مع كلّ نقطة** — يبقى ما دون السقف بمئة، فلا يُعاد كتبُ الملفّ
                // كلّه مع كلّ نقطةٍ في انقطاعٍ طويل. **والأحدثُ يبقى والأقدمُ يُرمى.**
                val newest = list.takeLast(MAX_POINTS - TRIM_STEP)
                list.clear()
                list.addAll(newest)
                rewrite(list)
            } else {
                file.appendText(encode(point) + "\n")
            }
        } catch (e: Exception) {
            warn("تعذّر حفظ النقطة", e)
        }
        return true
    }

    /**
     * **أقدمُ ما في الطابور — حتّى [max] نقطة** — جزءُ النداء التالي.
     *
     * **وما شاخ ساعتين عن أحدث نقطةٍ يُرمى أوّلاً** — يرفضه المحرّك، وإرسالُه حزمةٌ تُهدر.
     */
    @Synchronized
    fun oldest(max: Int): List<TrackPoint> {
        val list = load()
        val newest = list.mapNotNull { atMs(it) }.maxOrNull()
        if (newest != null) {
            val before = list.size
            list.removeAll { p -> atMs(p)?.let { it < newest - MAX_AGE_MS } ?: false }
            if (list.size != before) saveQuietly(list)
        }
        return list.take(max).toList()
    }

    /**
     * **يمحو ما أُرسل وحدَه** — كلُّ نقطةٍ مرّةً واحدة، **وما أُضيف أثناء النداء يبقى.**
     *
     * يردّ كم مُحي — وصفرٌ يعني أنّ الجزءَ لم يعد في الطابور.
     */
    @Synchronized
    fun remove(sent: List<TrackPoint>): Int {
        val list = load()
        var removed = 0
        for (p in sent) {
            if (list.remove(p)) removed++
        }
        if (removed > 0) saveQuietly(list)
        return removed
    }

    /** كم في الطابور. */
    @Synchronized
    fun size(): Int = load().size

    @Synchronized
    fun isEmpty(): Boolean = load().isEmpty()

    /** **يقرأ الملفَّ مرّةً** — وما كتبته نسخةٌ سابقةٌ يُقرأ ويُرسَل كذلك. */
    private fun load(): MutableList<TrackPoint> {
        cache?.let { return it }
        val list = mutableListOf<TrackPoint>()
        if (file.exists()) {
            try {
                file.readLines()
                    .filter { it.isNotBlank() }
                    // **وسطر معطوب لا يُسقط الباقي** — قد يُقطع الملفّ إن
                    // قُتل التطبيق أثناء الكتابة.
                    .mapNotNullTo(list) { line ->
                        runCatching { json.decodeFromString(TrackPoint.serializer(), line) }.getOrNull()
                    }
            } catch (e: Exception) {
                warn("تعذّرت قراءة الطابور", e)
            }
        }
        cache = list
        return list
    }

    private fun encode(p: TrackPoint): String = json.encodeToString(TrackPoint.serializer(), p)

    private fun rewrite(list: List<TrackPoint>) {
        if (list.isEmpty()) {
            file.delete()
            return
        }
        // **ملفٌّ جديدٌ ثمّ تبديل** — قتلُ التطبيق أثناء الكتابة لا يُفرغ الطابور.
        val tmp = File(file.parentFile, file.name + ".tmp")
        tmp.writeText(list.joinToString("\n", postfix = "\n") { encode(it) })
        if (!tmp.renameTo(file)) {
            file.delete()
            tmp.renameTo(file)
        }
    }

    private fun saveQuietly(list: List<TrackPoint>) {
        try {
            rewrite(list)
        } catch (e: Exception) {
            warn("تعذّر تحديث الطابور", e)
        }
    }

    companion object {
        private const val TAG = "RahalGo/queue"

        /**
         * **سقفُ الطابور بالعدد** — ساعةٌ كاملةٌ بلا شبكةٍ على أسرع نبضةٍ تسمح بها اللوحة
         * (ثلاثُ ثوانٍ أثناء الطلب، `drivers.location_ping_active_sec`)، **وأكثرُ بكثيرٍ دون ذلك.**
         */
        const val MAX_POINTS = 1200

        /** **كم يُقصّ تحت السقف حين يُبلَغ** — فلا يُعاد كتبُ الملفّ مع كلّ نقطة. */
        const val TRIM_STEP = 100

        /** **أقدمُ ما يُرسَل عن أحدث نقطة** — المحرّكُ يرفض ما جاوز الساعتين (`batchMaxAge`). */
        const val MAX_AGE_MS = 2 * 60 * 60 * 1000L

        /** **قاعدةُ الأثر** — مرآةُ `drivermonitor.KeepEvery` و`KeepMoveM` في المحرّك. */
        const val KEEP_EVERY_MS = 20_000L
        const val KEEP_MOVE_M = 25.0

        /**
         * **هل تُحفظ هذه النقطة؟** — دالةٌ صافية، **وهي قاعدةُ أثر المراقبة نفسُها**: أوّلُ نقطة،
         * أو مزيَّفةٌ (دائماً)، أو مرّ عشرون ثانيةً على آخر محفوظة، أو تحرّك خمسةً وعشرين متراً.
         * **ونقطةٌ أقدمُ من آخر محفوظة** (وصلت بغير ترتيبها) **أو بلا وقتٍ مقروء تُحفظ** — لا يُحكم
         * عليها بمرجعٍ لا يسبقها، والمحرّكُ يرتّب ويخفّف.
         */
        fun keep(prev: TrackPoint?, p: TrackPoint): Boolean {
            if (prev == null || p.mocked == true) return true
            val a = atMs(prev) ?: return true
            val b = atMs(p) ?: return true
            val dt = b - a
            if (dt < 0 || dt >= KEEP_EVERY_MS) return true
            return metersBetween(prev.lat, prev.lng, p.lat, p.lng) >= KEEP_MOVE_M
        }

        /**
         * **كلُّ نقاط الدفعة لا آخرُها** (A6.1، 0.9.0) — الأحدثُ يُرسَل حيّاً، **وما قبله يمرّ
         * بالطابور** فيصل الأثرَ في نداء الدفعة بعده. (كانت الأحدثُ وحدَها تُستعمل والباقي يُرمى.)
         *
         * يردّ (ما قبل الأحدث مرتّباً بالزمن، الأحدث).
         */
        fun <T> splitDelivered(fixes: List<T>, timeOf: (T) -> Long): Pair<List<T>, T?> {
            if (fixes.isEmpty()) return emptyList<T>() to null
            val sorted = fixes.sortedBy(timeOf)
            return sorted.dropLast(1) to sorted.last()
        }

        private fun atMs(p: TrackPoint): Long? =
            runCatching { java.time.Instant.parse(p.at).toEpochMilli() }.getOrNull()

        /** **المسافةُ بالمتر** — هافرساين بنصف القطر نفسِه في `routing.MetersBetween`. */
        private fun metersBetween(lat1: Double, lng1: Double, lat2: Double, lng2: Double): Double {
            val r = 6_371_000.0
            val p1 = Math.toRadians(lat1)
            val p2 = Math.toRadians(lat2)
            val dp = Math.toRadians(lat2 - lat1)
            val dl = Math.toRadians(lng2 - lng1)
            val h = Math.sin(dp / 2) * Math.sin(dp / 2) +
                Math.cos(p1) * Math.cos(p2) * Math.sin(dl / 2) * Math.sin(dl / 2)
            return 2 * r * Math.atan2(Math.sqrt(h), Math.sqrt(1 - h))
        }
    }
}
