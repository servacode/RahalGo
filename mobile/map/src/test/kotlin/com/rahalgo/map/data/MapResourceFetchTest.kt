package com.rahalgo.map.data

import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.json.JSONArray
import org.json.JSONObject
import java.io.ByteArrayInputStream
import java.io.File
import java.io.InputStream

/**
 * ══════════════════════════════════════════════════════════════════
 * **جلبُ الموارد وتركيبُها — البنود ١ إلى ٧ و١٦ إلى ١٩**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك**: «اختبر: glyph missing · glyph zero bytes · sprite
 * missing · sprite JSON without PNG · 2x missing · style missing ·
 * wrong sha · wrong size · wrong fontstack · missing required range.
 * كلها: resources version NOT ACTIVE · old resources preserved · map
 * does not switch to incompatible region · no crash».
 */
class MapResourceFetchTest {

    private lateinit var dir: File
    private lateinit var store: MapPackageStore

    private val stacks = listOf("RahalGo Regular", "RahalGo Bold")

    @Before
    fun setup() {
        dir = File(
            System.getProperty("java.io.tmpdir"),
            "rahalgo-rf-${System.nanoTime()}",
        ).also { it.mkdirs() }
        store = MapPackageStore(dir).also { it.prepare() }
    }

    @After
    fun teardown() {
        dir.deleteRecursively()
    }

    // ══════════════════════════════════════════════════════════════
    // **خادمٌ يخدم حزمةَ مواردَ كاملة**
    // ══════════════════════════════════════════════════════════════

    /** **محتوى ملفٍّ مشتقٌّ من مساره** — فبصمتُه ثابتةٌ ومتوقَّعة. */
    private fun bodyOf(path: String): ByteArray =
        ("محتوى:$path").toByteArray(Charsets.UTF_8) + ByteArray(32)

    private fun sha(bytes: ByteArray): String {
        val f = File(dir, "probe-${System.nanoTime()}.bin")
        f.writeBytes(bytes)
        return MapDownloader.sha256(f).also { f.delete() }
    }

    private fun resourcePaths(): List<String> = buildList {
        for (s in stacks) for (r in MapGlyphRanges.REQUIRED) add("glyphs/$s/$r.pbf")
        addAll(MapSpriteFiles.ALL)
        add(MapPaths.STYLE_FILE)
        add(MapResourceContract.LICENSE_PATH)
    }

    private fun contractJson(
        version: String = "1",
        paths: List<String> = resourcePaths(),
        fontstacks: List<String> = stacks,
        corruptSha: String? = null,
        wrongSize: String? = null,
        extraPath: String? = null,
    ): String {
        val files = JSONArray()
        for (p in paths) {
            val body = bodyOf(p)
            files.put(
                JSONObject().apply {
                    put("path", p)
                    put("bytes", if (p == wrongSize) body.size + 99L else body.size.toLong())
                    put("sha256", if (p == corruptSha) "c".repeat(64) else sha(body))
                },
            )
        }
        extraPath?.let {
            files.put(
                JSONObject().apply {
                    put("path", it); put("bytes", 10); put("sha256", "a".repeat(64))
                },
            )
        }
        return JSONObject().apply {
            put("resourcesVersion", version)
            put("styleVersion", "1")
            put("glyphVersion", "1")
            put("spriteVersion", "1")
            put("glyphUrlTemplate", "{fontstack}/{range}.pbf")
            put("fontstacks", JSONArray(fontstacks.map { JSONObject().put("name", it) }))
            put("fontLicense", JSONObject().put("spdx", "OFL-1.1"))
            put("files", files)
        }.toString()
    }

