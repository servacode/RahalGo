package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عقدُ وجهةِ اللافتة** (`CUST-DEF-008` / `CUST-09-026`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ما ظُنَّ عطباً في التوصيل هو قرارُ مالكٍ قائم.** لوحةُ الإدارة **لا تملك
 * حقلَ وجهةٍ أصلاً** (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «لا يوجد داعٍ لعنوان البانر ولا
 * للزرّ أيضاً») — فكلُّ لافتةٍ `target = ""`، **ولا سبيلَ لصنع لافتةٍ ذاتِ
 * وجهةٍ من المنتَج.** والسلايدر يصدق هذا العقد: **الفارغةُ لا تُضغط.**
 *
 * **فهذا الاختبارُ يُثبّت العقدَ القائم لا يبني ميزةً**:
 *  ١ · اللافتةُ بلا وجهةٍ غيرُ قابلةٍ للضغط (`BannerSlider`).
 *  ٢ · وإن وُجدت وجهةٌ يوماً، فالضغطةُ تُسلَّم للنداء `onOpen` لا تُبتلع.
 *  ٣ · حقلُ الوجهةِ يسقط إلى `""` حين يغيب (فاللافتةُ غيرُ فعّالةٍ افتراضاً).
 *  ٤ · `ShopScreen` يُبقي سباكةَ الوجهة (`target = it.target`) فإعادةُ
 *      التفعيلِ يوماً سطرٌ لا إعادةُ سباكة.
 *  ٥ · و**اليومَ `ShopScreen` لا يمرّر `onOpen`** — فاللافتةُ غيرُ فعّالةٍ في
 *      السوق بقرار المالك؛ ومن يفعّلها لاحقاً يكسر هذا فيراجع العقدَ عمداً.
 *
 * **فالضغطةُ الصامتةُ اليومَ هي السلوكُ الصحيح، لا خللٌ يُصلَح.** وأيُّ توصيلٍ
 * فعليٍّ ميزةٌ جديدةٌ تعبر الإدارةَ والمحرّكَ والملاحة — **تحتاج تحليلاً
 * مكتوباً وإذنَ المالك، لا إصلاحاً ليليّاً.**
 */
class CustBannerTargetTest {

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

    private val slider = "ui/src/main/kotlin/com/rahalgo/ui/BannerSlider.kt"
    private val shop = "app-customer/src/main/kotlin/com/rahalgo/customer/shop/ShopScreen.kt"
    private val model = "shared/src/main/kotlin/com/rahalgo/shared/model/Shop.kt"

    /** **١ · الفارغةُ لا تُضغط** — لا `clickable` إلّا بوجهة. */
    @Test
    fun emptyTargetIsNotActionable() {
        val s = read(slider)
        assertTrue(
            "**لا حارسَ على الضغط بالوجهة** — يجب `if (b.target.isNotEmpty())`",
            s.contains("if (b.target.isNotEmpty())"),
        )
    }

    /** **٢ · وجهةٌ ⇒ تُسلَّم الضغطةُ للنداء** لا تُبتلع. */
    @Test
    fun actionableTargetDelegatesToOnOpen() {
        val s = read(slider)
        assertTrue(
            "**الضغطةُ لا تُسلَّم `onOpen`** — يجب `Modifier.clickable { onOpen(b) }`",
            s.contains("Modifier.clickable { onOpen(b) }"),
        )
    }

    /** **٣ · الوجهةُ تسقط إلى `\"\"` حين تغيب** — فاللافتةُ غيرُ فعّالةٍ افتراضاً. */
    @Test
    fun targetDefaultsToEmpty() {
        assertTrue(
            "**نموذجُ اللافتة بلا وجهةٍ افتراضيّةٍ فارغة**",
            read(model).contains("val target: String = \"\""),
        )
        assertTrue(
            "**`BannerSlide` بلا وجهةٍ افتراضيّةٍ فارغة**",
            read(slider).contains("val target: String = \"\""),
        )
    }

    /** **٤ · `ShopScreen` يُبقي سباكةَ الوجهة** — إعادةُ التفعيلِ سطرٌ لا سباكة. */
    @Test
    fun shopPreservesTargetPlumbing() {
        assertTrue(
            "**`ShopScreen` لا يمرّر وجهةَ اللافتة إلى `BannerSlide`**",
            read(shop).contains("target = it.target"),
        )
    }

    /**
     * **٥ · اللافتةُ غيرُ فعّالةٍ في السوق اليومَ** (قرارُ المالك ٢٠٢٦-٠٨-٠٩).
     *
     * **`ShopScreen` لا يمرّر `onOpen` إلى `BannerSlider`** — فكلُّ لافتةٍ
     * (والكلُّ `target=""`) غيرُ قابلةٍ للضغط. **ومن يريد توصيلاً يكسر هذا
     * الاختبارَ فيراجع العقدَ عمداً** لا مصادفة.
     */
    @Test
    fun shopDoesNotWireNavigationYet() {
        val s = read(shop)
        val call = s.substring(s.indexOf("BannerSlider("))
        val block = call.substring(0, call.indexOf(')') + 1).let {
            // نأخذ نداءَ `BannerSlider(...)` حتّى `modifier` — كافٍ لرؤية `onOpen`
            call.substring(0, minOf(call.length, call.indexOf("modifier = Modifier.padding")))
        }
        assertFalse(
            "**`ShopScreen` صار يمرّر `onOpen`** — فُعّلت اللافتةُ؛ يجب مراجعةُ عقد " +
                "`CUST-DEF-008` وقرارِ المالك ٢٠٢٦-٠٨-٠٩ صراحةً قبل قبول هذا.",
            block.contains("onOpen"),
        )
    }
}
