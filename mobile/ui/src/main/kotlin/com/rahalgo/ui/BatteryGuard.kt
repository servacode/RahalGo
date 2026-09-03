package com.rahalgo.ui

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.PowerManager
import android.provider.Settings

/**
 * ══════════════════════════════════════════════════════════════════════
 * **النظامُ يقتل الخدمةَ والسائقُ لا يعلم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **(قِيس ٢٠٢٦-٠٩-٠٢: لا سطرَ واحدٌ في المشروع كلِّه يعالج توفيرَ
 *  الطاقة — لا إذنٌ في بيانٍ ولا فحصٌ في شيفرة.)**
 *
 * # وما يقع فعلاً
 *
 * **خدمةُ الموقع أماميّةٌ بنوع `location`** — وهذا كافٍ في أندرويد
 * الصافي. **وليس كافياً في هواتف الناس.**
 *
 * **وسامسونغ تُنيم التطبيقات** («التطبيقات النائمة» و«النائمة
 * بعمق»)، وشاومي وهواوي أشرس. **فيقفل السائقُ الشاشةَ ويضع الهاتفَ
 * في جيبه، فيُقتل التطبيقُ بعد دقائق.**
 *
 * **ولا شيءَ يقول له.** ورديّتُه مفتوحةٌ في الخادم، **وموضعُه واقفٌ
 * عند آخر نقطةٍ وصلت.** فيُسأل: أين أنت؟ فيقول: في الطريق. **والخريطةُ
 * تقول إنّه لم يتحرّك منذ عشرين دقيقة.**
 *
 * # ولماذا يُسأل ولا يُفرض
 *
 * **والإعفاءُ لا يُمنح بالشيفرة** — يفتح النظامُ نافذتَه ويقرّر
 * صاحبُ الهاتف. **ومن ظنّ أنّه يضبطه برمزٍ بنى وهماً.**
 *
 * # ويُسأل مرّةً لا في كلّ فتحة
 *
 * **وسؤالٌ يتكرّر يُرفض بالعادة** — تصير النافذةُ شيئاً يُغلق بلا
 * قراءة. **فيُسأل عند رفع الورديّة**، وهي اللحظةُ التي يفهم فيها
 * لماذا.
 *
 * # ولا يُدّعى ما لا يُعرف
 *
 * **والإعفاءُ ليس ضماناً**: سامسونغ تُنيم التطبيقاتِ بقائمةٍ ثانيةٍ
 * لا يمسّها هذا الإذن. **فهو يرفع أشيعَ الأسباب لا كلَّها.**
 */
object BatteryGuard {

    /** **أمعفًى من توفير الطاقة؟** — وما دون أندرويد ٦ لا قيدَ فيه. */
    fun exempt(context: Context): Boolean {
        val pm = context.getSystemService(Context.POWER_SERVICE) as? PowerManager
            ?: return true
        return pm.isIgnoringBatteryOptimizations(context.packageName)
    }

    /**
     * **يفتح نافذةَ النظام** — ويردّ `false` إن لم يقبلها الجهاز.
     *
     * **وبعضُ الأجهزة تمنع هذه النيّة** (مُهيّئاتٌ مخصَّصة، أو منعُ
     * سياسةٍ في هاتفٍ إداريّ). **فيُفتح حينها بابُ الإعدادات العامّ**
     * — أطولُ خطوةً وأسلمُ من زرٍّ لا يفعل شيئاً.
     */
    @Suppress("BatteryLife")
    fun ask(context: Context): Boolean {
        val direct = Intent(
            Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS,
            Uri.parse("package:" + context.packageName),
        ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        if (open(context, direct)) return true

        val list = Intent(Settings.ACTION_IGNORE_BATTERY_OPTIMIZATION_SETTINGS)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        if (open(context, list)) return true

        val app = Intent(
            Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
            Uri.parse("package:" + context.packageName),
        ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        return open(context, app)
    }

    private fun open(context: Context, intent: Intent): Boolean =
        runCatching { context.startActivity(intent); true }.getOrDefault(false)
}
