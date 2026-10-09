package com.rahalgo.map.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حارسُ حزمة المدينة أونلاين — وألّا تبيضّ الخريطةُ أبداً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-١٠-٠٩: أوّلُ فتحةٍ للخريطة بطيئة.)
 *
 * **داخلَ صندوق الرقّة تُقرأ حزمتُها** — فهرسُها صغيرٌ فتقلّ الطلبات.
 * **وخارجَه، أو قبل أن يثبت وجودُها على الخادم، أو إن غابت (404)،
 * يُقرأ أساسُ سوريا كما كان.**
 */
class OnlineRegionTest {

    /** **الفهرسُ المقترح** — الأساسُ والرقّةُ كما في `manifest.proposed.json`. */
    private fun manifest(regionMinZoom: Int = 0, regionMaxZoom: Int = 16) = MapManifest.parse(
        """
        {
          "schemaVersion": 1,
          "dataVersion": "2026-08-24",
          "tileSchema": "3.16.0",
          "resourcesVersion": "1",
          "base": {
            "url": "base/2026-08-24/syria.pmtiles",
            "bytes": 502157918,
            "sha256": "${"a".repeat(64)}",
            "minZoom": 0, "maxZoom": 16,
            "bbox": [35.277097, 32.3066, 42.38383, 37.322394]
          },
          "regions": [
            {
              "id": "syria", "name": "سوريا", "dataVersion": "2026-08-24",
              "url": "base/2026-08-24/syria.pmtiles",
              "bytes": 502157918, "sha256": "${"a".repeat(64)}",
              "minZoom": 0, "maxZoom": 16,
              "bbox": [35.277097, 32.3066, 42.38383, 37.322394]
            },
            {
              "id": "raqqa", "name": "الرقّة", "dataVersion": "2026-08-24",
              "url": "regions/2026-08-24/region-raqqa.pmtiles",
              "bytes": 1000, "sha256": "${"b".repeat(64)}",
              "minZoom": $regionMinZoom, "maxZoom": $regionMaxZoom,
              "bbox": [38.85, 35.88, 39.20, 36.02]
            }
          ],
          "resources": {
            "url": "map-resources/1/",
            "styleVersion": "1", "glyphVersion": "1", "spriteVersion": "1",
            "fontstacks": ["RahalGo Regular"]
          },
          "attribution": "© OpenStreetMap"
        }
        """.trimIndent(),
    )

    private val raqqaUrl = "regions/2026-08-24/region-raqqa.pmtiles"

    // مركزُ الرقّة، وحلب خارجَ صندوقها
    private val cityLat = 35.9528
    private val cityLng = 39.0079
    private val aleppoLat = 36.2021
    private val aleppoLng = 37.1343

