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

    /**
     * **ما طلبه الزبونُ بلفظه** — وهو كلُّ ما يعرفه السائقُ قبل أن
     * يتّفقا في المحادثة.
     *
     * (تصحيحُ المالك ٢٠٢٦-٠٨-١٣: «الطلبُ الخاصُّ يجب أن يُذكر به ما
     *  نوعُ الطلب — يعني وصفٌ للطلب».)
     *
     * **والمحرّكُ يرسله منذ اليوم الأوّل** (`custom_request`) —
     * **والتطبيقُ لم يكن يقرؤه.** فبطاقةٌ تقول «طلب خاصّ» وحدَها
     * **تطلب من السائق أن يقبل ما لا يعرفه**: أدواءٌ من صيدليّة أم
     * أسمنتٌ من مستودع؟ **والفرقُ بينهما درّاجةٌ ووقت.**
     */
    @SerialName("custom_request") val customRequest: String = "",

    /**
     * **ما وُثّق من ثمنٍ وأجرة** — و`null` تعني **«لم يُوثَّق بعد»**.
     *
     * **وهي علامةُ الطور في الطلب الخاصّ**: قبلها يتّفق، **وبعدها
     * يشتري.** (والمحرّك يقولها هكذا في `driver_handlers.go`.)
     *
     * **ولا حالَ ثانيةً في المحرّك تفرّق بينهما** — الحالُ يبقى
     * `assigned` قبل التوثيق وبعده، **والفارقُ طابعُ وقتٍ لا حال.**
     */
    @SerialName("custom_goods_amount") val customGoods: Long? = null,
    @SerialName("custom_fee") val customFee: Long? = null,

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
/**
 * ══════════════════════════════════════════════════════════════════════
 * **بديلٌ واحد — أرقامٌ لا صفات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٧، البنود ٥ و٦ و٣٣.)
 *
 * **لا «الأسرع» ولا «الأقصر»** — أمرُ المالك. **بل فرقان بالمتر
 * والثانية، والواجهةُ تصوغهما**: «أقصرُ بـ٤٫٥كم · +١٢د».
 *
 * **وجاهزٌ للملاحة**: `cumulativeM` و`maneuvers` معه في الجلبة نفسِها،
 * **فلا نداءَ ثانٍ عند الاختيار** (البند ٢١).
 */
@Serializable
data class RouteAlternative(
    @SerialName("route_id") val routeId: String = "",
    @SerialName("distance_m") val distanceM: Double = -1.0,
    @SerialName("engine_duration_s") val engineDurationS: Double = -1.0,
    @SerialName("delta_distance_m") val deltaDistanceM: Double = 0.0,
    @SerialName("delta_duration_s") val deltaDurationS: Double = 0.0,
    @SerialName("shared_ratio") val sharedRatio: Double = 0.0,
    /** **موضعُ القرار الأوّل** — إغلاقُ صحّة ٧، وعليه يُبنى التقادم. */
    @SerialName("decision_divergence_m") val decisionDivergenceM: Double = -1.0,
    @SerialName("first_divergence_m") val firstDivergenceM: Double = -1.0,
    val points: List<List<Double>> = emptyList(),
    @SerialName("cumulative_m") val cumulativeM: List<Double> = emptyList(),
    val maneuvers: List<RouteManeuver> = emptyList(),
) {
    /** **صالحٌ للملاحة** — كما في `OrderRoute.hasNavigation`. */
    val hasNavigation: Boolean
        get() = points.size >= 2 &&
            cumulativeM.size == points.size &&
            maneuvers.isNotEmpty()
}

@Serializable
data class OrderRoute(
    val available: Boolean = false,
    @SerialName("distance_m") val distanceM: Double = -1.0,
    @SerialName("duration_s") val durationS: Double = -1.0,
    val points: List<List<Double>> = emptyList(),
    // ══════════════════════════════════════════════════════════════════
    // **وبياناتُ الملاحة — المرحلة ٢**
    // ══════════════════════════════════════════════════════════════════
    //
    // (أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
    //
    // **ولها افتراضاتٌ فارغة**: خادمٌ لم يُحدَّث بعد، أو مسارٌ من
    // مخبأٍ قديم، أو محرّكٌ ردّ بلا خطوات — **والخريطةُ ترسم كما
    // كانت.**
    //
    // **ولا كلمةَ من OSRM هنا**: `kind` مفاهيمُنا، **فيومَ يُبدَّل
    // المحرّكُ لا يُمسّ التطبيق.**
    @SerialName("cumulative_m") val cumulativeM: List<Double> = emptyList(),
    val maneuvers: List<RouteManeuver> = emptyList(),

    // ══════════════════════════════════════════════════════════════════
    // **والبدائلُ — المرحلة ٧، بطلبٍ لا افتراضاً**
    // ══════════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ١٦.)
    //
    // **لا تُرسَل إلّا لمن طلبها** (`?alternatives=true`) — **فعميلٌ
    // قديمٌ لا يحمل حمولتَها.** وافتراضاتُها فارغةٌ فلا يسقط شيء.
    //
    // **و`durationS` أعلاه معدَّلةٌ بسرعة السائق** — دلالةٌ قديمةٌ لا
    // تُكسر. **والمقارنةُ بين المسارات على `engineDurationS` وحدَها**
    // (البند ٤).
    @SerialName("route_id") val routeId: String = "",
    val target: String = "",
    @SerialName("weight_name") val weightName: String = "",
    @SerialName("engine_duration_s") val engineDurationS: Double = -1.0,
    val alternatives: List<RouteAlternative> = emptyList(),
) {
    /**
     * **أثمّةَ ما يكفي للملاحة؟**
     *
     * **ومسارٌ بلا مناوراتٍ يُرسم ولا يُرشِد** — ولا يسقط شيء.
     */
    val hasNavigation: Boolean
        get() = maneuvers.isNotEmpty() && cumulativeM.size == points.size
}

