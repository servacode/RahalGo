package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverEditRequest
import com.rahalgo.shared.model.DriverItem
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.PickupStop
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **«طلب تعديل» بالطلب العادي** (قرار المالك ٢٠٢٦-١١-١٠) — حسبة المجموع التقريبي بالنافذة، ومتى
 * بيُبعت، ومتى الزرّ ظاهر، وحالة البطاقة.
 */
class EditRequestTripTest {

    private fun order(status: String = "picked_up", kind: String = "standard") = DriverOrder(
        id = "o1", status = status, kind = kind, total = 70_000,
        items = listOf(
            DriverItem(id = "a", name = "برغر", qty = 1, unitPrice = 50_000),
            DriverItem(id = "b", name = "كولا", qty = 2, unitPrice = 5_000),
            DriverItem(id = "r", name = "مرتجع", qty = 1, unitPrice = 0, returned = true),
        ),
    )

    @Test
    fun `missing part of a line lowers the total by its price`() {
        val d = EditDraft(lines = mapOf("b" to EditLineDraft(EditLineAction.MISSING, 1)))
        assertEquals(-5_000, editDraftDelta(order(), d))
        assertEquals(65_000, editDraftTotal(order(), d))
        assertTrue(editDraftValid(order(), d))
    }

    @Test
    fun `cancelled line and an added item add up`() {
        val d = EditDraft(
            lines = mapOf("b" to EditLineDraft(EditLineAction.CANCELLED, 2)),
            adds = listOf(EditAddDraft("خبز", 3, 1_000)),
        )
        assertEquals(63_000, editDraftTotal(order(), d))
        val body = d.toBody()
        assertEquals("cancelled", body.lines.single().action)
        assertEquals(2, body.lines.single().qty)
        assertEquals(3_000, body.adds.sumOf { it.qty * it.price })
    }

    @Test
    fun `nothing chosen, too many, or an empty order is not sent`() {
        assertFalse(editDraftValid(order(), EditDraft()))
        assertFalse(editDraftValid(order(), EditDraft(lines = mapOf("b" to EditLineDraft(EditLineAction.MISSING, 3)))))
        val all = EditDraft(
            lines = mapOf(
                "a" to EditLineDraft(EditLineAction.CANCELLED, 1),
                "b" to EditLineDraft(EditLineAction.CANCELLED, 2),
            ),
        )
        assertFalse(editDraftValid(order(), all))
        // **ومع صنف مضاف ما بيضل فاضي.**
        assertTrue(editDraftValid(order(), all.copy(adds = listOf(EditAddDraft("خبز", 1, 500)))))
        assertFalse(editDraftValid(order(), EditDraft(adds = listOf(EditAddDraft(" ", 1, 500)))))
        assertFalse(editDraftValid(order(), EditDraft(lines = mapOf("r" to EditLineDraft(EditLineAction.MISSING, 1)))))
    }

    @Test
    fun `the total never goes below zero`() {
        val d = EditDraft(lines = mapOf("a" to EditLineDraft(EditLineAction.MISSING, 1)))
        assertEquals(0, editDraftTotal(order().copy(total = 1_000), d))
    }

    @Test
    fun `button shows from accept to the door on standard orders only`() {
        for (s in listOf("assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff")) {
            assertTrue(s, canRequestEdit(order(s)))
        }
        assertFalse(canRequestEdit(order("delivered")))
        assertFalse(canRequestEdit(order("dispatching")))
        assertFalse(canRequestEdit(order(kind = "custom")))
    }

    @Test
    fun `card badge follows the office decision`() {
        val o = order()
        assertEquals(EditRequestBadge.NONE, editRequestBadge(o, null))
        assertEquals(EditRequestBadge.PENDING, editRequestBadge(o, DriverEditRequest(status = "pending")))
        assertEquals(EditRequestBadge.REJECTED, editRequestBadge(o, DriverEditRequest(status = "rejected")))
        assertEquals(EditRequestBadge.STALE, editRequestBadge(o, DriverEditRequest(status = "stale")))
        assertEquals(EditRequestBadge.NONE, editRequestBadge(o, DriverEditRequest(status = "approved")))
        assertEquals(EditRequestBadge.NONE, editRequestBadge(order("delivered"), DriverEditRequest(status = "pending")))
    }

    @Test
    fun `whatsapp amount can be corrected after pickup until the door`() {
        fun wa(s: String) = DriverOrder(id = "w", status = s, kind = "merchant_delivery", origin = "whatsapp")
        assertFalse(whatsAppAmountEditable(wa("at_pickup")))
        assertTrue(whatsAppAmountEditable(wa("picked_up")))
        assertTrue(whatsAppAmountEditable(wa("on_the_way")))
        assertTrue(whatsAppAmountEditable(wa("at_dropoff")))
        assertFalse(whatsAppAmountEditable(wa("delivered")))
        assertFalse(whatsAppAmountEditable(DriverOrder(status = "picked_up", kind = "merchant_delivery")))
    }

    @Test
    fun `pending request blocks delivery and withdrawing frees it`() {
        val o = order("at_dropoff")
        assertTrue(editRequestBlocksDelivery(o, DriverEditRequest(status = "pending")))
        assertFalse(editRequestBlocksDelivery(o, null))
        assertFalse(editRequestBlocksDelivery(o, DriverEditRequest(status = "withdrawn")))
        assertFalse(editRequestBlocksDelivery(o, DriverEditRequest(status = "rejected")))
        assertFalse(editRequestBlocksDelivery(o, DriverEditRequest(status = "approved")))
    }

    @Test
    fun `store picker shows only when the order has more than one store`() {
        assertTrue(editStoreChoices(order()).isEmpty())
        val one = order().copy(stops = listOf(PickupStop(merchantId = "m1", name = "أ")))
        assertTrue(editStoreChoices(one).isEmpty())
        val two = order().copy(
            stops = listOf(
                PickupStop(merchantId = "m1", name = "أ"),
                PickupStop(merchantId = "m2", name = "ب"),
                PickupStop(merchantId = "m3", name = "ج", status = "removed"),
            ),
        )
        assertEquals(listOf("m1", "m2"), editStoreChoices(two).map { it.merchantId })
    }

    @Test
    fun `chosen store goes into the request body`() {
        val d = EditDraft(adds = listOf(EditAddDraft("ماء", 1, 500, "m2", "ب"), EditAddDraft("خبز", 1, 700)))
        val body = d.toBody()
        assertEquals("m2", body.adds[0].merchantId)
        assertEquals(null, body.adds[1].merchantId)
    }
}
