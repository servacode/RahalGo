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
    // ══════════════════════════════════════════════════════════════
    // **وأيقونةُ الإشعار شعارُ رحّال غو — في التطبيقات الأربعة**
    // ══════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٩-٣٠: «أيقونة الإشعار لازم يكون لوغو رحّال
    //  غو لينعرف إنّ الإشعار تابع لرحّال غو، مو أيقونات مختلفة».)
    //
    // **وكانت ثلاثةً**: أيقونةُ الطلبات للزبون والسائق والمتجر،
    // وأيقونةُ المستخدم للمندوب — **ولا واحدةٌ منها شعارُ المنصّة.**
    // فيصل المستخدمَ إشعارٌ برمزٍ لا يعرفه، **ومن لا يعرف المُرسِل
    // لا يفتح.**
    //
    // **وهي ظلٌّ أحاديٌّ لا شعارٌ ملوّن**: أندرويد يُهمل ألوانَ أيقونة
    // الشريط ويقرأ الألفا وحدَها، **وشعارٌ ملوّنٌ يُرسَم بقعةً بيضاء.**
    // فتُولَّد من شعار العلامة بألفاه وحدَها، بخمس كثافات.
    override fun icon(): Int = com.rahalgo.ui.R.drawable.ic_notification
    override fun appName(): String = getString(R.string.app_name)
    override fun urgentChannelName(): String = getString(R.string.push_ch_order)
    override fun newsChannelName(): String = getString(R.string.push_ch_news)

    // **وخبرُ طلبِه يوقظه، والعرضُ التسويقيُّ لا.**
    //
    // **ورسالةُ السائق كخبر الطلب** — **سؤالٌ ينتظر جواباً الآن**:
    // «وين الباب؟» تُقرأ بعد ساعةٍ لا تنفع.
    override fun isUrgent(kind: String): Boolean = kind == KIND_ORDER || kind == KIND_CHAT

    companion object {
        const val KIND_ORDER = "order"
        const val KIND_CHAT = com.rahalgo.ui.Engagement.KIND_CHAT
    }
}
