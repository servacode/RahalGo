package com.rahalgo.driver

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حرّاسُ دورة السائق — ما لا يُختبر بدالّةٍ صافية** (٢٠٢٦-١٠-٠٢)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ما يُقرأ من المصدر**: شاشةٌ ميّتةٌ لا تعود، وخطأُ الحديث لا يُبتلع، وخطأُ
 * الخطوة يُرى والبطاقةُ مطويّة.
 */
class DriverLifecycleGuardsTest {

    private val src = File("src/main/kotlin/com/rahalgo/driver")
    private fun read(path: String) = File(src, path).readText()

    @Test
    fun `the dead second detail screen is gone`() {
        assertFalse(File(src, "orders/OrderDetailScreen.kt").exists())
        assertFalse(read("MainActivity.kt").contains("OrderDetailScreen"))
    }

    @Test
    fun `a failed chat send is shown not swallowed`() {
        val vm = read("orders/OrdersViewModel.kt")
        val send = vm.substringAfter("fun sendMessage(").substringBefore("private suspend fun loadChat")
        assertTrue("خطأُ الإرسال يُكتب في الحال", send.contains("error = describe(e)"))
        assertTrue("والنصُّ يعود", send.contains("unsent = body"))
    }

    @Test
    fun `a step error is visible while the card is folded`() {
        val trip = read("trip/TripScreen.kt")
        assertTrue(trip.contains("if (!TripCollapse.bottom && state.onRouteOffer == null)"))
        assertTrue(trip.contains("TopNotice(state.error"))
    }
}
