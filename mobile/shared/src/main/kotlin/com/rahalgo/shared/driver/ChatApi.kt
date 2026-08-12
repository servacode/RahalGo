package com.rahalgo.shared.driver

import com.rahalgo.shared.model.ChatMessage
import com.rahalgo.shared.model.ChatThread
import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod

/**
 * **حديث الطلب.**
 *
 * (`GET/POST /api/v1/orders/{id}/messages` — والمحرّك يقرّر من يحقّ له
 * الدخول: **طرفا الطلب وحدَهما**، وتُغلق بانتهائه.)
 */
class ChatApi(private val api: ApiClient) {

    suspend fun thread(orderId: String): ChatThread =
        api.call("/api/v1/orders/" + orderId + "/messages")

    suspend fun send(orderId: String, body: String): ChatMessage =
        api.call(
            "/api/v1/orders/" + orderId + "/messages",
            HttpMethod.Post,
            mapOf("body" to body),
        )
}
