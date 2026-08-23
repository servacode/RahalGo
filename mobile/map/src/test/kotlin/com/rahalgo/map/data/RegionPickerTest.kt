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

    // ══════════════════════════════════════════════════════════════════
    // **والإطاران الحقيقيّان من `manifest.json`** — لا مصنوعان
    // ══════════════════════════════════════════════════════════════════
    //
    // **قِيسا من الفهرس الحيّ ٢٠٢٦-٠٨-٢٣** بعد بناء حزمة دمشق:
    //
    //	raqqa      1,919,410 بايت
    //	damascus  13,070,736 بايت
    //
    // **وإطارٌ في الاختبار يفترق عن إطارٍ في الفهرس يجعل الحارسَ
    // يحرس شيئاً لا وجودَ له.**
    private val raqqa = region("raqqa", 38.92, 35.88, 39.12, 36.03)
    private val damascus = region("damascus", 36.15, 33.40, 36.45, 33.62)
    private val all = listOf(raqqa, damascus)

    @Test
    fun `a point inside a region picks it`() {
        // **موضعٌ حقيقيٌّ قِيس من جهاز المالك ٢٠٢٦-٠٨-٢٣.**
        assertEquals("raqqa", RegionPicker.of(all, 35.9528, 39.0079)?.id)
    }

    /** **ودمشقُ لمن هو فيها** — ٣٣٫٥١٣٨ شمالاً · ٣٦٫٢٧٦٥ شرقاً. */
    @Test
    fun `each city picks its own`() {
        assertEquals("damascus", RegionPicker.of(all, 33.5138, 36.2765)?.id)
    }

    /**
     * **ولا تُخمَّن منطقةٌ لمن هو خارجَها كلِّها.**
     *
     * **وخريطةُ مدينةٍ أخرى أسوأُ من لا خريطة** — تُنزَّل ميغاباتٌ
     * على حزمته ثمّ لا يجد شارعَه فيها.
     */
    @Test
    fun `a point outside every region yields nothing`() {
        assertNull(RegionPicker.of(all, 36.20, 37.15)) // حلب — ولا حزمةَ لها بعد
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
