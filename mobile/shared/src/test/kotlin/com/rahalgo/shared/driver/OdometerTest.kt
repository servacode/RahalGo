package com.rahalgo.shared.driver

import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * **قراءةُ العدّاد في التطبيق** (قرارُ المالك ٢٠٢٦-١١-٠٥) — K1 · K2 · K3.
 *
 * الرقمُ كما يكتبه السائق (عربيّاً أو غربيّاً)، وجسمُ مفتاح الدوام الذي لا يكسر المحرّك
 * القديم، **ومتى تُطلب الصورة** بحكم المحرّك.
 */
class OdometerTest {

    // ── K1 · الرقمُ كما يُكتب ───────────────────────────────────────────

    @Test
    fun `الأرقام العربية والفواصل تُوحَّد`() {
        assertEquals("1250", OdometerInput.accept("١٢٥٠"))
        assertEquals("1250", OdometerInput.accept("۱۲۵۰"))
        assertEquals("1250.5", OdometerInput.accept("1250,5"))
        assertEquals("1250.5", OdometerInput.accept("1250٫5"))
        assertEquals("1250.", OdometerInput.accept("1250."))
        assertEquals("", OdometerInput.accept(""))
    }

    @Test
    fun `ما لا يُقبل يُردّ أثناء الكتابة`() {
        assertNull("حرف", OdometerInput.accept("12a"))
        assertNull("سالب", OdometerInput.accept("-5"))
        assertNull("فاصلان", OdometerInput.accept("12.3.4"))
        assertNull("منزلتان", OdometerInput.accept("12.34"))
        assertNull("فاصلٌ أوّلاً", OdometerInput.accept(".5"))
        assertNull("ثماني خانات", OdometerInput.accept("12345678"))
        assertEquals("1234567.8", OdometerInput.accept("1234567.8"))
    }

    @Test
    fun `النصّ المرسَل`() {
        assertEquals("1250", OdometerInput.normalize(" 1250. "))
        assertNull(OdometerInput.normalize(""))
        assertNull(OdometerInput.normalize("abc"))
        assertEquals("1250", OdometerInput.show(1250.0))
        assertEquals("1250.5", OdometerInput.show(1250.5))
    }

    // ── K2 · جسمُ مفتاح الدوام ─────────────────────────────────────────

    /** **نسخةُ الإعداد كما في `ApiClient` حرفاً.** */
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
    }

    @Test
    fun `جسم الدوام بلا عدّاد هو جسم النسخة القديمة`() {
        assertEquals("""{"on":true}""", json.encodeToString(ShiftBody(true)))
        assertEquals("""{"on":false}""", json.encodeToString(ShiftBody(false)))
    }

    @Test
    fun `جسم الدوام يعلن أن التطبيق يعرف ورقة العدّاد`() {
        // **منه وحدَه يُطبّق المحرّكُ «إلزامي»** — لا من رقم النسخة.
        assertEquals(
            """{"on":true,"odometer_id":"r1","odometer_capable":true}""",
            json.encodeToString(ShiftBody(true, "r1", odometerCapable = true)),
        )
        assertEquals(
            """{"on":false,"odometer_capable":true}""",
            json.encodeToString(ShiftBody(false, odometerCapable = true)),
        )
    }

    @Test
    fun `جسم الدوام بقراءة وبخروج`() {
        assertEquals("""{"on":true,"odometer_id":"r1"}""", json.encodeToString(ShiftBody(true, "r1")))
        assertEquals(
            """{"on":false,"odometer_id":"r2","via":"logout"}""",
            json.encodeToString(ShiftBody(false, "r2", "logout")),
        )
    }

    // ── K3 · متى تُطلب الصورة ──────────────────────────────────────────

    @Test
    fun `مطفأ — بلا ورقة`() {
        assertEquals(OdometerFlow.Direct, OdometerFlow.onToggle(true, "off", false, 0))
        assertEquals(OdometerFlow.Direct, OdometerFlow.onToggle(false, "off", false, 0))
    }

    @Test
    fun `يطلب — ورقة بتخطٍّ`() {
        assertEquals(OdometerFlow.Sheet("start", true), OdometerFlow.onToggle(true, "ask", false, 0))
        assertEquals(OdometerFlow.Sheet("end", true), OdometerFlow.onToggle(false, "ask", false, 0))
    }

    @Test
    fun `إلزامي — بلا تخطٍّ عند الفتح ولا عند السكّر`() {
        assertEquals(OdometerFlow.Sheet("start", false), OdometerFlow.onToggle(true, "require", false, 0))
        assertEquals(OdometerFlow.Sheet("end", false), OdometerFlow.onToggle(false, "require", false, 0))
        // سماحُ العمليّات للفتح وحده
        assertEquals(OdometerFlow.Direct, OdometerFlow.onToggle(true, "require", true, 0))
        assertEquals(OdometerFlow.Sheet("end", false), OdometerFlow.onToggle(false, "require", true, 0))
    }

    @Test
    fun `سكّرٌ وبيده طلب — المحرّك يردّه قبل أيّ صورة`() {
        assertEquals(OdometerFlow.Direct, OdometerFlow.onToggle(false, "require", false, 2))
        assertEquals(OdometerFlow.Direct, OdometerFlow.onLogout(true, "require", 1))
    }

    @Test
    fun `الخروج من التطبيق`() {
        assertEquals(OdometerFlow.Direct, OdometerFlow.onLogout(false, "require", 0))
        assertEquals(OdometerFlow.Sheet("end", false), OdometerFlow.onLogout(true, "require", 0))
        assertEquals(OdometerFlow.Sheet("end", true), OdometerFlow.onLogout(true, "ask", 0))
    }
}
