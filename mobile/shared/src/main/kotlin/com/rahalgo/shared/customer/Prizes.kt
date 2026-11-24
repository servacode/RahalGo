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
/** **نقاطُ كلّ فعل** كما في إعدادات المسابقة. */
@Serializable
data class PrizePointValues(
    val checkin: Int = 0,
    val order: Int = 0,
    @SerialName("invite_signup") val inviteSignup: Int = 0,
    @SerialName("invite_first_order") val inviteFirstOrder: Int = 0,
    @SerialName("follow_facebook") val followFacebook: Int = 0,
    @SerialName("follow_instagram") val followInstagram: Int = 0,
)

/**
 * **حالُه في مسابقة النقاط** (قرارُ المالك ٢٠٢٦-١١-٢٥) — **نقاطُه وترتيبُه هو وحدَه**: لا أسماءَ
 * ولا نقاطَ لغيره.
 */
@Serializable
data class PrizesMe(
    val enabled: Boolean = false,
    /** **يومُ الافتتاح** `yyyy-MM-dd` — قبله تُعرض لافتةُ «انتظرونا». */
    @SerialName("launch_on") val launchOn: String = "",
    /** **بدأت المسابقة** — مفعّلةٌ وبلغنا يومَ الإطلاق؛ قبلها لا حضورَ ولا متابعة. */
    val started: Boolean = false,
    val month: String = "",
    /** **آخرُ لحظةٍ لجمع نقاط الشهر** RFC3339 — بعدها يُحسب كلُّ شيءٍ للشهر الجاي. */
    @SerialName("cutoff_at") val cutoffAt: String = "",
    @SerialName("announce_at") val announceAt: String = "",
    val points: Int = 0,
    /** **ترتيبُه** — وصفرٌ إن لم تكن له نقطةٌ بعد أو كان مستثنًى. */
    val rank: Int = 0,
    @SerialName("min_points") val minPoints: Int = 0,
    @SerialName("to_qualify") val toQualify: Int = 0,
    val qualified: Boolean = false,
    /** **ربح الشهر الماضي** — فهذا الشهرُ استراحة. */
    val excluded: Boolean = false,
    @SerialName("checked_in_today") val checkedInToday: Boolean = false,
    @SerialName("followed_facebook") val followedFacebook: Boolean = false,
    @SerialName("followed_instagram") val followedInstagram: Boolean = false,
    @SerialName("facebook_url") val facebookUrl: String = "",
    @SerialName("instagram_url") val instagramUrl: String = "",
    @SerialName("point_values") val pointValues: PrizePointValues = PrizePointValues(),
    /** **سلّمُ المبالغ** بالترتيب — الأوّلُ للمرتبة الأولى. */
    val amounts: List<Long> = emptyList(),
    @SerialName("winners_count") val winnersCount: Int = 0,
    /** **نتيجتُه هو في آخر إعلان** — صفرٌ إن لم يربح. */
    @SerialName("last_month") val lastMonth: String? = null,
    @SerialName("last_rank") val lastRank: Int = 0,
    @SerialName("last_amount") val lastAmount: Long = 0,
    @SerialName("prize_balance") val prizeBalance: Long = 0,
    @SerialName("prize_expires_at") val prizeExpiresAt: String? = null,
    /** **متجرُ الجوائز** — به تُطلب أصنافُها من السلّة العاديّة. */
    @SerialName("store_id") val storeId: String? = null,
)

/** **«تابعنا»** — `facebook` أو `instagram`. */
@Serializable
data class PrizeFollowInput(val network: String)

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
