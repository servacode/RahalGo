package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نسبةُ ما قُطع — يُقاس ولا يُنظر إليه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٤: «يجب ألّا يلاحظ الشخصُ قصَّها أصلاً، يجب
 *  أن يكون الخطُّ تحت السهم ويختفي مع السهم» · ثمّ: «أبداً لا يجب أن
 *  يظهر خلف السهم».)
 *
 * # وما حُذف ولماذا
 *
 * **كان القصُّ يُعيد بناءَ الهندسة** — فلا يقع إلّا مع قراءةِ موقعٍ
 * جديدة، **مرّةً في الثانية**، والسهمُ يمشي ستّين مرّةً فيها.
 * **فيقفز الخطُّ خلفه ثمانيةَ أمتارٍ دفعةً واحدة.**
 *
 * **وأمسك الاختبارُ عيباً ثانياً فيه**: عند تجاوز آخر الخطّ كان يردّ
 * آخرَ ضلعين — **فيطول الخطُّ فجأةً من ستّةَ عشرَ متراً إلى تسعةٍ
 * وتسعين** في اللحظة التي يصل فيها السائق.
 *
 * **فحُذف كلُّه**، وصار الإخفاءُ بتدرّجٍ لونيٍّ يُبدَّل في كلّ إطار
 * (`Markers.setTraveled`).
 */
class RouteTrimTest {

    /** **خطٌّ من عشرِ نقاطٍ في الرقّة** — بين كلِّ اثنتين مئةُ مترٍ تقريباً. */
    private val line: List<LatLng> = (0..9).map { LatLng(35.9500, 38.9900 + it * 0.0011) }

    @Test
    fun `الطولُ يُحسب ويقارب المتوقّع`() {
        val total = RouteTrim.lengthM(line)
        assertTrue("طولٌ غيرُ معقول: $total", total in 700.0..1100.0)
        assertEquals(0.0, RouteTrim.lengthM(emptyList()), 0.0)
        assertEquals(0.0, RouteTrim.lengthM(listOf(line[0])), 0.0)
    }

    @Test
    fun `النسبةُ تساوي ما قُطع على الطول`() {
        val total = RouteTrim.lengthM(line)
        for (progress in listOf(0.0, 100.0, 250.0, 500.0, 800.0)) {
            val expected = (progress / total).coerceIn(0.0, 0.999)
            assertEquals("عند $progress", expected, RouteTrim.fraction(progress, total), 0.001)
        }
    }

    /**
     * **ولا ذيلَ خلف السهم** — أمرُ المالك ٢٠٢٦-٠٨-٢٤ نصّاً.
     *
     * **وكان عشرين متراً** خشيةَ أن يسبق الطرفُ السهمَ حين تهتزّ
     * الدقّة، **فرآه المالكُ ذيلاً قبيحاً وقرارُه يسبق.**
     */
    @Test
    fun `القطعُ عند قدم السائق لا خلفه`() {
        assertEquals(0.0, RouteTrim.TAIL_M, 0.0)
        val total = RouteTrim.lengthM(line)
        // **ومئةُ مترٍ من ثمانمئةٍ نسبتُها الثُمن** — لا الثُمنُ ناقصاً ذيلاً.
        assertEquals(100.0 / total, RouteTrim.fraction(100.0, total), 0.0005)
    }

    /**
     * **ولا يرتدّ الخطُّ إلى الوراء.**
     *
     * **وتقدُّمٌ يزيد يعني نسبةً تزيد** — ومن رأى الخطَّ يطول ظنّ أنّه ضلّ.
     */
    @Test
    fun `كلَّما تقدّم زادت النسبةُ ولم تنقص`() {
        val total = RouteTrim.lengthM(line)
        var previous = -1.0
        for (p in 0..1200 step 25) {
            val f = RouteTrim.fraction(p.toDouble(), total)
            assertTrue("نقصت عند $p: $f بعد $previous", f >= previous)
            previous = f
        }
    }

    /**
     * **ولا يُبتلع الخطُّ كلُّه.**
     *
     * **وخطٌّ يختفي تماماً يُقرأ عطباً لا وصولاً** — فيبقى منه ما يُرى.
     */
    @Test
    fun `النهايةُ لا تمحو الخطَّ`() {
        val total = RouteTrim.lengthM(line)
        for (p in listOf(total, total * 2, 99_999.0)) {
            assertTrue("ابتُلع عند $p", RouteTrim.fraction(p, total) <= 0.999)
        }
    }

    /**
     * **والقيمُ الفاسدةُ لا تُسقط الخريطة.**
     *
     * **و`NaN` تأتي من قسمةٍ على صفرٍ في حسابٍ سابق** — **والخريطةُ
     * تسقط ولا يُعرف أنّ العلّةَ في الملاحة.**
     */
    @Test
    fun `والقيمُ الفاسدةُ تردّ صفراً`() {
        val total = RouteTrim.lengthM(line)
        assertEquals(0.0, RouteTrim.fraction(Double.NaN, total), 0.0)
        assertEquals(0.0, RouteTrim.fraction(100.0, Double.NaN), 0.0)
        assertEquals(0.0, RouteTrim.fraction(100.0, 0.0), 0.0)
        assertEquals(0.0, RouteTrim.fraction(-50.0, total), 0.0)
        assertEquals(0.0, RouteTrim.fraction(Double.NEGATIVE_INFINITY, total), 0.0)
    }
}
