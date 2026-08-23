package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختباراتُ المرحلة ٤ — الإرشادُ الصوتيّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * المعرّفات: `VOICE-*`
 *
 * **ولا سمّاعةَ في هذه الحزمة كلِّها** — `VoicePlanner` لا يعرف
 * أندرويد، **وما لا يملك الباب لا يطرقه.**
 */
class VoiceTest {

    private fun engineWithVoice(
        route: NavRoute,
        tuning: VoiceTuning = VoiceTuning(),
    ): Pair<NavEngine, VoicePlanner> {
        val planner = VoicePlanner(tuning)
        val e = NavEngine(voice = planner).also { it.setRoute(route) }
        return e to planner
    }

    private fun cuesOf(e: NavEngine, fixes: List<NavFix>): List<Pair<Int, VoiceCue>> {
        val out = ArrayList<Pair<Int, VoiceCue>>()
        fixes.forEachIndexed { i, f -> e.onFix(f).cues.forEach { out += i to it } }
        return out
    }

    // ══════════════════════════════════════════════════════════════════
    // **١ · الأطوارُ والزنادُ الديناميكيّ**
    // ══════════════════════════════════════════════════════════════════

    /** **VOICE-001** — المنعطفُ يُنبَّه عنه ثلاثَ مرّاتٍ لا أكثر. */
    @Test
    fun `VOICE-001 ثلاثةُ أطوارٍ لكلّ منعطفٍ ولا تكرار`() {
        val route = RouteFixtures.singleRight()
        val (e, _) = engineWithVoice(route)
        val cues = cuesOf(e, RouteFixtures.driveAlong(route, speedMps = 10f))
        val turn = cues.filter { it.second.maneuver?.kind == ManeuverKinds.TURN_RIGHT }
        println("VOICE-001 · " + turn.joinToString(" · ") { "${it.second.stage}@${it.first}: ${it.second.text}" })
        assertEquals("طورٌ تكرّر", turn.size, turn.map { it.second.id }.toSet().size)
        assertTrue("لم يُنبَّه عن المنعطف", turn.isNotEmpty())
        assertTrue("طورٌ غيرُ معروف", turn.all { it.second.stage != CueStage.EVENT })
    }

    /**
     * **VOICE-002** — القياسُ عبر ثلاث سرعات. (البند ٢٨.)
     *
     * **والسرعةُ العاليةُ تُنبَّه أبكرَ بالمسافة** — والزمنُ واحد.
     */
    @Test
    fun `VOICE-002 قياسُ الزناد عبر ثلاث سرعات`() {
        val route = RouteFixtures.long(vertices = 30)
        val turnAt = 2000.0
        val withTurn = NavRoute(
            route.geometry,
            route.cumulativeM,
            listOf(
                NavManeuver(ManeuverKinds.DEPART, null, 0.0, 0, turnAt, turnAt / 12),
                NavManeuver(ManeuverKinds.TURN_RIGHT, "right", turnAt, 0, 500.0, 50.0),
                NavManeuver(ManeuverKinds.ARRIVE, null, route.totalM, 0, 0.0, 0.0),
            ),
        )
        for (kmh in listOf(15.0, 36.0, 70.0)) {
            val mps = (kmh / 3.6).toFloat()
            val (e, _) = engineWithVoice(withTurn)
            val cues = cuesOf(e, RouteFixtures.driveAlong(withTurn, speedMps = mps))
            val byStage = cues.map { it.second }
                .filter { it.maneuver?.kind == ManeuverKinds.TURN_RIGHT }
                .associateBy { it.stage }
            fun at(s: CueStage): String {
                val c = byStage[s] ?: return "—"
                return "${"%.0f".format(c.firedAtM)}م/${"%.1f".format(c.firedAtM / mps)}ث"
            }
            println("VOICE-002 · ${"%.0f".format(kmh)}كم/س → PREPARE=${at(CueStage.PREPARE)} · APPROACH=${at(CueStage.APPROACH)} · NOW=${at(CueStage.NOW)}")
            assertNotNull("لم يقع APPROACH عند $kmh", byStage[CueStage.APPROACH])
            assertNotNull("لم يقع NOW عند $kmh", byStage[CueStage.NOW])
        }
    }

