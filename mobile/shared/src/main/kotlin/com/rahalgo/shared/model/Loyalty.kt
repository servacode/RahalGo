package com.rahalgo.shared.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقة رحّال غو وكرت اليوم** (قرارُ المالك ٢٠٢٦-١١-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * كما يردّها الخادم: `GET /my/stamp-card` للزبون و`GET /merchant/stores/{id}/stamp-card` للمتجر.
 * بطاقةُ رحّال غو في الجذر وكرتُ اليوم في `daily`. **والمطفأةُ لا تُرسم.**
 */
@Serializable
data class LoyaltyStamp(
    @SerialName("order_id") val orderId: String = "",
    val number: Long = 0,
    val at: String = "",
)

@Serializable
data class DailyCard(
    val enabled: Boolean = false,
    val size: Int = 0,
    val stamps: Int = 0,
    @SerialName("free_ready") val freeReady: Boolean = false,
    /** **سقفُ المعفى** — صفرٌ بلا سقف. */
    @SerialName("free_cap") val freeCap: Long = 0,
    @SerialName("reward_pending") val rewardPending: Boolean = false,
    /** **أيومُ الشغل قائم؟** — بعد إغلاق المنصّة: «يتجدّد مع بداية دوام الغد». */
    val open: Boolean = false,
    @SerialName("closes_at") val closesAt: String = "",
    @SerialName("renews_at") val renewsAt: String = "",
)

@Serializable
data class StampCard(
    val enabled: Boolean = false,
    val size: Int = 0,
    val stamps: Int = 0,
    @SerialName("free_ready") val freeReady: Boolean = false,
    @SerialName("free_cap") val freeCap: Long = 0,
    @SerialName("reward_pending") val rewardPending: Boolean = false,
    val recent: List<LoyaltyStamp> = emptyList(),
    val daily: DailyCard = DailyCard(),
)
