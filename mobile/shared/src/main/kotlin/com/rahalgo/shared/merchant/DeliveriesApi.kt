package com.rahalgo.shared.merchant

import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«لدي توصيلة» — المتجرُ يطلب سائقاً لغرضٍ جاهزٍ عنده** (الخطوة ١٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٩-٢٩ — والعقدُ في `docs/DELIVERY-MONEY-CONTRACT.md`.)
 *
 * **زبونٌ اشترى من المتجر خارجَ رحّال غو** (اتّصالٌ أو واتساب)، والغرضُ جاهز.
 * **لا أصنافَ ولا سلّة** — اسمُ المستلِم ورقمُه وعنوانُه ونقطتُه، ووصفُ ما يُحمل،
 * **ومن يدفع الأجرة.**
 */
@Serializable
data class DeliveryQuote(
    val fee: Long = 0,
    @SerialName("zone_name") val zoneName: String = "",
    @SerialName("wallet_balance") val walletBalance: Long = 0,
    @SerialName("credit_limit") val creditLimit: Long = 0,
    @SerialName("credit_owed") val creditOwed: Long = 0,
    /** **أيكفيه أن يدفع هو؟** — بقاعدة الخادم نفسِها، لا تقديرٍ في الجهاز. */
    @SerialName("merchant_can_pay") val merchantCanPay: Boolean = false,
)

@Serializable
data class NewDelivery(
    @SerialName("recipient_name") val recipientName: String,
    @SerialName("recipient_phone") val recipientPhone: String,
    @SerialName("address_text") val addressText: String,
    /** **اختياريّة** — فراغُها «لا عنوانَ على الخريطة»، والأجرةُ من منطقة المتجر. */
    val lat: Double? = null,
    val lng: Double? = null,
    @SerialName("parcel_note") val parcelNote: String = "",
    @SerialName("driver_note") val driverNote: String = "",
    /** `merchant` (محفظتي) · `merchant_cash` (أنا نقداً) · `recipient` (المستلم نقداً). */
    @SerialName("fee_payer") val feePayer: String,
)

/**
 * **توصيلةٌ كما تُعرض لصاحبها.**
 *
 * **والمستلِمُ في موضع الزبون** (`customer_name`/`customer_phone`) — كما يراه
 * السائقُ والمكتب، **فلا حقلان لشيءٍ واحد.**
 */
@Serializable
data class Delivery(
    val id: String = "",
    val number: Long = 0,
    val status: String = "",
    @SerialName("customer_name") val recipientName: String = "",
    @SerialName("customer_phone") val recipientPhone: String = "",
    @SerialName("address_text") val addressText: String = "",
    @SerialName("parcel_note") val parcelNote: String = "",
    @SerialName("fee_payer") val feePayer: String = "",
    @SerialName("delivery_fee") val fee: Long = 0,
    /** **أُسند سائق؟** — ولا هويّةَ سائقٍ لدى المتجر (كسائر طلباته). */
    @SerialName("driver_assigned") val driverAssigned: Boolean = false,
    @SerialName("picked_up_at") val pickedUpAt: String? = null,
    @SerialName("delivered_at") val deliveredAt: String? = null,
    @SerialName("closed_at") val closedAt: String? = null,
    @SerialName("cancel_reason") val cancelReason: String? = null,
    /** **أحُدّدت نقطةُ التسليم؟** — وإلّا فالسائقُ يتّصل بالمستلِم. */
    @SerialName("dropoff_known") val dropoffKnown: Boolean = true,
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class DeliveriesPage(val deliveries: List<Delivery> = emptyList())

@Serializable
data class CreatedDelivery(val id: String = "")

class DeliveriesApi(private val api: ApiClient) {

    /** **والنقطةُ اختياريّة** — بلاها فالأجرةُ من منطقة المتجر. */
    suspend fun quote(storeId: String, lat: Double?, lng: Double?): DeliveryQuote =
        api.call(
            "/api/v1/merchant/stores/$storeId/delivery-quote" +
                if (lat != null && lng != null) "?lat=$lat&lng=$lng" else "",
        )

    /**
     * **بمفتاح منع التكرار** — توصيلةٌ ضاع ردُّها فأُعيدت **لا تُنشأ ثانيةً ولا
     * يُخصم أجرُها مرّتين.** والمفتاحُ يثبت حتّى تنجح (`Attempt`).
     */
    suspend fun create(storeId: String, body: NewDelivery, idempotencyKey: String): CreatedDelivery =
        api.call(
            "/api/v1/merchant/stores/$storeId/deliveries",
            HttpMethod.Post,
            body,
            idempotencyKey = idempotencyKey,
        )

    /** **صفحةً صفحة** — خمسون في كلّ صفحة، الأحدثُ أوّلاً. */
    suspend fun list(storeId: String, page: Int = 1): DeliveriesPage =
        api.call("/api/v1/merchant/stores/$storeId/deliveries?page=$page")

    /** **توصيلةٌ واحدة** — لشاشة المراقبة، تُقرأ كلَّ ثوانٍ ما دامت جارية. */
    suspend fun get(orderId: String): Delivery =
        api.call("/api/v1/merchant/deliveries/$orderId")

    /** **يُلغيها ما دام الغرضُ عنده** — والمالُ يعود لمن دفعه (الخادمُ يحكم). */
    suspend fun cancel(orderId: String): Delivery =
        api.call(
            "/api/v1/merchant/deliveries/$orderId/cancel",
            HttpMethod.Post,
            emptyMap<String, String>(),
        )
}
