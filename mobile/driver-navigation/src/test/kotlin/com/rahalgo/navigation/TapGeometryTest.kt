package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **لمسُ الخطّ — البنود ١٣ إلى ١٦**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ صحّة واجهة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 */
class TapGeometryTest {

    private val density = 1f

    private fun pt(x: Float, y: Float) = TapGeometry.ScreenPoint(x, y)

    // ══════════════════════════════════════════════════════════════
    // **البند ١٤ — الرفيدةُ التي سمّاها المالك**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `قطعةٌ طويلةٌ بلا رؤوسَ تفوز على رأسٍ قريب`() {
        /**
         * **أمرُ المالك نصّاً**:
         *
         *	Route A: segment طويل من (0,0) إلى (1000,0)
         *	tap عند (500,5)
         *	Route B: vertex عند (500,30)
         *
         *	Nearest vertex algorithm قد يختار B خطأً.
         *	Point-to-segment يجب أن يختار A.
         */
        val a = "A" to listOf(pt(0f, 0f), pt(1000f, 0f))
        val b = "B" to listOf(pt(400f, 200f), pt(500f, 30f), pt(600f, 200f))

        // ── بالرؤوس: أقربُ رأسٍ في «أ» يبعد ٥٠٠ وفي «ب» ٢٥ ─────────
        val nearestVertexA = listOf(pt(0f, 0f), pt(1000f, 0f))
            .minOf { kotlin.math.hypot(it.x - 500f, it.y - 5f) }
        val nearestVertexB = b.second.minOf { kotlin.math.hypot(it.x - 500f, it.y - 5f) }
        assertTrue(
            "الرفيدةُ يجب أن تُوقع خوارزميّةَ الرؤوس: $nearestVertexA مقابل $nearestVertexB",
            nearestVertexB < nearestVertexA,
        )

        // ── وبالقطع: «أ» تبعد خمسةً و«ب» خمسةً وعشرين ─────────────
        assertEquals(5f, TapGeometry.distanceToPath(a.second, 500f, 5f), 0.01f)
        assertEquals(25f, TapGeometry.distanceToPath(b.second, 500f, 5f), 0.01f)