/**
 * **مناورةٌ واحدةٌ على المسار — بمفاهيم رحّال غو.**
 *
 * (المرحلة ٢، ٢٠٢٦-٠٨-٢٠.)
 *
 * **والمرجعُ `atDistanceM` لا `atIndex`** (تصحيحُ المالك): الفهرسُ
 * يتبدّل إن بُسِّطت الهندسةُ يوماً، **والمسافةُ على الطريق لا
 * تتبدّل.**
 *
 * **و`kind` نصٌّ لا تعداد**: نوعٌ جديدٌ من الخادم **يُقرأ نصّاً ولا
 * يُسقط التطبيق** — وتعدادُ كوتلن يرمي على قيمةٍ لا يعرفها.
 */
@Serializable
data class RouteManeuver(
    val kind: String = "UNKNOWN",
    val modifier: String? = null,
    @SerialName("at_distance_m") val atDistanceM: Double = 0.0,
    @SerialName("at_index") val atIndex: Int = 0,
    @SerialName("step_distance_m") val stepDistanceM: Double = 0.0,
    @SerialName("step_duration_s") val stepDurationS: Double = 0.0,
    /** **اختياريٌّ بالكامل** — ٨٣٪ من خطوات الرقّة بلا اسم. */
    @SerialName("street_name") val streetName: String? = null,
    @SerialName("roundabout_exit") val roundaboutExit: Int? = null,
    @SerialName("roundabout_name") val roundaboutName: String? = null,
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
    /**
     * **أيقبل متجرُه الإرجاع** — ومن لا يقبله لا يُعرض على سائقه زرٌّ
     * يُضغط فيُردّ.
     */
    /**
     * **ما اتُّفق عليه في الطلب الخاصّ** — ولا يُكتب في أعمدة المال.
     *
     * (قِيس في دورةٍ حقيقيّةٍ ٢٠٢٦-٠٨-١٣: طلبٌ خاصٌّ سُلِّم باتّفاق
     *  ٥٠٠٠ + ١٠٠٠، **وسجلُّه يقول «٠ ل.س».**)
     *
     * **والمنصّةُ توثّق ولا تحاسب** — فيبقى `total` صفراً عمداً، **وهو
     * صحيحٌ في الدفتر وكاذبٌ في عين صاحبه**: من فتح سجلَّه ليتذكّر ما
     * حمل وجد صفرا.
     */
    @SerialName("custom_goods_amount") val customGoods: Long? = null,
    @SerialName("custom_fee") val customFee: Long? = null,
    @SerialName("merchant_accepts_returns") val acceptsReturns: Boolean = false,
    /**
     * **متى أُعيدت البضاعة** — وفارغٌ يعني في يده بعد.
     *
     * **والبضاعةُ التي تعذّر تسليمُها تبقى معه** — ومستحقُّ المتجر
     * مدفوعٌ على شيءٍ رجع. **فحتّى تُسجَّل الإعادةُ يبقى الدفترُ يقول
     * غيرَ الحقيقة.**
     */
    @SerialName("returned_at") val returnedAt: String? = null,
)

/**
 * **سببُ بلاغٍ كما يرسله المحرّك** — رمزٌ ومن هو عليه.
 *
 * **والرموزُ من الخادم لا تُكتب في التطبيق**: قائمةٌ في مكانين تفترق
 * حين يُضاف سببٌ في أحدهما، **فيرسل التطبيقُ رمزاً لا يعرفه الخادم.**
 */
@Serializable
data class ReportReason(
    val code: String = "",
    /** `merchant` · `customer` · فارغٌ لـ«سببٌ آخر». */
    val against: String = "",
)

@Serializable
data class ReportReasons(val reasons: List<ReportReason> = emptyList())

/**
 * **تقييمُ السائق للمتجر — محوران لا نجمةٌ واحدة.**
 *
 * **«المتجر سيّئ» لا تُصلح شيئا**: أبطيءٌ في التجهيز أم سيّئُ التعامل؟
 * **والمكتبُ يعالج الاثنين بطريقتين** — فيُسأل عنهما منفصلين.
 */
@Serializable
data class MerchantRatingInput(
    @SerialName("speed_stars") val speedStars: Int,
    @SerialName("conduct_stars") val conductStars: Int,
    val comment: String = "",
)

/** **ما يُرسَل في البلاغ** — رمزُ السبب وتفصيلٌ اختياريّ. */
@Serializable
data class ReportInput(val reason: String, val note: String = "")


// ═══════════════════════════════════════════════════════════════════════
// **ارتباطُ الأثر بالمسار — المرحلة ٨ب**
// ═══════════════════════════════════════════════════════════════════════
//
// **ولا كلمةَ محرّكٍ فيها** — قراءاتٌ تُرسَل وحكمٌ يعود.

/** **قراءةُ موضعٍ تُرسَل للمقارنة.** */
@Serializable
data class CorrelationFix(
    val lat: Double,
    val lng: Double,
    @SerialName("accuracy_m") val accuracyM: Double,
    @SerialName("at_ms") val atMs: Long,
)

/**
 * **حكمُ الخادم.**
 *
 * **و`status` نصٌّ لا تعداد** — فردٌّ بحالةٍ لا نعرفها **يُقرأ ولا
 * يُسقط التطبيق** (كما في كلّ عقود المشروع).
 */
@Serializable
data class CorrelationVerdict(
    val status: String = "INSUFFICIENT_DATA",
    @SerialName("evidence_age_ms") val evidenceAgeMs: Long = 0,
)
