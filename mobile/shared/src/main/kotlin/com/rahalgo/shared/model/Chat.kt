package com.rahalgo.shared.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * **حديث الطلب** — بين السائق والزبون، **داخل الطلب وحدَه.**
 *
 * (قرار المالك ٢٠٢٦-٠٨-٠٩: «لازم الاثنان لا يقدران يوصلان لبعض إلّا عن
 * طريق المنصّة».)
 *
 * **ولا رقم هاتف في الطرفين** — لا للزبون ولا للسائق. **والقناة تُغلق
 * بانتهاء الطلب** (`open` و`closes_at`).
 */
@Serializable
data class ChatThread(
    val messages: List<ChatMessage> = emptyList(),
    @SerialName("peer_name") val peerName: String = "",
    /** **أما زالت مفتوحة؟** — تُغلق بانتهاء الطلب. */
    val open: Boolean = true,
    @SerialName("closes_at") val closesAt: String? = null,
)

@Serializable
data class ChatMessage(
    val id: String = "",
    val body: String = "",
    /** من كتبها: `driver` أو `customer` أو `system`. */
    val role: String = "",
    /** **أهي منّي؟** — يقرّرها المحرّك لا الشاشة. */
    val mine: Boolean = false,
    @SerialName("created_at") val createdAt: String = "",
    /**
     * **متى قرأها الطرف الآخر** — وفارغٌ يعني لم تُقرأ بعد.
     *
     * **ولرسائلي وحدَها معنى**: من رأى «قُرئت» على ما كتبه الآخر قرأ
     * خبرا عن نفسه.
     */
    @SerialName("read_at") val readAt: String? = null,
)

/** **قائمة محادثاتي** — ومنها تُقرأ الشارة. */
@Serializable
data class ChatThreads(val threads: List<ChatThreadRow> = emptyList())

@Serializable
data class ChatThreadRow(
    @SerialName("order_id") val orderId: String = "",
    val unread: Int = 0,
)
