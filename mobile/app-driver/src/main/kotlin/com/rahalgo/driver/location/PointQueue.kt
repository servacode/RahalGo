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
 * # وسقفٌ لا ينمو بلا حدّ
 *
 * **سائق بلا شبكة ساعتين يكتب مئات النقاط** — والمحرّك يرفض ما مضى عليه
 * أكثر من ساعتين أصلاً (`batchMaxAge`)، **فحفظُ ما لا يُقبل ملء قرصٍ بلا
 * فائدة.** فيُبقى الأحدث ويُرمى الأقدم.
 */
class PointQueue(context: Context) {

    private val file = File(context.filesDir, "points.jsonl")
    private val json = Json { ignoreUnknownKeys = true }

    /** يضيف نقطة إلى آخر الطابور. */
    @Synchronized
    fun add(point: TrackPoint) {
        try {
            file.appendText(json.encodeToString(TrackPoint.serializer(), point) + "\n")
            trim()
        } catch (e: Exception) {
            Log.w(TAG, "تعذّر حفظ النقطة", e)
        }
    }

    /** ما في الطابور — **والأقدم أوّلاً.** */
    @Synchronized
    fun all(): List<TrackPoint> {
        if (!file.exists()) return emptyList()
        return try {
            file.readLines()
                .filter { it.isNotBlank() }
                // **وسطر معطوب لا يُسقط الباقي** — قد يُقطع الملفّ إن
                // قُتل التطبيق أثناء الكتابة.
                .mapNotNull { line ->
                    runCatching { json.decodeFromString(TrackPoint.serializer(), line) }.getOrNull()
                }
        } catch (e: Exception) {
            Log.w(TAG, "تعذّرت قراءة الطابور", e)
            emptyList()
        }
    }

    @Synchronized
    fun clear() {
        runCatching { file.delete() }
    }

    @Synchronized
    fun isEmpty(): Boolean = !file.exists() || file.length() == 0L

    /** **يُبقي الأحدث** — والمحرّك يرفض ما جاوز الساعتين. */
    private fun trim() {
        if (file.length() < MAX_BYTES) return
        val lines = file.readLines().takeLast(MAX_POINTS)
        file.writeText(lines.joinToString("\n", postfix = "\n"))
    }

    private companion object {
        const val TAG = "RahalGo/queue"
        // **سقفُ الدفعة في المحرّك ٢٠٠** — فلا يُحفظ أكثر ممّا يُرسل مرّة.
        const val MAX_POINTS = 200
        const val MAX_BYTES = 64 * 1024L
    }
}
