package com.rahalgo.shared.merchant

import com.rahalgo.shared.model.Envelope
import com.rahalgo.shared.model.WalletStatement
import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أبوابُ المتجر — كما تناديها بوّابتُه في الويب حرفاً بحرف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (`backend/internal/server/server.go` — كتلةُ `/merchant`، قِيست نداءاتُها
 *  بنداً بنداً ٢٠٢٦-٠٨-٢٣.)
 *
 * **وكلُّها تحت `RequireRoles("merchant")`** — يفرضه المحرّكُ لا الشاشة.
 *
 * # ولماذا وُجد هذا الملفّ
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٣: «بناء تطبيق المتجر بشكل كامل ومتكامل».)
 *
 * **وكانت بوّابةُ الويب هي بيتَ المتجر الوحيد** — يفتح متصفّحاً ليقبل
 * طلباً. **وصاحبُ مطعمٍ يداه في العجين لا يفتح متصفّحاً**: يحتاج جرساً
 * يرنّ وزرّاً واحداً.
 *
 * # ولا منطقَ هنا
 *
 * **نداءاتٌ وأنواعٌ لا غير** (`GROUND-RULES.md` §7.1) — والقرارُ في
 * المحرّك، والعرضُ في الشاشة. **وطبقةٌ وسطى تقرّر تصير محرّكاً ثانياً
 * يفترق عن الأوّل.**
 */
class MerchantApi(private val api: ApiClient) {

    // ══════════════════════════════════════════════════════════════════
    // **متاجرُه** — واحدٌ في الغالب، وقد يملك اثنين
    // ══════════════════════════════════════════════════════════════════
    //
    // **ولا يُفترض أنّه واحد**: من ملك فرعين ثمّ رأى فرعاً واحداً ظنّ
    // الآخرَ ضاع. **والقائمةُ تُقرأ كما تأتي.**

    // ══════════════════════════════════════════════════════════════════
    // **ومغلَّفٌ لا مصفوفة** — كشفه أوّلُ تشغيلٍ على الجهاز ٢٠٢٦-٠٨-٢٣
    // ══════════════════════════════════════════════════════════════════
    //
    // **كتبتُ العقدَ تخميناً فردّ المحرّكُ كائناً** — والشاشةُ عرضت
    // `Expected start of the array '[' but had '{'`.
    //
    // **و`self_manage_orders` يأتي معها عمداً**: إعدادُ منصّةٍ لا يملك
    // المتجرُ قراءتَه من بابه، **ويحتاجه ليعرف أيَعرض أزرارَ القبول** —
    // فالجوابُ مع السؤال لا نداءٌ ثانٍ لسطرٍ واحد.
    suspend fun stores(): StoresPage = api.call("/api/v1/merchant/stores")

    /** **إنذاراتُه** — ومن أُنذر يرى إنذارَه قبل أن يُحظر. */
    suspend fun warnings(): WarningsPage = api.call("/api/v1/merchant/warnings")

    // ══════════════════════════════════════════════════════════════════
    // **الطلبات — قلبُ التطبيق**
    // ══════════════════════════════════════════════════════════════════

    /** **طلباتُ متجرٍ** — صفحةٌ بعدّادات، كما يردّها المحرّك. */
    suspend fun orders(storeId: String): OrdersPage =
        api.call("/api/v1/merchant/stores/$storeId/orders")

    /**
     * **سجلُّ ما انتهى** — قسمٌ منفصلٌ كما في الويب.
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-٢٣.)
     *
     * **و«الطلبات» لاستقبال الجديد وحدَه** — **وخلطُ ما يُعمل بما مضى
     * يجعل صاحبَ المتجر يبحث عن طلبٍ جارٍ بين مئةٍ سُلّمت.**
     *
     * **والمحرّكُ يفرض `closed_only` من نفسه في وضع المنصّة** — وهذا
     * النداءُ يطلبها صراحةً في الوضعين.
     */
    suspend fun history(storeId: String, page: Int = 1): OrdersPage =
        api.call("/api/v1/merchant/stores/$storeId/orders?closed_only=true&page=$page")

