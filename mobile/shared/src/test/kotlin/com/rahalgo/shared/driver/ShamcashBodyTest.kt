package com.rahalgo.shared.driver

import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Test

/** **جسم الانتقال**: حقل `shamcash` يظهر حين يُمرَّر ويغيب حين النقد، والباقي كما كان. */
class ShamcashBodyTest {
    private val json = Json { explicitNulls = false }

    @Test
    fun `cash body has no shamcash key`() {
        val s = json.encodeToString(transitionBody("delivered", deliveryCode = "1234"))
        assertEquals("""{"to":"delivered","reason":"","note":"","delivery_code":"1234"}""", s)
    }

    @Test
    fun `shamcash body carries currency and amount`() {
        val s = json.encodeToString(transitionBody("delivered", shamcash = ShamcashPay("USD", 12.5)))
        assertEquals("""{"to":"delivered","reason":"","note":"","shamcash":{"currency":"USD","amount":12.5}}""", s)
    }
}
