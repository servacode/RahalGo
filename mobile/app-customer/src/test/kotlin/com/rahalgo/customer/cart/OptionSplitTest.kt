package com.rahalgo.customer.cart

/**
 * **عدّادٌ لكلّ خيار** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — `CART-SPL-*`.
 *
 * «٢ عادي + ١ دبل + ١ عيران» لشاورما بـ٢٠ ودبل +١٠ وعيران ٥ = ٧٥، في سطرين.
 */
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.ModifierGroup
import com.rahalgo.shared.model.ModifierOption
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test

class OptionSplitTest {

    private val shawarma = Item(id = "sh", name = "شاورما دجاج", price = 20)
    private fun opt(id: String, delta: Long) = ModifierOption(id = id, name = id, priceDelta = delta)
    private val normal = opt("normal", 0)
    private val dbl = opt("double", 10)
    private val garlic = opt("garlic", 2)
    private val ayran = opt("ayran", 5)
    private val size = ModifierGroup(id = "size", name = "الحجم", minSelect = 1, maxSelect = 1, options = listOf(normal, dbl))
    private val extras = ModifierGroup(id = "ext", name = "إضافات", minSelect = 0, maxSelect = 3, options = listOf(garlic))
    private val drinksGroup = ModifierGroup(id = "dr", name = "مشروبات", minSelect = 0, maxSelect = 5, options = listOf(ayran))

    @Before
    fun empty() = Cart.clear()

    /** CART-SPL-001 **مجموعةٌ إلزاميّةٌ واحدةٌ بخيارٍ واحد** — والمشروباتُ والاختياريّةُ لا تُحسب. */
    @Test
    fun `مجموعة العدادات`() {
        assertEquals("size", OptionSplit.group(listOf(size, extras, drinksGroup))?.id)
        assertNull(OptionSplit.group(listOf(extras, drinksGroup)))
        assertNull(OptionSplit.group(listOf(size, size.copy(id = "bread", name = "الخبز"))))
        assertNull(OptionSplit.group(listOf(size.copy(maxSelect = 2))))
        assertNotNull(OptionSplit.group(listOf(size, drinksGroup.copy(minSelect = 1))))
    }

    /** CART-SPL-002 **٢ عادي + ١ دبل + ١ عيران = ٧٥ في سطرين، والعيرانُ في أوّلهما وحدَه.** */
    @Test
    fun `سطر لكل خيار`() {
        val picks = OptionSplit.picks(
            size,
            mapOf("normal" to 2, "double" to 1),
            emptyList(),
            listOf(Cart.DrinkPick(ayran, 1)),
        )
        assertEquals(2, picks.size)
        assertEquals(listOf("normal"), picks[0].options.map { it.id })
        assertEquals(2, picks[0].qty)
        assertEquals(1, picks[0].drinks.size)
        assertEquals(listOf("double"), picks[1].options.map { it.id })
        assertEquals(0, picks[1].drinks.size)
        assertEquals(75L, OptionSplit.total(20, picks))

        Cart.addAll(shawarma, picks)
        assertEquals(2, Cart.lines.size)
        assertEquals(75L, Cart.subtotal)
        assertEquals(mapOf("ayran" to 1), Cart.lines[0].toPayload().optionQty)
        assertEquals(emptyMap<String, Int>(), Cart.lines[1].toPayload().optionQty)
    }

    /** CART-SPL-003 **الإضافةُ تلحق كلَّ سطر وتُضرب بعدده**، والصفرُ لا يصير سطراً. */
    @Test
    fun `الإضافات لكل سطر`() {
        val picks = OptionSplit.picks(size, mapOf("normal" to 0, "double" to 3), listOf(garlic), emptyList())
        assertEquals(1, picks.size)
        assertEquals(listOf("double", "garlic"), picks[0].options.map { it.id })
        assertEquals((20L + 10 + 2) * 3, OptionSplit.total(20, picks))
    }

    /** CART-SPL-004 **عددُ المشروب يُبدَّل من السلّة**، والصفرُ يرفعه وحدَه ويبقى الصنف. */
    @Test
    fun `عدد المشروب من السلة`() {
        Cart.add(shawarma, 2, listOf(normal), listOf(Cart.DrinkPick(ayran, 1)))
        Cart.setDrinkQty(Cart.lines.single().key, "ayran", 3)
        assertEquals(mapOf("ayran" to 3), Cart.lines.single().toPayload().optionQty)
        assertEquals(40L + 15, Cart.subtotal)
        Cart.setDrinkQty(Cart.lines.single().key, "ayran", 0)
        assertEquals(2, Cart.lines.single().qty)
        assertEquals(emptyMap<String, Int>(), Cart.lines.single().toPayload().optionQty)
    }
}
