package com.rahalgo.customer.push

import com.rahalgo.customer.MainActivity
import com.rahalgo.customer.R
import com.rahalgo.ui.push.RahalPushService

/**
 * **إشعاراتُ الزبون — ما يصل والتطبيقُ مغلق.**
 *
 * **والمنطقُ كلُّه في `RahalPushService`** — وهنا ما يخصّ الزبونَ وحدَه.
 * (نُقل ٢٠٢٦-٠٨-٢٥: كان مئةً وتسعةً وعشرين سطراً تكرّر ٨٥٪ منها في
 *  تطبيق السائق.)
 */
class PushService : RahalPushService() {
    override fun home(): Class<*> = MainActivity::class.java
    override fun icon(): Int = R.drawable.ic_orders
    override fun appName(): String = getString(R.string.app_name)
    override fun urgentChannelName(): String = getString(R.string.push_ch_order)
    override fun newsChannelName(): String = getString(R.string.push_ch_news)

    // **وخبرُ طلبِه يوقظه، والعرضُ التسويقيُّ لا.**
    override fun isUrgent(kind: String): Boolean = kind == KIND_ORDER

    companion object {
        const val KIND_ORDER = "order"
    }
}