    /** **طلبٌ بتفصيله** — أصنافُه وخياراتُه وملاحظتُه. */
    suspend fun order(orderId: String): MerchantOrder =
        api.call("/api/v1/merchant/orders/$orderId")

    /**
     * **ينقل حالَ الطلب** — `accepted` أو `preparing` أو `rejected`.
     *
     * **والمحرّكُ يحرس الانتقال** (`statuses.go`): ما لا يجوز يُردّ،
     * **فلا تحرسه الشاشةُ مرّتين** — تعرضه وتقرأ الجواب.
     */
    suspend fun transition(orderId: String, to: String, note: String = ""): MerchantOrder =
        api.call(
            "/api/v1/merchant/orders/$orderId/transition",
            HttpMethod.Post,
            TransitionInput(to, note),
        )

    /**
     * **«جاهزٌ للاستلام»** — علامةٌ يرفعها من يعرف.
     *
     * **وبابٌ مستقلٌّ عن `transition`** لأنّه ليس انتقالَ حالٍ بل
     * إشارةُ مطبخ: **الطلبُ يبقى `preparing` ويصير السائقُ مدعوّاً.**
     */
    suspend fun markReady(orderId: String): MerchantOrder =
        api.call("/api/v1/merchant/orders/$orderId/ready", HttpMethod.Post, Unit)

    // ══════════════════════════════════════════════════════════════════
    // **القائمة — بضاعتُه بيده**
    // ══════════════════════════════════════════════════════════════════

    suspend fun menu(storeId: String): List<MenuSection> =
        api.call("/api/v1/merchant/stores/$storeId/menu")

    /**
     * **يبدّل توفّرَ صنفٍ بضغطة** — أكثرُ فعلٍ يتكرّر في يومه.
     *
     * **ونفد الصنفُ فعلٌ من ثانية**: لو احتاج فتحَ شاشةِ تعديلٍ كاملةٍ
     * **لَتركه معروضاً ونفد** — فيُطلب ما ليس عنده.
     */
    suspend fun setAvailability(itemId: String, available: Boolean): Ack =
        api.call(
            "/api/v1/merchant/menu/items/$itemId/availability",
            HttpMethod.Patch,
            AvailabilityInput(available),
        )

    /**
     * **أقسامُ السوق بصورها** — ليختار الصنفُ موضعَه، **ولتُبنى شبكةُ
     * الأصناف بوجوهها لا بحروف.**
     */
    suspend fun platformSections(): SectionsPage =
        api.call("/api/v1/merchant/platform-sections")

    /**
     * ══════════════════════════════════════════════════════════════════
     * **أقسامُ هذا المتجر — ما يبيعه هو**
     * ══════════════════════════════════════════════════════════════════
     *
     * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «مو معقول كلُّ الأقسام تطلع عند كلّ
     *  المتاجر… مطعمٌ شو علاقتُه بالأحذية؟»)
     *
     * **و`platformSections` تبقى للاختيار** — منها يعلن ما يبيع.
     * **وهذه تُقرأ في نموذج الصنف**، فلا يرى إلّا ما أعلن.
     *
     * **وفارغةٌ تعني الكلّ** — فمتجرٌ لم يختر بعدُ لا يُحبس بلا أقسام.
     */
    suspend fun storeSections(storeId: String): SectionsPage =
        api.call("/api/v1/merchant/stores/$storeId/sections")

    /** **يستبدل القائمةَ كلَّها** — فمن أزال قسماً يزول فعلاً. */
    suspend fun setStoreSections(storeId: String, ids: List<String>): Ack =
        api.call(
            "/api/v1/merchant/stores/$storeId/sections",
            HttpMethod.Put,
            mapOf("sections" to ids),
        )

    suspend fun createItem(storeId: String, input: MenuItemInput): MenuItem =
        api.call("/api/v1/merchant/stores/$storeId/menu/items", HttpMethod.Post, input)

    suspend fun updateItem(itemId: String, input: MenuItemInput): MenuItem =
        api.call("/api/v1/merchant/menu/items/$itemId", HttpMethod.Patch, input)

    suspend fun deleteItem(itemId: String): Ack =
        api.call("/api/v1/merchant/menu/items/$itemId", HttpMethod.Delete, Unit)

