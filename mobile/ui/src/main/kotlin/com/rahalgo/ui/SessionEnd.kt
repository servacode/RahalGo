package com.rahalgo.ui

import android.content.Context
import android.os.Handler
import android.os.Looper
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.shared.net.ApiClient
import com.rahalgo.shared.net.SessionFailure
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نهايةُ الجلسة القسريّة — حدٌّ واحدٌ للعمليّة كلِّها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-١٠-١٦: دخل بحساب المتجر من جهازٍ ثانٍ، **فبقي
 *  الأوّلُ يعرض بياناتٍ قديمة** — «مغلق» والمتجرُ مفتوح — **والسحبُ
 *  وإعادةُ الفتح لا تفعل شيئاً، ولا رسالة.**)
 *
 * # ما كان
 *
 * **كان الخطّافُ (`ApiClient.onSessionRejected`) يُوصَل من `AuthViewModel`**
 * ويُطلق عملَه في `viewModelScope` الخاصِّ بذاك النموذج. **فنطاقٌ أُتلف**
 * (نشاطٌ ثانٍ فُتح من إشعارٍ ثمّ أُغلق) **يبتلع الإشارةَ صامتاً**: لا تُمسح
 * الجلسةُ ولا يتبدّل `user` — **فتبقى الشاشةُ «داخلةً» على ما جُلب آخرَ مرّة.**
 * **وخدمةُ السائق في الخلفيّة لا نموذجَ لها أصلاً.** **والرسالةُ كانت طافيةً
 * تنصرف في ثوانٍ** — ومن رُفضت جلستُه وهو يُقلع لم يرَ شيئاً.
 *
 * # ما صار
 *
 * **`AppCore.install` يوصل الخطّافَ هنا مرّةً.** فعند أوّل رفضٍ صريح:
 *
 *	١ تُمسح الجلسةُ في الحال، في خيط النداء نفسِه — **فلا نداءٌ موثَّقٌ يخرج
 *	  بعدها**، وكلُّ عملٍ في الخلفيّة يُردّ في الجهاز بلا شبكة.
 *	٢ يُنادى `afterLogout` على الخيط الرئيس — **وهو بابُ الخروج القائم**:
 *	  خدمةُ موقع السائق تقف، وسلّةُ الزبون ووصلتُه تُمسحان.
 *	٣ يُرفع العدّادُ `ended` — **فيُسقط كلُّ غلافٍ صاحبَه إلى الدخول** (`AuthViewModel`).
 *	٤ يُحفظ السببُ (`pending`) — **وتعرضه شاشةُ الدخول حتّى يدخل من جديد**،
 *	  ولو ماتت العمليّةُ قبل أن يفتحها (السائقُ في جيبه).
 *
 * **والضيفُ لا يُمسّ**: من لا توكنَ له لا جلسةَ تُنهى، فلا رسالةَ ولا عدّاد.
 */
object SessionEnd {

    private const val TAG = "RahalGo/session"
    private const val PREFS = "rahalgo_session_end"
    private const val KEY = "reason"

    /**
     * **سببُ آخرِ خروجٍ قسريٍّ لم يُرَ بعد** — تقرؤه شاشةُ الدخول.
     *
     * **حالةٌ مُراقَبة** لا متغيّرٌ عاديّ — فيُعاد رسمُ الشاشة حين تُرفع.
     * **ويُمحى بالدخول أو بالخروج الصريح** — لا بفتح الشاشة: الزبونُ يتصفّح
     * ضيفاً ويفتح الدخولَ بعد ساعة، **فيجد السببَ هناك.**
     */
    var pending by mutableStateOf<String?>(null)
        private set

    private val count = MutableStateFlow(0)

    /** **عدّادُ النهايات** — يزيد مع كلّ جلسةٍ تُنهى قسراً. */
    val ended: StateFlow<Int> = count

    private val main by lazy { Handler(Looper.getMainLooper()) }

    /** **يُركَّب مع النواة** — يقرأ السببَ المحفوظَ من عمليّةٍ سابقة. */
    internal fun install(context: Context) {
        pending = runCatching { prefs(context).getString(KEY, null) }.getOrNull()
    }

    /**
     * **ينهي الجلسةَ لرفضٍ صريحٍ من المحرّك** — يُنادى من أيّ خيط.
     *
     * **ومرّةً واحدةً لكلّ جلسة**: نداءاتٌ متزامنةٌ ترتدّ كلُّها، **وأوّلُها
     * يمسح** — والباقي يجد المخزنَ فارغاً فيمضي.
     */
    fun end(code: String) {
        val core = AppCore.peek() ?: return
        synchronized(this) {
            val s = core.session
            if (s.refreshToken().isEmpty() && s.accessToken().isEmpty()) return
            s.clear()
        }
        Log.w(TAG, "الجلسة رُفضت ($code) — خروجٌ قسريّ")
        Crash.soft(IllegalStateException("session ended"), "session ended $code")
        ApiClient.passwordChangePending = false
        runCatching { prefs(core.app).edit().putString(KEY, code).apply() }
        main.post {
            pending = code
            runCatching { AppCore.afterLogout() }
                .onFailure { Log.w(TAG, "تعذّر تنظيفُ ما بعد الخروج", it) }
            count.value = count.value + 1
        }
    }

    /** **يُمحى السبب** — بدخولٍ ناجحٍ أو خروجٍ صريح. */
    fun clear() {
        pending = null
        AppCore.peek()?.let { core -> runCatching { prefs(core.app).edit().remove(KEY).apply() } }
    }

    /** **نصُّ السبب** — «جهازٌ آخر» بصريحه، وما عداه «انتهت جلستك». */
    fun messageRes(code: String): Int = when (code) {
        SessionFailure.SUPERSEDED -> R.string.err_session_superseded
        "session_password_changed" -> R.string.err_session_password_changed
        "session_phone_changed" -> R.string.err_session_phone_changed
        "session_admin_logout" -> R.string.err_session_admin_logout
        "session_blocked" -> R.string.err_session_blocked
        "session_staff_removed" -> R.string.err_session_staff_removed
        // **جلسةٌ فُتحت بحسابٍ بلا صفةِ هذا التطبيق قبل القفل** — يُقال لصاحبها
        // لماذا (٢٠٢٦-١٠-٠١).
        "not_merchant_account" -> R.string.err_not_merchant_account
        "not_rep_account" -> R.string.err_not_rep_account
        "not_driver_account" -> R.string.err_not_driver_account
        else -> R.string.err_invalid_refresh
    }

    private fun prefs(context: Context) =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
}
