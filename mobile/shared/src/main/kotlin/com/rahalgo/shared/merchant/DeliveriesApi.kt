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
    /** **المتاحُ للدفع** — الرصيدُ ناقصَ المحجوز (٢٠٢٦-١٠-١٥). */
    @SerialName("available_balance") val availableBalance: Long = 0,
    /**
     * **أيكفي المتاحُ في محفظته الأجرة؟** — بقاعدة الخادم نفسِها، لا تقديرٍ في الجهاز.
     * **ولا دين** (قرارُ المالك ٢٠٢٦-١٠-١٥): لا يكفي ⇒ يُعطَّل «من محفظتي».
     */
    @SerialName("merchant_can_pay") val merchantCanPay: Boolean = false,
    /** **من أين الأجرة** — `distance` · `no_point` (الأساسُ وحدَه) · `flat`. */
    @SerialName("fee_basis") val feeBasis: String = "",
    @SerialName("distance_m") val distanceM: Long = 0,
    val mode: String = "send",
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
    /** **«توصيل» أو «إحضار»** — `send` · `fetch` (قرارُ المالك ٢٠٢٦-١٠-١٥). */
    @SerialName("delivery_mode") val deliveryMode: String = "send",
    /**
     * **مبلغٌ يقبضه السائقُ من المستلِم لك** — عرضٌ لا يدخل الحساب، وفراغُه لا شيء.
     * **وبعملته كما كُتب** (قرارُ المالك ٢٠٢٦-١٠-٢٠): الليرةُ صحيحة، **والدولارُ بمنزلتين** (١٢٫٥).
     */
    @SerialName("collect_amount") val collectAmount: Double? = null,
    /** `SYP` أو `USD` — **والخادمُ يحوّل الدولارَ إلى سنتات بنفسه.** */
    @SerialName("amount_currency") val amountCurrency: String = "SYP",
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
    /** **«توصيل» أو «إحضار»** (٢٠٢٦-١٠-١٥). */
    @SerialName("delivery_mode") val deliveryMode: String = "send",
    /** **ما يقبضه السائقُ من المستلِم لك** — خارجَ الحساب. */
    @SerialName("collect_amount") val collectAmount: Long? = null,
    /**
     * **«إحضار»: ما دفعه السائقُ فعلاً بمكان الاستلام** (قرارُ المالك ٢٠٢٦-١٠-١٦) — ونسختُه وحالُه
     * (`pending` ينتظر موافقتك · `approved` · `rejected`). **والتقديرُ `collectAmount`.**
     */
    @SerialName("fetch_amount") val fetchAmount: Long? = null,
    @SerialName("fetch_amount_version") val fetchAmountVersion: Long = 0,
    @SerialName("fetch_amount_status") val fetchAmountStatus: String = "",
    /** **عملةُ المبلغين** (٢٠٢٦-١٠-٢٠) — `SYP` أو `USD`، **والدولارُ بالسنتات** (١٢٥٠ = ١٢٫٥٠$). */
    @SerialName("amount_currency") val amountCurrency: String = "SYP",
    /**
     * **كودُ الاستلام في «إحضار»** (قرارُ المالك ٢٠٢٦-١٠-١٦) — لك وحدَك: تعطيه للسائق لما يوصلك
     * الغرض. **وفارغٌ قبل الاستلام وفي «توصيل»** (هناك الكودُ للمستلِم).
     */
    @SerialName("delivery_code") val deliveryCode: String? = null,
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

/** **رابطُ موقعٍ قُرئ نقطة** — و`found=false` رابطٌ لا موقعَ فيه (٢٠٢٦-١٠-٠٨). */
@Serializable
data class ResolvedLocation(val found: Boolean = false, val lat: Double = 0.0, val lng: Double = 0.0)

@Serializable
private data class ResolveLocationInput(val url: String)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«اطلب الموقع من المستلم»** (قرارُ المالك ٢٠٢٦-١٠-٢٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * يصل المستلِمَ بالواتساب رابطٌ لصفحة «أرسل موقعي»، **وتصل نقطتُه إلى النموذج.**
 * **وإن لم تصل الرسالةُ** (`whatsappSent=false`) يشارك المتجرُ الرابطَ بنفسه (`waUrl`).
 */
@Serializable
data class LocationRequest(
    val id: String = "",
    @SerialName("whatsapp_sent") val whatsappSent: Boolean = false,
    val link: String = "",
    @SerialName("share_text") val shareText: String = "",
    @SerialName("wa_url") val waUrl: String = "",
)

/** **حالُ الطلب** — `pending` · `received` · `expired`، والنقطةُ ودقّتُها بعد الوصول. */
@Serializable
data class LocationRequestStatus(
    val id: String = "",
    val status: String = "pending",
    val lat: Double? = null,
    val lng: Double? = null,
    @SerialName("accuracy_m") val accuracyM: Double? = null,
)

@Serializable
private data class LocationRequestInput(val phone: String, val mode: String)

@Serializable
private data class FetchDecision(@SerialName("expected_version") val expectedVersion: Long)

class DeliveriesApi(private val api: ApiClient) {

    /** **يقرأ رابطَ موقعٍ شاركه الزبونُ** — نقطةً على خريطتنا (٢٠٢٦-١٠-٠٨). */
    suspend fun resolveLocation(storeId: String, url: String): ResolvedLocation =
        api.call("/api/v1/merchant/stores/$storeId/resolve-location", io.ktor.http.HttpMethod.Post, ResolveLocationInput(url))

    /** **يطلب موقعَ المستلِم برسالة واتساب** — `mode`: `send` أو `fetch`. */
    suspend fun requestLocation(storeId: String, phone: String, mode: String): LocationRequest =
        api.call(
            "/api/v1/merchant/stores/$storeId/location-requests",
            HttpMethod.Post,
            LocationRequestInput(phone, mode),
        )

    /** **أوصل الموقع؟** — يُسأل كلَّ ثوانٍ ما دام ينتظر. */
    suspend fun locationRequest(id: String): LocationRequestStatus =
        api.call("/api/v1/merchant/location-requests/$id")

    /**
     * **والنقطةُ اختياريّة** — بلاها فالأجرةُ الأساسُ وحدَه. **والوضعُ يغيّر المسافة**:
     * الإحضارُ ذهابٌ وإياب (قرارُ المالك ٢٠٢٦-١٠-١٥).
     */
    suspend fun quote(storeId: String, lat: Double?, lng: Double?, mode: String = "send"): DeliveryQuote =
        api.call(
            "/api/v1/merchant/stores/$storeId/delivery-quote?mode=$mode" +
                if (lat != null && lng != null) "&lat=$lat&lng=$lng" else "",
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

    /**
     * **«إحضار»: يوافق على المبلغ الذي دفعه السائق أو يرفضه** (قرارُ المالك ٢٠٢٦-١٠-١٦) — **بنسخته**:
     * مبلغٌ تبدّل بعد أن رآه يُردّ (`fetch_amount_stale`) فلا يُقفَل ما لم يره.
     */
    suspend fun decideFetchAmount(orderId: String, approve: Boolean, version: Long): Delivery =
        api.call(
            "/api/v1/merchant/deliveries/$orderId/fetch-amount/" + if (approve) "approve" else "reject",
            HttpMethod.Post,
            FetchDecision(version),
        )

    /** **يُلغيها ما دام الغرضُ عنده** — والمالُ يعود لمن دفعه (الخادمُ يحكم). */
    suspend fun cancel(orderId: String): Delivery =
        api.call(
            "/api/v1/merchant/deliveries/$orderId/cancel",
            HttpMethod.Post,
            emptyMap<String, String>(),
        )
}
