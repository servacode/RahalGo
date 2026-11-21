package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.StorePay
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

/**
 * **نافذةُ العمولة** (قرارُ المالك ٢٠٢٦-١١-٢٢) — بتطلع بس للمتجر اللي عليه عمولة وفي مبلغ ينطلو،
 * وبترجع تطلع إذا تغيّرت العمولة.
 */
class CommissionNoticeTest {

    private val withCut = StorePay(merchantId = "m1", merchantName = "أ", amount = 9000, gross = 10000, cut = 1000, cutPercent = 10.0)
    private val noCut = StorePay(merchantId = "m2", merchantName = "ب", amount = 5000, gross = 5000, cut = 0)

    @Test
    fun `only stores with a commission get the notice`() {
        val order = DriverOrder(id = "o1", payMerchants = listOf(noCut, withCut))
        assertEquals(listOf(withCut), CommissionSeen.pending(order))
        assertEquals(emptyList<StorePay>(), CommissionSeen.pending(DriverOrder(id = "o2", payMerchants = listOf(noCut))))
    }

    @Test
    fun `a changed commission is a new notice`() {
        val k1 = CommissionSeen.key("o1", withCut)
        assertNotEquals(k1, CommissionSeen.key("o1", withCut.copy(cut = 1500, cutPercent = 15.0)))
        assertNotEquals(k1, CommissionSeen.key("o2", withCut))
        assertEquals(k1, CommissionSeen.key("o1", withCut.copy(merchantName = "اسم تاني")))
    }

    @Test
    fun `percent reads plainly`() {
        assertEquals("10", commissionPercentText(10.0))
        assertEquals("1.5", commissionPercentText(1.5))
    }
}
