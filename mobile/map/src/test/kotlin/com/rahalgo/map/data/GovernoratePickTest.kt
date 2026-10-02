package com.rahalgo.map.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **حزمُ المحافظات الأربعَ عشرة — يُختار لكلّ مدينةٍ حزمةٌ تغطّيها**
 * ══════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠١: «نجزّئ الخريطة للـ١٤ محافظة».)
 *
 * **والمستطيلاتُ هنا هي المنشورةُ على التجهيز** (`regions/2026-10-01`) —
 * حدودُ OSM الإداريّةُ وهامشُ ٠٫٠٣°. **والمقيسُ شرطان:**
 *
 *  1. **مركزُ كلّ محافظةٍ يختار حزمتَها** — حيث لا تداخل يُربك.
 *  2. **والحزمةُ المختارةُ تغطّي النقطةَ دائماً** — ولو اختيرت جارةٌ قرب
 *     الحدود، **فالسائقُ يجد شارعَه فيها.**
 */
class GovernoratePickTest {

    /** **الحدودُ المنشورةُ نفسُها** — `regions/2026-10-01/polygons.json`. */
    private val polygons: org.json.JSONObject by lazy {
        org.json.JSONObject(
            javaClass.classLoader!!.getResource("governorates-2026-10-01.json")!!.readText(),
        )
    }

    private fun region(id: String, w: Double, s: Double, e: Double, n: Double): MapRegion {
        val o = org.json.JSONObject()
            .put("id", id).put("name", id).put("dataVersion", "2026-10-01")
            .put("url", "regions/2026-10-01/region-$id.pmtiles").put("bytes", 1)
            .put("sha256", "a".repeat(64)).put("minZoom", 10).put("maxZoom", 16)
            .put("bbox", org.json.JSONArray(listOf(w, s, e, n)))
            .put("polygon", polygons.getJSONArray(id))
        return MapRegion.fromJson(o)
    }

    private val regions = listOf(
        region("damascus", 36.1670, 33.4290, 36.3860, 33.6020),
        region("rif-dimashq", 35.7820, 32.6630, 39.1640, 34.2790),
        region("aleppo", 36.5140, 35.3540, 38.7510, 36.9540),
        region("homs", 36.0750, 33.5220, 40.1920, 35.4210),
        region("hama", 36.1040, 34.8230, 38.3340, 35.7920),
        region("latakia", 35.6870, 35.1870, 36.2850, 35.9710),
        region("tartus", 35.8240, 34.5970, 36.3540, 35.2940),
        region("idlib", 36.1190, 35.3470, 37.2460, 36.3730),
        region("raqqa", 38.0270, 35.2190, 39.8340, 36.8120),
        region("deir-ez-zor", 39.2120, 34.0460, 41.3540, 36.4940),
        region("hasakah", 39.4090, 35.5230, 42.4050, 37.3490),
        region("daraa", 35.7360, 32.3270, 36.6010, 33.3260),
        region("suwayda", 36.3060, 32.2820, 37.5240, 33.2580),
        region("quneitra", 35.7260, 32.7150, 36.0180, 33.3930),
    )

    /** مراكزُ المحافظات — (عرض، طول). */
    private val centers = mapOf(
        "damascus" to (33.5138 to 36.2765),
        "rif-dimashq" to (33.5700 to 36.4050), // دوما
        "aleppo" to (36.2021 to 37.1343),
        "homs" to (34.7324 to 36.7137),
        "hama" to (35.1318 to 36.7578),
        "latakia" to (35.5317 to 35.7901),
        "tartus" to (34.8890 to 35.8866),
        "idlib" to (35.9306 to 36.6339),
        "raqqa" to (35.9528 to 39.0079),
        "deir-ez-zor" to (35.3359 to 40.1408),
        "hasakah" to (36.5024 to 40.7477),
        "daraa" to (32.6189 to 36.1021),
        "suwayda" to (32.7090 to 36.5660),
        "quneitra" to (33.1260 to 35.8240), // البعث
    )

    @Test
    fun `الحزمةُ المختارةُ تغطّي مركزَ كلّ محافظة`() {
        for ((id, p) in centers) {
            val r = RegionPicker.of(regions, p.first, p.second)
            assertNotNull("لا حزمةَ لمركز $id", r)
            val (w, s, e, n) = r!!.bbox
            assert(p.second in w..e && p.first in s..n) { "حزمةُ ${r.id} لا تغطّي مركزَ $id" }
        }
    }

    @Test
    fun `الحدودُ تُقرأ من الفهرس لكلّ محافظة`() {
        regions.forEach { assertTrue("لا حدودَ لـ${it.id}", it.polygon.isNotEmpty()) }
    }

    @Test
    fun `كلُّ مركزٍ يختار حزمةَ محافظته`() {
        val wrong = centers.mapNotNull { (id, p) ->
            val got = RegionPicker.of(regions, p.first, p.second)?.id
            if (got == id) null else "$id ⇒ $got"
        }
        assertEquals("مراكزُ تختار حزمةَ جارة: $wrong", emptyList<String>(), wrong)
    }
}
