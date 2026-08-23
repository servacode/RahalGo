package com.rahalgo.map.data

import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.io.ByteArrayInputStream
import java.io.File
import java.io.InputStream

/**
 * ══════════════════════════════════════════════════════════════════
 * **التنزيلُ والتركيب — البنود ٩ و١٠ و١١ و١٢ و٣٤ و٤٦**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك**: «حتى بدون جهاز أريد اختبار: download · range
 * resume · 200-after-range behavior · SHA · atomic install · corrupt
 * package · version update · old package retention».
 *
 * **والخادمُ مُزيَّفٌ لا حقيقيّ** — فسلوكُ المدى لا يُختبر بشبكة:
 * **لا تُطلب من خادمٍ حقيقيٍّ أن يتجاهل `Range` عند الطلب.**
 */
class MapDownloadTest {

    private lateinit var dir: File
    private lateinit var store: MapPackageStore

    /** **محتوى الأثر** — ثابتٌ فبصمتُه ثابتة. */
    private val payload = ByteArray(4096) { (it % 251).toByte() }
    private val payloadSha by lazy {
        val f = File(dir, "probe.bin")
        f.writeBytes(payload)
        MapDownloader.sha256(f).also { f.delete() }
    }

    @Before
    fun setup() {
        dir = File(
            System.getProperty("java.io.tmpdir"),
            "rahalgo-dl-${System.nanoTime()}",
        ).also { it.mkdirs() }
        store = MapPackageStore(dir).also { it.prepare() }
    }

    @After
    fun teardown() {
        dir.deleteRecursively()
    }

    // ══════════════════════════════════════════════════════════════
    // **خادمٌ مُزيَّفٌ يُملى عليه سلوكُه**
    // ══════════════════════════════════════════════════════════════

    private class FakeHttp(
        private val body: ByteArray,
        /** **هل يفهم `Range`؟** — البند ١١. */
        private val honorsRange: Boolean = true,
        private val forcedStatus: Int? = null,
    ) : HttpSource {
        var lastRangeRequested: Long? = null
        var calls = 0

        override fun open(url: String, rangeStart: Long?): HttpSource.Response {
            calls += 1
            lastRangeRequested = rangeStart
            val useRange = honorsRange && rangeStart != null && rangeStart > 0
            val slice = if (useRange) body.copyOfRange(rangeStart!!.toInt(), body.size) else body
            val status = forcedStatus ?: if (useRange) 206 else 200
            return object : HttpSource.Response {
                override val status: Int = status
                override val rangeStart: Long = if (useRange) rangeStart!! else 0L
                override val remainingBytes: Long = slice.size.toLong()
                override val body: InputStream = ByteArrayInputStream(slice)
                override fun close() = Unit
            }
        }
    }

    private fun downloader(http: HttpSource, tuning: MapDownloader.Tuning? = null) =
        MapDownloader(
            store,
            http,
            tuning ?: MapDownloader.Tuning(
                safetyMarginBytes = 0,
                tempOverheadBytes = 0,
            ),
        )

    private fun target() = File(dir, "maps/regions/raqqa/2026-08-21/region.pmtiles")

    // ── التنزيلُ التامّ ────────────────────────────────────────────

    @Test
    fun `تنزيلٌ تامٌّ يُركَّب ويُطابق بصمتَه`() {
        val http = FakeHttp(payload)
        val out = downloader(http).fetch(
            "https://maps.rahalgo.com/x.pmtiles",
            payload.size.toLong(),
            payloadSha,
            target(),
        )
        assertTrue("$out", out is MapDownloader.Outcome.Installed)
        assertTrue(target().isFile)
        assertEquals(payload.size.toLong(), target().length())
        assertEquals(payloadSha, MapDownloader.sha256(target()))
    }

    @Test
    fun `لا يبقى ملفٌّ مؤقّتٌ بعد النجاح`() {
        downloader(FakeHttp(payload)).fetch(
            "https://a/x.pmtiles", payload.size.toLong(), payloadSha, target(),
        )
        val leftovers = store.tmpDir().listFiles()?.filter { it.isFile } ?: emptyList()
        assertTrue("بقيت مؤقّتات: $leftovers", leftovers.isEmpty())
    }

