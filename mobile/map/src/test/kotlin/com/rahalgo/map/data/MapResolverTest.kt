package com.rahalgo.map.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **اختيارُ المصدر وسياسةُ التبديل — البنود ١٥ و١٦ و١٧ و١٨ و٣٧**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك**: «لا تبدّل إلى Offline package لا تغطي الموضع…
 * استخدم bbox / region contract المعلن في Manifest»، و«منع online ↔
 * offline ↔ online بسبب شبكة متذبذبة».
 */
class MapResolverTest {

    private fun pkg(
        id: String,
        west: Double, south: Double, east: Double, north: Double,
        version: String = "2026-08-21",
        resourcesVersion: String = "1",
    ) = MapPackageStore.Installed(
        regionId = id,
        dataVersion = version,
        resourcesVersion = resourcesVersion,
        name = id,
        bbox = listOf(west, south, east, north),
        minZoom = 10,
        maxZoom = 16,
        bytes = 1000,
        sha256 = "a".repeat(64),
        archive = File("/tmp/$id-$version.pmtiles"),
    )

    private val raqqa = pkg("raqqa", 38.92, 35.88, 39.12, 36.03)
    private val aleppo = pkg("aleppo", 36.9, 36.0, 37.4, 36.4)
    /** **صندوقٌ أوسعُ يشمل الرقّة** — لاختبار الأخصّ. */
    private val north = pkg("north", 36.0, 35.0, 40.0, 37.0)

    private val ready = setOf("1")

    private fun request(
        online: Boolean,
        lat: Double?,
        lng: Double?,
        installed: List<MapPackageStore.Installed>,
        routeBbox: List<Double>? = null,
    ) = MapSourceResolver.Request(
        purpose = MapSourceResolver.Purpose.NAVIGATION,
        online = online,
        lat = lat,
        lng = lng,
        routeBbox = routeBbox,
        installed = installed,
        installedResourcesVersions = ready,
    )

    // ── اختيارُ الحزمة ────────────────────────────────────────────

    @Test
    fun `تُختار الحزمةُ التي تغطّي الموضع`() {
        val best = MapSourceResolver.bestCovering(
            listOf(aleppo, raqqa), 35.95, 39.01, ready,
        )
        assertEquals("raqqa", best?.regionId)
    }

    @Test
    fun `ولا تُختار حزمةٌ لا تغطّي الموضع`() {
        // **البند ١٧** — «لا تختَرها لمجرد أنها آخر حزمة مثبتة».
        val best = MapSourceResolver.bestCovering(listOf(raqqa), 36.20, 37.15, ready)
        assertEquals(null, best)
    }

    @Test
    fun `الأخصُّ يُقدَّم على الأوسع`() {
        // **البند ١٦** — «اختر الأكثر تحديدًا».
        val best = MapSourceResolver.bestCovering(
            listOf(north, raqqa), 35.95, 39.01, ready,
        )
        assertEquals("raqqa", best?.regionId)
    }

    @Test
    fun `الأحدثُ يُقدَّم عند تساوي الصندوق`() {
        val old = pkg("raqqa", 38.92, 35.88, 39.12, 36.03, version = "2026-08-01")
        val best = MapSourceResolver.bestCovering(
            listOf(old, raqqa), 35.95, 39.01, ready,
        )
        assertEquals("2026-08-21", best?.dataVersion)
    }

    @Test
    fun `حزمةٌ بمواردَ غيرِ مركَّبةٍ ليست خياراً`() {
        // **البند ١٣** — «ولا تستخدم Region مع Style/Glyphs غير متوافقة».
        val orphan = pkg("raqqa", 38.92, 35.88, 39.12, 36.03, resourcesVersion = "9")
        assertEquals(null, MapSourceResolver.bestCovering(listOf(orphan), 35.95, 39.01, ready))
    }

    @Test
    fun `بلا موضعٍ لا تُختار حزمة`() {
        assertEquals(null, MapSourceResolver.bestCovering(listOf(raqqa), null, null, ready))
    }

    // ── القرار ────────────────────────────────────────────────────

    @Test
    fun `لا شبكةَ وحزمةٌ تغطّي فأوفلاين`() {
        val d = MapSourceResolver.resolve(request(false, 35.95, 39.01, listOf(raqqa)))
        assertTrue("$d", d is MapSourceResolver.Decision.OfflineRegion)
    }

    @Test
    fun `لا شبكةَ ولا حزمةَ فلا مصدر`() {
        // **البند ٣٧** — لا انهيار، ولا عودةَ إلى راستر OSM.
        val d = MapSourceResolver.resolve(request(false, 35.95, 39.01, emptyList()))
        assertTrue("$d", d is MapSourceResolver.Decision.Unavailable)
    }

    @Test
    fun `لا شبكةَ وحزمةٌ لا تغطّي فلا مصدر`() {
        val d = MapSourceResolver.resolve(request(false, 36.20, 37.15, listOf(raqqa)))
        assertTrue("$d", d is MapSourceResolver.Decision.Unavailable)
        assertTrue((d as MapSourceResolver.Decision.Unavailable).why.contains("تغطّي"))
    }

