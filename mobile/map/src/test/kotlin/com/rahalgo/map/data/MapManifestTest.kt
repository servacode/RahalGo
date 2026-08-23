package com.rahalgo.map.data

import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **الفهرسُ ومخبؤه والحزمُ المعطوبة — البنود ٨ و٤٣ و٤٦**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك**: «ولا تثق بقيمة حجم أو path دون validation»،
 * و«اختبر: wrong sha · truncated file · wrong bytes · invalid
 * manifest · wrong schemaVersion · incompatible resourcesVersion.
 * كلها: لا Activate · old good package محفوظة · error surfaced ·
 * no Crash».
 */
class MapManifestTest {

    private lateinit var dir: File
    private lateinit var store: MapPackageStore
    private lateinit var client: MapManifestClient

    @Before
    fun setup() {
        dir = File(
            System.getProperty("java.io.tmpdir"),
            "rahalgo-mf-${System.nanoTime()}",
        ).also { it.mkdirs() }
        store = MapPackageStore(dir).also { it.prepare() }
        client = MapManifestClient(store, MapPaths.manifestCacheFile(dir))
    }

    @After
    fun teardown() {
        dir.deleteRecursively()
    }

    private fun manifestJson(
        schema: Int = 1,
        dataVersion: String = "2026-08-21",
        resourcesVersion: String = "1",
        regionId: String = "raqqa",
        bbox: String = "[38.92, 35.88, 39.12, 36.03]",
        baseBytes: Long = 310_000_000,
        attribution: String = "© مساهمو OpenStreetMap · OpenMapTiles",
        sha: String = "a".repeat(64),
    ) = """
    {
      "schemaVersion": $schema,
      "dataVersion": "$dataVersion",
      "tileSchema": "3.16.0",
      "resourcesVersion": "$resourcesVersion",
      "base": {
        "url": "base/$dataVersion/syria.pmtiles",
        "bytes": $baseBytes,
        "sha256": "$sha",
        "minZoom": 0, "maxZoom": 16,
        "bbox": [35.5, 32.0, 42.5, 37.5]
      },
      "regions": [{
        "id": "$regionId",
        "name": "الرقّة",
        "dataVersion": "$dataVersion",
        "url": "regions/$dataVersion/region-$regionId.pmtiles",
        "bytes": 110000000,
        "sha256": "${"b".repeat(64)}",
        "minZoom": 10, "maxZoom": 16,
        "bbox": $bbox
      }],
      "resources": {
        "url": "map-resources/$resourcesVersion/",
        "styleVersion": "1", "glyphVersion": "1", "spriteVersion": "1",
        "fontstacks": [
          {"name": "RahalGo Regular"}, {"name": "RahalGo Bold"}
        ]
      },
      "attribution": "$attribution"
    }
    """.trimIndent()

    // ── القراءةُ الصحيحة ────────────────────────────────────────────

    @Test
    fun `فهرسٌ سليمٌ يُقرأ كاملاً`() {
        val m = MapManifest.parse(manifestJson())
        assertEquals(1, m.schemaVersion)
        assertEquals("2026-08-21", m.dataVersion)
        assertEquals("1", m.resourcesVersion)
        assertEquals(1, m.regions.size)
        assertEquals("raqqa", m.regions[0].id)
        assertEquals("الرقّة", m.regions[0].name)
        assertEquals(listOf("RahalGo Regular", "RahalGo Bold"), m.resources.fontstacks)
        assertTrue(m.attribution.contains("OpenStreetMap"))
    }

    @Test
    fun `رصّاتُ الخطوط تُقرأ نصوصاً أيضاً`() {
        // **الشكلُ القديمُ من أوّل ٦أ** — فلا يسقط فهرسٌ لأنّ حقلاً
        // اغتنى في إغلاق الأدوات.
        val old = manifestJson().replace(
            """[
          {"name": "RahalGo Regular"}, {"name": "RahalGo Bold"}
        ]""",
            """["RahalGo Regular", "RahalGo Bold"]""",
        )
        val m = MapManifest.parse(old)
        assertEquals(listOf("RahalGo Regular", "RahalGo Bold"), m.resources.fontstacks)
    }

    // ── الرفض ──────────────────────────────────────────────────────

    private fun rejects(what: String, text: String) {
        var threw = false
        try {
            MapManifest.parse(text)
        } catch (e: Exception) {
            threw = true
        }
        assertTrue("«$what» يجب أن يُرفض", threw)
    }

    @Test
    fun `نسخةُ مخطّطٍ أحدثُ تُرفض لا تُحاوَل`() {
        rejects("schemaVersion=2", manifestJson(schema = 2))
        rejects("schemaVersion=0", manifestJson(schema = 0))
    }

    @Test
    fun `معرِّفُ منطقةٍ صاعدٌ يُرفض`() {
        rejects("regionId=../..", manifestJson(regionId = "../.."))
        rejects("regionId مطلق", manifestJson(regionId = "/etc"))
    }

    @Test
    fun `نسخةُ بياناتٍ غيرُ مقبولةٍ تُرفض`() {
        rejects("dataVersion=..", manifestJson(dataVersion = ".."))
    }

    @Test
    fun `حجمٌ غيرُ صحيحٍ يُرفض`() {
        rejects("bytes=0", manifestJson(baseBytes = 0))
        rejects("bytes سالب", manifestJson(baseBytes = -1))
    }

    @Test
    fun `بصمةٌ غيرُ صحيحةٍ تُرفض`() {
        rejects("sha قصيرة", manifestJson(sha = "abc"))
        rejects("sha ليست ستّ عشريّة", manifestJson(sha = "z".repeat(64)))
    }