    private fun runtime(): MapRuntime {
        val dir = kotlin.io.path.createTempDirectory("map-online-region").toFile()
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

    private fun tilesOf(out: MapRuntime.Binding): String {
        assertTrue("توقّعتُ Ready فجاء $out", out is MapRuntime.Binding.Ready)
        return (out as MapRuntime.Binding.Ready).bound.json
    }

    // ── الاختيار ──────────────────────────────────────────────────────

    @Test
    fun `داخلَ صندوق الرقّة تُختار حزمتُها لا سوريا`() {
        assertEquals("raqqa", RegionPicker.forOnline(manifest(), cityLat, cityLng)?.id)
    }

    @Test
    fun `خارجَ كلّ صندوقٍ لا حزمة — فيُقرأ الأساس`() {
        assertNull(RegionPicker.forOnline(manifest(), aleppoLat, aleppoLng))
    }

    @Test
    fun `بلا موضعٍ لا حزمة`() {
        assertNull(RegionPicker.forOnline(manifest(), null, null))
    }

    @Test
    fun `مسارٌ يخرج من الصندوق يُبقي الأساس`() {
        val outward = listOf(38.95, 35.90, 39.60, 36.00) // إلى الشرق خارجَ الصندوق
        assertNull(RegionPicker.forOnline(manifest(), cityLat, cityLng, outward))
        val inside = listOf(38.95, 35.90, 39.10, 36.00)
        assertEquals("raqqa", RegionPicker.forOnline(manifest(), cityLat, cityLng, inside)?.id)
    }

    @Test
    fun `حزمةٌ أضيقُ تقريباً من الأساس لا تُختار`() {
        // **وإلّا ابيضّت الخريطةُ عند تقريبٍ يعرفه الأساس.**
        assertNull(RegionPicker.forOnline(manifest(regionMinZoom = 10), cityLat, cityLng))
        assertNull(RegionPicker.forOnline(manifest(regionMaxZoom = 14), cityLat, cityLng))
    }

    @Test
    fun `حزمةٌ حُكم بأنّها غيرُ صالحةٍ لا تُختار`() {
        assertNull(RegionPicker.forOnline(manifest(), cityLat, cityLng) { false })
    }

    // ── حكمُ الفحص ────────────────────────────────────────────────────

    @Test
    fun `حكمُ الفحص من رمز الردّ وتوقيع الأرشيف`() {
        val good = "PMTiles\u0003rest".toByteArray(Charsets.US_ASCII)
        assertEquals(true, RegionPicker.verdictOf(206, good))
        assertEquals(true, RegionPicker.verdictOf(200, good))
        assertEquals(false, RegionPicker.verdictOf(404, null))
        assertEquals(false, RegionPicker.verdictOf(410, null))
        // **صفحةُ خطأٍ بـ200 ليست أرشيفاً**
        assertEquals(false, RegionPicker.verdictOf(200, "<html>".toByteArray()))
        assertEquals(false, RegionPicker.verdictOf(206, ByteArray(0)))
        // **وعطلٌ عابرٌ لا حكمَ فيه** — يُعاد الفحصُ لاحقاً
        assertNull(RegionPicker.verdictOf(503, null))
        assertNull(RegionPicker.verdictOf(-1, null))
    }

    // ── الربط ─────────────────────────────────────────────────────────

    @Test
    fun `قبل الفحص يُقرأ الأساس — ولا تبيضّ الخريطة`() {
        val json = tilesOf(runtime().bindOnline(manifest(), cityLat, cityLng))
        assertTrue(json.contains("base/2026-08-24/syria.pmtiles"))
        assertFalse(json.contains(raqqaUrl))
    }

    @Test
    fun `بعد ثبوت الحزمة على الخادم تُقرأ هي في الرقّة`() {
        val rt = runtime()
        assertEquals(listOf("raqqa"), rt.regionsToVerify(manifest()).map { it.first.id })
        assertEquals(
            "https://maps.rahalgo.com/regions/2026-08-24/region-raqqa.pmtiles",
            rt.regionsToVerify(manifest()).single().second,
        )
        rt.markRegion(raqqaUrl, true)
        assertTrue(rt.regionsToVerify(manifest()).isEmpty())

        val out = rt.bindOnline(manifest(), cityLat, cityLng)
        val json = tilesOf(out)
        assertTrue("لم تُقرأ حزمةُ الرقّة", json.contains("maps.rahalgo.com/$raqqaUrl"))
        assertFalse(json.contains("syria.pmtiles"))
        val why = ((out as MapRuntime.Binding.Ready).decision as MapSourceResolver.Decision.Online).why
        assertTrue(why.contains("raqqa"))
    }

    @Test
    fun `حزمةٌ غائبةٌ عن الخادم (404) يُقرأ بدلها الأساس`() {
        val rt = runtime()
        rt.markRegion(raqqaUrl, RegionPicker.verdictOf(404, null)!!)
        val json = tilesOf(rt.bindOnline(manifest(), cityLat, cityLng))
        assertTrue(json.contains("base/2026-08-24/syria.pmtiles"))
        assertFalse(json.contains(raqqaUrl))
    }

    @Test
    fun `خارجَ الرقّة يُقرأ الأساس ولو ثبتت حزمتُها`() {
        val rt = runtime()
        rt.markRegion(raqqaUrl, true)
        val json = tilesOf(rt.bindOnline(manifest(), aleppoLat, aleppoLng))
        assertTrue(json.contains("base/2026-08-24/syria.pmtiles"))
    }

    @Test
    fun `المحلِّلُ يمرّر الموضعَ والمسارَ إلى الربط`() {
        val rt = runtime()
        rt.markRegion(raqqaUrl, true)
        val request = MapSourceResolver.Request(
            purpose = MapSourceResolver.Purpose.PICK_POINT,
            online = true,
            lat = cityLat,
            lng = cityLng,
            manifest = manifest(),
        )
        val json = tilesOf(rt.resolve(request, navigating = false, nowMs = 0L))
        assertTrue(json.contains(raqqaUrl))
    }
}
