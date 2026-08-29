package com.rahalgo.merchant.push

import com.rahalgo.merchant.MainActivity
import com.rahalgo.merchant.R
import com.rahalgo.ui.push.RahalPushService

/**
 * **إشعاراتُ المتجر — ما يصل وصاحبُ المتجر لا ينظر إلى شاشته.**
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٢٥.)
 *
 * # ولماذا هي أهمُّ من نظيرتها في الزبون
 *
 * **والزبونُ يفتح التطبيقَ ليطلب، ثمّ ينتظر.** أمّا صاحبُ المتجر
 * فمشغولٌ بمطعمه — **وطلبٌ لا يُنبَّه به يبرد ويُلغى.**
 *
 * # والجديدُ عاجلٌ وما عداه لا
 *
 * **وطلبٌ جديدٌ يوقظ الجهاز**، وإنذارُ الإدارة أو خبرٌ لا يوقظ. **ومن
 * جعل كلَّ شيءٍ عاجلاً أطفأ صاحبُ المتجر الإشعاراتِ كلَّها** — فخسر
 * العاجلَ معها.
 */
class PushService : RahalPushService() {
    override fun home(): Class<*> = MainActivity::class.java
    override fun icon(): Int = com.rahalgo.ui.R.drawable.ic_orders
    override fun appName(): String = getString(R.string.app_name)
    override fun urgentChannelName(): String = getString(R.string.push_ch_new_order)
    override fun newsChannelName(): String = getString(com.rahalgo.ui.R.string.push_ch_default)

    // **والمحرّكُ يرسل `kind=order`** حين يصل طلبٌ جديدٌ للمتجر.
    override fun isUrgent(kind: String): Boolean = kind == KIND_ORDER

    companion object {
        const val KIND_ORDER = "order"
    }
}
