package com.rahalgo.driver.orders

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بلاغُ الطوارئ لا يُنتِج نجاحاً غامضاً** (`DRV-DEF-001`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كان `emergency()` يُغلق البابَ ثمّ يبتلع الخطأ في `Log.w`** — فمن سقطت
 * شبكتُه ظنّ العملياتِ أُبلغت وهي لم تُبلَّغ. **والعقدُ الآن**: لا يُبتلَع خطأٌ،
 * ونجاحٌ لا يُعرَض إلّا بعد إقرارِ الخادم، وفشلٌ يُقال صراحةً مع إعادةٍ آمنة.
 *
 * **اختبارُ مصدرٍ**: عقدُ سلامةٍ لا يُقاس بلقطةِ شاشة.
 */
class EmergencyContractTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        return f.readText().replace("\r\n", "\n")
    }

    private val vm = "app-driver/src/main/kotlin/com/rahalgo/driver/orders/OrdersViewModel.kt"

    /** يقتطع جسمَ `fun emergency(`. */
    private fun emergencyBody(): String {
        val s = read(vm)
        val start = s.indexOf("fun emergency(point: LastPoint.Point?")
        assertTrue("**`emergency()` غائبة**", start >= 0)
        val end = s.indexOf("fun retryEmergency(", start)
        assertTrue("**لم أجد نهايةَ `emergency()` (`retryEmergency` غائبة؟)**", end > start)
        return s.substring(start, end)
    }

    /** **لا يُبتلَع خطأُ البلاغ** — لا `runCatching{…}.onFailure { Log`. */
    @Test
    fun failureIsNotSwallowed() {
        val b = emergencyBody()
        assertFalse(
            "**عاد ابتلاعُ الخطأ** — `runCatching{ backend.driver.emergency }.onFailure { Log`",
            b.contains("runCatching { backend.driver.emergency") ||
                (b.contains("backend.driver.emergency") && b.contains(".onFailure { Log")),
        )
    }

    /** **الفشلُ يُعرَض** — `emergencyError = describe(e)` في `catch`. */
    @Test
    fun failureSurfacesError() {
        val b = emergencyBody()
        assertTrue(
            "**الفشلُ لا يُعرَض** — يجب `emergencyError = describe(e)`",
            b.contains("emergencyError = describe(e)"),
        )
        assertTrue(
            "**لا `catch` حول النداء**",
            b.contains("try {") && b.contains("catch (e: Exception)"),
        )
    }

    /**
     * **النجاحُ بعد الإقرارِ لا قبله** — `emergencyOpen = false` يقع بعد النداءِ
     * الناجح، لا في أوّل الدالّة.
     */
    @Test
    fun successOnlyAfterAck() {
        val b = emergencyBody()
        val call = b.indexOf("backend.driver.emergency(")
        val close = b.lastIndexOf("emergencyOpen = false")
        assertTrue("**النداءُ غائب**", call >= 0)
        assertTrue(
            "**`emergencyOpen=false` قبل الإقرار** — نجاحٌ كاذبٌ محتمل",
            close > call,
        )
        // ولا يُغلَق في أوّل الدالّة (قبل النداء) كما كان
        val firstClose = b.indexOf("emergencyOpen = false")
        assertTrue(
            "**البابُ يُغلَق قبل النداء** (كالعطبِ القديم)",
            firstClose > call,
        )
    }

    /** **إعادةٌ متاحة** — `retryEmergency` بالنقطةِ والسببِ نفسِهما. */
    @Test
    fun retryIsAvailable() {
        val s = read(vm)
        assertTrue(
            "**لا `retryEmergency`**",
            s.contains("fun retryEmergency(") && s.contains("emergency(lastEmergencyPoint, lastEmergencyNote)"),
        )
    }

    /** **لا يُغلَق وهو يُرسِل** — `dismissEmergency` يحرس `emergencyBusy`. */
    @Test
    fun dismissGuardsWhileBusy() {
        val s = read(vm)
        val d = s.substring(s.indexOf("fun dismissEmergency("))
        assertTrue(
            "**`dismissEmergency` لا يحرس الإرسال** — يجب `if (emergencyBusy) return`",
            d.substring(0, 200).contains("if (emergencyBusy) return"),
        )
    }
}
