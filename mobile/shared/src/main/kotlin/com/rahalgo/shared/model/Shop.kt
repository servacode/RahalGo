package com.rahalgo.shared.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما يراه الزبونُ في السوق — كما يرسله المحرّك**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ولا متاجرَ فيه** (`customer_handlers.go`: «المتاجرُ مخفيّةٌ عن الزبون
 * بالكامل») — **المنصّةُ سوقٌ يجلب منها**، وهو يشتري «من رحّال» لا «من
 * مطعم فلان». (قرارُ المالك ٢٠٢٦-٠٨-٠٥.)
 *
 * **وحقلٌ يصل الجهازَ يُقرأ** — حجبٌ في الشاشة ولا يفرضه المحرّك ليس
 * حجبا.
 */
@Serializable
data class HomePage(
    val banners: List<Banner> = emptyList(),
    val categories: List<Category> = emptyList(),
    val sections: List<Section> = emptyList(),
    @SerialName("support_phone") val supportPhone: String = "",
    /** **مهلةُ اللافتة والجولة** — من الإعدادات لا من الشاشة. */
    @SerialName("banner_auto") val bannerAuto: Boolean = false,
    @SerialName("banner_every_ms") val bannerEveryMs: Int = 0,
    @SerialName("rail_auto") val railAuto: Boolean = false,
    @SerialName("rail_every_ms") val railEveryMs: Int = 0,
)

/**
 * **لافتةٌ كما يرسلها المحرّك** — `catalog.Banner`.
 *
 * **و`target` لا `link_url`**: كان الاسمُ مخترَعاً **فيُقرأ فارغاً
 * دائماً** — وحقلٌ لا وجودَ له في الردّ يسقط إلى قيمته الافتراضيّة
 * بلا خطأ (`ignoreUnknownKeys`)، **فتُقرأ اللافتةُ بلا وجهةٍ أبدا.**
 */
@Serializable
data class Banner(
    val id: String = "",
    val title: String = "",
    @SerialName("image_url") val imageUrl: String? = null,
    @SerialName("image_thumb_url") val imageThumbUrl: String? = null,
    /** **وجهةُ الضغطة** — وفارغُها لافتةٌ تُرى ولا تُفتح. */
    val target: String = "",
    /**
     * **أوُلِّدت لها نسخٌ أصغر؟**
     *
     * **والصفوفُ القديمةُ بلا نسخ** — **ومن طلب مقاساً لم يُولَّد يأخذ
     * ٤٠٤ في وسط الشاشة.** فيُسأل هذا الحقلُ قبل أن يُشتقّ مسار.
     */
    val sizes: Boolean = false,
)

@Serializable
data class Category(
    val id: String = "",
    val name: String = "",
    val icon: String = "",
)

/**
 * **قسمٌ من السوق** — بصورته لا برمزه.
 *
 * **والسوقُ يُتصفَّح بالصور**: الزبونُ يعرف الشاورما من صورتها قبل أن
 * يقرأ اسمَها، **ورمزٌ رماديٌّ لعشرة أقسامٍ يجعلها كلَّها شيئاً واحدا.**
 */
@Serializable
data class Section(
    val id: String = "",
    val name: String = "",
    val icon: String = "",
    @SerialName("image_url") val imageUrl: String? = null,
    @SerialName("image_thumb_url") val imageThumbUrl: String? = null,
    val count: Int = 0,
)

/**
 * **صنفٌ معروض** — الشكلُ نفسُه في القسم والبحث والمفضّلة والعروض.
 *
 * **وشكلٌ ثانٍ يشبهه يفترق يوما** — كما افترقت المفضّلةُ في الويب.
 */
