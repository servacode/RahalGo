package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import kotlin.random.Random

/**
 * ══════════════════════════════════════════════════════════════════════
 * **موضعُ الملاحة — الوقوفُ والإلصاق** (المرحلة ٢، `docs/navigation/PLAN.md`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-١٠-٠٢: «السهمُ يختفي ويظهر، ومرّةً للخلف ومرّةً للأمام».)
 *
 * **والضجيجُ هنا عشوائيٌّ لا نقطةٌ مكرّرة** — اختبارُ الوقوف القديم (`RouteFixtures.standStill`)
 * يكرّر النقطةَ نفسَها **فلم يكشف زحفَ التقدّم** (فحصُ الملاحة ٢.٣).
 */
class PositionEngineTest {

    private val rnd = Random(20261002)

    /** **إزاحةٌ عشوائيّةٌ بالأمتار** حول نقطة. */
    private fun jitter(lat: Double, lng: Double, m: Double): Pair<Double, Double> =
        (lat + (rnd.nextDouble() * 2 - 1) * m * 0.000009) to
            (lng + (rnd.nextDouble() * 2 - 1) * m * 0.0000111)

    @Test
    fun `الواقف — السهم لا يهتزّ والسرعة المزعومة بدقّةٍ رديئةٍ لا تحرّكه`() {
        val engine = NavEngine()
        engine.setRoute(RouteFixtures.straight())
        val shown = ArrayList<Pair<Double, Double>>()
        for (i in 0 until 300) {
            val (lat, lng) = jitter(RouteFixtures.north(200.0), RouteFixtures.LNG0, 8.0)
            // **كما قِيس على الجهاز داخلَ البيت**: سرعةٌ مزعومةٌ ٧٫٤م/ث بدقّةٍ ٣٠–٤٠م.
            val bad = i % 3 == 0
            val fix = NavFix(
                lat, lng,
                if (bad) 38f else 9f,
                if (bad) 7.4f else 0.3f,
                if (bad) 280f else null,
                1_000L + i * 1_000L,
            )
            val s = engine.onFix(fix)
            if (i > 0) shown.add(s.lat!! to s.lng!!)
        }
        val first = shown.first()
        val drift = shown.maxOf { GpsQuality.metersBetween(first.first, first.second, it.first, it.second) }
        println("POS-001 · أقصى انحرافٍ للسهم في ٥ دقائق وقوف = ${"%.2f".format(drift)}م")
        assertEquals(PositionEngine.Motion.STATIONARY, engine.position.motion)
        assertTrue("السهمُ تحرّك ${drift}م والسائقُ واقف", drift < 0.5)
    }

    @Test
    fun `الواقف — التقدّم لا يزحف`() {
        val engine = NavEngine()
        val route = RouteFixtures.straight()
        engine.setRoute(route)
        // **أوّلاً يمشي إلى منتصف الطريق** ثمّ يقف.
        val drive = RouteFixtures.driveAlong(route).filter {
            GpsQuality.metersBetween(it.lat, it.lng, RouteFixtures.LAT0, RouteFixtures.LNG0) < 300
        }
        drive.forEach { engine.onFix(it) }
        val stopAt = drive.last()
        var t = stopAt.atMs
        // ثلاثُ ثوانٍ بطيئةٌ تُنهي الحركة، ثمّ خمسُ دقائق ضجيج.
        val progressAtStop = ArrayList<Double>()
        for (i in 0 until 303) {
            t += 1_000L
            val (lat, lng) = jitter(stopAt.lat, stopAt.lng, 8.0)
            val s = engine.onFix(NavFix(lat, lng, 9f, 0.2f, null, t))
            if (i > 10) progressAtStop += s.progress?.progressM ?: -1.0
            if (System.getenv("POS_DEBUG") != null && i < 40) println("DBG i=$i motion=${engine.position.motion} grade=${s.grade} p=${s.progress?.progressM} rem=${s.progress?.remainingM}")
        }
        val creep = progressAtStop.max() - progressAtStop.min()
        println("POS-002 · زحفُ التقدّم في ٥ دقائق وقوف = ${"%.2f".format(creep)}م")
        assertTrue("التقدّمُ زحف ${creep}م والسائقُ واقف", creep < 0.5)
    }

    @Test
    fun `السائر — يلتصق بالطريق رغم ضجيج جانبيّ ±8م`() {
        val engine = NavEngine()
        val route = RouteFixtures.long(40)
        engine.setRoute(route)
        val fixes = RouteFixtures.driveAlong(route, speedMps = 10f, accuracyM = 9f)
        var on = 0
        var total = 0
        fixes.forEachIndexed { i, f ->
            val (lat, lng) = jitter(f.lat, f.lng, 8.0)
            val s = engine.onFix(f.copy(lat = lat, lng = lng))
            if (i > 3) {
                total++
                val p = engine.currentRoute!!.let { r -> PositionEngine.pointAt(r, s.progress!!.progressM)!! }
                if (GpsQuality.metersBetween(p.lat, p.lng, s.lat!!, s.lng!!) < 0.5) on++
            }
        }
        val share = on * 100.0 / total
        println("POS-003 · على الخطّ = ${"%.1f".format(share)}٪ من $total قراءة")
        assertTrue("على الخطّ ${share}٪ فقط", share >= 95.0)
    }

    @Test
    fun `البعيد عن الطريق لا يُلصق به`() {
        val engine = NavEngine()
        val route = RouteFixtures.long(40)
        engine.setRoute(route)
        val fixes = RouteFixtures.driveAlong(route, speedMps = 10f, accuracyM = 6f).take(20)
        fixes.forEach { engine.onFix(it) }
        val last = fixes.last()
        // ٦٠م شرقاً بدقّةٍ جيّدة — ليس على هذا الطريق.
        val away = last.copy(lng = last.lng + 60 * 0.0000111, atMs = last.atMs + 1_000L)
        val s = engine.onFix(away)
        val d = GpsQuality.metersBetween(s.lat!!, s.lng!!, away.lat, away.lng)
        assertTrue("أُلصق بطريقٍ يبعد ٦٠م (${d}م عن القراءة)", d < 1.0)
    }

    @Test
    fun `النقطة على الطريق واتّجاهه`() {
        val route = RouteFixtures.straight()
        val p = PositionEngine.pointAt(route, 100.0)!!
        assertEquals(RouteFixtures.north(100.0), p.lat, 0.00002)
        val b = PositionEngine.routeBearingAt(route, 100.0)!!
        assertTrue("اتّجاهُ طريقٍ شماليّ = $b", b < 2f || b > 358f)
    }
}
