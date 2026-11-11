package com.rahalgo.customer.orders

import com.rahalgo.shared.customer.MyOrder

/**
 * **عنوانُ الطلب حين لا اسمَ متجر** (قرارُ المالك ٢٠٢٦-١١-١١) — توأمُه في الويب `orderTitle.ts`.
 *
 * **«+N» عددُ الأصناف الأخرى لا عددُ القطع**: سطرٌ واحدٌ بكميّة ٢ يُقال «برغر × 2» لا «برغر +1»،
 * وصنفان مختلفان «برغر +1». **والمشروبُ (`drinkFor`) تابعٌ لصنفه فلا يُعدّ صنفاً.**
 */
internal data class ItemsTitle(val first: String, val others: Int, val qty: Int) {
    fun text(): String = when {
        first.isEmpty() -> ""
        others > 0 -> "$first +$others"
        qty > 1 -> "$first × $qty"
        else -> first
    }

    companion object {
        fun of(order: MyOrder): ItemsTitle {
            val lines = order.items.orEmpty().filter { it.drinkFor == null && it.name.isNotBlank() }
            if (lines.isNotEmpty()) {
                fun key(l: com.rahalgo.shared.customer.MyOrderLine) = l.menuItemId?.takeIf { it.isNotBlank() } ?: l.name.trim()
                val firstKey = key(lines[0])
                return ItemsTitle(
                    first = lines[0].name.trim(),
                    others = lines.map(::key).toSet().size - 1,
                    qty = lines.filter { key(it) == firstKey }.sumOf { it.qty },
                )
            }
            // **بلا بنود: معاينةُ المحرّك** (أسماءٌ فقط، حتّى ثلاثة) فلا كميّاتٍ فيها.
            val parts = order.itemsPreview.split('،', ',').map { it.trim() }.filter { it.isNotEmpty() }
            return ItemsTitle(parts.firstOrNull().orEmpty(), maxOf(parts.size - 1, 0), 1)
        }
    }
}
