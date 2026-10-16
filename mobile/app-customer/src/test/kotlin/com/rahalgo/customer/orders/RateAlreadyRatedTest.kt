package com.rahalgo.customer.orders

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **«قيّمتَه من قبل» تُغلق نافذةَ التقييم** (٢٠٢٦-١٠-١٧) — كان ردُّ `already_rated` يُعرض خطأً
 * وتبقى النافذةُ مفتوحةً، فتعود بعد كلّ «إرسال».
 */
class RateAlreadyRatedTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        while (dir.parentFile != null) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    @Test
    fun `already rated closes the dialog and marks the order rated`() {
        val src = File(mobileRoot(), "app-customer/src/main/kotlin/com/rahalgo/customer/orders/OrdersViewModel.kt")
            .readText()
        val rate = src.substringAfter("fun rate(orderId: String").substringBefore("\n    }\n")
        assertTrue("لا معالجةَ لـ already_rated في rate()", rate.contains("\"already_rated\""))
        val branch = rate.substringAfter("\"already_rated\"")
        assertTrue("already_rated لا يُغلق النافذة", branch.substringBefore("} else {").contains("askRate = null"))
        assertTrue("already_rated لا يُحسب مقيَّماً", branch.substringBefore("} else {").contains("rated = rated + orderId"))
    }
}
