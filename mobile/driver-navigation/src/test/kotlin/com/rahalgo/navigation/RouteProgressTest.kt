package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import kotlin.math.abs

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مصفوفةُ اختبار المرحلة ٢ — إعادةُ تشغيلٍ بلا جهاز**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «المرحلة ٢ يجب أن تكون Replay-first».)
 *
 * **وكلُّ ما هنا حسابٌ محضٌ** — ولا شبكةَ ولا أندرويد.
 */
class RouteProgressTest {

    private fun run(route: NavRoute, fixes: List<NavFix>): List<RouteProgress.State> {
        val p = RouteProgress(route)
        val out = ArrayList<RouteProgress.State>()
        val bearing = BearingTracker()
        var prev: NavFix? = null
        for (f in fixes) {
            val h = bearing.update(f, prev, FixGrade.ACCEPTED)
            prev = f
            p.onFix(f, h)?.let { out.add(it) }
        }
        return out
    }

    // ══════════════════════════════════════════════════════════════════
    // **الإسقاطُ والتقدّم**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **والنقطةُ على المسار تُسقَط عليه.**
     *
     * **وأوّلُ إسقاطٍ يمسح المسارَ كلَّه** — كُتب هذا الاختبارُ أوّلاً
     * بتقدّمٍ صفرٍ ونقطةٍ على بُعد ١٢٠ متراً في ثانية، **فحبستها
     * النافذةُ عند ستّين** — وكان الأداةُ محقّاً والاختبارُ مخطئاً.
     * **فكشف فجوةً حقيقيّة**: من فتح التطبيقَ في منتصف الطريق كان
     * يبقى سهمُه عند بدايته.
     */
    @Test
    fun `النقطةُ على المسار تُسقَط عليه بلا انحراف`() {
        val r = RouteFixtures.straight()
        val hit = RouteProjector.project(
            r, RouteFixtures.north(120.0), RouteFixtures.east(0.0), 0.0, 1.0, 10f, 0f,
            fullScan = true,
        )!!
        assertTrue("انحرافٌ كبير: ${hit.offRouteM}", hit.offRouteM < 2.0)
        assertEquals(120.0, hit.progressM, 6.0)
    }

    /** **ومن فتح التطبيقَ في منتصف الطريق يُعرف موضعُه.** */
    @Test
    fun `أوّلُ إسقاطٍ يجد السائقَ في منتصف المسار`() {
        val r = RouteFixtures.straight()
        val mid = RouteFixtures.pointAt(r, 300.0)
        val p = RouteProgress(r)
        val st = p.onFix(NavFix(mid.lat, mid.lng, 6f, 10f, 0f, 1_000L), 0f)
        assertNotNull(st)
        assertEquals("**حُبس عند البداية**", 300.0, st!!.progressM, 15.0)
    }

    @Test
    fun `التقدّمُ يتصاعد على مسارٍ مستقيم`() {
        val r = RouteFixtures.straight()
        val states = run(r, RouteFixtures.driveAlong(r))
        assertTrue(states.size > 10)
        for (i in 1 until states.size) {
            assertTrue(
                "**تراجع** عند $i: ${states[i - 1].progressM} → ${states[i].progressM}",
                states[i].progressM >= states[i - 1].progressM - 0.001,
            )
        }
        assertTrue("لم يبلغ النهاية", states.last().fraction > 0.95)
    }

    /** **والمتبقّي ينقص أبداً على السير المستقيم.** */
    @Test
    fun `المسافةُ المتبقّيةُ تنقص`() {
        val r = RouteFixtures.singleRight()
        val states = run(r, RouteFixtures.driveAlong(r))
        for (i in 1 until states.size) {
            assertTrue(
                "المتبقّي زاد عند $i",
                states[i].remainingM <= states[i - 1].remainingM + 0.001,
            )
        }
        assertTrue("لم يقترب من الصفر: ${states.last().remainingM}", states.last().remainingM < 20)
    }

    // ══════════════════════════════════════════════════════════════════
    // **النافذةُ تمنع القفز**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **ومسارٌ يمرّ قربَ نفسِه** — الذهابُ والعودةُ بينهما ٢٠ متراً.
     *
     * **وبلا نافذةٍ يقفز التقدّمُ من ٢٥٪ إلى ٧٠٪** والسائقُ لم يتحرّك.
     */
    @Test
    fun `المسارُ القريبُ من نفسِه لا يجعل التقدّمَ يقفز`() {
        val r = RouteFixtures.nearItself()
        val states = run(r, RouteFixtures.driveAlong(r, speedMps = 8f))
        var maxJump = 0.0
        for (i in 1 until states.size) {
            maxJump = maxOf(maxJump, states[i].progressM - states[i - 1].progressM)
        }
        // **وأقصى ما يمكن قطعُه في ثانيةٍ بثمانيةٍ في الثانية.**
        assertTrue("**قفزةُ $maxJump متراً في ثانية**", maxJump < 40.0)
        assertTrue("لم يُكمل: ${states.last().fraction}", states.last().fraction > 0.9)
    }

