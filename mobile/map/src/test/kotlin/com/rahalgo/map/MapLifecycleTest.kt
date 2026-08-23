package com.rahalgo.map

import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **دورةُ الحياة والطبقات — البنود ١٩ و٢٨ و٢٩ و٣٠ و٤٠ و٤١**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك**: «اختبر: map created → low memory callback →
 * MapView receives it. ثم بعد destroy لا تستقبل شيئًا»، و«repeated
 * Compose recomposition لا تعيد lifecycle calls خطأ».
 */
class MapLifecycleTest {

    /** **مضيفٌ يعدّ ما وصله** — بلا `MapView` ولا جهاز. */
    private class Counting : MapLifecycleBridge.Host {
        val log = mutableListOf<String>()
        override fun onCreate() { log += "create" }
        override fun onStart() { log += "start" }
        override fun onResume() { log += "resume" }
        override fun onPause() { log += "pause" }
        override fun onStop() { log += "stop" }
        override fun onDestroy() { log += "destroy" }
        override fun onLowMemory() { log += "lowmem" }
        override fun onSaveInstanceState() { log += "save" }
        fun count(what: String) = log.count { it == what }
    }

    @Before
    fun setup() = MapLifecycleBridge.resetForTest()

    @After
    fun teardown() = MapLifecycleBridge.resetForTest()

    // ── TD-MAP-ONCREATE ───────────────────────────────────────────

    @Test
    fun `الدورةُ كاملةٌ ومرتَّبة`() {
        val host = Counting()
        val h = MapLifecycleBridge.register(host)
        h.create(); h.start(); h.resume()
        h.pause(); h.saveInstanceState(); h.stop(); h.destroy()
        assertEquals(
            listOf("create", "start", "resume", "pause", "save", "stop", "destroy"),
            host.log,
        )
    }

    @Test
    fun `onCreate مرّةً واحدةً مهما تكرّر النداء`() {
        // **البند ٣٠** — إعادةُ تركيب Compose تُنادي كثيراً.
        val host = Counting()
        val h = MapLifecycleBridge.register(host)
        repeat(10) { h.create() }
        assertEquals(1, host.count("create"))
    }

    @Test
    fun `onDestroy مرّةً واحدةً`() {
        val host = Counting()
        val h = MapLifecycleBridge.register(host)
        h.create()
        repeat(5) { h.destroy() }
        assertEquals(1, host.count("destroy"))
    }

    @Test
    fun `إعادةُ التركيب لا تُضاعف الأحداث`() {
        val host = Counting()
        val h = MapLifecycleBridge.register(host)
        repeat(20) { h.start(); h.resume() }
        assertEquals(1, host.count("start"))
        assertEquals(1, host.count("resume"))
    }

    @Test
    fun `دخولُ الشاشة وخروجُها دورةٌ متّزنة`() {
        val host = Counting()
        val h = MapLifecycleBridge.register(host)
        h.create()
        repeat(3) { h.start(); h.resume(); h.pause(); h.stop() }
        assertEquals(3, host.count("start"))
        assertEquals(3, host.count("resume"))
        assertEquals(3, host.count("pause"))
        assertEquals(3, host.count("stop"))
    }

    @Test
    fun `الهدمُ من حالٍ نشطٍ يمرّ بالمراحل`() {
        val host = Counting()
        val h = MapLifecycleBridge.register(host)
        h.create(); h.start(); h.resume()
        h.destroy()
        assertEquals(listOf("create", "start", "resume", "pause", "stop", "destroy"), host.log)
    }

    @Test
    fun `لا نداءَ بعد الهدم`() {
        val host = Counting()
        val h = MapLifecycleBridge.register(host)
        h.create(); h.destroy()
        val after = host.log.size
        h.start(); h.resume(); h.pause(); h.stop(); h.saveInstanceState()
        assertEquals(after, host.log.size)
    }

    // ── TD-MAP-LOWMEM ─────────────────────────────────────────────

    @Test
    fun `ضغطُ الذاكرة يصل كلَّ خريطةٍ حيّة`() {
        val a = Counting()
        val b = Counting()
        MapLifecycleBridge.register(a).create()
        MapLifecycleBridge.register(b).create()

        MapLifecycleBridge.dispatchLowMemory()
        assertEquals(1, a.count("lowmem"))
        assertEquals(1, b.count("lowmem"))
    }

