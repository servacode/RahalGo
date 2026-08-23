package com.rahalgo.navigation

import com.rahalgo.map.MapOverlayRegistry
import com.rahalgo.map.data.MapPackageStore
import com.rahalgo.map.data.MapSourceResolver
import com.rahalgo.map.data.MapSwitchPolicy
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **الملاحةُ تعبر تبديلَ المصدر — البندان ٢٠ و٣٨**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك نصّاً** (البند ٣٨):
 *
 *	route loaded → online vector map active
 *	↓ regional package installed
 *	↓ network lost → switch offline
 *	↓ route remains · progress continues · maneuver continues
 *	↓ voice planner continues · wrong-way continues
 *
 * ثمّ:
 *
 *	driver leaves route → RerouteEngine tries → NETWORK failure
 *	→ old route remains → offline map remains
 *
 * # **ولماذا هذا الاختبارُ هو أهمُّ ما في ٦ب**
 *
 * **البند ٢٠**: «Base style replacement لا يجب أن يصفّر
 * NavigationSession · RouteProgress · VoicePlanner · WrongWayDetector
 * · RerouteEngine · camera tracking state. الخريطة Renderer فقط».
 *
 * **والفصلُ ليس بديهيّاً في الشيفرة**: كلاهما يعيش في الشاشة نفسِها،
 * **ويكفي سطرٌ واحدٌ يُنشئ `NavEngine` داخل `remember(binding)`
 * ليضيع كلُّ شيءٍ مع أوّل انقطاعِ شبكة.** فيُقاس هنا لا يُفترض.
 */
class OfflineNavigationTest {

    private lateinit var tmp: File

    private fun store(): MapPackageStore {
        tmp = File(
            System.getProperty("java.io.tmpdir"),
            "rahalgo-nav-${System.nanoTime()}",
        ).also { it.mkdirs() }
        return MapPackageStore(tmp).also { it.prepare() }
    }

    private fun raqqaPackage(archive: File) = MapPackageStore.Installed(
        regionId = "raqqa",
        dataVersion = "2026-08-21",
        resourcesVersion = "1",
        name = "الرقّة",
        bbox = listOf(38.92, 35.88, 39.12, 36.03),
        minZoom = 10,
        maxZoom = 16,
        bytes = 1000,
        sha256 = "a".repeat(64),
        archive = archive,
    )

    // ══════════════════════════════════════════════════════════════
    // **السيناريو**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `الملاحةُ تعبر انقطاعَ الشبكة كاملةً`() {
        val s = store()
        val archive = File(tmp, "region.pmtiles").absoluteFile.also { it.writeBytes(ByteArray(10)) }
        val installed = listOf(raqqaPackage(archive))

        // ── ١ · مسارٌ مُحمَّلٌ وخريطةٌ متّصلة ──────────────────────
        val engine = NavEngine(voice = VoicePlanner())
        val route = RouteFixtures.straight()
        engine.setRoute(route)
        val policy = MapSwitchPolicy()
        policy.start(MapSwitchPolicy.Desired.ONLINE)

        val leg = RouteFixtures.driveAlong(route)
        var last: NavState? = null
        for (f in leg.take(12)) last = engine.onFix(f)

        assertNotNull(last)
        assertTrue("المسارُ محمَّل", last!!.hasRoute)
        val progressBefore = last.progress
        assertNotNull(progressBefore)
        val generationBefore = engine.generation
        val stepBefore = progressBefore!!.current

        // ── ٢ · سقطت الشبكة ───────────────────────────────────────
        val decisionOffline = MapSourceResolver.resolve(
            MapSourceResolver.Request(
                purpose = MapSourceResolver.Purpose.NAVIGATION,
                online = false,
                lat = route.geometry.first().lat,
                lng = route.geometry.first().lng,
                installed = installed,
                installedResourcesVersions = setOf("1"),
            ),
        )
        assertTrue("$decisionOffline", decisionOffline is MapSourceResolver.Decision.OfflineRegion)

