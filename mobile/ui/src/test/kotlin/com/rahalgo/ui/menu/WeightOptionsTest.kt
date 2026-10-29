package com.rahalgo.ui.menu

import com.rahalgo.shared.merchant.ModifierGroup
import com.rahalgo.shared.merchant.ModifierOption
import com.rahalgo.shared.merchant.OptionPreset
import com.rahalgo.shared.merchant.PRICE_MODE_NONE
import com.rahalgo.shared.merchant.PRICE_MODE_WEIGHT
import com.rahalgo.shared.merchant.PresetOption
import com.rahalgo.shared.merchant.weightName
import com.rahalgo.shared.merchant.weightPrice
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * **طريقةُ سعر الخيارات** (قرارُ المالك ٢٠٢٦-١٠-٢٨) — المعاينةُ تطابق معادلةَ
 * الخادم (`weight_option_delta`)، والاسمُ بالفصحى، وما يُرسَل لكلّ طريقة.
 */
class WeightOptionsTest {
    @Test fun names() {
        assertEquals("250 غ", weightName(250))
        assertEquals("نصف كيلو", weightName(500))
        assertEquals("كيلو", weightName(1000))
        assertEquals("كيلو ونصف", weightName(1500))
        assertEquals("2 كيلو", weightName(2000))
        assertEquals("1250 غ", weightName(1250))
    }

    @Test fun priceRoundsToFive() {
        // مثالُ المالك: ٦٠٠ لكلّ ١٠٠ غ.
        assertEquals(1200L, weightPrice(600, 200, 100))
        assertEquals(1500L, weightPrice(600, 250, 100))
        assertEquals(6000L, weightPrice(600, 1000, null))
        // ٣٣٣ × ٢٥٠ ÷ ٢٠٠ = ٤١٦٫٢٥ ⇒ ٤١٥.
        assertEquals(415L, weightPrice(333, 250, 200))
    }

    @Test fun saveSendsGramsOnlyForWeight() {
        val d = ItemDraft(
            price = "600",
            modifiers = listOf(
                ModifierGroup(
                    name = "الوزن", priceMode = PRICE_MODE_WEIGHT,
                    options = listOf(ModifierOption(name = "", grams = 250, priceDelta = 9), ModifierOption(name = "")),
                ),
                ModifierGroup(
                    name = "اللون", priceMode = PRICE_MODE_NONE,
                    options = listOf(ModifierOption(name = "أسود", priceDelta = 300)),
                ),
            ),
        )
        val out = d.modifiersForSave()
        assertEquals(100, out[0].baseGrams)
        assertEquals(listOf("250 غ"), out[0].options.map { it.name })
        assertEquals(0L, out[0].options[0].priceDelta)
        assertEquals(0L, out[1].options[0].priceDelta)
        assertNull(out[1].options[0].priceDeltaUsd)
    }

    @Test fun presetCarriesMode() {
        val g = OptionPreset(
            name = "الوزن", priceMode = PRICE_MODE_WEIGHT,
            options = listOf(PresetOption(name = "x", grams = 250), PresetOption(name = "y", grams = 500)),
        ).toGroup()
        assertEquals(PRICE_MODE_WEIGHT, g.priceMode)
        assertEquals(100, g.baseGrams)
        assertEquals("نصف كيلو", g.options[1].name)
        assertEquals(500, g.options[1].grams)
    }

    /** **«وزن ثقيل» سعرُه لكلّ كيلو** — كلُّ خياراته نصفُ كيلو فأكثر (٢٠٢٦-١٠-٢٩). */
    @Test fun heavyPresetPricesPerKilo() {
        val g = OptionPreset(
            name = "وزن ثقيل", priceMode = PRICE_MODE_WEIGHT,
            options = listOf(PresetOption(name = "a", grams = 500), PresetOption(name = "b", grams = 2000)),
        ).toGroup()
        assertEquals(1000, g.baseGrams)
    }
}
