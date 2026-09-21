package com.rahalgo.customer

import com.rahalgo.customer.orders.mergeById
import com.rahalgo.shared.customer.MyOrder
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **دمجُ صفحاتِ السجلّ — بلا تكرارٍ وبحفظ الترتيب** (`CAF-14`، `CUST-14-026`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **`mergeById` دالّةٌ صافيةٌ تُقاس بلا جهاز** — وهي قلبُ الترقيم: **صفحةٌ
 * تُضاف لا تكرّر بطاقةً ولا تقلب صفّا.**
 */
class OrdersMergeTest {

    private fun o(id: String) = MyOrder(id = id)
    private fun ids(list: List<MyOrder>) = list.map { it.id }

    @Test
    fun appendsDistinctInOrder() {
        val r = mergeById(listOf(o("a"), o("b")), listOf(o("c"), o("d")))
        assertEquals(listOf("a", "b", "c", "d"), ids(r))
    }

    @Test
    fun dedupesOverlap() {
        // **«b» في الصفحتين** (سباقُ إنعاش) — يبقى مرّةً وبموضعه الأوّل.
        val r = mergeById(listOf(o("a"), o("b")), listOf(o("b"), o("c")))
        assertEquals(listOf("a", "b", "c"), ids(r))
    }

    @Test
    fun emptyIncomingKeepsExisting() {
        val r = mergeById(listOf(o("a")), emptyList())
        assertEquals(listOf("a"), ids(r))
    }

    @Test
    fun emptyExistingTakesIncoming() {
        val r = mergeById(emptyList(), listOf(o("a"), o("b")))
        assertEquals(listOf("a", "b"), ids(r))
    }

    @Test
    fun fullOverlapAddsNothing() {
        val r = mergeById(listOf(o("a"), o("b")), listOf(o("a"), o("b")))
        assertEquals(listOf("a", "b"), ids(r))
    }
}
