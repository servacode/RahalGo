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
}
