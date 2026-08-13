package com.rahalgo.shared.push

import com.rahalgo.shared.net.Ack
import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod

/**
 * **تسجيل جهاز لاستقبال الإشعارات.**
 *
 * (`POST /api/v1/me/devices` — والمحرّك يقرأ نوع التطبيق من ترويسة
 * `X-RahalGo-Client` فيوجّه إشعارَ السائق إلى تطبيق السائق وحدَه.)
 *
 * **ويُعاد التسجيل عند كلّ إقلاع** — لا مرّة واحدة: **التوكن يتبدّل**
 * بتنصيبٍ جديد أو مسحِ بيانات، **ومن سجّله مرّة** بقي يرسل إلى جهازٍ لم
 * يعد يسمع.
 */
class DevicesApi(private val api: ApiClient) {

    suspend fun register(token: String, appVersion: String = "") {
        api.call<Ack>(
            "/api/v1/me/devices",
            HttpMethod.Post,
            mapOf(
                "token" to token,
                "platform" to "android",
                "app_version" to appVersion,
            ),
        )
    }

    /**
     * **فحصُ الإشعارات** — يرسل رسالةً إلى أجهزته ويقول ما وقع.
     *
     * (وقع ٢٠٢٦-٠٨-١٤: جُرّب على جهازٍ حقيقيٍّ فلم يرنّ، **والجهازُ
     *  مسجَّلٌ والمفتاحُ مضبوطٌ والمشروعُ متطابق** — ولم يكن في المنصّة
     *  ما يقول أين وقفت الرسالة.)
     *
     * **وإشعارٌ لا يصل لا يشتكي منه أحد**: السائقُ يظنّ أنّه لا طلبات،
     * **والمكتبُ يظنّه كسولا.**
     */
    suspend fun test(): PushCheck =
        api.call("/api/v1/me/devices/test", HttpMethod.Post, mapOf<String, String>())

    /** **يُلغى عند الخروج** — وإلّا وصلت طلباتُ حسابٍ خرج إلى جهازه. */
    suspend fun unregister(token: String) {
        api.call<Ack>(
            "/api/v1/me/devices",
            HttpMethod.Delete,
            mapOf("token" to token),
        )
    }
}

/**
 * **ما وجده فحصُ الإشعارات.**
 *
 * **ونصُّ الخطأ كما قالته غوغل** — لا يُترجَم ولا يُلطَّف:
 * «SENDER_ID_MISMATCH» كلمةٌ تُبحث فتُوجد، **و«تعذّر الإرسال» لا تُوجد.**
 */
@kotlinx.serialization.Serializable
data class PushCheck(
    val configured: Boolean = false,
    val devices: Int = 0,
    val fresh: Int = 0,
    val sent: Int = 0,
    val dead: Int = 0,
    val error: String = "",
)