@Serializable
data class Item(
    val id: String = "",
    val name: String = "",
    val description: String = "",
    val price: Long = 0,
    @SerialName("image_url") val imageUrl: String? = null,
    @SerialName("image_thumb_url") val imageThumbUrl: String? = null,
    val available: Boolean = true,
    /** **مصدرُه مغلقٌ الآن** — يُعرض ولا يُطلب. */
    @SerialName("source_closed") val sourceClosed: Boolean = false,
    @SerialName("source_opens_at") val sourceOpensAt: String? = null,
    @SerialName("section_id") val sectionId: String = "",
    @SerialName("section_name") val sectionName: String = "",
    /**
     * **سعرُ ما قبل الخصم — وفارغٌ حين لا خصم.**
     *
     * **والمشطوبُ هو ما يجعل الخصمَ خصما**: رقمٌ وحدَه رقم، **ورقمان
     * أحدُهما مشطوبٌ توفيرٌ يُرى.**
     */
    @SerialName("price_before") val priceBefore: Long? = null,
    @SerialName("discount_percent") val discountPercent: Int? = null,
    /**
     * **له خياراتٌ تُختار قبل الطلب** — حجمٌ أو إضافات.
     *
     * **والشاشةُ تحتاج أن تعرف قبل الضغطة**: صنفٌ بخياراتٍ يفتح نافذةَ
     * اختيار، وصنفٌ بلا خياراتٍ يدخل السلّةَ مباشرةً. **ولو لم يُعرف
     * إلّا بنداءٍ لكلّ بطاقةٍ لَكانت عشرون بطاقةً عشرين نداء.**
     *
     * **والمجموعةُ الإلزاميّةُ تُسقط الطلبَ كلَّه** إن لم تُرسَل
     * اختياراتُها — لا الصنفَ وحدَه.
     */
    @SerialName("has_options") val hasOptions: Boolean = false,
)

/**
 * **مجموعةُ خياراتٍ لصنف** — «الحجم» أو «يُضاف».
 *
 * **و`minSelect` هو الفرقُ بين إلزاميٍّ واختياريّ**: واحدٌ يعني «لا
 * يُطلب حتّى يُختار»، **وصفرٌ يعني «إن شئت».**
 */
@Serializable
data class ModifierGroup(
    val id: String = "",
    val name: String = "",
    @SerialName("min_select") val minSelect: Int = 0,
    @SerialName("max_select") val maxSelect: Int = 1,
    val options: List<ModifierOption> = emptyList(),
)

/** **خيارٌ واحدٌ وفرقُ سعره** — والفرقُ يُضاف إلى ثمن الصنف. */
@Serializable
data class ModifierOption(
    val id: String = "",
    val name: String = "",
    @SerialName("price_delta") val priceDelta: Long = 0,
    val available: Boolean = true,
)

/** **صنفٌ بتفصيله** — هو وخياراتُه معاً في نداءٍ واحد. */
@Serializable
data class ItemDetail(
    val item: Item = Item(),
    val modifiers: List<ModifierGroup> = emptyList(),
)

@Serializable
data class ItemsPage(val items: List<Item> = emptyList(), val total: Int = 0)

/**
 * **تسعيرةٌ قبل الإرسال** — من الخادم لا من الجهاز.
 *
 * **وحسبةٌ في الجهاز تفترق عمّا يُقيَّد في الطلب** — فيرى سعراً ويُحاسَب
 * بآخر، **وهي أسرعُ طريقةٍ لكسر الثقة.**
 */
@Serializable
data class Quote(
    val subtotal: Long = 0,
    @SerialName("delivery_fee") val deliveryFee: Long = 0,
    val total: Long = 0,
    /** **خارجَ نطاق التوصيل** — لا سعرَ ولا طلب. */
    @SerialName("out_of_zone") val outOfZone: Boolean = false,
)

/** **معاينةُ كود الخصم** — قبل أن يُرسَل الطلب. */
@Serializable
data class PromoPreview(
    val valid: Boolean = false,
    val discount: Long = 0,
    val total: Long = 0,
    val reason: String = "",
)

