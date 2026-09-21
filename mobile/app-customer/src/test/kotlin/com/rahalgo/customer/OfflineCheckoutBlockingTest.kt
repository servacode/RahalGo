package com.rahalgo.customer

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مقطوعُ الشبكةِ لا يُرسِل طلباً** (`P8-L1-019` §7 · `CUST-12-021`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **عقدُ المالكِ المحدَّث**: عند الانقطاعِ يُمنع كلُّ تفاعلٍ يعتمد على الشبكة
 * — لا نجاحَ كاذب، لا نقرَ صامت. **والاتّصالُ حالٌ مركزيّةٌ واحدة** `Net.online`
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «مركزيّةً بصورةٍ احترافيّة»).
 *
 * **وأخطرُ تفاعلٍ هو إرسالُ الطلب** — **فزرُّ الإرسالِ يقرأ `Net.online`
 * كما يقرأ `Serving.available`**، فلا يُطلق نداءً يعلّق ثمّ يفشل. (وشريطُ
 * الانقطاعِ فوقَه يقول السبب.) **اختبارُ مصدرٍ**: العقدُ لا يُقاس بلقطة.
 */
class OfflineCheckoutBlockingTest {

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

    private val cart = "app-customer/src/main/kotlin/com/rahalgo/customer/cart/CartScreen.kt"
    private val conn = "ui/src/main/kotlin/com/rahalgo/ui/Connectivity.kt"

    /** **يُقتطع نداءُ زرِّ الإرسال** (`RahalButton` الذي يستدعي `vm.send`). */
    private fun sendButtonBlock(): String {
        val s = read(cart)
        val call = s.indexOf("vm.send(")
        assertTrue("**زرُّ الإرسال غائب**", call >= 0)
        // من بداية الـ`RahalButton(` قبله حتّى `cart_send`
        val start = s.lastIndexOf("RahalButton(", call)
        val end = s.indexOf("R.string.cart_send", call)
        assertTrue("**لم أجد حدودَ زرِّ الإرسال**", start in 0 until end)
        return s.substring(start, end)
    }

    /**
     * **CUST-12-021 · زرُّ الإرسالِ يُعطَّل بلا شبكة** — `Net.online` في
     * شرطِ `enabled`.
     */
    @Test
    fun sendIsGatedOnConnectivity() {
        val b = sendButtonBlock()
        assertTrue(
            "**زرُّ الإرسالِ لا يقرأ الاتّصالَ** — يجب `com.rahalgo.ui.Net.online` في `enabled`",
            b.contains("com.rahalgo.ui.Net.online"),
        )
        // ويبقى الحارسُ داخلَ شرطِ التمكين لا في مكانٍ آخر
        assertTrue(
            "**حارسُ الاتّصالِ ليس في شرطِ `enabled`**",
            b.contains("enabled =") && b.indexOf("enabled =") < b.indexOf("com.rahalgo.ui.Net.online"),
        )
    }

    /**
     * **الحالُ مركزيّةٌ واحدة** — `Net.online` مصدرُ الحقيقةِ الوحيد
     * (لا مراقبٌ لكلّ شاشة).
     */
    @Test
    fun connectivityIsCentralSingleSource() {
        val c = read(conn)
        assertTrue(
            "**لا حالَ اتّصالٍ مركزيّة** — يجب `object Net` و`var online`",
            c.contains("object Net") && c.contains("var online by mutableStateOf(true)"),
        )
        assertTrue(
            "**الحالُ تُكتب من خارجِ `Net`** — يجب `private set`",
            c.contains("private set"),
        )
    }
}
