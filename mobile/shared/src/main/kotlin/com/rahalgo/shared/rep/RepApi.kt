package com.rahalgo.shared.rep

import com.rahalgo.shared.model.IncentivesPayload
import com.rahalgo.shared.model.WalletStatement
import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أبوابُ المندوب — كما تناديها لوحتُه في الويب حرفيّا**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (`web/apps/rahalgo/src/app/rep/` — قِيست نداءاتُها بندا بند.)
 *
 * **وكلُّها تحت `RequireRoles("sales")`** — **فلا يفتحها سائقٌ ولا
 * زبون**، ويفرضه المحرّكُ لا الشاشة.
 */
class RepApi(private val api: ApiClient) {

    /** **أرقامُه وهدفُه** — تُقرأ في لوحته. */
    suspend fun me(): RepMe = api.call("/api/v1/rep/me")

    /** **متاجرُه** — بعمولته عن كلٍّ منها. */
    suspend fun merchants(): List<RepMerchant> = api.call("/api/v1/rep/merchants")

    /** **تفصيلُ متجرٍ وطلباتُه** — وشفافيّةُ العمولة. */
    suspend fun merchant(id: String): RepMerchantDetail =
        api.call("/api/v1/rep/merchants/$id")

    /** **من سجّلهم ولم يُوافَق عليهم بعد.** */
    suspend fun leads(): List<Lead> = api.call("/api/v1/rep/leads")

    /**
     * **يُسجّل عميلاً من الميدان** — **ويبقى معلّقاً حتّى موافقة
     * الإدارة.**
     *
     * **ولا يُنشئ متجراً بضغطة**: من فتح هذا البابَ بلا مراجعةٍ فتح
     * بابَ من يُسجّل متاجرَ وهميّةً ليأخذ عمولتَها.
     */
    suspend fun createLead(input: NewLead): Lead =
        api.call("/api/v1/rep/leads", HttpMethod.Post, input)

    /** **تصنيفاتُ المتاجر** — لنموذج التسجيل. */
    suspend fun categories(): List<RepCategory> = api.call("/api/v1/rep/categories")

    suspend fun wallet(): WalletStatement = api.call("/api/v1/rep/wallet")

    /** **هدفُه ومكافأتُه** — كهدف السائق: المقياسُ يختلف والمعنى واحد. */
    suspend fun incentives(): IncentivesPayload = api.call("/api/v1/rep/incentives")
}

/**
 * **حالُ المندوب.**
 *
 * **والشهريُّ بجانب الإجماليّ** — **ورقمٌ إجماليٌّ وحدَه لا يقول أيَعمل
 * هذا الشهرَ أم يعيش على ما مضى.**
 */
@Serializable
data class RepMe(
    @SerialName("invite_code") val inviteCode: String? = null,
    @SerialName("full_name") val fullName: String = "",
    val merchants: Int = 0,
    @SerialName("delivered_orders") val deliveredOrders: Int = 0,
    @SerialName("total_commissions") val totalCommissions: Long = 0,
    val balance: Long = 0,
    @SerialName("month_merchants") val monthMerchants: Int = 0,
    @SerialName("month_delivered") val monthDelivered: Int = 0,
    @SerialName("month_commissions") val monthCommissions: Long = 0,
    @SerialName("monthly_target") val monthlyTarget: Int = 0,
    /** **ما ينتظر موافقةَ الإدارة** — ومنه يعرف أنّ عملَه لم يضع. */
    @SerialName("pending_leads") val pendingLeads: Int = 0,
    @SerialName("whatsapp_verified") val whatsappVerified: Boolean = false,
)

/**
 * **متجرٌ سجّله.**
 *
 * **والتفعيلُ يُقال بعددين لا بكلمة**: كم طلباً احتُسب وكم يلزم —
 * **و«قيد التفعيل» وحدَها لا تقول كم بقي.**
 */
@Serializable
data class RepMerchant(
    val id: String = "",
    val name: String = "",
    @SerialName("category_icon") val categoryIcon: String = "",
    @SerialName("category_name") val categoryName: String = "",
    @SerialName("logo_thumb_url") val logoThumbUrl: String? = null,
    val status: String = "",
    @SerialName("joined_at") val joinedAt: String = "",
    @SerialName("owner_phone") val ownerPhone: String? = null,
    @SerialName("delivered_orders") val deliveredOrders: Int = 0,
    @SerialName("cancelled_orders") val cancelledOrders: Int = 0,
    @SerialName("my_commission") val myCommission: Long = 0,
    @SerialName("last_order_at") val lastOrderAt: String? = null,
    @SerialName("activation_done") val activationDone: Int = 0,
    @SerialName("activation_needed") val activationNeeded: Long = 0,
)

