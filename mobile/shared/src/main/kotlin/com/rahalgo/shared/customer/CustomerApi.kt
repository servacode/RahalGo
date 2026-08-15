package com.rahalgo.shared.customer

import com.rahalgo.shared.model.HomePage
import com.rahalgo.shared.model.ItemsPage
import com.rahalgo.shared.model.OffersPage
import com.rahalgo.shared.model.Quote
import com.rahalgo.shared.model.Referral
import com.rahalgo.shared.net.Ack
import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أبوابُ الزبون — كما تناديها شاشاتُ الويب حرفيّا**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (`web/apps/rahalgo/src/app/(site)/` — قِيست نداءاتُها بندا بند.)
 *
 * **ومسارٌ ثانٍ للتطبيق كان سيفترق**: تُشدَّد قاعدةٌ في الويب وتُنسى في
 * الهاتف، **فيطلب الزبونُ من طريقٍ بلا حدٍّ يحرسه.**
 *
 * **ولا منطقَ عملٍ هنا** (`GROUND-RULES.md` §7.2 البند ٥): السعرُ
 * والتوصيلُ والخصمُ يحسبها المحرّك — **وهذه تنادي وتُعيد ما قيل.**
 */
class CustomerApi(private val api: ApiClient) {

    // ══════════════════════════════════════════════════════════════════
    // **التصفّح — عامٌّ بلا حساب**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ومن فتح التطبيقَ يتصفّح قبل أن يدخل** — وبابٌ يطلب حساباً ليُرى
    // سعرٌ يُغلق قبل أن يُفتح.

    /** **الصفحةُ الأولى** — لافتاتٌ وتصنيفاتٌ وأقسامُ سوق. */
    suspend fun home(): HomePage = api.raw("/api/v1/public/home")

    /** **أصنافُ قسم** — من كلّ المصادر مختلطةً بلا اسم متجر. */
    suspend fun sectionItems(sectionId: String): ItemsPage =
        api.raw("/api/v1/public/sections/$sectionId/items")

    /**
     * **بحثٌ في الأصناف.**
     *
     * **وحرفان حدُّه الأدنى** (كما في الويب) — **وحرفٌ واحدٌ يجلب السوقَ
     * كلَّه** فيبطئ ولا يفيد.
     */
    suspend fun search(query: String): ItemsPage =
        api.raw("/api/v1/public/search/items?q=" + query.trim())

    /** **العروضُ الساريةُ وحدَها** — والسريانُ يقوله الخادم. */
    suspend fun offers(): OffersPage = api.raw("/api/v1/public/offers")

    // ══════════════════════════════════════════════════════════════════
    // **الطلب**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **تسعيرةٌ قبل الإرسال** — الأصنافُ والنقطةُ فتُردّ الأرقام.
     *
     * **ولا تُحسب في الجهاز**: رسمُ المنطقة يتبدّل من اللوحة، **وحسبةٌ
     * محلّيّةٌ تفترق عمّا يُقيَّد.**
     */
    suspend fun quote(items: List<CartLine>, lat: Double, lng: Double): Quote =
        api.call(
            "/api/v1/public/quote",
            HttpMethod.Post,
            QuoteInput(items, lat, lng),
        )

    /** **يُرسل الطلب** — ويردّ الطلبَ كما قُيّد. */
    suspend fun createOrder(input: NewOrder): OrderRef =
        api.call("/api/v1/orders", HttpMethod.Post, input)

    /**
     * **طلبٌ خاصّ** — ما ليس في المنصّة.
     *
     * **ولا يُسأل عن سعرٍ ولا متجر**: هو يطلب ما لا نعرف سعرَه،
     * **والسائقُ يشتريه ويتّفق معه بعد الإسناد.** **وسؤالٌ لا جوابَ له
     * يُوقف من يملأ نموذجا.**
     */
    suspend fun createCustom(input: NewCustom): OrderRef =
        api.call("/api/v1/orders/custom", HttpMethod.Post, input)

