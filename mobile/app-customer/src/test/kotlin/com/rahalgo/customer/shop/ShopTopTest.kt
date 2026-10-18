package com.rahalgo.customer.shop

import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.StoreSection
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * **ترتيبُ ما فوق السوق وتجميعُ صفحة المتجر** (قرارُ المالك ٢٠٢٦-١٠-١٨).
 *
 *   سلايدرٌ ظاهرٌ ⇒ سلايدر · متاجر مميّزة · بحث
 *   مطفأٌ/فارغ   ⇒ متاجر مميّزة · بحث
 */
class ShopTopTest {

    @Test
    fun sliderShown_sliderThenStripThenSearch() {
        assertEquals(
            listOf(ShopBlock.Slider, ShopBlock.Strip, ShopBlock.Search),
            ShopTop.order(hasSlider = true, hasStrip = true, searching = false),
        )
    }

    @Test
    fun sliderOffOrEmpty_stripAboveSearch() {
        assertEquals(
            listOf(ShopBlock.Strip, ShopBlock.Search),
            ShopTop.order(hasSlider = false, hasStrip = true, searching = false),
        )
    }

    @Test
    fun noFeatured_noStrip() {
        assertEquals(
            listOf(ShopBlock.Slider, ShopBlock.Search),
            ShopTop.order(hasSlider = true, hasStrip = false, searching = false),
        )
        assertEquals(listOf(ShopBlock.Search), ShopTop.order(hasSlider = false, hasStrip = false, searching = false))
    }

    @Test
    fun searching_onlySearch() {
        assertEquals(listOf(ShopBlock.Search), ShopTop.order(hasSlider = true, hasStrip = true, searching = true))
    }

    private fun sec(id: String, vararg items: String) =
        StoreSection(id = id, name = id, items = items.map { Item(id = it, sectionId = id) })

    @Test
    fun storeMenu_skipsEmptySections_andKeepsOrder() {
        val got = StoreMenu.visible(listOf(sec("b", "1"), sec("empty"), sec("a", "2", "3")))
        assertEquals(listOf("b", "a"), got.map { it.id })
    }

    @Test
    fun storeMenu_headerIndex_countsHeaderAndItems() {
        val sections = listOf(sec("a", "1", "2"), sec("empty"), sec("b", "3"), sec("c", "4", "5", "6"))
        // [أ] 1 2 [ب] 3 [ج] 4 5 6
        assertEquals(mapOf("a" to 0, "b" to 3, "c" to 5), StoreMenu.headerIndex(sections))
        assertEquals(mapOf("a" to 2, "b" to 5, "c" to 7), StoreMenu.headerIndex(sections, leading = 2))
    }

    @Test
    fun storeMenu_sectionAt_followsScroll() {
        val sections = listOf(sec("a", "1", "2"), sec("b", "3"), sec("c", "4", "5", "6"))
        assertEquals("a", StoreMenu.sectionAt(sections, 0))
        assertEquals("a", StoreMenu.sectionAt(sections, 2))
        assertEquals("b", StoreMenu.sectionAt(sections, 3))
        assertEquals("b", StoreMenu.sectionAt(sections, 4))
        assertEquals("c", StoreMenu.sectionAt(sections, 5))
        assertEquals("c", StoreMenu.sectionAt(sections, 99))
        assertNull(StoreMenu.sectionAt(emptyList(), 0))
    }
}