    /** **وقفزةٌ إلى مقطعٍ لاحقٍ خارج النافذة لا تُقرأ.** */
    @Test
    fun `النافذةُ لا تنظر إلى ما بعدها`() {
        val r = RouteFixtures.long(200)
        // **آخرُ تقدّمٍ عند المئة متر، والقراءةُ عند ٥٠٠٠م.**
        val far = RouteFixtures.pointAt(r, 5_000.0)
        val hit = RouteProjector.project(r, far.lat, far.lng, 100.0, 1.0, 10f, 0f)
        assertNotNull(hit)
        assertTrue("**قفز إلى ${hit!!.progressM}**", hit.progressM < 300.0)
    }

    // ══════════════════════════════════════════════════════════════════
    // **الرجفةُ والوقوف**
    // ══════════════════════════════════════════════════════════════════

    /** **ورجفةٌ جانبيّةٌ لا تُذبذب التقدّم.** */
    @Test
    fun `رجفةُ GPS لا تجعل التقدّمَ يتذبذب`() {
        val r = RouteFixtures.straight()
        val states = run(r, RouteFixtures.jitter(RouteFixtures.driveAlong(r), 8.0))
        var back = 0
        for (i in 1 until states.size) {
            if (states[i].progressM < states[i - 1].progressM - 0.001) back++
        }
        assertEquals("**تراجع $back مرّةً على رجفةٍ جانبيّة**", 0, back)
    }

