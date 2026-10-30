package com.rahalgo.driver.orders

import com.rahalgo.shared.model.DriverOrder
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خطواتٌ تنتظر الإنترنت** (قرارُ المالك ٢٠٢٦-١٠-٣٠: «الرحلة تكمّل بلا إنترنت»)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وصلت · استلمت · انطلقت · سلّمت البضاعة (الإرجاع)** — تُحفظ على القرص بمفاتيحها حين
 * تنقطع الشبكة، **والشاشةُ تتقدّم**، ثمّ تُرسَل بترتيبها وبالمفاتيح نفسِها حين تعود
 * (`replayOwnTransition` في الخادم يردّ على المفتاح المكرّر بالطلب كما هو).
 *
 * **و«سلّمت» ليست منها** — تحتاج كودَ الزبون يُفحص في الخادم.
 */
data class PendingStep(
    val orderId: String,
    val to: String,
    val key: String,
    val lat: Double? = null,
    val lng: Double? = null,
)

object StepQueue {

    /** **نهايةُ مشوار الإرجاع** — ليست انتقالاً بل نداءَ `goods-handed`. */
    const val GOODS = "goods_handed"

    /** **ما يُحفظ حين تنقطع الشبكة** — ولا «سلّمت». */
    val QUEUEABLE = setOf("at_pickup", "picked_up", "on_the_way", "at_dropoff", GOODS)

    const val FILE = "driver-step-queue.json"

    private val json = Json { ignoreUnknownKeys = true }

    /** **ترتيبُ الحالات** — فلا ترجع الشاشةُ خطوةً إلى الوراء. */
    private val RANK = listOf("assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff")

    /** **الحالُ التي تُرى بعد الخطوة** — «استلمت» تعني «في الطريق» (تُرسَل معها). */
    fun statusAfter(to: String): String? = when (to) {
        "picked_up", "on_the_way" -> "on_the_way"
        "at_pickup" -> "at_pickup"
        "at_dropoff" -> "at_dropoff"
        else -> null
    }

    /**
     * **القائمةُ كما يجب أن تُرى** — حالاتُ الخطوات المنتظرة فوق ما قاله الخادم، **وإلى الأمام
     * وحدَه**، ومشوارُ إرجاعٍ سُلّمت بضاعتُه يخرج.
     */
    fun overlay(mine: List<DriverOrder>, pending: List<PendingStep>): List<DriverOrder> {
        if (pending.isEmpty()) return mine
        val gone = pending.filter { it.to == GOODS }.map { it.orderId }.toSet()
        return mine.filterNot { it.id in gone }.map { o ->
            var status = o.status
            for (p in pending) {
                if (p.orderId != o.id) continue
                val next = statusAfter(p.to) ?: continue
                if (RANK.indexOf(next) > RANK.indexOf(status)) status = next
            }
            if (status == o.status) o else o.copy(status = status)
        }
    }

    /** **حصيلةُ إعادة الإرسال.** */
    data class Replay(
        /** **ما بقي ينتظر** — الشبكةُ انقطعت ثانيةً عند أوّله. */
        val remaining: List<PendingStep>,
        /** **ما رفضه الخادم** — يُقال للسائق ولا يُعاد. */
        val rejected: List<Pair<PendingStep, Exception>>,
    )

    /**
     * **يُرسل بالترتيب وبالمفاتيح نفسِها.**
     *
     * **انقطاعٌ يوقف الإرسال** ويُبقي الباقي كما هو. **ورفضُ الخادم** يُسقط الخطوةَ وما بعدها
     * لطلبها (لن تُقبل بعد رفضِ ما قبلها)، **ويُكمل لغيره.**
     */
    suspend fun replay(pending: List<PendingStep>, send: suspend (PendingStep) -> Unit): Replay {
        val rejected = mutableListOf<Pair<PendingStep, Exception>>()
        val dead = mutableSetOf<String>()
        var i = 0
        while (i < pending.size) {
            val p = pending[i]
            if (p.orderId in dead) {
                i++
                continue
            }
            try {
                send(p)
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: Exception) {
                if (StepRetry.retriable(e)) {
                    return Replay(pending.subList(i, pending.size).filterNot { it.orderId in dead }, rejected)
                }
                rejected += p to e
                dead += p.orderId
            }
            i++
        }
        return Replay(emptyList(), rejected)
    }

    fun encode(pending: List<PendingStep>): String = buildJsonArray {
        for (p in pending) {
            add(
                buildJsonObject {
                    put("order_id", p.orderId)
                    put("to", p.to)
                    put("key", p.key)
                    p.lat?.let { put("lat", it) }
                    p.lng?.let { put("lng", it) }
                },
            )
        }
    }.toString()

    fun decode(text: String): List<PendingStep> = runCatching {
        (json.parseToJsonElement(text) as JsonArray).mapNotNull { el ->
            val o = el.jsonObject
            val id = o["order_id"]?.jsonPrimitive?.content ?: return@mapNotNull null
            val to = o["to"]?.jsonPrimitive?.content ?: return@mapNotNull null
            val key = o["key"]?.jsonPrimitive?.content ?: return@mapNotNull null
            if (to !in QUEUEABLE) return@mapNotNull null
            PendingStep(id, to, key, o["lat"]?.jsonPrimitive?.doubleOrNull, o["lng"]?.jsonPrimitive?.doubleOrNull)
        }
    }.getOrDefault(emptyList())

    fun read(dir: java.io.File): List<PendingStep> {
        val f = java.io.File(dir, FILE)
        return if (f.exists()) decode(f.readText()) else emptyList()
    }

    fun write(dir: java.io.File, pending: List<PendingStep>) {
        val f = java.io.File(dir, FILE)
        if (pending.isEmpty()) {
            f.delete()
            return
        }
        TripStore.atomicWrite(f, encode(pending))
    }
}
