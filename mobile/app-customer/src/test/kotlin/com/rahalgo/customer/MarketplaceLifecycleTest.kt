package com.rahalgo.customer

import com.rahalgo.shared.model.Section
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **دورةُ حياة السوق في الشاشة** (`ML`، ٢٠٢٦-٠٩-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما وقع على جهازٍ حقيقيّ
 *
 * **وفُتح التطبيقُ في الرقّة فرأى تسعةَ أقسامٍ كلُّها تقول «لا أصناف في
 * هذا القسم بعد» وتحتها «أعد المحاولة».** **ونُقر الزرُّ خمساً فلم
 * يتبدّل حرف.**
 *
 * # القرار (المالك ٢٠٢٦-٠٩-١٦)
 *
 * **بنيةُ السوق تُبنى على وجود محتوىً يخصّ مدينةَ الزبون** — **لا على
 * كون المتجر مفتوحاً في هذه الدقيقة.** **وإغلاقُ الساعة يُقال ولا
 * يمحو قسماً.**
 */
class MarketplaceLifecycleTest {

    private companion object {
        fun sec(name: String, count: Int, orderable: Int = 0) =
            Section(id = name, name = name, count = count, orderableNow = orderable)

        fun read(rel: String): String =
            java.io.File(appModuleDir(), rel).readText().replace("\r\n", "\n")

        fun appModuleDir(): java.io.File {
            var dir = java.io.File("").absoluteFile
            repeat(6) {
                if (java.io.File(dir, "build.gradle.kts").exists() &&
                    java.io.File(dir, "src/main").exists()
                ) {
                    return dir
                }
                dir = dir.parentFile ?: return@repeat
            }
            throw IllegalStateException("لم أجد مجلَّد الوحدة")
        }

        const val SCREEN = "src/main/kotlin/com/rahalgo/customer/shop/ShopScreen.kt"
        const val VM = "src/main/kotlin/com/rahalgo/customer/shop/ShopViewModel.kt"
    }

    // ── البنيةُ تُبنى على الوجود لا على الدوام ───────────────────────

    /** **وقسمٌ بلا محتوىً لا يُعرَض.** */
    @Test
    fun `الأقسامُ الخاويةُ لا تُعرَض`() {
        val all = listOf(sec("شاورما", 5), sec("برغر", 0), sec("بقالة", 1))
        val visible = all.filter { it.count > 0 }
        assertEquals(listOf("شاورما", "بقالة"), visible.map { it.name })
    }

    /**
     * **ومتجرٌ نائمٌ يبقى قسمُه** — **وهو لبُّ قرار المالك.**
     *
     * **ولو بُنيت البنيةُ على «يُطلب الآن» لاختفى السوقُ كلَّ ليلة.**
     */
    @Test
    fun `إغلاقُ الساعة لا يمحو قسماً`() {
        val night = listOf(sec("شاورما", 5, orderable = 0), sec("مشاوي", 6, orderable = 0))
        val visible = night.filter { it.count > 0 }
        assertEquals(
            "**اختفى السوقُ لأنّ متاجرَه نائمة**",
            2, visible.size,
        )
    }

    /** **وسوقٌ لم تمتلئ بعد** — **لا قسمَ فيه محتوى.** */
    @Test
    fun `السوقُ الفارغةُ تُعرَف بالمحتوى لا بعدد الأقسام`() {
        val nine = List(9) { sec("قسم$it", 0) }
        assertTrue("**تسعةُ أقسامٍ خاويةٍ ليست سوقاً عامرة**", nine.all { it.count == 0 })
        assertTrue(nine.isNotEmpty() && nine.none { it.count > 0 })
    }

    // ── والشاشةُ تُبنى على ذلك فعلاً ──────────────────────────────────

    /** **ولا يُقرأ الشرطُ من عدد الأقسام** — **فهي لا تفرغ أبداً.** */
    @Test
    fun `الشاشةُ تسأل عن المحتوى لا عن عدد الأقسام`() {
        val screen = read(SCREEN)
        assertTrue(
            "**ذهب شرطُ السوق الفارغة من الشاشة**",
            screen.contains("vm.marketEmpty"),
        )
        assertTrue(
            "**الشريطُ يعرض الأقسامَ كلَّها لا الظاهرةَ منها**",
            screen.contains("vm.visibleSections"),
        )
        val vm = read(VM)
        assertTrue(
            "**ذهب حسابُ الأقسام الظاهرة**",
            vm.contains("visibleSections") && vm.contains("it.count > 0"),
        )
        assertFalse(
            "**بنيةُ السوق بُنيت على «يُطلب الآن»** — **فتختفي كلَّ ليلة**",
            vm.contains("orderableNow > 0"),
        )
    }

    /**
     * **ولا زرَّ إعادةٍ على نجاحٍ بصفر.**
     *
     * **والردُّ `200` بقائمةٍ فارغةٍ حتميّ** — **وإعادتُه تردّ جوابَه
     * عينَه.** **وزرٌّ لا يغيّر شيئاً يُضغط ثمّ يُفقَد الرجاءُ بالتطبيق.**
     */
    @Test
    fun `لا إعادةَ محاولةٍ في حال الفراغ`() {
        val screen = read(SCREEN)
        val i = screen.indexOf("vm.marketEmpty")
        check(i > 0) { "**لم أجد حالَ السوق الفارغة**" }
        // **وما بين هذه الحال ونهاية فروع الفراغ لا `act_retry`.**
        val tail = screen.substring(i, minOf(i + 2400, screen.length))
        assertFalse(
            "**زرُّ إعادةٍ في حال فراغٍ ناجح** — **ولا يغيّر شيئاً أبداً**",
            tail.contains("act_retry"),
        )
    }

    /** **والخطأُ يبقى حالاً مستقلّةً — وفيها الإعادةُ تنفع.** */
    @Test
    fun `الخطأُ حالٌ مستقلّةٌ وفيها إعادة`() {
        val screen = read(SCREEN)
        assertTrue(
            "**ذهبت حالُ الخطأ** — **فيُقرأ انقطاعُ الشبكة نفادَ بضاعة**",
            screen.contains("vm.error.isNotEmpty()") && screen.contains("onRetry = vm::load"),
        )
    }
}
