package com.rahalgo.shared.rep

import com.rahalgo.shared.model.Envelope
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

    // ══════════════════════════════════════════════════════════════════
    // **أصنافُ عميله — يبنيها نيابةً عنه**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-١٨: «يقوم المندوبُ بإضافة أصناف المنتجات
    //  الموجودة لدى المتجر بدلاً عنه… نفس الفورم الموجود عند مدير
    //  المنصّة والموجود عند المتجر».)
    //
    // **ومتجرٌ ينضمّ ولا يفتح لوحتَه** — صاحبُه في متجره لا في حاسوب،
    // **وسوقٌ فيه متاجرُ بلا أصنافٍ سوقٌ فارغ.**
    //
    // **والحارسُ في المحرّك لا هنا**: كلُّ نداءٍ يُسأل عنه «أهذا المتجرُ
    // عميلُه؟» — **وإخفاءُ زرٍّ في شاشةٍ ليس حجبا.**

    /** **قائمةُ عميله** — أقسامٌ وأصنافٌ كما يقرؤها الأدمن. */
    suspend fun menu(merchantID: String): List<MenuSection> =
        api.call("/api/v1/rep/stores/$merchantID/menu")

    /**
     * **أقسامُ السوق** — **وبلاها يبني أصنافاً لا تظهر في التصفّح.**
     *
     * **وردُّها كائنٌ لا لائحة** (`{"sections": […]}`) — كما تقرؤه لوحةُ
     * الويب حرفا.
     */
    suspend fun platformSections(): List<PlatformSection> =
        api.call<PlatformSections>("/api/v1/rep/platform-sections").sections

    suspend fun createItem(merchantID: String, input: ItemInput): CreatedID =
        api.call("/api/v1/rep/stores/$merchantID/menu/items", HttpMethod.Post, input)

    suspend fun updateItem(itemID: String, input: ItemInput) {
        api.call<Map<String, Boolean>>(
            "/api/v1/rep/menu/items/$itemID", HttpMethod.Patch, input,
        )
    }

    suspend fun deleteItem(itemID: String) {
        api.call<Map<String, Boolean>>("/api/v1/rep/menu/items/$itemID", HttpMethod.Delete)
    }

    /**
     * **يرفع صورةَ صنفٍ ويردّ معرّفَها.**
     *
     * **والمعرّفُ هو المقصود** — يُرسَل بعدُ مع الصنف. **ومن رفع صورةً
     * ورمى ردَّها رفع ملفّاً لا يعرف اسمَه فلا يربطه بشيء.**
     */
    suspend fun uploadItemImage(fileName: String, bytes: ByteArray): String {
        val raw = api.upload(
            "/api/v1/rep/media", fileName, bytes, mapOf("kind" to "menu_item"),
        )
        return api.json.decodeFromString<Envelope<MediaRef>>(raw).data?.id.orEmpty()
    }
}

/** **ما يردّه الإنشاء** — معرّفُ ما أُنشئ لا أكثر. */
@Serializable
data class CreatedID(val id: String = "")

/** **ما يعني الشاشةَ من الوسيط المرفوع** — معرّفُه وعنوانُه. */
@Serializable
data class MediaRef(
    val id: String = "",
    @SerialName("thumb_url") val thumbUrl: String = "",
)

/**
 * **قسمٌ في القائمة — وهو قسمُ السوق لا قسمٌ يملكه المتجر.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «الأدمنُ هو من يزرع الأقسام، والمتجرُ يجد
 *  أقساماً جاهزة… لأنّ بكرا كلُّ متجرٍ رح ينزّل اسمَ قسمٍ مختلف».)
 *
 * **فلا يُنشأ ولا يُسمّى ولا يُحذف**: يظهر لأنّ فيه صنفاً، **ويختفي حين
 * يخرج آخرُ صنفٍ منه.** ومطعمٌ لا يبيع بقالةً لا يرى «بقالة».
 */
@Serializable
data class MenuSection(
    val id: String = "",
    val name: String = "",
    @SerialName("image_thumb_url") val imageThumbUrl: String? = null,
    val items: List<MenuItem> = emptyList(),
)

