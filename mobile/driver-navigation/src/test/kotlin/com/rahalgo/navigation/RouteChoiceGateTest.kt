package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **سباقاتُ الردّ وتوافقُ الموصى به — البنود ١ إلى ٧ و١٨ و١٩**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ صحّة واجهة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 */
class RouteChoiceGateTest {

    private fun ctx(
        seq: Long = 1,
        orderId: String = "o1",
        target: RouteTarget = RouteTarget.PICKUP,
        generation: Long = 4,
        fingerprint: Long = 111L,
    ) = RouteChoiceGate.RequestContext(seq, orderId, target, generation, fingerprint)

    private fun verdict(
        request: RouteChoiceGate.RequestContext,
        latestSeq: Long = request.seq,
        now: RouteChoiceGate.RequestContext = request,
        recommendedFingerprint: Long = request.navRouteFingerprint,
        alternativeCount: Int = 1,
        recommendedGeometry: List<GeoPoint>? = null,
        currentGeometry: List<GeoPoint>? = null,
    ) = RouteChoiceGate.verdict(
        request, latestSeq, now, recommendedFingerprint, alternativeCount,
        recommendedGeometry = recommendedGeometry,
        currentGeometry = currentGeometry,
    )

    // ══════════════════════════════════════════════════════════════
    // **البند ١٨ — سباقاتُ الردّ**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `أ - ردُّ المتجر يصل بعد الاستلام فيُطرح`() {
        /**
         * **السيناريو الذي سمّاه المالك بعينه**:
         *
         *	PICKUP request A starts
         *	↓ picked_up · clearChoices · DROPOFF request B starts
         *	↓ A returns late  →  must be discarded
         */
        val a = ctx(seq = 1, target = RouteTarget.PICKUP)
        val now = ctx(seq = 2, target = RouteTarget.DROPOFF)

