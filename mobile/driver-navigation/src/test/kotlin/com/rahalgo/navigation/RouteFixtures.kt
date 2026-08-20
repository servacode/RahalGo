package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رفائدُ المرحلة ٢ — مصطنعةٌ لا مسجَّلة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٠، والفرقُ مسجَّلٌ في `QA_ARCHITECTURE.md`.)
 *
 * **وهذه تختبر منطقَ الملاحة وحدَه** — **ولا تقول شيئاً عن دقّة قمرٍ
 * صناعيٍّ ولا عن بطّاريّةٍ ولا عن إحساسٍ بصريّ.** تلك تنتظر رحلةً
 * حقيقيّةً تُسجَّل بـ`TraceRecorder`.
 *
 * # وهندستُها من الرقّة بإحداثيّاتها
 *
 * **`0.00001` درجةِ عرضٍ ≈ متر** — فتُقرأ المسافاتُ في الرأس.
 *
 * # وأنواعُ المناورات وأسماءُ الدوّارات من قياسٍ حقيقيّ
 *
 * **`دوار النعيم` و`exit=2`** قِيسا من محرّكنا على مسار `short-raqqa`
 * (٢٠٢٦-٠٨-٢٠) — **لا أسماءٌ مخترعة.**
 */
object RouteFixtures {

    const val LAT0 = 35.9506
    const val LNG0 = 39.0094

    /** **متراً شمالاً.** */
    fun north(m: Double) = LAT0 + m * 0.000009

    /** **متراً شرقاً** — ودرجةُ الطول تضيق مع العرض. */
    fun east(m: Double) = LNG0 + m * 0.0000111

    private fun pt(latM: Double, lngM: Double) = GeoPoint(north(latM), east(lngM))

    private fun man(
        kind: String,
        atM: Double,
        stepM: Double,
        stepS: Double,
        modifier: String? = null,
        street: String? = null,
        exit: Int? = null,
        rname: String? = null,
    ) = NavManeuver(kind, modifier, atM, 0, stepM, stepS, street, exit, rname)

    // ══════════════════════════════════════════════════════════════════
    // **مستقيمٌ — خمسُ مئةِ مترٍ شمالاً**
    // ══════════════════════════════════════════════════════════════════

    fun straight(): NavRoute = NavRoute.of(
        (0..10).map { pt(it * 50.0, 0.0) },
        listOf(
            man(ManeuverKinds.DEPART, 0.0, 500.0, 50.0),
            man(ManeuverKinds.ARRIVE, 500.0, 0.0, 0.0),
        ),
    )

    // ══════════════════════════════════════════════════════════════════
    // **انعطافٌ واحد — ٣٠٠م شمالاً ثمّ ٣٠٠م شرقاً**
    // ══════════════════════════════════════════════════════════════════

    fun singleRight(): NavRoute = NavRoute.of(
        (0..6).map { pt(it * 50.0, 0.0) } + (1..6).map { pt(300.0, it * 50.0) },
        listOf(
            man(ManeuverKinds.DEPART, 0.0, 300.0, 30.0, "straight"),
            man(ManeuverKinds.TURN_RIGHT, 300.0, 300.0, 30.0, "right", "Noor Street"),
            man(ManeuverKinds.ARRIVE, 600.0, 0.0, 0.0),
        ),
    )

    fun singleLeft(): NavRoute = NavRoute.of(
        (0..6).map { pt(it * 50.0, 0.0) } + (1..6).map { pt(300.0, -it * 50.0) },
        listOf(
            man(ManeuverKinds.DEPART, 0.0, 300.0, 30.0),
            man(ManeuverKinds.TURN_LEFT, 300.0, 300.0, 30.0, "left"),
            man(ManeuverKinds.ARRIVE, 600.0, 0.0, 0.0),
        ),
    )

    // ══════════════════════════════════════════════════════════════════
    // **مناورتان متقاربتان — ثمانيةُ أمتارٍ بينهما**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وقِيس هذا فعلاً** على مسار `city-multi-turn`: خطوةٌ طولُها
    // ثمانيةُ أمتارٍ بين انعطافين. **وسماحٌ ثابتٌ بخمسةَ عشرَ متراً
    // يبتلع إحداهما.**

    fun closeManeuvers(): NavRoute = NavRoute.of(
        (0..4).map { pt(it * 25.0, 0.0) } +
            listOf(pt(100.0, 8.0)) +
            (1..4).map { pt(100.0 + it * 25.0, 8.0) },
        listOf(
            man(ManeuverKinds.DEPART, 0.0, 100.0, 12.0),
            man(ManeuverKinds.TURN_RIGHT, 100.0, 8.0, 6.0, "right"),
            man(ManeuverKinds.TURN_LEFT, 108.0, 100.0, 12.0, "left"),
            man(ManeuverKinds.ARRIVE, 208.0, 0.0, 0.0),
        ),
    )

    // ══════════════════════════════════════════════════════════════════
    // **دوّارٌ باسمٍ ورقمِ مخرج**
    // ══════════════════════════════════════════════════════════════════

    fun roundabout(): NavRoute = NavRoute.of(
        (0..4).map { pt(it * 50.0, 0.0) } + (1..6).map { pt(200.0, it * 50.0) },
        listOf(
            man(ManeuverKinds.DEPART, 0.0, 200.0, 20.0),
            man(
                ManeuverKinds.ROUNDABOUT, 200.0, 40.0, 8.0, "right",
                exit = 2, rname = "دوار النعيم",
            ),
            man(ManeuverKinds.EXIT_ROUNDABOUT, 240.0, 260.0, 26.0, "straight", exit = 2),
            man(ManeuverKinds.ARRIVE, 500.0, 0.0, 0.0),
        ),
    )

