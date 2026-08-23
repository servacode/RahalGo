package com.rahalgo.map.data

import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
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
 * **من جهازٍ فارغٍ إلى خريطةٍ تعمل — البنود ١٦ إلى ١٩**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك نصّاً** (البند ١٦):
 *
 *	مجلّدٌ فارغ → فهرسٌ يُجلب → موارُد غائبة → تُنزَّل → تُتحقَّق
 *	→ أرشيفُ الرقّة يُنزَّل → يُتحقَّق → تُفعَّل → المنطقةُ تُفعَّل
 *	→ المحلِّلُ يردّ OFFLINE_REGION جاهزة
 *
 * ثمّ إقلاعٌ جديد: **الحالُ يُعاد بناؤها من القرص · لا إعادةَ تنزيل ·
 * النسخُ نفسُها فعّالة.**
 *
 * # **وأرشيفُ الرقّة الحقيقيّ** (البند ٢٠)
 *
 * **يُستعمل إن كان على القرص** (`C:\\maps\\raqqa-offline.pmtiles`،
 * ١٫٩ م.ب · ١٧٠٩ بلاطة · مرَّ `validate.mjs package`). **ولا يدخل
 * Git** — فإن غاب يُصطنع أرشيفٌ بحجمه، **والعقدُ المُختبَرُ واحدٌ في
 * الحالين.**
 */
class MapEndToEndTest {

    private lateinit var dir: File
    private lateinit var store: MapPackageStore

    private val stacks = listOf("RahalGo Regular", "RahalGo Bold")
    private val canonical: String by lazy { CanonicalStyle.text() }

    /** **أرشيفُ الرقّة** — الحقيقيُّ إن وُجد، وإلّا مُصطنَع. */
    private val archiveBytes: ByteArray by lazy {
        val real = File("C:/maps/raqqa-offline.pmtiles")
        if (real.isFile) real.readBytes() else ByteArray(1_919_410) { (it % 251).toByte() }
    }

    @Before
    fun setup() {
        dir = File(
            System.getProperty("java.io.tmpdir"),
            "rahalgo-e2e-${System.nanoTime()}",
        ).also { it.mkdirs() }
        store = MapPackageStore(dir).also { it.prepare() }
    }

    @After
    fun teardown() {
        dir.deleteRecursively()
    }

    // ══════════════════════════════════════════════════════════════
    // **خادمٌ يخدم فهرساً ومواردَ وأرشيفاً**
    // ══════════════════════════════════════════════════════════════

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

    private fun contractJson(version: String, corrupt: String? = null): String {
        val files = JSONArray()
        for (p in resourcePaths()) {
            val body = bodyOf("$version/$p")
            files.put(
                JSONObject().apply {
                    put("path", p)
                    put("bytes", body.size.toLong())
                    put("sha256", if (p == corrupt) "c".repeat(64) else sha(body))
                },
            )
        }
        return JSONObject().apply {
            put("resourcesVersion", version)
            put("styleVersion", version)
            put("glyphVersion", version)
            put("spriteVersion", version)
            put("glyphUrlTemplate", "{fontstack}/{range}.pbf")
            put("fontstacks", JSONArray(stacks.map { JSONObject().put("name", it) }))
            put("fontLicense", JSONObject().put("spdx", "OFL-1.1"))
            put("files", files)
        }.toString()
    }

