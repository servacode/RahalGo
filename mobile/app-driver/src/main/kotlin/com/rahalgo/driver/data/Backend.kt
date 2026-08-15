package com.rahalgo.driver.data

import android.content.Context
import com.rahalgo.ui.AppCore
import com.rahalgo.ui.Core
import com.rahalgo.driver.push.Push
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import com.rahalgo.shared.driver.ChatApi
import com.rahalgo.shared.driver.DriverApi

/**
 * ══════════════════════════════════════════════════════════════════════
 * **وصلة التطبيق بالمحرّك — نسخة واحدة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ونسخة واحدة لا نسخة لكل شاشة**: كل عميل يفتح تجمّع اتّصالات خاصّا
 * به، **وعشر شاشات تعني عشرة تجمّعات** تستنزف الذاكرة والبطارية.
 *
 * **وأخطر منه أن يتصادم التجديد**: شاشتان تكتشفان انتهاء التوكن معا
 * فتُجدّدان معا، **والثانية تُجدّد بتوكن أبطلته الأولى** — فيخرج السائق
 * وهو يعمل.
 */
object Backend {

    /**
     * **عنوان المحرّك — الإنتاج وحده.**
     *
     * (قرار المالك ٢٠٢٦-٠٨-١١: «بحسابي الحقيقي، ما بدنا نرجع للمحلّي».)
     *
     * **ولو أُريد المحلّي يوما**: المحاكي يصل جهاز التطوير على
     * `10.0.2.2` لا `localhost` — الأخير هو المحاكي نفسه.
     */
    const val BASE_URL = "https://rahalgo-api.onrender.com"

    /**
     * **نوع العميل — كما تعرفه قائمة المحرّك المغلقة.**
     *
     * (`identity/client_kind.go`: `{android|ios}-{customer|driver|merchant|rep}`.)
     *
     * **ونصّ لا يطابقها يُقرأ متصفّحا** فيُخرج صاحبه من الويب كلّما فتح
     * التطبيق — **وهو عطب صامت لا يظهر إلّا بشكوى.**
     */
    const val CLIENT = "android-driver"

    @Volatile private var instance: Wired? = null

    fun of(context: Context): Wired =
        instance ?: synchronized(this) {
            instance ?: Wired(context.applicationContext).also { instance = it }
        }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **عنوانُ ملفٍّ من المحرّك — من مسارٍ نسبيّ**
     * ══════════════════════════════════════════════════════════════════
     *
     * (شكوى المالك ٢٠٢٦-٠٨-١٣: «الصورة ما زالت حرف خ مع أنّني قمتُ برفع
     *  صورة».)
     *
     * **والمحرّكُ يرسل `/media/...` نسبيّا** — والشريطُ العلويُّ كان
     * يُكمله بيده، **وشاشةُ الحساب نسيت** فمرّرت المسارَ خاماً إلى
     * محمّل الصور. **فلا يُفتح، فتظهر الحرفُ الأوّلُ بدلَ الصورة** —
     * ولا خطأَ ولا سجلّ: **الصورةُ الاحتياطيّةُ تبتلع العطب.**
     *
     * **وموضعان يُكملان عنواناً واحداً** يفترقان يوماً — **وقد افترقا
     * في أوّل يوم.**
     */
    fun media(path: String?): String? = core.media(path)

    /**
     * ══════════════════════════════════════════════════════════════════
     * **ما يخصّ السائقَ وحدَه — وما سواه في النواة**
     * ══════════════════════════════════════════════════════════════════
     *
     * **الجلسةُ والدخولُ و`me` والحسابُ والأجهزةُ والبثُّ** رُفعت إلى
     * `ui.Core` (٢٠٢٦-٠٨-١٤): **يحتاجها الزبونُ كما يحتاجها السائق.**
     *
     * **وما بقي هنا يعرف السائق**: طابورُه وطلباتُه ودردشتُه وخريطتُه.
     *
     * **وأسماءُ النواة تُمرَّر كما كانت** (`session` · `api` · `auth` …)
     * — **فأربعةَ عشرَ ملفّاً تناديها**، ولا يُبدَّل نداءٌ لأجل نقل.
     */
    class Wired(context: Context) {
        private val core: Core = AppCore.install(context, BASE_URL, CLIENT) {
            // **ونقطةُ الإشعارات تُسجَّل بعد ثبوت الجلسة لا قبلها** —
            // **تحتاج توكنَ حساب**، ومن سجّلها قبله سجّلها بلا صاحب:
            // **فلا يصل إشعارٌ ولا يظهر خطأ.**
            //
            // **وهي تخصّ السائقَ فتبقى عنده** — والنواةُ تُنادي ولا تعرف
            // ما تُنادي.
            val app = context.applicationContext
            CoroutineScope(Dispatchers.IO).launch { Push.register(app) }
        }

        val session get() = core.session
        val api get() = core.api
        val auth get() = core.auth
        val me get() = core.me
        val account get() = core.account
        val devices get() = core.devices
        val live get() = core.live

        val driver = DriverApi(api)
        val chat = ChatApi(api)

        /** **عنوان أسلوب الخريطة** — يقرؤه العارض والمنزّل معا. */
        val styleUrl = "$BASE_URL/api/v1/public/map-style.json"
    }

    private val core: Core get() = AppCore.get()
}
