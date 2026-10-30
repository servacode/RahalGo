package com.rahalgo.ui

import org.junit.Assert.assertFalse
import org.junit.Test
import java.io.File

/**
 * **لا وصفَ للصنف** (قرارُ المالك ٢٠٢٦-١٠-٣٠): لا يُكتب في المحرّر ولا يُعرض للزبون —
 * **فلا قلبَ للصورة إلى وصفها**، ولا سطرَ وصفٍ في نافذة الصنف، **ولا يُرسَل الحقلُ**
 * فيبقى ما كُتب قبلاً في القاعدة لا يُمحى بفراغ. حراسةُ مصدرٍ كأخواتها.
 */
class ItemFlipTest {

    private fun read(rel: String): String {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "app-customer").exists()) return File(dir, rel).readText().replace("\r\n", "\n")
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile")
    }

    @Test
    fun customerNeverSeesItemDescription() {
        val card = read("app-customer/src/main/kotlin/com/rahalgo/customer/shop/ShopScreen.kt")
        val sheet = read("app-customer/src/main/kotlin/com/rahalgo/customer/shop/ItemOptionsSheet.kt")
        assertFalse("**البطاقةُ ما زالت تقلب الصورة**", card.contains("rotationY = turn"))
        assertFalse("**البطاقةُ تقرأ الوصف**", card.contains("item.description"))
        assertFalse("**النافذةُ تقرأ الوصف**", sheet.contains("item.description"))
    }

    @Test
    fun editorHasNoDescriptionField() {
        val editor = read("ui/src/main/kotlin/com/rahalgo/ui/menu/ItemEditor.kt")
        assertFalse("**المحرّرُ ما زال يكتب الوصف**", editor.contains("d.description"))
        for (vm in listOf(
            "app-merchant/src/main/kotlin/com/rahalgo/merchant/menu/MenuViewModel.kt",
            "app-rep/src/main/kotlin/com/rahalgo/rep/menu/MenuViewModel.kt",
        )) {
            assertFalse("**$vm يُرسل الوصف** — فراغُه يمحو المكتوب", read(vm).contains("description ="))
        }
    }
}
