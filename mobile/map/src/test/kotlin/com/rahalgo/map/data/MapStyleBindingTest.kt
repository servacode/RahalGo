package com.rahalgo.map.data

import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.json.JSONObject
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **ربطُ النمط والحروفُ والأيقونات — البنود ٢٣ إلى ٢٧ و٤٠**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك**: «اختبر تعاقديًا: requested fontstack · requested
 * range → installed path exists. لا أريد runtime 404 محلي صامت».
 *
 * **والنمطُ المقروءُ هنا هو المرجعيُّ نفسُه** — يُقرأ من
 * `maps/style/rahalgo.style.json` **لا نسخةً مكتوبةً في اختبار**،
 * **فلو انحرف انكسر الاختبار.**
 */
class MapStyleBindingTest {

    private lateinit var dir: File
    private lateinit var store: MapPackageStore

    /**
     * **النمطُ من مصدره** — والمسارُ يصعد من مجلّد الوحدة إلى المستودع.
     */
    private val canonical: String by lazy {
        val here = File("").absoluteFile
        val candidates = listOf(
            File(here, "../maps/style/rahalgo.style.json"),
            File(here, "../../maps/style/rahalgo.style.json"),
            File(here, "src/main/res/raw/rahalgo_style.json"),
            File(here, "map/src/main/res/raw/rahalgo_style.json"),
        )
        candidates.firstOrNull { it.isFile }?.readText()
            ?: error("لم يُوجد النمطُ المرجعيُّ — جُرّب: ${candidates.map { it.absolutePath }}")
    }

    @Before
    fun setup() {
        dir = File(
            System.getProperty("java.io.tmpdir"),
            "rahalgo-style-${System.nanoTime()}",
        ).also { it.mkdirs() }
        store = MapPackageStore(dir).also { it.prepare() }
    }

    @After
    fun teardown() {
        dir.deleteRecursively()
    }

    // ── الربطُ لا يمسّ إلّا ثلاثةَ عناوين ──────────────────────────

    @Test
    fun `الربطُ لا يبدّل الطبقاتِ ولا الرسمَ ولا المصدر-الطبقة`() {
        // **البند ٢٣** — «أما layers · paint · layout · source-layer
        // فتبقى canonical».
        val online = MapStyleBinding.online(
            canonical,
            "https://maps.rahalgo.com/base/2026-08-21/syria.pmtiles",
            "https://maps.rahalgo.com/map-resources/1",
            "© مساهمو OpenStreetMap",
        )
        val offline = MapStyleBinding.offline(
            canonical,
            File(dir, "region.pmtiles").absoluteFile,
            File(dir, "resources").absoluteFile,
            "© مساهمو OpenStreetMap",
        )
        assertEquals(shape(online.json), shape(offline.json))
        assertEquals(shape(canonical), shape(online.json))
    }

    /** **الشكلُ الدلاليّ** — كلُّ شيءٍ إلّا العناوين. */
    private fun shape(json: String): String {
        val o = JSONObject(json)
        val layers = o.optJSONArray("layers")!!
        return (0 until layers.length()).joinToString("|") { i ->
            val l = layers.getJSONObject(i)
            listOf(
                l.optString("id"),
                l.optString("type"),
                l.optString("source"),
                l.optString("source-layer"),
                l.opt("filter")?.toString() ?: "",
                l.opt("layout")?.toString() ?: "",
                l.opt("paint")?.toString() ?: "",
            ).joinToString(",")
        }
    }

    @Test
    fun `الأونلاين يبني pmtiles فوق https`() {
        val b = MapStyleBinding.online(
            canonical,
            "https://maps.rahalgo.com/base/2026-08-21/syria.pmtiles",
            "https://maps.rahalgo.com/map-resources/1",
            "إسناد",
        )
        assertEquals(
            "pmtiles://https://maps.rahalgo.com/base/2026-08-21/syria.pmtiles",
            b.tileUri,
        )
        assertEquals(
            "https://maps.rahalgo.com/map-resources/1/glyphs/{fontstack}/{range}.pbf",
            b.glyphsUri,
        )
        assertEquals("https://maps.rahalgo.com/map-resources/1/sprite", b.spriteUri)
        assertEquals(MapStyleBinding.BINDING_ONLINE, b.binding)
    }

