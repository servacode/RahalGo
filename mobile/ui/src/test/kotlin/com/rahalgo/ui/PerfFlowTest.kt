package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مسارُ الطلب والقوائمُ تحت شبكةٍ رديئة** (`PF`·`PL`، ٢٠٢٦-٠٩-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وما يُحرَس هنا عقودٌ قائمةٌ أُثبتت في دفعاتٍ سابقة** — **تُقرأ في
 * دفعة الأداء لأنّ أوّلَ ما يُكسَر عند «التسريع» هو التحقّقُ قبل
 * الكتابة.**
 */
class PerfFlowTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() &&
                File(dir, "app-customer").exists()
            ) {
                return dir
            }
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        return f.readText()
    }

    private val cart = "app-customer/src/main/kotlin/com/rahalgo/customer/cart/CartScreen.kt"
    private val shop = "app-customer/src/main/kotlin/com/rahalgo/customer/shop/ShopScreen.kt"

    // ═════════════ PF-03 · PF-04 — الإرسالُ يُمسِك نفسَه ═════════════

    /**
     * **PF-03 · والضغطةُ تُرى قبل أن تصل الشبكة.**
     *
     * **PF-04 · وضغطتان طلبٌ واحد** — **والحارسان اثنان**: **رايةٌ في
     * النموذج، ومفتاحُ محاولةٍ على القرص.**
     */
    @Test
    fun `الإرسالُ يدخل حالَ الانشغال ولا يقبل ضغطتين`() {
        val src = read(cart)
        assertTrue("**سقط حارسُ الانشغال**", src.contains("if (busy) return"))
        assertTrue("**لا رايةَ انشغالٍ تُرسَم**", src.contains("enabled = !vm.busy"))
        // **ومفتاحُ المحاولة على القرص** — **فمن مات تطبيقُه قبل
        // الجواب يحمل مفتاحَه نفسَه** (`Attempt`).
        assertTrue("**ذهب مفتاحُ المحاولة**", src.contains("Attempt.key(com.rahalgo.ui.Attempt.ORDER)"))
        assertTrue("**يُرسَل بلا مفتاح**", src.contains("attemptKey = key"))
    }

    /**
     * **PF-07 · والمفتاحُ يُمحى بعد النجاح وحدَه.**
     *
     * **ومن محاه عند الفشل صنع طلباً ثانياً بمحاولةٍ ثانية** — **وهي
     * عينُ ما يمنعه العقد.**
     */
    @Test
    fun `مفتاحُ المحاولة يُمحى بعد النجاح لا قبله`() {
        val src = read(cart)
        val at = src.indexOf("Attempt.clear(com.rahalgo.ui.Attempt.ORDER)")
        assertTrue("**ذهب محوُ المفتاح**", at > 0)
        val create = src.indexOf("api.createOrder(")
        assertTrue("**المحوُ قبل الإنشاء**", create in 1 until at)
        // **والسلّةُ تُفرَّغ بعد القيد لا قبله.**
        // **وتفريغُ السلّة الذي يعنينا هو الذي يلي القيدَ** —
        // **ولزرِّ «أفرغ سلّتي» تفريغُه هو، وهو فعلُ صاحبها.**
        assertTrue("**فُرّغت السلّةُ قبل الجواب**", src.indexOf("Cart.clear()", at) > at)
    }

    // ═════════════ PF-11 · لا تسعيرَ من ذاكرةِ الجهاز ═════════════

    /**
     * **PF-11 · والسعرُ يُحسَب في المحرّك لا يُرسَل من الجهاز.**
     *
     * **وأخطرُ ما في دفعة أداءٍ أن يُحفَظ سعرٌ ليُسرَّع** — **فيُرسَل
     * طلبٌ بسعر أمس.** **والجسمُ المرسَلُ لا يحمل ثمناً أصلاً**:
     * **أصنافٌ وعنوانٌ ونقطةٌ ووسيلةُ دفع.**
     */
    @Test
    fun `الطلبُ المُرسَلُ لا يحمل ثمناً من الجهاز`() {
        val model = read("shared/src/main/kotlin/com/rahalgo/shared/customer/CustomerApi.kt")
        val at = model.indexOf("data class NewOrder(")
        assertTrue("**ذهب نموذجُ الطلب الجديد**", at > 0)
        val body = model.substring(at, model.indexOf(")\n", at))
        for (banned in listOf("price", "total", "subtotal", "deliveryFee", "discount")) {
            assertFalse(
                "**حقلُ مالٍ في جسم الطلب**: " + banned + " — **والسعرُ يُحسَب في المحرّك**",
                body.contains(banned),
            )
        }
    }

    /**
     * **والتسعيرُ يُعاد عند كلّ تبدّلٍ في السلّة أو النقطة.**
     *
     * **ولا يُخدَم من ذاكرةٍ** — **ومن سرّع الشاشةَ بحفظ آخرِ تسعيرةٍ
     * عرض سعراً لا يصمد عند الإرسال.**
     */
    @Test
    fun `التسعيرُ يُعاد عند تبدّل السلّة أو الموضع`() {
        assertTrue(
            "**سقطت إعادةُ التسعير عند التبدّل**",
            read(cart).contains("LaunchedEffect(Cart.lines, here) { vm.quote(here?.lat, here?.lng) }"),
        )
    }

    // ═════════════ PF-12 · خطأُ الجلسة ليس انقطاعَ شبكة ═════════════

    /**
     * **PF-12 · ولا يُقال «لا إنترنت» لمن انتهت جلستُه.**
     *
     * **ومن قرأ «تحقّق من اتصالك» وهو متّصلٌ أعاد المحاولةَ عشراً** —
     * **والعلاجُ دخولٌ لا شبكة.**
     */
    @Test
    fun `انتهاءُ الجلسة لا يُقرأ انقطاعَ شبكة`() {
        val src = read("ui/src/main/kotlin/com/rahalgo/ui/ApiErrors.kt")
        // **والشبكةُ تُعرَف بصنف الاستثناء** — **لا برمز المحرّك.**
        assertTrue(
            "**ذهب تمييزُ عطب الشبكة**",
            src.contains("is IOException, is HttpRequestTimeoutException, is SocketTimeoutException"),
        )
        // **وردُّ المحرّك يُترجَم برمزه** — **ورمزُ الجلسة له نصُّه.**
        assertTrue("**لا ترجمةَ لرمز المحرّك**", src.contains("e.body.code"))
        assertFalse(
            "**رُدّت أخطاءُ المحرّك كلُّها بنصّ الشبكة**",
            src.contains("return context.getString(R.string.err_network)\n    }\n    val code"),
        )
    }

    // ═════════════ PL-01 · PL-02 — القوائمُ كسولةٌ بمفاتيح ═════════════

    /**
     * **PL-01 · PL-02 · والقائمةُ الطويلةُ تُرسَم كسولةً بمفاتيحَ ثابتة.**
     *
     * **ومفتاحٌ غائبٌ يعني إعادةَ بناءِ كلّ بطاقةٍ عند كلّ تبدّل** —
     * **وصوراً تُفكّ من جديد.**
     */
    @Test
    fun `سوقُ الزبون قائمةٌ كسولةٌ بمفاتيح`() {
        val src = read(shop)
        assertTrue("**ذهبت الشبكةُ الكسولة**", src.contains("LazyVerticalGrid"))
        assertTrue("**بنودٌ بلا مفاتيح**", src.contains("items(vm.items, key = { it.id })"))
        assertTrue("**أقسامٌ بلا مفاتيح**", src.contains("items(sections, key = { it.id })"))
    }

    /**
     * **PL-05 · والقائمةُ تطلب المصغَّر لا الأصل.**
     *
     * **وصورةٌ بعرض ألفٍ في بطاقةٍ عرضُها مئةٌ تُنزَّل كلُّها ثمّ
     * تُفكّ كلُّها** — **حزمةٌ وذاكرةٌ لا تُرى.**
     */
    @Test
    fun `القوائمُ تطلب المصغَّر`() {
        assertTrue(
            "**طلبت قائمةُ السوق الأصلَ**",
            read(shop).contains("media(item.imageThumbUrl ?: item.imageUrl)"),
        )
        assertTrue(
            "**طلبت قائمةُ المفضّلة الأصلَ**",
            read("app-customer/src/main/kotlin/com/rahalgo/customer/mine/MineScreens.kt")
                .contains("media(item.imageThumbUrl ?: item.imageUrl)"),
        )
    }

    /**
     * **PL-06 · PL-07 · وذاكرةُ الصور قرصٌ ورامٌ، ولفشلها بديلٌ يُرى.**
     *
     * **وقِيس ٢٠٢٦-٠٨-٢٤**: **الشاشةُ الأولى خمسٌ وأربعون صورةً —
     * ١٫٣ ميغا** تُعاد في كلّ إقلاعٍ لولا ذاكرةُ القرص.
     */
    @Test
    fun `للصور ذاكرتان وبديلٌ عند الفشل`() {
        val img = read("ui/src/main/kotlin/com/rahalgo/ui/Images.kt")
        assertTrue("**ذهبت ذاكرةُ القرص**", img.contains("DiskCache.Builder()"))
        assertTrue("**ذهبت ذاكرةُ الرام**", img.contains("MemoryCache.Builder()"))
        val remote = read("ui/src/main/kotlin/com/rahalgo/ui/RemoteImage.kt")
        assertTrue("**لا بديلَ لصورةٍ سقطت**", remote.contains("error = { Fallback(name) }"))
    }

    /**
     * **PL-03 · وصفحاتُ الطلبات تُطلَب بصفحةٍ وحجمٍ معلومين.**
     *
     * **وقائمةٌ بلا حدٍّ تجلب تاريخَ سنةٍ في نداءٍ واحد** — **على شبكةٍ
     * ضعيفةٍ ذاك انتظارٌ طويلٌ لشاشةٍ تُقرأ أوّلُها.**
     */
    @Test
    fun `سجلُّ الطلبات مُصفَّح`() {
        assertTrue(
            "**ذهب تصفيحُ الطلبات**",
            read("shared/src/main/kotlin/com/rahalgo/shared/customer/CustomerApi.kt")
                .contains("page=\$page&per_page=30"),
        )
    }
}
