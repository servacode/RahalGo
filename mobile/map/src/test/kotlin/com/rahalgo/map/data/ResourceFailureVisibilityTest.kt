package com.rahalgo.map.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File
import java.io.IOException

/**
 * ══════════════════════════════════════════════════════════════════════
 * **كلُّ إخفاقِ مواردَ يخرج مصنَّفاً — إغلاقُ `TD-MAP-SILENT-RESOURCE-FAIL`**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٢، البند ٧.)
 *
 * # **ما وقع على الجهاز المرجعيّ**
 *
 * **سقط تنزيلُ الخريطة مرّتين متتاليتين ولم يُعرف أين** (٢٠٢٦-٠٨-٢٢،
 * `SM-A525F`): لا سطرَ في `logcat`، ولا رسالةَ في الشاشة، **والزرُّ
 * عاد إلى «نزّل الآن» كأنّ شيئاً لم يقع.**
 *
 * **والسببُ كان حارساً ثانياً على `https` في `MapUrls`** — وجدتُه
 * بالقراءة لا بالسجلّ، **لأنّ السجلَّ كان صامتاً.**
 *
 * # **وما يحرسه هذا الملفّ**
 *
 * **ليس أنّ الإخفاقات تقع** — بل **أنّ كلَّ واحدٍ منها يخرج بتصنيفٍ
 * غيرِ فارغ.** فالتصنيفُ هو ما يقرؤه المُجدوِلُ ليقرّر الإعادة،
 * **وما تقرؤه الشاشةُ لتختار عبارتَها.** **وإخفاقٌ بلا تصنيفٍ إخفاقٌ
 * صامت** — وهو العطبُ بعينه.
 *
 * **ولا يُختبر النصُّ العربيُّ هنا** — يتبدّل بقرار المالك، **والتصنيفُ
 * عقدٌ.**
 */
class ResourceFailureVisibilityTest {

    @get:Rule
    val tmp = TemporaryFolder()

    // ══════════════════════════════════════════════════════════════════
    //  مصدرٌ يُملي عليه أيَّ عطبٍ يصنع
    // ══════════════════════════════════════════════════════════════════

    private class Source(
        val onOpen: (String) -> ByteArray,
    ) : HttpSource {
        val asked = mutableListOf<String>()

        override fun open(url: String, rangeStart: Long?): HttpSource.Response {
            asked += url
            val body = onOpen(url)
            return object : HttpSource.Response {
                override val status = 200
                override val rangeStart = 0L
                override val remainingBytes = body.size.toLong()
                override val body = body.inputStream()
                override fun close() = Unit
            }
        }
    }

