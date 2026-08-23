package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **مُحلِّلُ الطريق الموازي — البنود ٣١ و٣٤ إلى ٣٦**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٨ب، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 */
class ParallelResolverTest {

    private fun rig(enabled: Boolean = true) =
        ParallelResolver(enabled = enabled)

    private fun fix(t: Long, acc: Float = 5f) =
        NavFix(35.95, 39.01, acc, 11f, 0f, t * 1000L)

    private fun hit(lateral: Double, progress: Double = 100.0) =
        RouteProgress.State(
            progressM = progress, remainingM = 500.0, remainingSec = 60.0,
            offRouteM = kotlin.math.abs(lateral), lateralSignedM = lateral,
            fraction = 0.2, current = null, next = null,
            distanceToManeuverM = 300.0, showNextTogether = false,
            nearDestination = false, arrived = false,
        )

    private fun healthy() = NavState(
        grade = FixGrade.ACCEPTED, reason = RejectReason.NONE,
        lat = 35.95, lng = 39.01, bearingDeg = 0f, animationMs = 0L,
        hasRoute = true, progress = null,
        offRoute = OffRouteDetector.Verdict(
            OffRouteDetector.State.ON_ROUTE, 0.0, 36.0, 0.0,
            OffRouteDetector.Skip.NONE, false, false, false, false,
        ),
    )

