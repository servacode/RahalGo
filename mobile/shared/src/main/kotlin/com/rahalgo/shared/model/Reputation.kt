package com.rahalgo.shared.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سُمعتُه — نجومُه ومن أعطاها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «ننتقل إلى التقييم، نبني الصفحة أيضاً…
 *  وعند النقر عليه يفتح صفحة التقييم كما هي بالويب، فيعرف ما هي
 *  التقييمات التي حصل عليها ومن أين».)
 *
 * **والنداءُ نفسُه الذي تقرؤه شاشةُ الويب** (`GET /api/v1/me/reputation`).
 */
@Serializable
data class Reputation(
    val rating: RatingSummary = RatingSummary(),
    val reviews: List<Review> = emptyList(),
    val complaints: List<ComplaintBrief> = emptyList(),
    /**
     * **أيُقيَّم هذا الدورُ أصلاً** — و«٠٫٠ من ٥» لمن لا يُقيَّم يُقرأ
     * **حكماً عليه** فيسأل عمّا فعل، ولم يفعل شيئا.
     */
    val rated: Boolean = false,
)

@Serializable
data class RatingSummary(
    val avg: Double = 0.0,
    val count: Int = 0,
    /** **متوسّطُ آخر ثلاثين يوما** — ومنه يُعرف الاتّجاه. */
    @SerialName("recent_avg") val recentAvg: Double = 0.0,
    /** `up` · `down` · `flat` */
    val trend: String = "flat",
)

/**
 * **تقييمٌ واحد — بنجومه واسم صاحبه وواقعته.**
 *
 * **والاسمُ هو ما يجعله يتعلّم**: تقييمٌ بلا نسبةٍ إلى واقعةٍ **يُقرأ
 * حكماً عامّاً على شخصه** لا ملاحظةً على خدمةٍ بعينها.
 */
@Serializable
data class Review(
    @SerialName("order_number") val orderNumber: Long = 0,
    @SerialName("merchant_name") val merchantName: String = "",
    @SerialName("customer_name") val customerName: String = "",
    val stars: Int = 0,
    val comment: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

/** **شكوى — بلا اسم صاحبها**: خصومةٌ تُحقَّق، وكشفُه يفتح باباً للردّ. */
@Serializable
data class ComplaintBrief(
    val number: Long = 0,
    @SerialName("order_number") val orderNumber: Long? = null,
    val subject: String = "",
    val status: String = "",
    @SerialName("created_at") val createdAt: String = "",
)
