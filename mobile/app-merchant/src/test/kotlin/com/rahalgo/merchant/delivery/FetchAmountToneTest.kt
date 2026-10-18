package com.rahalgo.merchant.delivery

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/** **نافذةُ مبلغ الإحضار: الموافقةُ خضراءُ والرفضُ أحمر** (طلبُ المالك ٢٠٢٦-١٠-١٨). */
class FetchAmountToneTest {
    @Test
    fun approveGreenRejectRed() {
        var dir = File("").absoluteFile
        var text: String? = null
        repeat(6) {
            val f = File(dir, "app-merchant/src/main/kotlin/com/rahalgo/merchant/delivery/FetchAmountDialog.kt")
            if (text == null && f.exists()) text = f.readText()
            dir = dir.parentFile ?: return@repeat
        }
        val s = text ?: throw AssertionError("لم أجد FetchAmountDialog.kt")
        val approve = s.substringBefore("onClick = onReject")
        assertTrue("الموافقة ليست خضراء", approve.substringAfter("onClick = onApprove").contains("tone = Tone.Success"))
        assertTrue("الرفض ليس أحمر", s.substringAfter("onClick = onReject").take(120).contains("tone = Tone.Danger"))
    }
}
