package com.rahalgo.customer.orders

import com.rahalgo.shared.customer.MyOrderLine
import com.rahalgo.shared.customer.MyOrderOption
import com.rahalgo.shared.model.Item
import com.rahalgo.shared.model.ItemDetail
import com.rahalgo.shared.model.ModifierGroup
import com.rahalgo.shared.model.ModifierOption
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/** **«اطلب مرة تانية»**: السطرُ يطابق كتالوجَ اليوم، وما ما عاد متوفراً يُترك (قرارُ المالك ٢٠٢٦-١١-١١). */
class ReorderPlanTest {
    private fun detail(available: Boolean = true, optAvailable: Boolean = true) = ItemDetail(
        item = Item(id = "i1", name = "برغر", price = 30, available = available),
        modifiers = listOf(
            ModifierGroup(id = "g1", name = "الحجم", options = listOf(ModifierOption(id = "o1", name = "كبير", available = optAvailable))),
            ModifierGroup(
                id = "g2",
                name = ModifierGroup.DRINKS_GROUP,
                options = listOf(ModifierOption(id = "d1", name = "كولا", priceDelta = 10)),
            ),
        ),
    )

    private val line = MyOrderLine(id = "l1", menuItemId = "i1", name = "برغر", qty = 2, options = listOf(MyOrderOption(id = "o1", name = "كبير")))

    @Test
    fun matchesOptionsAndKeepsQty() {
        val p = ReorderPlan(line, emptyList(), detail())
        assertNotNull(p)
        assertEquals(2, p!!.pick.qty)
        assertEquals(listOf("o1"), p.pick.options.map { it.id })
    }

    @Test
    fun unavailableItemIsSkipped() {
        assertNull(ReorderPlan(line, emptyList(), detail(available = false)))
    }

    @Test
    fun unavailableOptionSkipsTheLine() {
        assertNull(ReorderPlan(line, emptyList(), detail(optAvailable = false)))
    }

    @Test
    fun lostDrinkKeepsLineButFlags() {
        val gone = MyOrderLine(id = "l2", name = "ماء", qty = 1, drinkFor = "l1", drinkOptionId = "zzz")
        val p = ReorderPlan(line, listOf(gone), detail())!!
        assertTrue(p.lostDrink)
        assertTrue(p.pick.drinks.isEmpty())
        val ok = MyOrderLine(id = "l3", name = "كولا", qty = 2, drinkFor = "l1", drinkOptionId = "d1")
        assertEquals(2, ReorderPlan(line, listOf(ok), detail())!!.pick.drinks.single().qty)
    }
}

private fun ReorderPlan(line: MyOrderLine, drinks: List<MyOrderLine>, d: ItemDetail) = Reorder.plan(line, drinks, d)