    suspend fun createSection(storeId: String, name: String): MenuSection =
        api.call(
            "/api/v1/merchant/stores/$storeId/menu/sections",
            HttpMethod.Post,
            SectionInput(name),
        )

    // ══════════════════════════════════════════════════════════════════
    // **الساعاتُ والإعدادات**
    // ══════════════════════════════════════════════════════════════════

    suspend fun hours(storeId: String): List<DayHours> =
        api.call("/api/v1/merchant/stores/$storeId/hours")

    suspend fun setHours(storeId: String, days: List<DayHours>): Ack =
        api.call("/api/v1/merchant/stores/$storeId/hours", HttpMethod.Put, HoursInput(days))

    /**
     * **يحفظ إعداداتِه** — ويردّ `{"updated": true}` لا المتجرَ.
     *
     * **فالشاشةُ تُعيد القراءةَ بعده** — **ومن بنى الحالَ من ردٍّ لا
     * يحمله قرأ حقولاً فارغةً فمحا ما على الشاشة.**
     */
    suspend fun settings(storeId: String, input: StoreSettingsInput): Updated =
        api.call("/api/v1/merchant/stores/$storeId/settings", HttpMethod.Patch, input)

    /**
     * ══════════════════════════════════════════════════════════════════
     * **يفتح المتجرَ أو يغلقه**
     * ══════════════════════════════════════════════════════════════════
     *
     * (تصحيحُ المالك ٢٠٢٦-٠٨-٢٣: «المتجرُ لا يستقبل طلبات — إذا كان
     *  مفتوحاً يستقبل وإذا كان مغلقاً لا يستقبل».)
     *
     * **وهذا هو البابُ الوحيدُ للإغلاق** — `/settings` لا يقبل حقلاً
     * له إطلاقاً (قِيس من `merchant_ops_handlers.go`: يقبل مدّةَ
     * التحضير والحدَّ الأدنى والعنوانَ والدبّوس لا غير).
     *
     * **وكنتُ أرسل `accepts_orders` إلى `/settings`** — **فيُقرأ حقلاً
     * مجهولاً ويُطرح صامتاً**: يقلب المفتاحُ نفسَه في الشاشة ولا يتبدّل
     * شيءٌ في القاعدة، **ويظنّ صاحبُ المتجر أنّه أغلق وهو مفتوح.**
     *
     * **و`emergency_closed` جزءٌ من معادلة «مفتوح» نفسِها**
     * (`orders/hours.go`) — فليس إيقافَ استقبالٍ بل إغلاقاً حقيقيّاً.
     */
    suspend fun setClosed(storeId: String, closed: Boolean): ClosedAck =
        api.call(
            "/api/v1/merchant/stores/$storeId/emergency",
            HttpMethod.Post,
            ClosedInput(closed),
        )

    // ══════════════════════════════════════════════════════════════════
    // **المالُ والتقارير**
    // ══════════════════════════════════════════════════════════════════

    /** **محفظتُه** — والكشفُ عامٌّ لكلّ الأدوار، فلا نداءَ خاصّ. */
    suspend fun wallet(): WalletStatement = api.call("/api/v1/me/wallet")

    /**
     * **تقريرُ مدًى** — و`from`/`to` بصيغة `YYYY-MM-DD`.
     *
     * **واليومُ وحدَه يُطلب بمساواتهما** — **ومدًى من سبعة أيّامٍ يُقرأ
     * «اليوم» فيظنّ صاحبُ المتجر يومَه أكبرَ ممّا هو.**
     */
    suspend fun reports(storeId: String, from: String = "", to: String = ""): StoreReport {
        val q = if (from.isEmpty()) "" else "?from=$from&to=$to"
        return api.call("/api/v1/merchant/stores/$storeId/reports$q")
    }