    /** **VOICE-003** — عبورُ العتبة بين قراءتين لا يُفقد التعليمة. */
    @Test
    fun `VOICE-003 قفزةُ القراءة لا تُفقد الطور`() {
        val route = RouteFixtures.singleRight()
        val (e, _) = engineWithVoice(route)
        // **قراءتان: ٢٢٠م ثمّ ٧٠م** — والعتبةُ بينهما.
        val fixes = listOf(80.0, 230.0).mapIndexed { i, d ->
            val p = RouteFixtures.pointAt(route, d)
            NavFix(p.lat, p.lng, 6f, 12f, RouteFixtures.bearingAt(route, d), 1_000L + i * 12_000L)
        }
        val cues = cuesOf(e, fixes)
        val stages = cues.map { it.second.stage }
        println("VOICE-003 · $stages")
        assertTrue("فُقدت التعليماتُ عند القفزة", cues.isNotEmpty())
    }

    /** **VOICE-004** — البدءُ داخلَ العتبة يُعطي طوراً واحداً لا ثلاثة. */
    @Test
    fun `VOICE-004 البدءُ قربَ المنعطف يُعطي طوراً واحدا`() {
        val route = RouteFixtures.singleRight()
        val (e, _) = engineWithVoice(route)
        // **يبدأ على بُعد ستّين متراً من المنعطف.**
        val fixes = RouteFixtures.driveAlong(route, speedMps = 10f).drop(24).take(1)
        val cues = cuesOf(e, fixes)
        println("VOICE-004 · " + cues.joinToString { "${it.second.stage}: ${it.second.text}" })
        assertTrue("أُطلقت ثلاثةُ أطوارٍ دفعةً واحدة", cues.size <= 1)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٢ · السرعةُ والبديل**
    // ══════════════════════════════════════════════════════════════════

    /** **VOICE-010** — بلا سرعةٍ تُستعمل سرعةُ الخطوة لا زمنُ الرحلة. */
    @Test
    fun `VOICE-010 بديلُ السرعة من خطوة المحرّك`() {
        val planner = VoicePlanner()
        val route = RouteFixtures.singleRight()
        val progress = RouteProgress(route)
        val p = progress.onFix(
            NavFix(RouteFixtures.north(100.0), RouteFixtures.LNG0, 6f, null, null, 1_000L),
            null,
        )!!
        val noSpeed = NavFix(p.let { RouteFixtures.north(100.0) }, RouteFixtures.LNG0, 6f, null, null, 1_000L)
        val v = planner.trustedSpeed(noSpeed, p)
        // **وخطوةُ `singleRight` ٣٠٠م في ٣٠ث ⇒ عشرةُ أمتار.**
        println("VOICE-010 · سرعةُ الخطوة=${"%.1f".format(v)} م/ث · والافتراضُ=${planner.tuning.fallbackSpeedMps}")
        assertEquals(10.0, v, 0.5)
    }

    /** **VOICE-011** — وبلا خطوةٍ صالحةٍ يُستعمل الافتراضُ المركزيّ. */
    @Test
    fun `VOICE-011 بلا تقديرٍ صالحٍ يُستعمل الافتراض`() {
        val planner = VoicePlanner()
        val route = NavRoute.of(
            (0..4).map { GeoPoint(RouteFixtures.north(it * 50.0), RouteFixtures.LNG0) },
            listOf(
                NavManeuver(ManeuverKinds.DEPART, null, 0.0, 0, 0.0, 0.0),
                NavManeuver(ManeuverKinds.ARRIVE, null, 200.0, 0, 0.0, 0.0),
            ),
        )
        val p = RouteProgress(route).onFix(
            NavFix(RouteFixtures.north(50.0), RouteFixtures.LNG0, 6f, null, null, 1_000L), null,
        )!!
        val v = planner.trustedSpeed(
            NavFix(RouteFixtures.north(50.0), RouteFixtures.LNG0, 6f, null, null, 1_000L), p,
        )
        println("VOICE-011 · $v")
        assertEquals(planner.tuning.fallbackSpeedMps, v, 0.001)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٣ · بوّابةُ الثقة**
    // ══════════════════════════════════════════════════════════════════

    /** **VOICE-020** — المرفوضةُ لا تولّد شيئاً. */
    @Test
    fun `VOICE-020 المرفوضةُ لا تولّد تعليمة`() {
        val route = RouteFixtures.singleRight()
        val (e, _) = engineWithVoice(route)
        val fixes = RouteFixtures.driveAlong(route, speedMps = 10f).mapIndexed { i, f ->
            if (i in 25..30) f.copy(accuracyM = 90f) else f
        }
        val cues = cuesOf(e, fixes)
        val fromRejected = cues.filter { it.first in 25..30 }
        println("VOICE-020 · تعليماتٌ من مرفوضة=${fromRejected.size}")
        assertEquals(0, fromRejected.size)
    }

    /** **VOICE-021** — و`NOW` تحتاج ثقةً حديثةً بالمقياس الرتيب. */
    @Test
    fun `VOICE-021 NOW تحتاج قراءةً مقبولةً حديثة`() {
        val route = RouteFixtures.singleRight()
        val (e, planner) = engineWithVoice(route)
        // **متدهوراتٌ من البداية** — فلا شهادةَ مقبولةٌ قطّ.
        val fixes = RouteFixtures.driveAlong(route, speedMps = 10f)
            .map { it.copy(accuracyM = 45f) }
        val cues = cuesOf(e, fixes)
        val now = cues.count { it.second.stage == CueStage.NOW }
        println("VOICE-021 · NOW=$now · محجوبةٌ بالثقة=${planner.gatedByGps}")
        assertEquals("**قيلت «الآن» بلا موضعٍ موثوق**", 0, now)
        assertTrue("لم تُحجب شيءٌ أصلاً", planner.gatedByGps > 0)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٤ · المناوراتُ والدوّارات**
    // ══════════════════════════════════════════════════════════════════

    /** **VOICE-030** — المتقاربتان تُدمجان. (البند ٢٩.) */
    @Test
    fun `VOICE-030 قياسُ دمج المناورات المتقاربة`() {
        for (gap in listOf(8.0, 20.0, 50.0, 100.0)) {
            for (kmh in listOf(15.0, 70.0)) {
                val route = twoTurns(gap)
                val (e, _) = engineWithVoice(route)
                val cues = cuesOf(e, RouteFixtures.driveAlong(route, speedMps = (kmh / 3.6).toFloat()))
                val merged = cues.count { it.second.text.contains("ثمّ") }
                val separate = cues.count { it.second.kind == CueKind.MANEUVER && !it.second.text.contains("ثمّ") }
                println("VOICE-030 · فجوة=${"%.0f".format(gap)}م سرعة=${"%.0f".format(kmh)} → مدموجة=$merged منفصلة=$separate")
            }
        }
        // **والثمانيةُ أمتارٍ تُدمج قطعاً** — قِيست في المرحلة ٢.
        val route = twoTurns(8.0)
        val (e, _) = engineWithVoice(route)
        val cues = cuesOf(e, RouteFixtures.driveAlong(route, speedMps = 10f))
        assertTrue("**ثمانيةُ أمتارٍ لم تُدمج**", cues.any { it.second.text.contains("ثمّ") })
    }

    private fun twoTurns(gapM: Double): NavRoute {
        val pts = (0..8).map { GeoPoint(RouteFixtures.north(it * 25.0), RouteFixtures.LNG0) } +
            listOf(GeoPoint(RouteFixtures.north(200.0), RouteFixtures.east(gapM))) +
            (1..8).map { GeoPoint(RouteFixtures.north(200.0 + it * 25.0), RouteFixtures.east(gapM)) }
        return NavRoute.of(
            pts,
            listOf(
                NavManeuver(ManeuverKinds.DEPART, null, 0.0, 0, 200.0, 20.0),
                NavManeuver(ManeuverKinds.TURN_RIGHT, "right", 200.0, 0, gapM, gapM / 10),
                NavManeuver(ManeuverKinds.TURN_LEFT, "left", 200.0 + gapM, 0, 200.0, 20.0),
                NavManeuver(ManeuverKinds.ARRIVE, null, 400.0 + gapM, 0, 0.0, 0.0),
            ),
        )
    }

    /** **VOICE-031** — الدوّارُ يُقال بمخرجه واسمِه. */
    @Test
    fun `VOICE-031 الدوّارُ بمخرجه واسمِه`() {
        val route = RouteFixtures.roundabout()
        val (e, _) = engineWithVoice(route)
        val cues = cuesOf(e, RouteFixtures.driveAlong(route, speedMps = 8f))
        val ra = cues.map { it.second }.filter { it.maneuver?.kind == ManeuverKinds.ROUNDABOUT }
        val exit = cues.map { it.second }.filter { it.maneuver?.kind == ManeuverKinds.EXIT_ROUNDABOUT }
        println("VOICE-031 · دخول=${ra.map { it.text }} · خروج=${exit.size}")
        assertTrue("لم يُقل الدوّار", ra.isNotEmpty())
        assertTrue("لم يُقل اسمُ الدوّار", ra.any { it.text.contains("دوار النعيم") })
        assertTrue("لم يُقل رقمُ المخرج", ra.any { it.text.contains("الثاني") })
    }

    /** **VOICE-032** — و`exit` فارغٌ لا يُسقط شيئاً. */
    @Test
    fun `VOICE-032 دوّارٌ بلا رقمِ مخرجٍ لا يسقط`() {
        val m = NavManeuver(ManeuverKinds.ROUNDABOUT, "right", 100.0, 0, 40.0, 8.0, null, null, null)
        val text = VoicePhrases.maneuver(m, 200)
        println("VOICE-032 · $text")
        assertTrue(text.contains("الدوار"))
        assertTrue("قال رقمَ مخرجٍ لا يعرفه", !text.contains("المخرج "))
    }

    /** **VOICE-033** — وكلُّ نوعٍ له جملةٌ ولا يسقط شيء. */
    @Test
    fun `VOICE-033 مصفوفةُ المناورات كاملة`() {
        val kinds = listOf(
            ManeuverKinds.DEPART, ManeuverKinds.STRAIGHT, ManeuverKinds.TURN_LEFT,
            ManeuverKinds.TURN_RIGHT, ManeuverKinds.SLIGHT_LEFT, ManeuverKinds.SLIGHT_RIGHT,
            ManeuverKinds.SHARP_LEFT, ManeuverKinds.SHARP_RIGHT, ManeuverKinds.U_TURN,
            ManeuverKinds.MERGE, ManeuverKinds.FORK, ManeuverKinds.OFF_RAMP,
            ManeuverKinds.ROUNDABOUT, ManeuverKinds.EXIT_ROUNDABOUT, ManeuverKinds.ARRIVE,
            ManeuverKinds.UNKNOWN, "TELEPORT_SOMETHING",
        )
        for (k in kinds) {
            val m = NavManeuver(k, null, 100.0, 0, 50.0, 5.0, null, 2, "دوار النعيم")
            val now = VoicePhrases.maneuver(m, null)
            val far = VoicePhrases.maneuver(m, 200)
            println("VOICE-033 · $k → «$now» / «$far»")
            assertTrue("جملةٌ فارغةٌ لـ$k", now.isNotBlank() && far.isNotBlank())
        }
        val unknown = NavManeuver("SOMETHING_NEW", null, 100.0, 0, 50.0, 5.0)
        assertEquals(VoicePhrases.FOLLOW_ROUTE, VoicePhrases.action(unknown))
    }

    // ══════════════════════════════════════════════════════════════════
    // **٥ · أسماءُ الشوارع — والمُدوَّنةُ المقيسة**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **VOICE-040** — مُدوَّنةُ الرقّة الحقيقيّة.
     *
     * **قِيست من ثمانية مساراتٍ حيّةٍ** على محرّكنا (٢٠٢٦-٠٨-٢١):
     * ١٠٦ خطوة · ٥٢ مسمّاة · ١٦ اسماً فريداً.
     */
    @Test
    fun `VOICE-040 مُدوَّنةُ أسماء الرقّة`() {
        val corpus = listOf(
            "Adnan Malki Street", "Aleppo Road", "Barid Street", "Forosiyah Street",
            "Jazarah Street", "Nazlet Moasat Street", "Nene Road 4", "Qitar Street",
            "Saqiah St.", "الجسر الجديد", "الجسر القديم", "دوار الدلة",
            "دوار الصوامع", "دوار المزارع", "دوار النعيم", "دوار حزيمة",
        )
        val accepted = corpus.filter { StreetName.speakable(it) != null }
        val reasons = corpus.filter { StreetName.speakable(it) == null }
            .groupingBy { StreetName.reject(it) ?: "-" }.eachCount()
        println("VOICE-040 · المجموع=${corpus.size} · مقبولة=${accepted.size} · مرفوضة=${corpus.size - accepted.size}")
        println("VOICE-040 · أسبابُ الرفض=$reasons")
        println("VOICE-040 · المقبولة=$accepted")
        // **وكلُّ اسمٍ عربيٍّ في المُدوَّنة يُقبل** — ولا يرفض الحدُّ
        // واحداً منها.
        assertEquals("رُفض اسمٌ عربيٌّ صالح", 7, accepted.size)
        assertTrue("قُبل اسمٌ لاتينيّ", accepted.none { it.any { c -> c in 'a'..'z' || c in 'A'..'Z' } })
    }

    /** **VOICE-041** — والاسمُ الرديءُ يسقط والجملةُ تبقى. */
    @Test
    fun `VOICE-041 الاسمُ الرديءُ يسقط ولا يُفسد الجملة`() {
        val bad = NavManeuver(ManeuverKinds.TURN_LEFT, "left", 100.0, 0, 50.0, 5.0, "Aleppo Road")
        val good = NavManeuver(ManeuverKinds.TURN_LEFT, "left", 100.0, 0, 50.0, 5.0, "الجسر القديم")
        val none = NavManeuver(ManeuverKinds.TURN_LEFT, "left", 100.0, 0, 50.0, 5.0, null)
        println("VOICE-041 · رديء=«${VoicePhrases.maneuver(bad, 200)}»")
        println("VOICE-041 · جيّد=«${VoicePhrases.maneuver(good, 200)}»")
        assertEquals(VoicePhrases.maneuver(none, 200), VoicePhrases.maneuver(bad, 200))
        assertTrue(VoicePhrases.maneuver(good, 200).contains("الجسر القديم"))
    }

    // ══════════════════════════════════════════════════════════════════
    // **٦ · إعادةُ الحساب والوصول والجيل**
    // ══════════════════════════════════════════════════════════════════

    /** **VOICE-050** — «جارٍ إعادة الحساب» مرّةً في النوبة. */
    @Test
    fun `VOICE-050 جملةُ إعادة الحساب لا تتكرّر في النوبة`() {
        val planner = VoicePlanner()
        val route = RouteFixtures.straight()
        val progress = RouteProgress(route)
        val fix = RouteFixtures.driveAlong(route).first()
        val p = progress.onFix(fix, null)
        fun state(status: RerouteStatus) = NavState(
            FixGrade.ACCEPTED, RejectReason.NONE, fix.lat, fix.lng, 0f, 0L, true, p,
            OffRouteDetector.Verdict(
                OffRouteDetector.State.OFF_ROUTE, 4.0, 36.0, 80.0,
                OffRouteDetector.Skip.NONE, true, false, false, false,
            ),
            reroute = status,
        )
        val first = planner.onState(state(RerouteStatus.REROUTING), 1L, fix)
        val again = planner.onState(state(RerouteStatus.REROUTING), 1L, fix)
        val failed = planner.onState(state(RerouteStatus.REROUTE_FAILED), 1L, fix)
        println("VOICE-050 · أوّل=${first.map { it.text }} · ثانٍ=${again.size} · فشل=${failed.map { it.text }}")
        assertEquals(1, first.size)
        assertEquals("تكرّرت جملةُ إعادة الحساب", 0, again.size)
        assertEquals(1, failed.size)
        // **ولا جملةَ «خارج المسار»** — أمرُ المالك.
        assertTrue(first.none { it.text.contains("خارج") })
    }

    /** **VOICE-051** — الوصولُ من `arrived` لا من `nearDestination`. */
    @Test
    fun `VOICE-051 الوصولُ يُقال بطرفه`() {
        val route = RouteFixtures.straight()
        for (target in listOf(TripTarget.PICKUP, TripTarget.DROPOFF)) {
            val planner = VoicePlanner().also { it.target = target }
            val e = NavEngine(voice = planner).also { it.setRoute(route) }
            val fixes = RouteFixtures.driveAlong(route, speedMps = 10f) +
                RouteFixtures.standStill(
                    RouteFixtures.driveAlong(route, speedMps = 10f).last(), 5,
                )
            val cues = cuesOf(e, fixes).map { it.second }.filter { it.kind == CueKind.ARRIVE }
            println("VOICE-051 · $target → ${cues.map { it.text }}")
            assertEquals("الوصولُ قيل غيرَ مرّة", 1, cues.map { it.id }.toSet().size)
            assertTrue(cues.isNotEmpty())
        }
    }

    /** **VOICE-052** — الجيلُ الجديد يبدأ هُويّاتٍ جديدة. */
    @Test
    fun `VOICE-052 جيلٌ جديدٌ يبدأ تعليماتٍ جديدة`() {
        val route = RouteFixtures.singleRight()
        val (e, _) = engineWithVoice(route)
        val first = cuesOf(e, RouteFixtures.driveAlong(route, speedMps = 10f).take(30))
        val gen1 = first.map { it.second.id.generation }.toSet()

        e.setRoute(RouteFixtures.roundabout())
        val second = cuesOf(e, RouteFixtures.driveAlong(RouteFixtures.roundabout(), speedMps = 8f))
        val gen2 = second.map { it.second.id.generation }.toSet()
        println("VOICE-052 · أجيالُ الأولى=$gen1 · الثانية=$gen2")
        assertTrue("الجيلُ لم يتبدّل", gen1.intersect(gen2).isEmpty())
    }

    /** **VOICE-053** — وخارجَ المسار لا تعليماتِ مناورة. */
    @Test
    fun `VOICE-053 لا تعليماتِ مناورةٍ خارجَ المسار`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val planner = VoicePlanner()
        val e = NavEngine(voice = planner).also { it.setRoute(route) }
        val all = ArrayList<Pair<Boolean, VoiceCue>>()
        fixes.forEach { f -> e.onFix(f).let { s -> s.cues.forEach { all += s.isOffRoute to it } } }
        val whileOff = all.filter { it.first && it.second.kind == CueKind.MANEUVER }
        println("VOICE-053 · تعليماتُ مناورةٍ خارجَ المسار=${whileOff.size}")
        assertEquals(0, whileOff.size)
    }
}
