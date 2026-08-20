package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختباراتُ مسجّل الرحلة — بلا جهاز**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
 *
 * **والمسجّلُ يأخذ مجلّداً لا `Context`** — فيُختبر كاملاً في الحاسوب،
 * **بملفٍّ حقيقيٍّ يُكتب ويُقرأ.**
 */
class TraceRecorderTest {

    @get:Rule
    val tmp = TemporaryFolder()

    private fun fix(
        i: Int,
        acc: Float = 6f,
        speed: Float? = 11f,
        bearing: Float? = 90f,
        provider: String? = "fused",
    ) = NavFix(
        lat = 35.9506 + i * 0.0001,
        lng = 39.0094,
        accuracyM = acc,
        speedMps = speed,
        bearingDeg = bearing,
        atMs = 1_000L + i * 1_000L,
        elapsedNs = (1_000L + i * 1_000L) * 1_000_000L,
        wallMs = 1_755_000_000_000L + i * 1_000L,
        provider = provider,
    )

    private fun recorder(dir: File = tmp.root) =
        TraceRecorder(dir, "sess1234", nowMs = { 1_755_000_000_000L })

    private fun linesOf(f: File) = f.readLines().filter { it.isNotBlank() }

    // ══════════════════════════════════════════════════════════════════
    // **البدءُ والإغلاق**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `بدءُ الجلسة ينشئ ملفَّ الرحلة`() {
        val r = recorder()
        assertTrue(r.start())
        val f = r.file
        assertNotNull("لم يُنشأ ملفّ", f)
        assertTrue("الملفُّ غيرُ موجود", f!!.exists())
        assertTrue("الاسمُ لا يحمل الجلسة: ${f.name}", f.name.contains("sess1234"))
        assertTrue("الاسمُ لا يحمل وقتَ البدء: ${f.name}", f.name.contains("1755000000000"))
        assertTrue(f.name.endsWith(".jsonl"))
        r.stop()
    }

    /** **وملفٌّ لم يُغلق يفقد آخرَ ما في مخزنه** — ونهايةُ الرحلة أهمُّها. */
    @Test
    fun `الإغلاقُ يُفرغ كلَّ ما كُتب`() {
        val r = recorder()
        r.start()
        repeat(50) { r.add(fix(it), FixGrade.ACCEPTED, RejectReason.NONE) }
        val f = r.stop()
        assertNotNull(f)
        assertEquals("ضاعت أسطرٌ عند الإغلاق", 50, linesOf(f!!).size)
    }

    // ══════════════════════════════════════════════════════════════════
    // **كلُّ قراءةٍ تُسجَّل — بجيّدها ورديئها**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **ورحلةٌ فيها المقبولُ وحدَه لا تختبر مرشِّحاً** — **تختبر ما نجا
     * منه.**
     */
    @Test
    fun `المقبولةُ والمتدهورةُ والمرفوضةُ كلُّها تبقى في الملفّ`() {
        val r = recorder()
        r.start()
        r.add(fix(1), FixGrade.ACCEPTED, RejectReason.NONE)
        r.add(fix(2, acc = 45f), FixGrade.DEGRADED, RejectReason.NONE)
        r.add(fix(3, acc = 90f), FixGrade.REJECTED, RejectReason.ACCURACY)
        r.add(fix(4), FixGrade.REJECTED, RejectReason.TELEPORT)
        r.add(fix(5), FixGrade.REJECTED, RejectReason.STALE)
        val f = r.stop()!!
        val rows = TraceReader.parse(linesOf(f))
        assertEquals(5, rows.size)
        assertEquals(FixGrade.ACCEPTED, rows[0].grade)
        assertEquals(FixGrade.DEGRADED, rows[1].grade)
        assertEquals(RejectReason.ACCURACY, rows[2].reason)
        assertEquals(RejectReason.TELEPORT, rows[3].reason)
        assertEquals(RejectReason.STALE, rows[4].reason)
    }

    /** **والغائبُ يبقى غائباً** — أمرُ المالك: `null` لا صفر. */
    @Test
    fun `الاتّجاهُ الغائبُ يبقى null ولا يصير صفرا`() {
        val r = recorder()
        r.start()
        r.add(fix(1, bearing = null, speed = null), FixGrade.ACCEPTED, RejectReason.NONE)
        val f = r.stop()!!
        val line = linesOf(f).first()
        assertTrue("لم يُكتب null: $line", line.contains("\"bearing_deg\":null"))
        assertTrue(line.contains("\"has_bearing\":false"))
        assertTrue(line.contains("\"speed_mps\":null"))
        val row = TraceReader.parseLine(line)
        assertNull("قُرئ صفراً بدل الغياب", row.fix.bearingDeg)
        assertNull(row.fix.speedMps)
    }

    @Test
    fun `الاتّجاهُ الموجودُ يُكتب ويُقرأ كما هو`() {
        val r = recorder()
        r.start()
        r.add(fix(1, bearing = 271.5f), FixGrade.ACCEPTED, RejectReason.NONE)
        val row = TraceReader.parseLine(linesOf(r.stop()!!).first())
        assertEquals(271.5f, row.fix.bearingDeg!!, 1e-4f)
        assertTrue(row.fix.provider == "fused")
    }

    // ══════════════════════════════════════════════════════════════════
    // **الوقتُ الرتيبُ والوقتُ الجداريّ**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **وساعةُ الجدار تقفز حين يضبطها النظامُ من الشبكة** — **فلو
     * حُسب الفاصلُ منها لَظهر سالباً أو بساعة.**
     *
     * **والرتيبُ يبقى رتيباً** فيُحسَب منه الفاصلُ الصحيح.
     */
    @Test
    fun `قفزةُ ساعةِ الجدار لا تفسد حسابَ الفواصل`() {
        val r = recorder()
        r.start()
        // **ثلاثُ قراءاتٍ بفاصلٍ رتيبٍ ثابتٍ ثانيةً** — وساعةُ الجدار
        // تقفز ساعةً إلى الوراء في الثانية.
        r.add(fix(1).copy(wallMs = 1_000_000L), FixGrade.ACCEPTED, RejectReason.NONE)
        r.add(fix(2).copy(wallMs = 1_000_000L - 3_600_000L), FixGrade.ACCEPTED, RejectReason.NONE)
        r.add(fix(3).copy(wallMs = 1_000_000L - 3_600_000L + 1_000L), FixGrade.ACCEPTED, RejectReason.NONE)
        val rows = TraceReader.parse(linesOf(r.stop()!!))
        assertEquals(3, rows.size)

        val monotonic = rows.map { it.fix.elapsedNs }
        for (i in 1 until monotonic.size) {
            val gapMs = (monotonic[i] - monotonic[i - 1]) / 1_000_000
            assertEquals("الفاصلُ الرتيبُ تبدّل", 1_000L, gapMs)
        }
        // **والجداريُّ فعلاً يعطي فاصلاً سالباً** — وهو ما نحتمي منه.
        val wallGap = rows[1].fix.wallMs - rows[0].fix.wallMs
        assertTrue("لم تُحاكَ القفزة", wallGap < 0)
    }

    // ══════════════════════════════════════════════════════════════════
    // **الفشلُ لا يُسقط الملاحة**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **القرصُ يمتلئ والملفُّ يُقفل** — **ورحلةُ سائقٍ تنهار لأنّ
     * أداةَ فحصٍ عجزت عن الكتابة** خطأٌ أفدحُ من ضياع رفيدة.
     */
    @Test
    fun `مجلّدٌ لا يُكتب فيه لا يُسقط شيئا`() {
        // **ملفٌّ في موضع المجلّد** — فلا يستطيع أن يُنشأ.
        val blocker = tmp.newFile("blocked")
        val r = TraceRecorder(File(blocker, "sub"), "s1")
        assertFalse("ادّعى النجاح", r.start())
        assertTrue("لم يُعلَّم بالفشل", r.failed)
        assertNotNull("لا سببَ مسجَّل", r.failure)
        // **ولا يرمي شيئاً بعدها** — الملاحةُ تمضي.
        r.add(fix(1), FixGrade.ACCEPTED, RejectReason.NONE)
        r.stop()
        assertEquals(0, r.written)
    }

    @Test
    fun `الإضافةُ قبل البدء لا ترمي`() {
        val r = recorder()
        r.add(fix(1), FixGrade.ACCEPTED, RejectReason.NONE)
        assertEquals(0, r.written)
        assertNull(r.file)
    }

    // ══════════════════════════════════════════════════════════════════
    // **القارئُ يُعيد الرحلة**
    // ══════════════════════════════════════════════════════════════════

    /** **والرحلةُ تعود إلى السلسلة نفسِها بلا هاتف.** */
    @Test
    fun `القارئُ يعيد الرحلةَ إلى السلسلة`() {
        val r = recorder()
        r.start()
        repeat(12) { r.add(fix(it), FixGrade.ACCEPTED, RejectReason.NONE) }
        val rows = TraceReader.parse(linesOf(r.stop()!!))
        val pipeline = NavPipeline()
        var drawn = 0
        for (f in TraceReader.fixesOf(rows)) {
            if (pipeline.onFix(f).targetLat != null) drawn++
        }
        assertEquals("لم تُعَد الرحلةُ كاملة", 12, drawn)
        assertNotNull("لم يُحسب اتّجاه", pipeline.headingDeg)
    }

    @Test
    fun `الأسطرُ الفارغةُ تُتخطّى`() {
        val r = recorder()
        r.start()
        r.add(fix(1), FixGrade.ACCEPTED, RejectReason.NONE)
        val lines = r.stop()!!.readLines() + listOf("", "   ")
        assertEquals(1, TraceReader.parse(lines).size)
    }

    // ══════════════════════════════════════════════════════════════════
    // **نسخةُ الصيغة**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `النسخةُ المدعومةُ تُقرأ`() {
        val line = TraceRecorder.lineOf("s", fix(1), FixGrade.ACCEPTED, RejectReason.NONE)
        assertEquals(TraceRecorder.SCHEMA, TraceReader.parseLine(line).schema)
    }

    /** **ورفيدةٌ من صيغةٍ لا نعرفها تُقرأ أرقاماً خاطئةً بصمت.** */
    @Test
    fun `النسخةُ غيرُ المدعومةِ تُوقف ولا تُقرأ`() {
        val future = TraceRecorder.lineOf("s", fix(1), FixGrade.ACCEPTED, RejectReason.NONE)
            .replace("\"schema_version\":1", "\"schema_version\":99")
        try {
            TraceReader.parseLine(future)
            throw AssertionError("قُرئت نسخةٌ غيرُ مدعومة")
        } catch (e: TraceReader.BadTrace) {
            assertTrue(e.message!!.contains("99"))
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **ولا شبكةَ في المسجّل**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **يُقرأ المصدرُ لا السلوك** — من أضاف رفعاً يوماً يسقط بناؤه.
     *
     * **وملفُّ الرحلة يبقى على الجهاز** (أمرُ المالك: «لا ترفع ملف
     * الرحلة إلى Backend»).
     */
    @Test
    fun `المسجّلُ والقارئُ لا يعرفان شبكةً`() {
        val files = listOf("TraceRecorder.kt", "TraceReader.kt")
        val banned = listOf("http", "Backend", "DriverApi", "sendLocation", "Socket", "URL")
        val offenders = mutableListOf<String>()
        for (name in files) {
            val f = File("src/main/kotlin/com/rahalgo/navigation/$name")
            assertTrue("لا مصدر: ${f.absolutePath}", f.exists())
            val code = f.readText()
                .replace(Regex("(?s)/\\*.*?\\*/"), " ")
                .lines().joinToString("\n") { it.substringBefore("//") }
            for (w in banned) if (code.contains(w)) offenders += "$name: $w"
        }
        assertTrue("المسجّلُ يطرق بابَ الشبكة: $offenders", offenders.isEmpty())
    }
}
