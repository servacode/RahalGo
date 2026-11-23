package com.rahalgo.shared.customer

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **جوائزُ رحّال غو — ما يقوله المحرّك**
 * ══════════════════════════════════════════════════════════════════════
 *
 * كلُّ حقلٍ له قيمةٌ افتراضيّة: **حقلٌ ناقصٌ لا يُسقط الفكّ**، **والمبالغُ بالليرة
 * الجديدة** كما يرسلها المحرّك، **والحسابُ كلُّه عنده** — هنا تُقرأ وتُعرض.
 */
@Serializable
data class PrizesMe(
    val enabled: Boolean = false,
    /** **يومُ الافتتاح** `yyyy-MM-dd` — قبله تُعرض لافتةُ «انتظرونا». */
    @SerialName("launch_on") val launchOn: String = "",
    val month: String = "",
    @SerialName("month_invites") val monthInvites: Int = 0,
    @SerialName("invites_per_entry") val invitesPerEntry: Int = 5,
    val entries: Int = 0,
    @SerialName("invites_to_next") val invitesToNext: Int = 0,
    /** **موعدُ السحب** RFC3339. */
    @SerialName("draw_at") val drawAt: String = "",
    /** **سلّمُ المبالغ** بالترتيب — الأوّلُ للمرتبة الأولى. */
    val amounts: List<Long> = emptyList(),
    @SerialName("winners_count") val winnersCount: Int = 0,
    @SerialName("prize_balance") val prizeBalance: Long = 0,
    @SerialName("prize_expires_at") val prizeExpiresAt: String? = null,
    /** **متجرُ الجوائز** — به تُطلب أصنافُها من السلّة العاديّة. */
    @SerialName("store_id") val storeId: String? = null,
)

@Serializable
data class PrizeItem(
    val id: String = "",
    val name: String = "",
    val description: String? = null,
    val price: Long = 0,
    @SerialName("image_url") val imageUrl: String? = null,
    @SerialName("thumb_url") val thumbUrl: String? = null,
    val available: Boolean = true,
)

@Serializable
data class PrizeItems(
    @SerialName("store_id") val storeId: String? = null,
    val items: List<PrizeItem> = emptyList(),
)

@Serializable
data class PrizeWinner(
    val rank: Int = 0,
    val name: String = "",
    val amount: Long = 0,
)

@Serializable
data class PrizeWinners(
    val month: String? = null,
    val winners: List<PrizeWinner> = emptyList(),
)
