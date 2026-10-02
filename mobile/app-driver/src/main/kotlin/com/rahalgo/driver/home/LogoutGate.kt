package com.rahalgo.driver.home

import com.rahalgo.driver.R
import com.rahalgo.shared.model.DriverOrder

/**
 * **أيُمنع خروجُه؟** — وفارغٌ يعني «لا».
 *
 * **وما في `/driver/orders` مفتوحٌ كلُّه** (`closed_at IS NULL`) — فطلبٌ واحدٌ
 * يكفي: **من خرج وفي يده طلبٌ ترك زبوناً ينتظر من لن يأتي، وبضاعةً بلا صاحب.**
 */
object LogoutGate {
    fun block(open: List<DriverOrder>): Int? =
        if (open.isNotEmpty()) R.string.logout_has_order else null
}
