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

    /** **يُلغى عند الخروج** — وإلّا وصلت طلباتُ حسابٍ خرج إلى جهازه. */
    suspend fun unregister(token: String) {
        api.call<Ack>(
            "/api/v1/me/devices",
            HttpMethod.Delete,
            mapOf("token" to token),
        )
    }
}
