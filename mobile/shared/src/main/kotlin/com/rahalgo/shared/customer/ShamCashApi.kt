package com.rahalgo.shared.customer

import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Paying with Sham Cash. Switched off on the server until the owner has credentials:
 * [ShamConfig.enabled] is false (or the call 404s on an old server) and every
 * Sham Cash element in the apps stays hidden.
 *
 * No business logic here: the server computes the amount and creates the order only
 * after the payment is confirmed.
 */
class ShamCashApi(private val api: ApiClient) {

    suspend fun config(): ShamConfig = api.call("/api/v1/my/shamcash")

    suspend fun topup(amount: Long): ShamStarted =
        api.call("/api/v1/my/shamcash/topups", HttpMethod.Post, ShamTopupBody(amount, RETURN_APP))

    suspend fun order(order: ShamOrderBody): ShamStarted =
        api.call("/api/v1/my/shamcash/orders", HttpMethod.Post, ShamOrderRequest(order, RETURN_APP))

    suspend fun bill(id: String): Bill =
        api.call<ShamBillReply>("/api/v1/my/shamcash/bills/$id").bill

    companion object {
        /** Tells the server to redirect the browser back to a link that opens the app. */
        const val RETURN_APP = "app"
    }
}

@Serializable
data class ShamConfig(
    val enabled: Boolean = false,
    val min: Long = 0,
    val max: Long = 0,
    @SerialName("expires_minutes") val expiresMinutes: Int = 30,
)

@Serializable
data class Bill(
    val id: String = "",
    val number: String = "",
    /** `topup` or `order`. */
    val purpose: String = "",
    val amount: Long = 0,
    /** `pending` `paid` `expired` `failed` `refunded` `partly_refunded`. */
    val status: String = "pending",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("expires_at") val expiresAt: String = "",
    @SerialName("paid_at") val paidAt: String? = null,
    @SerialName("order_id") val orderId: String? = null,
    @SerialName("order_number") val orderNumber: Long? = null,
    /** Non-empty = the money reached the wallet but the order could not be created. */
    @SerialName("order_error") val orderError: String = "",
    @SerialName("payment_url") val paymentUrl: String = "",
) {
    val pending: Boolean get() = status == "pending"
    val paid: Boolean get() = status == "paid" || status == "partly_refunded"
    val isFinal: Boolean get() = !pending
}

@Serializable
data class ShamStarted(
    val bill: Bill = Bill(),
    @SerialName("payment_url") val paymentUrl: String = "",
)

@Serializable
data class ShamBillReply(val bill: Bill = Bill())

@Serializable
data class ShamTopupBody(
    val amount: Long,
    @SerialName("return") val returnTo: String,
)

/** The body of POST /orders, without payment_method. */
@Serializable
data class ShamOrderBody(
    val items: List<CartLine>,
    @SerialName("address_text") val addressText: String,
    val lat: Double,
    val lng: Double,
    @SerialName("promo_code") val promoCode: String = "",
    val notes: String = "",
)

@Serializable
data class ShamOrderRequest(
    val order: ShamOrderBody,
    @SerialName("return") val returnTo: String,
)