    // ══════════════════════════════════════════════════════════════════
    // **البلاغُ على السائق — وما صار به**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «بسجلّ الطلبات يجب أن يكون هناك زرُّ إبلاغٍ
    //  عن السائق»، و«بالقائمة الجانبيّة قسمُ الشكاوي والبلاغات».)
    //
    // **وشروطُ القبول ثلاثةٌ في المحرّك** (`support/merchant_report.go`)،
    // **تُقرأ في الشاشة قبل أن يُعرض الزرّ** — وإلّا ردّ المحرّكُ خطأً
    // لا يفهمه صاحبُ المتجر:
    //
    //   ١ · الطلبُ مُغلقٌ فعلاً   (`closed_at` ليس فارغاً)
    //   ٢ · له سائقٌ مُسنَد      (بلاغٌ بلا مشكوٍّ عليه يقف بلا طرف)
    //   ٣ · ضمن مهلة الشكوى      (تحدّدها الإعدادات)

    /** **أسبابُه أربعة** — والمحرّكُ يرفض ما سواها. */
    suspend fun reportReasons(): ReasonsPage = api.call("/api/v1/merchant/report-reasons")

    suspend fun reportDriver(orderId: String, reason: String, note: String): Ack =
        api.call("/api/v1/merchant/orders/$orderId/report", HttpMethod.Post,
            ReportInput(reason, note))

    /**
     * **بلاغاتُه هو لا شكاوى الزبائن عليه.**
     *
     * **وليس `/my/tickets`** — ذاك يفلتر بـ`customer_id`، **وبلاغُ المتجر
     * يحمل في تلك الخانة زبونَ الطلب.** فلو رُبط به لرأى صاحبُ المتجر
     * **ما شُكي به عليه**. البابُ الصحيح `/merchant/my-reports` يفلتر
     * بـ`created_by`.
     */
    /**
     * **الشكاوى التي رُفعت على متجره** — (بلاغُ المالك ٢٠٢٦-٠٨-٢٦:
     * «الشكاوي يلي عليه ويلي اله»).
     *
     * **ولا يُذكر من اشتكى** — اسمُ الزبون ليس من حقّ المتجر، **ومن
     * عرفه قد يعاقبه في طلبه القادم.** ورقمُ الطلب يكفي للفهم.
     */
    suspend fun reportsAgainst(storeId: String): ReportsPage =
        api.call("/api/v1/merchant/stores/$storeId/reports-against")

    suspend fun myReports(): ReportsPage = api.call("/api/v1/merchant/my-reports")

    // ══════════════════════════════════════════════════════════════════
    // **وصورةُ الصنف يرفعها بنفسه**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «كلُّ منتجٍ بالإضافة إلى تفاصيله يكون له
    //  صورة».)
    //
    // **وبابُه أضيقُ من باب الإدارة**: `merchantKinds` في المحرّك يحصر
    // الأنواعَ — **ولا يرفع تاجرٌ شعارَ المنصّة.**
    suspend fun uploadItemImage(fileName: String, bytes: ByteArray): String {
        val raw = api.upload(
            "/api/v1/merchant/media", fileName, bytes, mapOf("kind" to "menu_item"),
        )
        return api.json.decodeFromString<Envelope<MediaRef>>(raw).data?.id.orEmpty()
    }

    // **والقائمةُ والساعاتُ مصفوفتان خامّتان** — لا مغلَّفَ لهما،
    // **قِيس من `merchant_handlers.go` و`merchant_ops_handlers.go`.**
}

// ══════════════════════════════════════════════════════════════════════
// **الأنواع — كما يكتبها المحرّك**
// ══════════════════════════════════════════════════════════════════════

/**
 * **متاجرُه ومعها إعدادُ إدارة الطلبات** — مغلَّفٌ واحدٌ من المحرّك.
 *
 * **و`selfManageOrders` يقول أيَعرض أزرارَ القبول**: في وضع «المنصّةُ
 * تدير» **تُخفى الجاريةُ عنه ولا يقبل شيئاً** — والسجلُّ يبقى كاملاً.
 */
@Serializable
data class StoresPage(
    val stores: List<Store> = emptyList(),
    @SerialName("self_manage_orders") val selfManageOrders: Boolean = false,
)