    @Test
    fun `والمهدومةُ لا تستقبل شيئاً`() {
        val alive = Counting()
        val dead = Counting()
        MapLifecycleBridge.register(alive).create()
        val h = MapLifecycleBridge.register(dead)
        h.create()
        h.destroy()

        MapLifecycleBridge.dispatchLowMemory()
        assertEquals(1, alive.count("lowmem"))
        assertEquals("المهدومةُ لا تسمع", 0, dead.count("lowmem"))
    }

    @Test
    fun `السجلُّ لا يتسرّب`() {
        // **البند ٢٩** — «لا تنشئ global leaked references إلى Views».
        assertEquals(0, MapLifecycleBridge.liveCount())
        val handles = (1..5).map { MapLifecycleBridge.register(Counting()).also { h -> h.create() } }
        assertEquals(5, MapLifecycleBridge.liveCount())
        handles.forEach { it.destroy() }
        assertEquals("كلُّها نُزعت", 0, MapLifecycleBridge.liveCount())
    }

    @Test
    fun `تدويرُ الجهاز لا يُراكم خرائط`() {
        // **البند ٣٠** — «rotation/recomposition لا تنشئ MapViews متسربة».
        repeat(10) {
            val h = MapLifecycleBridge.register(Counting())
            h.create(); h.start(); h.resume()
            h.pause(); h.stop(); h.destroy()
        }
        assertEquals(0, MapLifecycleBridge.liveCount())
    }

    // ══════════════════════════════════════════════════════════════
    // **سجلُّ الطبقات** — البندان ١٩ و٤١
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `الطبقاتُ تُعاد بعد كلّ تحميل نمط`() {
        val registry = MapOverlayRegistry()
        var route = 0
        var markers = 0
        registry.register("route", MapOverlayRegistry.Priority.ROUTE) { route += 1 }
        registry.register("markers", MapOverlayRegistry.Priority.MARKERS) { markers += 1 }

        // **ثلاثةُ تحميلات** — أوّلٌ ثمّ تبديلان.
        repeat(3) { registry.restoreAll() }
        assertEquals(3, route)
        assertEquals(3, markers)
        assertEquals(3, registry.restoreCount)
    }

    @Test
    fun `الترتيبُ بالأولويّة لا بترتيب التسجيل`() {
        // **البند ٤٠** — الطرقُ ثمّ المسارُ ثمّ الدبابيس.
        val registry = MapOverlayRegistry()
        val order = mutableListOf<String>()
        registry.register("markers", MapOverlayRegistry.Priority.MARKERS) { order += "markers" }
        registry.register("zones", MapOverlayRegistry.Priority.ZONES) { order += "zones" }
        registry.register("route", MapOverlayRegistry.Priority.ROUTE) { order += "route" }
        registry.restoreAll()
        assertEquals(listOf("zones", "route", "markers"), order)
    }

    @Test
    fun `التسجيلُ بمعرِّفٍ لا يتراكم`() {
        val registry = MapOverlayRegistry()
        var n = 0
        repeat(50) { registry.register("route", MapOverlayRegistry.Priority.ROUTE) { n += 1 } }
        assertEquals(1, registry.size())
        registry.restoreAll()
        assertEquals("مرّةً لا خمسين", 1, n)
    }

    @Test
    fun `طبقةٌ تسقط لا تُخفي الباقيات`() {
        val registry = MapOverlayRegistry()
        var route = 0
        val errors = mutableListOf<String>()
        registry.register("zones", MapOverlayRegistry.Priority.ZONES) {
            throw IllegalStateException("مناطقُ التوصيل لم تُجلب")
        }
        registry.register("route", MapOverlayRegistry.Priority.ROUTE) { route += 1 }
        registry.restoreAll { id, _ -> errors += id }

        assertEquals("المسارُ رُسم رغم سقوط المناطق", 1, route)
        assertEquals(listOf("zones"), errors)
    }

    @Test
    fun `النزعُ يُخرج من السجلّ`() {
        val registry = MapOverlayRegistry()
        registry.register("route", MapOverlayRegistry.Priority.ROUTE) { }
        registry.unregister("route")
        assertEquals(0, registry.size())
    }

    @Test
    fun `المسارُ يعود والدبابيسُ تعود ومناطقُ التوصيل`() {
        // **البند ٣٩** — كلُّ ما فوق النمط يُختبر عودتُه.
        val registry = MapOverlayRegistry()
        val restored = mutableSetOf<String>()
        for (id in listOf("route-line", "driver-marker", "pickup", "dropoff", "zones")) {
            registry.register(id, MapOverlayRegistry.Priority.ROUTE) { restored += id }
        }
        registry.restoreAll()
        assertEquals(
            setOf("route-line", "driver-marker", "pickup", "dropoff", "zones"),
            restored,
        )
    }
}
