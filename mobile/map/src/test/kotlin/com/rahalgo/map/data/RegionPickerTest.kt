package com.rahalgo.map.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حارسُ اختيار المنطقة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وهذا الاختيارُ يقرّر ما يُنزَّل على حزمة السائق** — عشراتُ
 * الميغابايتات. **وخطأٌ فيه يستنزفها ثمّ لا يجد شارعَه في ما نزّل.**
 */
class RegionPickerTest {

    private fun region(id: String, w: Double, s: Double, e: Double, n: Double) =
        MapRegion(
            id = id,
            name = id,
            dataVersion = "2026-08-20",
            artifact = MapArtifact(
                url = "regions/$id.pmtiles",
                bytes = 1,
                sha256 = "x",
                minZoom = 10,
                maxZoom = 16,
                bbox = listOf(w, s, e, n),
            ),
        )

    // **والإطارُ الحقيقيُّ للرقّة من `manifest.json`.**
    private val raqqa = region("raqqa", 38.92, 35.88, 39.12, 36.03)
    private val aleppo = region("aleppo", 37.05, 36.13, 37.25, 36.28)
    private val all = listOf(raqqa, aleppo)

    @Test
    fun `a point inside a region picks it`() {
        // **موضعٌ حقيقيٌّ قِيس من جهاز المالك ٢٠٢٦-٠٨-٢٣.**
        assertEquals("raqqa", RegionPicker.of(all, 35.9528, 39.0079)?.id)
    }

    @Test
    fun `each city picks its own`() {
        assertEquals("aleppo", RegionPicker.of(all, 36.20, 37.15)?.id)
    }

    /**
     * **ولا تُخمَّن منطقةٌ لمن هو خارجَها كلِّها.**
     *
     * **وخريطةُ مدينةٍ أخرى أسوأُ من لا خريطة** — تُنزَّل ميغاباتٌ
     * على حزمته ثمّ لا يجد شارعَه فيها.
     */
    @Test
    fun `a point outside every region yields nothing`() {
        assertNull(RegionPicker.of(all, 33.51, 36.29)) // دمشق — ولا حزمةَ لها بعد
        assertNull(RegionPicker.of(emptyList(), 35.9528, 39.0079))
    }

    /**
     * **وعند التداخُل تُؤخذ الأصغر.**
     *
     * **والمحافظةُ تحتوي المدينة** — ومن أخذ الكبرى نزّل خريطةَ
     * محافظةٍ ليقود في حيّ.
     */
    @Test
    fun `overlapping regions resolve to the smallest`() {
        val governorate = region("raqqa-gov", 38.0, 35.0, 40.0, 37.0)
        val picked = RegionPicker.of(listOf(governorate, raqqa), 35.9528, 39.0079)
        assertEquals("raqqa", picked?.id)
    }

    /**
     * **والإطارُ `[غرب، جنوب، شرق، شمال]`** — وترتيبٌ يُخمَّن يقلب
     * الطولَ بالعرض **فيضع الرقّةَ في المحيط.**
     */
    @Test
    fun `swapping lat and lng finds nothing`() {
        assertNull(RegionPicker.of(all, 39.0079, 35.9528))
    }

    /** **وإطارٌ ناقصٌ لا يُقرأ نصفَه** — يُتخطّى ولا يُسقط الاختيار. */
    @Test
    fun `a malformed bbox is skipped`() {
        val broken = MapRegion(
            id = "broken", name = "broken", dataVersion = "1",
            artifact = MapArtifact("u", 1, "x", 10, 16, listOf(38.9, 35.8)),
        )
        assertEquals("raqqa", RegionPicker.of(listOf(broken, raqqa), 35.9528, 39.0079)?.id)
    }
}
