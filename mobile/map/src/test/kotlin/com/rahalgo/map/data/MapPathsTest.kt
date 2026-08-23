package com.rahalgo.map.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════
 * **صعودُ المسار والعناوين — البندان ٤٤ و٤٥**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك نصّاً**: «Region IDs وVersions القادمة من Manifest لا
 * تتحول مباشرةً إلى paths دون sanitization. اختبر: `../` · `%2e%2e` ·
 * absolute path · slashes · unicode edge cases».
 *
 * **والفهرسُ نصٌّ من الشبكة** — فمن ملك الخادمَ ملك هذه النصوص.
 */
class MapPathsTest {

    private val base = File(System.getProperty("java.io.tmpdir"), "rahalgo-paths-test")

    // ── الصعودُ يُرفض ────────────────────────────────────────────────

    @Test
    fun `الصعودُ بنقطتين مرفوض`() {
        assertFalse(MapPaths.isSafeId(".."))
        assertFalse(MapPaths.isSafeId("../.."))
        assertFalse(MapPaths.isSafeId("../../../databases"))
        assertFalse(MapPaths.isSafeId("raqqa/../../etc"))
    }

    @Test
    fun `الترميزُ لا يخفي الصعود`() {
        // **ولا يُفكُّ الترميزُ قبل الفحص** — لو فُكّ لصار `%2e%2e`
        // نقطتين ثمّ مرَّ الفحصُ على النصّ الأصليّ.
        assertFalse(MapPaths.isSafeId("%2e%2e"))
        assertFalse(MapPaths.isSafeId("%2e%2e%2f"))
        assertFalse(MapPaths.isSafeId("raqqa%2f..%2fx"))
        assertFalse(MapPaths.isSafeId("%252e%252e"))
    }

    @Test
    fun `المسارُ المطلقُ مرفوض`() {
        assertFalse(MapPaths.isSafeId("/data/data/com.other/files"))
        assertFalse(MapPaths.isSafeId("C:\\Windows"))
        assertFalse(MapPaths.isSafeId("\\\\server\\share"))
        assertFalse(MapPaths.isSafeId("/"))
    }

    @Test
    fun `الفاصلُ مرفوضٌ بأيّ اتّجاه`() {
        assertFalse(MapPaths.isSafeId("a/b"))
        assertFalse(MapPaths.isSafeId("a\\b"))
    }

    @Test
    fun `حروفُ يونيكود مرفوضة`() {
        // **العربيّةُ اسمُ عرضٍ لا معرِّفُ مسار** — والاسمُ في الفهرس
        // حقلٌ آخر.
        assertFalse(MapPaths.isSafeId("الرقة"))
        assertFalse(MapPaths.isSafeId("raqqa\u202E"))
        assertFalse(MapPaths.isSafeId("raqqa\u0000"))
        assertFalse(MapPaths.isSafeId("raqqa\n"))
        // **ونظيرُ ASCII بيونيكود** — يبدو كالنقطة وليس بها.
        assertFalse(MapPaths.isSafeId("raqqa\uFF0E\uFF0E"))
    }

    @Test
    fun `الفارغُ والطويلُ مرفوضان`() {
        assertFalse(MapPaths.isSafeId(null))
        assertFalse(MapPaths.isSafeId(""))
        assertFalse(MapPaths.isSafeId(" "))
        assertFalse(MapPaths.isSafeId("a".repeat(MapPaths.MAX_ID_LENGTH + 1)))
    }

    @Test
    fun `المعرِّفُ الصالحُ يمرّ`() {
        assertTrue(MapPaths.isSafeId("raqqa"))
        assertTrue(MapPaths.isSafeId("deir-ez-zor"))
        assertTrue(MapPaths.isSafeId("2026-08-21"))
        assertTrue(MapPaths.isSafeId("a"))
    }

    @Test
    fun `الرفضُ يرمي لا يُنظّف`() {
        // **ولا يُصلَح المعرِّفُ بحذف ما لا يصحّ** — التنظيفُ يترك
        // ثقوباً، والرفضُ لا يتركها.
        var threw = false
        try {
            MapPaths.regionDir(base, "../evil")
        } catch (e: IllegalArgumentException) {
            threw = true
        }
        assertTrue("مسارٌ صاعدٌ يجب أن يرمي", threw)
    }

    // ── بنيةُ المجلّدات ─────────────────────────────────────────────

    @Test
    fun `النسخةُ في المسار فتتعايش نسختان`() {
        val a = MapPaths.regionArchive(base, "raqqa", "2026-08-01")
        val b = MapPaths.regionArchive(base, "raqqa", "2026-08-21")
        assertTrue(a.absolutePath != b.absolutePath)
        assertTrue(a.parentFile.parentFile.absolutePath == b.parentFile.parentFile.absolutePath)
    }

    @Test
    fun `المؤقّتُ في نظام الملفّات نفسِه`() {
        // **وإلّا صارت النقلةُ نسخاً، والنسخُ يُقطع.**
        val tmp = MapPaths.tmpDir(base)
        val target = MapPaths.regionArchive(base, "raqqa", "2026-08-21")
        assertEquals(
            MapPaths.root(base).absolutePath,
            tmp.parentFile.absolutePath,
        )
        assertTrue(target.absolutePath.startsWith(MapPaths.root(base).absolutePath))
    }

    @Test
    fun `مسارُ الحرف يقبل المسافةَ في اسم الرصّة`() {
        // **`RahalGo Regular` اسمُ مجلّدٍ فيه مسافة** — وهو صحيحٌ،
        // وقد كلّف ٦أ ما كلّف حين لم يكن كذلك.
        val f = MapPaths.glyphFile(base, "1", "RahalGo Regular", "64768-65023")
        assertTrue(f.absolutePath.contains("RahalGo Regular"))
        assertTrue(f.name == "64768-65023.pbf")
    }

    @Test
    fun `مسارُ الحرف يرفض الصعودَ في اسم الرصّة`() {
        for (bad in listOf("../x", "a/b", "a\\b", "")) {
            var threw = false
            try {
                MapPaths.glyphFile(base, "1", bad, "0-255")
            } catch (e: IllegalArgumentException) {
                threw = true
            }
            assertTrue("رصّةٌ «$bad» يجب أن تُرفض", threw)
        }
    }

    @Test
    fun `النطاقُ يجب أن يكون رقمين`() {
        var threw = false
        try {
            MapPaths.glyphFile(base, "1", "RahalGo Regular", "../../etc/passwd")
        } catch (e: IllegalArgumentException) {
            threw = true
        }
        assertTrue(threw)
    }

    @Test
    fun `اسمُ المؤقّت مشتقٌّ من هويّة الأثر`() {
        // **فلا يُستأنف على بقايا أثرٍ آخر** (البند ١١).
        val a = MapPaths.partFile(base, "a".repeat(64))
        val b = MapPaths.partFile(base, "b".repeat(64))
        assertTrue(a.name != b.name)
        assertNotNull(a.parentFile)
    }
}
