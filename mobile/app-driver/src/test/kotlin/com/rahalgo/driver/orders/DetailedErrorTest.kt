package com.rahalgo.driver.orders

import com.rahalgo.ui.R
import com.rahalgo.ui.detailedErrorRes
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * **الخطأُ بأرقامه** (قرارُ المالك ٢٠٢٦-١٠-٠٣ مساءً: «منع — مع تنبيهٍ للسائق») — المسافةُ وحدّا
 * الأجرة من تفصيل الخادم، **وغيابُها يقع على النصّ العامّ.**
 */
class DetailedErrorTest {

    @Test
    fun proofTooFarSaysHowFar() {
        val (res, args) = detailedErrorRes("proof_too_far", mapOf("distance_m" to "320", "max_m" to "150"))!!
        assertEquals(R.string.err_proof_too_far_m, res)
        assertEquals(listOf("320 م"), args)
    }

    @Test
    fun feeRangeSaysBothBounds() {
        val (res, args) = detailedErrorRes("custom_fee_out_of_range", mapOf("min" to "1000", "max" to "100000"))!!
        assertEquals(R.string.err_custom_fee_between, res)
        assertEquals(listOf("1,000 ل.س", "100,000 ل.س"), args)
    }

    @Test
    fun missingDetailsFallBack() {
        assertNull(detailedErrorRes("proof_too_far", emptyMap()))
        assertNull(detailedErrorRes("proof_mocked", emptyMap()))
    }

    @Test
    fun codesHaveTheirText() {
        for (code in listOf("custom_fee_out_of_range", "custom_goods_too_high", "proof_too_far", "proof_mocked", "proof_no_location")) {
            val res = com.rahalgo.ui.resolveErrorRes(code)
            org.junit.Assert.assertNotEquals("رمزٌ بلا نصّ: $code", R.string.err_internal, res)
        }
    }
}
