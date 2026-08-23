package com.rahalgo.map.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **العناوين — البنود ٣ و٤ و٤٤**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك**: «انتبه إلى: URI encoding · المسافات · Unicode
 * filenames إن وجدت · absolute path. لا تبنِ URI بتجميع String هش».
 */
class MapUrlsTest {

    private val tmp = File(System.getProperty("java.io.tmpdir")).absoluteFile

    // ── عنوانُ الأونلاين ────────────────────────────────────────────

    @Test
    fun `عنوانُ PMTiles البعيد يسبقه المخطّط`() {
        assertEquals(
            "pmtiles://https://maps.rahalgo.com/base/2026-08-21/syria.pmtiles",
            MapUrls.onlinePmtiles("https://maps.rahalgo.com/base/2026-08-21/syria.pmtiles"),
        )
    }

    @Test
    fun `العنوانُ البعيدُ لا يقبل غيرَ https`() {
        for (bad in listOf(
            "http://maps.rahalgo.com/x.pmtiles",
            "file:///data/x.pmtiles",
            "ftp://a/x.pmtiles",
            "content://media/x",
        )) {
            var threw = false
            try {
                MapUrls.onlinePmtiles(bad)
            } catch (e: Exception) {
                threw = true
            }
            assertTrue("«$bad» يجب أن يُرفض", threw)
        }
    }

    // ── الضمُّ إلى الأصل ────────────────────────────────────────────

    @Test
    fun `الضمُّ يبقى في الأصل`() {
        assertEquals(
            "https://maps.rahalgo.com/regions/2026-08-21/region-raqqa.pmtiles",
            MapUrls.resolveRemote(
                "https://maps.rahalgo.com",
                "regions/2026-08-21/region-raqqa.pmtiles",
            ),
        )
    }

    @Test
    fun `الفهرسُ لا يُخرج الضمَّ عن الأصل`() {
        // **فمن ملك الفهرسَ لا يملك أن يوجّه التطبيقَ إلى مضيفٍ آخر.**
        for (bad in listOf(
            "https://evil.example/x.pmtiles",
            "//evil.example/x.pmtiles",
            "../../evil",
            "/absolute",
            "regions/../../../etc/passwd",
            "regions/%2e%2e/%2e%2e/x",
        )) {
            var threw = false
            try {
                MapUrls.resolveRemote("https://maps.rahalgo.com", bad)
            } catch (e: Exception) {
                threw = true
            }
            assertTrue("«$bad» يجب أن يُرفض", threw)
        }
    }

    @Test
    fun `الفهرسُ لا يستطيع أن يفتح ملفّاً محلّيّاً`() {
        // **البند ٤٤** — «Remote manifest لا يستطيع إجبار التطبيق على
        // فتح ملف محلي تعسفي».
        assertFalse(MapUrls.isSafeRelative("file:///data/data/com.rahalgo.driver/databases/x"))
        assertFalse(MapUrls.isSafeRelative("/data/data/com.rahalgo.driver/x"))
        assertFalse(MapUrls.isSafeRelative("content://settings/secure"))
    }

    // ── عنوانُ الأوفلاين ───────────────────────────────────────────

    @Test
    fun `المسافةُ في المسار تُرمَّز لا تقطع العنوان`() {
        // **وهذا بعينه ما يكسر جمعَ النصوص** — العنوانُ ينقطع عند
        // أوّل مسافة، **والحروفُ لا تُحمَّل والخريطةُ بلا أسماء.**
        val f = File(tmp, "RahalGo Regular/64768-65023.pbf").absoluteFile
        val uri = MapUrls.localDirUri(f.parentFile)
        assertFalse("لا مسافةَ في عنوان", uri.contains(' '))
        assertTrue(uri.contains("RahalGo%20Regular"))
        assertTrue(uri.startsWith("file:/"))
    }

    @Test
    fun `العربيّةُ في اسم الملفّ تُرمَّز`() {
        val f = File(tmp, "الرقة/region.pmtiles").absoluteFile
        val uri = MapUrls.offlinePmtiles(f)
        assertTrue(uri.startsWith("pmtiles://file:/"))
        // **`toASCIIString` تُرمّز بايتاتِ UTF-8** — فلا محرفَ خارجَ ASCII.
        assertTrue(uri.all { it.code < 128 })
        assertTrue(uri.contains("%D8"))
    }

    @Test
    fun `الأرشيفُ المحلّيُّ يشترط مساراً مطلقاً`() {
        var threw = false
        try {
            MapUrls.offlinePmtiles(File("region.pmtiles"))
        } catch (e: IllegalArgumentException) {
            threw = true
        }
        assertTrue("MapLibre لا تعرف مجلَّدَ تشغيلنا", threw)
    }

    @Test
    fun `MBTiles بابٌ مفتوحٌ لا مسارُ إنتاج`() {
        // **البند ٥** — قدرةٌ تُضاف لاحقاً بلا إعادة كتابة.
        val f = File(tmp, "region.mbtiles").absoluteFile
        assertTrue(MapUrls.offlineMbtiles(f).startsWith("mbtiles://"))
    }

    // ── البصمة ─────────────────────────────────────────────────────

    @Test
    fun `البصمةُ ستّون وأربعةُ محارفَ ستّ عشريّة`() {
        assertTrue(MapUrls.isSha256("a".repeat(64)))
        assertTrue(MapUrls.isSha256("A".repeat(64)))
        assertFalse(MapUrls.isSha256("a".repeat(63)))
        assertFalse(MapUrls.isSha256("a".repeat(65)))
        assertFalse(MapUrls.isSha256("z".repeat(64)))
        assertFalse(MapUrls.isSha256(""))
        assertFalse(MapUrls.isSha256(null))
    }
}