/**
 * **صنفٌ في القائمة.**
 *
 * **وسعران لا سعر**: `merchantPrice` ما يضعه المتجرُ، و`price` ما تبيع
 * به المنصّةُ بعد هامشها. **والمندوبُ يحتاج الرقمين** — يبني نيابةً
 * فيقول لصاحب المتجر «تقبض هذا» وللزبون «يُباع بهذا».
 */
@Serializable
data class MenuItem(
    val id: String = "",
    @SerialName("section_id") val sectionID: String = "",
    val name: String = "",
    val description: String = "",
    val price: Long = 0,
    @SerialName("merchant_price") val merchantPrice: Long = 0,
    @SerialName("platform_section_id") val platformSectionID: String? = null,
    @SerialName("platform_section_name") val platformSectionName: String = "",
    @SerialName("image_thumb_url") val imageThumbURL: String? = null,
    /** **رفعه المتجرُ بيده** — «نفد الصنف» يُطفئه ولا يحذفه. */
    val available: Boolean = true,
    /** **مصدرُه خارجَ دوامه الآن** — يقوله الوقتُ لا صاحبُه. */
    @SerialName("source_closed") val sourceClosed: Boolean = false,
    /** **أنُشر للزبائن؟** — حين يُرفع مفتاحُ مراجعة القائمة. */
    val approved: Boolean = true,
    @SerialName("review_note") val reviewNote: String = "",
    val modifiers: List<ModifierGroup> = emptyList(),
)

/**
 * **مجموعةُ مُعدِّلات** — الحجمُ أو الإضافاتُ أو ما شابه.
 *
 * **و«إلزاميّة» تُشتقّ من الحدّ الأدنى ولا تُكتب**: `minSelect > 0` تعني
 * إلزاميّة. **ورقمٌ واحدٌ لا حقلان يتناقضان** — راية «إلزاميّ» مرفوعةٌ
 * وأدنى اختيارٍ صفرٌ حالٌ لا معنى لها، ولا يعرف المحرّكُ أيَّهما يصدّق.
 */
@Serializable
data class ModifierGroup(
    val name: String = "",
    @SerialName("min_select") val minSelect: Int = 0,
    @SerialName("max_select") val maxSelect: Int = 1,
    val options: List<ModifierOption> = emptyList(),
)

/** **خيارٌ في مجموعة** — واسمُه وفرقُ سعره. */
@Serializable
data class ModifierOption(
    val name: String = "",
    @SerialName("price_delta") val priceDelta: Long = 0,
)

@Serializable
data class PlatformSection(
    val id: String = "",
    val name: String = "",
)

/** **غلافُ ردّ أقسام السوق** — المحرّكُ يضعها تحت مفتاح. */
@Serializable
data class PlatformSections(val sections: List<PlatformSection> = emptyList())

/**
 * **ما يُرسَل للصنف.**
 *
 * **وكلُّ حقلٍ غائبٍ يعني «لا تمسّه»** — فالتعديلُ الجزئيُّ لا يمحو ما
 * لم يُذكَر. **ومن أرسل الحقولَ كلَّها في كلّ تعديلٍ محا ما لا يعرفه.**
 *
 * **و`price` هو سعرُ المتجر** لا سعرُ البيع — اسمُه في العقد كذلك منذ
 * أوّل شاشة، **وتغييرُ اسمِ حقلٍ يكسر كلَّ شاشةٍ لم تُحدَّث.**
 *
 * **ولا `section_id` هنا** — **قسمُ الصنف هو قسمُ السوق**، ولوحةُ الويب
 * لا ترسله أصلاً.
 */
@Serializable
data class ItemInput(
    val name: String? = null,
    val description: String? = null,
    val price: Long? = null,
    @SerialName("platform_section_id") val platformSectionID: String? = null,
    val available: Boolean? = null,
    @SerialName("image_media_id") val imageMediaID: String? = null,
    /**
     * **وإن أُرسلت — ولو فارغةً — استُبدلت الشجرةُ كلُّها.**
     *
     * **فلا تُرسَل إلّا من نموذجٍ يعرفها**: `toggleAvailable` تقلب راية
     * التوفّر وحدَها، **ولو حملت `modifiers` فارغةً لَمحت مُعدِّلاتِ
     * الصنف كلَّها بضغطةٍ على «إيقاف مؤقت».**
     */
    val modifiers: List<ModifierGroup>? = null,
)

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