        val hit = TapGeometry.classify(listOf(a, b), 500f, 5f, density)
        assertTrue("$hit", hit is TapGeometry.Hit.One)
        assertEquals("A", (hit as TapGeometry.Hit.One).routeId)
    }

    @Test
    fun `مساران متقاطعان`() {
        val a = "A" to listOf(pt(0f, 100f), pt(1000f, 100f))
        val b = "B" to listOf(pt(500f, 0f), pt(500f, 1000f))

        // ── لمسةٌ على «أ» بعيداً عن التقاطع ───────────────────────
        val onA = TapGeometry.classify(listOf(a, b), 200f, 102f, density)
        assertEquals("A", (onA as TapGeometry.Hit.One).routeId)

        // ── ولمسةٌ على «ب» بعيداً عنه ─────────────────────────────
        val onB = TapGeometry.classify(listOf(a, b), 498f, 700f, density)
        assertEquals("B", (onB as TapGeometry.Hit.One).routeId)

        // ── وعند التقاطع نفسِه: التباس ────────────────────────────
        val atCross = TapGeometry.classify(listOf(a, b), 500f, 100f, density)
        assertTrue("$atCross", atCross is TapGeometry.Hit.Ambiguous)
    }

    @Test
    fun `مساران شبهُ متطابقين فالتباس`() {
        // **البند ١٥** — «لا تخمّن».
        val a = "A" to listOf(pt(0f, 100f), pt(1000f, 100f))
        val b = "B" to listOf(pt(0f, 102f), pt(1000f, 102f))

        val hit = TapGeometry.classify(listOf(a, b), 500f, 101f, density)
        assertTrue("$hit", hit is TapGeometry.Hit.Ambiguous)
        assertNull("ولا معاينةَ تلقائيّة", TapGeometry.pick(listOf(a, b), 500f, 101f, density))
    }

    @Test
    fun `والمسافةُ نفسُها تماماً فالتباس`() {
        val a = "A" to listOf(pt(0f, 90f), pt(1000f, 90f))
        val b = "B" to listOf(pt(0f, 110f), pt(1000f, 110f))
        val hit = TapGeometry.classify(listOf(a, b), 500f, 100f, density)
        assertTrue("$hit", hit is TapGeometry.Hit.Ambiguous)
    }

    @Test
    fun `ومتباعدان فيفوز الأقرب`() {
        val a = "A" to listOf(pt(0f, 100f), pt(1000f, 100f))
        val b = "B" to listOf(pt(0f, 122f), pt(1000f, 122f))
        // **الفرقُ عشرون بكسلاً** — فوق عتبة الالتباس (٨).
        val hit = TapGeometry.classify(listOf(a, b), 500f, 101f, density)
        assertEquals("A", (hit as TapGeometry.Hit.One).routeId)
    }

    // ══════════════════════════════════════════════════════════════
    // **الغيابُ والبعد**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `لا مرشَّحين فلا شيء`() {
        assertEquals(TapGeometry.Hit.None, TapGeometry.classify(emptyList(), 0f, 0f, density))
        assertNull(TapGeometry.pick(emptyList(), 0f, 0f, density))
    }

    @Test
    fun `ولمسةٌ بعيدةٌ عن كلّ خطٍّ لا تختار`() {
        // **البند ١٧** — لمسةٌ على غير بديلٍ تُلغي المعاينة.
        val a = "A" to listOf(pt(0f, 0f), pt(1000f, 0f))
        val hit = TapGeometry.classify(listOf(a), 500f, 300f, density)
        assertEquals(TapGeometry.Hit.None, hit)
    }

    @Test
    fun `والكثافةُ تُقاس بالـdp لا بالبكسل`() {
        // **شاشةٌ كثيفةٌ لا تصير أدقَّ إصبعاً.**
        val a = "A" to listOf(pt(0f, 0f), pt(1000f, 0f))

        // **بكثافة ١: ٣٠ بكسلاً خارجَ ٢٤dp.**
        assertEquals(TapGeometry.Hit.None, TapGeometry.classify(listOf(a), 500f, 30f, 1f))
        // **وبكثافة ٣: ٣٠ بكسلاً داخلَ ٧٢ بكسلاً.**
        assertTrue(TapGeometry.classify(listOf(a), 500f, 30f, 3f) is TapGeometry.Hit.One)
    }

    // ══════════════════════════════════════════════════════════════
    // **الهندسةُ نفسُها**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `مسافةُ القطعة محصورةٌ بين الطرفين`() {
        val a = pt(0f, 0f)
        val b = pt(100f, 0f)
        // **داخلَ القطعة** — إسقاطٌ عموديّ.
        assertEquals(10f, TapGeometry.distanceToSegment(a, b, 50f, 10f), 0.01f)
        // **وخارجَها** — المسافةُ إلى الطرف.
        assertEquals(50f, TapGeometry.distanceToSegment(a, b, 150f, 0f), 0.01f)
        assertEquals(50f, TapGeometry.distanceToSegment(a, b, -50f, 0f), 0.01f)
    }

    @Test
    fun `وقطعةٌ صفريّةُ الطول لا تُسقط`() {
        val a = pt(10f, 10f)
        assertEquals(0f, TapGeometry.distanceToSegment(a, a, 10f, 10f), 0.01f)
        assertEquals(5f, TapGeometry.distanceToSegment(a, a, 15f, 10f), 0.01f)
    }

    @Test
    fun `ومسارٌ برأسٍ واحدٍ لا يُسقط`() {
        assertEquals(5f, TapGeometry.distanceToPath(listOf(pt(0f, 0f)), 5f, 0f), 0.01f)
        assertEquals(Float.MAX_VALUE, TapGeometry.distanceToPath(emptyList(), 0f, 0f), 0.01f)
    }
}
