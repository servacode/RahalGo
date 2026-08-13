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
    /**
     * **عنوان الاستلام كما كُتب** — لا اسم منطقة.
     *
     * (تصحيح المالك ٢٠٢٦-٠٨-١٢.) **والمنطقة وحدة تسعير قد تشمل المدينة
     * كلّها**، فيقرأ السائق «مركز المدينة» في الطرفين ولا يعرف من أين
     * ولا إلى أين.
     *
     * **وهو موضع الاستلام البديل إن وُجد** — البضاعة ليست في المتجر،
     * **ومن قرأ عنوان المتجر ذهب إلى حيث لا شيء.**
     */
    @SerialName("pickup_address") val pickupAddress: String = "",

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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مسارُ الطلب — بالشوارع لا بالهواء**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرار المالك ٢٠٢٦-٠٨-١٢.)
 *
 * **و`available = false` ليست خطأ**: محرّكُ المسارات قد ينام أو لا يجد
 * طريقا، **والشاشة ترسم خطَّها المستقيم كما كانت** ولا تقف.
 *
 * **والنقاط عرضٌ ثمّ طول** — كما تكتبها الشاشة والقاعدة، **والمحرّك
 * يقلبها عن OSRM قبل أن يرسلها** فلا يُقلَب في مكانين.
 */
@Serializable
data class OrderRoute(
    val available: Boolean = false,
    @SerialName("distance_m") val distanceM: Double = -1.0,
    @SerialName("duration_s") val durationS: Double = -1.0,
    val points: List<List<Double>> = emptyList(),
)

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صفحةٌ من سجلّ السائق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٣: «الآن يجب أن نبني سجلّ الطلبات… يجب أن يعرف
 *  السائقُ ماذا عمل بالفترات السابقة» · «واسم سجلّ الطلبات أفضل».)
 *
 * **والاسمُ «سجلّ» لا «سابقة»**: «السابقة» نسبةٌ إلى الآن تُقرأ «ما مضى
 * وانتهى ولا شأنَ لك به»، **و«السجلّ» يقول إنّه مرجعٌ يُعاد إليه** —
 * وهو ما تسمّيه لوحةُ الإدارة، **فكلمةٌ واحدةٌ في المنصّة كلِّها.**
 */
@Serializable
data class HistoryPage(
    val orders: List<HistoryOrder> = emptyList(),
    val total: Int = 0,
)

/**
 * **طلبٌ منتهٍ** — بحاله وتاريخه وما جرى فيه.
 *
 * **ولا أجرةَ في هذا الردّ**: قِيس فوُجد أنّ المحرّك لا يرسل أجرَ
 * السائق مع الطلب — **حُذف من البطاقة بقرار المالك ٢٠٢٦-٠٨-٠٤** («لم
 * أطلبها أصلاً»)، **وموضعُه دفترُه**: المحفظةُ وكشفُ الحساب حيث يُقرأ
 * مجموعا.
 *
 * **وحقلٌ يُخترع هنا يُقرأ صفراً دائما** — ورقمُ أجرٍ صفريٌّ في سجلٍّ
 * أسوأُ من غيابه.
 *
 * **والحقولُ هي التي تقرؤها شاشةُ الويب حرفيّاً** — فلا يفترق سجلّان.
 */
@Serializable
data class HistoryOrder(
    val id: String = "",
    val number: Long = 0,
    val status: String = "",
    @SerialName("merchant_name") val merchantName: String = "",
    @SerialName("customer_name") val customerName: String = "",
    @SerialName("address_text") val addressText: String = "",
    val total: Long = 0,
    @SerialName("cash_due") val cashDue: Long = 0,
    @SerialName("fail_reason") val failReason: String = "",
    @SerialName("created_at") val createdAt: String = "",
    /** **أقيّمتُ متجرَه** — ومن قيّم لا يُعرض عليه الزرُّ ثانيةً. */
    @SerialName("merchant_rated") val merchantRated: Boolean = false,
    /** **وقف عند بابه فعلاً** — ومن لم يقف لا رأيَ له فيه. */
    @SerialName("can_rate_merchant") val canRateMerchant: Boolean = false,
)