    /** **خادمٌ يردّ ما يُملى عليه** — والحذفُ يُملى أيضاً. */
    private inner class Server(
        val contract: String,
        /** **ملفّاتٌ يمتنع عن خدمتها** — لمحاكاة الناقص. */
        val missing: Set<String> = emptySet(),
        /** **وملفّاتٌ يخدمها فارغةً.** */
        val empty: Set<String> = emptySet(),
        val failContract: Boolean = false,
    ) : HttpSource {
        var requests = 0

        override fun open(url: String, rangeStart: Long?): HttpSource.Response {
            requests += 1
            /**
             * **والمسارُ يُفكُّ ترميزُه** — الجالبُ يرمّز المسافةَ في
             * `RahalGo Regular` إلى `%20` كما يجب، **فالخادمُ الحقيقيُّ
             * يفكّها ليجد الملفّ.** ولو قُرئ مرمَّزاً هنا **لاختلف
             * المحتوى عن الذي حُسبت بصمتُه.**
             */
            val path = java.net.URI(url).path
            val rel = path.substringAfter("/map-resources/1/", path.substringAfterLast('/'))
            if (rel == MapResourceFetcher.CONTRACT_FILE) {
                if (failContract) throw HttpSource.HttpException("سقطت الشبكة")
                return respond(contract.toByteArray(Charsets.UTF_8))
            }
            if (rel in missing) throw HttpSource.HttpException("الخادمُ ردَّ 404")
            if (rel in empty) return respond(ByteArray(0))
            return respond(bodyOf(rel))
        }

        private fun respond(body: ByteArray) = object : HttpSource.Response {
            override val status: Int = 200
            override val rangeStart: Long = 0
            override val remainingBytes: Long = body.size.toLong()
            override val body: InputStream = ByteArrayInputStream(body)
            override fun close() = Unit
        }
    }

    private fun manifest(resourcesVersion: String = "1") = MapManifest(
        schemaVersion = 1,
        dataVersion = "2026-08-21",
        tileSchema = "3.16.0",
        resourcesVersion = resourcesVersion,
        base = MapArtifact(
            "base/2026-08-21/syria.pmtiles", 1000, "a".repeat(64), 0, 16,
            listOf(35.5, 32.0, 42.5, 37.5),
        ),
        regions = emptyList(),
        resources = MapResources(
            resourcesVersion, "map-resources/$resourcesVersion/", "1", "1", "1", stacks,
        ),
        attribution = "© مساهمو OpenStreetMap",
    )

    private fun fetcher(http: HttpSource) = MapResourceFetcher(
        store,
        MapDownloader(store, http, MapDownloader.Tuning(safetyMarginBytes = 0, tempOverheadBytes = 0)),
        http,
        MapConfig("https://maps.rahalgo.com"),
    )

    // ══════════════════════════════════════════════════════════════
    // **المسارُ السعيد**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `جهازٌ فارغٌ يجلب الموارد كاملةً ويُفعّلها`() {
        val out = fetcher(Server(contractJson())).ensure(manifest(), stacks)
        assertTrue("$out", out is MapResourceFetcher.Outcome.Installed)
        assertEquals(18, (out as MapResourceFetcher.Outcome.Installed).files)