/**
 * **عرضٌ من المنصّة** — يُرى بلا أن يُبحث عنه.
 *
 * **والفرقُ عن كود الخصم**: الكودُ يُكتب والعرضُ يُرى — **ومن لم يسمع
 * بالكود لا يستفيد منه ولا يعلم أنّه فاته.**
 *
 * **والسعران يُحسبان في الخادم** — حسبةٌ في الجهاز تفترق عمّا يُقيَّد في
 * الطلب، **فيرى سعراً ويُحاسَب بآخر.**
 */
@Serializable
data class Offer(
    val id: String = "",
    val kind: String = "",
    val title: String = "",
    val body: String = "",
    @SerialName("image_url") val imageUrl: String? = null,
    val href: String = "",
    @SerialName("menu_item_id") val menuItemId: String? = null,
    @SerialName("item_name") val itemName: String = "",
    @SerialName("item_image_url") val itemImageUrl: String? = null,
    @SerialName("price_before") val priceBefore: Long = 0,
    @SerialName("price_after") val priceAfter: Long = 0,
    @SerialName("discount_percent") val discountPercent: Int? = null,
    @SerialName("ends_at") val endsAt: String? = null,
    /**
     * **سارٍ الآن — يقوله الخادمُ ولا يُستنتج في الشاشة.**
     *
     * **وشرطٌ يُحسب في موضعين يفترق يوما** فتُعرض عروضٌ انتهت.
     */
    val live: Boolean = false,
    /** **للصنف خياراتٌ تُختار** — وشاشةُ العروض تفتح النافذةَ بدل الإضافة. */
    @SerialName("has_options") val hasOptions: Boolean = false,
)

@Serializable
data class OffersPage(val offers: List<Offer> = emptyList())

/**
 * **مدنُ المنصّة** — (`GET /api/v1/public/cities`).
 *
 * **ومركزُها ونصفُ قطرِها يخرجان معها**: من اختار مدينةً بيده يتصفّح
 * من مركزها، **ومن أراد أن يعرف أهو داخلَها يقيس بنفسه بلا نداءٍ ثانٍ.**
 */
@Serializable
data class CitiesPage(val cities: List<City> = emptyList())

@Serializable
data class City(
    val id: String = "",
    val name: String = "",
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    @SerialName("radius_m") val radiusM: Int = 0,
)

/**
 * **حالُ دعواته** — ورمزُه ورابطُه.
 *
 * **والرابطُ يُبنى في الخادم لا في الجهاز**: عنوانُ الموقع يتغيّر من
 * نشرٍ إلى نشر، **ورابطٌ يُركَّب في الجهاز يحمل عنوانَ من ركّبه.**
 */
@Serializable
data class Referral(
    val code: String = "",
    val link: String = "",
    val invited: Int = 0,
    /** **من دُعوا وطلبوا فعلا** — لا من سجّلوا وحدَهم. */
    val rewarded: Int = 0,
    val earned: Long = 0,
    /**
     * **ما سيناله عن الدعوة القادمة** — والرقمُ يُقال قبل الفعل.
     *
     * **ووعدٌ مبهمٌ لا يُحرّك أحدا**: «ادعُ أصدقاءك» لا تعني شيئاً،
     * **و«ادعُ صديقاً واربح ٥٬٠٠٠» تعني.**
     */
    @SerialName("next_reward") val nextReward: Long = 0,
    /** **مكافآتُ الأولى والثانية والثالثة** — والصورةُ كاملةٌ تُقنع. */
    val tiers: List<Long> = emptyList(),
    val rest: Long = 0,
    /**
     * **متى تُصرف** — `signup` أو `first_order`.
     *
     * **والشرطُ يُقال لا يُفترض**: كُتب في الويب «تُصرف عند أوّل طلب»
     * **والإعدادُ يقول عند التسجيل** — **ووعدٌ يخالف ما يقع أسوأُ من
     * ألّا يُوعَد.**
     */
    @SerialName("reward_on") val rewardOn: String = "",
)
