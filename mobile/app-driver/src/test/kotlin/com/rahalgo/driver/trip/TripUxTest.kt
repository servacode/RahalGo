package com.rahalgo.driver.trip

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

    @Test
    fun `edit request note field is gone and the draft sends an empty note`() {
        assertEquals("", EditDraft().note)
        val dir = listOf(File("src/main/res/values/strings.xml"), File("app-driver/src/main/res/values/strings.xml"))
            .firstOrNull { it.exists() }
        assertNotNull(dir)
        assertTrue(!dir!!.readText().contains("edit_req_note"))
    }
}
