package com.rahalgo.navigation

import kotlin.math.abs

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رفائدُ الانحراف — مصطنعةٌ لا مسجَّلة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٣أ، أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
 *
 * **وتُبنى على رفائد المرحلة ٢** (`RouteFixtures`) — ولا يُصنع مسارٌ
 * من الصفر: **مسارٌ جديدٌ لكلّ اختبارٍ يعني أنّ الاختبارَ يفحص المسارَ
 * لا الكاشف.**
 *
 * **وهذه تفحص منطقَ القرار وحدَه** — ولا تقول شيئاً عن دقّة قمرٍ
 * صناعيٍّ حقيقيّ. **تلك تنتظر رحلةً تُسجَّل بـ`TraceRecorder`**، وهو
 * الفرقُ المسجَّلُ في `QA_ARCHITECTURE.md`.
 */
object OffRouteFixtures {

    /** **يُشغّل مجرًى على محرّكٍ ويردّ الحالات.** */
    fun replay(engine: NavEngine, fixes: List<NavFix>): List<NavState> =
        fixes.map { engine.onFix(it) }

    /** **سلسلةُ الحالات كما تُقرأ في المتوقَّع.** */
    fun states(out: List<NavState>): List<OffRouteDetector.State> = out.map { it.offRoute.state }

    /** **محرّكٌ بمسار** — بأرقامٍ افتراضيّةٍ أو مضبوطة. */
    fun engineOn(route: NavRoute, tuning: OffRouteDetector.Tuning = OffRouteDetector.Tuning()): NavEngine {
        val e = NavEngine(detector = OffRouteDetector(tuning))
        e.setRoute(route)
        return e
    }

    // ══════════════════════════════════════════════════════════════════
    // **إزاحةُ مجرًى عن مساره**
    // ══════════════════════════════════════════════════════════════════

    /** **يزيح القراءاتِ شرقاً بمتر** — بلا مسّ اتّجاهٍ ولا سرعة. */
    fun shiftEast(fixes: List<NavFix>, meters: Double): List<NavFix> =
        fixes.map { it.copy(lng = it.lng + meters * 0.0000111) }

    /** **يزيح ابتداءً من قراءةٍ بعينها، بازديادٍ مطّرد.** */
    fun divergeFrom(fixes: List<NavFix>, index: Int, metersPerFix: Double): List<NavFix> =
        fixes.mapIndexed { i, f ->
            if (i < index) f else f.copy(lng = f.lng + (i - index + 1) * metersPerFix * 0.0000111)
        }

    /** **يضبط الدقّةَ للجميع.** */
    fun withAccuracy(fixes: List<NavFix>, m: Float): List<NavFix> =
        fixes.map { it.copy(accuracyM = m) }

    /** **يضبط السرعةَ للجميع** — والاتّجاهُ كما هو. */
    fun withSpeed(fixes: List<NavFix>, mps: Float): List<NavFix> =
        fixes.map { it.copy(speedMps = mps) }

    // ══════════════════════════════════════════════════════════════════
    // **قياسُ الانحراف الحقيقيّ — لا «نجح» وحدَها**
    // ══════════════════════════════════════════════════════════════════
    //
    // (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «لا يكفيني أن الاختبار PASS».)

    data class Timing(
        /** **رقمُ أوّل قراءةٍ شُكّ فيها** — وسالبٌ يعني لم يقع. */
        val firstSuspect: Int,
        /** **ورقمُ التي أكّدت.** */
        val confirmed: Int,
        /** **وكم ثانيةً من أوّل قراءةٍ منحرفةٍ إلى التأكيد.** */
        val secondsToOffRoute: Double,
        /** **وكم متراً بعد نقطة الانحراف.** */
        val metersAfterDeviation: Double,
        /** **وتدرّجُ النقاط.** */
        val scores: List<Double>,
    )

    /**
     * **يقيس متى شكَّ ومتى تأكّد** — بالنسبة إلى قراءة الانحراف.
     *
     * `deviationAt` رقمُ أوّل قراءةٍ تركت المسار.
     */
    fun measure(out: List<NavState>, fixes: List<NavFix>, deviationAt: Int): Timing {
        val st = states(out)
        val firstSuspect = st.indexOfFirst { it == OffRouteDetector.State.SUSPECTED_OFF_ROUTE }
        val confirmed = st.indexOfFirst { it == OffRouteDetector.State.OFF_ROUTE }
        val sec = if (confirmed < 0 || deviationAt >= fixes.size) -1.0
        else (fixes[confirmed].atMs - fixes[deviationAt].atMs) / 1000.0
        val meters = if (confirmed < 0) -1.0 else {
            var m = 0.0
            for (i in (deviationAt + 1)..confirmed) {
                m += GpsQuality.metersBetween(fixes[i - 1], fixes[i])
            }
            m
        }
        return Timing(firstSuspect, confirmed, sec, meters, out.map { it.offRoute.score })
    }