    /** **يقود قراءاتٍ حتّى ينطق المُحلِّلُ بطلب.** */
    private fun feed(
        r: ParallelResolver, n: Int, lateral: Double,
        acc: Float = 5f, from: Long = 0, routeId: String? = "R1",
        nav: NavState? = null,
    ): CorrelationRequest? {
        var req: CorrelationRequest? = null
        for (i in 0 until n) {
            val q = r.onFix(fix(from + i, acc), FixGrade.ACCEPTED,
                hit(lateral), nav ?: healthy(), routeId, 1L)
            if (q != null && req == null) req = q
        }
        return req
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ٣١ — الرايةُ مطفأة**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `مطفأً فلا طلبَ ولا حال`() {
        val r = rig(enabled = false)
        assertNull(feed(r, 40, 30.0))
        assertEquals(ParallelResolver.State.NO_SUSPICION, r.state)
        assertFalse(r.confirmedParallel)
    }

    @Test
    fun `والرايةُ مطفأةٌ افتراضاً في الإنتاج`() {
        assertFalse("رايةُ الإنتاج مشتعلة", NavFeatures.parallelResolver)
        assertFalse(ParallelResolver().enabled)
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ٣٤ — الاشتباه**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `إزاحةٌ دون خمسةَ عشرَ فلا اشتباه`() {
        assertNull(feed(rig(), 20, 12.0))
    }

    @Test
    fun `وخمسةَ عشرَ لأقلَّ من ثمانِ ثوانٍ فلا طلب`() {
        assertNull(feed(rig(), 7, 20.0))
    }

    @Test
    fun `وثمانٍ بدقّةٍ مقبولةٍ فطلبٌ واحد`() {
        val req = feed(rig(), 8, 20.0)
        assertNotNull("لم يُطلَب", req)
        assertEquals("R1", req!!.routeId)
        assertEquals("الدليلُ هو نافذةُ الاشتباه نفسُها", 8, req.fixes.size)
    }

    @Test
    fun `ودقّةٌ سيّئةٌ لا تُنتج طلباً`() {
        // **بلا هذه البوّابة**: ١٠٤ نوبةً في الساعة على المسار الصحيح.
        assertNull(feed(rig(), 20, 30.0, acc = 25f))
    }

    @Test
    fun `والمتدهورةُ والمرفوضةُ لا دليلَ فيهما`() {
        val r = rig()
        for (i in 0 until 20) {
            r.onFix(fix(i.toLong()), FixGrade.DEGRADED, hit(30.0), healthy(), "R1", 1L)
        }
        assertEquals(ParallelResolver.State.NO_SUSPICION, r.state)
        for (i in 0 until 20) {
            r.onFix(fix(i.toLong()), FixGrade.REJECTED, hit(30.0), healthy(), "R1", 1L)
        }
        assertEquals(ParallelResolver.State.NO_SUSPICION, r.state)
    }

    @Test
    fun `ولا هويّةَ فنقصٌ لا سؤال`() {
        // **البند ١٩** — ولا ارتدادَ إلى مسارٍ مضى.
        val r = rig()
        assertNull(feed(r, 12, 30.0, routeId = null))
        assertEquals(ParallelResolver.State.COOLDOWN, r.state)
    }

    // ══════════════════════════════════════════════════════════════
    // **البندان ٢٧ و٢٩ — الأولويّة**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `وآخرُ ميلٍ لا يُستجوَب`() {
        val nav = healthy().copy(arrivalPhase = ArrivalPhase.LAST_MILE_TO_TARGET)
        assertNull(feed(rig(), 20, 30.0, nav = nav))
    }

    @Test
    fun `وإعادةُ الحساب توقفه`() {
        val nav = healthy().copy(reroute = RerouteStatus.REROUTING)
        assertNull(feed(rig(), 20, 30.0, nav = nav))
    }

    @Test
    fun `والخروجُ المؤكَّدُ حسمَ الأمر`() {
        val nav = healthy().copy(
            offRoute = OffRouteDetector.Verdict(
                OffRouteDetector.State.OFF_ROUTE, 8.0, 36.0, 90.0,
                OffRouteDetector.Skip.NONE, true, true, false, false,
            ),
        )
        assertNull(feed(rig(), 20, 30.0, nav = nav))
    }

    @Test
    fun `والاتّجاهُ المعاكسُ كذلك`() {
        val nav = healthy().copy(
            wrongWay = WrongWayDetector.Verdict(
                WrongWayDetector.State.WRONG_WAY, 175f, 9000L, 90.0, true,
                WrongWayDetector.Skip.NONE, 1L,
            ),
        )
        assertNull(feed(rig(), 20, 30.0, nav = nav))
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ٣٥ — التأكيد**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `ردٌّ موازٍ ثمّ دليلٌ طازجٌ ثمّ اتّفاقٌ فتأكيد`() {
        val r = rig()
        assertNotNull(feed(r, 8, 20.0))
        assertEquals(ParallelResolver.State.RESOLVING_FIRST, r.state)

        r.onResult(RoadCorrelation.PARALLEL_ROUTE, hit(20.0), fix(8))
        assertEquals(ParallelResolver.State.WAITING_FRESH_FIX, r.state)
        assertFalse(r.confirmedParallel)

        // ── قراءةٌ واحدةٌ طازجة — البند ١٢ ────────────────────────
        val second = r.onFix(fix(9), FixGrade.ACCEPTED, hit(20.0), healthy(), "R1", 1L)
        assertNotNull("لم يُطلَب الثاني", second)
        assertEquals(ParallelResolver.State.RESOLVING_SECOND, r.state)

        r.onResult(RoadCorrelation.PARALLEL_ROUTE, hit(20.0), fix(9))
        assertTrue(r.confirmedParallel)
        assertEquals(ParallelResolver.State.CONFIRMED_PARALLEL, r.state)
    }

    @Test
    fun `واختلافُ الثاني التباس`() {
        val r = rig()
        feed(r, 8, 20.0)
        r.onResult(RoadCorrelation.PARALLEL_ROUTE, hit(20.0), fix(8))
        r.onFix(fix(9), FixGrade.ACCEPTED, hit(20.0), healthy(), "R1", 1L)
        r.onResult(RoadCorrelation.ON_PLANNED_ROUTE, hit(20.0), fix(9))
        assertFalse(r.confirmedParallel)
        assertEquals(ParallelResolver.State.COOLDOWN, r.state)
    }

    @Test
    fun `والتباسُ الأوّلِ لا يُكمل`() {
        val r = rig()
        feed(r, 8, 20.0)
        r.onResult(RoadCorrelation.AMBIGUOUS, hit(20.0), fix(8))
        assertFalse(r.confirmedParallel)
        assertEquals(ParallelResolver.State.COOLDOWN, r.state)
    }

    @Test
    fun `وسقوطُ الشبكة نقصٌ لا حكم`() {
        val r = rig()
        feed(r, 8, 20.0)
        r.onResult(RoadCorrelation.INSUFFICIENT_DATA, hit(20.0), fix(8))
        assertFalse(r.confirmedParallel)
    }

    @Test
    fun `وعلى المسار المخطَّط فلا شيء`() {
        val r = rig()
        feed(r, 8, 20.0)
        r.onResult(RoadCorrelation.ON_PLANNED_ROUTE, hit(20.0), fix(8))
        assertFalse(r.confirmedParallel)
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ٣٦ — منعُ العاصفة**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `ثلاثون دقيقةً في ممرٍّ ملتبسٍ ولا استجوابَ دوريّ`() {
        /**
         * **أمرُ المالك (البند ٢٢) نصّاً**: «driver remains in
         * physically ambiguous corridor for 10 minutes. لا أريد:
         * 2 Match calls every 30s».
         *
         * **والتبريدُ وحدَه لا يمنعها** — يلزم دليلٌ جديدٌ معتبَر.
         */
        val r = rig()
        var calls = 0
        var episodes = 0
        // **ثمانمئةُ ثانيةٍ ونيّف — إزاحةٌ ثابتةٌ وتقدّمٌ زاحفٌ بطيء.**
        for (t in 0 until 1800) {
            val q = r.onFix(
                fix(t.toLong()), FixGrade.ACCEPTED,
                // **تقدّمٌ بطيءٌ جدّاً** — أقلُّ من عتبة الدليل الجديد.
                hit(20.0, progress = 100.0 + t * 0.05),
                healthy(), "R1", 1L,
            )
            if (q != null) {
                calls++
                if (r.state == ParallelResolver.State.RESOLVING_FIRST) episodes++
                // **والخادمُ يلتبس دائماً.**
                r.onResult(RoadCorrelation.AMBIGUOUS, hit(20.0), fix(t.toLong()))
            }
        }
        println("  ثلاثون دقيقة: نوبات=$episodes · نداءات=$calls")
        assertTrue("استجوابٌ دوريّ: $calls نداءً في نصف ساعة", calls <= 4)
        assertTrue("نوباتٌ بلا سقف: $episodes", episodes <= 4)
    }

    @Test
    fun `والسقفُ الصلبُ يحرس`() {
        assertEquals(4, ParallelResolver.MAX_EPISODES)
    }

    @Test
    fun `ومسارٌ جديدٌ يفكّ كلَّ شيء`() {
        val r = rig()
        feed(r, 8, 20.0)
        r.onResult(RoadCorrelation.AMBIGUOUS, hit(20.0), fix(8))
        assertEquals(ParallelResolver.State.COOLDOWN, r.state)

        // **جيلٌ جديد** — تركيبُ مسارٍ آخر.
        r.onFix(fix(100), FixGrade.ACCEPTED, hit(0.0), healthy(), "R3", 2L)
        assertEquals(ParallelResolver.State.NO_SUSPICION, r.state)
        assertEquals(0, r.episodes)
    }

    // ══════════════════════════════════════════════════════════════
    // **العتباتُ مُعايَرة**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `والعتباتُ هي المقيسة`() {
        assertEquals(15.0, ParallelSuspicion.MIN_OFFSET_M, 0.0)
        assertEquals(8, ParallelSuspicion.SUSTAIN_SEC)
        assertEquals(10f, ParallelSuspicion.MAX_ACCURACY_M)
        assertEquals(30_000L, ParallelResolver.COOLDOWN_MS)
    }
}
