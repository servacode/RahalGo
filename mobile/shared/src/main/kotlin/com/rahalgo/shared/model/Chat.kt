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
)
