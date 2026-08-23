package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رفائدُ الاتّجاه المعاكس**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٥، أمرُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 * # والرفيدةُ المرجعيّةُ هي «العودةُ على المسار نفسِه»
 *
 * (أمرُ المالك، البند ٢٦: «لا تعتمد فقط على Parallel Road لإثبات
 *  Wrong-Way».)
 *
 * **فالشارعُ الموازي استنتاجٌ نسبيٌّ للمسار** — قد يكون شارعاً آخرَ
 * أصلاً. **وأمّا من سار ثمّ استدار على الهندسة نفسِها فهو الحالُ
 * الصريحة.**
 */
object WrongWayFixtures {

    /** **يبني قراءةً عند مسافةٍ على المسار** — باتّجاه السير أو عكسه. */
    fun fixAt(
        route: NavRoute,
        meters: Double,
        atMs: Long,
        speedMps: Float,
        reverse: Boolean,
        accuracyM: Float = 6f,
        lateralM: Double = 0.0,
    ): NavFix {
        val p = RouteFixtures.pointAt(route, meters.coerceIn(0.0, route.totalM))
        val course = RouteFixtures.bearingAt(route, meters.coerceIn(0.0, route.totalM))
        val heading = if (reverse) GpsQuality.normalize(course + 180f) else course
        return NavFix(
            lat = p.lat,
            lng = p.lng + lateralM * 0.0000111,
            accuracyM = accuracyM,
            speedMps = speedMps,
            bearingDeg = heading,
            atMs = atMs,
        )
    }

    /**
     * **سيرٌ أماميٌّ ثمّ عودةٌ على المسار نفسِه.**
     *
     * `forwardTo` كم يتقدّم قبل أن يستدير · `backM` كم يرجع.
     *
     * **والتردّدُ معامل**: `stepMs` بين قراءتين، **والمسافةُ تتبع
     * السرعةَ لا القراءات** — فرحلةٌ عند نصف هرتزٍ وأخرى عند اثنين
     * تقطعان الشيءَ نفسَه.
     */
    fun forwardThenReverse(
        route: NavRoute,
        forwardTo: Double = 250.0,
        backM: Double = 120.0,
        speedMps: Float = 8f,
        stepMs: Long = 1_000L,
        accuracyM: Float = 6f,
        startMs: Long = 1_000L,
    ): List<NavFix> {
        val out = ArrayList<NavFix>()
        val step = speedMps * (stepMs / 1000.0)
        var t = startMs
        var d = 20.0
        while (d < forwardTo) {
            out += fixAt(route, d, t, speedMps, reverse = false, accuracyM = accuracyM)
            d += step
            t += stepMs
        }
        val turnAt = d
        while (d > turnAt - backM && d > 0) {
            out += fixAt(route, d, t, speedMps, reverse = true, accuracyM = accuracyM)
            d -= step
            t += stepMs
        }
        return out
    }

    /** **سيرٌ سليمٌ على المسار** — لا شيءَ يُشكّ فيه. */
    fun correctDirection(route: NavRoute, speedMps: Float = 8f): List<NavFix> {
        val out = ArrayList<NavFix>()
        var d = 20.0
        var t = 1_000L
        while (d < route.totalM - 20) {
            out += fixAt(route, d, t, speedMps, reverse = false)
            d += speedMps
            t += 1_000L
        }
        return out
    }

    /** **رجوعٌ قصيرٌ بمسافةٍ محدّدة** — ثمّ يتابع أماماً. */
    fun shortReverse(route: NavRoute, backM: Double, speedMps: Float = 4f): List<NavFix> {
        val out = ArrayList<NavFix>()
        var t = 1_000L
        var d = 150.0
        repeat(6) {
            out += fixAt(route, d, t, speedMps, reverse = false); d += speedMps; t += 1_000L
        }
        val from = d
        while (d > from - backM) {
            out += fixAt(route, d, t, speedMps, reverse = true); d -= speedMps; t += 1_000L
        }
        repeat(10) {
            out += fixAt(route, d, t, speedMps, reverse = false); d += speedMps; t += 1_000L
        }
        return out
    }

    /** **واقفٌ ينجرف** — والاتّجاهُ يدور مع الضجيج. */
    fun stationary(route: NavRoute): List<NavFix> =
        (0..14).map { i ->
            fixAt(route, 150.0, 1_000L + i * 1_000L, 0.2f, reverse = i % 2 == 0)
        }

    /** **رجفةُ اتّجاهٍ واحدةٌ** — قراءةٌ معكوسةٌ بين سليمات. */
    fun bearingGlitch(route: NavRoute): List<NavFix> {
        val base = correctDirection(route)
        return base.mapIndexed { i, f ->
            if (i == 8) f.copy(bearingDeg = GpsQuality.normalize((f.bearingDeg ?: 0f) + 180f)) else f
        }
    }

    /** **لفُّ الزاوية عند الشمال** — والمسارُ شماليّ. */
    fun wrapAround(route: NavRoute): List<NavFix> =
        correctDirection(route).mapIndexed { i, f ->
            f.copy(bearingDeg = if (i % 2 == 0) 359.5f else 0.5f)
        }

    /** **شارعٌ موازٍ معاكس** — قريبٌ هندسيّاً وعكسيُّ الاتّجاه. */
    fun parallelOpposite(route: NavRoute, lateralM: Double = 15.0): List<NavFix> {
        val out = ArrayList<NavFix>()
        var d = 300.0
        var t = 1_000L
        while (d > 60) {
            out += fixAt(route, d, t, 9f, reverse = true, lateralM = lateralM)
            d -= 9.0
            t += 1_000L
        }
        return out
    }

    /** **وموازٍ بالاتّجاه نفسِه** — لا يُحسم بلا Map Matching. */
    fun parallelSameDirection(route: NavRoute, lateralM: Double = 15.0): List<NavFix> {
        val out = ArrayList<NavFix>()
        var d = 60.0
        var t = 1_000L
        while (d < 400) {
            out += fixAt(route, d, t, 9f, reverse = false, lateralM = lateralM)
            d += 9.0
            t += 1_000L
        }
        return out
    }

    /** **دورانٌ مطلوبٌ على مسارٍ فيه `U_TURN`.** */
    fun uTurnRoute(): NavRoute = RouteFixtures.nearItself()

    /** **يمشي المسارَ الملتفَّ كما رُسم** — تنفيذٌ صحيح. */
    fun driveUTurn(): List<NavFix> {
        val r = uTurnRoute()
        return RouteFixtures.driveAlong(r, speedMps = 7f)
    }

    /** **خروجٌ هندسيٌّ حقيقيّ** — يجب أن تفوز حالُ الخروج. */
    fun geometricDeparture(route: NavRoute): List<NavFix> {
        val base = RouteFixtures.driveAlong(route, speedMps = 10f)
        return OffRouteFixtures.divergeFrom(base, 4, 12.0)
    }
}
