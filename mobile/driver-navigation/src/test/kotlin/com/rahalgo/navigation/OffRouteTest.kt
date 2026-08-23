package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختباراتُ المرحلة ٣أ — «خرج عن المسار» بثقةٍ لا بظنّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * المعرّفات: `OFFR-*`
 *
 * **ولا نداءَ شبكةٍ في هذه الحزمة كلِّها** — `NavEngine` لا يملك باباً
 * إليها، **وما لا يملك الباب لا يطرقه.**
 */
class OffRouteTest {

    private val ON = OffRouteDetector.State.ON_ROUTE
    private val SUS = OffRouteDetector.State.SUSPECTED_OFF_ROUTE
    private val OFF = OffRouteDetector.State.OFF_ROUTE

    // ══════════════════════════════════════════════════════════════════
    // **١ · الوصلةُ نفسُها — قبل أيّ حكم**
    // ══════════════════════════════════════════════════════════════════

    /** **OFFR-001** — المسارُ يصل المحرّكَ ويعمل التقدّمُ فعلاً. */
    @Test
    fun `OFFR-001 المسارُ يصل المحرّكَ والتقدّمُ يعمل`() {
        val (route, fixes) = OffRouteFixtures.stayOnRoute()
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, fixes)

        assertNotNull("المسارُ لم يصل", e.currentRoute)
        val last = out.last()
        assertTrue("التقدّمُ لم يُحسب", last.progress != null)
        assertTrue("التقدّمُ لم يتحرّك", last.progress!!.progressM > 400.0)
        assertTrue("ما بقي لم ينقص", last.remainingM < 100.0)
        assertTrue("الزمنُ المتبقّي لم يُحسب", last.remainingSec >= 0.0)
        assertNotNull("المناورةُ الجاريةُ فارغة", last.currentManeuver)
    }

    /** **OFFR-002** — بلا مسارٍ لا يسقط شيءٌ ولا يُعلَن خروج. */
    @Test
    fun `OFFR-002 بلا مسارٍ لا خروجَ ولا سقوط`() {
        val e = NavEngine()
        e.setRoute(null)
        val out = OffRouteFixtures.replay(e, RouteFixtures.driveAlong(RouteFixtures.straight()))
        assertTrue(out.all { it.offRoute.skip == OffRouteDetector.Skip.NO_ROUTE })
        assertTrue(out.all { it.offRoute.state == ON })
        assertNull(out.last().progress)
        assertTrue("الموضعُ لم يُرسم", out.last().lat != null)
    }

    /** **OFFR-003** — مسارٌ بلا مناوراتٍ غيرُ صالحٍ فيُهمَل بلا سقوط. */
    @Test
    fun `OFFR-003 مسارٌ غيرُ صالحٍ يُهمَل`() {
        val e = NavEngine()
        e.setRoute(NavRoute.of(listOf(GeoPoint(35.95, 39.0)), emptyList()))
        assertNull(e.currentRoute)
        val out = OffRouteFixtures.replay(e, RouteFixtures.driveAlong(RouteFixtures.straight()))
        assertTrue(out.all { it.offRoute.state == ON })
    }

    /** **OFFR-004** — `setRoute` يُنسي التقدّمَ والشكَّ معاً. */
    @Test
    fun `OFFR-004 مسارٌ جديدٌ يمحو التقدّمَ والشكّ`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val e = OffRouteFixtures.engineOn(route)
        OffRouteFixtures.replay(e, fixes)
        assertTrue("لم يُعلَن خروجٌ أصلاً", e.detector.state == OFF)

        // **مسارٌ آخرُ يبدأ من منتصفه** — وهو حالُ إعادة الحساب.
        val fresh = RouteFixtures.singleRight()
        e.setRoute(fresh)
        assertEquals(ON, e.detector.state)
        assertEquals(0.0, e.detector.score, 0.001)

        val mid = RouteFixtures.driveAlong(fresh).drop(30)
        val out = OffRouteFixtures.replay(e, mid)
        // **وأوّلُ إسقاطٍ يمسح المسارَ كلَّه** — فلا يبقى السهمُ عند
        // البداية. (آليّةُ المرحلة ٢، `fullScan`.)
        assertTrue(
            "التقدّمُ بدأ من الصفر لا من الوسط: ${out.first().progress?.progressM}",
            (out.first().progress?.progressM ?: 0.0) > 200.0,
        )
        assertTrue("المناورةُ لم تُقرأ", out.last().currentManeuver != null)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٢ · لا إنذارَ كاذب**
    // ══════════════════════════════════════════════════════════════════

    /** **OFFR-010** — مسارٌ سليمٌ: صفرُ شكٍّ وصفرُ خروج. */
    @Test
    fun `OFFR-010 مسارٌ سليمٌ بلا إنذارٍ كاذب`() {
        val (route, fixes) = OffRouteFixtures.stayOnRoute()
        val out = OffRouteFixtures.replay(OffRouteFixtures.engineOn(route), fixes)
        val suspects = OffRouteFixtures.falseSuspects(out)
        val offs = OffRouteFixtures.falseOffRoutes(out)
        println("OFFR-010 · شكٌّ كاذب=$suspects · خروجٌ كاذب=$offs · قراءات=${fixes.size}")
        assertEquals("شكٌّ كاذبٌ على مسارٍ سليم", 0, suspects)
        assertEquals("خروجٌ كاذب", 0, offs)
    }

    /** **OFFR-011** — لفُّ الزاوية `359↔0` لا يُعلن خروجاً. */
    @Test
    fun `OFFR-011 لفُّ الزاوية عند الشمال سليم`() {
        val (route, fixes) = OffRouteFixtures.bearingWrapAround()
        val out = OffRouteFixtures.replay(OffRouteFixtures.engineOn(route), fixes)
        println("OFFR-011 · خروج=${OffRouteFixtures.falseOffRoutes(out)}")
        assertEquals(0, OffRouteFixtures.falseOffRoutes(out))
        assertEquals(0, OffRouteFixtures.falseSuspects(out))
    }

    /** **OFFR-012** — مسارٌ يمرّ قربَ نفسِه لا يُربك الكاشف. */
    @Test
    fun `OFFR-012 مسارٌ يمرّ قربَ نفسِه`() {
        val (route, fixes) = OffRouteFixtures.routeNearItself()
        val out = OffRouteFixtures.replay(OffRouteFixtures.engineOn(route), fixes)
        println("OFFR-012 · شكّ=${OffRouteFixtures.falseSuspects(out)} خروج=${OffRouteFixtures.falseOffRoutes(out)}")
        assertEquals("خروجٌ كاذبٌ على مسارٍ ملتفّ", 0, OffRouteFixtures.falseOffRoutes(out))
    }

    /** **OFFR-013** — دقّةٌ رديئةٌ وبعدٌ متوسّط: العتبةُ تتّسع. */
    @Test
    fun `OFFR-013 دقّةٌ رديئةٌ لا تُعلن خروجا`() {
        val (route, fixes) = OffRouteFixtures.accuracyDrift()
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, fixes)
        val v = out.last().offRoute
        println("OFFR-013 · عتبة=${"%.1f".format(v.thresholdM)} بعد=${"%.1f".format(v.offRouteM)} خروج=${OffRouteFixtures.falseOffRoutes(out)}")
        assertEquals(0, OffRouteFixtures.falseOffRoutes(out))
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **OFFR-014 — أسماحُ الدقّةِ الكاملُ متسامحٌ أكثرَ ممّا يلزم؟**
     * ══════════════════════════════════════════════════════════════════
     *
     * (شاغلُ المالك ٢٠٢٦-٠٨-٢٠: «GPS بدقّة 55m قد يجعل Threshold
     *  قريباً من 85m، وهذا قد يخفي انحرافاً حقيقيّاً داخل المدينة…
     *  أعطني False Positive / False Negative results».)
     *
     * **ولا يُغيَّر شيءٌ هنا** — يُقاس ويُعرَض. والقرارُ قرارُ المالك.
     */
    @Test
    fun `OFFR-014 قياسُ سماح الدقّة`() {
        val route = RouteFixtures.straight()
        val base = RouteFixtures.driveAlong(route, speedMps = 10f)

        // **كم متراً يبتعد قبل أن يُعلَن خروجُه** — بدقّةِ ٥٥م.
        fun detectAt(factor: Double, cap: Double): Double {
            val e = OffRouteFixtures.engineOn(
                route,
                OffRouteDetector.Tuning(accuracySlackFactor = factor, maxAccuracySlackM = cap),
            )
            val fixes = OffRouteFixtures.withAccuracy(
                OffRouteFixtures.divergeFrom(base, 5, 10.0), 55f,
            )
            val out = OffRouteFixtures.replay(e, fixes)
            // **والمقياسُ هو الشكُّ لا التأكيد**: بعد قاعدة
            // `DEGRADED` لا تؤكّد قراءةٌ دقّتُها ٥٥م خروجاً أبداً —
            // **فالسؤالُ متى يرتفع الشكّ.**
            val i = OffRouteFixtures.states(out).indexOfFirst { it != ON }
            return if (i < 0) -1.0 else out[i].offRoute.offRouteM
        }

        // **وكم شكّاً كاذباً على مسارٍ سليمٍ بالدقّة نفسِها.**
        fun falseOn(factor: Double, cap: Double): Int {
            val e = OffRouteFixtures.engineOn(
                route,
                OffRouteDetector.Tuning(accuracySlackFactor = factor, maxAccuracySlackM = cap),
            )
            // **وانجرافٌ جانبيٌّ بعشرين متراً** — ما تفعله دقّةُ ٥٥م حقّاً.
            val fixes = OffRouteFixtures.withAccuracy(RouteFixtures.jitter(base, 20.0), 55f)
            return OffRouteFixtures.falseOffRoutes(OffRouteFixtures.replay(e, fixes))
        }

        val full = detectAt(1.0, 60.0)
        val half = detectAt(0.5, 60.0)
        val capped = detectAt(1.0, 25.0)
        println(
            "OFFR-014 · بُعدُ الشكِّ بدقّةِ ٥٥م — " +
                "كاملٌ=${"%.0f".format(full)}م · نصفٌ=${"%.0f".format(half)}م · مسقوفٌ بـ٢٥=${"%.0f".format(capped)}م",
        )
        println(
            "OFFR-014 · خروجٌ كاذبٌ على مسارٍ سليمٍ منجرف — " +
                "كاملٌ=${falseOn(1.0, 60.0)} · نصفٌ=${falseOn(0.5, 60.0)} · مسقوفٌ=${falseOn(1.0, 25.0)}",
        )
        assertTrue("النموذجُ الكاملُ لم يشكّ أصلاً", full > 0)
        assertTrue("التضييقُ لم يُقرّب الكشف", half < full && capped < full)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٣ · جودةُ القراءة**
    // ══════════════════════════════════════════════════════════════════

    /** **OFFR-020** — القفزةُ تُرفض ولا تولّد خروجاً. */
    @Test
    fun `OFFR-020 قفزةُ GPS لا تولّد خروجا`() {
        val (route, fixes) = OffRouteFixtures.singleGpsJump()
        val out = OffRouteFixtures.replay(OffRouteFixtures.engineOn(route), fixes)
        val rej = OffRouteFixtures.rejected(out)
        val offs = OffRouteFixtures.falseOffRoutes(out)
        println("OFFR-020 · مرفوضة=$rej · انتقالاتُ خروج=$offs")
        assertTrue("القفزةُ لم تُرفض أصلاً", rej >= 1)
        assertEquals("قفزةٌ واحدةٌ ولّدت خروجا", 0, offs)
    }

    /** **OFFR-021** — المرفوضةُ لا تزيد النقاطَ ولا تنقصها ولا تُعيد الحال. */
    @Test
    fun `OFFR-021 المرفوضةُ تُتجاهل كأنّها لم تصل`() {
        val (route, fixes) = OffRouteFixtures.rejectedBetweenSuspicious()
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, fixes)
        val i = out.indexOfFirst { it.grade == FixGrade.REJECTED }
        assertTrue("لا قراءةَ مرفوضةٌ في الرفيدة", i > 0)
        assertEquals(
            "المرفوضةُ غيّرت النقاط",
            out[i - 1].offRoute.score, out[i].offRoute.score, 0.0001,
        )
        assertEquals("المرفوضةُ غيّرت الحال", out[i - 1].offRoute.state, out[i].offRoute.state)
        assertEquals(OffRouteDetector.Skip.REJECTED, out[i].offRoute.skip)
    }

    // ══════════════════════════════════════════════════════════════════
    // **والمتدهورةُ تُبقي الشكَّ ولا تحسم — قرارُ المالك ٢٠٢٦-٠٨-٢٠**
    // ══════════════════════════════════════════════════════════════════
    //
    // «**مهما تراكمت قراءاتُ DEGRADED وحدَها، لا تنتقل الحالُ إلى
    //  `OFF_ROUTE`**… لتأكيد `OFF_ROUTE` يجب أن يوجد دليلٌ `ACCEPTED`
    //  موثوقٌ يؤكّد الانحراف.»

    /** **OFFR-DEG-001** — متدهورةٌ وحدَها مهما تراكمت: شكٌّ لا خروج. */
    @Test
    fun `OFFR-DEG-001 المتدهورةُ وحدَها لا تحسم أبدا`() {
        val (route, fixes) = OffRouteFixtures.deviationWithAccuracy(count = 30) { 40f }
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, fixes)
        val st = OffRouteFixtures.states(out)
        println(
            "OFFR-DEG-001 · نقاطٌ نهائيّة=${e.detector.score} · حالٌ نهائيّة=${e.detector.state} · " +
                "خروج=${st.count { it == OFF }}",
        )
        assertTrue(
            "النقاطُ لم تبلغ حدَّ التأكيد — فالاختبارُ لا يفحص القاعدة",
            e.detector.score >= e.detector.tuning.confirmScore,
        )
        assertTrue("المتدهورةُ لم تُبقِ شكّاً", st.any { it == SUS })
        assertEquals("**متدهورةٌ وحدَها أعلنت خروجا**", 0, st.count { it == OFF })
    }

    /** **OFFR-DEG-002** — ومقبولةٌ مريبةٌ بعدها تحسم. */
    @Test
    fun `OFFR-DEG-002 مقبولةٌ مريبةٌ تحسم بعد المتدهورات`() {
        val (route, fixes) = OffRouteFixtures.deviationWithAccuracy(count = 14) {
            if (it == 9) 6f else 40f
        }
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, fixes)
        val at = OffRouteFixtures.states(out).indexOfFirst { it == OFF }
        println("OFFR-DEG-002 · تأكيدٌ عند=$at · وقراءةُ المقبولة=9")
        assertTrue("لم يُحسم رغمَ الشهادة المقبولة", at >= 0)
        assertTrue("حُسم قبل وصول المقبولة", at >= 9)
    }

    /** **OFFR-DEG-003** — وشهادةُ المقبولة تشيخ فلا تحسم إلى الأبد. */
    @Test
    fun `OFFR-DEG-003 شهادةُ المقبولة تشيخ`() {
        // **مقبولةٌ مبكّرةٌ والنقاطُ دون الحدّ**، ثمّ ضبابٌ طويلٌ يبلغ
        // بالنقاط حدَّ التأكيد بعد أن شاخت الشهادة.
        val (route, fixes) = OffRouteFixtures.deviationWithAccuracy(
            from = 3, metersPerFix = 12.0, count = 30,
        ) { if (it == 4) 6f else 40f }
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, fixes)
        println(
            "OFFR-DEG-003 · نقاطٌ نهائيّة=${e.detector.score} · " +
                "شهادةٌ حيّة=${e.detector.hasFreshAcceptedEvidence} · " +
                "خروج=${OffRouteFixtures.falseOffRoutes(out)}",
        )
        assertTrue(
            "النقاطُ لم تبلغ الحدّ — فالاختبارُ لا يفحص الشيخوخة",
            e.detector.score >= e.detector.tuning.confirmScore,
        )
        assertEquals(
            "**شهادةٌ عمرُها عشرون قراءةً أكّدت خروجا**",
            0, OffRouteFixtures.falseOffRoutes(out),
        )
    }

    /** **OFFR-DEG-004** — ومقبولةٌ نظيفةٌ تخفض الشكَّ لا ترفعه. */
    @Test
    fun `OFFR-DEG-004 المقبولةُ النظيفةُ تخفض الشكّ`() {
        val route = RouteFixtures.straight()
        val e = OffRouteFixtures.engineOn(route)
        val base = RouteFixtures.driveAlong(route, speedMps = 10f)

        // **ثلاثُ متدهوراتٍ مريبةٍ ترفع الشكّ.**
        // **وبعيداً عن الانطلاق** — قربَ `DEPART` تتّسع العتبةُ
        // عشرين متراً **فلا يبلغ الشكُّ حدَّه.**
        val murky = (10..12).map {
            base[it].copy(lng = base[it].lng + 130.0 * 0.0000111, accuracyM = 40f)
        }
        OffRouteFixtures.replay(e, murky)
        val before = e.detector.score
        assertTrue("لم يرتفع شكٌّ أصلاً", before > 0)
        assertEquals("المتدهورةُ حسمت", SUS, e.detector.state)

        // **ثمّ مقبولةٌ نظيفةٌ على المسار.**
        val clean = (13..16).map { base[it].copy(accuracyM = 6f) }
        val states = clean.map { e.onFix(it).offRoute.state }
        println(
            "OFFR-DEG-004 · نقاطٌ قبل=$before بعد=${e.detector.score} · " +
                states.joinToString(",") { it.name.take(3) },
        )
        assertTrue("النظيفةُ لم تخفض الشكّ", e.detector.score < before)
        assertEquals("النظيفةُ أدّت إلى خروج", ON, states.last())
        assertEquals(0, states.count { it == OFF })
    }

    // ══════════════════════════════════════════════════════════════════
    // **وشهادةُ المقبولة تُقاس بالزمن لا بعدد القراءات**
    // ══════════════════════════════════════════════════════════════════
    //
    // (تصحيحُ المالك ٢٠٢٦-٠٨-٢٠، بداية ٣ب: «لا أريد ربطَ صحّة القرار
    //  بتردّد GPS المفترَض… القياسُ يكون milliseconds/monotonic time
    //  وليس عددَ قراءات».)

    /** **OFFR-CLK-001** — خمسُ قراءاتٍ في ثانيةٍ ليست خمسَ ثوانٍ. */
    @Test
    fun `OFFR-CLK-001 خمسُ قراءاتٍ في ثانيةٍ لا تُشيخ الشهادة`() {
        val route = RouteFixtures.straight()
        val e = OffRouteFixtures.engineOn(route)
        val base = RouteFixtures.driveAlong(route, speedMps = 10f)

        // **مقبولةٌ مريبةٌ أوّلاً** — تُنشئ الشهادة.
        val at = base[10]
        e.onFix(at.copy(lng = at.lng + 130.0 * 0.0000111, accuracyM = 6f))
        assertTrue("لم تُنشأ شهادة", e.detector.hasFreshAcceptedEvidence)

        // **ثمّ خمسُ متدهوراتٍ في ثانيةٍ واحدة** — ٢٠٠ملّي بينها.
        (1..5).forEach { i ->
            val f = base[10]
            e.onFix(
                f.copy(
                    lng = f.lng + 130.0 * 0.0000111,
                    accuracyM = 40f,
                    atMs = at.atMs + i * 200L,
                ),
            )
        }
        println("OFFR-CLK-001 · شهادةٌ حيّةٌ بعد خمسِ قراءاتٍ في ثانية=${e.detector.hasFreshAcceptedEvidence}")
        assertTrue("**عدَّ القراءاتِ فأشاخ شهادةً عمرُها ثانية**", e.detector.hasFreshAcceptedEvidence)
    }

    /** **OFFR-CLK-002** — وفجوةٌ طويلةٌ بين قراءتين تُشيخها. */
    @Test
    fun `OFFR-CLK-002 فجوةٌ طويلةٌ تُشيخ الشهادة`() {
        val route = RouteFixtures.straight()
        val e = OffRouteFixtures.engineOn(route)
        val base = RouteFixtures.driveAlong(route, speedMps = 10f)
        val at = base[10]
        e.onFix(at.copy(lng = at.lng + 130.0 * 0.0000111, accuracyM = 6f))
        assertTrue(e.detector.hasFreshAcceptedEvidence)

        // **قراءةٌ واحدةٌ بعد عشرين ثانية** — وسرعتُها معقولة.
        val far = base[10]
        e.onFix(
            far.copy(
                lat = far.lat + 60 * 0.000009,
                lng = far.lng + 130.0 * 0.0000111,
                accuracyM = 40f,
                atMs = at.atMs + 20_000L,
            ),
        )
        println("OFFR-CLK-002 · شهادةٌ حيّةٌ بعد عشرين ثانية=${e.detector.hasFreshAcceptedEvidence}")
        assertTrue("**قراءةٌ واحدةٌ بعد عشرين ثانيةً أبقت الشهادة**", !e.detector.hasFreshAcceptedEvidence)
    }

    /** **OFFR-CLK-003** — وساعةُ الجدار لا تُغيّر شيئاً. */
    @Test
    fun `OFFR-CLK-003 قفزةُ ساعة الجدار لا تؤثّر`() {
        val route = RouteFixtures.straight()
        val base = RouteFixtures.driveAlong(route, speedMps = 10f)
        val deviated = OffRouteFixtures.divergeFrom(base, 5, 10.0)

        val plain = OffRouteFixtures.engineOn(route)
        val plainOut = OffRouteFixtures.replay(plain, deviated)

        // **ونسخةٌ ساعةُ جدارها تقفز ساعةً إلى الوراء** — `wallMs`
        // تُسجَّل ولا يُحسَب بها. (انظر `NavFix`.)
        val skewed = OffRouteFixtures.engineOn(route)
        val skewedOut = OffRouteFixtures.replay(
            skewed,
            deviated.mapIndexed { i, f -> f.copy(wallMs = if (i % 2 == 0) 1_000L else 3_600_000L) },
        )
        println(
            "OFFR-CLK-003 · تأكيدٌ بلا قفزة=${OffRouteFixtures.states(plainOut).indexOfFirst { it == OFF }} · " +
                "ومعها=${OffRouteFixtures.states(skewedOut).indexOfFirst { it == OFF }}",
        )
        assertEquals(
            OffRouteFixtures.states(plainOut),
            OffRouteFixtures.states(skewedOut),
        )
    }

    /** **OFFR-CLK-004** — والمرفوضةُ لا تصنع شهادةً جديدة. */
    @Test
    fun `OFFR-CLK-004 المرفوضةُ لا تصنع شهادة`() {
        val route = RouteFixtures.straight()
        val e = OffRouteFixtures.engineOn(route)
        val base = RouteFixtures.driveAlong(route, speedMps = 10f)

        // **قراءةٌ دقّتُها فوق الستّين ⇒ مرفوضة** — ومريبةٌ جدّاً.
        val at = base[10]
        e.onFix(at.copy(lng = at.lng + 300.0 * 0.0000111, accuracyM = 90f))
        println("OFFR-CLK-004 · شهادةٌ من مرفوضة=${e.detector.hasFreshAcceptedEvidence}")
        assertTrue("**المرفوضةُ صنعت شهادة**", !e.detector.hasFreshAcceptedEvidence)
        assertEquals(0.0, e.detector.score, 0.001)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٤ · السرعة**
    // ══════════════════════════════════════════════════════════════════

    /** **OFFR-030** — الواقفُ ينجرف ولا يُعلَن خروجُه. */
    @Test
    fun `OFFR-030 انجرافُ الواقف لا يُعلن خروجا`() {
        val (route, fixes) = OffRouteFixtures.stationaryDrift()
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, fixes)
        println("OFFR-030 · خروج=${OffRouteFixtures.falseOffRoutes(out)} نقاط=${e.detector.score}")
        assertEquals("انجرافُ الوقوف ولّد خروجا", 0, OffRouteFixtures.falseOffRoutes(out))
        assertTrue(out.any { it.offRoute.skip == OffRouteDetector.Skip.STATIONARY })
    }

    /**
     * **OFFR-031** — والوقوفُ يُجمّد ولا يمحو.
     *
     * (أمرُ المالك: «لكن لا تجعل `speed = 0` يمسح suspicion موجودا».)
     */
    @Test
    fun `OFFR-031 الوقوفُ يجمّد الدليلَ ولا يمحوه`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val e = OffRouteFixtures.engineOn(route)
        // **يُسار حتّى يقع الشكّ** ثمّ يقف.
        val walking = fixes.take(10)
        OffRouteFixtures.replay(e, walking)
        val before = e.detector.score
        assertTrue("لم يقع شكٌّ قبل الوقوف", before > 0)

        val last = walking.last()
        val standing = (1..5).map {
            last.copy(speedMps = 0.2f, atMs = last.atMs + it * 1_000L)
        }
        OffRouteFixtures.replay(e, standing)
        println("OFFR-031 · نقاطٌ قبلَ الوقوف=$before وبعده=${e.detector.score}")
        assertEquals("الوقوفُ محا الدليل", before, e.detector.score, 0.0001)
    }

    /** **OFFR-032** — السرعةُ العالية تُسرّع التأكيد. */
    @Test
    fun `OFFR-032 السرعةُ العاليةُ تُسرّع التأكيد`() {
        val (route, fixes, at) = OffRouteFixtures.highSpeedDeviation()
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, fixes)
        val t = OffRouteFixtures.measure(out, fixes, at)
        println("OFFR-032 · شكٌّ عند=${t.firstSuspect} تأكيدٌ عند=${t.confirmed} ثوانٍ=${t.secondsToOffRoute}")
        assertTrue("لم يُعلَن خروجٌ عند سرعةٍ عالية", t.confirmed >= 0)
        assertTrue("تأخّر التأكيدُ أكثرَ من ثلاثِ ثوانٍ", t.secondsToOffRoute <= 3.0)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٥ · الاتّجاهُ شاهدا**
    // ══════════════════════════════════════════════════════════════════

    /**
     * ══════════════════════════════════════════════════════════════════
     * **OFFR-040 — والموازي المعاكسُ لم يعد «خروجا»**
     * ══════════════════════════════════════════════════════════════════
     *
     * (المرحلة ٥، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٤.)
     *
     * **كان يُعلَن خروجاً بشاهد الاتّجاه وحدَه** — والبعدُ عشرون
     * متراً **دون العتبة**، أي أنّه لم يترك الممرَّ أصلاً.
     *
     * **و«خرج عن المسار» دعوى هندسيّة**: من كان عليه يسير بعكسه لم
     * يتركه. **فصار تأكيدُ الخروج يشترط أن يكون البعدُ قد تجاوز
     * العتبةَ مرّةً** — وهذه الحالُ صارت لـ`WrongWayDetector`
     * (`WW-050`).
     *
     * **والشاهدُ باقٍ**: الاتّجاهُ يرفع النقاطَ ويُبقي الشكَّ، **ولا
     * يحسم وحدَه.**
     */
    @Test
    fun `OFFR-040 شارعٌ موازٍ معاكسٌ يُكشف بالاتّجاه`() {
        val (route, fixes) = OffRouteFixtures.parallelRoadOpposite()
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, fixes)
        val maxOff = out.mapNotNull { it.progress?.offRouteM }.maxOrNull() ?: 0.0
        val thr = out.last().offRoute.thresholdM
        val confirmed = OffRouteFixtures.states(out).indexOfFirst { it == OFF }
        val suspected = OffRouteFixtures.states(out).count { it == SUS }
        println(
            "OFFR-040 · أقصى بعد=${"%.1f".format(maxOff)} عتبة=${"%.1f".format(thr)} " +
                "خروجٌ مؤكَّد=$confirmed شكّ=$suspected",
        )
        assertTrue("البعدُ تجاوز العتبةَ — فالاختبارُ لا يفحص الاتّجاه", maxOff <= thr)
        assertEquals("**أُعلن خروجٌ بشاهد الاتّجاه وحدَه**", -1, confirmed)
        assertTrue("الاتّجاهُ لم يعد شاهدا", out.any { it.offRoute.bearingAgainst })
        assertTrue("لم يُبقِ شكّاً", suspected > 0)
    }

    /** **OFFR-041** — والاتّجاهُ وحدَه لا يُعلن خروجاً فوريّاً. */
    @Test
    fun `OFFR-041 الاتّجاهُ شاهدٌ لا حكم`() {
        val (route, fixes) = OffRouteFixtures.parallelRoadOpposite()
        val out = OffRouteFixtures.replay(OffRouteFixtures.engineOn(route), fixes)
        assertTrue(
            "أُعلن الخروجُ من أوّل قراءةٍ — والاتّجاهُ حكمَ وحدَه",
            OffRouteFixtures.states(out).first() != OFF,
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **٦ · سماحُ المناورة والدوّار**
    // ══════════════════════════════════════════════════════════════════

    /** **OFFR-050** — انزياحٌ عند الانعطاف لا يُعلن خروجاً. */
    @Test
    fun `OFFR-050 سماحُ المناورة يمنع خروجاً كاذبا`() {
        val (route, fixes) = OffRouteFixtures.deviationNearTurn()
        val out = OffRouteFixtures.replay(OffRouteFixtures.engineOn(route), fixes)
        val widest = out.map { it.offRoute.thresholdM }.maxOrNull() ?: 0.0
        println("OFFR-050 · أوسعُ عتبة=${"%.1f".format(widest)} خروج=${OffRouteFixtures.falseOffRoutes(out)}")
        assertEquals("انعطافٌ عاديٌّ ولّد خروجا", 0, OffRouteFixtures.falseOffRoutes(out))
        assertTrue("السماحُ لم يتّسع عند المناورة", widest > 36.0)
    }

    /** **OFFR-051** — والدوّارُ أوسع. */
    @Test
    fun `OFFR-051 سماحُ الدوّار أوسع`() {
        val (route, fixes) = OffRouteFixtures.deviationNearRoundabout()
        val out = OffRouteFixtures.replay(OffRouteFixtures.engineOn(route), fixes)
        println("OFFR-051 · خروج=${OffRouteFixtures.falseOffRoutes(out)} أوسعُ عتبة=${"%.1f".format(out.map { it.offRoute.thresholdM }.max())}")
        assertEquals(0, OffRouteFixtures.falseOffRoutes(out))
    }

    /** **OFFR-052** — والسماحُ يزول بمغادرة سياق المناورة. */
    @Test
    fun `OFFR-052 السماحُ يزول بعد المناورة`() {
        val route = RouteFixtures.roundabout()
        val e = OffRouteFixtures.engineOn(route)
        val fixes = RouteFixtures.driveAlong(route, speedMps = 8f)
        val out = OffRouteFixtures.replay(e, fixes)
        val inRoundabout = out.filter {
            it.progress?.current?.isRoundabout == true || it.progress?.next?.isRoundabout == true
        }
        val after = out.filter {
            it.progress != null && it.progress!!.progressM > 300.0
        }
        val widestIn = inRoundabout.map { it.offRoute.thresholdM }.maxOrNull() ?: 0.0
        val widestAfter = after.map { it.offRoute.thresholdM }.maxOrNull() ?: 0.0
        println("OFFR-052 · عتبةٌ في الدوّار=${"%.1f".format(widestIn)} وبعده=${"%.1f".format(widestAfter)}")
        assertTrue("السماحُ لم يتّسع في الدوّار", widestIn > widestAfter)
    }

    /**
     * **OFFR-053** — العتبةُ في أربعة أطوارٍ صريحة.
     *
     * (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «أريد اختباراً صريحاً يزيل أيَّ
     *  التباس… ويجب أن يعود في النهاية إلى Threshold الطبيعيّ فعلاً».)
     */
    @Test
    fun `OFFR-053 العتبةُ تتّسع عند المناورة وتعود بعدها`() {
        val route = RouteFixtures.singleRight()
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, RouteFixtures.driveAlong(route, speedMps = 10f))

        fun at(from: Double, to: Double): Double =
            out.filter { (it.progress?.progressM ?: -1.0) in from..to }
                .map { it.offRoute.thresholdM }.max()

        // **والمناورةُ عند ٣٠٠م، والوصولُ عند ٦٠٠.**
        val farBefore = at(50.0, 200.0)
        val near = at(270.0, 320.0)
        val wellAfter = at(360.0, 520.0)
        val atArrival = at(570.0, 600.0)
        println(
            "OFFR-053 · بعيداً قبلها=${"%.0f".format(farBefore)} · قربَها=${"%.0f".format(near)} · " +
                "بعدها بكثير=${"%.0f".format(wellAfter)} · عند الوصول=${"%.0f".format(atArrival)}",
        )
        assertEquals("العتبةُ بعيداً ليست الطبيعيّة", 36.0, farBefore, 0.5)
        assertTrue("لم تتّسع عند المناورة", near > farBefore)
        assertEquals("**لم تعد إلى الطبيعيّة بعد المناورة**", 36.0, wellAfter, 0.5)
        // **وعند الوصول تتّسع أيضاً** — و`ARRIVE` مناورةٌ كغيرها:
        // **وهذا هو سببُ الـ٥٦م التي رُصدت في تقرير ٣أ بعد الدوّار**،
        // لا سماحُ دوّارٍ بقي.
        assertTrue("وعند الوصول لم تتّسع", atArrival > 36.0)
    }

    /** **OFFR-054** — والدوّارُ: قبلَه وداخلَه وبعده. */
    @Test
    fun `OFFR-054 عتبةُ الدوّار قبلَه وداخلَه وبعده`() {
        val route = RouteFixtures.roundabout()
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, RouteFixtures.driveAlong(route, speedMps = 8f))

        fun at(from: Double, to: Double) =
            out.filter { (it.progress?.progressM ?: -1.0) in from..to }
                .map { it.offRoute.thresholdM }.max()

        // **الدوّارُ عند ٢٠٠ ومخرجُه عند ٢٤٠، والوصولُ عند ٥٠٠.**
        // **وتبدأ من ستّين** — دونها نحن في سياق الانطلاق (`DEPART`
        // عند الصفر)، **فيُقاس سماحُه لا العتبةُ الطبيعيّة.**
        val before = at(60.0, 150.0)
        val inside = at(200.0, 240.0)
        val after = at(300.0, 430.0)
        println(
            "OFFR-054 · قبلَه=${"%.0f".format(before)} · داخلَه=${"%.0f".format(inside)} · " +
                "بعده=${"%.0f".format(after)}",
        )
        assertEquals(36.0, before, 0.5)
        assertTrue("سماحُ الدوّار لم يتّسع", inside > before)
        assertEquals("**سماحُ الدوّار بقي بعد مغادرته**", 36.0, after, 0.5)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٧ · التذبذبُ والعودة**
    // ══════════════════════════════════════════════════════════════════

    /** **OFFR-060** — شكٌّ ثمّ عودةٌ قبل التأكيد: لا خروج. */
    @Test
    fun `OFFR-060 العودةُ قبل التأكيد تُلغي الشكّ`() {
        val (route, fixes) = OffRouteFixtures.suspectThenReturn()
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, fixes)
        val seq = OffRouteFixtures.states(out)
        println("OFFR-060 · التسلسل=${seq.joinToString(",") { it.name.take(3) }}")
        assertTrue("لم يقع شكٌّ أصلاً", seq.any { it == SUS })
        assertEquals("العودةُ لم تُلغِ الشكّ", 0, OffRouteFixtures.falseOffRoutes(out))
        assertEquals("لم يعد إلى المسار", ON, seq.last())
    }

    /** **OFFR-061** — والعودةُ من الخروج المؤكَّد تحتاج ثلاثاً متتالية. */
    @Test
    fun `OFFR-061 العودةُ من الخروج تحتاج دليلا`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val e = OffRouteFixtures.engineOn(route)
        // **ويُوقَف عند التأكيد** — لا يُترك يبتعد أربعَ مئةِ متر:
        // **عودةٌ من هناك قفزةٌ يرفضها `GpsQuality` بـ`TELEPORT`**،
        // فيُختبر المرشِّحُ لا العودة.
        val leg = fixes.take(13)
        OffRouteFixtures.replay(e, leg)
        assertEquals("لم يُعلَن خروجٌ أصلاً", OFF, e.detector.state)

        // **ويعود غرباً خمسةً وعشرين متراً في كلّ قراءةٍ** وهو ماضٍ
        // شمالاً — **سيرٌ ممكنٌ لا قفزة.**
        val last = leg.last()
        val offsets = listOf(55.0, 30.0, 5.0, 0.0, 0.0)
        val clean = offsets.mapIndexed { i, off ->
            last.copy(
                lat = last.lat + (i + 1) * 10.0 * 0.000009,
                lng = RouteFixtures.LNG0 + off * 0.0000111,
                atMs = last.atMs + (i + 1) * 1_000L,
            )
        }
        val states = clean.map { e.onFix(it).offRoute.state }
        println("OFFR-061 · العودة=${states.joinToString(",") { it.name.take(3) }}")
        assertTrue("عاد بقراءةٍ واحدة", states.first() == OFF)
        assertEquals("لم يعد بعد ثلاثِ قراءاتٍ نظيفة", ON, states.last())
    }

    /** **OFFR-062** — ولا تتذبذب الحالُ كلَّ ثانية. */
    @Test
    fun `OFFR-062 لا تذبذبَ على الحافّة`() {
        val route = RouteFixtures.straight()
        val e = OffRouteFixtures.engineOn(route)
        val base = RouteFixtures.driveAlong(route, speedMps = 10f)
        // **قراءاتٌ تتناوب حولَ العتبة بالضبط** — وهي الحالُ التي
        // تُذبذب من لا حارسَ عنده.
        val edgy = base.mapIndexed { i, f ->
            val d = if (i % 2 == 0) 40.0 else 5.0
            f.copy(lng = f.lng + d * 0.0000111)
        }
        val out = OffRouteFixtures.replay(e, edgy)
        val trans = OffRouteFixtures.transitions(out)
        println("OFFR-062 · انتقالات=$trans من ${out.size} قراءة · التسلسل=${OffRouteFixtures.states(out).joinToString("") { if (it == ON) "." else if (it == SUS) "?" else "X" }}")
        assertTrue("الحالُ تتذبذب: $trans انتقالاً", trans <= 2)
        assertEquals("تناوبٌ حولَ العتبة أعلن خروجا", 0, OffRouteFixtures.falseOffRoutes(out))
    }

    // ══════════════════════════════════════════════════════════════════
    // **٨ · الانحرافُ الحقيقيُّ — بالقياس**
    // ══════════════════════════════════════════════════════════════════

    /** **OFFR-070** — انحرافٌ حقيقيٌّ يُؤكَّد في زمنٍ معقول. */
    @Test
    fun `OFFR-070 الانحرافُ الحقيقيُّ يُؤكَّد`() {
        val (route, fixes, at) = OffRouteFixtures.realDeviation()
        val e = OffRouteFixtures.engineOn(route)
        val out = OffRouteFixtures.replay(e, fixes)
        val t = OffRouteFixtures.measure(out, fixes, at)
        println(
            "OFFR-070 · أوّلُ شكٍّ=${t.firstSuspect} · تأكيدٌ=${t.confirmed} · " +
                "ثوانٍ=${t.secondsToOffRoute} · أمتارٌ بعد الانحراف=${"%.1f".format(t.metersAfterDeviation)}",
        )
        println("OFFR-070 · تدرّجُ النقاط=${t.scores.joinToString(",") { "%.1f".format(it) }}")
        assertTrue("لم يُعلَن خروجٌ أبداً", t.confirmed >= 0)
        assertTrue("الشكُّ لم يسبق التأكيد", t.firstSuspect in 0 until t.confirmed)
        assertTrue("تأخّر التأكيدُ أكثرَ من عشرِ ثوانٍ", t.secondsToOffRoute in 0.0..10.0)
    }

    /** **OFFR-071** — والمناورةُ والزمنُ يعملان بعد الخروج بلا انكسار. */
    @Test
    fun `OFFR-071 الحالُ تبقى كاملةً بعد الخروج`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val out = OffRouteFixtures.replay(OffRouteFixtures.engineOn(route), fixes)
        val last = out.last()
        assertTrue(last.isOffRoute)
        assertNotNull("المناورةُ ضاعت بعد الخروج", last.currentManeuver)
        assertTrue("ما بقي انكسر", last.remainingM >= 0.0)
        assertTrue("الزمنُ انكسر", last.remainingSec >= 0.0)
    }
}
