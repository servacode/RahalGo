package com.rahalgo.driver.home

import com.rahalgo.driver.location.Readiness
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * **دفعةُ السائق ٢٠٢٦-١١-١١** — دمجُ ساعات «أفضل الأوقات»، وعدُّ ما ينقصه ليصله العمل.
 */
class DriverBatchTest {

    @Test
    fun `الساعات المتجاورة تندمج والبعيدة تبقى منفصلة`() {
        assertEquals(
            listOf(BusyWindow(13, 15), BusyWindow(19, 20)),
            BusyWindows.of(listOf(19, 14, 13)),
        )
        assertEquals(listOf(BusyWindow(9, 10)), BusyWindows.of(listOf(9, 9)))
        assertEquals(listOf(BusyWindow(23, 24)), BusyWindows.of(listOf(23, 99, -1)))
    }

    @Test
    fun `الساعة على اثنتي عشرة`() {
        assertEquals(12, BusyWindows.hour12(0))
        assertEquals(1, BusyWindows.hour12(13))
        assertEquals(12, BusyWindows.hour12(24))
        assertEquals(7, BusyWindows.hour12(19))
    }

    @Test
    fun `القائمة تعد ما تعده اللوحة`() {
        // الموقع «أثناء الاستخدام» فقط: اللوحة تقول ينقصك (1) والقائمة كانت تقول كل شيء جاهز.
        assertEquals(1, Readiness.missingCount(true, true, false, false, false, false))
        // جاهز تماماً.
        assertEquals(0, Readiness.missingCount(true, true, false, true, false, false))
        // لا موقع: سطر واحد، ولا يُعد الخلفية معه.
        assertEquals(1, Readiness.missingCount(false, true, false, false, false, false))
        // الإشارة ضائعة تُعد للعاملين فقط.
        assertEquals(1, Readiness.missingCount(true, true, true, true, false, false))
        assertEquals(0, Readiness.missingCount(true, false, true, true, false, false))
        // الكل ناقص.
        assertEquals(3, Readiness.missingCount(true, true, false, false, true, true))
    }
}
