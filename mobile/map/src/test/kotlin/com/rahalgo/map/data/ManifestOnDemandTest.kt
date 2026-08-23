package com.rahalgo.map.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لا خريطةَ بلا فهرس — وهذا يحرس أن يُقال ذلك لا أن يُصمَت عليه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ `TD-CUSTOMER-MAP-NO-MANIFEST`، شكوى المالك ٢٠٢٦-٠٨-٢٢:
 *  «الخريطة لاختيار عنوان لا تعمل بتطبيق الزبون».)
 *
 * # **ما وقع**
 *
 * **`MapStyleRepository.init` تقرأ الفهرسَ من ذاكرةٍ محلّيّةٍ وحدَها.**
 * وعلى تثبيتٍ جديدٍ لا ذاكرة — **فيعيد الربطُ `Unavailable`، وتخرج
 * `MapCanvas` صامتةً، وتبقى الشاشةُ بيضاء.**
 *
 * **وتطبيقُ السائق نجا بالصدفة** لا بالتصميم: فيه عاملٌ يجلب الفهرسَ
 * حين يضغط السائقُ «نزّل خريطة الرقة». **والزبونُ لا زرَّ له.**
 *
 * # **وما يُختبر هنا**
 *
 * **العقدُ لا الشبكة**: أنّ الربطَ بلا فهرسٍ يخرج **مصنَّفاً ومعلَّلاً**
 * لا فارغاً. **فالشاشةُ تقرأ السببَ لتقول للمستخدم**، وإخفاقٌ بلا سببٍ
 * إخفاقٌ صامت — وهو العطبُ بعينه.
 *
 * **وجلبُ الفهرس نفسُه يُختبر على الجهاز** — يحتاج شبكةً وسياقَ أندرويد.
 */
class ManifestOnDemandTest {

    private fun runtime(): MapRuntime {
        val dir = kotlin.io.path.createTempDirectory("map-test").toFile()
        val store = MapPackageStore(dir).also { it.prepare() }
        val canonical = """
            {
              "version": 8,
              "sources": { "base": { "type": "vector", "url": "{{TILES}}" } },
              "glyphs": "{{GLYPHS}}",
              "sprite": "{{SPRITE}}",
              "layers": []
            }
        """.trimIndent()
        return MapRuntime(store, canonical, MapConfig(baseUrl = "https://maps.rahalgo.com"))
    }

    /**
     * **بلا فهرسٍ لا يُبنى نمط — ويُقال لماذا.**
     *
     * **وهذا هو العطبُ الذي شكا منه المالك**: كان يخرج هكذا فعلاً،
     * **لكنّ الشاشةَ لم تكن تقرأ السبب.**
     */
    @Test
    fun `الربطُ بلا فهرسٍ يخرج معلَّلاً لا فارغاً`() {
        val out = runtime().bindOnline(null)
        assertTrue("توقّعتُ Unavailable فجاء $out", out is MapRuntime.Binding.Unavailable)
        val why = (out as MapRuntime.Binding.Unavailable).why
        assertNotNull("إخفاقٌ بلا سبب — والشاشةُ لا تجد ما تقول", why)
        assertTrue("سببٌ فارغ", why.isNotBlank())
    }

    /** **وبفهرسٍ صالحٍ يُبنى النمطُ ويحمل عناوينَ أصلنا.** */
    @Test
    fun `الربطُ بفهرسٍ يبني نمطاً من أصلنا`() {
        val m = MapManifest.parse(
            """
            {
              "schemaVersion": 1,
              "dataVersion": "2026-08-20",
              "tileSchema": "3.16.0",
              "resourcesVersion": "1",
              "base": {
                "url": "base/2026-08-20/syria.pmtiles",
                "bytes": 325356964,
                "sha256": "${"7".repeat(64)}",
                "minZoom": 0, "maxZoom": 16,
                "bbox": [35.27, 32.30, 42.38, 37.32]
              },
              "regions": [],
              "resources": {
                "url": "map-resources/1/",
                "styleVersion": "1", "glyphVersion": "1", "spriteVersion": "1",
                "fontstacks": ["RahalGo Regular"]
              },
              "attribution": "© OpenStreetMap"
            }
            """.trimIndent(),
        )
        val out = runtime().bindOnline(m)
        assertTrue("توقّعتُ Ready فجاء $out", out is MapRuntime.Binding.Ready)
        val json = (out as MapRuntime.Binding.Ready).bound.json
        assertTrue(
            "النمطُ لا يشير إلى أصلنا",
            json.contains("maps.rahalgo.com/base/2026-08-20/syria.pmtiles"),
        )
        assertTrue("لا حروفَ من أصلنا", json.contains("maps.rahalgo.com/map-resources/1"))
        // **ولا مصدرَ عموميٌّ يتسلّل** — الحارسُ نفسُه في كلّ طبقة.
        for (bad in listOf("tile.openstreetmap.org", "basemaps.cartocdn.com", "api.maptiler.com")) {
            assertTrue("تسلّل مصدرٌ عموميّ: $bad", !json.contains(bad))
        }
    }

    /** **واسمُ ملفّ الفهرس عقدٌ** — يتبدّل هنا فيتبدّل عند الجميع. */
    @Test
    fun `اسمُ ملفّ الفهرس واحدٌ للجميع`() {
        assertEquals("manifest.json", com.rahalgo.map.MapStyleRepository.MANIFEST_FILE)
    }
}
