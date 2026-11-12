package com.rahalgo.driver.trip

import com.rahalgo.driver.orders.approxDistanceText
import com.rahalgo.driver.orders.legDistanceText
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/** **ملاحظات المالك ٢٠٢٦-١١-١٠ على شاشة الرحلة** — الاسم العامّ والمسافة الصفريّة وحقل الملاحظة. */
class TripUxTest {

    @Test
    fun `generic or blank customer name means no name`() {
        assertNull(customerShownName(""))
        assertNull(customerShownName("   "))
        assertNull(customerShownName("زبون"))
        assertNull(customerShownName("الزبون"))
        assertEquals("أحمد", customerShownName(" أحمد "))
    }

    @Test
    fun `zero or unknown leg distance is hidden`() {
        assertEquals("", legDistanceText(0.0))
        assertEquals("", legDistanceText(0.4))
        assertEquals("", legDistanceText(-1.0))
        assertEquals("56 م", legDistanceText(56.0))
        assertEquals("1.4 كم", legDistanceText(1400.0))
    }

    /** **بطاقةُ العرض بخطٍّ مستقيم — تُقال تقريبيّة** (تجربةُ المالك ٢٠٢٦-١١-١٢). */
    @Test
    fun `offer card distance is marked approximate`() {
        assertEquals("≈ 371 م", approxDistanceText(371.0))
        assertEquals("", approxDistanceText(-1.0))
        val src = File("src/main/kotlin/com/rahalgo/driver/orders/OrdersScreen.kt").takeIf { it.exists() }
            ?: File("app-driver/src/main/kotlin/com/rahalgo/driver/orders/OrdersScreen.kt")
        val text = src.readText()
        assertTrue("البطاقة تكتب المسافة بلا «≈»", !text.contains("far = legDistanceText("))
        assertTrue("سطرُ «تقريبية» غائب", text.contains("R.string.card_dist_approx"))
    }

    @Test
    fun `edit request note field is gone and the draft sends an empty note`() {
        assertEquals("", EditDraft().note)
        val dir = listOf(File("src/main/res/values/strings.xml"), File("app-driver/src/main/res/values/strings.xml"))
            .firstOrNull { it.exists() }
        assertNotNull(dir)
        assertTrue(!dir!!.readText().contains("edit_req_note"))
    }

    @Test
    fun `zero distance tail is stripped from the address, others stay`() {
        assertEquals("حي الرشيد · قرب فرن", cleanAddress("حي الرشيد · قرب فرن (0 م)"))
        assertEquals("حي الرشيد", cleanAddress("حي الرشيد (0.0 م)"))
        assertEquals("قرب فرن (40 م)", cleanAddress("قرب فرن (40 م)"))
    }

    @Test
    fun `arabic minute plurals`() {
        assertEquals("1 دقيقة", com.rahalgo.ui.minutes(1))
        assertEquals("2 دقيقتان", com.rahalgo.ui.minutes(2))
        assertEquals("3 دقائق", com.rahalgo.ui.minutes(3))
        assertEquals("10 دقائق", com.rahalgo.ui.minutes(10))
        assertEquals("11 دقيقة", com.rahalgo.ui.minutes(11))
        assertEquals("25 دقيقة", com.rahalgo.ui.minutes(25))
    }

    @Test
    fun `filled add fields are added on send, empty ones are not`() {
        val d = EditDraft().withPendingAdd(" كولا ", 2, 500, null, "")
        assertEquals(1, d.adds.size)
        assertEquals("كولا", d.adds[0].name)
        assertEquals(EditDraft(), EditDraft().withPendingAdd("", 1, 500, null, ""))
        assertEquals(EditDraft(), EditDraft().withPendingAdd("كولا", 1, 0, null, ""))
    }

    @Test
    fun `bidi isolates wrap numbers and amounts`() {
        val out = com.rahalgo.ui.BidiText.isolateNumbers("طلب #1201 – +100 ل.س")
        assertTrue(out.contains("⁨#1201⁩"))
        assertTrue(out.contains("⁨+100 ل.س⁩"))
        assertEquals("⁨x⁩".length, com.rahalgo.driver.Bidi.isolate("x").length)
    }
}