    /**
     * **يُلغي طلباً في مهلته** — والمهلةُ من الخادم لا من حسبةِ الشاشة.
     *
     * **ولا سببَ معه** — (قرارُ المالك ٢٠٢٦-٠٨-١٥): الزبونُ حرٌّ في
     * طلبه، **وحقلٌ إلزاميٌّ ليُلغيَه ضريبةٌ على حقّه.**
     *
     * **والمحرّكُ يقرأ `note`** — **وكنتُ أرسل `reason`**: فما كتبه
     * كان يُطرح في الطريق ويُقيَّد الإلغاءُ بلا سببٍ على كلّ حال.
     */
    suspend fun cancel(orderId: String) {
        api.call<Ack>(
            "/api/v1/orders/$orderId/cancel",
            HttpMethod.Post,
            mapOf("note" to ""),
        )
    }

    /**
     * **يُقيّم طلباً سُلّم — تقييمان لا واحد.**
     *
     * **نجومٌ للمنصّة ونجومٌ للسائق** (`orders/ratings.go`): الزبونُ لا
     * يعرف المتجرَ ولا يختاره، **فنجمةٌ تُنسب إليه بلا معنى** — لكنّه
     * يعرف من أوصل إليه.
     *
     * **و`driverStars` فارغةٌ حيث لا سائق** — والمحرّكُ يُلغيها أصلاً
     * إن لم يكن للطلب سائق.
     */
    suspend fun rate(orderId: String, platformStars: Int, driverStars: Int?, note: String) {
        api.call<Ack>(
            "/api/v1/orders/$orderId/rating",
            HttpMethod.Post,
            RateInput(platformStars, driverStars, note),
        )
    }

    /**
     * **ما قيّمه من قبل** — ومنه يُعرف على أيّ طلبٍ يُعرض زرُّ النجوم.
     *
     * **ولا `rated` في ردّ الطلبات** — فتُقرأ من هنا، **كما تقرؤها
     * الويب حرفا** (`orders/page.tsx`). **ونجومٌ مرّتين تُقرأ أنّ
     * الأولى لم تصل.**
     */
    suspend fun ratings(): RatingsPage = api.call("/api/v1/my/ratings")

    // ══════════════════════════════════════════════════════════════════
    // **ما يخصّه**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **طلباتُه** — والجاري وحدَه إن طُلب.
     *
     * **ومهلةُ الإلغاء تصل مع كلّ طلب** (`cancel_seconds_left`): **زرٌّ
     * بلا مهلةٍ إمّا يظهر دائماً فيعتذر، أو لا يظهر أبداً فيُحبس صاحبُه
     * في طلبٍ لم يبدأ.**
     */
    suspend fun orders(openOnly: Boolean = false, page: Int = 1): MyOrdersPage =
        api.call("/api/v1/my/orders?open_only=$openOnly&page=$page&per_page=30")

    suspend fun order(id: String): MyOrder = api.call("/api/v1/my/orders/$id")

    /** **مفضّلتُه** — أصنافٌ لا متاجر. */
    suspend fun favorites(): ItemsPage = api.call("/api/v1/my/favorites")

    /**
     * **يُضيف أو يحذف** — بابٌ واحدٌ يقلب الحال ويردّ ما صار.
     *
     * **والردُّ `favorite` لا `added`** (`discover_handlers.go`) —
     * **وكنتُ أقرأ الاسمَ الخطأ فيصل `false` أبدا**: القلبُ يُضغط،
     * والقيدُ يُكتب في القاعدة، **ولا يتغيّر شيءٌ في الشاشة.**
     */
    suspend fun toggleFavorite(itemId: String): Boolean =
        api.call<FavoriteResult>("/api/v1/my/favorites/$itemId", HttpMethod.Post).favorite

    /** **رمزُ دعوته ورابطُه وما ناله.** */
    suspend fun referral(): Referral = api.call("/api/v1/auth/referral")

