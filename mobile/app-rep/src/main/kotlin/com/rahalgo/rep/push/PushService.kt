package com.rahalgo.rep.push

import com.rahalgo.rep.MainActivity
import com.rahalgo.rep.R
import com.rahalgo.ui.push.RahalPushService

/**
 * **إشعاراتُ المندوب.**
 *
 * # ولماذا لم تكن موجودة
 *
 * **كان المحرّكُ يرسل إليه ولا يصل شيء** — `orders/notify.go:274`
 * يبعث عمولتَه إلى `AppRep`، **وتطبيقُ المندوب بلا نقطةِ استقبالٍ
 * إطلاقاً.** فالرسالةُ تُكتب في القاعدة ولا تُوقظ جهازاً.
 *
 * **وكان يُسأل إذنَ إشعاراتٍ لا يستعمله**: `POST_NOTIFICATIONS` معلَنٌ
 * في البيان منذ أوّل يوم، **بلا خدمةٍ تتلقّى.**
 *
 * # والعميلُ الجديدُ عاجلٌ والعمولةُ لا
 *
 * **عميلٌ أُسند إليه يحتاج زيارةً اليوم** — فيوقظ. **وعمولةٌ أُودعت
 * خبرٌ سارٌّ لا يستعجل**، فتنتظر أن يفتح.
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
    override fun urgentChannelName(): String = getString(R.string.push_ch_lead)
    override fun newsChannelName(): String = getString(com.rahalgo.ui.R.string.push_ch_default)

    // **والمحرّكُ يرسل `kind=lead`** حين يُسند عميلٌ جديد.
    override fun isUrgent(kind: String): Boolean = kind == KIND_LEAD

    companion object {
        const val KIND_LEAD = "lead"
    }
}