    @Test
    fun `الأوفلاين يبني pmtiles فوق file`() {
        val archive = File(dir, "maps/regions/raqqa/2026-08-21/region.pmtiles").absoluteFile
        val resources = File(dir, "maps/resources/1").absoluteFile
        val b = MapStyleBinding.offline(canonical, archive, resources, "إسناد")

        assertTrue(b.tileUri.startsWith("pmtiles://file:/"))
        assertTrue(b.glyphsUri.endsWith("/glyphs/{fontstack}/{range}.pbf"))
        assertTrue(b.spriteUri.endsWith("/sprite"))
        assertEquals(MapStyleBinding.BINDING_OFFLINE, b.binding)
        assertFalse("لا مسافةَ في عنوان", b.glyphsUri.contains(' '))
    }

    @Test
    fun `الإسنادُ يُحقن في المصدر فيراه المستخدم`() {
        // **البند ٢٧** — «المستخدم يجب أن يستطيع رؤيته».
        val b = MapStyleBinding.online(
            canonical, "https://a/x.pmtiles", "https://a/r", "© مساهمو OpenStreetMap · OpenMapTiles",
        )
        val src = JSONObject(b.json).getJSONObject("sources").getJSONObject("base")
        assertEquals("© مساهمو OpenStreetMap · OpenMapTiles", src.getString("attribution"))
    }

    // ── الرصّاتُ والمراسي ─────────────────────────────────────────

    @Test
    fun `الرصّاتُ تُقرأ من النمط لا من الإعداد`() {
        val stacks = MapStyleBinding.fontstacksOf(canonical)
        assertEquals(setOf("RahalGo Regular", "RahalGo Bold"), stacks.toSet())
    }

    @Test
    fun `المراسي معرَّفةٌ في النمط`() {
        // **البند ٤٠** — «إذا Anchor غير موجود بسبب Style invalid:
        // Fail fast في QA».
        val anchors = MapStyleBinding.anchors(canonical)
        val ids = MapStyleBinding.layerIds(canonical).toSet()
        assertTrue("لا مراسيَ في النمط", anchors.isNotEmpty())
        for ((role, anchor) in anchors) {
            assertTrue("المِرساةُ $role تشير إلى $anchor وليست طبقةً", ids.contains(anchor))
        }
        assertTrue(anchors.containsValue(MapStyleBinding.ANCHOR_ROUTE))
        assertTrue(anchors.containsValue(MapStyleBinding.ANCHOR_MARKERS))
    }

    @Test
    fun `مِرساةُ المسار قبل مِرساةِ الدبابيس`() {
        // **فالمسارُ تحتَ الدبابيس** — وإلّا مرَّ الخطُّ فوق دبّوس
        // التسليم فأخفاه.
        val ids = MapStyleBinding.layerIds(canonical)
        val route = ids.indexOf(MapStyleBinding.ANCHOR_ROUTE)
        val markers = ids.indexOf(MapStyleBinding.ANCHOR_MARKERS)
        assertTrue("مِرساةُ المسار غائبة", route >= 0)
        assertTrue("مِرساةُ الدبابيس غائبة", markers >= 0)
        assertTrue("الترتيبُ معكوس", route < markers)
    }

    @Test
    fun `الأسماءُ عربيّةٌ أوّلاً`() {
        // **البند ٢٤** — `name:ar` ثمّ `name` ثمّ لا اسم.
        assertTrue(MapStyleBinding.labelExpressionsAreArabicFirst(canonical))
    }

    // ── عقدُ الملفّات المحلّيّة ────────────────────────────────────

