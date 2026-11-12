package com.rahalgo.shared.driver

import com.rahalgo.shared.model.ChatMessage
import com.rahalgo.shared.model.ChatThread
import com.rahalgo.shared.model.ChatThreads
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

    /**
     * **محادثاتي وما ينتظرني في كلٍّ منها.**
     *
     * **ولماذا لا يُقرأ العددُ من الحديث نفسه**: فتحُ الحديث يوسمه
     * مقروءا — **فالعددُ فيه صفرٌ دائما.** والشارةُ تُقرأ والحديثُ
     * مطويّ.
     */
    suspend fun threads(): ChatThreads = api.call("/api/v1/my/chats")

    suspend fun send(orderId: String, body: String): ChatMessage =
        api.call(
            "/api/v1/orders/" + orderId + "/messages",
            HttpMethod.Post,
            mapOf("body" to body),
        )

    /** **صورةٌ في الحديث** (قرارُ المالك ٢٠٢٦-١١-١٢) — JPEG مصغَّرٌ من الجهاز، والمحرّكُ يصغّره أيضاً. */
    suspend fun sendPhoto(orderId: String, jpeg: ByteArray): ChatMessage =
        decode(
            api.upload(
                "/api/v1/orders/" + orderId + "/messages/attachment",
                "photo.jpg",
                jpeg,
                fields = mapOf("kind" to "photo"),
            ),
        )

    /** **رسالةٌ صوتيّة** — AAC في `.m4a`، حتّى ٦٠ ثانية. */
    suspend fun sendVoice(orderId: String, m4a: ByteArray, durationMs: Long): ChatMessage =
        decode(
            api.upload(
                "/api/v1/orders/" + orderId + "/messages/attachment",
                "voice.m4a",
                m4a,
                fields = mapOf("kind" to "audio", "duration_ms" to durationMs.toString()),
                contentType = "audio/mp4",
            ),
        )

    private fun decode(raw: String): ChatMessage = json.decodeFromString(ChatMessage.serializer(), raw)

    private companion object {
        val json = kotlinx.serialization.json.Json { ignoreUnknownKeys = true }
    }
}