    /** **كم مرّةً شُكَّ زوراً على مسارٍ سليم.** */
    fun falseSuspects(out: List<NavState>): Int =
        states(out).count { it == OffRouteDetector.State.SUSPECTED_OFF_ROUTE }

    /** **وكم مرّةً أُعلن خروجٌ زوراً.** */
    fun falseOffRoutes(out: List<NavState>): Int =
        states(out).count { it == OffRouteDetector.State.OFF_ROUTE }

    /** **وكم قراءةً رُفضت.** */
    fun rejected(out: List<NavState>): Int = out.count { it.grade == FixGrade.REJECTED }

    /** **وكم مرّةً انتقلت الحالُ فعلاً** — لقياس التذبذب. */
    fun transitions(out: List<NavState>): Int {
        val st = states(out)
        var n = 0
        for (i in 1 until st.size) if (st[i] != st[i - 1]) n++
        return n
    }

    // ══════════════════════════════════════════════════════════════════
    // **الرفائدُ الأربعَ عشرةَ**
    // ══════════════════════════════════════════════════════════════════

    /** ١ · **سيرٌ سليمٌ على المسار** — لا شكَّ ولا خروج. */
    fun stayOnRoute(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.straight()
        return r to RouteFixtures.driveAlong(r, speedMps = 10f)
    }

    /**
     * ٢ · **قفزةٌ واحدةٌ ثمّ عودة.**
     *
     * **ومئةُ مترٍ في ثانيةٍ تتجاوز `MAX_IMPLIED_MPS`** — فتُرفض
     * بـ`TELEPORT` **ولا تبلغ الكاشفَ أصلاً.**
     */
    fun singleGpsJump(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.straight()
        val base = RouteFixtures.driveAlong(r, speedMps = 10f)
        val jumped = base.mapIndexed { i, f ->
            if (i == 5) f.copy(lng = f.lng + 100.0 * 0.0000111) else f
        }
        return r to jumped
    }

    /**
     * ٣ · **دقّةٌ رديئةٌ وبعدٌ متوسّط** — العتبةُ تتّسع فلا شكّ.
     *
     * **وهي حالُ المالك بعينها**: `دقّة 55م · بعد 35م`.
     */
    fun accuracyDrift(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.straight()
        val base = RouteFixtures.driveAlong(r, speedMps = 10f)
        return r to withAccuracy(shiftEast(base, 35.0), 55f)
    }

    /**
     * ٤ · **شارعٌ موازٍ بعشرين متراً — وباتّجاهٍ معاكس.**
     *
     * **والمسافةُ وحدَها لا تكشفه**: عشرون متراً دون العتبة.
     * **والاتّجاهُ يكشفه** — وهو المطلوب في البند ١٤.
     */
    fun parallelRoadOpposite(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.straight()
        // **يسير جنوباً على خطٍّ يوازي المسارَ الذاهبَ شمالاً.**
        val fixes = (0..11).map { i ->
            NavFix(
                lat = RouteFixtures.north(300.0 - i * 10.0),
                lng = RouteFixtures.east(20.0),
                accuracyM = 6f,
                speedMps = 10f,
                bearingDeg = 180f,
                atMs = 1_000L + i * 1_000L,
            )
        }
        return r to fixes
    }

    /**
     * ٥ · **انحرافٌ حقيقيٌّ مستمرّ** — يترك المسارَ عند القراءة ٥.
     *
     * **وعشرةُ أمتارٍ لكلّ قراءةٍ** ≈ زاويةُ خروجٍ واقعيّةٌ عند
     * ٣٦ كم/س.
     */
    fun realDeviation(): Triple<NavRoute, List<NavFix>, Int> {
        val r = RouteFixtures.straight()
        val base = RouteFixtures.driveAlong(r, speedMps = 10f)
        return Triple(r, divergeFrom(base, 5, 10.0), 5)
    }

    /** ٦ · **انحرافٌ عند الانعطاف** — سماحُ المناورة يمنع الإعلان. */
    fun deviationNearTurn(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.singleRight()
        val base = RouteFixtures.driveAlong(r, speedMps = 10f)
        // **القراءاتُ حولَ المناورة عند ٣٠٠م** — أي نحو الثلاثين.
        val at = 30
        val nudged = base.mapIndexed { i, f ->
            if (abs(i - at) <= 2) f.copy(lng = f.lng + 34.0 * 0.0000111) else f
        }
        return r to nudged
    }