    /**
     * **أسبابُ الشكوى لهذا الطلب** — مصنّفةٌ لا نصٌّ حرّ.
     *
     * **وتختلف بحال الطلب**: «لم أستلم طلبي» على طلبٍ لم يُسلَّم لغوٌ
     * — حالتُه تقول ذلك، **ومن سُئل عمّا نعرفه شكّ فيما نعرف.**
     */
    suspend fun complaintReasons(orderId: String): ReasonsPage =
        api.call("/api/v1/my/orders/$orderId/complaint-reasons")

    /** **يفتح شكوى** — ويردّ رقمَها ليُتابَع. */
    suspend fun complain(orderId: String, reason: String, note: String) {
        api.call<Ack>(
            "/api/v1/my/orders/$orderId/complaint",
            HttpMethod.Post,
            mapOf("reason" to reason, "note" to note),
        )
    }

    /** **شكاواه وأين وصلت** — ومن اشتكى ولم يرَ جواباً ظنّ شكواه ضاعت. */
    suspend fun tickets(): TicketsPage = api.call("/api/v1/my/tickets")
}

/** **سطرٌ في السلّة** — صنفٌ وعددُه. */
@Serializable
data class CartLine(
    @SerialName("menu_item_id") val menuItemId: String,
    val qty: Int,
    val note: String = "",
)

@Serializable
private data class QuoteInput(
    val items: List<CartLine>,
    val lat: Double,
    val lng: Double,
)

/**
 * **طلبٌ جديد.**
 *
 * **ولا `merchant_id` من الجهاز**: المحرّكُ يستنتجه من الأصناف —
 * **والمتاجرُ مخفيّةٌ عن الزبون أصلا.**
 */
@Serializable
data class NewOrder(
    val items: List<CartLine>,
    @SerialName("address_text") val addressText: String,
    val lat: Double,
    val lng: Double,
    /** `cash` أو `wallet` — **ولا خليط.** */
    @SerialName("payment_method") val paymentMethod: String = "cash",
    @SerialName("promo_code") val promoCode: String = "",
    val notes: String = "",
)

@Serializable
data class NewCustom(
    /** **ما يريده بلفظه** — لا اختيارٌ من قائمة. */
    val request: String,
    @SerialName("address_text") val addressText: String,
    val lat: Double,
    val lng: Double,
    val notes: String = "",
)

@Serializable
data class OrderRef(
    val id: String = "",
    val code: String = "",
    val status: String = "",
)

@Serializable
private data class FavoriteResult(val favorite: Boolean = false)

@Serializable
private data class RateInput(
    @SerialName("platform_stars") val platformStars: Int,
    @SerialName("driver_stars") val driverStars: Int?,
    val comment: String,
)

@Serializable
data class RatingsPage(val ratings: List<RatedOrder> = emptyList())

@Serializable
data class RatedOrder(
    @SerialName("order_id") val orderId: String = "",
    val rated: Boolean = false,
)

@Serializable
data class ReasonsPage(val reasons: List<ComplaintReason> = emptyList())

/**
 * **سببُ شكوى — رمزٌ لا نصّ.**
 *
 * **والمحرّكُ يرسل أشياءَ لا نصوصا** (`support/complaints.go`)، **وكنتُ
 * أقرأ نصوصا**: فيسقط التحويلُ وتصل القائمةُ فارغة — **فتبدو الشكوى
 * نصّاً حرّاً وهي أسبابٌ مصنّفةٌ لم تصل.**
 *
 * **والاسمُ العربيُّ في الجهاز** — المحرّكُ لا يعرف لغةَ من يقرأ.
 */
@Serializable
data class ComplaintReason(
    val code: String = "",
    /** **مَن يعنيه** — سائقٌ أو متجرٌ أو فراغٌ للمنصّة. **ويُشتقّ ولا يُسأل عنه.** */
    val against: String = "",
)

@Serializable
data class TicketsPage(val tickets: List<Ticket> = emptyList())

@Serializable
data class Ticket(
    val id: String = "",
    val subject: String = "",
    val reason: String = "",
    val status: String = "",
    @SerialName("order_code") val orderCode: String = "",
    @SerialName("created_at") val createdAt: String = "",
    val resolution: String = "",
)

