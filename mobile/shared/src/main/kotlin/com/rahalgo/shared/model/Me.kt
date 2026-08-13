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
    val transactions: List<WalletTx> = emptyList(),
    /**
     * **الافتتاحيُّ والختاميُّ للمدى المعروض.**
     *
     * **ولا معنى لكشفٍ بدونهما**: من يبدأ من منتصف التاريخ يجمع الأسطر
     * **فلا تساوي رصيدَه فيظنّ الخللَ في المنصّة.**
     */
    val opening: Long = 0,
    val closing: Long = 0,
    /** **قُصّ الكشفُ عند السقف** — وناقصٌ يجب أن يقول إنّه ناقص. */
    val truncated: Boolean = false,
)

/**
 * **حركةٌ في المحفظة** — بمبلغها ونوعها ومرجعها.
 *
 * **والمرجعُ هو ما يجعلها مفهومة**: «‎+٣٠٠ ل.س» وحدَها لا تقول شيئا،
 * **و«‎+٣٠٠ · أجرة الطلب #١٠٠٩» تُقرأ بلا سؤال.**
 */
@Serializable
data class WalletTx(
    val id: Long = 0,
    /** **موجبٌ له وسالبٌ عليه** — والإشارةُ هي الخبر. */
    val amount: Long = 0,
    val kind: String = "",
    val note: String = "",
    @SerialName("order_number") val orderNumber: Long? = null,
    @SerialName("ticket_number") val ticketNumber: Long? = null,
    @SerialName("by_name") val byName: String? = null,
    @SerialName("created_at") val createdAt: String = "",
)

/** **طلبُ سحبٍ** — بمبلغه وحاله وقرار المالية. */
@Serializable
data class Payout(
    val id: String = "",
    val amount: Long = 0,
    val status: String = "",
    val note: String = "",
    val decision: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

/** **ما يُرسَل لطلب السحب.** */
@Serializable
data class PayoutInput(val amount: Long, val note: String)
