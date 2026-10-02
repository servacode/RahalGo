package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **الوقتُ الباقي والتقريبُ الديناميكيّ** — المرحلتان ٣ و٤ من `docs/navigation/PLAN.md`.
 */
class EtaAndZoomTest {

    @Test
    fun `الوقت لا يقل عن حد المدينة — ١٫٣ كم ليست دقيقة`() {
        val eta = EtaSmoother()
        // **زمنُ المحرّك ٦٠ث لـ١٣٠٠م** — كما قِيس (٧٨ كم/س).
        val sec = eta.update(1300.0, 60.0, 0.0, 1300.0)
        assertTrue("الوقت $sec ث", sec >= 1300.0 / EtaSmoother.MAX_CITY_MPS - 0.01)
    }

    @Test
    fun `الوقت يتبع سرعته ولا يقفز`() {
        val eta = EtaSmoother()
        val first = eta.update(2000.0, 240.0, 0.0, 2000.0)
        // يسير بـ٥م/ث (١٨ كم/س) — المحسوبُ يصير أطول، والمعروضُ يقترب بربع الفرق.
        val second = eta.update(1995.0, 239.0, 5.0, 2000.0)
        val target = 0.5 * 239.0 + 0.5 * (1995.0 / 5.0)
        assertTrue("لم يتحرّك نحو سرعته", second > first)
        assertTrue("قفز إلى المحسوب دفعةً واحدة", second < target)
    }

    @Test
    fun `طريق جديد يؤخذ كما هو`() {
        val eta = EtaSmoother()
        eta.update(2000.0, 240.0, 0.0, 2000.0)
        val fresh = eta.update(3000.0, 400.0, 0.0, 3000.0)
        assertEquals(400.0, fresh, 0.01)
    }

    @Test
    fun `التقريب — واقف ١٧٫٥ وسريع أبعد وقرب المنعطف أقرب`() {
        assertEquals(NavCamera.NAV_ZOOM, NavCamera.zoomFor(0.0, -1.0), 1e-9)
        val fast = NavCamera.zoomFor(14.0, 800.0)
        assertEquals(16.5, fast, 0.01)
        assertEquals(NavCamera.NAV_ZOOM, NavCamera.zoomFor(14.0, 100.0), 1e-9)
        assertEquals(NavCamera.MIN_NAV_ZOOM, NavCamera.zoomFor(40.0, -1.0), 1e-9)
    }
}
