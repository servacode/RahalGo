package com.rahalgo.map.data

import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.json.JSONArray
import org.json.JSONObject
import java.io.ByteArrayInputStream
import java.io.File
import java.io.IOException
import java.io.InputStream

/**
 * ══════════════════════════════════════════════════════════════════
 * **تصنيفُ الإعادة — ونقصُ المساحة ليس عابراً**
 * ══════════════════════════════════════════════════════════════════
 *
 * (`PHASE 6B FINAL POLICY PATCH`، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 * **أمرُ المالك نصّاً**: «WorkManager retry/backoff لا يخلق مساحة
 * تخزين جديدة. وقد تبقى storageNotLow=true بينما المساحة الفعلية أقل
 * من: expectedArtifactBytes + temporaryOverhead + safetyMargin. فإعادة
 * المهمة خمس مرات قد تعطي نفس النتيجة بلا أي تغير».
 *
 * # **والفرقُ الذي يُقاس هنا**
 *
 * **«قد يتغيّر يوماً» ليست «يتغيّر بالإعادة».** الشبكةُ تعود من نفسها
 * بعد ثوانٍ، **والقرصُ لا يفرغ إلّا بفعلِ إنسان.** فالإعادةُ في الأولى
 * تنفع وفي الثانية **تشغل الطابورَ وتُعيد النتيجةَ نفسَها.**
 */
class MapRetryPolicyTest {

    private lateinit var dir: File
    private lateinit var store: MapPackageStore

    private val stacks = listOf("RahalGo Regular", "RahalGo Bold")

    @Before
    fun setup() {
        dir = File(
            System.getProperty("java.io.tmpdir"),
            "rahalgo-retry-${System.nanoTime()}",
        ).also { it.mkdirs() }
        store = MapPackageStore(dir).also { it.prepare() }
    }

    @After
    fun teardown() {
        dir.deleteRecursively()
    }