    @Test
    fun `صندوقٌ مقلوبٌ أو خارجَ المدى يُرفض`() {
        rejects("مقلوب", manifestJson(bbox = "[39.12, 36.03, 38.92, 35.88]"))
        rejects("خطُّ عرضٍ خارجَ المدى", manifestJson(bbox = "[38.92, -95.0, 39.12, 36.03]"))
        rejects("ثلاثةُ أعداد", manifestJson(bbox = "[38.92, 35.88, 39.12]"))
    }

    @Test
    fun `فهرسٌ بلا إسنادٍ يُرفض`() {
        // **الإسنادُ شرطُ ترخيصٍ لا زينة** (البند ٢٧).
        rejects("لا attribution", manifestJson(attribution = ""))
    }

    @Test
    fun `نصٌّ ليس JSON يُرفض ولا يُسقط`() {
        rejects("نصٌّ عاديّ", "ليس فهرساً")
        rejects("فارغ", "{}")
    }

    // ── المخبأ ─────────────────────────────────────────────────────

    @Test
    fun `الفهرسُ الصالحُ يُخزَّن ويُقرأ`() {
        val r = client.accept(manifestJson())
        assertTrue("$r", r is MapManifestClient.Result.Fresh)
        assertNotNull(client.cached())
        assertEquals("2026-08-21", client.cached()!!.dataVersion)
    }

    @Test
    fun `فهرسٌ جديدٌ معطوبٌ لا يمحو صالحاً قديماً`() {
        // **البند ٤٣** — «إذا fetch جديد فشل: استخدم Last Known Good».
        client.accept(manifestJson(dataVersion = "2026-08-01"))
        val r = client.accept("{ معطوب }")
        assertTrue("$r", r is MapManifestClient.Result.Cached)
        assertEquals("2026-08-01", (r as MapManifestClient.Result.Cached).manifest.dataVersion)
        assertEquals("2026-08-01", client.cached()!!.dataVersion)
    }

    @Test
    fun `ولا مخبأَ يعني لا فهرس`() {
        val r = client.accept("{ معطوب }")
        assertTrue("$r", r is MapManifestClient.Result.None)
        assertNull(client.cached())
    }

    @Test
    fun `سقوطُ الشبكة يردّ آخرَ صالح`() {
        client.accept(manifestJson())
        val r = client.offline("لا اتّصال")
        assertTrue("$r", r is MapManifestClient.Result.Cached)
    }

    @Test
    fun `مخبأٌ فسد على القرص لا يُقرأ ولا يُسقط`() {
        client.accept(manifestJson())
        MapPaths.manifestCacheFile(dir).writeText("   ليس JSON")
        assertNull("والقراءةُ تحقُّقٌ لا نسخ", client.cached())
    }

    // ── الحزمُ المعطوبة ────────────────────────────────────────────

    @Test
    fun `حزمةٌ بلا عقدٍ ليست مركَّبة`() {
        val archive = MapPaths.regionArchive(dir, "raqqa", "2026-08-21")
        archive.parentFile.mkdirs()
        archive.writeBytes(ByteArray(1000))
        assertNull("العقدُ آخرُ ما يُكتب", store.installedRegion("raqqa", "2026-08-21"))
    }

    @Test
    fun `حزمةٌ اقتُطع أرشيفُها بعد التركيب لا تُقرأ`() {
        installRegion(bytes = 1000)
        assertNotNull(store.installedRegion("raqqa", "2026-08-21"))
        // **والحجمُ يُقاس لا يُصدَّق.**
        MapPaths.regionArchive(dir, "raqqa", "2026-08-21").writeBytes(ByteArray(400))
        assertNull(store.installedRegion("raqqa", "2026-08-21"))
    }

    @Test
    fun `عقدٌ معطوبٌ لا يُسقط التطبيق`() {
        installRegion(bytes = 1000)
        MapPaths.regionPackageFile(dir, "raqqa", "2026-08-21").writeText("{{{")
        assertNull(store.installedRegion("raqqa", "2026-08-21"))
        assertTrue(store.installedRegions().isEmpty())
    }

    @Test
    fun `مجلّدٌ بمعرِّفٍ غيرِ مقبولٍ يُتجاهل في المسح`() {
        installRegion(bytes = 1000)
        File(MapPaths.root(dir), "regions/..evil/2026-08-21").mkdirs()
        val all = store.installedRegions()
        assertEquals(1, all.size)
        assertEquals("raqqa", all[0].regionId)
    }

    @Test
    fun `النسخةُ القديمةُ تبقى مع الجديدة ثمّ تُنظَّف`() {
        installRegion(version = "2026-08-01", bytes = 1000)
        installRegion(version = "2026-08-21", bytes = 1000)
        assertEquals(2, store.installedRegions().size)

        assertTrue(store.deleteRegionVersion("raqqa", "2026-08-01"))
        val left = store.installedRegions()
        assertEquals(1, left.size)
        assertEquals("2026-08-21", left[0].dataVersion)
    }

    private fun installRegion(
        version: String = "2026-08-21",
        bytes: Int = 1000,
        resourcesVersion: String = "1",
    ) {
        val archive = MapPaths.regionArchive(dir, "raqqa", version)
        archive.parentFile.mkdirs()
        archive.writeBytes(ByteArray(bytes))
        store.markRegionInstalled(
            MapRegion(
                id = "raqqa",
                name = "الرقّة",
                dataVersion = version,
                artifact = MapArtifact(
                    url = "regions/$version/region-raqqa.pmtiles",
                    bytes = bytes.toLong(),
                    sha256 = "b".repeat(64),
                    minZoom = 10,
                    maxZoom = 16,
                    bbox = listOf(38.92, 35.88, 39.12, 36.03),
                ),
            ),
            resourcesVersion,
        )
    }
}
