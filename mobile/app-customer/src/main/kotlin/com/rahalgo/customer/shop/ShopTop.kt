package com.rahalgo.customer.shop

import com.rahalgo.shared.model.StoreSection

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما فوق أصناف السوق — وبأيّ ترتيب** (قرارُ المالك ٢٠٢٦-١٠-١٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 *   السلايدرُ ظاهرٌ   ←  سلايدر · متاجر مميّزة · بحث
 *   مطفأٌ أو فارغ    ←  متاجر مميّزة · بحث
 *
 * **والويبُ بالترتيب نفسِه** (`(shop)/shop/page.tsx`). **ولا يُرسمان وهو
 * يبحث** — نتائجُه أولى، كما كان السلايدرُ يغيب من قبل.
 *
 * **دالّةٌ خالصة** — تُختبر بلا شاشة.
 */
enum class ShopBlock { Slider, Strip, Search }

object ShopTop {
    fun order(hasSlider: Boolean, hasStrip: Boolean, searching: Boolean): List<ShopBlock> = buildList {
        if (!searching && hasSlider) add(ShopBlock.Slider)
        if (!searching && hasStrip) add(ShopBlock.Strip)
        add(ShopBlock.Search)
    }
}

/**
 * **قائمةُ صفحة المتجر في شبكةٍ واحدة** — عنوانُ قسمٍ يملأ السطر ثمّ أصنافُه.
 *
 *   [عنوان أ] [صنف] [صنف] [عنوان ب] [صنف] …
 *
 * **وشريطُ الأقسام يقفز إلى موضع العنوان**، **ويُضيء القسمَ الذي تقف فيه
 * الشبكة** — فالموضعان يُحسبان هنا لا في الشاشة.
 */
object StoreMenu {
    /** **قسمٌ بلا صنفٍ لا يُعنوَن** — عنوانٌ فوق فراغٍ يُقرأ عطباً. */
    fun visible(sections: List<StoreSection>): List<StoreSection> = sections.filter { it.items.isNotEmpty() }

    /** **موضعُ عنوان كلّ قسمٍ في الشبكة** — و`leading` ما قبل الأقسام من عناصر. */
    fun headerIndex(sections: List<StoreSection>, leading: Int = 0): Map<String, Int> {
        val out = LinkedHashMap<String, Int>()
        var at = leading
        for (s in visible(sections)) {
            out[s.id] = at
            at += 1 + s.items.size
        }
        return out
    }

    /** **أيُّ قسمٍ يملك هذا الموضع** — وما قبل الأوّل يُنسب إلى الأوّل. */
    fun sectionAt(sections: List<StoreSection>, index: Int, leading: Int = 0): String? {
        var owner: String? = null
        for ((id, at) in headerIndex(sections, leading)) {
            if (owner == null || index >= at) owner = id
            if (index < at) break
        }
        return owner
    }
}