    @Test
    fun `كلُّ رصّةٍ في كلّ نطاقٍ لازمٍ يجب أن يوجد ملفُّها`() {
        // **البند ٢٥** — «لا أريد runtime 404 محلي صامت».
        val stacks = MapStyleBinding.fontstacksOf(canonical)
        val missing = store.missingGlyphs("1", stacks)
        assertEquals(
            "بلا تركيبٍ يجب أن ينقص كلُّ شيء",
            stacks.size * MapGlyphRanges.REQUIRED.size,
            missing.size,
        )

        installResources(stacks)
        assertTrue("بعد التركيب لا نقص", store.missingGlyphs("1", stacks).isEmpty())
    }

    @Test
    fun `نطاقٌ فارغٌ كالمفقود`() {
        val stacks = MapStyleBinding.fontstacksOf(canonical)
        installResources(stacks)
        MapPaths.glyphFile(dir, "1", stacks[0], "64768-65023").writeBytes(ByteArray(0))
        val missing = store.missingGlyphs("1", stacks)
        assertEquals(1, missing.size)
        assertTrue(missing[0].contains("64768-65023"))
    }

    @Test
    fun `الأيقوناتُ أربعةُ ملفّاتٍ لا اثنان`() {
        // **البند ٢٦** — 1x و2x فهرساً وصورة.
        assertEquals(4, store.missingSprites("1").size)
        installResources(MapStyleBinding.fontstacksOf(canonical))
        assertTrue(store.missingSprites("1").isEmpty())
    }

    @Test
    fun `مواردُ ناقصةٌ تمنع الربطَ دونَ اتّصال`() {
        // **البند ١٣** — «ولا تستخدم Region مع Style/Glyphs غير متوافقة».
        val runtime = MapRuntime(
            store,
            canonical,
            MapConfig("https://maps.rahalgo.com"),
        )
        val region = MapPackageStore.Installed(
            "raqqa", "2026-08-21", "1", "الرقّة",
            listOf(38.92, 35.88, 39.12, 36.03), 10, 16, 1000, "a".repeat(64),
            File(dir, "region.pmtiles").absoluteFile.also { it.writeBytes(ByteArray(10)) },
        )
        val before = runtime.bindOffline(region)
        assertTrue("$before", before is MapRuntime.Binding.Unavailable)

        installResources(MapStyleBinding.fontstacksOf(canonical))
        val after = runtime.bindOffline(region)
        assertTrue("$after", after is MapRuntime.Binding.Ready)
    }

    @Test
    fun `آخرُ ربطٍ صالحٍ هو البديلُ لا راستر عامّ`() {
        // **البند ٢** — «الـfallback الحقيقي: last known valid vector
        // configuration/resources».
        val runtime = MapRuntime(store, canonical, MapConfig("https://maps.rahalgo.com"))
        installResources(MapStyleBinding.fontstacksOf(canonical))
        val region = MapPackageStore.Installed(
            "raqqa", "2026-08-21", "1", "الرقّة",
            listOf(38.92, 35.88, 39.12, 36.03), 10, 16, 1000, "a".repeat(64),
            File(dir, "region.pmtiles").absoluteFile.also { it.writeBytes(ByteArray(10)) },
        )
        assertTrue(runtime.bindOffline(region) is MapRuntime.Binding.Ready)

        // **ثمّ يُطلب أونلاين بلا فهرس** — فيُردّ آخرُ صالحٍ لا انهيار.
        val out = runtime.bindOnline(null)
        assertTrue("$out", out is MapRuntime.Binding.Ready)
        val json = (out as MapRuntime.Binding.Ready).bound.json
        assertFalse("ولا راستر OSM بحال", json.contains("openstreetmap.org"))
    }

    private fun installResources(stacks: List<String>) {
        for (s in stacks) {
            for (r in MapGlyphRanges.REQUIRED) {
                val f = MapPaths.glyphFile(dir, "1", s, r)
                f.parentFile.mkdirs()
                f.writeBytes(ByteArray(64))
            }
        }
        for (s in MapSpriteFiles.ALL) {
            val f = MapPaths.spriteFile(dir, "1", s)
            f.parentFile.mkdirs()
            f.writeBytes(ByteArray(16))
        }
        store.markResourcesInstalled(
            MapResources("1", "map-resources/1/", "1", "1", "1", stacks),
        )
    }
}
