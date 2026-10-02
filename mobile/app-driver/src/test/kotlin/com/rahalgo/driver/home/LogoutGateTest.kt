package com.rahalgo.driver.home

import com.rahalgo.driver.R
import com.rahalgo.shared.model.DriverOrder
import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **الخروجُ ببوّابة** (٢٠٢٦-١٠-٠٢) — كان الخروجُ يمسح الجلسةَ وحدَها: **الورديّةُ
 * مفتوحةٌ في الخادم، وخدمةُ الموقع تعمل، والطلبُ في يده يتيم.**
 */
class LogoutGateTest {

    @Test
    fun `an open order blocks logout with a clear message`() {
        assertEquals(R.string.logout_has_order, LogoutGate.block(listOf(DriverOrder(id = "x", status = "assigned"))))
        assertNull(LogoutGate.block(emptyList()))
    }

    /** **والبوّابةُ موصولةٌ فعلاً** — لا دالّةٌ ميّتةٌ كما كانت `HomeViewModel.logout`. */
    @Test
    fun `the drawer and the home screen go through the gate, and the logout hook stops the service`() {
        val src = File("src/main/kotlin/com/rahalgo/driver")
        val main = File(src, "MainActivity.kt").readText()
        assertTrue("القائمة تخرج بالبوّابة", main.contains("onLogout = gatedLogout"))
        assertTrue("اللوحة تخرج بالبوّابة", main.contains("logout = gatedLogout"))
        assertTrue(main.contains("home.logout(onLogout)"))
        val backend = File(src, "data/Backend.kt").readText()
        assertTrue("مِعراضُ الخروج يُوقف الخدمة", backend.contains("afterLogout") && backend.contains("LocationService.stop"))
    }
}
