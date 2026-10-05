package com.rahalgo.driver.orders

import java.io.File
import kotlinx.coroutines.CoroutineStart
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import kotlinx.coroutines.yield
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **فتحُ الورديّة يُعيد قراءةَ الطابور** — طلبٌ نزل قبل الفتح كان لا يظهر
 * حتّى يُعاد فتحُ التطبيق (رُئي على المحاكي ٢٠٢٦-١٠-٠٥).
 */
class QueuePulseTest {

    private fun src(rel: String): String {
        var dir = File("").absoluteFile
        repeat(6) {
            val f = File(dir, "app-driver/src/main/kotlin/com/rahalgo/driver/$rel")
            if (f.exists()) return f.readText()
            val g = File(dir, "src/main/kotlin/com/rahalgo/driver/$rel")
            if (g.exists()) return g.readText()
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد $rel")
    }

    @Test
    fun `النبضة تصل من يسمعها`() = runBlocking {
        val got = async(start = CoroutineStart.UNDISPATCHED) { withTimeout(2000) { QueuePulse.flow.first() } }
        yield()
        QueuePulse.bump()
        got.await()
        Unit
    }

    @Test
    fun `الورديّة تنبض والطابور يسمع`() {
        assertTrue(
            "فتحُ الورديّة لا يُعيد قراءةَ الطابور",
            src("home/HomeViewModel.kt").contains("QueuePulse.bump()"),
        )
        assertTrue(
            "قائمةُ الطلبات لا تسمع نبضةَ الورديّة",
            src("orders/OrdersViewModel.kt").contains("QueuePulse.flow.collect { refresh() }"),
        )
    }
}