@Serializable
data class MyOrdersPage(
    val orders: List<MyOrder> = emptyList(),
    val total: Int = 0,
    val page: Int = 1,
)

/**
 * **طلبُ الزبون كما يراه هو.**
 *
 * **ولا اسمَ متجرٍ فيه** — يُنقّى في المحرّك (`redactAllForCustomer`)،
 * **وحقلٌ يصل الجهازَ يُقرأ.**
 */
@Serializable
data class MyOrder(
    val id: String = "",
    /**
     * **رقمُ الطلب كما يعرفه الجميع** — لا ستّةُ أحرفٍ من معرّفه.
     *
     * **والمحرّكُ يرسل `number` لا `code`** — **ورقمٌ لا يعرفه صاحبُه
     * ولا المكتبُ لا يُستعمل في شكوى.**
     */
    val number: Long = 0,
    val kind: String = "",
    val status: String = "",
    val subtotal: Long = 0,
    @SerialName("delivery_fee") val deliveryFee: Long = 0,
    val discount: Long = 0,
    val total: Long = 0,
    @SerialName("payment_method") val paymentMethod: String = "",
    @SerialName("address_text") val addressText: String = "",
    val notes: String = "",
    @SerialName("custom_request") val customRequest: String = "",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("delivered_at") val deliveredAt: String? = null,
    @SerialName("cancel_reason") val cancelReason: String = "",
    /** **ما بقي من مهلة الإلغاء** — و`-1` تعني «بلا مهلة». */
    @SerialName("cancel_seconds_left") val cancelSecondsLeft: Int = -1,
    /**
     * ══════════════════════════════════════════════════════════════════
     * **وما كان `*` في Go يكون `?` هنا — بلا استثناء**
     * ══════════════════════════════════════════════════════════════════
     *
     * **والافتراضُ لا يُنقذ**: `kotlinx` يستعمله حين **يغيب** الحقل،
     * **لا حين يصل `null`.**
     *
     * **وقِيس على الجوّال (٢٠٢٦-٠٨-١٥)**: اسمُ السائق يصل `null` قبل
     * الإسناد **فتسقط الشاشةُ كلُّها** — ولا يُقرأ طلبٌ واحد.
     */
    @SerialName("driver_name") val driverName: String? = null,
    @SerialName("driver_phone") val driverPhone: String? = null,
    @SerialName("promo_code") val promoCode: String? = null,
    @SerialName("merchant_logo_thumb_url") val merchantLogoThumbUrl: String? = null,
    /** **ما في الطلب باختصار** — يرسله المحرّكُ نصّاً لا قائمة. */
    @SerialName("items_count") val itemsCount: Int = 0,
    @SerialName("items_preview") val itemsPreview: String = "",
    @SerialName("wallet_paid") val walletPaid: Long = 0,
    @SerialName("cash_due") val cashDue: Long = 0,
    /**
     * ══════════════════════════════════════════════════════════════════
     * **مسارُه ومرحلتُه — يحسبهما المحرّكُ لا الشاشة**
     * ══════════════════════════════════════════════════════════════════
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-١٢: «الأفضل يكون بشكلٍ مركزيّ، مو كلّ
     *  صفحةٍ تاخذ من مكانٍ مختلف».)
     *
     * **وأربعَ عشرةَ حالاً تُطوى في ستّ مراحل** — والطيُّ هو ما
     * يفترق بين الشاشات إن كُتب في كلٍّ منها. **وقد افترق فعلا**:
     * كتب التطبيقُ خمساً وأسقط **«وصل إليك»** (شهده المالك
     * ٢٠٢٦-٠٨-١٥).
     *
     * **والمفاتيحُ تصل بالإنجليزيّة** — المحرّكُ لا يعرف لغةَ من
     * يقرأ، **والترجمةُ وحدَها في الجهاز.**
     *
     * **و`stageAt = -1` تعني لا مسارَ له** — انتهى قبل أن يصل.
     */
    val stages: List<String> = emptyList(),
    @SerialName("stage_at") val stageAt: Int = -1,
)