    // ── الاستئناف ─────────────────────────────────────────────────

    @Test
    fun `الاستئنافُ يطلب المدى ولا يُعيد ما نُزّل`() {
        val part = store.partFile(payloadSha)
        part.writeBytes(payload.copyOfRange(0, 1000))

        val http = FakeHttp(payload, honorsRange = true)
        val out = downloader(http).fetch(
            "https://a/x.pmtiles", payload.size.toLong(), payloadSha, target(),
        )

        assertEquals(1000L, http.lastRangeRequested)
        assertTrue("$out", out is MapDownloader.Outcome.Installed)
        assertEquals(payloadSha, MapDownloader.sha256(target()))
    }

    @Test
    fun `خادمٌ يتجاهل المدى ويردّ ٢٠٠ يُبدأ من الصفر`() {
        /**
         * **وهذا هو الفخُّ الذي سمّاه المالكُ نصّاً.**
         *
         * **لو لُصق ردُّ 200 بآخر الجزئيّ** لصار الملفُّ أطولَ من
         * الأصل ونصفُه مكرَّر — **والبصمةُ تكشفه بعد أن يُنزَّل كلُّه.**
         */
        val part = store.partFile(payloadSha)
        part.writeBytes(payload.copyOfRange(0, 1000))

        val http = FakeHttp(payload, honorsRange = false)
        val out = downloader(http).fetch(
            "https://a/x.pmtiles", payload.size.toLong(), payloadSha, target(),
        )

        assertEquals("طُلب المدى", 1000L, http.lastRangeRequested)
        assertTrue("$out", out is MapDownloader.Outcome.Installed)
        assertEquals("لا لصقَ — الحجمُ هو الأصل", payload.size.toLong(), target().length())
        assertEquals(payloadSha, MapDownloader.sha256(target()))
    }

    @Test
    fun `مؤقّتٌ أطولُ من المنتظَر يُطرَح`() {
        val part = store.partFile(payloadSha)
        part.writeBytes(ByteArray(payload.size + 500))

        val out = downloader(FakeHttp(payload)).fetch(
            "https://a/x.pmtiles", payload.size.toLong(), payloadSha, target(),
        )
        assertTrue("$out", out is MapDownloader.Outcome.Installed)
        assertEquals(payloadSha, MapDownloader.sha256(target()))
    }

    @Test
    fun `مؤقّتُ أثرٍ آخرَ لا يُستأنف عليه`() {
        // **الهويّةُ في الاسم** — فبقايا نسخةٍ أقدمَ لا تُلصق بالجديدة.
        val other = store.partFile("b".repeat(64))
        other.writeBytes(ByteArray(2000))

        val http = FakeHttp(payload)
        downloader(http).fetch(
            "https://a/x.pmtiles", payload.size.toLong(), payloadSha, target(),
        )
        assertEquals("لم يُطلب مدىً — لا مؤقّتَ لهذا الأثر", null, http.lastRangeRequested)
    }

    // ── البصمةُ والحجم ─────────────────────────────────────────────

    @Test
    fun `بصمةٌ مخالفةٌ لا تُركَّب`() {
        val out = downloader(FakeHttp(payload)).fetch(
            "https://a/x.pmtiles", payload.size.toLong(), "c".repeat(64), target(),
        )
        assertTrue("$out", out is MapDownloader.Outcome.Failed)
        assertEquals(
            MapDownloader.Reason.CHECKSUM_MISMATCH,
            (out as MapDownloader.Outcome.Failed).reason,
        )
        assertFalse("لا حزمةٌ ناقصةٌ تُركَّب", target().exists())
    }

    @Test
    fun `ملفٌّ مقتطَعٌ يُكشف بالحجم`() {
        val truncated = payload.copyOfRange(0, 1000)
        val out = downloader(FakeHttp(truncated)).fetch(
            "https://a/x.pmtiles", payload.size.toLong(), payloadSha, target(),
        )
        assertTrue("$out", out is MapDownloader.Outcome.Failed)
        assertFalse(target().exists())
    }

