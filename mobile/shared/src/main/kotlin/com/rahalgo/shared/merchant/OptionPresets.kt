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
) {
    /** **مجموعةٌ تُنسخ إلى الصنف** حين يُعلَّم القالب. */
    fun toGroup(): ModifierGroup = ModifierGroup(
        name = name,
        minSelect = minSelect,
        maxSelect = maxSelect,
        options = options.map { ModifierOption(name = it.name, priceDelta = it.priceDelta) },
    )
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
    val percent: Double,
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
