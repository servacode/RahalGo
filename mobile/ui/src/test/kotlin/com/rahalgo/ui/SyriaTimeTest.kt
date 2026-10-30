package com.rahalgo.ui

import java.time.LocalDate
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * **الوقتُ يُعرض بتوقيت سوريا — لا بغرينتش.**
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: الشكوى «فُتحت 16:51» والساعةُ 19:51، والمحفظةُ
 *  «18:09» والساعةُ 21:09.)
 */
class SyriaTimeTest {

    /** **وطابعُ غرينتش يُحوَّل** — وهو ما يرسله المحرّك. */
    @Test
    fun `طابعُ غرينتش يُعرض بساعة دمشق`() {
        val today = LocalDate.parse("2026-10-30")
        assertEquals("3 تشرين الأول · 19:51", whenText("2026-10-03T16:51:12.345Z", today))
        assertEquals("3 تشرين الأول · 21:09", whenText("2026-10-03T18:09:00Z", today))
        assertEquals("2026-10-03", dateText("2026-10-03T16:51:12Z"))
        assertEquals("19:51", timeText("2026-10-03T16:51:12Z"))
    }

    /** **وطابعٌ بإزاحة دمشق أصلاً لا يتزحزح.** */
    @Test
    fun `طابعُ دمشق يبقى كما هو`() {
        assertEquals("13 آب · 03:12", whenText("2026-08-13T03:12:44+03:00", LocalDate.parse("2026-10-30")))
        assertEquals("03:12", timeText("2026-08-13T03:12:44+03:00"))
    }

    /** **وما بعد منتصف ليل دمشق يومٌ جديد** — وإن كان في غرينتش أمسَ. */
    @Test
    fun `منتصفُ الليل يقلب التاريخ`() {
        val today = LocalDate.parse("2026-10-04")
        assertEquals("اليوم · 01:30", whenText("2026-10-03T22:30:00Z", today))
        assertEquals("أمس · 23:30", whenText("2026-10-03T20:30:00Z", today))
        assertEquals("27 أيلول 2025 · 13:00", whenText("2025-09-27T10:00:00Z", today))
        assertEquals("اليوم", dayText("2026-10-03T22:30:00Z", today))
        assertEquals("أمس", dayText("2026-10-03T20:30:00Z", today))
        assertEquals("27 أيلول", dayText("2026-09-27T10:00:00Z", today))
        assertEquals("27 أيلول 2025", dayText("2025-09-27T10:00:00Z", today))
    }

    /**
     * **وكلُّ شاشةٍ تعرض وقتاً تمرّ بالدالّة المركزيّة** — لا قصٌّ ولا طابعٌ خام.
     *
     * المحفظةُ والشكوى والإشعاراتُ وسجلُّ المحادثات وكشفُ الحساب.
     */
    @Test
    fun `الشاشاتُ تمرّ بالتحويل`() {
        var dir = java.io.File("").absoluteFile
        while (!java.io.File(dir, "settings.gradle.kts").exists()) dir = dir.parentFile
        fun src(name: String) = java.io.File(dir, "ui/src/main/kotlin/com/rahalgo/ui/$name").readText()
        org.junit.Assert.assertTrue(src("WalletScreen.kt").contains("fmtWhen(iso: String): String = whenText(iso)"))
        org.junit.Assert.assertTrue(src("TicketThreadScreen.kt").contains("whenText(createdAt)"))
        org.junit.Assert.assertTrue(src("InboxSheet.kt").contains("dayText(notice.createdAt)"))
        org.junit.Assert.assertTrue(src("Chats.kt").contains("text = whenText(it)"))
        org.junit.Assert.assertTrue(src("StatementPrint.kt").contains("dateText(t.createdAt)"))
        org.junit.Assert.assertTrue(src("ChatBubble.kt").contains("DAMASCUS: java.time.ZoneId = SYRIA_ZONE"))
    }

    /** **وما لا يُفهم لا يُخترَع** — يُعرض كما وصل. */
    @Test
    fun `النصُّ القصيرُ يبقى`() {
        assertEquals("", whenText(""))
        assertEquals("abc", timeText("abc"))
    }
}
