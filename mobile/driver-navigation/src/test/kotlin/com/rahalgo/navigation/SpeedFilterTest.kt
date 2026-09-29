package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حارسُ السرعة — الواقفُ صفرٌ، والسائرُ رقمُه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-٠٩-٢٩: «الهاتفُ واقفٌ وأندرويد يقول ٤ و٥ و٦ كم/س».)
 *
 * **والأرقامُ هنا من الجهاز نفسِه** — قِيست على SM-A525F واقفاً:
 * `speedMps` بين ١٫١ و١٫٧، والدقّةُ ١٣م، والإزاحةُ بين قراءتين أمتارٌ
 * قليلةٌ دون الدقّة. **فلو صُدِّقت لظهرت ٤ و٥ و٦.**
 */
class SpeedFilterTest {

    private fun fix(
        lat: Double,
        lng: Double,
        atMs: Long,
        speed: Float?,
        acc: Float = 13f,
    ) = NavFix(
        lat = lat, lng = lng, accuracyM = acc, speedMps = speed,
        bearingDeg = null, atMs = atMs, elapsedNs = atMs * 1_000_000,
        wallMs = atMs, provider = "gps",
    )

    /** **هاتفٌ واقفٌ وضجيجٌ يقول «يسير»** — يجب أن يستقرّ على صفر. */
    @Test
    fun `الواقف يستقر على صفر رغم ضجيج التموضع`() {
        val f = SpeedFilter()
        // **نقطةٌ واحدةٌ تتراقص في دائرةِ شكٍّ نصفُها ١٣م** — كما قِيس.
        val jitter = listOf(
            Triple(35.960710, 39.014012, 1.2f),
            Triple(35.960715, 39.014019, 1.6f),
            Triple(35.960706, 39.014005, 1.1f),
            Triple(35.960712, 39.014021, 1.7f),
            Triple(35.960708, 39.014010, 1.4f),
        )
        var t = 1_000L
        val seen = mutableListOf<Int>()
        for ((lat, lng, sp) in jitter) {
            seen += f.next(fix(lat, lng, t, sp))
            t += 1_000L
        }
        assertEquals("**ضجيجُ الواقف عُرض حركةً**: $seen", listOf(0, 0, 0, 0, 0), seen)
    }

    /** **وقراءةٌ بلا سرعةٍ ولا سابقةٍ وقوفٌ** — لا رقمٌ مخترَع. */
    @Test
    fun `أول قراءة بلا سرعة تعطي صفرا`() {
        assertEquals(0, SpeedFilter().next(fix(35.96, 39.01, 1_000L, null)))
    }

    /** **وسائرٌ حقيقيٌّ يُقرأ رقمَه** — ٣٦ كم/س = ١٠ م/ث. */
    @Test
    fun `السير الحقيقي يظهر برقمه`() {
        val f = SpeedFilter()
        // **عشرةُ أمتارٍ في الثانية شمالاً** — إزاحةٌ تُصدّق الرقم.
        var lat = 35.96000
        var t = 1_000L
        var last = 0
        repeat(5) {
            last = f.next(fix(lat, 39.01, t, 10f))
            lat += 0.00009 // ≈ 10 م
            t += 1_000L
        }
        assertEquals("**سيرٌ حقيقيٌّ لم يُقرأ**", 36, last)
    }

    /**
     * **والهدنةُ تمنع التراقص حول العتبة** — من انطلق يبقى سائراً حتّى
     * ينزل تحت الأدنى، **ولا يظهر ويغيب في كلّ ثانية.**
     */
    @Test
    fun `الهدنة تمنع التراقص حول العتبة`() {
        val f = SpeedFilter()
        var lat = 35.96000
        var t = 1_000L
        // **ينطلق** — سرعةٌ وإزاحةٌ تصدّقها.
        repeat(3) {
            f.next(fix(lat, 39.01, t, 10f)); lat += 0.00009; t += 1_000L
        }
        // **ثمّ يبطئ إلى ٦ كم/س (١٫٧ م/ث)** — فوق `movingOff` (١٫١) فيبقى
        // سائراً، **ولا يقفز إلى صفرٍ لمجرّد أنّه أبطأ من عتبة الانطلاق.**
        val slow = f.next(fix(lat, 39.01, t, 1.7f))
        assertTrue("**سائرٌ أبطأَ فصُفِّر** — تراقصٌ حول العتبة: $slow", slow > 0)
        // **ثمّ يقف** — تحت `movingOff` فيعود صفرا.
        t += 1_000L
        val stopped = f.next(fix(lat, 39.01, t, 0.3f))
        assertEquals("**واقفٌ لم يُصفَّر**", 0, stopped)
    }

    /** **ودقّةٌ رديئةٌ لا تُصدَّق إزاحتُها** — نقطةٌ بدقّة ٦٠م تتراقص ٦٠م. */
    @Test
    fun `الدقة الرديئة لا تنطلق بها حركة`() {
        val f = SpeedFilter()
        // **إزاحةُ ٢٥م في ثانيةٍ بدقّة ٦٠م** — قد تكون النقطةَ نفسَها،
        // **والسرعةُ المعلنةُ صامتة.**
        f.next(fix(35.96000, 39.01, 1_000L, null, acc = 60f))
        val out = f.next(fix(35.96022, 39.01, 2_000L, null, acc = 60f))
        assertEquals("**دقّةٌ رديئةٌ صُدِّقت حركةً**", 0, out)
    }

    /** **وجلسةٌ جديدةٌ لا ترث حالَ سابقتها.** */
    @Test
    fun `الإنساء يعيد الحال إلى واقف`() {
        val f = SpeedFilter()
        var lat = 35.96000
        var t = 1_000L
        repeat(3) { f.next(fix(lat, 39.01, t, 10f)); lat += 0.00009; t += 1_000L }
        f.reset()
        // **بعد الإنساء: قراءةٌ بطيئةٌ وحدَها لا تُقرأ سيراً** — ولو كانت
        // الجلسةُ السابقةُ «تسير».
        assertEquals(0, f.next(fix(lat, 39.01, t, 1.7f)))
    }
}