    // ══════════════════════════════════════════════════════════════
    // **التصنيفُ نفسُه**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `الشبكةُ عابرةٌ فتُعاد`() {
        assertTrue(MapFailure.NETWORK.transient)
    }

    @Test
    fun `وطولُ الانتظار عابرٌ فيُعاد`() {
        assertTrue(MapFailure.TIMEOUT.transient)
    }

    @Test
    fun `ونقصُ المساحة ليس عابراً فلا يُعاد`() {
        // **والإعادةُ لا تخلق مساحة.**
        assertFalse(MapFailure.NO_SPACE.transient)
    }

    @Test
    fun `والبصمةُ المخالفةُ لا تُعاد`() {
        assertFalse(MapFailure.CHECKSUM_MISMATCH.transient)
    }

    @Test
    fun `ولا يُعاد شيءٌ سوى الشبكةِ والانتظار`() {
        // **قائمةٌ صريحة** — فإن أُضيف سببٌ جديدٌ يوماً **قرَّر كاتبُه
        // تصنيفَه بقصد**، ولا يرث «عابراً» لأنّه كُتب بجانب عابر.
        val expected = setOf(MapFailure.NETWORK, MapFailure.TIMEOUT)
        assertEquals(expected, MapFailure.entries.filter { it.transient }.toSet())
    }

    @Test
    fun `وسببُ المنزّل يُترجَم كما هو`() {
        assertEquals(MapFailure.NO_SPACE, MapFailure.of(MapDownloader.Reason.NO_SPACE))
        assertEquals(MapFailure.NETWORK, MapFailure.of(MapDownloader.Reason.NETWORK))
        assertEquals(
            MapFailure.CHECKSUM_MISMATCH,
            MapFailure.of(MapDownloader.Reason.CHECKSUM_MISMATCH),
        )
        assertEquals(MapFailure.SECURITY, MapFailure.of(MapDownloader.Reason.BAD_URL))
    }

    // ══════════════════════════════════════════════════════════════
    // **قبل التنزيل** — لا نداءَ شبكةٍ أصلاً
    // ══════════════════════════════════════════════════════════════

    /** **خادمٌ يعدّ نداءاته** — فيُقاس أنّ صفراً منها وقع. */
    private inner class CountingServer(val contract: String) : HttpSource {
        var calls = 0
        var fileCalls = 0

        override fun open(url: String, rangeStart: Long?): HttpSource.Response {
            calls += 1
            val path = java.net.URI(url).path
            val rel = path.substringAfterLast('/')
            if (rel == MapResourceFetcher.CONTRACT_FILE) {
                return respond(contract.toByteArray(Charsets.UTF_8))
            }
            fileCalls += 1
            return respond(bodyOf(path.substringAfter("/map-resources/1/")))
        }

        private fun respond(b: ByteArray) = object : HttpSource.Response {
            override val status: Int = 200
            override val rangeStart: Long = 0
            override val remainingBytes: Long = b.size.toLong()
            override val body: InputStream = ByteArrayInputStream(b)
            override fun close() = Unit
        }
    }

    private fun bodyOf(path: String) =
        ("محتوى:$path").toByteArray(Charsets.UTF_8) + ByteArray(32)

    private fun sha(b: ByteArray): String {
        val f = File(dir, "probe-${System.nanoTime()}.bin")
        f.writeBytes(b)
        return MapDownloader.sha256(f).also { f.delete() }
    }

    private fun contractJson(): String {
        val paths = buildList {
            for (s in stacks) for (r in MapGlyphRanges.REQUIRED) add("glyphs/$s/$r.pbf")
            addAll(MapSpriteFiles.ALL)
            add(MapPaths.STYLE_FILE)
            add(MapResourceContract.LICENSE_PATH)
        }
        val files = JSONArray()
        for (p in paths) {
            val b = bodyOf(p)
            files.put(
                JSONObject().apply {
                    put("path", p); put("bytes", b.size.toLong()); put("sha256", sha(b))
                },
            )
        }
        return JSONObject().apply {
            put("resourcesVersion", "1")
            put("styleVersion", "1"); put("glyphVersion", "1"); put("spriteVersion", "1")
            put("glyphUrlTemplate", "{fontstack}/{range}.pbf")
            put("fontstacks", JSONArray(stacks.map { JSONObject().put("name", it) }))
            put("files", files)
        }.toString()
    }

    private fun manifest() = MapManifest(
        schemaVersion = 1,
        dataVersion = "2026-08-21",
        tileSchema = "3.16.0",
        resourcesVersion = "1",
        base = MapArtifact(
            "base/2026-08-21/syria.pmtiles", 1000, "a".repeat(64), 0, 16,
            listOf(35.5, 32.0, 42.5, 37.5),
        ),
        regions = listOf(
            MapRegion(
                "raqqa", "الرقّة", "2026-08-21",
                MapArtifact(
                    "regions/2026-08-21/region-raqqa.pmtiles", 5000, "b".repeat(64), 10, 16,
                    listOf(38.92, 35.88, 39.12, 36.03),
                ),
            ),
        ),
        resources = MapResources("1", "map-resources/1/", "1", "1", "1", stacks),
        attribution = "© مساهمو OpenStreetMap",
    )

    /** **مخزنٌ يزعم أنّ القرصَ ممتلئ** — بلا ملءِ قرصِ المُختبِر. */
    private inner class FullDiskStore(base: File) : MapPackageStore(base) {
        override fun usableBytes(): Long = 1024
    }

    @Test
    fun `نقصُ المساحة قبل التنزيل — صفرُ نداءِ شبكةٍ للملفّات`() {
        val full = FullDiskStore(dir).also { it.prepare() }
        val server = CountingServer(contractJson())
        val fetcher = MapResourceFetcher(
            full,
            MapDownloader(full, server),
            server,
            MapConfig("https://maps.rahalgo.com"),
        )

        val out = fetcher.ensure(manifest(), stacks)

        assertTrue("$out", out is MapResourceFetcher.Outcome.Failed)
        val f = (out as MapResourceFetcher.Outcome.Failed).reason
        assertEquals(MapFailure.NO_SPACE, f)
        assertFalse("ولا يُعاد", f.transient)
        assertEquals("لا ملفَّ طُلب من الشبكة", 0, server.fileCalls)
        assertFalse("ولا نسخةَ فُعّلت", full.resourcesComplete("1", stacks))
    }

    @Test
    fun `والمنزّلُ لا يبدأ تنزيلاً لا يمكن إكمالُه`() {
        val full = FullDiskStore(dir).also { it.prepare() }
        val server = CountingServer(contractJson())
        val out = MapDownloader(full, server).fetch(
            "https://maps.rahalgo.com/x.pmtiles",
            10_000_000,
            "a".repeat(64),
            File(dir, "x.pmtiles"),
        )
        assertTrue("$out", out is MapDownloader.Outcome.Failed)
        assertEquals(
            MapDownloader.Reason.NO_SPACE,
            (out as MapDownloader.Outcome.Failed).reason,
        )
        assertEquals("ولا نداءَ شبكةٍ واحد", 0, server.calls)
    }

    // ══════════════════════════════════════════════════════════════
    // **وأثناء التنفيذ** — يمتلئ القرصُ في المنتصف
    // ══════════════════════════════════════════════════════════════

    /**
     * **خادمٌ يسقط بامتلاء القرص أثناء القراءة.**
     *
     * **والرسالةُ من النواة** (`ENOSPC`) — وهي ما يفرّق، **إذ لا
     * استثناءَ مخصَّصٌ لامتلاء القرص في جافا.**
     */
    private inner class DiskFullMidway(private val payload: ByteArray) : HttpSource {
        override fun open(url: String, rangeStart: Long?): HttpSource.Response {
            val half = payload.size / 2
            val stream = object : InputStream() {
                private var read = 0
                override fun read(): Int = throw IOException("write failed: ENOSPC")
                override fun read(b: ByteArray, off: Int, len: Int): Int {
                    if (read >= half) {
                        throw IOException("write failed: ENOSPC (No space left on device)")
                    }
                    val n = minOf(len, half - read)
                    System.arraycopy(payload, read, b, off, n)
                    read += n
                    return n
                }
            }
            return object : HttpSource.Response {
                override val status: Int = 200
                override val rangeStart: Long = 0
                override val remainingBytes: Long = payload.size.toLong()
                override val body: InputStream = stream
                override fun close() = Unit
            }
        }
    }

    @Test
    fun `امتلاءُ القرص أثناء التنزيل يُعرض نقصَ مساحةٍ ولا يُعاد`() {
        val payload = ByteArray(8192) { (it % 251).toByte() }
        val target = File(dir, "maps/regions/raqqa/2026-08-21/region.pmtiles")

        val out = MapDownloader(
            store,
            DiskFullMidway(payload),
            MapDownloader.Tuning(safetyMarginBytes = 0, tempOverheadBytes = 0),
        ).fetch("https://a/x.pmtiles", payload.size.toLong(), "a".repeat(64), target)

        assertTrue("$out", out is MapDownloader.Outcome.Failed)
        val reason = (out as MapDownloader.Outcome.Failed).reason
        assertEquals("يُعرض نقصَ مساحةٍ لا خطأَ كتابة", MapDownloader.Reason.NO_SPACE, reason)
        assertFalse("ولا يُعاد", MapFailure.of(reason).transient)
        assertFalse("ولا هدفَ نصفَ مكتوب", target.exists())
    }

    @Test
    fun `امتلاءُ القرص أثناء تركيب الموارد لا يُفعّل ويُبقي القديم`() {
        // ── نسخةٌ سليمةٌ أوّلاً ────────────────────────────────────
        val server = CountingServer(contractJson())
        val ok = MapResourceFetcher(
            store,
            MapDownloader(store, server, MapDownloader.Tuning(safetyMarginBytes = 0, tempOverheadBytes = 0)),
            server,
            MapConfig("https://maps.rahalgo.com"),
        )
        assertTrue(ok.ensure(manifest(), stacks) is MapResourceFetcher.Outcome.Installed)
        assertTrue(store.resourcesComplete("1", stacks))
        val styleBefore = MapPaths.styleFile(dir, "1").readBytes()

        // ── ثمّ نسخةٌ ثانيةٌ يمتلئ القرصُ في وسطها ─────────────────
        val v2 = manifest().copy(
            resourcesVersion = "2",
            resources = MapResources("2", "map-resources/2/", "2", "2", "2", stacks),
        )
        val failing = object : HttpSource {
            var served = 0
            override fun open(url: String, rangeStart: Long?): HttpSource.Response {
                val path = java.net.URI(url).path
                if (path.endsWith(MapResourceFetcher.CONTRACT_FILE)) {
                    val text = contractJson().replace("\"resourcesVersion\":\"1\"", "\"resourcesVersion\":\"2\"")
                    return simple(text.toByteArray(Charsets.UTF_8))
                }
                served += 1
                if (served > 3) throw IOException("write failed: ENOSPC")
                return simple(bodyOf(path.substringAfter("/map-resources/2/")))
            }

            private fun simple(b: ByteArray) = object : HttpSource.Response {
                override val status: Int = 200
                override val rangeStart: Long = 0
                override val remainingBytes: Long = b.size.toLong()
                override val body: InputStream = ByteArrayInputStream(b)
                override fun close() = Unit
            }
        }

        val out = MapResourceFetcher(
            store,
            MapDownloader(store, failing, MapDownloader.Tuning(safetyMarginBytes = 0, tempOverheadBytes = 0)),
            failing,
            MapConfig("https://maps.rahalgo.com"),
        ).ensure(v2, stacks)

        assertTrue("$out", out is MapResourceFetcher.Outcome.Failed)
        val f = (out as MapResourceFetcher.Outcome.Failed).reason
        assertFalse("ولا يُعاد", f.transient)

        assertFalse("الجديدةُ لم تُفعَّل", store.resourcesComplete("2", stacks))
        assertNull(store.installedResources("2"))
        assertFalse("ولا مؤقّتَ بقي", MapPaths.resourcesStagingDir(dir, "2").exists())
        assertTrue("والقديمةُ باقيةٌ كاملة", store.resourcesComplete("1", stacks))
        assertTrue(MapPaths.styleFile(dir, "1").readBytes().contentEquals(styleBefore))
    }

    // ══════════════════════════════════════════════════════════════
    // **وقرارُ العامل** — نفسُ الشرط الذي يُنفّذه `retryOrFail`
    // ══════════════════════════════════════════════════════════════

    /**
     * **مرآةُ `MapPackageWorker.retryOrFail`.**
     *
     * **والعاملُ نفسُه يحتاج `WorkerParameters`** فلا يُنشأ في اختبار
     * JVM. **والشرطُ هو `failure.transient` نفسُه**، فيُقاس هنا
     * ويُقرأ هناك — **ولا منطقَ ثانٍ يُخفي انحرافاً.**
     */
    private fun wouldRetry(failure: MapFailure?, attempt: Int, max: Int = 5): Boolean = when {
        failure == null -> false
        !failure.transient -> false
        attempt < max -> true
        else -> false
    }

    @Test
    fun `العاملُ يُعيد للشبكة ولا يُعيد لنقص المساحة`() {
        assertTrue("شبكة", wouldRetry(MapFailure.NETWORK, attempt = 0))
        assertTrue("انتظار", wouldRetry(MapFailure.TIMEOUT, attempt = 0))
        assertFalse("مساحة", wouldRetry(MapFailure.NO_SPACE, attempt = 0))
        assertFalse("بصمة", wouldRetry(MapFailure.CHECKSUM_MISMATCH, attempt = 0))
        assertFalse("أمان", wouldRetry(MapFailure.SECURITY, attempt = 0))
        assertFalse("عقد", wouldRetry(MapFailure.INVALID_CONTRACT, attempt = 0))
    }

    @Test
    fun `ونقصُ المساحة لا يُعاد ولو في المحاولة الأولى`() {
        for (attempt in 0 until 5) {
            assertFalse("المحاولة $attempt", wouldRetry(MapFailure.NO_SPACE, attempt))
        }
    }

    @Test
    fun `والشبكةُ تتوقّف عند السقف`() {
        assertTrue(wouldRetry(MapFailure.NETWORK, attempt = 4))
        assertFalse("وبعده تسقط", wouldRetry(MapFailure.NETWORK, attempt = 5))
    }
}