    /** **والوقوفُ لا يُحرّك التقدّم.** */
    @Test
    fun `الوقوفُ يُبقي التقدّمَ والمناورةَ كما هما`() {
        val r = RouteFixtures.singleRight()
        val drive = RouteFixtures.driveAlong(r).take(20)
        val last = drive.last()
        val states = run(r, drive + RouteFixtures.standStill(last, 10))
        val tail = states.takeLast(10)
        for (i in 1 until tail.size) {
            assertEquals(
                "**تحرّك وهو واقف**",
                tail[0].progressM, tail[i].progressM, 3.0,
            )
            assertEquals(tail[0].current?.kind, tail[i].current?.kind)
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **المناورات**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `المناورةُ تتبدّل عند تجاوزها`() {
        val r = RouteFixtures.singleRight()
        val states = run(r, RouteFixtures.driveAlong(r))
        val kinds = states.map { it.current?.kind }.distinct()
        assertTrue("لم تتبدّل المناورةُ: $kinds", kinds.size >= 2)
        assertEquals(ManeuverKinds.DEPART, states.first().current?.kind)
    }

    /**
     * **ومناورتان بينهما ثمانيةُ أمتارٍ لا تُبتلع إحداهما.**
     *
     * **وقِيس هذا فعلاً** — وسماحٌ ثابتٌ بخمسةَ عشرَ متراً يبتلعها.
     */
    @Test
    fun `مناورتان متقاربتان تُرَيان كلتاهما`() {
        val r = RouteFixtures.closeManeuvers()
        val states = run(r, RouteFixtures.driveAlong(r, speedMps = 3f))
        val seen = states.mapNotNull { it.current?.kind }.distinct()
        assertTrue("**ابتُلعت مناورة**: $seen", seen.contains(ManeuverKinds.TURN_RIGHT))
        assertTrue("**ابتُلعت مناورة**: $seen", seen.contains(ManeuverKinds.TURN_LEFT))
    }

    /** **والعرضُ المزدوجُ شأنُ عرضٍ لا حكم.** */
    @Test
    fun `المتقاربتان تُعرضان معاً`() {
        val r = RouteFixtures.closeManeuvers()
        val states = run(r, RouteFixtures.driveAlong(r, speedMps = 3f))
        assertTrue(
            "**لم تُطلب عرضاً مزدوجاً قطّ**",
            states.any { it.showNextTogether },
        )
    }

    /** **والمتباعدتان لا تُعرضان معاً.** */
    @Test
    fun `المتباعدتان لا تُعرضان معا`() {
        val r = RouteFixtures.singleRight()
        val states = run(r, RouteFixtures.driveAlong(r))
        assertFalse(
            "**عرضٌ مزدوجٌ لمناورتين بينهما ٣٠٠م**",
            states.first().showNextTogether,
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **الدوّار**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `الدوّارُ يحمل رقمَ المخرج واسمَه`() {
        val r = RouteFixtures.roundabout()
        val states = run(r, RouteFixtures.driveAlong(r, speedMps = 5f))
        val rb = states.mapNotNull { it.current }.firstOrNull { it.kind == ManeuverKinds.ROUNDABOUT }
        assertNotNull("**لم تُبلَغ مناورةُ الدوّار**", rb)
        assertEquals(2, rb!!.roundaboutExit)
        assertEquals("دوار النعيم", rb.roundaboutName)
        assertTrue(rb.isRoundabout)
    }

    // ══════════════════════════════════════════════════════════════════
    // **الأسماءُ الغائبةُ والأنواعُ المجهولة**
    // ══════════════════════════════════════════════════════════════════

    /** **ونوعٌ لا نعرفه لا يُسقط شيئاً.** */
    @Test
    fun `النوعُ المجهولُ لا يكسر ولا يُعدّ معروفا`() {
        val r = RouteFixtures.unnamedAndUnknown()
        val states = run(r, RouteFixtures.driveAlong(r, speedMps = 5f))
        assertTrue(states.isNotEmpty())
        val unknown = r.maneuvers.first { !ManeuverKinds.isKnown(it.kind) }
        assertFalse(ManeuverKinds.isKnown(unknown.kind))
        // **ولا اسمَ يُخترع.**
        assertTrue(r.maneuvers.all { it.streetName == null })
    }

    // ══════════════════════════════════════════════════════════════════
    // **زمنُ الوصول**
    // ══════════════════════════════════════════════════════════════════

    /** **ومن وقف عند إشارةٍ لا يرتفع زمنُه** — الطريقُ لم يتبدّل. */
    @Test
    fun `الوقوفُ لا يرفع زمنَ الوصول`() {
        val r = RouteFixtures.singleRight()
        val drive = RouteFixtures.driveAlong(r).take(15)
        val states = run(r, drive + RouteFixtures.standStill(drive.last(), 20))
        val atStop = states[14].remainingSec
        val afterStop = states.last().remainingSec
        assertEquals("**ارتفع الزمنُ وهو واقف**", atStop, afterStop, 3.0)
    }

    @Test
    fun `زمنُ الوصول ينقص مع التقدّم`() {
        val r = RouteFixtures.singleRight()
        val states = run(r, RouteFixtures.driveAlong(r))
        for (i in 1 until states.size) {
            assertTrue(
                "الزمنُ زاد عند $i",
                states[i].remainingSec <= states[i - 1].remainingSec + 0.001,
            )
        }
        assertTrue("لم يقترب من الصفر: ${states.last().remainingSec}", states.last().remainingSec < 5)
    }

    /** **والخطواتُ المكتملةُ لا تدخل.** */
    @Test
    fun `زمنُ الوصول عند البداية يساوي مدّةَ الخطوات كلِّها`() {
        val r = RouteFixtures.singleRight()
        val p = RouteProgress(r)
        val total = r.maneuvers.sumOf { it.stepDurationS }
        assertEquals(total, p.remainingSeconds(), 0.001)
    }

    // ══════════════════════════════════════════════════════════════════
    // **البدايةُ والنهايةُ والوصول**
    // ══════════════════════════════════════════════════════════════════

    /** **ومن كان قبل بداية المسار يُسقَط على أوّله لا يُرفض.** */
    @Test
    fun `ما قبل البداية يُسقَط على أوّل المسار`() {
        val r = RouteFixtures.straight()
        val hit = RouteProjector.project(
            r, RouteFixtures.north(-40.0), RouteFixtures.east(0.0), 0.0, 1.0, 5f, 0f,
        )
        assertNotNull(hit)
        assertEquals(0.0, hit!!.progressM, 1.0)
    }

    @Test
    fun `ما بعد النهاية لا يتجاوز طولَ المسار`() {
        val r = RouteFixtures.straight()
        val states = run(r, RouteFixtures.driveAlong(r))
        assertTrue(states.all { it.progressM <= r.totalM + 0.001 })
        assertTrue(states.all { it.remainingM >= 0.0 })
    }

    /** **ولا وصولَ بقراءةٍ واحدة** — ودقّةُ أربعين متراً تكذب. */
    @Test
    fun `الوصولُ يحتاج تكراراً لا قراءةً واحدة`() {
        val r = RouteFixtures.straight()
        val end = RouteFixtures.pointAt(r, r.totalM)
        val p = RouteProgress(r)
        // **قراءةٌ واحدةٌ عند النهاية بدقّةٍ سيّئة.**
        val one = p.onFix(NavFix(end.lat, end.lng, 40f, 8f, 0f, 1_000L), 0f)
        assertNotNull(one)
        assertFalse("**أعلن الوصولَ بقراءةٍ واحدة**", one!!.arrived)
    }

    @Test
    fun `الوقوفُ عند النهاية يُعلن الوصول`() {
        val r = RouteFixtures.straight()
        val states = run(
            r,
            RouteFixtures.driveAlong(r) +
                RouteFixtures.standStill(
                    RouteFixtures.pointAt(r, r.totalM).let {
                        NavFix(it.lat, it.lng, 6f, 0.1f, 0f, 900_000L)
                    },
                    5,
                ),
        )
        assertTrue("**لم يُعلن الوصول**", states.last().arrived)
        assertTrue(states.last().nearDestination)
    }

    // ══════════════════════════════════════════════════════════════════
    // **مسارٌ لا يصلح**
    // ══════════════════════════════════════════════════════════════════

    /** **ومسارٌ بلا مناوراتٍ يُرسم ولا يُرشِد** — ولا يسقط. */
    @Test
    fun `مسارٌ بلا مناوراتٍ لا يُنتج حالاً ولا يرمي`() {
        val bare = NavRoute.of(
            listOf(GeoPoint(35.95, 39.00), GeoPoint(35.96, 39.01)),
            emptyList(),
        )
        assertFalse(bare.usable)
        val p = RouteProgress(bare)
        assertNull(p.onFix(NavFix(35.955, 39.005, 6f, 10f, 0f, 1_000L), 0f))
    }

    @Test
    fun `مسارٌ برأسٍ واحدٍ لا يصلح`() {
        val one = NavRoute.of(listOf(GeoPoint(35.95, 39.00)), emptyList())
        assertFalse(one.usable)
    }

    // ══════════════════════════════════════════════════════════════════
    // **الأداءُ والبناء**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **ومسارُ الرقّة–دمشق ٢١١٣ رأساً** (قِيس).
     *
     * **والنافذةُ تعني أنّ التكلفةَ لا تتبع طولَ المسار.**
     */
    @Test
    fun `المسارُ الطويلُ لا يُبطئ الإسقاط`() {
        val long = RouteFixtures.long(2113)
        val short = RouteFixtures.straight()

        fun timeOf(r: NavRoute, at: Double): Long {
            val p = RouteFixtures.pointAt(r, at)
            val t0 = System.nanoTime()
            repeat(2_000) {
                RouteProjector.project(r, p.lat, p.lng, at, 1.0, 15f, 0f)
            }
            return System.nanoTime() - t0
        }
        val tLong = timeOf(long, 100_000.0)
        val tShort = timeOf(short, 200.0)
        // **ولا تُقاس نسبةٌ دقيقةٌ على جهازِ بناءٍ مزدحم** — يكفي ألّا
        // يتضاعف بمراتب.
        assertTrue(
            "**الطويلُ أبطأُ بمراتب**: طويل=${tLong / 1_000_000}ms قصير=${tShort / 1_000_000}ms",
            tLong < tShort * 12 + 60_000_000,
        )
    }

    /** **والتراكميّةُ تصاعديّةٌ أبدا.** */
    @Test
    fun `التراكميّةُ تصاعديّةٌ ومطابقةٌ للهندسة`() {
        val r = RouteFixtures.singleRight()
        assertEquals(r.geometry.size, r.cumulativeM.size)
        assertEquals(0.0, r.cumulativeM[0], 1e-9)
        for (i in 1 until r.cumulativeM.size) {
            assertTrue(r.cumulativeM[i] >= r.cumulativeM[i - 1])
        }
        assertTrue("الطولُ ${r.totalM}", abs(r.totalM - 600.0) < 15.0)
    }

    /**
     * **ولا شبكةَ في الحلقة الساخنة.**
     *
     * **يُقرأ المصدرُ لا السلوك** — من أضاف نداءً يوماً يسقط بناؤه.
     */
    @Test
    fun `منطقُ المسار لا يعرف شبكةً`() {
        val files = listOf("RouteProjector.kt", "RouteProgress.kt", "NavRoute.kt")
        val banned = listOf("http", "Backend", "DriverApi", "HttpClient", "Socket", "URL", "match")
        val bad = mutableListOf<String>()
        for (n in files) {
            val f = java.io.File("src/main/kotlin/com/rahalgo/navigation/$n")
            assertTrue("لا مصدر: ${f.absolutePath}", f.exists())
            val code = f.readText()
                .replace(Regex("(?s)/\\*.*?\\*/"), " ")
                .lines().joinToString("\n") { it.substringBefore("//") }
            for (w in banned) if (code.contains(w)) bad += "$n: $w"
        }
        assertTrue("**منطقُ المسار يطرق بابَ الشبكة**: $bad", bad.isEmpty())
    }
}
