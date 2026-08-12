package com.rahalgo.driver.data

import android.content.Context
import com.rahalgo.shared.auth.AuthApi
import com.rahalgo.shared.driver.ChatApi
import com.rahalgo.shared.driver.DriverApi
import com.rahalgo.shared.net.LiveSocket
import com.rahalgo.shared.push.DevicesApi
import com.rahalgo.shared.net.ApiClient

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

    class Wired(context: Context) {
        val session = AndroidSession(context)
        val api = ApiClient(BASE_URL, CLIENT, session)
        val auth = AuthApi(api)
        val driver = DriverApi(api)
    val chat = ChatApi(api)
    val devices = DevicesApi(api)

    /** **عنوان أسلوب الخريطة** — يقرؤه العارض والمنزّل معا. */
    val styleUrl = "$BASE_URL/api/v1/public/map-style.json"

    /** **البثّ الحيّ** — واحدٌ للتطبيق كلّه، لا واحدٌ لكلّ شاشة. */
    val live = LiveSocket(BASE_URL, session, CLIENT)
    }
}