        val out = verdict(a, latestSeq = 2, now = now)
        assertFalse(RouteChoiceGate.accepted(out))
        // **والتسلسلُ يمسكه أوّلاً** — وهو الحارسُ الأعمّ.
        assertEquals(RouteChoiceGate.Reject.STALE_SEQ, out)
    }

    @Test
    fun `أ٢ - وحتّى لو كان التسلسلُ هو الأحدث فالوجهةُ تمسك`() {
        val a = ctx(seq = 1, target = RouteTarget.PICKUP)
        val now = ctx(seq = 1, target = RouteTarget.DROPOFF)
        assertEquals(RouteChoiceGate.Reject.TARGET_CHANGED, verdict(a, 1, now))
    }

    @Test
    fun `ب - ردُّ جيلٍ خمسةٍ بعد تركيب ستّةٍ يُطرح`() {
        val a = ctx(seq = 1, generation = 5)
        val now = ctx(seq = 1, generation = 6)
        assertEquals(RouteChoiceGate.Reject.GENERATION_CHANGED, verdict(a, 1, now))
    }

    @Test
    fun `ج - الأقدمُ لا يدهس الأحدثَ ولو وصل بعده`() {
        /**
         * **`B` انطلق بعد `A` وردَّ أوّلاً ونُشر.** ثمّ يصل `A`.
         *
         * **ولا يدهسه** — التسلسلُ لا الوقت.
         */
        val a = ctx(seq = 1)
        val b = ctx(seq = 2)

        assertTrue("ب يُنشَر", RouteChoiceGate.accepted(verdict(b, latestSeq = 2, now = b)))
        assertEquals(
            "وأ يُطرح ولو وصل أخيراً",
            RouteChoiceGate.Reject.STALE_SEQ,
            verdict(a, latestSeq = 2, now = b.copy(seq = 1)),
        )
    }

    @Test
    fun `وتبدّلُ الطلب يُطرح`() {
        val a = ctx(orderId = "o1")
        assertEquals(
            RouteChoiceGate.Reject.ORDER_CHANGED,
            verdict(a, now = a.copy(orderId = "o2")),
        )
    }

    @Test
    fun `وردٌّ بلا بدائلَ لا يُنشَر`() {
        // **البند ٥ من الواجهة** — لا لوحةَ فارغة.
        assertEquals(RouteChoiceGate.Reject.EMPTY, verdict(ctx(), alternativeCount = 0))
    }

    @Test
    fun `والردُّ السليمُ يُنشَر`() {
        assertTrue(RouteChoiceGate.accepted(verdict(ctx())))
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ١٩ — توافقُ الموصى به**
    // ══════════════════════════════════════════════════════════════

    private fun line(vararg pts: Pair<Double, Double>) = pts.map { GeoPoint(it.first, it.second) }

    private fun straight(lengthM: Double, stepM: Double, latOffsetM: Double = 0.0): List<GeoPoint> {
        val out = mutableListOf<GeoPoint>()
        var d = 0.0
        while (d <= lengthM) {
            out += GeoPoint(35.95 + latOffsetM * 0.000009, 39.005 + d * 0.0000111)
            d += stepM
        }
        return out
    }

    @Test
    fun `مسارٌ مختلفٌ تماماً يُطرح`() {
        // **الملاحةُ على R1 والردُّ موصًى به R2** — ليست بدائلَ لما نقود.
        val current = straight(3000.0, 25.0)
        val other = straight(3000.0, 25.0, latOffsetM = 500.0)

        val out = verdict(
            ctx(fingerprint = 111L),
            recommendedFingerprint = 999L,
            recommendedGeometry = other,
            currentGeometry = current,
        )
        assertEquals(RouteChoiceGate.Reject.RECOMMENDED_MISMATCH, out)
    }

    @Test
    fun `والمسارُ نفسُه بتقطيعٍ مختلفٍ يُقبل`() {
        /**
         * **أمرُ المالك**: «same logical route · different harmless
         * geometry sampling → إذا identity الحالية شديدة الصرامة،
         * استخدم compatibility check مناسبًا حتى لا نرفض Response
         * سليمة».
         *
         * **والبصمةُ صارمة**: رأسٌ واحدٌ يتزحزح يُبدّلها. **فلو
         * اكتُفي بها لرُفض كلُّ ردٍّ تقريباً** — أصلُ الطلب الثاني
         * موضعُ السائق وقد تحرّك.
         */
        val dense = straight(3000.0, 10.0)
        val sparse = straight(3000.0, 100.0)

        val out = verdict(
            ctx(fingerprint = 111L),
            recommendedFingerprint = 222L,
            recommendedGeometry = sparse,
            currentGeometry = dense,
        )
        assertTrue("$out", RouteChoiceGate.accepted(out))
    }

    @Test
    fun `والبصمةُ المطابقةُ تكفي بلا هندسة`() {
        val out = verdict(ctx(fingerprint = 111L), recommendedFingerprint = 111L)
        assertTrue(RouteChoiceGate.accepted(out))
    }

    @Test
    fun `ولا هندسةَ ولا بصمةً مطابقةً فيُطرح`() {
        val out = verdict(ctx(fingerprint = 111L), recommendedFingerprint = 222L)
        assertEquals(RouteChoiceGate.Reject.RECOMMENDED_MISMATCH, out)
    }

    // ══════════════════════════════════════════════════════════════
    // **والتوافقُ ليس التنوّع**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `التوافقُ يقبل الإزاحةَ الصغيرةَ ويرفض الطريقَ الآخر`() {
        val base = straight(3000.0, 25.0)
        assertTrue("نفسُه", RouteCompatibility.sameRoute(base, base))
        assertTrue("مُزاحٌ عشرين متراً", RouteCompatibility.sameRoute(base, straight(3000.0, 25.0, 20.0)))
        assertFalse("ومُزاحٌ ستّين", RouteCompatibility.sameRoute(base, straight(3000.0, 25.0, 60.0)))
    }

    @Test
    fun `ويرفض ما اختلف طولُه كثيراً`() {
        val base = straight(3000.0, 25.0)
        val longer = straight(5000.0, 25.0)
        assertFalse(RouteCompatibility.sameRoute(base, longer))
    }

    @Test
    fun `والتوافقُ أصرمُ من التنوّع`() {
        /**
         * **قِيس في الخادم أنّ أقصى تشابهِ بديلين حقيقيّين ٨٢٫٦٪.**
         *
         * **وعتبةُ التوافق ٩٠٪** — فبديلٌ قريبٌ لا يُعدّ «هو هو»،
         * **والمسارُ نفسُه بتقطيعٍ مختلفٍ يُعدّ.**
         */
        assertTrue(RouteCompatibility.Tuning().minShared > 0.826)
    }

    // ══════════════════════════════════════════════════════════════
    // **البندان ٦ و٧ — سببُ التركيب**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `اختيارُ السائق لا يُطلق جلباً`() {
        assertFalse(RouteInstallReason.USER_SELECTION.fetchesAlternatives)
    }

    @Test
    fun `وما عداه يطلق`() {
        assertTrue(RouteInstallReason.INITIAL.fetchesAlternatives)
        assertTrue(RouteInstallReason.TARGET_CHANGED.fetchesAlternatives)
        assertTrue(RouteInstallReason.AUTOMATIC_REROUTE.fetchesAlternatives)
    }

    @Test
    fun `والسببان يتميّزان ولا يُخلطان بالجيل`() {
        /**
         * **البند ٩**: «successful AUTOMATIC_REROUTE → يمكن طلب
         * Alternatives جديدة… لكن يجب أن تكون مميزة عن
         * USER_ROUTE_SELECTION ولا تعتمد على generation وحدها».
         *
         * **وكلاهما يزيد الجيل** — فالتمييزُ بالسبب لا به.
         */
        assertTrue(
            RouteInstallReason.AUTOMATIC_REROUTE != RouteInstallReason.USER_SELECTION,
        )
        assertTrue(RouteInstallReason.AUTOMATIC_REROUTE.fetchesAlternatives)
        assertFalse(RouteInstallReason.USER_SELECTION.fetchesAlternatives)
    }
}