@Serializable
data class RepMerchantDetail(
    val merchant: RepMerchantHead = RepMerchantHead(),
    val summary: RepSummary = RepSummary(),
    val orders: List<RepOrderLine> = emptyList(),
    val total: Int = 0,
    val page: Int = 1,
)

@Serializable
data class RepMerchantHead(
    val name: String = "",
    @SerialName("category_icon") val categoryIcon: String = "",
    @SerialName("category_name") val categoryName: String = "",
    @SerialName("logo_thumb_url") val logoThumbUrl: String? = null,
    val status: String = "",
    @SerialName("joined_at") val joinedAt: String = "",
    @SerialName("owner_phone") val ownerPhone: String? = null,
)

/**
 * **مجاميعُ العميل — أربعةُ أرقامٍ تُقرأ بنظرة.**
 *
 * **وصافي العمولة يُقرأ من الدفتر لا يُحسب في الشاشة** — **وطلبٌ رُجّع
 * تُسحب عمولتُه بقيدٍ معاكس**، وحسبةٌ في الجهاز لا تعرف ذلك.
 */
@Serializable
data class RepSummary(
    val orders: Int = 0,
    val delivered: Int = 0,
    val cancelled: Int = 0,
    @SerialName("delivered_sales") val deliveredSales: Long = 0,
    @SerialName("my_earnings") val myEarnings: Long = 0,
)

/**
 * **طلبٌ من طلبات العميل — وشفافيّةُ العمولة هي المقصودة.**
 *
 * **والمندوبُ يقبض نسبةً ولم يكن يرى من أين جاءت** — رقمٌ مجمَّعٌ في
 * لوحته وكفى. **وثقةُ من يعمل بالعمولة تُبنى على أن يُراجع بنفسه لا
 * على أن يُصدّق.**
 *
 * **والملغى يبقى في القائمة لا يُحذف سطرُه** — **إخفاؤه يجعل المجموعَ
 * لا يُطابَق، وهو نقيضُ الشفافيّة.**
 */
@Serializable
data class RepOrderLine(
    val number: Long = 0,
    val status: String = "",
    @SerialName("cancel_reason") val cancelReason: String = "",
    val total: Long = 0,
    val subtotal: Long = 0,
    @SerialName("delivery_fee") val deliveryFee: Long = 0,
    @SerialName("platform_commission") val platformCommission: Long = 0,
    /** **ما كان سيُحتسب لولا الإلغاء** — يُعرض مشطوباً ولا يدخل مجموعا. */
    @SerialName("forfeited_commission") val forfeitedCommission: Long = 0,
    @SerialName("forfeited_share") val forfeitedShare: Long = 0,
    @SerialName("my_share") val myShare: Long = 0,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("delivered_at") val deliveredAt: String? = null,
)

@Serializable
data class Lead(
    val id: String = "",
    @SerialName("store_name") val storeName: String = "",
    @SerialName("owner_name") val ownerName: String = "",
    val phone: String = "",
    val area: String = "",
    val status: String = "",
    val note: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

/**
 * **عميلٌ يُسجَّل من الميدان.**
 *
 * **والنقطةُ اختياريّةٌ في العقد** — **وليست اختياريّةً في الواقع**:
 * متجرٌ بلا نقطةٍ لا يُحسب توصيلُه ولا يُسنَد لأقرب سائق.
 */
@Serializable
data class NewLead(
    @SerialName("store_name") val storeName: String,
    @SerialName("owner_name") val ownerName: String,
    val phone: String,
    val area: String,
    @SerialName("category_id") val categoryId: String,
    /** **كلمةُ مرورِ صاحب المتجر** — يدخل بها يومَ يُوافَق عليه. */
    val password: String,
    val lat: Double? = null,
    val lng: Double? = null,
)

@Serializable
data class RepCategory(
    val id: String = "",
    val name: String = "",
    val icon: String = "",
)
