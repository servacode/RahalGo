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
    override fun urgentChannelName(): String = getString(R.string.push_ch_offer)
    override fun newsChannelName(): String = getString(R.string.push_ch_news)

    // **ورسالةُ الزبون عاجلةٌ كذلك** — **سؤالٌ في طريقٍ ينتظر جواباً
    // الآن**: «وين الباب؟» تُقرأ بعد التسليم لا تنفع.
    override fun isUrgent(kind: String): Boolean = kind == KIND_OFFER || kind == KIND_CHAT

    // **وعرضُ الطلب يصل عاجلاً بعلامة المحرّك** — كان يُبحث عن `order_offer` والمحرّكُ
    // يرسل `order` مع `urgent=1`، فوصل العرضُ على قناة الأخبار (فحصُ دورة السائق ٢٠٢٦-١٠-٠٢).
    override fun honorsServerUrgency(): Boolean = true

    companion object {
        const val KIND_OFFER = "order_offer"
        const val KIND_CHAT = com.rahalgo.ui.Engagement.KIND_CHAT
    }
}
