package com.rahalgo.driver.push

import com.rahalgo.driver.MainActivity
import com.rahalgo.driver.R
import com.rahalgo.ui.push.RahalPushService

/**
 * **إشعاراتُ السائق — والعرضُ لا ينتظر.**
 *
 * **والمنطقُ كلُّه في `RahalPushService`** — وهنا ما يخصّ السائقَ وحدَه.
 *
 * # ولماذا العرضُ عاجلٌ عنده والطلبُ عاجلٌ عند الزبون
 *
 * **وعرضُ الطلب له مهلةٌ تنتهي** — فمن لم يُوقَظ خسره. **وخبرُ الزبون
 * لا مهلةَ له**، إنّما يطمئنه.
 */
class PushService : RahalPushService() {
    override fun home(): Class<*> = MainActivity::class.java
    override fun icon(): Int = R.drawable.ic_orders
    override fun appName(): String = getString(R.string.app_name)
    override fun urgentChannelName(): String = getString(R.string.push_ch_offer)
    override fun newsChannelName(): String = getString(R.string.push_ch_news)

    override fun isUrgent(kind: String): Boolean = kind == KIND_OFFER

    companion object {
        const val KIND_OFFER = "order_offer"
    }
}
