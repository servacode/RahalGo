package com.rahalgo.customer.cart

/**
 * **المشروباتُ بأعدادها** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — `CART-DRK-*`.
 *
 * المشروبُ يُحسب بعدده هو لا بعدد الصنف، ويُرسَل في `option_qty`.
 * **والسلّةُ المحفوظةُ قبل هذا تبقى بسلوكها القديم** — لا يُرسل لها `option_qty`.
 */
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.ModifierGroup
import com.rahalgo.shared.model.ModifierOption
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class CartDrinksTest {

    private val shawarma = Item(id = "sh", name = "شاورما دجاج", price = 20)
    private fun opt(id: String, delta: Long) = ModifierOption(id = id, name = id, priceDelta = delta)
    private val normal = opt("normal", 0)
    private val dbl = opt("double", 10)
    private val ayran = opt("ayran", 5)
    private val pepsi = opt("pepsi", 7)
    private val kinza = opt("kinza", 6)
    private val mirinda = opt("mirinda", 7)

    @Before
    fun empty() = Cart.clear()

    /** CART-DRK-001 **سيناريو #1177**: ٢ عادي + عيران وببسي، ٣ دبل + كنزا وعيران وميرندا = ١٦٠. */
    @Test
    fun `المشروبُ لا يُضرب بعدد الصنف`() {
        Cart.add(shawarma, 2, listOf(normal), listOf(Cart.DrinkPick(ayran, 1), Cart.DrinkPick(pepsi, 1)))
        Cart.add(shawarma, 3, listOf(dbl), listOf(Cart.DrinkPick(kinza, 1), Cart.DrinkPick(ayran, 1), Cart.DrinkPick(mirinda, 1)))
        assertEquals(160L, Cart.subtotal)
        assertEquals(5, Cart.count)
    }

    /** CART-DRK-002 **والحمولةُ تحمل `option_qty` ولا تحمل المشروبَ في `option_ids`.** */
    @Test
    fun `الحمولةُ تحملُ عددَ المشروب`() {
        Cart.add(shawarma, 3, listOf(dbl), listOf(Cart.DrinkPick(ayran, 2)))
        val p = Cart.lines.single().toPayload()
        assertEquals(listOf("double"), p.optionIds)
        assertEquals(mapOf("ayran" to 2), p.optionQty)
    }

    /** CART-DRK-003 **وبلا مشروبٍ يُرسَل `option_qty` فارغاً لا غائباً.** */
    @Test
    fun `بلا مشروبٍ يُرسَلُ فارغاً`() {
        Cart.add(shawarma, 1, listOf(normal))
        assertEquals(emptyMap<String, Int>(), Cart.lines.single().toPayload().optionQty)
    }

    /** CART-DRK-004 **والسطرُ القديمُ لا يُرسَل له `option_qty`** — يبقى سعرُه كما رآه. */
    @Test
    fun `السطرُ القديمُ بسلوكه القديم`() {
        val legacy = Cart.Line(shawarma, 2, listOf(normal, ayran), legacy = true)
        assertNull(legacy.toPayload().optionQty)
        assertEquals(50L, legacy.total)
    }

    /** CART-DRK-005 **ومجموعةُ «مشروبات» تُعرف باسمها حين لا علَم.** */
    @Test
    fun `مجموعةُ المشروبات تُعرف`() {
        assertTrue(ModifierGroup(name = "مشروبات").isDrinks)
        assertTrue(!ModifierGroup(name = "الحجم").isDrinks)
    }
}