    @Test
    fun `بصمةٌ منتظَرةٌ غيرُ صحيحةٍ تُرفض قبل أيّ نداء`() {
        val http = FakeHttp(payload)
        val out = downloader(http).fetch(
            "https://a/x.pmtiles", payload.size.toLong(), "ليست بصمة", target(),
        )
        assertTrue(out is MapDownloader.Outcome.Failed)
        assertEquals("لا نداءَ شبكةٍ أصلاً", 0, http.calls)
    }

    // ── المساحة ───────────────────────────────────────────────────

    @Test
    fun `لا يبدأ تنزيلاً لا يمكن إكمالُه`() {
        val huge = MapDownloader.Tuning(
            safetyMarginBytes = Long.MAX_VALUE / 4,
            tempOverheadBytes = Long.MAX_VALUE / 4,
        )
        val http = FakeHttp(payload)
        val out = MapDownloader(store, http, huge).fetch(
            "https://a/x.pmtiles", payload.size.toLong(), payloadSha, target(),
        )
        assertTrue("$out", out is MapDownloader.Outcome.Failed)
        assertEquals(
            MapDownloader.Reason.NO_SPACE,
            (out as MapDownloader.Outcome.Failed).reason,
        )
        assertEquals("ولا نداءَ شبكةٍ يُهدر", 0, http.calls)
    }

    // ── الإلغاء ───────────────────────────────────────────────────

    @Test
    fun `الإلغاءُ يوقف ولا يُركّب`() {
        val out = downloader(FakeHttp(payload)).fetch(
            "https://a/x.pmtiles",
            payload.size.toLong(),
            payloadSha,
            target(),
            cancellation = { true },
        )
        assertTrue("$out", out is MapDownloader.Outcome.Cancelled)
        assertFalse(target().exists())
    }

    @Test
    fun `التقدّمُ يُبلَّغ بالبايتات`() {
        val seen = mutableListOf<Long>()
        downloader(FakeHttp(payload)).fetch(
            "https://a/x.pmtiles",
            payload.size.toLong(),
            payloadSha,
            target(),
            progress = { done, _ -> seen += done },
        )
        assertTrue(seen.isNotEmpty())
        assertEquals(payload.size.toLong(), seen.last())
        // **ولا يتراجع** — العدّادُ يزيد أو يثبت.
        assertEquals(seen.sorted(), seen)
    }

    // ── التركيبُ الذرّيُّ وبقاءُ القديم ────────────────────────────

    @Test
    fun `القديمُ الصالحُ يبقى إن سقط الجديد`() {
        // **البند ٣٤** — «إذا الجديد فشل: القديم يستمر».
        val old = File(dir, "maps/regions/raqqa/2026-08-01/region.pmtiles")
        old.parentFile.mkdirs()
        old.writeBytes(payload)

        val newTarget = File(dir, "maps/regions/raqqa/2026-08-21/region.pmtiles")
        val out = downloader(FakeHttp(payload)).fetch(
            "https://a/x.pmtiles", payload.size.toLong(), "d".repeat(64), newTarget,
        )
        assertTrue(out is MapDownloader.Outcome.Failed)
        assertTrue("القديمُ باقٍ", old.isFile)
        assertEquals(payload.size.toLong(), old.length())
    }

    @Test
    fun `النقلةُ ذرّيّةٌ — لا هدفَ نصفَ مكتوب`() {
        val part = File(store.tmpDir(), "x.part")
        part.writeBytes(payload)
        val t = target()
        store.installFile(part, t)
        assertFalse("المؤقّتُ اختفى", part.exists())
        assertEquals(payload.size.toLong(), t.length())
    }

    @Test
    fun `تنظيفُ المؤقّت بسياسةٍ معلنة`() {
        val fresh = File(store.tmpDir(), "fresh.part").apply { writeBytes(ByteArray(10)) }
        val stale = File(store.tmpDir(), "stale.part").apply { writeBytes(ByteArray(10)) }
        val now = System.currentTimeMillis()
        stale.setLastModified(now - 10L * 24 * 3600 * 1000)

        val n = store.cleanStaleTemp(7L * 24 * 3600 * 1000, now)
        assertEquals(1, n)
        assertTrue("الحديثُ يُستأنف", fresh.exists())
        assertFalse("والقديمُ يُنظَّف", stale.exists())
    }
}
