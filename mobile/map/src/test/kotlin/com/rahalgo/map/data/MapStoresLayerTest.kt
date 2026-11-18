package com.rahalgo.map.data

import org.json.JSONObject
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **متاجرُنا نقاطٌ على كلّ خريطة** (قرارُ المالك ٢٠٢٦-١٠-٢١) — تُحقن في النمط حين يُعطى
 * عنوانُها، **وفوقَ الطبقات كلّها**؛ وبلا عنوانٍ لا شيء.
 */
class MapStoresLayerTest {
    private val canonical = """
        {"version":8,"sources":{"base":{"type":"vector"}},
         "layers":[{"id":"poi","type":"symbol","source":"base","layout":{"text-font":["RahalGo Regular"]}}]}
    """.trimIndent()

    @After
    fun reset() {
        MapStores.url = ""
        MapStores.landmarksUrl = ""
        MapStores.areasUrl = ""
    }

    private fun bound(): JSONObject = JSONObject(
        MapStyleBinding.online(canonical, "https://maps.example/base.pmtiles", "https://maps.example/res", "").json,
    )

    @Test
    fun noUrlNoLayer() {
        val s = bound()
        assertFalse(s.getJSONObject("sources").has("rahalgo-stores"))
        assertEquals(1, s.getJSONArray("layers").length())
    }

    @Test
    fun storesOnTopWithTheStyleFont() {
        MapStores.url = "https://api.example/api/v1/public/map/stores?format=geojson"
        val s = bound()
        val src = s.getJSONObject("sources").getJSONObject("rahalgo-stores")
        assertEquals("geojson", src.getString("type"))
        val layers = s.getJSONArray("layers")
        assertEquals("rahalgo-stores-dot", layers.getJSONObject(layers.length() - 2).getString("id"))
        val name = layers.getJSONObject(layers.length() - 1)
        assertEquals("rahalgo-stores-name", name.getString("id"))
        assertTrue(name.getJSONObject("layout").getJSONArray("text-font").getString(0) == "RahalGo Regular")
    }

    @Test
    fun landmarksClonePoiLayersFromTheStyle() {
        MapStores.landmarksUrl = "https://api.example/api/v1/public/map/landmarks?format=geojson"
        val withPoi = """
            {"version":8,"sources":{"base":{"type":"vector"}},
             "layers":[{"id":"poi-label","type":"symbol","source":"base","source-layer":"poi","minzoom":15,
                        "layout":{"text-font":["RahalGo Regular"]}}]}
        """.trimIndent()
        val s = JSONObject(
            MapStyleBinding.online(withPoi, "https://maps.example/base.pmtiles", "https://maps.example/res", "").json,
        )
        assertTrue(s.getJSONObject("sources").has("rahalgo-landmarks"))
        val layers = s.getJSONArray("layers")
        val last = layers.getJSONObject(layers.length() - 1)
        assertEquals("rahalgo-lm-poi-label", last.getString("id"))
        assertEquals("rahalgo-landmarks", last.getString("source"))
        assertFalse(last.has("source-layer"))
    }

    @Test
    fun landmarksWithoutPoiLayerAddNothing() {
        MapStores.landmarksUrl = "https://api.example/x"
        assertFalse(bound().getJSONObject("sources").has("rahalgo-landmarks"))
    }

    /** **أسماءُ الأحياء والشوارع المُقرَّة تظهر عند الكلّ** (قرارُ المالك ٢٠٢٦-١١-١٨). */
    @Test
    fun areaNamesCloneHoodAndStreetLayers() {
        MapStores.areasUrl = "https://api.example/api/v1/public/map/areas"
        val withNames = """
            {"version":8,"sources":{"base":{"type":"vector"}},
             "layers":[
               {"id":"road-label-minor","type":"symbol","source":"base","source-layer":"transportation_name",
                "layout":{"symbol-placement":"line","text-field":["get","name"],"text-font":["RahalGo Regular"]}},
               {"id":"place-neighbourhood","type":"symbol","source":"base","source-layer":"place","minzoom":11,
                "layout":{"text-field":["get","name"],"text-font":["RahalGo Regular"],"symbol-sort-key":1}}]}
        """.trimIndent()
        val s = JSONObject(
            MapStyleBinding.online(withNames, "https://maps.example/base.pmtiles", "https://maps.example/res", "").json,
        )
        val src = s.getJSONObject("sources").getJSONObject("rahalgo-areas")
        assertEquals("https://api.example/api/v1/public/map/areas", src.getString("data"))
        val layers = s.getJSONArray("layers")
        val byId = (0 until layers.length()).map { layers.getJSONObject(it) }.associateBy { it.getString("id") }
        val street = byId.getValue("rahalgo-area-street")
        val hood = byId.getValue("rahalgo-area-neighbourhood")
        for (l in listOf(street, hood)) {
            assertEquals("rahalgo-areas", l.getString("source"))
            assertFalse(l.has("source-layer"))
            assertEquals("[\"get\",\"name\"]", l.getJSONObject("layout").getJSONArray("text-field").toString())
        }
        assertEquals("[\"==\",[\"get\",\"kind\"],\"street\"]", street.getJSONArray("filter").toString())
        assertEquals("line", street.getJSONObject("layout").getString("symbol-placement"))
        assertEquals("[\"==\",[\"get\",\"kind\"],\"neighbourhood\"]", hood.getJSONArray("filter").toString())
        assertEquals("RahalGo Bold", hood.getJSONObject("layout").getJSONArray("text-font").getString(0))
        assertFalse(hood.getJSONObject("layout").has("symbol-sort-key"))
        // **والأصلُ لم يُمسّ** — ما زال يقرأ البلاطة.
        assertEquals("place", byId.getValue("place-neighbourhood").getString("source-layer"))
    }

    @Test
    fun areaNamesWithoutUrlAddNothing() {
        assertFalse(bound().getJSONObject("sources").has("rahalgo-areas"))
    }

    /** **والنمطُ الحقيقيّ فيه الطبقتان** — لو أُعيدت تسميتُهما سقط هذا لا الأسماءُ صامتةً. */
    @Test
    fun realStyleHasBothSourceLayers() {
        MapStores.areasUrl = "https://api.example/a"
        val raw = java.io.File("src/main/res/raw/rahalgo_style.json").readText(Charsets.UTF_8)
        val s = JSONObject(MapStyleBinding.online(raw, "https://maps.example/base.pmtiles", "https://maps.example/res", "").json)
        val ids = s.getJSONArray("layers").let { a -> (0 until a.length()).map { a.getJSONObject(it).getString("id") } }
        assertTrue("rahalgo-area-street" in ids)
        assertTrue("rahalgo-area-neighbourhood" in ids)
    }
}
