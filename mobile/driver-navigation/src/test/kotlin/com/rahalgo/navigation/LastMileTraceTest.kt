package com.rahalgo.navigation

import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **تتبُّعٌ قبل الحكم — البند ١**
 * ══════════════════════════════════════════════════════════════════
 *
 * **لا يُدّعى عيبٌ ولا سلامةٌ بلا قياس.** هذه تطبع كلَّ حقلٍ عند كلّ
 * قراءةٍ بعد نهاية الخطّ، **فيُرى بالعين أيشتعل الخروجُ عن المسار
 * وإعادةُ الحساب أم لا.**
 *
 * (إغلاقُ آخر ميل، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 */
class LastMileTraceTest {

    @Test
    fun `تتبُّعُ آخرِ ميلٍ — مئةٌ وثمانون متراً خارجَ نهاية الخطّ`() {
        val engine = NavEngine(voice = VoicePlanner(), source = RecordingSource())
        val route = RouteFixtures.straight()
        engine.setRoute(route)

        val end = route.geometry.last()
        // **الهدفُ شمالَ نهاية الخطّ بمئةٍ وثمانين متراً.**
        val target = GeoPoint(end.lat + 180.0 / 111320.0, end.lng)
        engine.arrivalTarget = target

        println()
        println("  ── يقود المسارَ ─────────────────────────────────────")
        var t = 0L
        for (f in RouteFixtures.driveAlong(route)) {
            engine.onFix(f)
            t = f.atMs
        }
        dump(engine, route, target, "بلغ النهاية")

        println("  ── ثمّ يمشي نحو الهدف ──────────────────────────────")
        for (step in listOf(20, 40, 60, 90, 120, 150, 175, 180)) {
            t += 4000
            val p = GeoPoint(end.lat + step / 111320.0, end.lng)
            engine.onFix(NavFix(p.lat, p.lng, 6f, 2.0f, 0f, t))
            dump(engine, route, target, "على %3dم من النهاية".format(step))
        }
    }

    private fun dump(engine: NavEngine, route: NavRoute, target: GeoPoint, label: String) {
        val s = engine.state ?: return
        val p = s.progress
        val fromRoute = p?.offRouteM ?: -1.0
        val toTarget = s.targetDistanceM
        println(
            "  %-22s طور=%-24s arrived=%-5s خروج=%-20s حال=%-20s عكس=%-22s إعادة=%-14s جيل=%d · عن المسار=%6.1fم · عن الهدف=%6.1fم".format(
                label,
                s.arrivalPhase,
                p?.arrived,
                s.offRoute.state,
                s.situation,
                s.wrongWay.state,
                s.reroute,
                engine.generation,
                fromRoute,
                toTarget,
            ),
        )
    }


    @Test
    fun `وأثرٌ كثيفٌ بسرعةِ قيادة — لئلّا تكون السلامةُ مصادفة`() {
        /**
         * **أثرٌ قصيرٌ لا يُثبت شيئاً**: كاشفُ الخروج يؤكّد
         * بعدَ قراءات، **والواقفُ يُتخطّى أصلاً** (`STATIONARY`).
         *
         * **فهذا أربعون قراءةً بسرعة ستّةَ عشرَ متراً في الثانية**
         * ومسافةُ ابتعادٍ تبلغ خمسمئة — أضعافُ أيّ عتبة.
         */
        val src = RecordingSource()
        val engine = NavEngine(voice = VoicePlanner(), source = src)
        val route = RouteFixtures.straight()
        engine.setRoute(route)
        val end = route.geometry.last()
        engine.arrivalTarget = GeoPoint(end.lat + 500.0 / 111320.0, end.lng)

        var t = 0L
        for (f in RouteFixtures.driveAlong(route)) { engine.onFix(f); t = f.atMs }
        println()
        println("  بلغ النهاية — ثمّ أربعون قراءةً مبتعدةً بسرعة ١٦ م/ث")
        var worstOff = OffRouteDetector.State.ON_ROUTE
        var worstSit = NavSituation.ON_ROUTE
        var gen = engine.generation
        for (i in 1..40) {
            t += 1000
            val m = i * 12.5
            val p = GeoPoint(end.lat + m / 111320.0, end.lng)
            val st = engine.onFix(NavFix(p.lat, p.lng, 6f, 16f, 0f, t))
            if (st.offRoute.state.ordinal > worstOff.ordinal) worstOff = st.offRoute.state
            if (st.situation.ordinal > worstSit.ordinal) worstSit = st.situation
            if (i % 10 == 0) {
                dump(engine, route, engine.arrivalTarget!!, "بعد %2d قراءة".format(i))
                val v = st.offRoute
                println("      نقاط=%.2f عتبة=%.1f خروج=%.1f تخطٍّ=%s فوق=%s فاحش=%s اتّجاه=%s تعثّر=%s".format(
                    v.score, v.thresholdM, v.offRouteM, v.skip, v.overThreshold, v.grossly, v.bearingAgainst, v.stalled))
            }
        }
        println()
        println("  ══ الحصيلة ══")
        println("  أسوأُ حالِ خروج  : $worstOff")
        println("  أسوأُ حالٍ موحّدة: $worstSit")
        println("  طلباتُ إعادة الحساب: ${src.asked.size}")
        println("  الجيل: $gen → ${engine.generation}")
        println("  البُعدُ عن المسار: ${"%.0f".format(engine.state?.progress?.offRouteM ?: -1.0)}م")
    }

    /** **مصدرٌ يسجّل الطلبات ولا يشبك** — فيُعرف أطُلبت إعادةُ حساب. */
    class RecordingSource : RouteSource {
        val asked = mutableListOf<Triple<Double, Double, Long>>()
        override fun request(lat: Double, lng: Double, seq: Long, done: (RouteReply) -> Unit) {
            asked += Triple(lat, lng, seq)
            done(RouteReply.Failed(RerouteFailure.NETWORK))
        }
    }
}