    @Test
    fun `شبكةٌ متاحةٌ فأونلاين`() {
        val d = MapSourceResolver.resolve(request(true, 35.95, 39.01, listOf(raqqa)))
        assertTrue("$d", d is MapSourceResolver.Decision.Online)
    }

    @Test
    fun `مسارٌ يتجاوز الصندوق يُرجّح الأونلاين`() {
        // **البند ١٧** — «إذا Network متوفرة، Online source أفضل
        // لتغطية Route كاملة».
        val d = MapSourceResolver.resolve(
            request(true, 35.95, 39.01, listOf(raqqa), routeBbox = listOf(37.0, 35.5, 39.1, 36.5)),
        )
        assertTrue("$d", d is MapSourceResolver.Decision.Online)
        assertTrue((d as MapSourceResolver.Decision.Online).why.contains("صندوق"))
    }

    @Test
    fun `تغطيةُ المسار تُقاس بالصندوق`() {
        assertTrue(MapSourceResolver.routeCovered(raqqa, listOf(38.95, 35.90, 39.10, 36.00)))
        assertFalse(MapSourceResolver.routeCovered(raqqa, listOf(37.00, 35.90, 39.10, 36.00)))
        assertTrue("لا مسارَ فلا سؤال", MapSourceResolver.routeCovered(raqqa, null))
    }

    // ══════════════════════════════════════════════════════════════
    // **سياسةُ التبديل** — البند ١٨
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `ومضةُ شبكةٍ لا تُبدّل المصدر`() {
        val p = MapSwitchPolicy()
        p.start(MapSwitchPolicy.Desired.ONLINE)

        // **سقطت في اللحظة صفر وعادت بعد ثانيتين** — ولا تبديل.
        p.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.OFFLINE, false, 0))
        p.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.OFFLINE, false, 2_000))
        val back = p.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.ONLINE, false, 2_500))

        assertEquals(MapSwitchPolicy.Desired.ONLINE, back.active)
        assertEquals(MapSwitchPolicy.State.ONLINE_ACTIVE, back.state)
    }

    @Test
    fun `انقطاعٌ مؤكَّدٌ يُبدّل إلى أوفلاين`() {
        val p = MapSwitchPolicy()
        p.start(MapSwitchPolicy.Desired.ONLINE)
        p.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.OFFLINE, false, 0))
        val out = p.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.OFFLINE, false, 7_000))
        assertEquals(MapSwitchPolicy.Desired.OFFLINE, out.active)
        assertEquals(MapSwitchPolicy.State.OFFLINE_ACTIVE, out.state)
    }

    @Test
    fun `العودةُ إلى الاتّصال أبطأُ من الذهاب`() {
        val p = MapSwitchPolicy()
        p.start(MapSwitchPolicy.Desired.OFFLINE)
        p.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.ONLINE, false, 0))
        // **بعد سبعِ ثوانٍ لا تكفي** — مهلةُ العودة خمسٌ وأربعون.
        val early = p.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.ONLINE, false, 7_000))
        assertEquals(MapSwitchPolicy.Desired.OFFLINE, early.active)
        val late = p.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.ONLINE, false, 50_000))
        assertEquals(MapSwitchPolicy.Desired.ONLINE, late.active)
    }

    @Test
    fun `الاتّصالُ يعود أثناء ملاحةٍ عاملةٍ فلا تبديل`() {
        // **أمرُ المالك نصّاً**: «Network يعود → لا ضرورة لتبديل فوري
        // أثناء Navigation إذا Offline تعمل جيدًا».
        val p = MapSwitchPolicy()
        p.start(MapSwitchPolicy.Desired.OFFLINE)
        val out = p.update(
            MapSwitchPolicy.Input(MapSwitchPolicy.Desired.ONLINE, navigating = true, nowMs = 600_000),
        )
        assertEquals(MapSwitchPolicy.Desired.OFFLINE, out.active)
        assertEquals(MapSwitchPolicy.State.OFFLINE_ACTIVE, out.state)
    }

    @Test
    fun `انعدامُ المصدر يُبلَّغ فوراً بلا مهلة`() {
        val p = MapSwitchPolicy()
        p.start(MapSwitchPolicy.Desired.ONLINE)
        val out = p.update(
            MapSwitchPolicy.Input(MapSwitchPolicy.Desired.UNAVAILABLE, false, 0),
        )
        assertEquals(MapSwitchPolicy.Desired.UNAVAILABLE, out.active)
    }

    @Test
    fun `التذبذبُ لا يُنتج تبديلاً واحداً`() {
        // **عشرون تغيّراً في عشرين ثانية** — ولا تبديل.
        val p = MapSwitchPolicy()
        p.start(MapSwitchPolicy.Desired.ONLINE)
        var switches = 0
        var last = MapSwitchPolicy.Desired.ONLINE
        for (i in 0 until 20) {
            val want = if (i % 2 == 0) {
                MapSwitchPolicy.Desired.OFFLINE
            } else {
                MapSwitchPolicy.Desired.ONLINE
            }
            val out = p.update(MapSwitchPolicy.Input(want, false, i * 1000L))
            if (out.active != last) switches += 1
            last = out.active
        }
        assertEquals("لا تبديلَ مع شبكةٍ متذبذبة", 0, switches)
    }
}