    // ══════════════════════════════════════════════════════════════════
    // **بلا أسماءٍ ومعدِّلٍ ونوعٍ معروف**
    // ══════════════════════════════════════════════════════════════════
    //
    // **و٨٣٪ من خطوات الرقّة بلا اسم** — فهذه الحالُ العاديّة.

    fun unnamedAndUnknown(): NavRoute = NavRoute.of(
        (0..8).map { pt(it * 40.0, 0.0) },
        listOf(
            man(ManeuverKinds.DEPART, 0.0, 120.0, 12.0, modifier = null, street = null),
            man("TELEPORT_SOMETHING", 120.0, 100.0, 10.0, modifier = null, street = null),
            man(ManeuverKinds.TURN_RIGHT, 220.0, 100.0, 10.0, "right", null),
            man(ManeuverKinds.ARRIVE, 320.0, 0.0, 0.0),
        ),
    )

    // ══════════════════════════════════════════════════════════════════
    // **مسارٌ يمرّ قربَ نفسِه — ذهابٌ وعودةٌ بعشرين متراً**
    // ══════════════════════════════════════════════════════════════════
    //
    // **وهو ما يجعل «أقربَ نقطةٍ في المسار كلِّه» تقفز من ٢٥٪ إلى
    // ٧٠٪** — انظر `RouteProjector`.

    fun nearItself(): NavRoute = NavRoute.of(
        (0..10).map { pt(it * 40.0, 0.0) } +
            (9 downTo 0).map { pt(it * 40.0, 20.0) },
        listOf(
            man(ManeuverKinds.DEPART, 0.0, 400.0, 40.0),
            man(ManeuverKinds.U_TURN, 400.0, 400.0, 40.0, "uturn"),
            man(ManeuverKinds.ARRIVE, 820.0, 0.0, 0.0),
        ),
    )

    /** **شارعان متوازيان** — يفصلهما الاتّجاهُ لا المسافة. */
    fun parallelLike(): NavRoute = nearItself()

    // ══════════════════════════════════════════════════════════════════
    // **مسارٌ طويل — لقياس الأداء**
    // ══════════════════════════════════════════════════════════════════
    //
    // **٢١١٣ رأساً هو ما قِيس على الرقّة–دمشق.**

    fun long(vertices: Int = 2113): NavRoute {
        val pts = (0 until vertices).map { pt(it * 200.0, 0.0) }
        val total = (vertices - 1) * 200.0
        return NavRoute.of(
            pts,
            listOf(
                man(ManeuverKinds.DEPART, 0.0, total, total / 20.0),
                man(ManeuverKinds.ARRIVE, total, 0.0, 0.0),
            ),
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **مجارٍ من القراءات**
    // ══════════════════════════════════════════════════════════════════

    /** **سيرٌ على المسار** — قراءةٌ كلَّ ثانيةٍ بسرعةٍ ثابتة. */
    fun driveAlong(
        route: NavRoute,
        speedMps: Float = 10f,
        accuracyM: Float = 6f,
        startMs: Long = 1_000L,
    ): List<NavFix> {
        val out = ArrayList<NavFix>()
        var d = 0.0
        var t = startMs
        while (d <= route.totalM) {
            val p = pointAt(route, d)
            val b = bearingAt(route, d)
            out.add(NavFix(p.lat, p.lng, accuracyM, speedMps, b, t))
            d += speedMps.toDouble()
            t += 1_000
        }
        return out
    }

    /** **موضعٌ على المسار بمسافةٍ من بدايته.** */
    fun pointAt(route: NavRoute, meters: Double): GeoPoint {
        val i = RouteProjector.indexAtOrBefore(route.cumulativeM, meters)
        if (i >= route.geometry.size - 1) return route.geometry.last()
        val segLen = route.cumulativeM[i + 1] - route.cumulativeM[i]
        val t = if (segLen <= 0) 0.0 else ((meters - route.cumulativeM[i]) / segLen).coerceIn(0.0, 1.0)
        val a = route.geometry[i]
        val b = route.geometry[i + 1]
        return GeoPoint(a.lat + (b.lat - a.lat) * t, a.lng + (b.lng - a.lng) * t)
    }

    fun bearingAt(route: NavRoute, meters: Double): Float {
        val i = RouteProjector.indexAtOrBefore(route.cumulativeM, meters)
        val j = minOf(i + 1, route.geometry.size - 1)
        if (i == j) return 0f
        val a = route.geometry[i]
        val b = route.geometry[j]
        return GpsQuality.courseBetween(
            NavFix(a.lat, a.lng, 0f, null, null, 0),
            NavFix(b.lat, b.lng, 0f, null, null, 0),
        )
    }

    /** **يضيف رجفةً جانبيّة** — بلا تقدّمٍ حقيقيّ. */
    fun jitter(fixes: List<NavFix>, meters: Double = 8.0): List<NavFix> =
        fixes.mapIndexed { i, f ->
            val sign = if (i % 2 == 0) 1 else -1
            f.copy(lng = f.lng + sign * meters * 0.0000111)
        }

    /** **وقوفٌ في موضعٍ واحد** — سرعةٌ صفرٌ وقراءاتٌ متتالية. */
    fun standStill(at: NavFix, count: Int, everyMs: Long = 1_000L): List<NavFix> =
        (1..count).map { at.copy(speedMps = 0.1f, atMs = at.atMs + it * everyMs) }
}
