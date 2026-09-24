package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حرّاسُ الدفعة الثالثة (ب · ج) — تصحيحُ الإتاحة اللحظيّ واستعراضُ السوق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **مسحُ مصدرٍ لا واجهةٍ حيّة** — كـ`AbuseMatrixTest`: يُثبَّت أنّ الأسلاكَ
 * قائمةٌ في المصدر فلا تُنزَع بإصلاحٍ لاحق. (السلوكُ الحيُّ يُشهَد على الجهاز.)
 */
class Batch3WiringTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        while (true) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile ?: break
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    private fun read(rel: String): String = File(mobileRoot(), rel).readText()

    // ── 3b — التصحيحُ اللحظيّ للإتاحة ─────────────────────────────────

    @Test
    fun `3b عودةُ الوصلة تُصحّح إجباريّاً`() {
        val shell = read("ui/src/main/kotlin/com/rahalgo/ui/ShellViewModel.kt")
        // onState عند القيام يُنعش ويرفع الإشارة — فالأحداثُ الفائتةُ تُصحَّح.
        assertTrue("**عودةُ الوصلة لا تُنعش**", shell.contains("if (up) {") && shell.contains("Refresh.bump()"))
    }

    @Test
    fun `3b إشارةُ اللوحة تُصحّح حالَ المنصّة إجباريّاً`() {
        val main = read("app-customer/src/main/kotlin/com/rahalgo/customer/MainActivity.kt")
        assertTrue(
            "**Refresh.tick لا يُصحّح Serving إجباريّاً**",
            main.contains("Refresh.tick.collect { Serving.refresh(scope, force = true) }"),
        )
        assertTrue(
            "**العودةُ لا تُصحّح Serving إجباريّاً**",
            main.contains("Serving.refresh(scope, force = true)"),
        )
    }

    @Test
    fun `3b السوقُ يُصحّح إتاحةَ النقطة على الإشارة والعودة`() {
        val shop = read("app-customer/src/main/kotlin/com/rahalgo/customer/shop/ShopScreen.kt")
        assertTrue(
            "**السوقُ لا يُصحّح الإتاحةَ على الإشارة**",
            shop.contains("serviceVm.refreshAt(point, ctx0, force = true)"),
        )
        assertTrue("**لا مراقبَ عودةٍ في السوق**", shop.contains("Lifecycle.Event.ON_RESUME"))
    }

    @Test
    fun `3b Serving وOrderable يقبلان التجديدَ الإجباريّ`() {
        val serving = read("app-customer/src/main/kotlin/com/rahalgo/customer/Serving.kt")
        assertTrue("**Serving.refresh بلا force**", serving.contains("fun refresh(scope: kotlinx.coroutines.CoroutineScope, force: Boolean"))
        assertTrue("**refreshIfStale لم يعد يفوّض**", serving.contains("refreshIfStale") && serving.contains("stale("))
        val precart = read("app-customer/src/main/kotlin/com/rahalgo/customer/PreCart.kt")
        assertTrue("**refreshAt بلا force**", precart.contains("force: Boolean = false"))
        assertTrue("**force لا يتجاوز العمر**", precart.contains("if (!force && !Orderable.stale(point, now)) return"))
    }

    // ── 3c — استعراضُ سوقِ المدينةِ المُطلَقة (للقراءة فقط) ─────────────

    @Test
    fun `3c الاستعراضُ للقراءة فقط لا يضيف`() {
        val shop = read("app-customer/src/main/kotlin/com/rahalgo/customer/shop/ShopScreen.kt")
        assertTrue("**الاستعراضُ يضيف**", shop.contains("if (previewing) return@ItemCard"))
        assertTrue("**الإضافةُ لا تُحجَب في الاستعراض**", shop.contains("action == AddAction.BLOCKED || previewing"))
        assertTrue("**لا لافتةَ استعراضٍ دائمة**", shop.contains("preview_banner_named"))
        assertTrue("**لا خروجَ من الاستعراض**", shop.contains("CityScope.exitPreview(context)"))
    }

    @Test
    fun `3c CityScope يحمل رايةَ الاستعراض ويُثبّتها`() {
        val cs = read("app-customer/src/main/kotlin/com/rahalgo/customer/CityScope.kt")
        assertTrue("**لا رايةَ استعراض**", cs.contains("var preview by mutableStateOf(false)"))
        assertTrue("**لا دخولَ استعراض**", cs.contains("fun enterPreview("))
        assertTrue("**لا خروجَ استعراض**", cs.contains("fun exitPreview("))
        assertTrue("**الرايةُ لا تبقى بين الجلسات**", cs.contains("KEY_PREVIEW"))
        assertTrue("**لا مدينةَ رائدة**", cs.contains("fun flagship("))
    }

    @Test
    fun `3c الزرّان متمايزان بالنصّ الصحيح`() {
        val sr = read("ui/src/main/kotlin/com/rahalgo/ui/ServiceReason.kt")
        // «أخبرني» نصٌّ ثابتٌ للـINTEREST (لا variant بالاسم).
        assertTrue("**INTEREST لا يستعمل النصَّ الثابت**", sr.contains("KIND_INTEREST -> ctx.getString(R.string.cta_notify_me)"))
        val strings = read("ui/src/main/res/values/strings.xml")
        assertTrue("**نصُّ «اطلب تغطية منطقتي» غيرُ دقيق**", strings.contains(">اطلب تغطية منطقتي<"))
        assertTrue("**نصُّ «أخبرني عند وصول رحال غو» غيرُ دقيق**", strings.contains(">أخبرني عند وصول رحال غو<"))
        assertTrue("**لا نصَّ استعراضٍ**", strings.contains("preview_browse_cta_named"))
    }
}
