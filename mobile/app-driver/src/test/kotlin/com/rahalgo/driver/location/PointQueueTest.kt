package com.rahalgo.driver.location

import com.rahalgo.shared.model.TrackPoint
import java.io.File
import java.nio.file.Files
import java.time.Instant
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طابورُ النقاط في 0.9.0 — ساعةٌ كاملةٌ بلا شبكة، ويُرسَل أجزاءً** (A6.1–A6.3)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كان يُقصّ إلى مئتين حين يبلغ ٦٤ ك.ب** (ربعُ ساعةٍ في طلب)، **ويُرسَل كلُّه في نداءٍ واحد**
 * ثمّ يُحذف الملفّ — فنقطةٌ أُضيفت أثناء النداء تُمحى (`R19`). **والآن الطابورُ ملفٌّ حقيقيٌّ
 * في مجلّدٍ مؤقّت** — بلا جهازٍ ولا `Context`.
 */
class PointQueueTest {

    private val dir: File = Files.createTempDirectory("rahalgo-queue").toFile()
    private val file = File(dir, "points.jsonl")

    @After
    fun tearDown() {
        dir.deleteRecursively()
    }

    private val t0: Instant = Instant.parse("2026-11-05T08:00:00Z")

    /** **نقطةٌ على خطٍّ شماليّ** — كلُّ خطوةٍ [stepM] متراً تقريباً، كلَّ [everySec] ثانية. */
    private fun walk(n: Int, everySec: Long, stepM: Double, from: Instant = t0): List<TrackPoint> =
        (0 until n).map { i ->
            TrackPoint(
                lat = 35.95 + i * stepM / 111_195.0,
                lng = 39.01,
                at = from.plusSeconds(i * everySec).toString(),
                speedMps = stepM / everySec,
                accuracyM = 6.0,
            )
        }

    @Test
    fun `ساعةٌ كاملةٌ بلا شبكةٍ وهو يسير في طلب تُحفظ كلُّها`() {
        // **نقطةٌ كلَّ أربع ثوانٍ، ٣٣ متراً بينهما** (٣٠ كم/س) — ساعةٌ = ٩٠٠ نقطة.
        val q = PointQueue(file)
        val hour = walk(900, 4, 33.0)
        hour.forEach { q.add(it) }
        assertEquals("**كان يبقى منها نحو ٣٥٠ — ربعُ ساعة**", 900, q.size())
        // **وتبقى بعد إعادة تشغيل التطبيق** — الملفُّ لا الذاكرة.
        val again = PointQueue(file)
        assertEquals(900, again.size())
        assertEquals(hour.first(), again.oldest(1).single())
    }

    @Test
    fun `رجفةُ الواقف لا تملأ الطابور — قاعدةُ الأثر`() {
        // **عشرُ دقائق واقفاً، نقطةٌ كلَّ أربع ثوانٍ تتأرجح خمسةَ أمتار.**
        val q = PointQueue(file)
        val standing = (0 until 150).map { i ->
            TrackPoint(lat = 35.95 + (i % 2) * 5 / 111_195.0, lng = 39.01, at = t0.plusSeconds(i * 4L).toString())
        }
        standing.forEach { q.add(it) }
        // **واحدةٌ كلَّ عشرين ثانية** — ٦٠٠ ث ÷ ٢٠ = ٣٠.
        assertEquals(30, q.size())
    }

    @Test
    fun `قاعدةُ الأثر مرآةُ المحرّك`() {
        val a = TrackPoint(lat = 35.95, lng = 39.01, at = "2026-11-05T08:00:00Z")
        fun at(sec: Long, dLatM: Double = 0.0, mocked: Boolean? = null) = TrackPoint(
            lat = 35.95 + dLatM / 111_195.0, lng = 39.01, at = t0.plusSeconds(sec).toString(), mocked = mocked,
        )
        assertTrue("أوّلُ نقطة", PointQueue.keep(null, a))
        assertFalse("١٠ ث و١٠ م", PointQueue.keep(a, at(10, 10.0)))
        assertTrue("٢٠ ث", PointQueue.keep(a, at(20)))
        assertTrue("٢٦ م في ٤ ث", PointQueue.keep(a, at(4, 26.0)))
        assertTrue("المزيَّفةُ دائماً", PointQueue.keep(a, at(4, mocked = true)))
        assertTrue("أقدمُ من آخر محفوظة", PointQueue.keep(a, at(-30)))
        assertTrue("بلا وقتٍ مقروء", PointQueue.keep(a, TrackPoint(lat = 35.95, lng = 39.01, at = "")))
    }

