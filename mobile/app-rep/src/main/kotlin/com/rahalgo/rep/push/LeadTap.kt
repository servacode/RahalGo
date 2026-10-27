package com.rahalgo.rep.push

import android.content.Context
import android.content.Intent
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.ui.push.RahalPushService

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إشعارُ عميلٍ يُفتح على «عملائي» لا على «لوحتي»** (٢٠٢٦-١٠-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **إشعاراتُ الرفض وطلب المعلومات وإسناد العميل تصل بنوع `lead`** —
 * **ووجهةُ المحرّك لها صفحةُ ويبٍ للموظّفين** (`/portal/leads`)، **فلا
 * تعرفها `Engagement.route` وتسقط إلى البيت**: فيفتح «لوحتي» ويبحث
 * المندوبُ بنفسه عن العميل.
 *
 * **ولا يُمسّ المسارُ المشترك** — والوجهاتُ هناك للتطبيقات الأربعة.
 * **فهنا علامةٌ صغيرة**: يُعلَّم وصولُ إشعار عميلٍ في الخدمة، **ونقرةُ
 * إشعارٍ بعده تفتح «عملائي».** وتُمحى العلامةُ حين تُستهلك، **ولا تعيش
 * أكثرَ من يومين** فلا تسوقه إلى «عملائي» نقرةُ خبرٍ لا علاقةَ له.
 */
object LeadTap {

    private const val PREFS = "rep_lead_tap"
    private const val KEY_AT = "armed_at"
    private const val TTL_MS = 2L * 24 * 60 * 60 * 1000

    /** **افتح «عملائي»** — تُرفع عند النقرة وتُخفض حين تُفتح. */
    var openClients by mutableStateOf(false)

    /** **وصل إشعارُ عميل** — يُعلَّم ليُفتح على «عملائي» إن نُقر. */
    fun arm(context: Context) {
        runCatching {
            context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .edit().putLong(KEY_AT, System.currentTimeMillis()).apply()
        }
    }

    /**
     * **فُتح التطبيقُ من إشعار؟** — فإن سبقه إشعارُ عميلٍ رُفعت `openClients`.
     *
     * **ويُقرأ قبل `Opened.from`** — فهي تمحو علامةَ الإشعار من المقصد.
     */
    fun from(context: Context, intent: Intent?) {
        if (intent?.hasExtra(RahalPushService.EXTRA_DEST_TYPE) != true) return
        val prefs = runCatching { context.getSharedPreferences(PREFS, Context.MODE_PRIVATE) }.getOrNull() ?: return
        val at = prefs.getLong(KEY_AT, 0L)
        if (at == 0L) return
        prefs.edit().remove(KEY_AT).apply()
        if (System.currentTimeMillis() - at <= TTL_MS) openClients = true
    }
}
