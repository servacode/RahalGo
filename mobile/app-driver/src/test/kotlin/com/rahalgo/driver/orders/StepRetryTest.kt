package com.rahalgo.driver.orders

import java.io.File
import java.io.IOException
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test

/**
 * **خطوةٌ ضاع ردُّها تُعاد بمفتاحها نفسِه** (٢٠٢٦-١٠-٠٢) — ورفضُ المحرّك لا يُعاد.
 */
class StepRetryTest {

    @Test
    fun `a lost response is retried once with the same key`() = runBlocking {
        val keys = mutableListOf<String>()
        StepRetry.send("k-1") { key ->
            keys += key
            if (keys.size == 1) throw IOException("انقطعت")
        }
        assertEquals(listOf("k-1", "k-1"), keys)
    }

    @Test
    fun `an engine refusal is not retried`() = runBlocking {
        var calls = 0
        try {
            StepRetry.send("k-2") {
                calls++
                throw IllegalStateException("invalid_transition")
            }
            fail("ابتُلع رفضُ المحرّك")
        } catch (e: IllegalStateException) {
            assertEquals(1, calls)
        }
    }

    /** **وكلُّ خطوةٍ في نموذج الطلبات تحمل مفتاحاً، وكلُّ خطأٍ يُتبَع بقراءة.** */
    @Test
    fun `steps carry a key and errors refresh`() {
        val src = File("src/main/kotlin/com/rahalgo/driver/orders/OrdersViewModel.kt").readText()
        val bare = Regex("""backend\.driver\.transition\(id, "[a-z_]+"\)""").findAll(src).count()
        assertEquals("انتقالٌ بلا مفتاح", 0, bare)
        assertTrue(src.contains("idempotencyKey = key"))
    }
}