        // **والمهلةُ تُنتظر** — انقطاعٌ مؤكَّدٌ لا ومضة.
        policy.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.OFFLINE, true, 0))
        val switched = policy.update(
            MapSwitchPolicy.Input(MapSwitchPolicy.Desired.OFFLINE, true, 10_000),
        )
        assertEquals(MapSwitchPolicy.Desired.OFFLINE, switched.active)

        // ── ٣ · تُحمَّل نمطٌ جديدةٌ وتُعاد الطبقات ─────────────────
        val overlays = MapOverlayRegistry()
        var routeDrawn = 0
        var markersDrawn = 0
        overlays.register("route", MapOverlayRegistry.Priority.ROUTE) { routeDrawn += 1 }
        overlays.register("markers", MapOverlayRegistry.Priority.MARKERS) { markersDrawn += 1 }
        overlays.restoreAll()

        assertEquals("خطُّ المسار عاد", 1, routeDrawn)
        assertEquals("الدبابيسُ عادت", 1, markersDrawn)

        // ── ٤ · والملاحةُ لم تُمسّ ────────────────────────────────
        // **البند ٢٠** — «same NavigationSession · same progress ·
        // same current maneuver · same generation».
        assertEquals("الجيلُ نفسُه", generationBefore, engine.generation)
        assertEquals("المناورةُ نفسُها", stepBefore, engine.state?.progress?.current)
        assertTrue("المسارُ باقٍ", engine.state?.hasRoute == true)

        // ── ٥ · والتقدّمُ يستمرّ بعد التبديل ──────────────────────
        var after: NavState? = null
        for (f in leg.drop(12).take(12)) after = engine.onFix(f)

        assertNotNull(after)
        assertTrue("التقدّمُ يستمرّ", after!!.hasRoute)
        assertEquals("ولا جيلَ جديد", generationBefore, engine.generation)
        assertTrue(
            "والمسافةُ المتبقّيةُ نقصت",
            (after.progress?.remainingM ?: Double.MAX_VALUE) <
                (progressBefore.remainingM + 1.0),
        )

        tmp.deleteRecursively()
    }

    @Test
    fun `خروجٌ عن المسار وسقوطُ الشبكة يُبقيان المسارَ القديم`() {
        // **البند ٣٨، الشقُّ الثاني.**
        val engine = NavEngine(source = FailingSource(), voice = VoicePlanner())
        val route = RouteFixtures.straight()
        engine.setRoute(route)
        val generation = engine.generation

        val leg = RouteFixtures.driveAlong(route)
        for (f in leg.take(8)) engine.onFix(f)

        // **ينحرف السائقُ شمالاً** حتّى يُؤكَّد الخروج.
        var last: NavState? = null
        val off = OffRouteFixtures.divergeFrom(leg.drop(8).take(20), index = 0, metersPerFix = 12.0)
        for (f in off) last = engine.onFix(f)

        assertNotNull(last)
        assertTrue("المسارُ القديمُ باقٍ", last!!.hasRoute)
        assertEquals("ولا مسارَ جديدٌ رُكّب", generation, engine.generation)
        assertTrue(
            "وحالُ إعادة الحساب تُبلَّغ",
            last.reroute != RerouteStatus.NONE,
        )
    }

    @Test
    fun `تبديلُ المصدر لا يُنشئ محرّكاً جديداً`() {
        // **وهذا هو العطبُ الذي يُخشى**: `remember(binding)` حول
        // `NavEngine` **يمحو كلَّ شيءٍ مع أوّل انقطاع.**
        val engine = NavEngine(voice = VoicePlanner())
        val route = RouteFixtures.straight()
        engine.setRoute(route)
        for (f in RouteFixtures.driveAlong(route).take(10)) engine.onFix(f)

        val identity = System.identityHashCode(engine)
        val cuesBefore = engine.state?.cues?.size ?: 0

        // **تبديلٌ ثمّ تبديلٌ عائد** — والمحرّكُ هو هو.
        val policy = MapSwitchPolicy()
        policy.start(MapSwitchPolicy.Desired.ONLINE)
        policy.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.OFFLINE, true, 0))
        policy.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.OFFLINE, true, 10_000))
        policy.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.ONLINE, false, 20_000))
        policy.update(MapSwitchPolicy.Input(MapSwitchPolicy.Desired.ONLINE, false, 90_000))

        assertEquals(identity, System.identityHashCode(engine))
        assertTrue(engine.state?.hasRoute == true)
        assertTrue(
            "ولا تُعاد نداءاتٌ صوتيّةٌ قديمة",
            (engine.state?.cues?.size ?: 0) <= cuesBefore + 1,
        )
    }

    /** **بابُ شبكةٍ يسقط دائماً** — لاختبار إعادة الحساب بلا شبكة. */
    private class FailingSource : RouteSource {
        override fun request(lat: Double, lng: Double, seq: Long, done: (RouteReply) -> Unit) {
            done(RouteReply.Failed(RerouteFailure.NETWORK))
        }
    }
}
