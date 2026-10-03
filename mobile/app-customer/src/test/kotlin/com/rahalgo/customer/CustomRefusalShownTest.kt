package com.rahalgo.customer

import com.rahalgo.ui.R
import com.rahalgo.ui.resolveErrorRes
import java.io.File
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **رفضُ الطلب يُقال بسببه — ويبقى ظاهراً.**
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: أوقف المكتبُ المنصّةَ مؤقّتاً، فضغط الزبونُ
 *  «أرسل الطلب» في الطلب الخاصّ ولم يرَ سبباً — ومضةٌ ثانيتين ثمّ لا شيء.)
 */
class CustomRefusalShownTest {

    private fun read(rel: String): String {
        var dir = File("").absoluteFile
        while (!File(dir, "settings.gradle.kts").exists()) dir = dir.parentFile
        return File(dir, rel).readText().replace("\r\n", "\n")
    }

    /** **كلُّ رمزِ رفضٍ له جملتُه** — لا «تعذّر إتمام العملية». */
    @Test
    fun `كلُّ رمزِ رفضٍ له نصّ`() {
        for (code in listOf(
            "temporarily_unavailable", "platform_closed_now", "zone_closed_now",
            "launch_closed", "out_of_zone", "city_not_supported", "coverage_unavailable",
            "text_too_long", "text_offensive", "text_bad_chars",
        )) {
            assertNotEquals("الرمز $code بلا نصّ", R.string.err_internal, resolveErrorRes(code))
        }
    }

    /** **والسببُ يبقى تحت الزرّ** — في الطلب الخاصّ كما في السلّة. */
    @Test
    fun `سببُ الرفض يبقى تحت زرّ الإرسال`() {
        val custom = read("app-customer/src/main/kotlin/com/rahalgo/customer/custom/CustomScreen.kt")
        assertTrue(
            "**الطلبُ الخاصّ يقول السببَ ومضةً تختفي**",
            custom.contains("error = apiError(getApplication(), e)") && custom.contains("Note(vm.error"),
        )
        val cart = read("app-customer/src/main/kotlin/com/rahalgo/customer/cart/CartScreen.kt")
        assertTrue(
            "**السلّةُ لا تُبقي السبب**",
            cart.contains("error = apiError(getApplication(), e)") && cart.contains("Note(vm.error"),
        )
    }
}