    /** ٧ · **وعند الدوّار** — والسماحُ أوسع. */
    fun deviationNearRoundabout(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.roundabout()
        val base = RouteFixtures.driveAlong(r, speedMps = 8f)
        val at = 25
        val nudged = base.mapIndexed { i, f ->
            if (abs(i - at) <= 2) f.copy(lat = f.lat + 40.0 * 0.000009) else f
        }
        return r to nudged
    }

    /** ٨ · **شكٌّ ثمّ عودةٌ قبل التأكيد** — يُمحى بلا خروج. */
    fun suspectThenReturn(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.straight()
        val base = RouteFixtures.driveAlong(r, speedMps = 10f)
        val out = base.mapIndexed { i, f ->
            if (i in 6..7) f.copy(lng = f.lng + 45.0 * 0.0000111) else f
        }
        return r to out
    }

    /**
     * ٩ · **واقفٌ وقراءتُه تنجرف ثلاثين متراً.**
     *
     * **وسرعتُه دون المتر** — فالدليلُ يُجمَّد. (البند ٢٠.)
     */
    fun stationaryDrift(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.straight()
        val at = RouteFixtures.pointAt(r, 200.0)
        val fixes = (0..11).map { i ->
            val side = if (i % 2 == 0) 1 else -1
            NavFix(
                lat = at.lat,
                lng = at.lng + side * 38.0 * 0.0000111,
                accuracyM = 8f,
                speedMps = 0.2f,
                bearingDeg = null,
                atMs = 1_000L + i * 1_000L,
            )
        }
        return r to fixes
    }

    /** ١٠ · **انحرافٌ بسرعةٍ عالية** — يُصدَّق أسرع. */
    fun highSpeedDeviation(): Triple<NavRoute, List<NavFix>, Int> {
        val r = RouteFixtures.long(vertices = 40)
        val base = RouteFixtures.driveAlong(r, speedMps = 25f)
        return Triple(r, divergeFrom(base, 4, 30.0), 4)
    }

    /** ١١ · **متدهورةٌ وحدَها** — تُسهم ولا تحسم بالسرعة نفسِها. */
    fun degradedOnly(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.straight()
        val base = RouteFixtures.driveAlong(r, speedMps = 10f)
        // **دقّةُ ٤٠م ⇒ متدهورة** (فوق ٢٥ ودون ٦٠)، **وبعدُ ١٢٠م
        // يتجاوز العتبةَ ٧٠م.**
        return r to withAccuracy(shiftEast(base, 120.0), 40f)
    }

    /** ١٢ · **مرفوضةٌ بين مشبوهتين** — لا تزيد ولا تنقص ولا تُعيد. */
    fun rejectedBetweenSuspicious(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.straight()
        val base = RouteFixtures.driveAlong(r, speedMps = 10f)
        val out = base.mapIndexed { i, f ->
            when {
                // **دقّةٌ فوق الستّين ⇒ مرفوضة.**
                i == 7 -> f.copy(lng = f.lng + 200.0 * 0.0000111, accuracyM = 90f)
                i >= 4 -> f.copy(lng = f.lng + 70.0 * 0.0000111)
                else -> f
            }
        }
        return r to out
    }

    /**
     * ١٣ · **لفُّ الزاوية عند الشمال.**
     *
     * **مسارٌ يسير شمالاً واتّجاهُ السائق `359°`** — والفرقُ درجةٌ
     * واحدةٌ لا ٣٥٩. **ومن طرح مباشرةً أعلن خروجاً على مستقيم.**
     */
    fun bearingWrapAround(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.straight()
        val base = RouteFixtures.driveAlong(r, speedMps = 10f)
        return r to base.mapIndexed { i, f ->
            f.copy(bearingDeg = if (i % 2 == 0) 359.5f else 0.5f)
        }
    }

    /**
     * ١٥ · **مجرًى منحرفٌ بدقّةٍ تُختار لكلّ قراءة.**
     *
     * (معايرةُ ٣أ ٢٠٢٦-٠٨-٢٠ — لاختبار قاعدة `DEGRADED`.)
     *
     * **٤٠م متدهورةٌ، و٦م مقبولة.**
     */
    fun deviationWithAccuracy(
        from: Int = 3,
        metersPerFix: Double = 20.0,
        count: Int = 20,
        accuracyAt: (Int) -> Float,
    ): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.straight()
        val base = RouteFixtures.driveAlong(r, speedMps = 10f).take(count)
        val diverged = divergeFrom(base, from, metersPerFix)
        return r to diverged.mapIndexed { i, f -> f.copy(accuracyM = accuracyAt(i)) }
    }

    /** ١٤ · **مسارٌ يمرّ قربَ نفسِه** — ولا يقفز التقدّمُ فيُعلَن خروج. */
    fun routeNearItself(): Pair<NavRoute, List<NavFix>> {
        val r = RouteFixtures.nearItself()
        return r to RouteFixtures.driveAlong(r, speedMps = 9f)
    }
}