        assertTrue("النسخةُ كاملة", store.resourcesComplete("1", stacks))
        assertNotNull(store.installedResources("1"))
        assertTrue(MapPaths.styleFile(dir, "1").isFile)
        assertTrue(store.missingGlyphs("1", stacks).isEmpty())
        assertTrue(store.missingSprites("1").isEmpty())
    }

    @Test
    fun `المركَّبةُ لا تُجلب ثانيةً`() {
        val server = Server(contractJson())
        fetcher(server).ensure(manifest(), stacks)
        val after = server.requests

        val out = fetcher(server).ensure(manifest(), stacks)
        assertTrue("$out", out is MapResourceFetcher.Outcome.AlreadyInstalled)
        assertEquals("ولا نداءَ شبكةٍ واحد", after, server.requests)
    }

    @Test
    fun `لا يبقى مجلّدٌ مؤقّتٌ بعد التفعيل`() {
        fetcher(Server(contractJson())).ensure(manifest(), stacks)
        assertFalse(MapPaths.resourcesStagingDir(dir, "1").exists())
    }

    // ══════════════════════════════════════════════════════════════
    // **لا مواردَ ناقصة** — البند ٤
    // ══════════════════════════════════════════════════════════════

    private fun assertNotActivated(out: MapResourceFetcher.Outcome, what: String) {
        assertTrue("$what: $out", out is MapResourceFetcher.Outcome.Failed)
        assertFalse("$what: النسخةُ فُعّلت وهي ناقصة", store.resourcesComplete("1", stacks))
        assertNull("$what: عُلّمت مركَّبةً", store.installedResources("1"))
        assertFalse("$what: بقي مجلّدٌ مؤقّت", MapPaths.resourcesStagingDir(dir, "1").exists())
    }

    @Test
    fun `ملفُّ حرفٍ مفقودٌ يمنع التفعيل`() {
        val path = "glyphs/RahalGo Regular/64768-65023.pbf"
        assertNotActivated(
            fetcher(Server(contractJson(), missing = setOf(path))).ensure(manifest(), stacks),
            "حرفٌ مفقود",
        )
    }

    @Test
    fun `ملفُّ حرفٍ فارغٌ يمنع التفعيل`() {
        // **والفارغُ يردّ ٢٠٠ ويرسم مربّعات** — فهو كالمفقود.
        val path = "glyphs/RahalGo Bold/1536-1791.pbf"
        assertNotActivated(
            fetcher(Server(contractJson(), empty = setOf(path))).ensure(manifest(), stacks),
            "حرفٌ فارغ",
        )
    }

    @Test
    fun `نطاقٌ لازمٌ غائبٌ من العقد يمنع التفعيل`() {
        val without = resourcePaths().filterNot { it.endsWith("64512-64767.pbf") }
        assertNotActivated(
            fetcher(Server(contractJson(paths = without))).ensure(manifest(), stacks),
            "نطاقٌ ناقص",
        )
    }

    @Test
    fun `أيقونةٌ مفقودةٌ تمنع التفعيل`() {
        assertNotActivated(
            fetcher(Server(contractJson(), missing = setOf("sprite.png")))
                .ensure(manifest(), stacks),
            "أيقونةٌ مفقودة",
        )
    }

    @Test
    fun `فهرسُ أيقوناتٍ بلا صورته يمنع التفعيل`() {
        // **وهو عطبٌ يبدو سليماً**: MapLibre تقرأ الفهرسَ فتجد
        // الأيقونةَ معلنةً ثمّ لا تجد صورتَها.
        val without = resourcePaths().filterNot { it == "sprite.png" }
        assertNotActivated(
            fetcher(Server(contractJson(paths = without))).ensure(manifest(), stacks),
            "فهرسٌ بلا صورة",
        )
    }

    @Test
    fun `غيابُ 2x يمنع التفعيل`() {
        val without = resourcePaths().filterNot { it.startsWith("sprite@2x") }
        assertNotActivated(
            fetcher(Server(contractJson(paths = without))).ensure(manifest(), stacks),
            "لا 2x",
        )
    }

    @Test
    fun `غيابُ النمط يمنع التفعيل`() {
        val without = resourcePaths().filterNot { it == MapPaths.STYLE_FILE }
        assertNotActivated(
            fetcher(Server(contractJson(paths = without))).ensure(manifest(), stacks),
            "لا نمط",
        )
    }

    @Test
    fun `بصمةٌ مخالفةٌ تمنع التفعيل`() {
        assertNotActivated(
            fetcher(Server(contractJson(corruptSha = "sprite.json"))).ensure(manifest(), stacks),
            "بصمةٌ مخالفة",
        )
    }

    @Test
    fun `حجمٌ مخالفٌ يمنع التفعيل`() {
        assertNotActivated(
            fetcher(Server(contractJson(wrongSize = MapPaths.STYLE_FILE)))
                .ensure(manifest(), stacks),
            "حجمٌ مخالف",
        )
    }

    @Test
    fun `رصّةٌ يطلبها النمطُ وليست في العقد تمنع التفعيل`() {
        val out = fetcher(Server(contractJson(fontstacks = listOf("RahalGo Regular"))))
            .ensure(manifest(), stacks)
        assertTrue("$out", out is MapResourceFetcher.Outcome.Failed)
        assertEquals(
            MapFailure.INVALID_CONTRACT,
            (out as MapResourceFetcher.Outcome.Failed).reason,
        )
    }

    @Test
    fun `النسخةُ القديمةُ تبقى إن سقط الجلب`() {
        // **البند ٤** — «old resources preserved».
        fetcher(Server(contractJson())).ensure(manifest(), stacks)
        assertTrue(store.resourcesComplete("1", stacks))
        val styleBefore = MapPaths.styleFile(dir, "1").readBytes()

        val v2 = contractJson(version = "2")
        val out = fetcher(Server(v2, missing = setOf("sprite@2x.png")))
            .ensure(manifest("2"), stacks)

        assertTrue("$out", out is MapResourceFetcher.Outcome.Failed)
        assertFalse("الجديدةُ لم تُفعَّل", store.resourcesComplete("2", stacks))
        assertTrue("والقديمةُ باقية", store.resourcesComplete("1", stacks))
        assertTrue(MapPaths.styleFile(dir, "1").readBytes().contentEquals(styleBefore))
    }

    @Test
    fun `عقدٌ لا يُفهم لا يُعاد جلبُه`() {
        val out = fetcher(Server("{ معطوب }")).ensure(manifest(), stacks)
        assertTrue("$out", out is MapResourceFetcher.Outcome.Failed)
        val f = (out as MapResourceFetcher.Outcome.Failed).reason
        assertEquals(MapFailure.INVALID_CONTRACT, f)
        assertFalse("ولا يُعاد", f.transient)
    }

    @Test
    fun `سقوطُ الشبكة يُصنَّف عابراً`() {
        val out = fetcher(Server(contractJson(), failContract = true)).ensure(manifest(), stacks)
        assertTrue("$out", out is MapResourceFetcher.Outcome.Failed)
        val f = (out as MapResourceFetcher.Outcome.Failed).reason
        assertEquals(MapFailure.NETWORK, f)
        assertTrue("ويُعاد", f.transient)
    }

    @Test
    fun `الإلغاءُ لا يترك نسخةً مفعَّلة`() {
        val out = fetcher(Server(contractJson()))
            .ensure(manifest(), stacks, cancellation = { true })
        assertTrue("$out", out is MapResourceFetcher.Outcome.Cancelled)
        assertFalse(store.resourcesComplete("1", stacks))
    }

    @Test
    fun `محاولةٌ ثانيةٌ تُكمل ولا تبدأ`() {
        // **البند ٦** — الأبسطُ الصحيح: ما نُزّل وطابق بصمتَه لا يُعاد.
        val first = Server(contractJson(), missing = setOf(MapResourceContract.LICENSE_PATH))
        assertTrue(fetcher(first).ensure(manifest(), stacks) is MapResourceFetcher.Outcome.Failed)

        // **ولا يبقى مؤقّتٌ بعد السقوط** — فالمحاولةُ الثانيةُ نظيفة.
        assertFalse(MapPaths.resourcesStagingDir(dir, "1").exists())

        val second = Server(contractJson())
        val out = fetcher(second).ensure(manifest(), stacks)
        assertTrue("$out", out is MapResourceFetcher.Outcome.Installed)
    }

    // ══════════════════════════════════════════════════════════════
    // **الأمان** — البند ٥
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `مسارٌ صاعدٌ في العقد يُرفض`() {
        for (bad in listOf(
            "../../../databases/x",
            "glyphs/../../etc/passwd",
            "/data/data/com.other/x",
            "glyphs/%2e%2e/x.pbf",
            "..\\windows\\system32",
        )) {
            var threw = false
            try {
                MapResourceContract.parse(contractJson(extraPath = bad), "1")
            } catch (e: IllegalArgumentException) {
                threw = true
            }
            assertTrue("«$bad» يجب أن يُرفض", threw)
        }
    }

    @Test
    fun `عنوانٌ مطلقٌ في العقد يُرفض`() {
        for (bad in listOf(
            "https://evil.example/x.pbf",
            "file:///data/data/com.rahalgo.driver/databases/x",
            "content://settings/secure",
        )) {
            var threw = false
            try {
                MapResourceContract.parse(contractJson(extraPath = bad), "1")
            } catch (e: IllegalArgumentException) {
                threw = true
            }
            assertTrue("«$bad» يجب أن يُرفض", threw)
        }
    }

    @Test
    fun `نوعُ ملفٍّ غيرُ معروفٍ يُرفض`() {
        // **قائمةٌ بيضاءُ لا تنظيف** — فلا يكتب العقدُ ملفّاً لا نتوقّعه.
        for (bad in listOf("evil.sh", "glyphs/x.txt", "libs/native.so", "fonts/font.ttf")) {
            var threw = false
            try {
                MapResourceContract.parse(contractJson(extraPath = bad), "1")
            } catch (e: IllegalArgumentException) {
                threw = true
            }
            assertTrue("«$bad» يجب أن يُرفض", threw)
        }
    }

    @Test
    fun `مسارُ مورِدٍ لا يخرج عن مجلّد نسخته`() {
        val base = File(dir, "maps/resources/.tmp-1").also { it.mkdirs() }
        for (bad in listOf("../x", "a/../../b", "/abs", "a//b", "")) {
            var threw = false
            try {
                MapPaths.resourceFileIn(base, bad)
            } catch (e: IllegalArgumentException) {
                threw = true
            }
            assertTrue("«$bad» يجب أن يُرفض", threw)
        }
        // **والصحيحُ يمرّ** — بمسافةٍ في اسم الرصّة.
        val ok = MapPaths.resourceFileIn(base, "glyphs/RahalGo Regular/0-255.pbf")
        assertTrue(ok.canonicalPath.startsWith(base.canonicalPath))
    }

    @Test
    fun `عقدٌ بنسخةٍ مخالفةٍ يُرفض`() {
        var threw = false
        try {
            MapResourceContract.parse(contractJson(version = "9"), "1")
        } catch (e: IllegalArgumentException) {
            threw = true
        }
        assertTrue(threw)
    }

    @Test
    fun `عقدٌ بعقدِ عنوانٍ مخالفٍ يُرفض`() {
        val bad = JSONObject(contractJson()).put("glyphUrlTemplate", "{range}.pbf").toString()
        var threw = false
        try {
            MapResourceContract.parse(bad, "1")
        } catch (e: IllegalArgumentException) {
            threw = true
        }
        assertTrue(threw)
    }

    @Test
    fun `عقدٌ بلا files يُرفض`() {
        val old = JSONObject(contractJson()).apply { remove("files") }.toString()
        var threw = false
        try {
            MapResourceContract.parse(old, "1")
        } catch (e: IllegalArgumentException) {
            threw = true
        }
        assertTrue("عقدٌ من قبل الإغلاق الوظيفيّ", threw)
    }

    // ══════════════════════════════════════════════════════════════
    // **التنظيف** — البند ٧
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `نسخةُ مواردَ تطلبها حزمةٌ مركَّبةٌ لا تُحذف`() {
        fetcher(Server(contractJson())).ensure(manifest(), stacks)
        installRegion("raqqa", "2026-08-21", resourcesVersion = "1")

        assertFalse("محميّةٌ لأنّ حزمةً تطلبها", store.deleteResourceVersion("1", emptySet()))
        assertTrue(store.resourcesComplete("1", stacks))
    }

    @Test
    fun `والمحميّةُ صراحةً لا تُحذف`() {
        fetcher(Server(contractJson())).ensure(manifest(), stacks)
        assertFalse(store.deleteResourceVersion("1", setOf("1")))
        assertTrue(store.deleteResourceVersion("1", setOf("9")))
    }

    @Test
    fun `المؤقّتاتُ لا تُعدُّ نسخاً مركَّبة`() {
        store.resourcesStagingDir("7")
        assertTrue(store.installedResourceVersions().isEmpty())
    }

    private fun installRegion(id: String, version: String, resourcesVersion: String) {
        val archive = MapPaths.regionArchive(dir, id, version)
        archive.parentFile.mkdirs()
        archive.writeBytes(ByteArray(500))
        store.markRegionInstalled(
            MapRegion(
                id, "الرقّة", version,
                MapArtifact(
                    "regions/$version/region-$id.pmtiles", 500, "b".repeat(64), 10, 16,
                    listOf(38.92, 35.88, 39.12, 36.03),
                ),
            ),
            resourcesVersion,
        )
    }
}
