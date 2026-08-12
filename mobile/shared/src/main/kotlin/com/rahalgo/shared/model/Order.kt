package com.rahalgo.shared.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الطلب كما يراه السائق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (`driverOrder` في `server/driver_handlers.go` — نفس الأسماء.)
 *
 * # ولا رقم للزبون هنا
 *
 * (قرار المالك ٢٠٢٦-٠٨-٠٩: «لازم الاثنان لا يقدران يوصلان لبعض إلّا عن
 * طريق المنصّة».)
 *
 * **والمحرّك لا يرسله أصلا** — لا يخفيه في الشاشة. **والإخفاء في الشاشة
 * ستارة لا قفل.** والتواصل من قناة الطلب.
 */
@Serializable
data class DriverOrder(
    val id: String = "",
    /** **رقمه المنطوق** — يُقال على الهاتف ويُكتب في الشكوى. */
    val number: Long = 0,
    val status: String = "",
    /**
     * **نوعه** — `standard` أو `custom`.
     *
     * **والخاصّ لا سعر له حتّى يتّفقا** (قرار المالك): السائق يدفع من
     * جيبه ويستردّ عند التسليم، **والمنصّة توثّق ولا تحاسب.**
     */
    val kind: String = "standard",

    @SerialName("merchant_name") val merchantName: String = "",
    @SerialName("merchant_phone") val merchantPhone: String? = null,

    @SerialName("address_text") val addressText: String = "",
    @SerialName("customer_name") val customerName: String = "",

    // ── المواضع على الأرض ──
    /** **باب الزبون** — إلزاميّ في المحرّك (`orders.dropoff NOT NULL`). */
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    /**
     * **نقطة الاستلام** — المتجر، أو موضعٌ بديلٌ إن كانت البضاعة ليست فيه.
     *
     * **وتأتي محلولةً من المحرّك** (`nav_lat`): فلا يحسب التطبيق أيّهما
     * الصحيح، **وحسابٌ يقع في مكانين يفترق.**
     *
     * **وفارغة تعني متجراً بلا دبّوس** — ومنذ ٢٠٢٦-٠٨-١٢ لا يُفتح متجر
     * بلا موضع، **لكن القديم منها قد يبقى.**
     */
    @SerialName("nav_lat") val navLat: Double? = null,
    @SerialName("nav_lng") val navLng: Double? = null,

    val total: Long = 0,
    /** **ما يقبضه نقدا من الزبون** — وصفر يعني مدفوع سلفا. */
    @SerialName("cash_due") val cashDue: Long = 0,
    @SerialName("items_count") val itemsCount: Int = 0,
    /** **ما يكسبه هو** — لا ما يقبضه للمتجر. */
    @SerialName("delivery_fee") val deliveryFee: Long = 0,
    /** **اسم الحيّ لا الإحداثيات** — «حي الروضة» يعرفه في لحظة. */
    @SerialName("pickup_area") val pickupArea: String = "",
    @SerialName("dropoff_area") val dropoffArea: String = "",

    /** **كم بينه وبين المتجر بالمتر** — **وسالب يعني «لا يُعرف»**، لا قريب. */
    @SerialName("to_pickup_m") val toPickupM: Double = -1.0,
    /** طول المشوار: من المتجر إلى باب الزبون. */
    @SerialName("leg_m") val legM: Double = -1.0,

    /** متى ينقضي دوره على هذا الطلب — **وفارغ يعني بلا مهلة.** */
    @SerialName("offer_expires_at") val offerExpiresAt: String? = null,
    @SerialName("ready_at") val readyAt: String? = null,
    @SerialName("prep_minutes") val prepMinutes: Int? = null,
    @SerialName("accepted_at") val acceptedAt: String? = null,
    @SerialName("created_at") val createdAt: String = "",
)

/**
 * **أسباب التعذّر — مصنّفة لا حرّة.**
 *
 * (`orders/failreasons.go` — ولكلّ سبب ذنب يقرّر التعويض.)
 *
 * **وتُطلب من المحرّك لكلّ حال** (`at_pickup` أو `at_dropoff`): أسباب
 * المتجر لا تصلح عند باب الزبون، **ومن عرضها كلّها** جعله يختار سببا
 * يرفضه المحرّك.
 */
@Serializable
data class FailReasons(val reasons: List<FailReasonItem> = emptyList())

@Serializable
data class FailReasonItem(val code: String = "", val fault: String = "")
