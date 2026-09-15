package com.rahalgo.shared.offers

import com.rahalgo.shared.net.ApiClient
import io.ktor.http.HttpMethod
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عروضُ المتجر — عقدٌ واحدٌ يقرؤه بابان** (`OF`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولمَ هنا لا في تطبيقين
 *
 * **وصاحبُ المتجر ومندوبُه يقرآن الشيءَ نفسَه** — **ونسختان تفترقان
 * يوماً**: **فيقول أحدُهما «سارٍ» ويقول الآخرُ «مجدول»** عن صفٍّ واحد.
 *
 * **والمساران يفترقان والعقدُ واحد** — **وهو الفرقُ كلُّه**: نطاقٌ لا
 * منطق.
 *
 * # ولا حسبةَ سعرٍ هنا
 *
 * **والسعران يجيئان محسوبين** (`price_before` و`price_after`) —
 * **وحسبةٌ في الجهاز تفترق عمّا يُقيَّد في الطلب**: **فيرى سعراً
 * ويُحاسَب بآخر.** (وهي قاعدةُ `AfterDiscount` نفسُها.)
 *
 * # والحالُ يقولها الخادم
 *
 * **ولا تُشتقّ من `ends_at` بساعة الجهاز** — **ومن قدّم ساعتَه رأى
 * عرضاً منتهياً ساريا.**
 */
@Serializable
data class StoreOffer(
    val id: String = "",
    val title: String = "",
    @SerialName("menu_item_id") val menuItemId: String? = null,
    @SerialName("item_name") val itemName: String = "",
    @SerialName("discount_percent") val discountPercent: Int? = null,
    @SerialName("price_before") val priceBefore: Long = 0,
    @SerialName("price_after") val priceAfter: Long = 0,
    @SerialName("starts_at") val startsAt: String? = null,
    @SerialName("ends_at") val endsAt: String? = null,
    /**
     * **الحالُ المشتقّة** — `scheduled` · `active` · `stopped` · `expired`.
     *
     * **ورمزٌ آليٌّ لا يُعرض على إنسان** — **تترجمه الشاشةُ** (`OfferText`).
     */
    val status: String = "",
)

@Serializable
data class StoreOffersPage(val offers: List<StoreOffer> = emptyList())

/**
 * **ما يُرسَل لإنشاء عرض.**
 *
 * **ولا حقلَ لمن يتحمّل الخصم** — **ولو أُرسل لم يُقرأ**: **المتجرُ
 * يخصم من مستحقّه هو، والقرارُ بيد الإدارة** (`CreateScoped`).
 *
 * **والزمنُ يُرسَل بصيغة `ISO-8601` بالتوقيت العالميّ** — **والخادمُ
 * يحكم، والجهازُ يعرض بتوقيت صاحبه.**
 */
@Serializable
data class NewOffer(
    val title: String,
    @SerialName("menu_item_id") val menuItemId: String,
    @SerialName("discount_percent") val discountPercent: Int,
    @SerialName("starts_at") val startsAt: String? = null,
    @SerialName("ends_at") val endsAt: String? = null,
)

/**
 * **بابُ العروض** — **ويُبنى بجذرِ مساره.**
 *
 * **فالمتجرُ `/merchant` والمندوبُ `/rep`** — **والمسارُ بعدهما واحدٌ
 * حرفاً بحرف**، **فلا صفّان يفترقان.**
 */
class StoreOffersApi(private val api: ApiClient, private val root: String) {

    suspend fun list(storeId: String): StoreOffersPage =
        api.call("/api/v1/$root/stores/$storeId/offers")

    suspend fun create(storeId: String, body: NewOffer): StoreOffer =
        api.call("/api/v1/$root/stores/$storeId/offers", HttpMethod.Post, body)

    /**
     * **يُنزل العرضَ** — **ولا يحذفه.**
     *
     * **وإعادتُه بلا أثرٍ ثانٍ** — **ومن ضغط مرّتين لأنّ الشبكةَ
     * تأخّرت لا يُعاقَب.**
     */
    suspend fun stop(storeId: String, offerId: String): StoreOffer =
        api.call(
            "/api/v1/$root/stores/$storeId/offers/$offerId/stop",
            HttpMethod.Post,
            emptyMap<String, String>(),
        )

    companion object {
        fun merchant(api: ApiClient) = StoreOffersApi(api, "merchant")
        fun rep(api: ApiClient) = StoreOffersApi(api, "rep")
    }
}
