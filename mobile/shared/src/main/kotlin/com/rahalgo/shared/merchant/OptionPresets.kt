package com.rahalgo.shared.merchant

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خياراتُ الصنف الجاهزة · نسخُ صنف · الأسعارُ دفعةً واحدة** (٢٠٢٦-١٠-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * لكلّ تصنيفٍ خياراتُه («النمرة» للأحذية، «المقاس» للألبسة)، **والمتجرُ
 * يختار لنفسه فيعلو اختيارُه.** والنماذجُ هنا مشتركةٌ بين المتجر والمندوب.
 */

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** خيارٌ داخلَ القالب — بدايةٌ يغيّرها صاحبُ المتجر. */
@Serializable
data class PresetOption(
    val name: String = "",
    @SerialName("price_delta") val priceDelta: Long = 0,
    /** وزنُ الخيار بالغرام لقالبٍ «حسب الوزن» (٢٠٢٦-١٠-٢٨). */
    val grams: Int? = null,
)

/** **قالبُ مجموعة خيارات** — و`chosen` يعني أنّه يطلع عند هذا المتجر. */
@Serializable
data class OptionPreset(
    val id: String = "",
    val name: String = "",
    @SerialName("min_select") val minSelect: Int = 0,
    @SerialName("max_select") val maxSelect: Int = 1,
    val options: List<PresetOption> = emptyList(),
    val chosen: Boolean = false,
    /** **طريقةُ السعر** «priced» «none» «weight» — تحدّدها اللوحة (٢٠٢٦-١٠-٢٨). */
    @SerialName("price_mode") val priceMode: String = PRICE_MODE_PRICED,
) {
    /** **مجموعةٌ تُنسخ إلى الصنف** حين يُعلَّم القالب — بطريقة سعرها. */
    fun toGroup(): ModifierGroup = ModifierGroup(
        name = name,
        minSelect = minSelect,
        maxSelect = maxSelect,
        priceMode = priceMode,
        baseGrams = if (priceMode == PRICE_MODE_WEIGHT) DEFAULT_BASE_GRAMS else null,
        options = options.map {
            when (priceMode) {
                PRICE_MODE_WEIGHT -> ModifierOption(name = it.grams?.let(::weightName) ?: it.name, grams = it.grams)
                PRICE_MODE_NONE -> ModifierOption(name = it.name)
                else -> ModifierOption(name = it.name, priceDelta = it.priceDelta)
            }
        },
    )
}

// ══════════════════════════════════════════════════════════════════════
// **طريقةُ سعر الخيارات** (قرارُ المالك ٢٠٢٦-١٠-٢٨)
// ══════════════════════════════════════════════════════════════════════
//
// «بدون سعر» للمقاس واللون والنمرة، و«حسب الوزن» يُحسب من سعر الصنف.
// **والخادمُ يحسب السعرَ الحقيقيّ** — وهذه المعادلةُ للمعاينة وحدَها،
// وهي نفسُها في `weight_option_delta` (هجرة 0482) وفي `optionModes.ts` للويب.

const val PRICE_MODE_PRICED = "priced"
const val PRICE_MODE_NONE = "none"
const val PRICE_MODE_WEIGHT = "weight"
const val DEFAULT_BASE_GRAMS = 100

/** **أوزانٌ سريعة** — ٢٥٠ غ ونصف كيلو وكيلو. */
val QUICK_GRAMS = listOf(250, 500, 1000)

/** **اسمُ الوزن بالفصحى** — «250 غ»، «نصف كيلو»، «كيلو»، «1.5 كيلو». */
fun weightName(g: Int): String = when {
    g == 500 -> "نصف كيلو"
    g == 1000 -> "كيلو"
    g > 1000 && g % 500 == 0 ->
        java.math.BigDecimal(g).movePointLeft(3).stripTrailingZeros().toPlainString() + " كيلو"
    else -> "$g غ"
}

/** **سعرُ الوزن كاملاً** = سعرُ الصنف × غراماته ÷ الأساس، لأقرب ٥ ل.س. */
fun weightPrice(itemPrice: Long, grams: Int, baseGrams: Int?): Long {
    val base = (baseGrams ?: DEFAULT_BASE_GRAMS).takeIf { it > 0 } ?: DEFAULT_BASE_GRAMS
    return Math.round(itemPrice.toDouble() * grams / base / 5.0) * 5
}

/** **ما يُردّ** — القائمةُ كلُّها معلَّمةً، و`custom` أنّه اختار لنفسه. */
@Serializable
data class OptionPresetsBody(
    val presets: List<OptionPreset> = emptyList(),
    val custom: Boolean = false,
)

/** **ما يُرسَل** — قائمةٌ تستبدل الاختيار، أو `reset` يعيده إلى التصنيف. */
@Serializable
data class SetOptionPresets(
    @SerialName("preset_ids") val presetIds: List<String>? = null,
    val reset: Boolean = false,
)

/** **تغييرُ الأسعار دفعةً واحدة** — وبلا `confirm` معاينةٌ لا تكتب. */
@Serializable
data class BulkPriceInput(
    val percent: Double = 0.0,
    /** **مبلغٌ ثابتٌ بالليرة** (٢٠٢٦-١٠-٢٩) — موجبٌ زيادةً وسالبٌ تخفيضاً، ومعه النسبةُ صفر. */
    val amount: Long? = null,
    @SerialName("platform_section_id") val platformSectionId: String? = null,
    val confirm: Boolean = false,
)

@Serializable
data class BulkPriceSample(
    val name: String = "",
    @SerialName("old_price") val oldPrice: Long = 0,
    @SerialName("new_price") val newPrice: Long = 0,
)

@Serializable
data class BulkPriceResult(
    val count: Int = 0,
    val samples: List<BulkPriceSample> = emptyList(),
    val applied: Boolean = false,
)

@Serializable
data class DuplicatedItem(val id: String = "")