    @Test
    fun `السقفُ بالعدد يُبقي الأحدث`() {
        val q = PointQueue(file)
        val pts = walk(1500, 4, 33.0)
        pts.forEach { q.add(it) }
        assertTrue("الطابورُ ${q.size()} فوق السقف", q.size() <= PointQueue.MAX_POINTS)
        // **والأحدثُ لا يُرمى** — آخرُ نقطةٍ ما زالت آخرَ الطابور.
        val all = q.oldest(PointQueue.MAX_POINTS)
        assertEquals(pts.last(), all.last())
        assertTrue("الأقدمُ هو ما يُرمى", all.first() != pts.first())
    }

    @Test
    fun `ما شاخ ساعتين عن أحدث نقطةٍ لا يُرسَل`() {
        val q = PointQueue(file)
        val old = walk(3, 60, 100.0, from = t0)
        val recent = walk(3, 60, 100.0, from = t0.plusSeconds(3 * 3600))
        (old + recent).forEach { q.add(it) }
        assertEquals("**المحرّكُ يرفضها أصلاً — إرسالُها حزمةٌ تُهدر**", recent, q.oldest(10))
        assertEquals(3, q.size())
    }

    @Test
    fun `يُرسَل أجزاءً من مئتين ويُمحى ما أُرسل وحده — R19`() {
        val q = PointQueue(file)
        val pts = walk(450, 4, 33.0)
        pts.forEach { q.add(it) }

        val part = q.oldest(200)
        assertEquals(200, part.size)
        assertEquals(pts.take(200), part)

        // **ونقطتان تصلان أثناء نداء الشبكة** — كان المسحُ يحذف الملفَّ كلَّه فتضيعان.
        val during = walk(2, 4, 33.0, from = t0.plusSeconds(450 * 4L))
        during.forEach { q.add(it) }

        assertEquals(200, q.remove(part))
        assertEquals(252, q.size())
        val rest = q.oldest(1000)
        assertEquals(pts.drop(200) + during, rest)
        // **وعلى القرص كذلك** — نسخةٌ جديدةٌ تقرأ الملفَّ نفسَه.
        assertEquals(rest, PointQueue(file).oldest(1000))
        // **وجزءٌ أُرسل مرّتين لا يمحو غيرَه.**
        assertEquals(0, q.remove(part))
        assertEquals(252, q.size())
    }

    @Test
    fun `طابورُ النسخة السابقة يُقرأ ويُفرَغ أجزاءً — وسطرٌ معطوبٌ لا يُسقطه`() {
        // **ملفُّ 0.8.0 كما تركه**: ٧٠٠ سطرٍ بلا قاعدة أثر، وسطرٌ قُطع حين قُتل التطبيق.
        val json = kotlinx.serialization.json.Json { ignoreUnknownKeys = true }
        val legacy = walk(700, 4, 5.0)
        file.writeText(
            legacy.joinToString("\n", postfix = "\n") { json.encodeToString(TrackPoint.serializer(), it) } +
                "{\"lat\":35.9,\"lng\":",
        )
        val q = PointQueue(file)
        assertEquals(700, q.size())
        var calls = 0
        val sent = mutableListOf<TrackPoint>()
        while (!q.isEmpty()) {
            val part = q.oldest(200)
            assertTrue(part.size <= 200)
            sent += part
            q.remove(part)
            calls++
        }
        assertEquals(4, calls)
        assertEquals(legacy, sent)
        assertFalse("الملفُّ الفارغ يُحذف", file.exists())
    }

    @Test
    fun `كلُّ نقاط الدفعة لا آخرُها — A6_1`() {
        // **والنظامُ يسلّم الدفعة بلا طلبٍ كلَّ دقيقتين** — كانت الأخيرةُ وحدَها تُستعمل.
        val (older, newest) = PointQueue.splitDelivered(listOf(30L, 10L, 20L)) { it }
        assertEquals(listOf(10L, 20L), older)
        assertEquals(30L, newest)
        val (none, alone) = PointQueue.splitDelivered(listOf(5L)) { it }
        assertTrue(none.isEmpty())
        assertEquals(5L, alone)
        val (empty, nothing) = PointQueue.splitDelivered(emptyList<Long>()) { it }
        assertTrue(empty.isEmpty())
        assertNull(nothing)
    }
}
