package com.rahalgo.shared.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * **صندوق الإشعارات.**
 *
 * (`GET /api/v1/me/notifications` — والعدد غير القائمة: **الشارة تُرسم
 * من `unread` وحدَه**، ولا تُعدّ العناصر في الشاشة.)
 */
@Serializable
data class Inbox(
    val items: List<Notice> = emptyList(),
    val unread: Int = 0,
)

@Serializable
data class Notice(
    val id: String = "",
    val kind: String = "",
    val title: String = "",
    val body: String = "",
    val read: Boolean = false,
    @SerialName("created_at") val createdAt: String = "",
)

/**
 * **كشف المحفظة.**
 *
 * (`GET /api/v1/my/wallet` — ومنه الرصيد وحدَه يُعرض في الشريط:
 * **رقم واحد يقرؤه في لحظة**، وتفصيله في شاشته.)
 */
@Serializable
data class WalletStatement(
    val balance: Long = 0,
)