    private fun manifestJson(dataVersion: String, resourcesVersion: String): String = JSONObject()
        .apply {
            put("schemaVersion", 1)
            put("dataVersion", dataVersion)
            put("tileSchema", "3.16.0")
            put("resourcesVersion", resourcesVersion)
            put(
                "base",
                JSONObject().apply {
                    put("url", "base/$dataVersion/syria.pmtiles")
                    put("bytes", 325_356_964L)
                    put("sha256", "a".repeat(64))
                    put("minZoom", 0); put("maxZoom", 16)
                    put("bbox", JSONArray(listOf(35.5, 32.0, 42.5, 37.5)))
                },
            )
            put(
                "regions",
                JSONArray().put(
                    JSONObject().apply {
                        put("id", "raqqa")
                        put("name", "الرقّة")
                        put("dataVersion", dataVersion)
                        put("url", "regions/$dataVersion/region-raqqa.pmtiles")
                        put("bytes", archiveBytes.size.toLong())
                        put("sha256", sha(archiveBytes))
                        put("minZoom", 10); put("maxZoom", 16)
                        put("bbox", JSONArray(listOf(38.92, 35.88, 39.12, 36.03)))
                    },
                ),
            )
            put(
                "resources",
                JSONObject().apply {
                    put("url", "map-resources/$resourcesVersion/")
                    put("styleVersion", resourcesVersion)
                    put("glyphVersion", resourcesVersion)
                    put("spriteVersion", resourcesVersion)
                    put("fontstacks", JSONArray(stacks))
                },
            )
            put("attribution", "© مساهمو OpenStreetMap · OpenMapTiles")
        }.toString()

    private inner class Server(
        var dataVersion: String = "2026-08-21",
        var resourcesVersion: String = "1",
        var corruptResource: String? = null,
    ) : HttpSource {
        var downloads = 0

        override fun open(url: String, rangeStart: Long?): HttpSource.Response {
            val path = java.net.URI(url).path
            return when {
                path.endsWith("/manifest.json") ->
                    respond(manifestJson(dataVersion, resourcesVersion).toByteArray(Charsets.UTF_8))

                path.endsWith(".pmtiles") -> {
                    downloads += 1
                    respond(archiveBytes)
                }

                path.contains("/map-resources/") -> {
                    val v = path.substringAfter("/map-resources/").substringBefore('/')
                    val rel = path.substringAfter("/map-resources/$v/")
                    if (rel == MapResourceFetcher.CONTRACT_FILE) {
                        respond(contractJson(v, corruptResource).toByteArray(Charsets.UTF_8))
                    } else {
                        downloads += 1
                        respond(bodyOf("$v/$rel"))
                    }
                }

                else -> throw HttpSource.HttpException("لا شيءَ في $path")
            }
        }

        private fun respond(body: ByteArray) = object : HttpSource.Response {
            override val status: Int = 200
            override val rangeStart: Long = 0
            override val remainingBytes: Long = body.size.toLong()
            override val body: InputStream = ByteArrayInputStream(body)
            override fun close() = Unit
        }
    }

    private fun installerFor(http: HttpSource, s: MapPackageStore): MapPackageInstaller {
        val config = MapConfig("https://maps.rahalgo.com")
        val downloader = MapDownloader(
            s, http,
            MapDownloader.Tuning(safetyMarginBytes = 0, tempOverheadBytes = 0),
        )
        return MapPackageInstaller(
            s, downloader, config, MapResourceFetcher(s, downloader, http, config),
        )
    }

    private fun manifestOf(http: HttpSource): MapManifest =
        http.open("https://maps.rahalgo.com/manifest.json", null).use {
            MapManifest.parse(it.body.readBytes().toString(Charsets.UTF_8))
        }

    // ══════════════════════════════════════════════════════════════
    // **١٦ · جهازٌ نظيفٌ تماماً**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `من مجلّدٍ فارغٍ إلى حزمةٍ جاهزةٍ دونَ اتّصال`() {
        assertTrue("المجلّدُ فارغ", store.installedRegions().isEmpty())
        assertTrue(store.installedResourceVersions().isEmpty())

        val server = Server()
        val manifest = manifestOf(server)
        val installer = installerFor(server, store)

        assertEquals(MapPackageInstaller.Step.Ok, installer.ensureResources(manifest, stacks))
        assertTrue("الموارُد كاملة", store.resourcesComplete("1", stacks))

        assertEquals(
            MapPackageInstaller.Step.Ok,
            installer.ensureRegion(manifest, "raqqa", stacks),
        )
        val installed = store.installedRegion("raqqa", "2026-08-21")
        assertNotNull(installed)
        assertEquals(archiveBytes.size.toLong(), installed!!.bytes)

