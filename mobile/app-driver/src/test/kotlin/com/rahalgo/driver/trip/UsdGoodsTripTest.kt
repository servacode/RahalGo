package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.StorePay
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * **ثمنُ البضاعة بالدولار أو بالليرة** (قرارُ المالك ٢٠٢٦-١١-٠٩) — الدولارُ كاشٌ للكابتن والأجرةُ ليرة،
 * **ولا يُجمع الاثنان أبداً.**
 */
class UsdGoodsTripTest {

    private fun order(usd: Long?, cashDue: Long) = DriverOrder(
        id = "u1", status = "on_the_way", kind = "custom", customMode = "purchase",
        customGoods = if (usd == null) 5000 else 0, customGoodsUsd = usd, customFee = 150, cashDue = cashDue,
    )

    @Test
    fun `usd goods plus syp fee are two amounts`() {
        assertEquals("12.50$ + 150 ل.س", usdCollectText(order(1250, 150)))
    }

    @Test
    fun `wallet paid the fee so only dollars are collected`() {
        assertEquals("12$", usdCollectText(order(1200, 0)))
    }

    @Test
    fun `syp goods have no usd collect text`() {
        assertNull(usdCollectText(order(null, 5150)))
    }

    @Test
    fun `store pay text follows its currency`() {
        assertEquals("3.25$", storePayText(StorePay(amount = 325, currency = "USD")))
        assertEquals("25,000 ل.س", storePayText(StorePay(amount = 25000, currency = "")))
        assertEquals("25,000 ل.س", storePayText(StorePay(amount = 25000, currency = "SYP")))
    }
}