    private fun manifest(resourcesUrl: String = "map-resources/1/"): MapManifest =
        MapManifest.parse(
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
              "regions": [{
                "id": "raqqa", "name": "الرقّة", "dataVersion": "2026-08-20",
                "url": "regions/2026-08-20/region-raqqa.pmtiles",
                "bytes": 1919410,
                "sha256": "${"a".repeat(64)}",
                "minZoom": 10, "maxZoom": 16,
                "bbox": [38.92, 35.88, 39.12, 36.03]
              }],
              "resources": {
                "url": "$resourcesUrl",
                "styleVersion": "1", "glyphVersion": "1", "spriteVersion": "1",
                "fontstacks": ["RahalGo Regular"]
              },
              "attribution": "© OpenStreetMap"
            }
            """.trimIndent(),
        )

    private fun fetcher(http: HttpSource, base: String = "https://maps.rahalgo.com"): MapResourceFetcher {
        val store = MapPackageStore(tmp.newFolder()).also { it.prepare() }
        val config = MapConfig(baseUrl = base)
        return MapResourceFetcher(store, MapDownloader(store, http), http, config)
    }

    private fun failureOf(out: MapResourceFetcher.Outcome): MapFailure {
        assertTrue("توقّعتُ إخفاقاً فجاء $out", out is MapResourceFetcher.Outcome.Failed)
        val f = (out as MapResourceFetcher.Outcome.Failed)
        assertNotNull("إخفاقٌ بلا تصنيف — وهو العطبُ بعينه", f.reason)
        assertTrue("إخفاقٌ بلا شرح", f.detail.isNotBlank())
        return f.reason
    }

    // ══════════════════════════════════════════════════════════════════
    //  ١ · عنوانُ مواردَ لا يُقبل
    // ══════════════════════════════════════════════════════════════════
    //
    // **وهذا هو العطبُ الحقيقيُّ الذي وقع**: العنوانُ يُبنى فيُرفض
    // في `MapUrls` **قبل أيّ طلبِ شبكة** — فلا `HttpException` تُلتقط،
    // **ولا شيءَ في السجلّ.**

    /**
     * **والحارسُ أسبقُ ممّا ظننت** — قِيس ٢٠٢٦-٠٨-٢٢.
     *
     * **كتبتُ الاختبارَ متوقّعاً `SECURITY` من الجالب، فسقط**: العنوانُ
     * الصاعدُ **يُرفض عند قراءة الفهرس نفسِه** (`MapResources.fromJson`
     * تشترط `isSafeRelative`)، **فلا يصل الجالبَ أصلاً.**
     *
     * **وهذا أقوى لا أضعف** — الرفضُ قبل أن يُبنى شيء. **وأُصحّح ما
     * أختبره لا ما أتمنّاه.**
     */
    @Test
    fun `عنوانُ مواردَ صاعدٌ يُرفض عند قراءة الفهرس`() {
        val e = runCatching { manifest(resourcesUrl = "map%2e%2e/1/") }.exceptionOrNull()
        assertNotNull("قُبل عنوانٌ صاعدٌ في الفهرس", e)
        assertTrue(
            "رُفض بسببٍ غيرِ المسار: ${e?.message}",
            e is IllegalArgumentException && (e.message ?: "").contains("url"),
        )
    }

    /** **ومخطّطٌ لا يُقبل** — `http` إلى مضيفٍ بعيدٍ بلا إذن. */
    @Test
    fun `مخطّطٌ غيرُ مسموحٍ يخرج SECURITY`() {
        val out = runCatching {
            fetcher(Source { ByteArray(0) }, base = "http://maps.rahalgo.com")
        }
        assertTrue("`MapConfig` كان يجب أن يرفض قبل الجلب", out.isFailure)
    }

    // ══════════════════════════════════════════════════════════════════
    //  ٢ · الشبكةُ ساقطة
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `انقطاعُ الشبكة يخرج NETWORK`() {
        val out = fetcher(Source { throw IOException("لا شبكة") })
            .ensure(manifest(), listOf("RahalGo Regular"))
        assertEquals(MapFailure.NETWORK, failureOf(out))
    }

    /** **وردُّ الخادم بخطأٍ يخرج مصنَّفاً أيضاً.** */
    @Test
    fun `ردُّ HTTP خاطئٌ يخرج NETWORK`() {
        val out = fetcher(Source { throw HttpSource.HttpException("الخادمُ ردَّ 503") })
            .ensure(manifest(), listOf("RahalGo Regular"))
        assertEquals(MapFailure.NETWORK, failureOf(out))
    }

    // ══════════════════════════════════════════════════════════════════
    //  ٣ · عقدُ مواردَ معطوب
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `عقدُ مواردَ لا يُفهم يخرج INVALID_CONTRACT`() {
        val out = fetcher(Source { "{ ليس جسون".toByteArray() })
            .ensure(manifest(), listOf("RahalGo Regular"))
        assertEquals(MapFailure.INVALID_CONTRACT, failureOf(out))
    }

    /**
     * **ورصّةٌ يطلبها النمطُ وليست في العقد** — وهي التي أوقعتني:
     * الفهرسُ يعلن واحدةً والنمطُ يطلب اثنتين، **فسقط قبل أيّ طلبِ
     * شبكةٍ للحروف.**
     */
    @Test
    fun `رصّةٌ ناقصةٌ في العقد تخرج INVALID_CONTRACT`() {
        val contract = """
            {
              "resourcesVersion": "1",
              "styleVersion": "1", "glyphVersion": "1", "spriteVersion": "1",
              "glyphUrlTemplate": "{fontstack}/{range}.pbf",
              "fontstacks": [{ "name": "RahalGo Regular", "glyphRanges": ["0-255"] }],
              "files": []
            }
        """.trimIndent().toByteArray()
        val out = fetcher(Source { contract })
            .ensure(manifest(), listOf("RahalGo Regular", "RahalGo Bold"))
        assertEquals(MapFailure.INVALID_CONTRACT, failureOf(out))
    }

    // ══════════════════════════════════════════════════════════════════
    //  ٤ · ونقصُ المساحة دائمٌ لا عابر
    // ══════════════════════════════════════════════════════════════════
    //
    // **البند ٥ من قرار ٢٠٢٦-٠٨-٢٢**: «NO_SPACE = non-transient».
    // **و`WorkManager` لا تخلق مساحةً بالانتظار.**

    @Test
    fun `تصنيفُ الإعادة لم يتبدّل`() {
        assertTrue(MapFailure.NETWORK.transient)
        assertTrue(MapFailure.TIMEOUT.transient)
        for (f in MapFailure.entries) {
            if (f != MapFailure.NETWORK && f != MapFailure.TIMEOUT) {
                assertTrue("$f صار عابراً — وهذا يفتح حلقةَ إعادة", !f.transient)
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    //  ٥ · ولا إخفاقَ بلا تصنيف
    // ══════════════════════════════════════════════════════════════════
    //
    // **وهذا الحارسُ هو الأصل**: من أضاف فرعَ إخفاقٍ جديداً بلا تصنيف
    // **أعاد العطبَ الصامت** — فالشاشةُ تختار عبارتَها بالتصنيف،
    // **وبلا تصنيفٍ لا عبارة.**

    @Test
    fun `كلُّ إخفاقٍ يحمل تصنيفاً وشرحاً`() {
        val cases: List<Pair<String, HttpSource>> = listOf(
            "شبكة" to Source { throw IOException("x") },
            "HTTP" to Source { throw HttpSource.HttpException("x") },
            "عقد" to Source { "{".toByteArray() },
            "فارغ" to Source { ByteArray(0) },
        )
        for ((what, http) in cases) {
            val out = fetcher(http).ensure(manifest(), listOf("RahalGo Regular"))
            val reason = failureOf(out)
            assertTrue("$what: تصنيفٌ فارغ", reason.name.isNotBlank())
        }
    }

    /** **ولا يُلمس القرصُ حين يسقط الجلب** — فلا نصفُ تركيبٍ يبقى. */
    @Test
    fun `الإخفاقُ لا يترك موارَد نصفَ مركَّبة`() {
        val root = tmp.newFolder()
        val store = MapPackageStore(root).also { it.prepare() }
        val f = MapResourceFetcher(
            store,
            MapDownloader(store, Source { throw IOException("لا شبكة") }),
            Source { throw IOException("لا شبكة") },
            MapConfig(baseUrl = "https://maps.rahalgo.com"),
        )
        f.ensure(manifest(), listOf("RahalGo Regular"))
        assertTrue(
            "بقيت مواردُ نصفَ مركَّبة",
            !store.resourcesComplete("1", listOf("RahalGo Regular")),
        )
        val stray = File(root, "maps/resources/1").listFiles()?.size ?: 0
        assertEquals("بقي أثرٌ على القرص بعد إخفاق", 0, stray)
    }
}
