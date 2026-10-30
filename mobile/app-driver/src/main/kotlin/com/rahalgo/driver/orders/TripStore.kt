package com.rahalgo.driver.orders

import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.OrderRoute
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull
import kotlinx.serialization.json.put

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الرحلةُ القائمة تُحفظ على القرص** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت في الذاكرة وحدَها** — فمن أُغلق تطبيقُه بلا إنترنت فتحه على شاشةٍ فارغة: لا طلبَ
 * ولا طريقَ ولا وجهة. **والآن تُكتب بعد كلّ قراءةٍ ناجحة، وتُستعاد حين تفشل القراءةُ الأولى.**
 */
data class SavedTrip(
    val mine: List<DriverOrder>,
    val route: OrderRoute?,
    val leg: String?,
    val openId: String?,
    val savedAtMs: Long,
)

object TripStore {

    const val FILE = "driver-active-trip.json"

    /** **رحلةٌ أقدمُ من هذا لا تُستعاد** — طلبٌ من أمسِ أُغلق غالباً من المكتب. */
    const val MAX_AGE_MS = 12 * 60 * 60 * 1000L

    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
    }

    private val orders = ListSerializer(DriverOrder.serializer())

    fun encode(trip: SavedTrip): String = buildJsonObject {
        put("v", 1)
        put("saved_at", trip.savedAtMs)
        put("mine", json.encodeToJsonElement(orders, trip.mine))
        trip.route?.let { put("route", json.encodeToJsonElement(OrderRoute.serializer(), it)) }
        trip.leg?.let { put("leg", it) }
        trip.openId?.let { put("open_id", it) }
    }.toString()

    fun decode(text: String): SavedTrip? = runCatching {
        val o = json.parseToJsonElement(text) as JsonObject
        SavedTrip(
            mine = json.decodeFromJsonElement(orders, o["mine"] ?: return null),
            route = o["route"]?.let { json.decodeFromJsonElement(OrderRoute.serializer(), it) },
            leg = o["leg"]?.jsonPrimitive?.contentOrNull,
            openId = o["open_id"]?.jsonPrimitive?.contentOrNull,
            savedAtMs = o["saved_at"]?.jsonPrimitive?.longOrNull ?: 0L,
        )
    }.getOrNull()

    /** **يُقرأ إن وُجد وكان حديثاً وفيه طلب.** */
    fun read(dir: java.io.File, nowMs: Long): SavedTrip? {
        val f = java.io.File(dir, FILE)
        if (!f.exists()) return null
        val t = decode(f.readText()) ?: return null
        if (t.mine.isEmpty() || nowMs - t.savedAtMs > MAX_AGE_MS) return null
        return t
    }

    /** **يكتب الرحلةَ كاملةً بدل القديمة.** */
    fun write(dir: java.io.File, text: String) = atomicWrite(java.io.File(dir, FILE), text)

    fun clear(dir: java.io.File) {
        java.io.File(dir, FILE).delete()
        java.io.File(dir, StepQueue.FILE).delete()
    }

    /** **كتابةٌ لا تُبقي ملفّاً نصفيّاً** — يُكتب جانباً ثمّ يُبدَّل. */
    internal fun atomicWrite(f: java.io.File, text: String) {
        val tmp = java.io.File(f.parentFile, f.name + ".tmp")
        tmp.writeText(text)
        if (!tmp.renameTo(f)) {
            f.delete()
            tmp.renameTo(f)
        }
    }
}