        // ── والمحلِّلُ يردّ حزمةً جاهزة ───────────────────────────
        val decision = MapSourceResolver.resolve(
            MapSourceResolver.Request(
                purpose = MapSourceResolver.Purpose.NAVIGATION,
                online = false,
                lat = 35.95,
                lng = 39.01,
                installed = store.installedRegions(),
                installedResourcesVersions = setOf("1"),
                manifest = manifest,
            ),
        )
        assertTrue("$decision", decision is MapSourceResolver.Decision.OfflineRegion)

        // ── والربطُ يُنتج نمطاً جاهزاً ────────────────────────────
        val runtime = MapRuntime(store, canonical, MapConfig("https://maps.rahalgo.com"))
        val bound = runtime.bindOffline(
            (decision as MapSourceResolver.Decision.OfflineRegion).region,
        )
        assertTrue("$bound", bound is MapRuntime.Binding.Ready)
    }

    // ══════════════════════════════════════════════════════════════
    // **١٦ · وإقلاعٌ جديد** — الحالُ من القرص
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `إقلاعٌ جديدٌ يُعيد الحالَ من القرص بلا تنزيل`() {
        val server = Server()
        val manifest = manifestOf(server)
        installerFor(server, store).let {
            it.ensureResources(manifest, stacks)
            it.ensureRegion(manifest, "raqqa", stacks)
        }
        val downloadsBefore = server.downloads
        assertTrue("نُزّل شيءٌ فعلاً", downloadsBefore > 0)

        /**
         * **إقلاعٌ جديد** — مخزنٌ جديدٌ على المجلّد نفسِه.
         *
         * **ولا حالةَ في الذاكرة تعبر** — كلُّ ما يُعرف يُقرأ من
         * القرص، **وهو بعينه ما يقع بعد قتل النظام للعمليّة.**
         */
        val fresh = MapPackageStore(dir).also { it.prepare() }
        assertEquals(1, fresh.installedRegions().size)
        assertEquals(listOf("1"), fresh.installedResourceVersions())
        assertTrue(fresh.resourcesComplete("1", stacks))

        val again = installerFor(server, fresh)
        assertEquals(MapPackageInstaller.Step.Ok, again.ensureResources(manifest, stacks))
        assertEquals(
            MapPackageInstaller.Step.Ok,
            again.ensureRegion(manifest, "raqqa", stacks),
        )
        assertEquals("ولا بايتَ أُعيد تنزيلُه", downloadsBefore, server.downloads)
    }

    // ══════════════════════════════════════════════════════════════
    // **١٧ · تحديثٌ ورجوعُه**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `تحديثٌ ناجحٌ يُبدّل النسختين ويُبقي القديمةَ حتّى ينجح`() {
        val server = Server()
        val v1 = manifestOf(server)
        installerFor(server, store).let {
            it.ensureResources(v1, stacks)
            it.ensureRegion(v1, "raqqa", stacks)
        }

        server.dataVersion = "2026-09-01"
        server.resourcesVersion = "2"
        val v2 = manifestOf(server)
        val installer = installerFor(server, store)

        assertEquals(MapPackageInstaller.Step.Ok, installer.ensureResources(v2, stacks))
        assertEquals(
            MapPackageInstaller.Step.Ok,
            installer.ensureRegion(v2, "raqqa", stacks),
        )

        // **والاثنتان موجودتان الآن** — البند ٣٤ من ٦ب.
        assertEquals(setOf("1", "2"), store.installedResourceVersions().toSet())
        assertEquals(2, store.installedRegions().size)

        // ── ثمّ التنظيفُ بعد الاستقرار ────────────────────────────
        val lease = MapArchiveLease()
        val removedRegions = installer.cleanupOldVersions("raqqa", "2026-09-01", lease)
        val removedRes = installer.cleanupOldResources("2", emptySet())
        assertEquals(1, removedRegions)
        assertEquals(1, removedRes)
        assertEquals(listOf("2"), store.installedResourceVersions())
        assertEquals(1, store.installedRegions().size)
    }

    @Test
    fun `تحديثٌ فاسدٌ لا يمسّ القديم`() {
        val server = Server()
        val v1 = manifestOf(server)
        installerFor(server, store).let {
            it.ensureResources(v1, stacks)
            it.ensureRegion(v1, "raqqa", stacks)
        }
        assertTrue(store.resourcesComplete("1", stacks))

        server.dataVersion = "2026-09-01"
        server.resourcesVersion = "2"
        server.corruptResource = "sprite@2x.png"
        val v2 = manifestOf(server)

        val out = installerFor(server, store).ensureResources(v2, stacks)
        assertTrue("$out", out is MapPackageInstaller.Step.Failed)

        assertFalse("الجديدةُ لم تُفعَّل", store.resourcesComplete("2", stacks))
        assertTrue("والقديمةُ باقيةٌ كاملة", store.resourcesComplete("1", stacks))
        assertEquals(listOf("1"), store.installedResourceVersions())
        assertNotNull("وحزمةُ الرقّةِ القديمةُ صالحة", store.installedRegion("raqqa", "2026-08-21"))
    }

    // ══════════════════════════════════════════════════════════════
    // **١٨ · التوافق**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `حزمةٌ تطلب نسخةَ مواردَ غيرَ مركَّبةٍ لا تُفعَّل`() {
        val server = Server()
        val v1 = manifestOf(server)
        installerFor(server, store).let {
            it.ensureResources(v1, stacks)
            it.ensureRegion(v1, "raqqa", stacks)
        }

        // **حزمةٌ تقول نسخةَ مواردَ ٢ والجهازُ عنده ١** — لا تُختار.
        val orphan = store.installedRegions().first().copy(resourcesVersion = "2")
        assertEquals(
            null,
            MapSourceResolver.bestCovering(listOf(orphan), 35.95, 39.01, setOf("1")),
        )

        // ── ثمّ تُركَّب ٢ فتصير مؤهَّلة ───────────────────────────
        server.resourcesVersion = "2"
        val v2 = manifestOf(server)
        assertEquals(
            MapPackageInstaller.Step.Ok,
            installerFor(server, store).ensureResources(v2, stacks),
        )
        assertNotNull(
            MapSourceResolver.bestCovering(listOf(orphan), 35.95, 39.01, setOf("1", "2")),
        )
    }

    @Test
    fun `أرشيفٌ مركَّبٌ ومواردُه ناقصةٌ لا يُعدُّ جاهزاً`() {
        // **أمرُ المالك**: «لا يجوز اعتبار Region INSTALLED/READY إذا
        // كانت الموارد المطلوبة غير مثبتة».
        val server = Server()
        val manifest = manifestOf(server)
        val installer = installerFor(server, store)
        installer.ensureResources(manifest, stacks)
        installer.ensureRegion(manifest, "raqqa", stacks)

        // **ثمّ يُفسد ملفُّ حرفٍ بعد التركيب** — قرصٌ نظّف، أو قُطع.
        MapPaths.glyphFile(dir, "1", stacks[0], "64768-65023").writeBytes(ByteArray(0))

        val again = installer.ensureRegion(manifest, "raqqa", stacks)
        assertTrue("$again", again is MapPackageInstaller.Step.Failed)
        assertEquals(
            MapFailure.INCOMPLETE,
            (again as MapPackageInstaller.Step.Failed).failure,
        )
    }

    // ══════════════════════════════════════════════════════════════
    // **١٩ · عقدُ الأوفلاين — صفرُ عنوانٍ بعيد**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `النمطُ دونَ اتّصالٍ لا يحوي عنواناً بعيداً واحداً`() {
        /**
         * **أمرُ المالك نصّاً**: «OFFLINE style after binding → zero
         * https:// · zero http:// · zero remote glyph/sprite/tile
         * resources. حتى Airplane-mode Device Test لاحقًا لا يكشف
         * أننا نسينا URL بعيدة».
         *
         * **وهذا أهمُّ فحصٍ في الإغلاق**: عنوانٌ بعيدٌ واحدٌ منسيٌّ في
         * النمط **يجعل الخريطةَ تنتظر شبكةً في وضع الطيران** —
         * **ولا يظهر في أيّ اختبارٍ إلّا هذا.**
         */
        val server = Server()
        val manifest = manifestOf(server)
        val installer = installerFor(server, store)
        installer.ensureResources(manifest, stacks)
        installer.ensureRegion(manifest, "raqqa", stacks)

        val region = store.installedRegion("raqqa", "2026-08-21")!!
        val runtime = MapRuntime(store, canonical, MapConfig("https://maps.rahalgo.com"))
        val bound = runtime.bindOffline(region)
        assertTrue("$bound", bound is MapRuntime.Binding.Ready)

        val json = (bound as MapRuntime.Binding.Ready).bound.json

        assertFalse("لا https في النمط دونَ اتّصال", json.contains("https://"))
        assertFalse("ولا http", json.contains("http://"))
        assertFalse("ولا مخطّطٌ بلا بروتوكول", json.contains("\"//"))
        assertFalse("ولا راستر OSM بحال", json.contains("openstreetmap.org"))
        assertFalse("ولا مضيفُنا", json.contains("rahalgo.com"))

        // **والعناوينُ الثلاثةُ كلُّها محلّيّة.**
        val b = bound.bound
        assertTrue("البلاطات: ${b.tileUri}", b.tileUri.startsWith("pmtiles://file:/"))
        assertTrue("الحروف: ${b.glyphsUri}", b.glyphsUri.startsWith("file:/"))
        assertTrue("الأيقونات: ${b.spriteUri}", b.spriteUri.startsWith("file:/"))

        // **وكلُّ ملفٍّ يشير إليه العنوانُ موجودٌ فعلاً** — لا ٤٠٤ محلّيّ.
        assertTrue(region.archive.isFile)
        for (s in stacks) {
            for (r in MapGlyphRanges.REQUIRED) {
                assertTrue("$s/$r مفقود", MapPaths.glyphFile(dir, "1", s, r).isFile)
            }
        }
        for (f in MapSpriteFiles.ALL) {
            assertTrue("$f مفقود", MapPaths.spriteFile(dir, "1", f).isFile)
        }
        assertTrue(MapPaths.styleFile(dir, "1").isFile)
    }

    @Test
    fun `والنمطُ المتّصلُ يحوي عناوينَ بعيدةً وحدَه`() {
        // **الفحصُ ضدَّه** — فلو ردَّ الربطُ محلّيّاً دائماً لمرَّ
        // الفحصُ السابقُ بلا معنى.
        val runtime = MapRuntime(store, canonical, MapConfig("https://maps.rahalgo.com"))
        val out = runtime.bindOnline(manifestOf(Server()))
        assertTrue("$out", out is MapRuntime.Binding.Ready)
        val json = (out as MapRuntime.Binding.Ready).bound.json
        assertTrue(json.contains("https://maps.rahalgo.com"))
        assertFalse("ولا راستر OSM حتّى في المتّصل", json.contains("openstreetmap.org"))
    }
}

/** **النمطُ المرجعيّ** — يُقرأ من المستودع لا يُكتب في اختبار. */
internal object CanonicalStyle {
    fun text(): String {
        val here = File("").absoluteFile
        val candidates = listOf(
            File(here, "../maps/style/rahalgo.style.json"),
            File(here, "../../maps/style/rahalgo.style.json"),
            File(here, "src/main/res/raw/rahalgo_style.json"),
            File(here, "map/src/main/res/raw/rahalgo_style.json"),
        )
        return candidates.firstOrNull { it.isFile }?.readText()
            ?: error("لم يُوجد النمطُ المرجعيّ — جُرّب: ${candidates.map { it.absolutePath }}")
    }
}
