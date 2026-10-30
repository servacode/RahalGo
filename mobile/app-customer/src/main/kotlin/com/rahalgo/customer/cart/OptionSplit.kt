package com.rahalgo.customer.cart

import com.rahalgo.shared.model.ModifierGroup
import com.rahalgo.shared.model.ModifierOption

/**
 * **عدّادٌ لكلّ خيار** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
 *
 * صنفٌ له **مجموعةٌ إلزاميّةٌ واحدةٌ بخيارٍ واحد** (الحجم، الوزن، النمرة) — بلا حسابِ
 * «مشروبات» ولا المجموعات الاختياريّة — **يُختار منه بعدّادٍ جنب كلّ خيار**:
 * «عادي (− ٢ +)» و«دبل (− ١ +)»، ويصير في السلّة سطراً لكلّ خيار.
 *
 * **والإضافاتُ الاختياريّةُ تلحق كلَّ سطر، والمشروباتُ أوّلَها وحدَه** — فلا تتكرّر.
 * ومرآتُه في الويب `splitGroup`/`splitPicks` (`lib/customer/cart.tsx`).
 */
object OptionSplit {

    /** **سقفُ الخيار الواحد** — كسقف المشروب. */
    const val MAX_PER_OPTION = 20

    /** **مجموعةُ العدّادات** — أو `null` فتبقى النافذةُ كما كانت. */
    fun group(groups: List<ModifierGroup>): ModifierGroup? {
        val required = groups.filter { !it.isDrinks && it.minSelect > 0 }
        val g = required.singleOrNull() ?: return null
        return g.takeIf { it.minSelect == 1 && it.maxSelect == 1 }
    }

    /** **سطرٌ لكلّ خيارٍ عددُه فوق الصفر** — والإضافاتُ لكلّها، والمشروباتُ لأوّلها. */
    fun picks(
        group: ModifierGroup,
        counts: Map<String, Int>,
        extras: List<ModifierOption>,
        drinks: List<Cart.DrinkPick>,
    ): List<Cart.Pick> =
        group.options
            .mapNotNull { o -> counts[o.id]?.takeIf { it > 0 }?.let { Cart.Pick(listOf(o) + extras, it) } }
            .mapIndexed { i, p -> if (i == 0) p.copy(drinks = drinks.filter { it.qty > 0 }) else p }

    /** **ثمنُ السطور كلّها** — كلُّ سطرٍ بسعر خياره وعدده، ثمّ مشروباتُه. */
    fun total(itemPrice: Long, picks: List<Cart.Pick>): Long =
        picks.sumOf { p ->
            (itemPrice + p.options.sumOf { it.priceDelta }) * p.qty +
                p.drinks.sumOf { it.option.priceDelta * it.qty }
        }

    /** **ثمنُها بالدولار** لمتجرٍ يعرض به. */
    fun totalUsd(itemPriceUsd: Double, picks: List<Cart.Pick>): Double =
        picks.sumOf { p ->
            (itemPriceUsd + p.options.sumOf { it.priceDeltaUsd ?: 0.0 }) * p.qty +
                p.drinks.sumOf { (it.option.priceDeltaUsd ?: 0.0) * it.qty }
        }
}