@Serializable
data class Store(
    val id: String = "",
    val name: String = "",
    val status: String = "",
    @SerialName("category_icon") val categoryIcon: String = "",
    @SerialName("logo_thumb_url") val logoThumbUrl: String? = null,
    /**
     * **أغلقه طارئاً** — وهو المفتاحُ الذي بيده فعلاً.
     *
     * **ولا `open_now` في هذا الباب**: الدوامُ يُقرأ من الساعات،
     * **وحقلٌ خمّنتُه ولا وجودَ له يُقرأ فارغاً ويكذب على الشاشة.**
     */
    @SerialName("emergency_closed") val emergencyClosed: Boolean = false,
    @SerialName("default_prep_minutes") val prepMinutes: Int = 0,
    @SerialName("min_order") val minOrder: Long = 0,
    @SerialName("address_text") val addressText: String = "",
    val lat: Double? = null,
    val lng: Double? = null,
)

@Serializable
data class WarningsPage(val warnings: List<Warning> = emptyList())

@Serializable
data class Warning(
    val id: String = "",
    val reason: String = "",
    val note: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

/**
 * **صفحةُ طلباتٍ بعدّاداتها** — كما يردّها المحرّك.
 *
 * **والعدّاداتُ تُقرأ ولا تُحسب في الشاشة**: عدٌّ في الجهاز يقرأ الصفحةَ
 * الحاضرةَ وحدَها، **فيقول «طلبان» ووراءهما عشرون.**
 */
@Serializable
data class OrdersPage(
    val orders: List<MerchantOrder> = emptyList(),
    val total: Int = 0,
    val page: Int = 1,
    @SerialName("per_page") val perPage: Int = 20,
    @SerialName("status_counts") val statusCounts: Map<String, Int> = emptyMap(),
    /**
     * **أرقامُ الطلبات التي يملك الإبلاغَ عنها** — يقولها المحرّك.
     *
     * **ولا تُشتقّ في الشاشة**: شروطُ القبول ثلاثة (`support/merchant_report.go`)،
     * **وأحدُها محجوبٌ عن التطبيق عمداً** — `redactForMerchant` يمحو
     * `driver_id` قبل أن يخرج الردّ. **فزرٌّ يُعرض على طلبٍ بلا سائقٍ
     * يردّ خطأً لا يفهمه صاحبُ المطعم.**
     *
     * **وتخلو ممّا أُبلغ عنه مرّةً** — فلا يُبلَّغ على الطلب مرّتين.
     */
    val reportable: List<String> = emptyList(),
)

/**
 * **طلبٌ كما يراه المتجر — ولا اسمَ زبونٍ ولا رقمَه.**
 *
 * **الزبونُ لا يعرف المتجرَ والمتجرُ لا يحتاج الزبون** — السائقُ بينهما.
 * **ورقمُ زبونٍ في يد متجرٍ بابُ اتّصالٍ مباشرٍ يوماً ما.**
 */
@Serializable
data class MerchantOrder(
    val id: String = "",
    /** **رقمُه المعروض** — عددٌ لا نصّ، كما يكتبه المحرّك. */
    val number: Long = 0,
    val status: String = "",
    val subtotal: Long = 0,
    @SerialName("delivery_fee") val deliveryFee: Long = 0,
    val total: Long = 0,
    /**
     * ══════════════════════════════════════════════════════════════════
     * **شفافيّةُ المتجر — كم أخذت المنصّة وكم بقي له**
     * ══════════════════════════════════════════════════════════════════
     *
     * (بلاغُ المالك ٢٠٢٦-٠٨-٢٦: «سجلّ الطلبات ما فيه شقد المبلغ المباع
     *  وشقد نسبة العمولة للمنصّة — هيك لازم يكون بشفافية».)
     *
     * **وتصل من المحرّك لصاحب المتجر وحدَه** — ولا تُرسل للزبون.
     *
     * **وطلبٌ لم يُسلَّم بعدُ عمولتُه مقدَّرةٌ بالنسبة** — فتُعرض
     * النسبةُ معها كي لا يُقرأ التقديرُ رقماً نهائيّا.
     */
    @SerialName("platform_commission") val platformCommission: Long = 0,
    @SerialName("merchant_net") val merchantNet: Long = 0,
    @SerialName("commission_percent") val commissionPercent: Int = 0,
    val items: List<OrderLine> = emptyList(),
    val notes: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class OrderLine(
    val id: String = "",
    val name: String = "",
    val qty: Int = 1,
    @SerialName("unit_price") val unitPrice: Long = 0,
    val note: String = "",
    /** **ما اختاره الزبون** — حجمٌ وإضافات، وبلاها يُصنع الطلبُ ناقصاً. */
    val options: List<LineOption> = emptyList(),
)

@Serializable
data class LineOption(
    val group: String = "",
    val name: String = "",
    @SerialName("price_delta") val priceDelta: Long = 0,
)

@Serializable
data class MenuSection(
    val id: String = "",
    val name: String = "",
    val items: List<MenuItem> = emptyList(),
)

@Serializable
data class MenuItem(
    val id: String = "",
    val name: String = "",
    val description: String = "",
    /**
     * **سعرُ الشراء** — ما يقبضه هو، لا ما يدفعه الزبون.
     *
     * **والمحرّكُ يحجب سعرَ البيع عنه** (`merchant_handlers.go`): «هو يضع
     * سعرَه ويقبض عليه، وما تبيع به المنصّةُ ليس شأنَه».
     */
    val price: Long = 0,
    val available: Boolean = true,
    val approved: Boolean = true,
    @SerialName("review_note") val reviewNote: String = "",
    @SerialName("image_url") val imageUrl: String? = null,
    @SerialName("image_thumb_url") val imageThumbUrl: String? = null,
    /**
     * **قسمُ السوق الذي يظهر فيه للزبون** — لا قسمُ مطبخه.
     *
     * **وهو ما تُرتَّب به الشاشة** (طلبُ المالك ٢٠٢٦-٠٨-٢٣): **الزبونُ
     * يتصفّح بأقسام السوق**، فمن رتّب أصنافَه بترتيبِ مطبخه لم يعرف
     * أين تقع بضاعتُه في السوق.
     */
    @SerialName("platform_section_id") val platformSectionId: String? = null,
    @SerialName("platform_section_name") val platformSectionName: String = "",
    /** **خياراتُه** — تُقرأ لتُعرض في المحرّر وتُعاد كما هي إن لم تُمسّ. */
    val modifiers: List<ModifierGroup> = emptyList(),
)

@Serializable
data class SectionsPage(val sections: List<PlatformSectionRef> = emptyList())

@Serializable
data class PlatformSectionRef(
    val id: String = "",
    val name: String = "",
    @SerialName("image_url") val imageUrl: String? = null,
    @SerialName("image_thumb_url") val imageThumbUrl: String? = null,
)

@Serializable
data class DayHours(
    /** ٠ الأحد … ٦ السبت — كما يرقّمها المحرّك (`catalog/hours.go`). */
    @SerialName("day_of_week") val dayOfWeek: Int = 0,
    val closed: Boolean = false,
    @SerialName("open_time") val openTime: String = "",
    @SerialName("close_time") val closeTime: String = "",
)

@Serializable
data class StoreReport(val summary: ReportSummary = ReportSummary())

/**
 * **مجملُ المدّة** — والحقولُ كما يكتبها المحرّك حرفاً بحرف.
 *
 * **ومبيعاتُه بسعره هو لا بما دفعه الزبون** (قرارُ المالك ٢٠٢٦-٠٨-١٠):
 * **متجرٌ يقرأ مبيعاتٍ فيها هامشُ المنصّة يحسب أرباحاً ليست له.**
 */
@Serializable
data class ReportSummary(
    val orders: Int = 0,
    val delivered: Int = 0,
    val cancelled: Int = 0,
    val sales: Long = 0,
    @SerialName("platform_commission") val commission: Long = 0,
) {
    /** **مستحقُّه** — مبيعاتُه ناقصَ عمولة المنصّة. */
    val due: Long get() = sales - commission
}

@Serializable
data class ReportReason(val code: String = "", val label: String = "")

@Serializable
data class Ack(val ok: Boolean = true)

@Serializable
data class Updated(val updated: Boolean = false)

/** **ما يعني الشاشةَ من الوسيط المرفوع** — معرّفُه وعنوانُه. */
@Serializable
data class MediaRef(
    val id: String = "",
    @SerialName("thumb_url") val thumbUrl: String = "",
)

/**
 * **مجموعةُ خياراتٍ ترافق الصنفَ في الحفظ** — «الحجم» و«يُضاف».
 *
 * **وتُرسَل الشجرةُ كاملةً في كلّ حفظ** — **والحذفُ لا بابَ له غير
 * الاستبدال**: من أزال مجموعةً ولم تُرسَل الشجرةُ بقيت في القاعدة.
 */
@Serializable
data class ModifierGroup(
    val id: String = "",
    val name: String = "",
    @SerialName("min_select") val minSelect: Int = 0,
    @SerialName("max_select") val maxSelect: Int = 1,
    val options: List<ModifierOption> = emptyList(),
)

@Serializable
data class ModifierOption(
    val id: String = "",
    val name: String = "",
    @SerialName("price_delta") val priceDelta: Long = 0,
    val available: Boolean = true,
)

// ── ما يُرسَل ──────────────────────────────────────────────────────────

@Serializable
data class TransitionInput(val to: String, val note: String = "")

@Serializable
data class ReportInput(val reason: String, val note: String = "")

@Serializable
data class AvailabilityInput(val available: Boolean)

@Serializable
data class SectionInput(val name: String)

@Serializable
data class HoursInput(val days: List<DayHours>)

@Serializable
data class ClosedInput(val closed: Boolean)

@Serializable
data class ReasonsPage(val reasons: List<String> = emptyList())

@Serializable
data class ReportsPage(val reports: List<MerchantReport> = emptyList())

/**
 * **بلاغٌ أرسله — وما ردّت به المنصّة.**
 *
 * **ولا اسمَ سائقٍ فيه ولا رقم** — البلاغُ عند المنصة وهي تفصل، **واسمُ
 * من شُكي عليه في يد الشاكي بابُ ثأرٍ لا بابُ عدل.**
 */
@Serializable
data class MerchantReport(
    val id: String = "",
    val number: Long = 0,
    @SerialName("order_number") val orderNumber: Long? = null,
    val reason: String = "",
    val status: String = "",
    /** كلمةُ المنصّة عند الإغلاق — **وبلاغٌ يُغلق بلا كلمةٍ يُقرأ تجاهلاً.** */
    val resolution: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class ClosedAck(@SerialName("emergency_closed") val closed: Boolean = false)

/**
 * **ما يقبله `/settings` فعلاً** — قِيس من `merchant_ops_handlers.go`.
 *
 * **ولا `accepts_orders` فيه**: الإغلاقُ بابُه `/emergency` وحدَه،
 * **وحقلٌ يُرسَل ولا يُقرأ يُطرح صامتاً** فيظنّ المرسِلُ أنّه فعل.
 */
@Serializable
data class StoreSettingsInput(
    /** **واسمُه بيده** — (طلبُ المالك ٢٠٢٦-٠٨-٢٣). */
    val name: String? = null,
    @SerialName("default_prep_minutes") val prepMinutes: Int? = null,
    @SerialName("min_order") val minOrder: Long? = null,
    /**
     * **عنوانُه ودبّوسُه** — (طلبُ المالك ٢٠٢٦-٠٨-٢٣).
     *
     * **والنقطةُ تُرسَل كاملةً أو لا تُرسَل**: المحرّكُ يردّ نصفَها خطأً
     * — **ومن أرسل `lat` وحدَه نقل متجرَه إلى خطّ الاستواء.**
     *
     * **وما لم يُرسَل لا يُمحى** — `COALESCE` في المحرّك، **فحفظُ
     * مدّةِ التحضير وحدَها لا يضيّع موضعَ المتجر.**
     */
    @SerialName("address_text") val addressText: String? = null,
    val lat: Double? = null,
    val lng: Double? = null,
)

@Serializable
data class MenuItemInput(
    @SerialName("section_id") val sectionId: String? = null,
    val name: String? = null,
    val description: String? = null,
    val price: Long? = null,
    val available: Boolean? = null,
    /** **وقسمُ السوق إلزاميّ** — صنفٌ بلا قسمٍ لا يراه زبون. */
    @SerialName("platform_section_id") val platformSectionId: String? = null,
    @SerialName("image_media_id") val imageMediaId: String? = null,
    val modifiers: List<ModifierGroup>? = null,
)
