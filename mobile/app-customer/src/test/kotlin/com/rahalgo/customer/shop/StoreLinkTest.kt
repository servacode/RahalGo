package com.rahalgo.customer.shop

import com.rahalgo.shared.model.Item
import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **اسمُ المتجر بابُ صفحته** (قرارُ المالك ٢٠٢٦-١٠-١٨) — في السوق والمفضّلة
 * ونافذةِ الخيارات، **ولا رابطَ إلى الصفحة التي هو فيها.**
 */
class StoreLinkTest {

    private val item = Item(id = "i1", name = "شاورما", merchantId = "m1", merchantName = "مطعم الشام")

    @Test
    fun linksToTheItemsStore() {
        assertEquals("m1", StoreLink.target(item, current = null))
    }

    @Test
    fun hiddenOnThatStoresOwnPage() {
        assertNull(StoreLink.target(item, current = "m1"))
    }

    @Test
    fun shownOnAnotherStoresPage() {
        assertEquals("m1", StoreLink.target(item, current = "m2"))
    }

    @Test
    fun hiddenWithoutIdOrName() {
        assertNull(StoreLink.target(item.copy(merchantId = ""), current = null))
        assertNull(StoreLink.target(item.copy(merchantName = null), current = null))
        assertNull(StoreLink.target(item.copy(merchantName = "  "), current = null))
    }

    private fun src(path: String): String {
        var dir = File("").absoluteFile
        repeat(6) {
            val f = File(dir, "app-customer/src/main/kotlin/com/rahalgo/customer/$path")
            if (f.exists()) return f.readText()
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد $path")
    }

    /** **المفضّلةُ والنافذةُ تمرّران الرابطَ بالقاعدة نفسِها** — لا شرطٌ ثانٍ يفترق. */
    @Test
    fun favouritesAndSheetUseTheRule() {
        val mine = src("mine/MineScreens.kt")
        assertTrue("المفضّلة بلا رابط متجر", mine.contains("onStore = com.rahalgo.customer.shop.StoreLink.target(item, null)"))
        assertTrue("نافذة العروض بلا رابط متجر", mine.contains("onStore = com.rahalgo.customer.shop.StoreLink.target(target, null)"))
        val shop = src("shop/ShopScreen.kt")
        assertTrue("نافذة السوق لا تخفي الرابط في صفحة المتجر", shop.contains("StoreLink.target(target, vm.storeId)"))
        val sheet = src("shop/ItemOptionsSheet.kt")
        assertTrue("النافذة لا تعرض اسم المتجر", sheet.contains("onStore: (() -> Unit)? = null"))
        val main = src("MainActivity.kt")
        assertTrue("المفضّلة لا تفتح صفحة المتجر", main.contains("shopVm.openStore(id)"))
    }
}
